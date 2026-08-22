package postgres

import (
	"context"
	"errors"

	"github.com/wyw14/cry-082/internal/domain/site"
)

var ErrStaleTopologyVersion = errors.New("topology version is not newer than stored version")

func (s *Store) applyTopologyMutation(ctx context.Context, statement string, arguments ...any) error {
	command, err := s.db(ctx).Exec(ctx, statement, arguments...)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return ErrStaleTopologyVersion
	}
	return nil
}

func (s *Store) SaveSite(ctx context.Context, value site.Site) error {
	return s.applyTopologyMutation(ctx, `
		MERGE INTO sites AS current
		USING (VALUES ($1::text,$2::text,$3::text,$4::text,$5::timestamptz,$6::bigint))
			AS incoming(id,name,timezone,responsible_unit,created_at,version)
		ON current.id = incoming.id
		WHEN MATCHED AND current.version < incoming.version THEN
			UPDATE SET name=incoming.name, timezone=incoming.timezone,
				responsible_unit=incoming.responsible_unit, version=incoming.version
		WHEN NOT MATCHED THEN
			INSERT(id,name,timezone,responsible_unit,created_at,version)
			VALUES(incoming.id,incoming.name,incoming.timezone,incoming.responsible_unit,incoming.created_at,incoming.version)`,
		value.ID, value.Name, value.Timezone, value.ResponsibleUnit, value.CreatedAt, value.Version)
}

func (s *Store) SaveZone(ctx context.Context, value site.Zone) error {
	return s.applyTopologyMutation(ctx, `
		MERGE INTO zones AS current
		USING (VALUES ($1::text,$2::text,$3::text,$4::text,$5::bigint))
			AS incoming(id,site_id,name,purpose,version)
		ON current.id = incoming.id
		WHEN MATCHED AND current.version < incoming.version THEN
			UPDATE SET site_id=incoming.site_id, name=incoming.name,
				purpose=incoming.purpose, version=incoming.version
		WHEN NOT MATCHED THEN
			INSERT(id,site_id,name,purpose,version)
			VALUES(incoming.id,incoming.site_id,incoming.name,incoming.purpose,incoming.version)`,
		value.ID, value.SiteID, value.Name, value.Purpose, value.Version)
}

func (s *Store) SavePoint(ctx context.Context, value site.MonitoringPoint) error {
	return s.applyTopologyMutation(ctx, `
		MERGE INTO monitoring_points AS current
		USING (VALUES ($1::text,$2::text,$3::text,$4::text,$5::double precision,$6::double precision,$7::boolean,$8::bigint))
			AS incoming(id,site_id,zone_id,name,longitude,latitude,active,version)
		ON current.id = incoming.id
		WHEN MATCHED AND current.version < incoming.version THEN
			UPDATE SET site_id=incoming.site_id, zone_id=incoming.zone_id,
				name=incoming.name, longitude=incoming.longitude, latitude=incoming.latitude,
				active=incoming.active, version=incoming.version
		WHEN NOT MATCHED THEN
			INSERT(id,site_id,zone_id,name,longitude,latitude,active,version)
			VALUES(incoming.id,incoming.site_id,incoming.zone_id,incoming.name,incoming.longitude,incoming.latitude,incoming.active,incoming.version)`,
		value.ID, value.SiteID, value.ZoneID, value.Name, value.Longitude, value.Latitude, value.Active, value.Version)
}

func (s *Store) SaveMembership(ctx context.Context, value site.Membership) error {
	_, err := s.db(ctx).Exec(ctx, `
		MERGE INTO memberships AS current
		USING (VALUES ($1::text,$2::text,$3::text)) AS incoming(user_id,site_id,role)
		ON current.user_id = incoming.user_id AND current.site_id = incoming.site_id
		WHEN MATCHED AND current.role IS DISTINCT FROM incoming.role THEN
			UPDATE SET role=incoming.role
		WHEN NOT MATCHED THEN
			INSERT(user_id,site_id,role) VALUES(incoming.user_id,incoming.site_id,incoming.role)`,
		value.UserID, value.SiteID, value.Role)
	return err
}

func (s *Store) Membership(ctx context.Context, userID, siteID string) (site.Membership, error) {
	var value site.Membership
	err := s.db(ctx).QueryRow(ctx, `
		SELECT user_id,site_id,role
		FROM memberships
		WHERE user_id=$1 AND site_id=$2`, userID, siteID).
		Scan(&value.UserID, &value.SiteID, &value.Role)
	return value, err
}
