package alerts

import (
	"context"
	"errors"
	"time"

	"github.com/wyw14/cry-082/internal/domain/alert"
	"github.com/wyw14/cry-082/internal/domain/audit"
	"github.com/wyw14/cry-082/internal/domain/site"
	"github.com/wyw14/cry-082/internal/platform/transaction"
)

var ErrAlertVersionConflict = errors.New("alert version conflict")

type Clock interface{ Now() time.Time }
type IDGenerator interface{ NewID() string }
type Repository interface {
	FindAlert(context.Context, string) (alert.Alert, error)
	ListAlerts(context.Context, AlertFilter) ([]alert.Alert, int, error)
	SaveAlert(context.Context, alert.Alert) error
	SaveWorkOrder(context.Context, alert.WorkOrder) error
}

type AlertFilter struct {
	SiteID     string
	Kind       alert.Kind
	Status     alert.Status
	Sort       string
	Descending bool
	Offset     int
	Limit      int
}

func (s *Service) List(ctx context.Context, filter AlertFilter, actorID string) ([]alert.Alert, int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if filter.Offset < 0 || filter.Limit < 1 || filter.Limit > 200 {
		return nil, 0, errors.New("invalid pagination")
	}
	if filter.Sort != "started_at" && filter.Sort != "last_signal_at" && filter.Sort != "status" {
		return nil, 0, errors.New("unsupported sort field")
	}
	membership, err := s.access.Membership(ctx, actorID, filter.SiteID)
	if err != nil {
		return nil, 0, err
	}
	if err := site.Require(membership, filter.SiteID, site.PermissionSiteRead); err != nil {
		return nil, 0, err
	}
	return s.alerts.ListAlerts(ctx, filter)
}

type AccessRepository interface {
	Membership(context.Context, string, string) (site.Membership, error)
}
type AuditRepository interface {
	AppendAudit(context.Context, audit.Entry) error
}
type Outbox interface {
	Enqueue(context.Context, string, string, any) error
}

type Service struct {
	alerts Repository
	access AccessRepository
	audits AuditRepository
	outbox Outbox
	tx     transaction.Manager
	clock  Clock
	ids    IDGenerator
}

func New(repository Repository, access AccessRepository, audits AuditRepository, outbox Outbox, tx transaction.Manager, clock Clock, ids IDGenerator) *Service {
	return &Service{alerts: repository, access: access, audits: audits, outbox: outbox, tx: tx, clock: clock, ids: ids}
}

type TransitionInput struct {
	AlertID                                string
	Next                                   alert.Status
	ActorID, Reason, AssigneeID, RequestID string
	ExpectedVersion                        int64
}

func (s *Service) Transition(ctx context.Context, input TransitionInput) (updated alert.Alert, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = transaction.Execute(ctx, s.tx, func(txctx context.Context) error {
		updated, err = s.transition(txctx, input)
		return err
	})
	return updated, err
}

func (s *Service) transition(ctx context.Context, input TransitionInput) (alert.Alert, error) {
	current, err := s.alerts.FindAlert(ctx, input.AlertID)
	if err != nil {
		return alert.Alert{}, err
	}
	membership, err := s.access.Membership(ctx, input.ActorID, current.SiteID)
	if err != nil {
		return alert.Alert{}, err
	}
	if err := site.Require(membership, current.SiteID, site.PermissionAlertDispatch); err != nil {
		return alert.Alert{}, err
	}
	if input.ExpectedVersion != current.Version {
		return alert.Alert{}, ErrAlertVersionConflict
	}
	before := current.Status
	if err := current.Transition(input.Next, input.ActorID, input.Reason, input.AssigneeID, s.clock.Now()); err != nil {
		return alert.Alert{}, err
	}
	if err := s.alerts.SaveAlert(ctx, current); err != nil {
		return alert.Alert{}, err
	}
	entry, err := audit.New(s.ids.NewID(), current.SiteID, input.ActorID, "api", "alert.transitioned", "alert", current.ID, input.Reason, input.RequestID, map[string]string{"status": string(before)}, map[string]string{"status": string(current.Status), "assignee_id": current.AssigneeID}, s.clock.Now())
	if err != nil {
		return alert.Alert{}, err
	}
	if err := s.audits.AppendAudit(ctx, entry); err != nil {
		return alert.Alert{}, err
	}
	if err := s.outbox.Enqueue(ctx, "alert.transitioned", current.ID, map[string]any{"alert_id": current.ID, "from": before, "to": current.Status}); err != nil {
		return alert.Alert{}, err
	}
	return current, nil
}

func (s *Service) Dispatch(ctx context.Context, input TransitionInput, description string, dueAt time.Time) (alert.Alert, alert.WorkOrder, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	input.Next = alert.StatusDispatched
	var updated alert.Alert
	var workOrder alert.WorkOrder
	err := transaction.Execute(ctx, s.tx, func(txctx context.Context) error {
		var err error
		updated, err = s.transition(txctx, input)
		if err != nil {
			return err
		}
		workOrder, err = alert.NewWorkOrder(s.ids.NewID(), updated.ID, input.AssigneeID, description, s.clock.Now(), dueAt)
		if err != nil {
			return err
		}
		if err := s.alerts.SaveWorkOrder(txctx, workOrder); err != nil {
			return err
		}
		return s.outbox.Enqueue(txctx, "work-order.assigned", workOrder.ID, map[string]any{"work_order_id": workOrder.ID, "assignee_id": workOrder.AssigneeID})
	})
	return updated, workOrder, err
}
