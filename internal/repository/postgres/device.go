package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/wyw14/cry-082/internal/domain/device"
)

func (s *Store) Save(ctx context.Context, value device.Device) error {
	network, err := json.Marshal(value.Network)
	if err != nil {
		return err
	}
	command, err := s.db(ctx).Exec(ctx, `INSERT INTO devices(id,code,model,site_id,point_id,install_location,network,status,last_seen_at,replacement_id,version) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10,''),$11) ON CONFLICT(id) DO UPDATE SET code=EXCLUDED.code,model=EXCLUDED.model,point_id=EXCLUDED.point_id,install_location=EXCLUDED.install_location,network=EXCLUDED.network,status=EXCLUDED.status,last_seen_at=EXCLUDED.last_seen_at,replacement_id=EXCLUDED.replacement_id,version=EXCLUDED.version WHERE devices.version < EXCLUDED.version`, value.ID, value.Code, value.Model, value.SiteID, value.PointID, value.InstallLocation, network, value.Status, value.LastSeenAt, value.ReplacementID, value.Version)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return errors.New("device version conflict")
	}
	return nil
}

const deviceColumns = `id,code,model,site_id,point_id,install_location,network,status,last_seen_at,replacement_id,version`

func scanDevice(row pgx.Row) (device.Device, error) {
	var value device.Device
	var network []byte
	var replacement *string
	err := row.Scan(&value.ID, &value.Code, &value.Model, &value.SiteID, &value.PointID, &value.InstallLocation, &network, &value.Status, &value.LastSeenAt, &replacement, &value.Version)
	if err != nil {
		return device.Device{}, err
	}
	if replacement != nil {
		value.ReplacementID = *replacement
	}
	if err := json.Unmarshal(network, &value.Network); err != nil {
		return device.Device{}, err
	}
	return value, nil
}
func (s *Store) Find(ctx context.Context, id string) (device.Device, error) {
	return scanDevice(s.db(ctx).QueryRow(ctx, `SELECT `+deviceColumns+` FROM devices WHERE id=$1`, id))
}
func (s *Store) FindByCode(ctx context.Context, code string) (device.Device, error) {
	return scanDevice(s.db(ctx).QueryRow(ctx, `SELECT `+deviceColumns+` FROM devices WHERE code=$1`, code))
}
