package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap/zaptest"
	"golang.org/x/crypto/bcrypt"

	alertapp "github.com/wyw14/cry-082/internal/application/alerts"
	"github.com/wyw14/cry-082/internal/application/artifacts"
	"github.com/wyw14/cry-082/internal/application/authn"
	deviceapp "github.com/wyw14/cry-082/internal/application/device"
	"github.com/wyw14/cry-082/internal/application/ingest"
	"github.com/wyw14/cry-082/internal/application/monitor"
	"github.com/wyw14/cry-082/internal/application/reporting"
	ruleapp "github.com/wyw14/cry-082/internal/application/rules"
	"github.com/wyw14/cry-082/internal/domain/auth"
	"github.com/wyw14/cry-082/internal/domain/device"
	"github.com/wyw14/cry-082/internal/domain/report"
	"github.com/wyw14/cry-082/internal/domain/site"
	"github.com/wyw14/cry-082/internal/domain/telemetry"
	"github.com/wyw14/cry-082/internal/middleware"
	clockpkg "github.com/wyw14/cry-082/internal/platform/clock"
	"github.com/wyw14/cry-082/internal/platform/files"
	"github.com/wyw14/cry-082/internal/platform/idempotency"
	metricspkg "github.com/wyw14/cry-082/internal/platform/metrics"
	"github.com/wyw14/cry-082/internal/platform/notification"
	"github.com/wyw14/cry-082/internal/platform/outbox"
	"github.com/wyw14/cry-082/internal/platform/security"
	"github.com/wyw14/cry-082/internal/repository/memory"
)

func TestTelemetryEndpointReturnsStableEnvelope(t *testing.T) {
	router, now := testRouter(t)
	body := `{"batch_id":"http-batch","samples":[{"device_code":"D1","schema_id":"pm10","value":120,"sampled_at":"` + now.Format(time.RFC3339) + `"}]}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/batches", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Request-ID", "request-http-0001")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["request_id"] != "request-http-0001" {
		t.Fatalf("response=%v", response)
	}
}

func TestValidationErrorContainsCodeFieldsAndRequestID(t *testing.T) {
	router, _ := testRouter(t)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/batches", strings.NewReader(`{"batch_id":"","samples":[]}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Request-ID", "request-http-0002")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != "VALIDATION_FAILED" || response.RequestID != "request-http-0002" || len(response.FieldErrors) == 0 {
		t.Fatalf("response=%+v", response)
	}
}

func TestAlertListRejectsSortOutsideWhitelist(t *testing.T) {
	router, _ := testRouter(t)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/sites/s1/alerts?sort=rule_id&page=1&page_size=20", nil)
	request.Header.Set("X-Request-ID", "request-http-0003")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != "BUSINESS_RULE_VIOLATION" {
		t.Fatalf("response=%+v", response)
	}
}

