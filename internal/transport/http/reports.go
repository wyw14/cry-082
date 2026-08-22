package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/wyw14/cry-082/internal/application/reporting"
	"github.com/wyw14/cry-082/internal/domain/report"
	"github.com/wyw14/cry-082/internal/middleware"
)

type exportRequest struct {
	Format    string   `json:"format" validate:"required,oneof=csv json"`
	ReportIDs []string `json:"report_ids" validate:"required,min=1,max=31,dive,required"`
}

func (r exportRequest) normalized() (exportRequest, bool) {
	result := exportRequest{Format: strings.ToLower(strings.TrimSpace(r.Format))}
	seen := make(map[string]struct{}, len(r.ReportIDs))
	for _, rawID := range r.ReportIDs {
		id := strings.TrimSpace(rawID)
		if id == "" {
			return exportRequest{}, false
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result.ReportIDs = append(result.ReportIDs, id)
	}
	return result, len(result.ReportIDs) > 0
}

type regulatoryExportView struct {
	ExportID     string    `json:"export_id"`
	SiteID       string    `json:"site_id"`
	Format       string    `json:"format"`
	ReportCount  int       `json:"report_count"`
	RequestedAt  time.Time `json:"requested_at"`
	DownloadFile string    `json:"download_file_id,omitempty"`
	Checksum     string    `json:"checksum,omitempty"`
}

func newRegulatoryExportView(created report.Export) regulatoryExportView {
	return regulatoryExportView{
		ExportID:     created.ID,
		SiteID:       created.SiteID,
		Format:       created.Format,
		ReportCount:  len(created.ReportIDs),
		RequestedAt:  created.RequestedAt,
		DownloadFile: created.FileID,
		Checksum:     created.Checksum,
	}
}

type ReportHandler struct {
	service  *reporting.Service
	validate *validator.Validate
}

func NewReportHandler(service *reporting.Service, validate *validator.Validate) *ReportHandler {
	return &ReportHandler{service: service, validate: validate}
}

func (h *ReportHandler) Export(c *gin.Context) {
	var wire exportRequest
	if err := c.ShouldBindJSON(&wire); err != nil {
		fail(c, http.StatusBadRequest, "INVALID_JSON", "请求体不是有效 JSON", err)
		return
	}
	request, ok := wire.normalized()
	if !ok {
		fail(c, http.StatusUnprocessableEntity, "EMPTY_REPORT_SET", "监管导出至少需要一份有效日报", nil)
		return
	}
	if err := h.validate.Struct(request); err != nil {
		fail(c, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "导出参数不符合要求", err)
		return
	}
	created, err := h.service.Export(
		c.Request.Context(),
		c.Param("site_id"),
		actorID(c),
		request.Format,
		request.ReportIDs,
	)
	if err != nil {
		domainError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Location", "/api/v1/sites/"+created.SiteID+"/files/"+created.FileID)
	c.JSON(http.StatusAccepted, gin.H{
		"data":       newRegulatoryExportView(created),
		"request_id": middleware.GetRequestID(c),
	})
}
