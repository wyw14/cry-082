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

func TestNoiseSampleDoesNotEnterParticleRule(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC)
	store := memory.New()
	mustSeed(t, ctx, store, now)
	noise, err := telemetry.NewSchema("noise", telemetry.MetricNoise, "dB", time.Minute, 20, 180, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveSchema(ctx, noise); err != nil {
		t.Fatal(err)
	}
	service := New(Dependencies{Devices: store, Schemas: store, Observations: store, Rules: store, Evaluations: store, Alerts: store, Outbox: outbox.NewStore(), Transactions: memory.TransactionManager{}, Clock: clockpkg.NewManual(now), IDs: &idempotency.Generator{}}, telemetry.QualityPolicy{FutureTolerance: time.Minute, LateAfter: time.Hour, SpikeMultiplier: 3}, 10)
	result, err := service.Ingest(ctx, Batch{BatchID: "noise-only", ReceivedAt: now, Samples: []Sample{{DeviceCode: "D1", SchemaID: "noise", Value: 170, SampledAt: now}}})
	if err != nil || result.Accepted != 1 {
		t.Fatalf("ingest result=%+v err=%v", result, err)
	}
	alerts, err := store.RangeAlerts(ctx, "s1", now.Add(-time.Minute), now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 0 {
		t.Fatalf("noise sample generated particle alert: %+v", alerts)
	}
}
