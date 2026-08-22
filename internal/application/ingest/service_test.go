package ingest

import (
	"context"
	"testing"
	"time"

	"github.com/wyw14/cry-082/internal/domain/device"
	"github.com/wyw14/cry-082/internal/domain/rule"
	"github.com/wyw14/cry-082/internal/domain/site"
	"github.com/wyw14/cry-082/internal/domain/telemetry"
	clockpkg "github.com/wyw14/cry-082/internal/platform/clock"
	"github.com/wyw14/cry-082/internal/platform/idempotency"
	"github.com/wyw14/cry-082/internal/platform/outbox"
	"github.com/wyw14/cry-082/internal/repository/memory"
)

func TestBatchDeduplicatesAndLateSampleDoesNotAlert(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC)
	store := memory.New()
	mustSeed(t, ctx, store, now)
	clock := clockpkg.NewManual(now)
	ids := &idempotency.Generator{}
	service := New(Dependencies{Devices: store, Schemas: store, Observations: store, Rules: store, Evaluations: store, Alerts: store, Outbox: outbox.NewStore(), Transactions: memory.TransactionManager{}, Clock: clock, IDs: ids}, telemetry.QualityPolicy{FutureTolerance: time.Minute, LateAfter: 10 * time.Minute, SpikeMultiplier: 3}, 50)
	late := now.Add(-30 * time.Minute)
	first, err := service.Ingest(ctx, Batch{BatchID: "batch-1", ReceivedAt: now, Samples: []Sample{{DeviceCode: "D1", SchemaID: "pm10", Value: 200, SampledAt: late}}})
	if err != nil {
		t.Fatal(err)
	}
	if first.Accepted != 1 || !first.Items[0].Late {
		t.Fatalf("first=%+v", first)
	}
	second, err := service.Ingest(ctx, Batch{BatchID: "batch-2", ReceivedAt: now, Samples: []Sample{{DeviceCode: "D1", SchemaID: "pm10", Value: 200, SampledAt: late}}})
	if err != nil {
		t.Fatal(err)
	}
	if second.Duplicates != 1 {
		t.Fatalf("second=%+v", second)
	}
	alerts, err := store.RangeAlerts(ctx, "s1", now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 0 {
		t.Fatalf("late data generated alerts: %+v", alerts)
	}
}

func mustSeed(t *testing.T, ctx context.Context, store *memory.Store, now time.Time) {
	t.Helper()
	siteEntity, _ := site.New("s1", "site", "Asia/Shanghai", "unit", now)
	zone, _ := site.NewZone("z1", "s1", "zone", "")
	point, _ := site.NewMonitoringPoint("p1", "s1", "z1", "point", 120, 30)
	_ = store.SaveSite(ctx, siteEntity)
	_ = store.SaveZone(ctx, zone)
	_ = store.SavePoint(ctx, point)
	value, _ := device.New("d1", "D1", "sim", "s1", "p1", "gate", device.NetworkConfig{Host: "sim", Port: 1883, Protocol: "mqtt"})
	_ = value.MarkSeen(now)
	_ = store.Save(ctx, value)
	schema, _ := telemetry.NewSchema("pm10", telemetry.MetricPM10, "ug/m3", time.Minute, 0, 2000, time.Minute)
	_ = store.SaveSchema(ctx, schema)
	version, _ := rule.NewVersion("r1", "s1", "rule", "Asia/Shanghai", "u1", 1, []rule.Condition{{Metric: telemetry.MetricPM10, Operator: rule.OperatorAtLeast, Value: 150}}, 0, 10*time.Minute, 10*time.Minute, now.Add(-time.Hour), now)
	_ = version.Activate(now)
	_ = store.SaveRule(ctx, version)
}
