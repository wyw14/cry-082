package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"

	"github.com/wyw14/cry-082/internal/middleware"
	"github.com/wyw14/cry-082/internal/platform/metrics"
)

type Health interface{ Ready(context.Context) error }
type Dependencies struct {
	Logger         *zap.Logger
	Validator      *validator.Validate
	Health         Health
	Ingest         *IngestHandler
	Alerts         *AlertHandler
	Devices        *DeviceHandler
	Rules          *RuleHandler
	Monitor        *MonitorHandler
	Reports        *ReportHandler
	Auth           *AuthHandler
	Artifacts      *ArtifactHandler
	AccessVerifier middleware.AccessVerifier
	AllowDemoActor bool
	Metrics        *metrics.Registry
	AllowedOrigins []string
}

func NewRouter(deps Dependencies) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	requestRuntime := middleware.NewRequestRuntime(deps.Logger, deps.Metrics, deps.AllowedOrigins, 30, 60)
	router.Use(requestRuntime.Handle())
	router.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	router.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if deps.Health != nil {
			if err := deps.Health.Ready(ctx); err != nil {
				fail(c, http.StatusServiceUnavailable, "NOT_READY", "依赖尚未就绪", err)
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	router.GET("/metrics", func(c *gin.Context) {
		var body bytes.Buffer
		if deps.Metrics == nil {
			c.String(http.StatusServiceUnavailable, "metrics unavailable\n")
			return
		}
		if err := deps.Metrics.WritePrometheus(&body); err != nil {
			c.String(http.StatusInternalServerError, "metrics unavailable\n")
			return
		}
		c.Data(http.StatusOK, "text/plain; version=0.0.4; charset=utf-8", body.Bytes())
	})
	authRoutes := router.Group("/api/v1/auth")
	{
		authRoutes.POST("/login", deps.Auth.Login)
		authRoutes.POST("/refresh", deps.Auth.Refresh)
	}
	api := router.Group("/api/v1", middleware.RequireIdentity(deps.AccessVerifier, deps.AllowDemoActor))
	{
		api.POST("/telemetry/batches", deps.Ingest.Batch)
		api.GET("/sites/:site_id/alerts", deps.Alerts.List)
		api.PATCH("/alerts/:id", deps.Alerts.Transition)
		api.PATCH("/devices/:id/status", deps.Devices.Transition)
		api.POST("/sites/:site_id/rules/versions", deps.Rules.CreateVersion)
		api.PATCH("/sites/:site_id/rules/:rule_id/versions/:version/activation", deps.Rules.ActivateVersion)
		api.POST("/sites/:site_id/rules/:rule_id/recalculations", deps.Rules.RequestRecalculation)
		api.GET("/sites/:site_id/dashboard", deps.Monitor.Dashboard)
		api.GET("/sites/:site_id/trends", deps.Monitor.Trend)
		api.POST("/sites/:site_id/regulatory-exports", deps.Reports.Export)
		api.POST("/sites/:site_id/files", deps.Artifacts.Upload)
		api.GET("/sites/:site_id/files/:file_id", deps.Artifacts.Download)
	}
	return router
}
