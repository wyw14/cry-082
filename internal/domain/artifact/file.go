package artifact

import (
	"errors"
	"strings"
	"time"
)

var ErrInvalidFile = errors.New("invalid stored file")

type Purpose string

const (
	PurposeMaintenanceCertificate Purpose = "maintenance-certificate"
	PurposeRegulatoryExport       Purpose = "regulatory-export"
)

type File struct {
	ID          string    `json:"id"`
	SiteID      string    `json:"site_id"`
	DisplayName string    `json:"display_name"`
	MediaType   string    `json:"media_type"`
	Purpose     Purpose   `json:"purpose"`
	Checksum    string    `json:"checksum"`
	Size        int64     `json:"size"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

func NewFile(id, siteID, displayName, mediaType string, purpose Purpose, checksum string, size int64, actor string, now time.Time) (File, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(siteID) == "" || strings.TrimSpace(displayName) == "" || strings.TrimSpace(mediaType) == "" || strings.TrimSpace(checksum) == "" || len(checksum) != 64 || size < 1 || strings.TrimSpace(actor) == "" {
		return File{}, ErrInvalidFile
	}
	if purpose != PurposeMaintenanceCertificate && purpose != PurposeRegulatoryExport {
		return File{}, ErrInvalidFile
	}
	return File{ID: id, SiteID: siteID, DisplayName: displayName, MediaType: mediaType, Purpose: purpose, Checksum: checksum, Size: size, CreatedBy: actor, CreatedAt: now.UTC()}, nil
}
