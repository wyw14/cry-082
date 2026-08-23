package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	alertapp "github.com/wyw14/cry-082/internal/application/alerts"
	"github.com/wyw14/cry-082/internal/domain/alert"
)

const alertColumns = `id,site_id,point_id,device_id,kind,rule_id,rule_version,status,started_at,last_signal_at,recovered_at,closed_at,assignee_id,merge_key,occurrence_count,version`

func scanAlert(row pgx.Row) (alert.Alert, error) {
	var value alert.Alert
	var pointID, ruleID, assignee *string
	err := row.Scan(&value.ID, &value.SiteID, &pointID, &value.DeviceID, &value.Kind, &ruleID, &value.RuleVersion, &value.Status, &value.StartedAt, &value.LastSignalAt, &value.RecoveredAt, &value.ClosedAt, &assignee, &value.MergeKey, &value.OccurrenceCount, &value.Version)
	if err != nil {
		return alert.Alert{}, err
	}
	if pointID != nil {
		value.PointID = *pointID
	}
	if ruleID != nil {
		value.RuleID = *ruleID
	}
	if assignee != nil {
		value.AssigneeID = *assignee
	}
	return value, nil
}
func (s *Store) SaveAlert(ctx context.Context, value alert.Alert) error {
	_, err := s.db(ctx).Exec(ctx, `INSERT INTO alerts(id,site_id,point_id,device_id,kind,rule_id,rule_version,status,started_at,last_signal_at,recovered_at,closed_at,assignee_id,merge_key,occurrence_count,version) VALUES($1,$2,NULLIF($3,''),$4,$5,NULLIF($6,''),$7,$8,$9,$10,$11,$12,NULLIF($13,''),$14,$15,$16) ON CONFLICT(id) DO UPDATE SET status=EXCLUDED.status,last_signal_at=EXCLUDED.last_signal_at,recovered_at=EXCLUDED.recovered_at,closed_at=EXCLUDED.closed_at,assignee_id=EXCLUDED.assignee_id,occurrence_count=EXCLUDED.occurrence_count,version=EXCLUDED.version WHERE alerts.version < EXCLUDED.version`, value.ID, value.SiteID, value.PointID, value.DeviceID, value.Kind, value.RuleID, value.RuleVersion, value.Status, value.StartedAt, value.LastSignalAt, value.RecoveredAt, value.ClosedAt, value.AssigneeID, value.MergeKey, value.OccurrenceCount, value.Version)
	return err
}
func (s *Store) FindAlert(ctx context.Context, id string) (alert.Alert, error) {
	return scanAlert(s.db(ctx).QueryRow(ctx, `SELECT `+alertColumns+` FROM alerts WHERE id=$1`, id))
}
func (s *Store) FindMergeable(ctx context.Context, key string, kind alert.Kind, since time.Time) (*alert.Alert, error) {
	value, err := scanAlert(s.db(ctx).QueryRow(ctx, `SELECT `+alertColumns+` FROM alerts WHERE merge_key=$1 AND kind=$2 AND last_signal_at >= $3 AND status <> 'closed' ORDER BY last_signal_at DESC LIMIT 1`, key, kind, since))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &value, nil
}
func (s *Store) RangeAlerts(ctx context.Context, siteID string, start, end time.Time) ([]alert.Alert, error) {
	rows, err := s.db(ctx).Query(ctx, `SELECT `+alertColumns+` FROM alerts WHERE site_id=$1 AND started_at >= $2 AND started_at < $3 ORDER BY started_at`, siteID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]alert.Alert, 0)
	for rows.Next() {
		value, err := scanAlert(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}
func (s *Store) SaveWorkOrder(ctx context.Context, value alert.WorkOrder) error {
	_, err := s.db(ctx).Exec(ctx, `INSERT INTO work_orders(id,alert_id,assignee_id,status,description,created_at,due_at,resolved_at,version) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(id) DO UPDATE SET status=EXCLUDED.status,resolved_at=EXCLUDED.resolved_at,version=EXCLUDED.version WHERE work_orders.version < EXCLUDED.version`, value.ID, value.AlertID, value.AssigneeID, value.Status, value.Description, value.CreatedAt, value.DueAt, value.ResolvedAt, value.Version)
	return err
}
func (s *Store) ListAlerts(ctx context.Context, filter alertapp.AlertFilter) ([]alert.Alert, int, error) {
	sortColumn := map[string]string{"started_at": "started_at", "last_signal_at": "last_signal_at", "status": "status"}[filter.Sort]
	if sortColumn == "" {
		sortColumn = "started_at"
	}
	direction := "ASC"
	if filter.Descending {
		direction = "DESC"
	}
	where := `site_id=$1 AND ($2='' OR kind=$2) AND ($3='' OR status=$3)`
	var total int
	if err := s.db(ctx).QueryRow(ctx, `SELECT count(*) FROM alerts WHERE `+where, filter.SiteID, filter.Kind, filter.Status).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db(ctx).Query(ctx, `SELECT `+alertColumns+` FROM alerts WHERE `+where+` ORDER BY `+sortColumn+` `+direction+`,id LIMIT $4 OFFSET $5`, filter.SiteID, filter.Kind, filter.Status, filter.Limit, filter.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	result := make([]alert.Alert, 0)
	for rows.Next() {
		value, err := scanAlert(rows)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, value)
	}
	return result, total, rows.Err()
}
