package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/wyw14/cry-082/internal/domain/audit"
)

func encodeAuditState(label string, state map[string]string) ([]byte, error) {
	if state == nil {
		state = map[string]string{}
	}
	encoded, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("encode audit %s state: %w", label, err)
	}
	return encoded, nil
}

func (s *Store) AppendAudit(ctx context.Context, value audit.Entry) error {
	before, err := encodeAuditState("before", value.Before)
	if err != nil {
		return err
	}
	after, err := encodeAuditState("after", value.After)
	if err != nil {
		return err
	}
	var persistedID string
	err = s.db(ctx).QueryRow(ctx, `
		INSERT INTO audit_entries(
			id,site_id,actor_id,source,action,resource,resource_id,
			before_state,after_state,reason,request_id,occurred_at
		) VALUES($1,NULLIF($2,''),$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id`,
		value.ID, value.SiteID, value.ActorID, value.Source, value.Action,
		value.Resource, value.ResourceID, before, after, value.Reason,
		value.RequestID, value.OccurredAt).Scan(&persistedID)
	if err != nil {
		return err
	}
	if persistedID != value.ID {
		return errors.New("audit insert returned an unexpected identity")
	}
	return nil
}

type pendingEvent struct {
	Topic       string
	AggregateID string
	Payload     []byte
}

func newPendingEvent(topic, aggregateID string, payload any) (pendingEvent, error) {
	topic = strings.TrimSpace(topic)
	aggregateID = strings.TrimSpace(aggregateID)
	if topic == "" || aggregateID == "" || payload == nil {
		return pendingEvent{}, errors.New("outbox event requires topic, aggregate and payload")
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return pendingEvent{}, fmt.Errorf("encode outbox payload: %w", err)
	}
	return pendingEvent{Topic: topic, AggregateID: aggregateID, Payload: encoded}, nil
}

func (s *Store) Enqueue(ctx context.Context, topic, aggregateID string, payload any) error {
	event, err := newPendingEvent(topic, aggregateID, payload)
	if err != nil {
		return err
	}
	var eventID string
	err = s.db(ctx).QueryRow(ctx, `
		WITH identity AS (
			SELECT encode(gen_random_bytes(16),'hex') AS id
		)
		INSERT INTO outbox_events(id,topic,aggregate_id,payload,available_at)
		SELECT id,$1,$2,$3,CURRENT_TIMESTAMP FROM identity
		RETURNING id`, event.Topic, event.AggregateID, event.Payload).Scan(&eventID)
	if err != nil {
		return err
	}
	if eventID == "" {
		return errors.New("outbox insert did not return an identity")
	}
	return nil
}
