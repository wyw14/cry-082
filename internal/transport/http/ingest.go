package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/wyw14/cry-082/internal/application/ingest"
	"github.com/wyw14/cry-082/internal/middleware"
)

type IngestHandler struct {
	service  *ingest.Service
	validate *validator.Validate
	now      func() time.Time
}

func NewIngestHandler(service *ingest.Service, validate *validator.Validate) *IngestHandler {
	return &IngestHandler{service: service, validate: validate, now: time.Now}
}
func (h *IngestHandler) Batch(c *gin.Context) {
	var request TelemetryBatchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "INVALID_JSON", "请求体不是有效 JSON", err)
		return
	}
	if err := h.validate.Struct(request); err != nil {
		fail(c, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "请求字段不符合要求", err)
		return
	}
	samples := make([]ingest.Sample, 0, len(request.Samples))
	for _, item := range request.Samples {
		samples = append(samples, ingest.Sample{DeviceCode: item.DeviceCode, SchemaID: item.SchemaID, Value: item.Value, SampledAt: item.SampledAt})
	}
	result, err := h.service.Ingest(c.Request.Context(), ingest.Batch{BatchID: request.BatchID, ActorID: actorID(c), ReceivedAt: h.now().UTC(), Samples: samples})
	if err != nil {
		domainError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"data": result, "request_id": middleware.GetRequestID(c)})
}
func actorID(c *gin.Context) string {
	if actor, exists := c.Get(middleware.ActorContextKey); exists {
		if value, ok := actor.(string); ok && value != "" {
			return value
		}
	}
	return ""
}
