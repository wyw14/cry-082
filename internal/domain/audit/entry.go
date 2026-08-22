package audit

import (
	"errors"
	"strings"
	"time"
)

var ErrInvalidEntry = errors.New("invalid audit entry")

type Entry struct {
	ID         string
	SiteID     string
	ActorID    string
	Source     string
	Action     string
	Resource   string
	ResourceID string
	Before     map[string]string
	After      map[string]string
	Reason     string
	RequestID  string
	OccurredAt time.Time
}

func New(id, siteID, actorID, source, action, resource, resourceID, reason, requestID string, before, after map[string]string, at time.Time) (Entry, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(actorID) == "" || strings.TrimSpace(source) == "" || strings.TrimSpace(action) == "" || strings.TrimSpace(resource) == "" || strings.TrimSpace(resourceID) == "" || strings.TrimSpace(reason) == "" || at.IsZero() {
		return Entry{}, ErrInvalidEntry
	}
	return Entry{ID: id, SiteID: siteID, ActorID: actorID, Source: source, Action: action, Resource: resource, ResourceID: resourceID, Before: clone(before), After: clone(after), Reason: reason, RequestID: requestID, OccurredAt: at.UTC()}, nil
}

func clone(input map[string]string) map[string]string {
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
