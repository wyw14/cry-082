package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	ruleapp "github.com/wyw14/cry-082/internal/application/rules"
	"github.com/wyw14/cry-082/internal/domain/rule"
	"github.com/wyw14/cry-082/internal/domain/telemetry"
	"github.com/wyw14/cry-082/internal/middleware"
)

type RuleHandler struct {
	service  *ruleapp.Service
	validate *validator.Validate
}

func NewRuleHandler(service *ruleapp.Service, validate *validator.Validate) *RuleHandler {
	return &RuleHandler{service: service, validate: validate}
}
func (h *RuleHandler) CreateVersion(c *gin.Context) {
	var request CreateRuleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "INVALID_JSON", "请求体不是有效 JSON", err)
		return
	}
	if err := h.validate.Struct(request); err != nil {
		fail(c, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "请求字段不符合要求", err)
		return
	}
	conditions := make([]rule.Condition, 0, len(request.Conditions))
	for _, item := range request.Conditions {
		conditions = append(conditions, rule.Condition{Metric: telemetry.Metric(item.Metric), Operator: rule.Operator(item.Operator), Value: item.Value})
	}
	created, err := h.service.CreateVersion(c.Request.Context(), ruleapp.CreateVersionInput{RuleID: request.RuleID, SiteID: c.Param("site_id"), Name: request.Name, Timezone: request.Timezone, ActorID: actorID(c), Reason: "create rule version", RequestID: middleware.GetRequestID(c), Conditions: conditions, RequireAll: request.RequireAll, Duration: time.Duration(request.DurationSeconds) * time.Second, MergeWindow: time.Duration(request.MergeWindowSeconds) * time.Second, LateGrace: time.Duration(request.LateGraceSeconds) * time.Second, EffectiveFrom: request.EffectiveFrom})
	if err != nil {
		domainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": created, "request_id": middleware.GetRequestID(c)})
}

func (h *RuleHandler) ActivateVersion(c *gin.Context) {
	version, err := strconv.ParseInt(c.Param("version"), 10, 64)
	if err != nil || version < 1 {
		fail(c, http.StatusUnprocessableEntity, "INVALID_RULE_VERSION", "规则版本必须是正整数", err)
		return
	}
	var request ActivateRuleRequest
	if !h.bind(c, &request) {
		return
	}
	activated, err := h.service.Activate(
		c.Request.Context(),
		c.Param("site_id"),
		c.Param("rule_id"),
		actorID(c),
		request.Reason,
		middleware.GetRequestID(c),
		version,
	)
	if err != nil {
		domainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": activated, "request_id": middleware.GetRequestID(c)})
}

func (h *RuleHandler) RequestRecalculation(c *gin.Context) {
	var request RecalculateRuleRequest
	if !h.bind(c, &request) {
		return
	}
	job, err := h.service.RequestRecalculation(
		c.Request.Context(),
		c.Param("site_id"),
		c.Param("rule_id"),
		actorID(c),
		request.Reason,
		request.FromVersion,
		request.ToVersion,
		request.WindowStart,
		request.WindowEnd,
	)
	if err != nil {
		domainError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"data": job, "request_id": middleware.GetRequestID(c)})
}

func (h *RuleHandler) bind(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		fail(c, http.StatusBadRequest, "INVALID_JSON", "请求体不是有效 JSON", err)
		return false
	}
	if err := h.validate.Struct(target); err != nil {
		fail(c, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "请求字段不符合要求", err)
		return false
	}
	return true
}
