package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/wyw14/cry-082/internal/application/monitor"
	"github.com/wyw14/cry-082/internal/domain/telemetry"
	"github.com/wyw14/cry-082/internal/middleware"
)

type MonitorHandler struct {
	service *monitor.Service
	now     func() time.Time
}

func NewMonitorHandler(service *monitor.Service) *MonitorHandler {
	return &MonitorHandler{service: service, now: time.Now}
}
func (h *MonitorHandler) Dashboard(c *gin.Context) {
	dashboard, err := h.service.Dashboard(c.Request.Context(), c.Param("site_id"), h.now())
	if err != nil {
		domainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": dashboard, "request_id": middleware.GetRequestID(c)})
}
func (h *MonitorHandler) Trend(c *gin.Context) {
	start, err := time.Parse(time.RFC3339, c.Query("start"))
	if err != nil {
		fail(c, http.StatusBadRequest, "INVALID_TIME", "start 必须是 RFC3339 时间", err)
		return
	}
	end, err := time.Parse(time.RFC3339, c.Query("end"))
	if err != nil {
		fail(c, http.StatusBadRequest, "INVALID_TIME", "end 必须是 RFC3339 时间", err)
		return
	}
	bucketSeconds, err := strconv.Atoi(c.DefaultQuery("bucket_seconds", "300"))
	if err != nil || bucketSeconds < 1 {
		fail(c, http.StatusBadRequest, "INVALID_BUCKET", "bucket_seconds 必须是正整数", err)
		return
	}
	trend, err := h.service.Trend(c.Request.Context(), c.Param("site_id"), c.Query("point_id"), telemetry.Metric(c.Query("metric")), start, end, time.Duration(bucketSeconds)*time.Second)
	if err != nil {
		domainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": trend, "request_id": middleware.GetRequestID(c)})
}
