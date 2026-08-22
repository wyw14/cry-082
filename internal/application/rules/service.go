package rules

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wyw14/cry-082/internal/domain/audit"
	"github.com/wyw14/cry-082/internal/domain/rule"
	"github.com/wyw14/cry-082/internal/domain/site"
	"github.com/wyw14/cry-082/internal/platform/transaction"
)

var ErrVersionConflict = errors.New("rule version conflict")

type Clock interface{ Now() time.Time }
type IDGenerator interface{ NewID() string }
type Repository interface {
	LatestRule(context.Context, string) (*rule.Version, error)
	SaveRule(context.Context, rule.Version) error
	SaveRecalculation(context.Context, rule.Recalculation) error
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
	rules  Repository
	access AccessRepository
	audits AuditRepository
	outbox Outbox
	tx     transaction.Manager
	clock  Clock
	ids    IDGenerator
}

func New(repository Repository, access AccessRepository, audits AuditRepository, outbox Outbox, tx transaction.Manager, clock Clock, ids IDGenerator) *Service {
	return &Service{rules: repository, access: access, audits: audits, outbox: outbox, tx: tx, clock: clock, ids: ids}
}

type CreateVersionInput struct {
	RuleID, SiteID, Name, Timezone, ActorID, Reason, RequestID string
	Conditions                                                 []rule.Condition
	RequireAll                                                 bool
	Duration, MergeWindow, LateGrace                           time.Duration
	EffectiveFrom                                              time.Time
}

func (s *Service) CreateVersion(ctx context.Context, input CreateVersionInput) (rule.Version, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	membership, err := s.access.Membership(ctx, input.ActorID, input.SiteID)
	if err != nil {
		return rule.Version{}, err
	}
	if err := site.Require(membership, input.SiteID, site.PermissionRuleManage); err != nil {
		return rule.Version{}, err
	}
	latest, err := s.rules.LatestRule(ctx, input.RuleID)
	if err != nil {
		return rule.Version{}, err
	}
	next := int64(1)
	if latest != nil {
		next = latest.Version + 1
	}
	created, err := rule.NewVersion(input.RuleID, input.SiteID, input.Name, input.Timezone, input.ActorID, next, input.Conditions, input.Duration, input.MergeWindow, input.LateGrace, input.EffectiveFrom, s.clock.Now())
	if err != nil {
		return rule.Version{}, err
	}
	created.RequireAll = input.RequireAll
	err = transaction.Execute(ctx, s.tx, func(txctx context.Context) error {
		if err := s.rules.SaveRule(txctx, created); err != nil {
			return err
		}
		entry, err := audit.New(s.ids.NewID(), input.SiteID, input.ActorID, "api", "rule.version.created", "rule", input.RuleID, input.Reason, input.RequestID, versionView(latest), versionView(&created), s.clock.Now())
		if err != nil {
			return err
		}
		return s.audits.AppendAudit(txctx, entry)
	})
	return created, err
}

func (s *Service) Activate(ctx context.Context, siteID, ruleID, actorID, reason, requestID string, version int64) (rule.Version, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	membership, err := s.access.Membership(ctx, actorID, siteID)
	if err != nil {
		return rule.Version{}, err
	}
	if err := site.Require(membership, siteID, site.PermissionRuleManage); err != nil {
		return rule.Version{}, err
	}
	latest, err := s.rules.LatestRule(ctx, ruleID)
	if err != nil {
		return rule.Version{}, err
	}
	if latest == nil || latest.Version != version {
		return rule.Version{}, ErrVersionConflict
	}
	plan, err := rule.PlanActivation(*latest, s.clock.Now())
	if err != nil {
		return rule.Version{}, err
	}
	if err := plan.Apply(latest); err != nil {
		return rule.Version{}, err
	}
	err = transaction.Execute(ctx, s.tx, func(txctx context.Context) error {
		if err := s.rules.SaveRule(txctx, *latest); err != nil {
			return err
		}
		entry, err := audit.New(s.ids.NewID(), siteID, actorID, "api", "rule.version.activated", "rule", ruleID, reason, requestID, map[string]string{"status": string(rule.StatusDraft)}, map[string]string{"status": string(latest.Status), "version": formatVersion(latest.Version)}, s.clock.Now())
		if err != nil {
			return err
		}
		if err := s.audits.AppendAudit(txctx, entry); err != nil {
			return err
		}
		return s.outbox.Enqueue(txctx, "rule.activated", ruleID, map[string]any{"rule_id": ruleID, "version": latest.Version, "site_id": siteID})
	})
	return *latest, err
}

func (s *Service) RequestRecalculation(ctx context.Context, siteID, ruleID, actor, reason string, fromVersion, toVersion int64, start, end time.Time) (job rule.Recalculation, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	membership, err := s.access.Membership(ctx, actor, siteID)
	if err != nil {
		return rule.Recalculation{}, err
	}
	if err := site.Require(membership, siteID, site.PermissionRuleManage); err != nil {
		return rule.Recalculation{}, err
	}
	job, err = rule.NewRecalculation(s.ids.NewID(), siteID, ruleID, reason, actor, fromVersion, toVersion, start, end, s.clock.Now())
	if err != nil {
		return rule.Recalculation{}, err
	}
	err = transaction.Execute(ctx, s.tx, func(txctx context.Context) error {
		if err := s.rules.SaveRecalculation(txctx, job); err != nil {
			return err
		}
		return s.outbox.Enqueue(txctx, "rule.recalculation.requested", job.ID, map[string]any{"job_id": job.ID, "site_id": siteID})
	})
	return job, err
}

func versionView(version *rule.Version) map[string]string {
	if version == nil {
		return nil
	}
	return map[string]string{"version": formatVersion(version.Version), "status": string(version.Status)}
}

func formatVersion(version int64) string { return fmt.Sprintf("%d", version) }
