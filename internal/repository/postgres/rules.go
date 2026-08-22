package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/wyw14/cry-082/internal/domain/rule"
)

func (s *Store) SaveRule(ctx context.Context, value rule.Version) error {
	conditions, err := json.Marshal(value.Conditions)
	if err != nil {
		return err
	}
	_, err = s.db(ctx).Exec(ctx, `INSERT INTO rule_versions(rule_id,version,site_id,name,timezone,conditions,require_all,duration_ns,merge_window_ns,late_grace_ns,effective_from,status,created_by,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) ON CONFLICT(rule_id,version) DO UPDATE SET name=EXCLUDED.name,timezone=EXCLUDED.timezone,conditions=EXCLUDED.conditions,require_all=EXCLUDED.require_all,duration_ns=EXCLUDED.duration_ns,merge_window_ns=EXCLUDED.merge_window_ns,late_grace_ns=EXCLUDED.late_grace_ns,effective_from=EXCLUDED.effective_from,status=EXCLUDED.status`, value.RuleID, value.Version, value.SiteID, value.Name, value.Timezone, conditions, value.RequireAll, int64(value.Duration), int64(value.MergeWindow), int64(value.LateGrace), value.EffectiveFrom, value.Status, value.CreatedBy, value.CreatedAt)
	return err
}
func scanRule(row pgx.Row) (rule.Version, error) {
	var value rule.Version
	var conditions []byte
	var duration, mergeWindow, lateGrace int64
	err := row.Scan(&value.RuleID, &value.Version, &value.SiteID, &value.Name, &value.Timezone, &conditions, &value.RequireAll, &duration, &mergeWindow, &lateGrace, &value.EffectiveFrom, &value.Status, &value.CreatedBy, &value.CreatedAt)
	if err != nil {
		return rule.Version{}, err
	}
	if err := json.Unmarshal(conditions, &value.Conditions); err != nil {
		return rule.Version{}, err
	}
	value.Duration = time.Duration(duration)
	value.MergeWindow = time.Duration(mergeWindow)
	value.LateGrace = time.Duration(lateGrace)
	return value, nil
}

const ruleColumns = `rule_id,version,site_id,name,timezone,conditions,require_all,duration_ns,merge_window_ns,late_grace_ns,effective_from,status,created_by,created_at`

func (s *Store) LatestRule(ctx context.Context, ruleID string) (*rule.Version, error) {
	value, err := scanRule(s.db(ctx).QueryRow(ctx, `SELECT `+ruleColumns+` FROM rule_versions WHERE rule_id=$1 ORDER BY version DESC LIMIT 1`, ruleID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &value, nil
}
func (s *Store) ActiveForSite(ctx context.Context, siteID string, at time.Time) ([]rule.Version, error) {
	rows, err := s.db(ctx).Query(ctx, `SELECT `+ruleColumns+` FROM rule_versions WHERE site_id=$1 AND status='active' AND effective_from <= $2 ORDER BY rule_id,version`, siteID, at)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]rule.Version, 0)
	for rows.Next() {
		value, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}
func (s *Store) AppendEvaluation(ctx context.Context, value rule.Evaluation) error {
	observations, err := json.Marshal(value.ObservationIDs)
	if err != nil {
		return err
	}
	_, err = s.db(ctx).Exec(ctx, `INSERT INTO evaluations(rule_id,rule_version,timezone,site_id,point_id,window_start,window_end,matched,observation_ids,conclusion,recalculation_id,evaluated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, value.RuleID, value.RuleVersion, value.Timezone, value.SiteID, value.PointID, nullableTime(value.WindowStart), nullableTime(value.WindowEnd), value.Matched, observations, value.Conclusion, value.RecalculationID, value.EvaluatedAt)
	return err
}
func (s *Store) SaveRecalculation(ctx context.Context, value rule.Recalculation) error {
	_, err := s.db(ctx).Exec(ctx, `INSERT INTO recalculations(id,site_id,rule_id,from_version,to_version,window_start,window_end,reason,requested_by,requested_at,status,processed_points,failure) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) ON CONFLICT(id) DO UPDATE SET status=EXCLUDED.status,processed_points=EXCLUDED.processed_points,failure=EXCLUDED.failure`, value.ID, value.SiteID, value.RuleID, value.FromVersion, value.ToVersion, value.WindowStart, value.WindowEnd, value.Reason, value.RequestedBy, value.RequestedAt, value.Status, value.ProcessedPoints, value.Failure)
	return err
}
func nullableTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}
