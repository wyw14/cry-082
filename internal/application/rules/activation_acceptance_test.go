package rules

import (
	"context"
	"testing"
	"time"

	"github.com/wyw14/cry-082/internal/domain/rule"
	"github.com/wyw14/cry-082/internal/domain/site"
	"github.com/wyw14/cry-082/internal/domain/telemetry"
	clockpkg "github.com/wyw14/cry-082/internal/platform/clock"
	"github.com/wyw14/cry-082/internal/platform/idempotency"
	"github.com/wyw14/cry-082/internal/platform/outbox"
	"github.com/wyw14/cry-082/internal/repository/memory"
)

func TestActivatingDraftVersionMakesItActive(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC)
	store := memory.New()
	if err := store.SaveMembership(ctx, site.Membership{UserID: "supervisor-1", SiteID: "site-1", Role: site.RoleSupervisor}); err != nil {
		t.Fatal(err)
	}
	version, err := rule.NewVersion("dust", "site-1", "dust limit", "Asia/Shanghai", "supervisor-1", 2, []rule.Condition{{Metric: telemetry.MetricPM10, Operator: rule.OperatorAtLeast, Value: 150}}, 0, 10*time.Minute, time.Minute, now.Add(-time.Minute), now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveRule(ctx, version); err != nil {
		t.Fatal(err)
	}
	service := New(store, store, store, outbox.NewStore(), memory.TransactionManager{}, clockpkg.NewManual(now), &idempotency.Generator{})
	activated, err := service.Activate(ctx, "site-1", "dust", "supervisor-1", "publish current limits", "request-1", 2)
	if err != nil {
		t.Fatal(err)
	}
	if activated.Status != rule.StatusActive {
		t.Fatalf("activated version has status %q", activated.Status)
	}
}
