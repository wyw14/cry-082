package ingest

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/wyw14/cry-082/internal/domain/telemetry"
	clockpkg "github.com/wyw14/cry-082/internal/platform/clock"
	"github.com/wyw14/cry-082/internal/platform/idempotency"
	"github.com/wyw14/cry-082/internal/platform/outbox"
	"github.com/wyw14/cry-082/internal/repository/memory"
)

type synchronizedIdentityJournal struct {
	ObservationJournal
	arrived chan struct{}
	release chan struct{}
}

func (j *synchronizedIdentityJournal) ExistsIdentity(ctx context.Context, identity string) (bool, error) {
	exists, err := j.ObservationJournal.ExistsIdentity(ctx, identity)
	j.arrived <- struct{}{}
	select {
	case <-j.release:
		return exists, err
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

func TestConcurrentIdentityReplayReturnsOneDuplicate(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC)
	store := memory.New()
	mustSeed(t, ctx, store, now)
	journal := &synchronizedIdentityJournal{ObservationJournal: store, arrived: make(chan struct{}, 2), release: make(chan struct{})}
	service := New(Dependencies{Devices: store, Schemas: store, Observations: journal, Rules: store, Evaluations: store, Alerts: store, Outbox: outbox.NewStore(), Transactions: memory.TransactionManager{}, Clock: clockpkg.NewManual(now), IDs: &idempotency.Generator{}}, telemetry.QualityPolicy{FutureTolerance: time.Minute, LateAfter: time.Hour, SpikeMultiplier: 3}, 10)
	sample := Sample{DeviceCode: "D1", SchemaID: "pm10", Value: 80, SampledAt: now.Add(-time.Minute)}

	results := make(chan BatchResult, 2)
	errors := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for index := 0; index < 2; index++ {
		go func(batchID string) {
			ready.Done()
			result, err := service.Ingest(ctx, Batch{BatchID: batchID, ReceivedAt: now, Samples: []Sample{sample}})
			results <- result
			errors <- err
		}("replay-" + string(rune('a'+index)))
	}
	ready.Wait()
	<-journal.arrived
	<-journal.arrived
	close(journal.release)

	accepted, duplicates := 0, 0
	for index := 0; index < 2; index++ {
		if err := <-errors; err != nil {
			t.Fatalf("ingest returned error: %v", err)
		}
		result := <-results
		accepted += result.Accepted
		duplicates += result.Duplicates
		if len(result.Items) != 1 || result.Items[0].ErrorCode != "" {
			t.Fatalf("unexpected item result: %+v", result)
		}
	}
	if accepted != 1 || duplicates != 1 {
		t.Fatalf("accepted=%d duplicates=%d", accepted, duplicates)
	}
}
