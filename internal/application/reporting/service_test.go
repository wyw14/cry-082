package reporting

import (
	"context"
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

type fileCapture struct {
	mime    string
	payload []byte
}

func (f *fileCapture) Put(ctx context.Context, name, mime string, payload []byte) (string, string, error) {
	f.mime = mime
	f.payload = append([]byte(nil), payload...)
	return "file-1", strings.Repeat("a", 64), nil
}

func TestRegulatoryExportChecksOwnershipAndNotifies(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC)
	store := memory.New()
	siteEntity, _ := site.New("s1", "site", "Asia/Shanghai", "unit", now)
	if err := store.SaveSite(ctx, siteEntity); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMembership(ctx, site.Membership{UserID: "u1", SiteID: "s1", Role: site.RoleSupervisor}); err != nil {
		t.Fatal(err)
	}
	daily, _ := report.NewDaily("r1", "s1", "2026-08-23", "Asia/Shanghai", []report.DailyMetric{{PointID: "p1", Metric: "pm10", AcceptedSamples: 1}}, 1, 2, now)
	if err := store.SaveDaily(ctx, daily); err != nil {
		t.Fatal(err)
	}
	files := &fileCapture{}
	notifier := notification.NewLocal()
	service := New(store, files, store, notifier, store, outbox.NewStore(), memory.TransactionManager{}, clockpkg.NewManual(now), &idempotency.Generator{})
	created, err := service.Export(ctx, "s1", "u1", "csv", []string{"r1"})
	if err != nil {
		t.Fatal(err)
	}
	if created.FileID != "file-1" || files.mime != "text/csv" || len(files.payload) == 0 || len(notifier.Messages()) != 1 {
		t.Fatalf("created=%+v mime=%s notifications=%d", created, files.mime, len(notifier.Messages()))
	}
}
