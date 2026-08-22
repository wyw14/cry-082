package postgres

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/wyw14/cry-082/internal/domain/artifact"
	"github.com/wyw14/cry-082/internal/domain/auth"
	"github.com/wyw14/cry-082/internal/domain/maintenance"
	"github.com/wyw14/cry-082/internal/domain/report"
)

func (s *Store) SaveRecord(ctx context.Context, value maintenance.Record) error {
	attachments, err := json.Marshal(value.AttachmentIDs)
	if err != nil {
		return err
	}
	_, err = s.db(ctx).Exec(ctx, `INSERT INTO maintenance_records(id,device_id,type,performed_by,started_at,completed_at,reason,result,replacement_id,attachment_ids) VALUES($1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,''),$10)`, value.ID, value.DeviceID, value.Type, value.PerformedBy, value.StartedAt, value.CompletedAt, value.Reason, value.Result, value.ReplacementID, attachments)
	return err
}
func (s *Store) SaveCalibration(ctx context.Context, value maintenance.Calibration) error {
	_, err := s.db(ctx).Exec(ctx, `INSERT INTO calibrations(id,device_id,schema_id,reference_value,observed_value,offset_value,performed_by,performed_at,expires_at,certificate_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10,''))`, value.ID, value.DeviceID, value.SchemaID, value.ReferenceValue, value.ObservedValue, value.Offset, value.PerformedBy, value.PerformedAt, value.ExpiresAt, value.CertificateID)
	return err
}

