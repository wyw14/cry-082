package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

var ErrEventNotFound = errors.New("outbox event not found")

type Event struct {
	ID, Topic, AggregateID string
	Payload                []byte
	Attempts               int
	AvailableAt            time.Time
	PublishedAt            *time.Time
	LastError              string
}
type Store struct {
	mu       sync.Mutex
	events   map[string]Event
	sequence uint64
}

func NewStore() *Store { return &Store{events: make(map[string]Event)} }

func (s *Store) Enqueue(ctx context.Context, topic, aggregateID string, payload any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	id := fmt.Sprintf("event-%020d", s.sequence)
	s.events[id] = Event{ID: id, Topic: topic, AggregateID: aggregateID, Payload: encoded, AvailableAt: time.Now().UTC()}
	return nil
}
func (s *Store) Pending(ctx context.Context, now time.Time, limit int) ([]Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]Event, 0)
	for _, event := range s.events {
		if event.PublishedAt == nil && !event.AvailableAt.After(now) {
			result = append(result, event)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}
func (s *Store) MarkPublished(ctx context.Context, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	event, ok := s.events[id]
	if !ok {
		return ErrEventNotFound
	}
	when := at.UTC()
	event.PublishedAt = &when
	s.events[id] = event
	return nil
}
func (s *Store) MarkFailed(ctx context.Context, id, reason string, next time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	event, ok := s.events[id]
	if !ok {
		return ErrEventNotFound
	}
	event.Attempts++
	event.LastError = reason
	event.AvailableAt = next.UTC()
	s.events[id] = event
	return nil
}
