package rule

import (
	"errors"
	"strings"
	"time"
)

var ErrInvalidRecalculation = errors.New("invalid recalculation job")

type RecalculationStatus string

const (
	RecalculationPending   RecalculationStatus = "pending"
	RecalculationRunning   RecalculationStatus = "running"
	RecalculationCompleted RecalculationStatus = "completed"
	RecalculationFailed    RecalculationStatus = "failed"
	RecalculationCancelled RecalculationStatus = "cancelled"
)

type Recalculation struct {
	ID              string
	SiteID          string
	RuleID          string
	FromVersion     int64
	ToVersion       int64
	WindowStart     time.Time
	WindowEnd       time.Time
	Reason          string
	RequestedBy     string
	RequestedAt     time.Time
	Status          RecalculationStatus
	ProcessedPoints int
	Failure         string
}

type RecalculationPlan struct {
	SiteID      string
	RuleID      string
	Reason      string
	ActorID     string
	FromVersion int64
	ToVersion   int64
	WindowStart time.Time
	WindowEnd   time.Time
}

func (p RecalculationPlan) Validate() error {
	if strings.TrimSpace(p.SiteID) == "" || strings.TrimSpace(p.RuleID) == "" {
		return ErrInvalidRecalculation
	}
	if strings.TrimSpace(p.Reason) == "" || strings.TrimSpace(p.ActorID) == "" {
		return ErrInvalidRecalculation
	}
	if p.FromVersion < 1 || p.ToVersion < 1 || p.FromVersion == p.ToVersion {
		return ErrInvalidRecalculation
	}
	if !p.WindowStart.Before(p.WindowEnd) {
		return ErrInvalidRecalculation
	}
	return nil
}

func (p RecalculationPlan) Schedule(id string, now time.Time) (Recalculation, error) {
	if err := p.Validate(); err != nil {
		return Recalculation{}, err
	}
	job, err := NewRecalculation(id, p.SiteID, p.RuleID, p.Reason, p.ActorID, p.FromVersion, p.ToVersion, p.WindowStart, p.WindowEnd, now)
	if err != nil {
		return Recalculation{}, err
	}
	if err := job.Start(); err != nil {
		return Recalculation{}, err
	}
	return job, nil
}

func NewRecalculation(id, siteID, ruleID, reason, actor string, fromVersion, toVersion int64, start, end, now time.Time) (Recalculation, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(siteID) == "" || strings.TrimSpace(ruleID) == "" || strings.TrimSpace(reason) == "" || strings.TrimSpace(actor) == "" || fromVersion < 1 || toVersion < 1 || toVersion == fromVersion || !start.Before(end) {
		return Recalculation{}, ErrInvalidRecalculation
	}
	return Recalculation{ID: id, SiteID: siteID, RuleID: ruleID, FromVersion: fromVersion, ToVersion: toVersion, WindowStart: start.UTC(), WindowEnd: end.UTC(), Reason: reason, RequestedBy: actor, RequestedAt: now.UTC(), Status: RecalculationPending}, nil
}

func (r *Recalculation) Start() error {
	if r.Status != RecalculationPending {
		return ErrInvalidRecalculation
	}
	r.Status = RecalculationRunning
	return nil
}

func (r *Recalculation) Complete(processed int) error {
	if r.Status != RecalculationRunning || processed < 0 {
		return ErrInvalidRecalculation
	}
	r.ProcessedPoints = processed
	r.Status = RecalculationCompleted
	return nil
}

func (r *Recalculation) Fail(reason string) error {
	if r.Status != RecalculationRunning || strings.TrimSpace(reason) == "" {
		return ErrInvalidRecalculation
	}
	r.Failure = reason
	r.Status = RecalculationFailed
	return nil
}
