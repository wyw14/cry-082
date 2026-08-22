package httpapi

import (
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/wyw14/cry-082/internal/application/artifacts"
	"github.com/wyw14/cry-082/internal/domain/artifact"
	"github.com/wyw14/cry-082/internal/middleware"
	"github.com/wyw14/cry-082/internal/platform/files"
)

type ArtifactHandler struct {
	service *artifacts.Service
	maximum int64
}

func NewArtifactHandler(service *artifacts.Service, maximum int64) *ArtifactHandler {
	return &ArtifactHandler{service: service, maximum: maximum}
}

func (h *ArtifactHandler) Upload(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maximum+(1<<20))
	part, err := c.FormFile("file")
	if err != nil {
		fail(c, http.StatusBadRequest, "FILE_REQUIRED", "必须提交一个文件", err)
		return
	}
	if part.Size < 1 || part.Size > h.maximum {
		fail(c, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "文件大小超出限制", files.ErrFileTooLarge)
		return
	}
	reader, err := part.Open()
	if err != nil {
		fail(c, http.StatusBadRequest, "FILE_UNREADABLE", "无法读取上传文件", err)
		return
	}
	defer reader.Close()
	payload, err := io.ReadAll(io.LimitReader(reader, h.maximum+1))
	if err != nil || int64(len(payload)) > h.maximum {
		fail(c, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "文件大小超出限制", errors.Join(err, files.ErrFileTooLarge))
		return
	}
	purpose := artifact.Purpose(c.PostForm("purpose"))
	created, err := h.service.Upload(c.Request.Context(), artifacts.UploadInput{SiteID: c.Param("site_id"), ActorID: actorID(c), RequestID: middleware.GetRequestID(c), Name: part.Filename, MediaType: part.Header.Get("Content-Type"), Purpose: purpose, Payload: payload})
	if err != nil {
		domainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": created, "request_id": middleware.GetRequestID(c)})
}

func (h *ArtifactHandler) Download(c *gin.Context) {
	metadata, reader, err := h.service.Download(c.Request.Context(), c.Param("site_id"), c.Param("file_id"), actorID(c))
	if err != nil {
		domainError(c, err)
		return
	}
	defer reader.Close()
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": metadata.DisplayName})
	if disposition == "" {
		disposition = "attachment"
	}
	c.Header("Content-Disposition", disposition)
	c.Header("X-Content-SHA256", metadata.Checksum)
	c.DataFromReader(http.StatusOK, metadata.Size, metadata.MediaType, reader, nil)
}
