package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	alertapp "github.com/wyw14/cry-082/internal/application/alerts"
	"github.com/wyw14/cry-082/internal/application/artifacts"
	"github.com/wyw14/cry-082/internal/application/authn"
	deviceapp "github.com/wyw14/cry-082/internal/application/device"
	"github.com/wyw14/cry-082/internal/application/ingest"
	maintenanceapp "github.com/wyw14/cry-082/internal/application/maintenance"
	"github.com/wyw14/cry-082/internal/application/monitor"
	"github.com/wyw14/cry-082/internal/application/reporting"
	ruleapp "github.com/wyw14/cry-082/internal/application/rules"
	siteapp "github.com/wyw14/cry-082/internal/application/site"
	"github.com/wyw14/cry-082/internal/config"
	"github.com/wyw14/cry-082/internal/domain/auth"
	"github.com/wyw14/cry-082/internal/domain/device"
	"github.com/wyw14/cry-082/internal/domain/report"
	"github.com/wyw14/cry-082/internal/domain/rule"
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
	"github.com/wyw14/cry-082/internal/platform/transaction"
	"github.com/wyw14/cry-082/internal/repository/memory"
	"github.com/wyw14/cry-082/internal/repository/postgres"
	httpapi "github.com/wyw14/cry-082/internal/transport/http"
)

type App struct {
	Config   config.Config
	Logger   *zap.Logger
	Server   *http.Server
	postgres *postgres.Store
}

type Services struct {
	Ingest      *ingest.Service
	Alerts      *alertapp.Service
	Devices     *deviceapp.Service
	Rules       *ruleapp.Service
	Monitor     *monitor.Service
	Reports     *reporting.Service
	Auth        *authn.Service
	Files       *artifacts.Service
	Maintenance *maintenanceapp.Service
	Sites       *siteapp.Service
}

type eventSink interface {
	Enqueue(context.Context, string, string, any) error
}

type Runtime struct {
	Store    businessStore
	Events   eventSink
	Services Services
}

type repositories struct {
	store  businessStore
	tx     transaction.Manager
	health httpapi.Health
}

type businessStore interface {
	siteapp.Repository
	siteapp.AuditRepository
	deviceapp.Repository
	deviceapp.AccessRepository
	ingest.MeasurementCatalog
	ingest.ObservationJournal
	ingest.RuleCatalog
	ingest.EvaluationJournal
	ingest.IncidentSink
	alertapp.Repository
	ruleapp.Repository
	monitor.ObservationRepository
	monitor.AlertRepository
	reporting.Repository
	reporting.AccessRepository
	artifacts.Repository
	authn.Repository
	maintenanceapp.Repository
	SaveMembership(context.Context, site.Membership) error
	SaveSchema(context.Context, telemetry.Schema) error
	SaveDaily(context.Context, report.DailyReport) error
}

func Build(ctx context.Context, cfg config.Config) (*App, error) {
	app, _, err := BuildWithRuntime(ctx, cfg)
	return app, err
}

