CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS sites (
    id text PRIMARY KEY,
    name text NOT NULL,
    timezone text NOT NULL,
    responsible_unit text NOT NULL,
    created_at timestamptz NOT NULL,
    version bigint NOT NULL CHECK (version > 0)
);

CREATE TABLE IF NOT EXISTS zones (
    id text PRIMARY KEY,
    site_id text NOT NULL REFERENCES sites(id) ON DELETE RESTRICT,
    name text NOT NULL,
    purpose text NOT NULL DEFAULT '',
    version bigint NOT NULL CHECK (version > 0),
    UNIQUE (site_id, name)
);

CREATE TABLE IF NOT EXISTS monitoring_points (
    id text PRIMARY KEY,
    site_id text NOT NULL REFERENCES sites(id) ON DELETE RESTRICT,
    zone_id text NOT NULL REFERENCES zones(id) ON DELETE RESTRICT,
    name text NOT NULL,
    longitude double precision NOT NULL CHECK (longitude BETWEEN -180 AND 180),
    latitude double precision NOT NULL CHECK (latitude BETWEEN -90 AND 90),
    active boolean NOT NULL,
    version bigint NOT NULL CHECK (version > 0),
    UNIQUE (site_id, name)
);

CREATE INDEX IF NOT EXISTS monitoring_points_zone_idx
    ON monitoring_points (zone_id, active);

CREATE TABLE IF NOT EXISTS memberships (
    user_id text NOT NULL,
    site_id text NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    role text NOT NULL CHECK (role IN ('administrator', 'supervisor', 'dispatcher', 'maintainer', 'viewer')),
    PRIMARY KEY (user_id, site_id)
);
