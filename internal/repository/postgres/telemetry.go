package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/wyw14/cry-082/internal/domain/telemetry"
)

func (s *Store) SaveSchema(ctx context.Context, value telemetry.Schema) error {
	_, err := s.db(ctx).Exec(ctx, `INSERT INTO measurement_schemas(id,metric,unit,sampling_period_ns,minimum,maximum,max_clock_skew_ns,version) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(id) DO UPDATE SET metric=EXCLUDED.metric,unit=EXCLUDED.unit,sampling_period_ns=EXCLUDED.sampling_period_ns,minimum=EXCLUDED.minimum,maximum=EXCLUDED.maximum,max_clock_skew_ns=EXCLUDED.max_clock_skew_ns,version=EXCLUDED.version`, value.ID, value.Metric, value.Unit, int64(value.SamplingPeriod), value.Minimum, value.Maximum, int64(value.MaxClockSkew), value.Version)
	return err
}
func (s *Store) FindSchema(ctx context.Context, id string) (telemetry.Schema, error) {
	var value telemetry.Schema
	var period, skew int64
	err := s.db(ctx).QueryRow(ctx, `SELECT id,metric,unit,sampling_period_ns,minimum,maximum,max_clock_skew_ns,version FROM measurement_schemas WHERE id=$1`, id).Scan(&value.ID, &value.Metric, &value.Unit, &period, &value.Minimum, &value.Maximum, &skew, &value.Version)
	if err != nil {
		return telemetry.Schema{}, err
	}
	value.SamplingPeriod = time.Duration(period)
	value.MaxClockSkew = time.Duration(skew)
	return value, nil
}
func (s *Store) ExistsIdentity(ctx context.Context, identity string) (bool, error) {
	var exists bool
	err := s.db(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM observations WHERE idempotency_key=$1)`, identity).Scan(&exists)
	return exists, err
}
func (s *Store) Append(ctx context.Context, value telemetry.Observation) error {
	reasons, err := json.Marshal(value.QualityReasons)
	if err != nil {
		return err
	}
	var correctedAt any
	if !value.CorrectedAt.IsZero() {
		correctedAt = value.CorrectedAt
	}
	var correctionOf any
	if value.CorrectionOf != "" {
		correctionOf = value.CorrectionOf
	}
	_, err = s.db(ctx).Exec(ctx, `INSERT INTO observations(id,device_id,site_id,point_id,schema_id,metric,value,unit,sampled_at,received_at,corrected_at,correction_of,quality,quality_reasons,idempotency_key,source_batch_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`, value.ID, value.DeviceID, value.SiteID, value.PointID, value.SchemaID, value.Metric, value.Value, value.Unit, value.SampledAt, value.ReceivedAt, correctedAt, correctionOf, value.Quality, reasons, value.IdempotencyKey, value.SourceBatchID)
	return err
}
func scanObservation(row pgx.Row) (telemetry.Observation, error) {
	var value telemetry.Observation
	var correctedAt *time.Time
	var correctionOf *string
	var reasons []byte
	err := row.Scan(&value.ID, &value.DeviceID, &value.SiteID, &value.PointID, &value.SchemaID, &value.Metric, &value.Value, &value.Unit, &value.SampledAt, &value.ReceivedAt, &correctedAt, &correctionOf, &value.Quality, &reasons, &value.IdempotencyKey, &value.SourceBatchID)
	if err != nil {
		return telemetry.Observation{}, err
	}
	if correctedAt != nil {
		value.CorrectedAt = *correctedAt
	}
	if correctionOf != nil {
		value.CorrectionOf = *correctionOf
	}
	if err := json.Unmarshal(reasons, &value.QualityReasons); err != nil {
		return telemetry.Observation{}, err
	}
	return value, nil
}

const observationColumns = `id,device_id,site_id,point_id,schema_id,metric,value,unit,sampled_at,received_at,corrected_at,correction_of,quality,quality_reasons,idempotency_key,source_batch_id`

func (s *Store) Latest(ctx context.Context, deviceID, schemaID string) (*telemetry.Observation, error) {
	value, err := scanObservation(s.db(ctx).QueryRow(ctx, `SELECT `+observationColumns+` FROM observations WHERE device_id=$1 AND schema_id=$2 ORDER BY sampled_at DESC LIMIT 1`, deviceID, schemaID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &value, nil
}
func (s *Store) FindObservation(ctx context.Context, id string) (telemetry.Observation, error) {
	return scanObservation(s.db(ctx).QueryRow(ctx, `SELECT `+observationColumns+` FROM observations WHERE id=$1`, id))
}
func (s *Store) Range(ctx context.Context, siteID string, start, end time.Time) ([]telemetry.Observation, error) {
	rows, err := s.db(ctx).Query(ctx, `SELECT `+observationColumns+` FROM observations WHERE site_id=$1 AND sampled_at >= $2 AND sampled_at < $3 ORDER BY sampled_at`, siteID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]telemetry.Observation, 0)
	for rows.Next() {
		value, err := scanObservation(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}
