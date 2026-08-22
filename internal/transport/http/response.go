package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/wyw14/cry-082/internal/domain/site"
	"github.com/wyw14/cry-082/internal/middleware"
)

type FieldError struct {
	Field   string `json:"field"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}
type ErrorResponse struct {
	Code        string       `json:"code"`
	Message     string       `json:"message"`
	FieldErrors []FieldError `json:"field_errors"`
	RequestID   string       `json:"request_id"`
}

type errorEnvelope struct {
	status  int
	code    string
	message string
	fields  []FieldError
	meta    middleware.ResponseMetadata
}

func newErrorEnvelope(c *gin.Context, status int, code, message string, err error) errorEnvelope {
	return errorEnvelope{
		status:  status,
		code:    code,
		message: message,
		fields:  validationFields(err),
		meta:    middleware.ResponseMetadataFor(c),
	}
}

func validationFields(err error) []FieldError {
	fields := make([]FieldError, 0)
	var validation validator.ValidationErrors
	if !errors.As(err, &validation) {
		return fields
	}
	for _, item := range validation {
		fields = append(fields, FieldError{
			Field:   item.Field(),
			Rule:    item.Tag(),
			Message: "字段值不符合要求",
		})
	}
	return fields
}

func (e errorEnvelope) write(c *gin.Context) {
	c.AbortWithStatusJSON(e.status, ErrorResponse{
		Code:        e.code,
		Message:     e.message,
		FieldErrors: e.fields,
		RequestID:   e.meta.RequestID,
	})
}

func fail(c *gin.Context, status int, code, message string, err error) {
	newErrorEnvelope(c, status, code, message, err).write(c)
}
func domainError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, site.ErrAccessDenied):
		fail(c, http.StatusForbidden, "ACCESS_DENIED", "没有该工地资源的操作权限", err)
	default:
		fail(c, http.StatusUnprocessableEntity, "BUSINESS_RULE_VIOLATION", err.Error(), err)
	}
}
func pageBounds(page, pageSize int) (int, int, error) {
	if page < 1 || pageSize < 1 || pageSize > 200 {
		return 0, 0, errors.New("invalid pagination")
	}
	return (page - 1) * pageSize, pageSize, nil
}
