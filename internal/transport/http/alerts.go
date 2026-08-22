package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	alertapp "github.com/wyw14/cry-082/internal/application/alerts"
	"github.com/wyw14/cry-082/internal/domain/alert"
	"github.com/wyw14/cry-082/internal/middleware"
)

type AlertHandler struct {
	service  *alertapp.Service
	validate *validator.Validate
}

func NewAlertHandler(service *alertapp.Service, validate *validator.Validate) *AlertHandler {
	return &AlertHandler{service: service, validate: validate}
}
func (h *AlertHandler) Transition(c *gin.Context) {
	var request AlertTransitionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "INVALID_JSON", "请求体不是有效 JSON", err)
		return
	}
	if err := h.validate.Struct(request); err != nil {
		fail(c, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "请求字段不符合要求", err)
		return
	}
	updated, err := h.service.Transition(c.Request.Context(), alertapp.TransitionInput{AlertID: c.Param("id"), Next: alert.Status(request.Status), ActorID: actorID(c), Reason: request.Reason, AssigneeID: request.AssigneeID, RequestID: middleware.GetRequestID(c), ExpectedVersion: request.ExpectedVersion})
	if err != nil {
		domainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": updated, "request_id": middleware.GetRequestID(c)})
}

func (h *AlertHandler) List(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		fail(c, http.StatusBadRequest, "INVALID_PAGE", "page 必须是正整数", err)
		return
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if err != nil {
		fail(c, http.StatusBadRequest, "INVALID_PAGE_SIZE", "page_size 必须是正整数", err)
		return
	}
	offset, limit, err := pageBounds(page, pageSize)
	if err != nil {
		fail(c, http.StatusBadRequest, "INVALID_PAGINATION", "分页参数超出允许范围", err)
		return
	}
	sortValue := c.DefaultQuery("sort", "-started_at")
	descending := strings.HasPrefix(sortValue, "-")
	sortField := strings.TrimPrefix(sortValue, "-")
	rows, total, err := h.service.List(c.Request.Context(), alertapp.AlertFilter{SiteID: c.Param("site_id"), Kind: alert.Kind(c.Query("kind")), Status: alert.Status(c.Query("status")), Sort: sortField, Descending: descending, Offset: offset, Limit: limit}, actorID(c))
	if err != nil {
		domainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows, "pagination": gin.H{"page": page, "page_size": pageSize, "total": total}, "request_id": middleware.GetRequestID(c)})
}