func BuildWithRuntime(ctx context.Context, cfg config.Config) (*App, *Runtime, error) {
	loggerConfig := zap.NewProductionConfig()
	if cfg.Environment != "production" {
		loggerConfig = zap.NewDevelopmentConfig()
	}
	logger, err := loggerConfig.Build()
	if err != nil {
		return nil, nil, err
	}
	repositories, pg, err := buildRepositories(ctx, cfg, logger)
	if err != nil {
		_ = logger.Sync()
		return nil, nil, err
	}
	clock := clockpkg.NewSystem()
	ids := &idempotency.Generator{}
	validate := validator.New()
	var events eventSink = outbox.NewStore()
	if pg != nil {
		events = pg
	}
	notifier := notification.NewLocal()
	fileStore, err := files.NewLocal(cfg.FileRoot, cfg.MaximumFileSize)
	if err != nil {
		return nil, nil, err
	}
	if cfg.SeedDemo {
		if err := seed(ctx, repositories.store, clock, ids); err != nil {
			return nil, nil, fmt.Errorf("seed demo: %w", err)
		}
	}
	ingestService := ingest.New(ingest.Dependencies{Devices: repositories.store, Schemas: repositories.store, Observations: repositories.store, Rules: repositories.store, Evaluations: repositories.store, Alerts: repositories.store, Outbox: events, Transactions: repositories.tx, Clock: clock, IDs: ids}, telemetry.QualityPolicy{FutureTolerance: 2 * time.Minute, LateAfter: 10 * time.Minute, SpikeMultiplier: 2.5}, 500)
	alertService := alertapp.New(repositories.store, repositories.store, repositories.store, events, repositories.tx, clock, ids)
	deviceService := deviceapp.New(repositories.store, repositories.store, repositories.store, repositories.tx, clock, ids)
	ruleService := ruleapp.New(repositories.store, repositories.store, repositories.store, events, repositories.tx, clock, ids)
	monitorService := monitor.New(repositories.store, repositories.store)
	reportService := reporting.New(repositories.store, fileStore, repositories.store, notifier, repositories.store, events, repositories.tx, clock, ids)
	siteService := siteapp.New(repositories.store, repositories.store, repositories.tx, clock, ids)
	maintenanceService := maintenanceapp.New(repositories.store, repositories.store, repositories.store, repositories.store, repositories.tx, clock, ids)
	fileService := artifacts.New(repositories.store, repositories.store, repositories.store, fileStore, repositories.tx, clock, ids)
	tokens, err := security.NewAccessTokens(cfg.AccessTokenKey, clock.Now)
	if err != nil {
		return nil, nil, err
	}
	authService := authn.New(repositories.store, tokens, clock, ids, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	accessVerifier := middleware.VerifyFunc(func(ctx context.Context, raw string) (middleware.Claims, error) {
		claims, err := tokens.VerifyAccessToken(ctx, raw)
		return middleware.Claims{Subject: claims.Subject}, err
	})
	metricRegistry := metricspkg.NewRegistry(clock.Now())
	router := httpapi.NewRouter(httpapi.Dependencies{Logger: logger, Validator: validate, Health: repositories.health, Ingest: httpapi.NewIngestHandler(ingestService, validate), Alerts: httpapi.NewAlertHandler(alertService, validate), Devices: httpapi.NewDeviceHandler(deviceService, validate), Rules: httpapi.NewRuleHandler(ruleService, validate), Monitor: httpapi.NewMonitorHandler(monitorService), Reports: httpapi.NewReportHandler(reportService, validate), Auth: httpapi.NewAuthHandler(authService, validate), Artifacts: httpapi.NewArtifactHandler(fileService, cfg.MaximumFileSize), AccessVerifier: accessVerifier, AllowDemoActor: cfg.Environment != "production", Metrics: metricRegistry, AllowedOrigins: []string{"http://localhost:5173", "http://127.0.0.1:5173"}})
	server := &http.Server{Addr: cfg.HTTPAddress, Handler: router, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	app := &App{Config: cfg, Logger: logger, Server: server, postgres: pg}
	runtime := &Runtime{Store: repositories.store, Events: events, Services: Services{Ingest: ingestService, Alerts: alertService, Devices: deviceService, Rules: ruleService, Monitor: monitorService, Reports: reportService, Auth: authService, Files: fileService, Maintenance: maintenanceService, Sites: siteService}}
	return app, runtime, nil
}

func buildRepositories(ctx context.Context, cfg config.Config, logger *zap.Logger) (repositories, *postgres.Store, error) {
	store := memory.New()
	result := repositories{store: store, tx: memory.TransactionManager{}, health: alwaysReady{}}
	if cfg.DatabaseURL == "" {
		logger.Warn("DATABASE_URL not configured; deterministic memory adapter is active")
		return result, nil, nil
	}
	pg, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return repositories{}, nil, err
	}
	if err := pg.Migrate(ctx); err != nil {
		pg.Close()
		return repositories{}, nil, err
	}
	result.store = pg
	result.tx = pg
	result.health = pg
	return result, pg, nil
}

func (a *App) Close(ctx context.Context) error {
	var errs []error
	if a.Server != nil {
		if err := a.Server.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if a.postgres != nil {
		a.postgres.Close()
	}
	if a.Logger != nil {
		if err := a.Logger.Sync(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

type alwaysReady struct{}

func (alwaysReady) Ready(context.Context) error { return nil }

func seed(ctx context.Context, store businessStore, clock clockpkg.System, ids *idempotency.Generator) error {
	now := clock.Now()
	siteEntity, err := site.New("site-demo", "滨江综合施工现场", "Asia/Shanghai", "滨江建设责任单位", now)
	if err != nil {
		return err
	}
	if err := store.SaveSite(ctx, siteEntity); err != nil {
		return err
	}
	zone, err := site.NewZone("zone-gate", siteEntity.ID, "东门施工区", "车辆与土方作业")
	if err != nil {
		return err
	}
	if err := store.SaveZone(ctx, zone); err != nil {
		return err
	}
	point, err := site.NewMonitoringPoint("point-east-gate", siteEntity.ID, zone.ID, "东门监测点", 121.486, 31.239)
	if err != nil {
		return err
	}
	if err := store.SavePoint(ctx, point); err != nil {
		return err
	}
	for _, membership := range []site.Membership{{UserID: "demo-admin", SiteID: siteEntity.ID, Role: site.RoleAdministrator}, {UserID: "demo-supervisor", SiteID: siteEntity.ID, Role: site.RoleSupervisor}, {UserID: "demo-maintainer", SiteID: siteEntity.ID, Role: site.RoleMaintainer}} {
		if err := store.SaveMembership(ctx, membership); err != nil {
			return err
		}
	}
	network := device.NetworkConfig{Host: "simulator.local", Port: 1883, Protocol: "mqtt"}
	deviceEntity, err := device.New("device-demo-001", "DUST-EAST-001", "LocalSim-X1", siteEntity.ID, point.ID, "东门围挡内侧", network)
	if err != nil {
		return err
	}
	if err := deviceEntity.MarkSeen(now); err != nil {
		return err
	}
	if err := store.Save(ctx, deviceEntity); err != nil {
		return err
	}
	schemas := []telemetry.Schema{}
	for _, spec := range []struct {
		id       string
		metric   telemetry.Metric
		unit     string
		min, max float64
	}{{"schema-pm25", telemetry.MetricPM25, "ug/m3", 0, 1000}, {"schema-pm10", telemetry.MetricPM10, "ug/m3", 0, 2000}, {"schema-noise", telemetry.MetricNoise, "dB", 20, 140}, {"schema-temperature", telemetry.MetricTemperature, "C", -50, 80}, {"schema-humidity", telemetry.MetricHumidity, "%", 0, 100}, {"schema-wind-speed", telemetry.MetricWindSpeed, "m/s", 0, 80}} {
		schema, err := telemetry.NewSchema(spec.id, spec.metric, spec.unit, time.Minute, spec.min, spec.max, 2*time.Minute)
		if err != nil {
			return err
		}
		schemas = append(schemas, schema)
		if err := store.SaveSchema(ctx, schema); err != nil {
			return err
		}
	}
	version, err := rule.NewVersion("rule-dust-combined", siteEntity.ID, "扬尘与静风组合告警", "Asia/Shanghai", "demo-supervisor", 1, []rule.Condition{{Metric: telemetry.MetricPM10, Operator: rule.OperatorAtLeast, Value: 150}}, 0, 15*time.Minute, 10*time.Minute, now.Add(-time.Hour), now)
	if err != nil {
		return err
	}
	if err := version.Activate(now); err != nil {
		return err
	}
	if err := store.SaveRule(ctx, version); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("DustDemo!2026"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user, err := auth.NewUser("demo-supervisor", "demo.supervisor", "演示监管员", hash)
	if err != nil {
		return err
	}
	if err := store.SaveUser(ctx, user); err != nil {
		return err
	}
	localDate, err := clockpkg.LocalDate(now, siteEntity.Timezone)
	if err != nil {
		return err
	}
	daily, err := report.NewDaily("report-demo-today", siteEntity.ID, localDate, siteEntity.Timezone, []report.DailyMetric{{PointID: point.ID, Metric: "pm10", Maximum: 92, Average: 48, AcceptedSamples: 1200, SuspectSamples: 2}}, 0, 0, now)
	if err != nil {
		return err
	}
	return store.SaveDaily(ctx, daily)
}
