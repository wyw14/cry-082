package siteapp

import (
	"context"
	"errors"
	"time"

	"github.com/wyw14/cry-082/internal/domain/audit"
	"github.com/wyw14/cry-082/internal/domain/site"
	"github.com/wyw14/cry-082/internal/platform/transaction"
)

var ErrConflict = errors.New("optimistic concurrency conflict")

type Clock interface{ Now() time.Time }
type IDGenerator interface{ NewID() string }
type Repository interface {
	SaveSite(context.Context, site.Site) error
	SaveZone(context.Context, site.Zone) error
	SavePoint(context.Context, site.MonitoringPoint) error
	Membership(context.Context, string, string) (site.Membership, error)
}
type AuditRepository interface {
	AppendAudit(context.Context, audit.Entry) error
}

type Service struct {
	repo   Repository
	audits AuditRepository
	tx     transaction.Manager
	clock  Clock
	ids    IDGenerator
}

func New(repo Repository, audits AuditRepository, tx transaction.Manager, clock Clock, ids IDGenerator) *Service {
	return &Service{repo: repo, audits: audits, tx: tx, clock: clock, ids: ids}
}

type CreateSiteInput struct{ Name, Timezone, ResponsibleUnit, ActorID, RequestID string }

func (s *Service) CreateSite(ctx context.Context, input CreateSiteInput) (created site.Site, err error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	created, err = site.New(s.ids.NewID(), input.Name, input.Timezone, input.ResponsibleUnit, s.clock.Now())
	if err != nil {
		return site.Site{}, err
	}
	tx, err := s.tx.Begin(ctx)
	if err != nil {
		return site.Site{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()
	ctx = tx.Bind(ctx)
	if err = s.repo.SaveSite(ctx, created); err != nil {
		return site.Site{}, err
	}
	entry, err := audit.New(s.ids.NewID(), created.ID, input.ActorID, "api", "site.created", "site", created.ID, "create site", input.RequestID, nil, map[string]string{"name": created.Name, "timezone": created.Timezone}, s.clock.Now())
	if err != nil {
		return site.Site{}, err
	}
	if err = s.audits.AppendAudit(ctx, entry); err != nil {
		return site.Site{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return site.Site{}, err
	}
	return created, nil
}

const defaultTimeout = 5 * time.Second
