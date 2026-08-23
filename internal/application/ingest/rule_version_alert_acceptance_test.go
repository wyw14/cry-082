package ingest

import (
	"context"
	"testing"
	"time"

	"github.com/wyw14/cry-082/internal/domain/alert"
	"github.com/wyw14/cry-082/internal/domain/device"
	"github.com/wyw14/cry-082/internal/domain/rule"
	"github.com/wyw14/cry-082/internal/domain/site"
	"github.com/wyw14/cry-082/internal/domain/telemetry"
	clockpkg "github.com/wyw14/cry-082/internal/platform/clock"
	"github.com/wyw14/cry-082/internal/platform/idempotency"
	"github.com/wyw14/cry-082/internal/platform/outbox"
	"github.com/wyw14/cry-082/internal/repository/memory"
)

func TestNewRuleVersionStartsIndependentAlert(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC)
	store := memory.New()
	siteEntity, _ := site.New("site-1", "site", "Asia/Shanghai", "unit", now)
	zone, _ := site.NewZone("zone-1", "site-1", "zone", "")
	point, _ := site.NewMonitoringPoint("point-1", "site-1", "zone-1", "point", 120, 30)
	_ = store.SaveSite(ctx, siteEntity)
	_ = store.SaveZone(ctx, zone)
	_ = store.SavePoint(ctx, point)
	deviceEntity, _ := device.New("device-1", "D1", "sim", "site-1", "point-1", "gate", device.NetworkConfig{Host: "sim", Port: 1883, Protocol: "mqtt"})
	_ = deviceEntity.MarkSeen(now)
	_ = store.Save(ctx, deviceEntity)
	schema, _ := telemetry.NewSchema("pm10", telemetry.MetricPM10, "ug/m3", time.Minute, 0, 2000, time.Minute)
	_ = store.SaveSchema(ctx, schema)
	version, _ := rule.NewVersion("dust", "site-1", "dust limit", "Asia/Shanghai", "supervisor-1", 2, []rule.Condition{{Metric: telemetry.MetricPM10, Operator: rule.OperatorAtLeast, Value: 150}}, 0, 10*time.Minute, time.Minute, now.Add(-time.Hour), now.Add(-time.Hour))
	_ = version.Activate(now)
	_ = store.SaveRule(ctx, version)
	previous, _ := alert.New("old-alert", "site-1", "point-1", "device-1", alert.KindEnvironmentalExceedance, "dust", 1, "site-1:point-1:dust", now.Add(-5*time.Minute))
	_ = store.SaveAlert(ctx, previous)
	service := New(Dependencies{Devices: store, Schemas: store, Observations: store, Rules: store, Evaluations: store, Alerts: store, Outbox: outbox.NewStore(), Transactions: memory.TransactionManager{}, Clock: clockpkg.NewManual(now), IDs: &idempotency.Generator{}}, telemetry.QualityPolicy{FutureTolerance: time.Minute, LateAfter: 10 * time.Minute, SpikeMultiplier: 3}, 50)
	if _, err := service.Ingest(ctx, Batch{BatchID: "version-2-batch", ReceivedAt: now, Samples: []Sample{{DeviceCode: "D1", SchemaID: "pm10", Value: 200, SampledAt: now}}}); err != nil {
		t.Fatal(err)
	}
	alerts, err := store.RangeAlerts(ctx, "site-1", now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 2 {
		t.Fatalf("new rule version did not create an independent alert: %+v", alerts)
	}
}
