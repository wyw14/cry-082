package alert

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidAlert           = errors.New("invalid alert")
	ErrInvalidAlertTransition = errors.New("invalid alert state transition")
)

type Kind string

const (
	KindEnvironmentalExceedance Kind = "environmental-exceedance"
	KindDeviceOffline           Kind = "device-offline"
	KindDeviceDrift             Kind = "device-drift"
)

type Status string

const (
	StatusOpen         Status = "open"
	StatusAcknowledged Status = "acknowledged"
	StatusDispatched   Status = "dispatched"
	StatusRecovering   Status = "recovering"
	StatusRecovered    Status = "recovered"
	StatusClosed       Status = "closed"
)

type Event struct {
	From       Status
	To         Status
	ActorID    string
	Reason     string
	OccurredAt time.Time
}

type Alert struct {
	ID              string
	SiteID          string
	PointID         string
	DeviceID        string
	Kind            Kind
	RuleID          string
	RuleVersion     int64
	Status          Status
	StartedAt       time.Time
	LastSignalAt    time.Time
	RecoveredAt     *time.Time
	ClosedAt        *time.Time
	AssigneeID      string
	MergeKey        string
	OccurrenceCount int
	Version         int64
	Events          []Event
}

func New(id, siteID, pointID, deviceID string, kind Kind, ruleID string, ruleVersion int64, mergeKey string, at time.Time) (Alert, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(siteID) == "" || strings.TrimSpace(deviceID) == "" || strings.TrimSpace(mergeKey) == "" {
		return Alert{}, ErrInvalidAlert
	}
	if kind == KindEnvironmentalExceedance && (pointID == "" || ruleID == "" || ruleVersion < 1) {
		return Alert{}, ErrInvalidAlert
	}
	if kind == KindDeviceOffline && ruleID != "" {
		return Alert{}, ErrInvalidAlert
	}
	return Alert{ID: id, SiteID: siteID, PointID: pointID, DeviceID: deviceID, Kind: kind, RuleID: ruleID, RuleVersion: ruleVersion, Status: StatusOpen, StartedAt: at.UTC(), LastSignalAt: at.UTC(), MergeKey: mergeKey, OccurrenceCount: 1, Version: 1}, nil
}

func (a *Alert) Merge(at time.Time) error {
	if a.Status == StatusClosed || a.Status == StatusRecovered {
		return ErrInvalidAlertTransition
	}
	a.LastSignalAt = at.UTC()
	a.OccurrenceCount++
	a.Version++
	return nil
}

func (a *Alert) Transition(next Status, actor, reason, assignee string, at time.Time) error {
	allowed := map[Status]map[Status]bool{
		StatusOpen:         {StatusAcknowledged: true},
		StatusAcknowledged: {StatusDispatched: true, StatusRecovering: true},
		StatusDispatched:   {StatusRecovering: true},
		StatusRecovering:   {StatusRecovered: true},
		StatusRecovered:    {StatusClosed: true, StatusOpen: true},
	}
	if strings.TrimSpace(actor) == "" || strings.TrimSpace(reason) == "" || !allowed[a.Status][next] {
		return ErrInvalidAlertTransition
	}
	if next == StatusDispatched && strings.TrimSpace(assignee) == "" {
		return ErrInvalidAlertTransition
	}
	previous := a.Status
	a.Status = next
	if assignee != "" {
		a.AssigneeID = assignee
	}
	when := at.UTC()
	if next == StatusRecovered {
		a.RecoveredAt = &when
	}
	if next == StatusClosed {
		a.ClosedAt = &when
	}
	if next == StatusOpen {
		a.RecoveredAt = nil
		a.ClosedAt = nil
	}
	a.Events = append(a.Events, Event{From: previous, To: next, ActorID: actor, Reason: reason, OccurredAt: when})
	a.Version++
	return nil
}