func TestLoginBearerAndMetrics(t *testing.T) {
	router, _ := testRouterMode(t, false)
	unauthorized := httptest.NewRecorder()
	router.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/v1/sites/s1/alerts?page=1&page_size=20", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", unauthorized.Code, unauthorized.Body.String())
	}
	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"demo.supervisor","password":"DustDemo!2026"}`))
	login.Header.Set("Content-Type", "application/json")
	loginResult := httptest.NewRecorder()
	router.ServeHTTP(loginResult, login)
	if loginResult.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", loginResult.Code, loginResult.Body.String())
	}
	var loginBody struct {
		Data authn.TokenPair `json:"data"`
	}
	if err := json.Unmarshal(loginResult.Body.Bytes(), &loginBody); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/sites/s1/alerts?page=1&page_size=20", nil)
	request.Header.Set("Authorization", "Bearer "+loginBody.Data.AccessToken)
	protected := httptest.NewRecorder()
	router.ServeHTTP(protected, request)
	if protected.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", protected.Code, protected.Body.String())
	}
	metricsRequest := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsResult := httptest.NewRecorder()
	router.ServeHTTP(metricsResult, metricsRequest)
	if metricsResult.Code != http.StatusOK || !strings.Contains(metricsResult.Body.String(), "dust_http_requests_total") || !strings.Contains(metricsResult.Body.String(), "/api/v1/sites/:site_id/alerts") {
		t.Fatalf("status=%d body=%s", metricsResult.Code, metricsResult.Body.String())
	}
}

func TestAuthorizedFileUploadAndDownload(t *testing.T) {
	router, _ := testRouter(t)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="calibration.json"`)
	header.Set("Content-Type", "application/json")
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"device_id":"d1","offset":1.25}`)
	if _, err := part.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("purpose", "maintenance-certificate"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	upload := httptest.NewRequest(http.MethodPost, "/api/v1/sites/s1/files", &body)
	upload.Header.Set("Content-Type", writer.FormDataContentType())
	uploadResult := httptest.NewRecorder()
	router.ServeHTTP(uploadResult, upload)
	if uploadResult.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", uploadResult.Code, uploadResult.Body.String())
	}
	var uploaded struct {
		Data struct {
			ID       string `json:"id"`
			Checksum string `json:"checksum"`
		} `json:"data"`
	}
	if err := json.Unmarshal(uploadResult.Body.Bytes(), &uploaded); err != nil {
		t.Fatal(err)
	}
	download := httptest.NewRequest(http.MethodGet, "/api/v1/sites/s1/files/"+uploaded.Data.ID, nil)
	downloadResult := httptest.NewRecorder()
	router.ServeHTTP(downloadResult, download)
	if downloadResult.Code != http.StatusOK || !bytes.Equal(downloadResult.Body.Bytes(), payload) || downloadResult.Header().Get("X-Content-SHA256") != uploaded.Data.Checksum || strings.Contains(downloadResult.Header().Get("Content-Disposition"), "\\") {
		t.Fatalf("status=%d headers=%v body=%s", downloadResult.Code, downloadResult.Header(), downloadResult.Body.String())
	}
}

func testRouter(t *testing.T) (http.Handler, time.Time) {
	return testRouterMode(t, true)
}

func testRouterMode(t *testing.T, allowDemo bool) (http.Handler, time.Time) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC)
	store := memory.New()
	siteEntity, _ := site.New("s1", "site", "Asia/Shanghai", "unit", now)
	zone, _ := site.NewZone("z1", "s1", "zone", "")
	point, _ := site.NewMonitoringPoint("p1", "s1", "z1", "point", 120, 30)
	_ = store.SaveSite(ctx, siteEntity)
	_ = store.SaveZone(ctx, zone)
	_ = store.SavePoint(ctx, point)
	_ = store.SaveMembership(ctx, site.Membership{UserID: "demo-supervisor", SiteID: "s1", Role: site.RoleAdministrator})
	hash, _ := bcrypt.GenerateFromPassword([]byte("DustDemo!2026"), bcrypt.MinCost)
	user, _ := auth.NewUser("demo-supervisor", "demo.supervisor", "演示监管员", hash)
	_ = store.SaveUser(ctx, user)
	value, _ := device.New("d1", "D1", "sim", "s1", "p1", "gate", device.NetworkConfig{Host: "sim", Port: 1883, Protocol: "mqtt"})
	_ = value.MarkSeen(now)
	_ = store.Save(ctx, value)
	schema, _ := telemetry.NewSchema("pm10", telemetry.MetricPM10, "ug/m3", time.Minute, 0, 2000, time.Minute)
	_ = store.SaveSchema(ctx, schema)
	daily, _ := report.NewDaily("daily-1", "s1", "2026-08-23", "Asia/Shanghai", []report.DailyMetric{{PointID: "p1", Metric: "pm10", AcceptedSamples: 1}}, 0, 0, now)
	_ = store.SaveDaily(ctx, daily)
	clock := clockpkg.NewManual(now)
	ids := &idempotency.Generator{}
	events := outbox.NewStore()
	validate := validator.New()
	ingestService := ingest.New(ingest.Dependencies{Devices: store, Schemas: store, Observations: store, Rules: store, Evaluations: store, Alerts: store, Outbox: events, Transactions: memory.TransactionManager{}, Clock: clock, IDs: ids}, telemetry.QualityPolicy{FutureTolerance: time.Minute, LateAfter: 10 * time.Minute, SpikeMultiplier: 3}, 50)
	transactionManager := memory.TransactionManager{}
	fileStore, err := files.NewLocal(t.TempDir(), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	tokens, err := security.NewAccessTokens("router-test-access-token-key-082-0001", clock.Now)
	if err != nil {
		t.Fatal(err)
	}
	authService := authn.New(store, tokens, clock, ids, 15*time.Minute, 24*time.Hour)
	verifier := middleware.VerifyFunc(func(ctx context.Context, raw string) (middleware.Claims, error) {
		claims, err := tokens.VerifyAccessToken(ctx, raw)
		return middleware.Claims{Subject: claims.Subject}, err
	})
	artifactService := artifacts.New(store, store, store, fileStore, transactionManager, clock, ids)
	return NewRouter(Dependencies{Logger: zaptest.NewLogger(t), Validator: validate, Health: ready{}, Ingest: NewIngestHandler(ingestService, validate), Alerts: NewAlertHandler(alertapp.New(store, store, store, events, transactionManager, clock, ids), validate), Devices: NewDeviceHandler(deviceapp.New(store, store, store, transactionManager, clock, ids), validate), Rules: NewRuleHandler(ruleapp.New(store, store, store, events, transactionManager, clock, ids), validate), Monitor: NewMonitorHandler(monitor.New(store, store)), Reports: NewReportHandler(reporting.New(store, fileStore, store, notification.NewLocal(), store, events, transactionManager, clock, ids), validate), Auth: NewAuthHandler(authService, validate), Artifacts: NewArtifactHandler(artifactService, 1<<20), AccessVerifier: verifier, AllowDemoActor: allowDemo, Metrics: metricspkg.NewRegistry(now)}), now
}

type ready struct{}

func (ready) Ready(context.Context) error { return nil }