func (s *Store) SaveDaily(ctx context.Context, value report.DailyReport) error {
	metrics, err := json.Marshal(value.Metrics)
	if err != nil {
		return err
	}
	_, err = s.db(ctx).Exec(ctx, `INSERT INTO daily_reports(id,site_id,local_date,timezone,metrics,environmental_alerts,offline_alerts,generated_at,revision) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(id) DO UPDATE SET metrics=EXCLUDED.metrics,environmental_alerts=EXCLUDED.environmental_alerts,offline_alerts=EXCLUDED.offline_alerts,generated_at=EXCLUDED.generated_at,revision=EXCLUDED.revision WHERE daily_reports.revision < EXCLUDED.revision`, value.ID, value.SiteID, value.LocalDate, value.Timezone, metrics, value.EnvironmentalAlerts, value.OfflineAlerts, value.GeneratedAt, value.Revision)
	return err
}
func (s *Store) FindDaily(ctx context.Context, id string) (report.DailyReport, error) {
	var value report.DailyReport
	var metrics []byte
	err := s.db(ctx).QueryRow(ctx, `SELECT id,site_id,local_date,timezone,metrics,environmental_alerts,offline_alerts,generated_at,revision FROM daily_reports WHERE id=$1`, id).Scan(&value.ID, &value.SiteID, &value.LocalDate, &value.Timezone, &metrics, &value.EnvironmentalAlerts, &value.OfflineAlerts, &value.GeneratedAt, &value.Revision)
	if err != nil {
		return report.DailyReport{}, err
	}
	if err := json.Unmarshal(metrics, &value.Metrics); err != nil {
		return report.DailyReport{}, err
	}
	return value, nil
}
func (s *Store) SaveExport(ctx context.Context, value report.Export) error {
	reportIDs, err := json.Marshal(value.ReportIDs)
	if err != nil {
		return err
	}
	_, err = s.db(ctx).Exec(ctx, `INSERT INTO regulatory_exports(id,site_id,format,report_ids,requested_by,requested_at,file_id,checksum) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, value.ID, value.SiteID, value.Format, reportIDs, value.RequestedBy, value.RequestedAt, value.FileID, value.Checksum)
	return err
}

func (s *Store) SaveUser(ctx context.Context, value auth.User) error {
	_, err := s.db(ctx).Exec(ctx, `INSERT INTO users(id,username,password_hash,display_name,masked_phone,active,failed_attempts,locked_until,version) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(id) DO UPDATE SET password_hash=EXCLUDED.password_hash,display_name=EXCLUDED.display_name,masked_phone=EXCLUDED.masked_phone,active=EXCLUDED.active,failed_attempts=EXCLUDED.failed_attempts,locked_until=EXCLUDED.locked_until,version=EXCLUDED.version WHERE users.version < EXCLUDED.version`, value.ID, value.Username, value.PasswordHash, value.DisplayName, value.MaskedPhone, value.Active, value.FailedAttempts, nullableTime(value.LockedUntil), value.Version)
	return err
}
func scanUser(row interface{ Scan(...any) error }) (auth.User, error) {
	var value auth.User
	var lockedUntil *time.Time
	err := row.Scan(&value.ID, &value.Username, &value.PasswordHash, &value.DisplayName, &value.MaskedPhone, &value.Active, &value.FailedAttempts, &lockedUntil, &value.Version)
	if lockedUntil != nil {
		value.LockedUntil = *lockedUntil
	}
	return value, err
}
func (s *Store) FindUserByUsername(ctx context.Context, username string) (auth.User, error) {
	return scanUser(s.db(ctx).QueryRow(ctx, `SELECT id,username,password_hash,display_name,masked_phone,active,failed_attempts,locked_until,version FROM users WHERE username=$1`, strings.ToLower(username)))
}
func (s *Store) FindUserByID(ctx context.Context, id string) (auth.User, error) {
	return scanUser(s.db(ctx).QueryRow(ctx, `SELECT id,username,password_hash,display_name,masked_phone,active,failed_attempts,locked_until,version FROM users WHERE id=$1`, id))
}
func (s *Store) SaveRefreshToken(ctx context.Context, value auth.RefreshToken) error {
	_, err := s.db(ctx).Exec(ctx, `INSERT INTO refresh_tokens(id,user_id,digest,issued_at,expires_at,revoked_at,replaced_by) VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,'')) ON CONFLICT(id) DO UPDATE SET revoked_at=EXCLUDED.revoked_at,replaced_by=EXCLUDED.replaced_by`, value.ID, value.UserID, value.Digest, value.IssuedAt, value.ExpiresAt, value.RevokedAt, value.ReplacedBy)
	return err
}
func (s *Store) FindRefreshToken(ctx context.Context, id string) (auth.RefreshToken, error) {
	var value auth.RefreshToken
	var replacement *string
	err := s.db(ctx).QueryRow(ctx, `SELECT id,user_id,digest,issued_at,expires_at,revoked_at,replaced_by FROM refresh_tokens WHERE id=$1`, id).Scan(&value.ID, &value.UserID, &value.Digest, &value.IssuedAt, &value.ExpiresAt, &value.RevokedAt, &replacement)
	if replacement != nil {
		value.ReplacedBy = *replacement
	}
	return value, err
}

func (s *Store) RotateRefreshToken(ctx context.Context, current, replacement auth.RefreshToken) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	command, err := tx.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=$2,replaced_by=$3 WHERE id=$1 AND digest=$4 AND revoked_at IS NULL`, current.ID, current.RevokedAt, current.ReplacedBy, current.Digest)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return auth.ErrInvalidToken
	}
	if _, err := tx.Exec(ctx, `INSERT INTO refresh_tokens(id,user_id,digest,issued_at,expires_at,revoked_at,replaced_by) VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,''))`, replacement.ID, replacement.UserID, replacement.Digest, replacement.IssuedAt, replacement.ExpiresAt, replacement.RevokedAt, replacement.ReplacedBy); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) SaveFile(ctx context.Context, value artifact.File) error {
	_, err := s.db(ctx).Exec(ctx, `INSERT INTO stored_files(id,site_id,display_name,media_type,purpose,checksum,size_bytes,created_by,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, value.ID, value.SiteID, value.DisplayName, value.MediaType, value.Purpose, value.Checksum, value.Size, value.CreatedBy, value.CreatedAt)
	return err
}

func (s *Store) FindFile(ctx context.Context, id string) (artifact.File, error) {
	var value artifact.File
	err := s.db(ctx).QueryRow(ctx, `SELECT id,site_id,display_name,media_type,purpose,checksum,size_bytes,created_by,created_at FROM stored_files WHERE id=$1`, id).Scan(&value.ID, &value.SiteID, &value.DisplayName, &value.MediaType, &value.Purpose, &value.Checksum, &value.Size, &value.CreatedBy, &value.CreatedAt)
	return value, err
}
