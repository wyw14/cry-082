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

func TestFutureSampleIsQuarantinedBeforeRuleEvaluation(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC)
	store := memory.New()
	mustSeed(t, ctx, store, now)
	service := New(Dependencies{Devices: store, Schemas: store, Observations: store, Rules: store, Evaluations: store, Alerts: store, Outbox: outbox.NewStore(), Transactions: memory.TransactionManager{}, Clock: clockpkg.NewManual(now), IDs: &idempotency.Generator{}}, telemetry.QualityPolicy{FutureTolerance: time.Minute, LateAfter: 10 * time.Minute, SpikeMultiplier: 3}, 50)
	result, err := service.Ingest(ctx, Batch{BatchID: "future-batch", ReceivedAt: now, Samples: []Sample{{DeviceCode: "D1", SchemaID: "pm10", Value: 200, SampledAt: now.Add(10 * time.Minute)}}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Quarantined != 1 || result.Accepted != 0 || len(result.Items) != 1 || result.Items[0].Quality != telemetry.QualityQuarantined {
		t.Fatalf("future sample was not isolated: %+v", result)
	}
}
