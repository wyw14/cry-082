package bootstrap

import (
	"context"
	"testing"
	"time"

	"github.com/wyw14/cry-082/internal/config"
)

func TestSeededReportIsVisibleThroughReportingService(t *testing.T) {
	cfg := config.Config{Environment: "test", HTTPAddress: ":0", Timezone: "Asia/Shanghai", AccessTokenTTL: 15 * time.Minute, AccessTokenKey: "test-access-token-key-with-32-bytes", RefreshTokenTTL: time.Hour, RequestTimeout: time.Second, ShutdownTimeout: time.Second, FileRoot: t.TempDir(), MaximumFileSize: 1 << 20, SeedDemo: true, LogLevel: "error"}
	app, runtime, err := BuildWithRuntime(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Logger.Sync()
	exported, err := runtime.Services.Reports.Export(context.Background(), "site-demo", "demo-supervisor", "csv", []string{"report-demo-today"})
	if err != nil {
		t.Fatalf("export seeded report: %v", err)
	}
	if len(exported.ReportIDs) != 1 || exported.ReportIDs[0] != "report-demo-today" {
		t.Fatalf("unexpected export: %+v", exported)
	}
}
