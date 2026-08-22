package reporting

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wyw14/cry-082/internal/domain/artifact"
	"github.com/wyw14/cry-082/internal/domain/audit"
	"github.com/wyw14/cry-082/internal/domain/report"
	"github.com/wyw14/cry-082/internal/domain/site"
	"github.com/wyw14/cry-082/internal/platform/transaction"
)

type Clock interface{ Now() time.Time }
type IDGenerator interface{ NewID() string }
type Repository interface {
	FindDaily(context.Context, string) (report.DailyReport, error)
	SaveExport(context.Context, report.Export) error
	SaveFile(context.Context, artifact.File) error
}
type FileStore interface {
	Put(context.Context, string, string, []byte) (string, string, error)
}
type AccessRepository interface {
	Membership(context.Context, string, string) (site.Membership, error)
}
type Notification interface {
	Notify(context.Context, string, string, map[string]string) error
}
type AuditRepository interface {
	AppendAudit(context.Context, audit.Entry) error
}
type Outbox interface {
	Enqueue(context.Context, string, string, any) error
}

type Service struct {
	reports       Repository
	files         FileStore
	access        AccessRepository
	notifications Notification
	audits        AuditRepository
	outbox        Outbox
	tx            transaction.Manager
	clock         Clock
	ids           IDGenerator
}

func New(reports Repository, files FileStore, access AccessRepository, notifications Notification, audits AuditRepository, outbox Outbox, tx transaction.Manager, clock Clock, ids IDGenerator) *Service {
	return &Service{reports: reports, files: files, access: access, notifications: notifications, audits: audits, outbox: outbox, tx: tx, clock: clock, ids: ids}
}

func (s *Service) Export(ctx context.Context, siteID, actor, format string, reportIDs []string) (created report.Export, err error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	membership, err := s.access.Membership(ctx, actor, siteID)
	if err != nil {
		return report.Export{}, err
	}
	if err := site.Require(membership, siteID, site.PermissionReportExport); err != nil {
		return report.Export{}, err
	}
	requested, err := report.NewExport(s.ids.NewID(), siteID, format, actor, reportIDs, s.clock.Now())
	if err != nil {
		return report.Export{}, err
	}
	rows := make([]report.DailyReport, 0, len(reportIDs))
	for _, reportID := range reportIDs {
		daily, err := s.reports.FindDaily(ctx, reportID)
		if err != nil {
			return report.Export{}, err
		}
		if daily.SiteID != siteID {
			return report.Export{}, site.ErrAccessDenied
		}
		rows = append(rows, daily)
	}
	payload, mime, err := encode(rows, format)
	if err != nil {
		return report.Export{}, err
	}
	fileID, checksum, err := s.files.Put(ctx, fmt.Sprintf("regulatory-%s.%s", requested.ID, format), mime, payload)
	if err != nil {
		return report.Export{}, err
	}
	requested.FileID, requested.Checksum = fileID, checksum
	storedFile, err := artifact.NewFile(fileID, siteID, "regulatory-"+requested.ID+"."+format, mime, artifact.PurposeRegulatoryExport, checksum, int64(len(payload)), actor, s.clock.Now())
	if err != nil {
		return report.Export{}, err
	}
	err = transaction.Execute(ctx, s.tx, func(txctx context.Context) error {
		if err := s.reports.SaveFile(txctx, storedFile); err != nil {
			return err
		}
		if err := s.reports.SaveExport(txctx, requested); err != nil {
			return err
		}
		entry, err := audit.New(s.ids.NewID(), siteID, actor, "api", "regulatory-export.created", "regulatory-export", requested.ID, "generate regulatory export", "", nil, map[string]string{"format": format, "checksum": checksum}, s.clock.Now())
		if err != nil {
			return err
		}
		if err := s.audits.AppendAudit(txctx, entry); err != nil {
			return err
		}
		return s.outbox.Enqueue(txctx, "regulatory-export.ready", requested.ID, map[string]any{"export_id": requested.ID, "user_id": actor, "checksum": checksum})
	})
	if err != nil {
		return report.Export{}, err
	}
	if err := s.notifications.Notify(ctx, actor, "监管导出已生成", map[string]string{"export_id": requested.ID, "checksum": checksum}); err != nil {
		return report.Export{}, err
	}
	return requested, nil
}

func encode(reports []report.DailyReport, format string) ([]byte, string, error) {
	if format == "json" {
		payload, err := json.MarshalIndent(reports, "", "  ")
		return payload, "application/json", err
	}
	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)
	_ = writer.Write([]string{"site_id", "local_date", "timezone", "point_id", "metric", "maximum", "average", "exceedance_seconds", "accepted_samples", "suspect_samples", "quarantined_samples"})
	for _, daily := range reports {
		for _, metric := range daily.Metrics {
			_ = writer.Write([]string{daily.SiteID, daily.LocalDate, daily.Timezone, metric.PointID, metric.Metric, fmt.Sprintf("%.3f", metric.Maximum), fmt.Sprintf("%.3f", metric.Average), fmt.Sprintf("%d", metric.ExceedanceSeconds), fmt.Sprintf("%d", metric.AcceptedSamples), fmt.Sprintf("%d", metric.SuspectSamples), fmt.Sprintf("%d", metric.QuarantinedSamples)})
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", err
	}
	payload := buffer.Bytes()
	sum := sha256.Sum256(payload)
	_ = hex.EncodeToString(sum[:])
	return payload, "text/csv", nil
}
