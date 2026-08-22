package report

import (
	"errors"
	"strings"
	"time"
)

var ErrInvalidReport = errors.New("invalid report")

type DailyMetric struct {
	PointID            string
	Metric             string
	Maximum            float64
	Average            float64
	ExceedanceSeconds  int64
	AcceptedSamples    int
	SuspectSamples     int
	QuarantinedSamples int
}

type DailyReport struct {
	ID                  string
	SiteID              string
	LocalDate           string
	Timezone            string
	Metrics             []DailyMetric
	EnvironmentalAlerts int
	OfflineAlerts       int
	GeneratedAt         time.Time
	Revision            int64
}

func NewDaily(id, siteID, date, timezone string, metrics []DailyMetric, environmentalAlerts, offlineAlerts int, now time.Time) (DailyReport, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(siteID) == "" || len(date) != len("2006-01-02") || len(metrics) == 0 || environmentalAlerts < 0 || offlineAlerts < 0 {
		return DailyReport{}, ErrInvalidReport
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return DailyReport{}, ErrInvalidReport
	}
	return DailyReport{ID: id, SiteID: siteID, LocalDate: date, Timezone: timezone, Metrics: append([]DailyMetric(nil), metrics...), EnvironmentalAlerts: environmentalAlerts, OfflineAlerts: offlineAlerts, GeneratedAt: now.UTC(), Revision: 1}, nil
}

type Export struct {
	ID          string
	SiteID      string
	Format      string
	ReportIDs   []string
	RequestedBy string
	RequestedAt time.Time
	FileID      string
	Checksum    string
	Status      ExportStatus
	ReadyAt     *time.Time
	AnnouncedAt *time.Time
}

type ExportStatus string

const (
	ExportPreparing ExportStatus = "preparing"
	ExportReady     ExportStatus = "ready"
	ExportAnnounced ExportStatus = "announced"
)

func NewExport(id, siteID, format, actor string, reportIDs []string, now time.Time) (Export, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(siteID) == "" || strings.TrimSpace(actor) == "" || len(reportIDs) == 0 || (format != "csv" && format != "json") {
		return Export{}, ErrInvalidReport
	}
	return Export{ID: id, SiteID: siteID, Format: format, RequestedBy: actor, ReportIDs: append([]string(nil), reportIDs...), RequestedAt: now.UTC(), Status: ExportPreparing}, nil
}

func (e *Export) MarkReady(fileID, checksum string, at time.Time) error {
	if e.Status != ExportPreparing || strings.TrimSpace(fileID) == "" || strings.TrimSpace(checksum) == "" {
		return ErrInvalidReport
	}
	e.FileID = fileID
	e.Checksum = checksum
	readyAt := at.UTC()
	e.ReadyAt = &readyAt
	e.Status = ExportReady
	return nil
}

func (e *Export) MarkAnnounced(at time.Time) error {
	if e.Status != ExportReady {
		return ErrInvalidReport
	}
	announcedAt := at.UTC()
	e.AnnouncedAt = &announcedAt
	e.Status = ExportAnnounced
	return nil
}
