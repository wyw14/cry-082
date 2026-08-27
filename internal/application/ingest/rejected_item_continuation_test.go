package ingest

import (
	"context"
	"testing"
	"time"

	"github.com/wyw14/cry-082/internal/domain/telemetry"
	clockpkg "github.com/wyw14/cry-082/internal/platform/clock"
	"github.com/wyw14/cry-082/internal/platform/idempotency"
	"github.com/wyw14/cry-082/internal/platform/outbox"
	"github.com/wyw14/cry-082/internal/repository/memory"
)

func TestRejectedItemDoesNotSkipLaterValidSample(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC)
	store := memory.New()
	mustSeed(t, ctx, store, now)
	service := New(Dependencies{Devices: store, Schemas: store, Observations: store, Rules: store, Evaluations: store, Alerts: store, Outbox: outbox.NewStore(), Transactions: memory.TransactionManager{}, Clock: clockpkg.NewManual(now), IDs: &idempotency.Generator{}}, telemetry.QualityPolicy{FutureTolerance: time.Minute, LateAfter: 10 * time.Minute, SpikeMultiplier: 3}, 50)
	result, err := service.Ingest(ctx, Batch{BatchID: "mixed-batch", ReceivedAt: now, Samples: []Sample{
		{DeviceCode: "UNKNOWN", SchemaID: "pm10", Value: 120, SampledAt: now.Add(-2 * time.Minute)},
		{DeviceCode: "D1", SchemaID: "pm10", Value: 120, SampledAt: now.Add(-time.Minute)},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("later sample was skipped: %+v", result)
	}
	if result.Items[0].ErrorCode != "INGEST_ITEM_REJECTED" || result.Items[1].ErrorCode != "" || result.Accepted != 1 {
		t.Fatalf("unexpected mixed batch result: %+v", result)
	}
}
