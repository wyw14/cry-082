package integration

import (
	"context"
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/wyw14/cry-082/internal/domain/audit"
	"github.com/wyw14/cry-082/internal/domain/site"
	"github.com/wyw14/cry-082/internal/domain/telemetry"
	"github.com/wyw14/cry-082/internal/repository/postgres"
)

func TestPostgresSchemaRoundTrip(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	store, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	schema, err := telemetry.NewSchema("integration-pm10", telemetry.MetricPM10, "ug/m3", time.Minute, 0, 2000, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveSchema(ctx, schema); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.FindSchema(ctx, schema.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Metric != schema.Metric || loaded.SamplingPeriod != schema.SamplingPeriod {
		t.Fatalf("loaded=%+v", loaded)
	}
}

func TestPostgresTopologyAuditAndOutboxRoundTrip(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	store, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	suffix := strconv.FormatInt(time.Now().UnixNano(), 36)
	now := time.Now().UTC().Truncate(time.Millisecond)
	siteEntity, err := site.New("site-"+suffix, "集成测试工地", "Asia/Shanghai", "测试责任单位", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveSite(ctx, siteEntity); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveSite(ctx, siteEntity); !errors.Is(err, postgres.ErrStaleTopologyVersion) {
		t.Fatalf("same topology version error=%v", err)
	}
	zone, err := site.NewZone("zone-"+suffix, siteEntity.ID, "测试区域", "拓扑持久化")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveZone(ctx, zone); err != nil {
		t.Fatal(err)
	}
	point, err := site.NewMonitoringPoint("point-"+suffix, siteEntity.ID, zone.ID, "测试测点", 121.48, 31.23)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SavePoint(ctx, point); err != nil {
		t.Fatal(err)
	}
	membership := site.Membership{UserID: "user-" + suffix, SiteID: siteEntity.ID, Role: site.RoleSupervisor}
	if err := store.SaveMembership(ctx, membership); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Membership(ctx, membership.UserID, membership.SiteID)
	if err != nil || loaded.Role != membership.Role {
		t.Fatalf("membership=%+v error=%v", loaded, err)
	}

	entry, err := audit.New("audit-"+suffix, siteEntity.ID, membership.UserID, "integration", "topology.verified", "site", siteEntity.ID, "verify persistence", "request-"+suffix, nil, map[string]string{"site": siteEntity.ID}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AppendAudit(ctx, entry); err != nil {
		t.Fatal(err)
	}
	if err := store.Enqueue(ctx, "topology.verified", siteEntity.ID, map[string]string{"site_id": siteEntity.ID}); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresTransactionRollback(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	store, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	unit, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	txctx := unit.Bind(ctx)
	schema, err := telemetry.NewSchema("rollback-pm25", telemetry.MetricPM25, "ug/m3", time.Minute, 0, 1000, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveSchema(txctx, schema); err != nil {
		t.Fatal(err)
	}
	if err := unit.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := store.FindSchema(ctx, schema.ID); err == nil {
		t.Fatal("rolled back schema remained visible")
	}
}
