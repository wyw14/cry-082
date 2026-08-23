package ingest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wyw14/cry-082/internal/domain/telemetry"
	clockpkg "github.com/wyw14/cry-082/internal/platform/clock"
	"github.com/wyw14/cry-082/internal/platform/idempotency"
	"github.com/wyw14/cry-082/internal/platform/outbox"
	"github.com/wyw14/cry-082/internal/repository/memory"
)

// racingObservationJournal reproduces the concurrent-replay condition deterministically:
// two ingests for the same device/point/sample both pass ExistsIdentity (both see
// false), then both Append. The store rejects the second as a duplicate identity —
// exactly what the Postgres unique constraint and memory store surface to a loser
// that raced the winner. The ingest service must treat that conflict as a duplicate,
// not an ingest rejection.
type racingObservationJournal struct {
	*memory.Store
	failAppendWith error // when non-nil, Append returns this instead of writing
}

func (j *racingObservationJournal) Append(ctx context.Context, value telemetry.Observation) error {
	if j.failAppendWith != nil {
		return j.failAppendWith
	}
	return j.Store.Append(ctx, value)
}

// TestConcurrentReplayLoserSurfacesAsDuplicate verifies that when Append loses the
// race on the unique identity, the item is classified as a duplicate (counted in
// Duplicates) instead of INGEST_ITEM_REJECTED. Before the fix the loser returned the
// raw Append error, so it was neither Accepted nor a Duplicate — it read as an ingest
// failure and the duplicate count never increased.
func TestConcurrentReplayLoserSurfacesAsDuplicate(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC)
	store := memory.New()
	mustSeed(t, ctx, store, now)
	journal := &racingObservationJournal{Store: store, failAppendWith: telemetry.ErrObservationIdentityConflict}
	clock := clockpkg.NewManual(now)
	ids := &idempotency.Generator{}
	service := New(Dependencies{Devices: journal, Schemas: journal, Observations: journal, Rules: journal, Evaluations: journal, Alerts: journal, Outbox: outbox.NewStore(), Transactions: memory.TransactionManager{}, Clock: clock, IDs: ids}, telemetry.QualityPolicy{FutureTolerance: time.Minute, LateAfter: 10 * time.Minute, SpikeMultiplier: 3}, 50)

	sampled := now.Add(-time.Minute)
	result, err := service.Ingest(ctx, Batch{BatchID: "batch-1", ReceivedAt: now, Samples: []Sample{{DeviceCode: "D1", SchemaID: "pm10", Value: 200, SampledAt: sampled}}})
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	if result.Accepted != 0 || result.Duplicates != 1 || result.Quarantined != 0 {
		t.Fatalf("result=%+v want Accepted=0 Duplicates=1 Quarantined=0", result)
	}
	if result.Items[0].ErrorCode != "" {
		t.Fatalf("loser item rejected with code %q, want duplicate with no error", result.Items[0].ErrorCode)
	}
	if !result.Items[0].Duplicate {
		t.Fatalf("loser item not marked duplicate")
	}
}

// TestGenuineAppendErrorIsStillRejected ensures the conflict-idempotence path does not
// swallow real storage failures: a non-conflict Append error must still surface as an
// item rejection (INGEST_ITEM_REJECTED) so genuine ingest failures keep being counted.
func TestGenuineAppendErrorIsStillRejected(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC)
	store := memory.New()
	mustSeed(t, ctx, store, now)
	journal := &racingObservationJournal{Store: store, failAppendWith: errors.New("connection refused")}
	clock := clockpkg.NewManual(now)
	ids := &idempotency.Generator{}
	service := New(Dependencies{Devices: journal, Schemas: journal, Observations: journal, Rules: journal, Evaluations: journal, Alerts: journal, Outbox: outbox.NewStore(), Transactions: memory.TransactionManager{}, Clock: clock, IDs: ids}, telemetry.QualityPolicy{FutureTolerance: time.Minute, LateAfter: 10 * time.Minute, SpikeMultiplier: 3}, 50)

	sampled := now.Add(-time.Minute)
	result, err := service.Ingest(ctx, Batch{BatchID: "batch-1", ReceivedAt: now, Samples: []Sample{{DeviceCode: "D1", SchemaID: "pm10", Value: 200, SampledAt: sampled}}})
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	if result.Accepted != 0 || result.Duplicates != 0 {
		t.Fatalf("result=%+v want Accepted=0 Duplicates=0", result)
	}
	if result.Items[0].ErrorCode != "INGEST_ITEM_REJECTED" {
		t.Fatalf("genuine append error classified as %q, want INGEST_ITEM_REJECTED", result.Items[0].ErrorCode)
	}
}
