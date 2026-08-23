package alert

import (
	"errors"
	"strings"
)

var ErrInvalidListScope = errors.New("invalid alert list scope")

type ListScope struct {
	SiteID string
	Kind   Kind
	Status Status
	Offset int
	Limit  int
}

func NewListScope(siteID string, kind Kind, status Status, offset, limit int) (ListScope, error) {
	scope := ListScope{SiteID: strings.TrimSpace(siteID), Kind: kind, Status: status, Offset: offset, Limit: limit}
	if scope.SiteID == "" || scope.Offset < 0 || scope.Limit < 1 || scope.Limit > 200 {
		return ListScope{}, ErrInvalidListScope
	}
	if kind != "" && kind != KindEnvironmentalExceedance && kind != KindDeviceOffline && kind != KindDeviceDrift {
		return ListScope{}, ErrInvalidListScope
	}
	validStatuses := map[Status]bool{
		"":                 true,
		StatusOpen:         true,
		StatusAcknowledged: true,
		StatusDispatched:   true,
		StatusRecovering:   true,
		StatusRecovered:    true,
		StatusClosed:       true,
	}
	if !validStatuses[status] {
		return ListScope{}, ErrInvalidListScope
	}
	return scope, nil
}

func (s ListScope) EffectiveStatus() Status {
	return s.Status
}
