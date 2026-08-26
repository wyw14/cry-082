package reporting

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/wyw14/cry-082/internal/domain/report"
	"github.com/wyw14/cry-082/internal/domain/site"
	clockpkg "github.com/wyw14/cry-082/internal/platform/clock"
	"github.com/wyw14/cry-082/internal/platform/idempotency"
	"github.com/wyw14/cry-082/internal/platform/notification"
	"github.com/wyw14/cry-082/internal/platform/outbox"
	"github.com/wyw14/cry-082/internal/repository/memory"
)

type failingExportRepository struct{ *memory.Store }

func (r failingExportRepository) SaveExport(context.Context, report.Export) error {
	return errors.New("export persistence failed")
}

type exportFileStore struct{}

func (exportFileStore) Put(context.Context, string, string, []byte) (string, string, error) {
	return "file-1", strings.Repeat("a", 64), nil
}

func TestFailedExportTransactionDoesNotNotifyRecipient(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC)
	store := memory.New()
	siteEntity, _ := site.New("site-1", "site", "Asia/Shanghai", "unit", now)
	if err := store.SaveSite(ctx, siteEntity); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMembership(ctx, site.Membership{UserID: "supervisor-1", SiteID: "site-1", Role: site.RoleSupervisor}); err != nil {
		t.Fatal(err)
	}
	daily, _ := report.NewDaily("daily-1", "site-1", "2026-08-23", "Asia/Shanghai", []report.DailyMetric{{PointID: "point-1", Metric: "pm10", AcceptedSamples: 1}}, 0, 0, now)
	if err := store.SaveDaily(ctx, daily); err != nil {
		t.Fatal(err)
	}
	notifier := notification.NewLocal()
	service := New(failingExportRepository{Store: store}, exportFileStore{}, store, notifier, store, outbox.NewStore(), memory.TransactionManager{}, clockpkg.NewManual(now), &idempotency.Generator{})
	if _, err := service.Export(ctx, "site-1", "supervisor-1", "csv", []string{"daily-1"}); err == nil {
		t.Fatal("expected export persistence failure")
	}
	if got := len(notifier.Messages()); got != 0 {
		t.Fatalf("failed export sent %d notifications", got)
	}
}
