package httpapi

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	deviceapp "github.com/wyw14/cry-082/internal/application/device"
	"github.com/wyw14/cry-082/internal/domain/device"
	"github.com/wyw14/cry-082/internal/middleware"
)

type DeviceHandler struct {
	service  *deviceapp.Service
	validate *validator.Validate
}

func NewDeviceHandler(service *deviceapp.Service, validate *validator.Validate) *DeviceHandler {
	return &DeviceHandler{service: service, validate: validate}
}

func (h *DeviceHandler) decodeTransition(c *gin.Context) (DeviceTransitionRequest, bool) {
	var request DeviceTransitionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "INVALID_JSON", "请求体不是有效 JSON", err)
		return DeviceTransitionRequest{}, false
	}
	if err := h.validate.Struct(request); err != nil {
		fail(c, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "请求字段不符合要求", err)
		return DeviceTransitionRequest{}, false
	}
	if request.Status == string(device.StatusReplaced) && request.ReplacementID == "" {
		fail(c, http.StatusUnprocessableEntity, "REPLACEMENT_REQUIRED", "更换设备时必须指定接替设备", nil)
		return DeviceTransitionRequest{}, false
	}
	if request.Status != string(device.StatusReplaced) && request.ReplacementID != "" {
		fail(c, http.StatusUnprocessableEntity, "REPLACEMENT_NOT_ALLOWED", "当前状态不能指定接替设备", nil)
		return DeviceTransitionRequest{}, false
	}
	return request, true
}

func transitionInput(c *gin.Context, request DeviceTransitionRequest) deviceapp.TransitionInput {
	return deviceapp.TransitionInput{
		DeviceID:        c.Param("id"),
		Next:            device.Status(request.Status),
		ReplacementID:   request.ReplacementID,
		ActorID:         actorID(c),
		Reason:          request.Reason,
		RequestID:       middleware.GetRequestID(c),
		ExpectedVersion: request.ExpectedVersion,
	}
}

type deviceLifecycleView struct {
	ID            string        `json:"id"`
	Code          string        `json:"code"`
	SiteID        string        `json:"site_id"`
	PointID       string        `json:"point_id"`
	Status        device.Status `json:"status"`
	ReplacementID string        `json:"replacement_id,omitempty"`
	Version       int64         `json:"version"`
}

func lifecycleView(value device.Device) deviceLifecycleView {
	return deviceLifecycleView{
		ID:            value.ID,
		Code:          value.Code,
		SiteID:        value.SiteID,
		PointID:       value.PointID,
		Status:        value.Status,
		ReplacementID: value.ReplacementID,
		Version:       value.Version,
	}
}

func (h *DeviceHandler) Transition(c *gin.Context) {
	request, ok := h.decodeTransition(c)
	if !ok {
		return
	}
	updated, err := h.service.Transition(c.Request.Context(), transitionInput(c, request))
	if err != nil {
		domainError(c, err)
		return
	}
	c.Header("ETag", fmt.Sprintf("\"device-%d\"", updated.Version))
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{
		"data":       lifecycleView(updated),
		"request_id": middleware.GetRequestID(c),
	})
}
