package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/wyw14/cry-082/internal/application/authn"
	"github.com/wyw14/cry-082/internal/middleware"
)

type loginRequest struct {
	Username string `json:"username" validate:"required,min=3,max=120"`
	Password string `json:"password" validate:"required,min=8,max=200"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required,min=40,max=256"`
}

type AuthHandler struct {
	service  *authn.Service
	validate *validator.Validate
}

func NewAuthHandler(service *authn.Service, validate *validator.Validate) *AuthHandler {
	return &AuthHandler{service: service, validate: validate}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var request loginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "INVALID_JSON", "请求体不是有效 JSON", err)
		return
	}
	if err := h.validate.Struct(request); err != nil {
		fail(c, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "登录字段不符合要求", err)
		return
	}
	pair, err := h.service.Login(c.Request.Context(), request.Username, request.Password)
	if err != nil {
		if errors.Is(err, authn.ErrInvalidCredentials) {
			fail(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "用户名或密码错误", err)
			return
		}
		domainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": pair, "request_id": middleware.GetRequestID(c)})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var request refreshRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "INVALID_JSON", "请求体不是有效 JSON", err)
		return
	}
	if err := h.validate.Struct(request); err != nil {
		fail(c, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "刷新令牌不符合要求", err)
		return
	}
	id, raw, ok := strings.Cut(request.RefreshToken, ".")
	if !ok || id == "" || raw == "" {
		fail(c, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "刷新令牌无效", nil)
		return
	}
	pair, err := h.service.Refresh(c.Request.Context(), id, raw)
	if err != nil {
		fail(c, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "刷新令牌无效", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": pair, "request_id": middleware.GetRequestID(c)})
}
