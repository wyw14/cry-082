CREATE TABLE IF NOT EXISTS devices (
    id text PRIMARY KEY,
    code text NOT NULL UNIQUE,
    model text NOT NULL,
    site_id text NOT NULL REFERENCES sites(id) ON DELETE RESTRICT,
    point_id text NOT NULL REFERENCES monitoring_points(id) ON DELETE RESTRICT,
    install_location text NOT NULL,
    network jsonb NOT NULL,
    status text NOT NULL CHECK (status IN ('registered', 'online', 'offline', 'maintenance', 'replaced', 'retired')),
    last_seen_at timestamptz,
    replacement_id text REFERENCES devices(id) ON DELETE RESTRICT,
    version bigint NOT NULL CHECK (version > 0)
);

CREATE INDEX IF NOT EXISTS devices_site_status_idx
    ON devices (site_id, status, code);

CREATE TABLE IF NOT EXISTS measurement_schemas (
    id text PRIMARY KEY,
    metric text NOT NULL,
    unit text NOT NULL,
    sampling_period_ns bigint NOT NULL CHECK (sampling_period_ns > 0),
    minimum double precision NOT NULL,
    maximum double precision NOT NULL CHECK (maximum > minimum),
    max_clock_skew_ns bigint NOT NULL CHECK (max_clock_skew_ns >= 0),
    version bigint NOT NULL CHECK (version > 0),
    UNIQUE (metric, unit, version)
);

CREATE TABLE IF NOT EXISTS observations (
    id text PRIMARY KEY,
    device_id text NOT NULL REFERENCES devices(id) ON DELETE RESTRICT,
    site_id text NOT NULL REFERENCES sites(id) ON DELETE RESTRICT,
    point_id text NOT NULL REFERENCES monitoring_points(id) ON DELETE RESTRICT,
    schema_id text NOT NULL REFERENCES measurement_schemas(id) ON DELETE RESTRICT,
    metric text NOT NULL,
    value double precision NOT NULL,
    unit text NOT NULL,
    sampled_at timestamptz NOT NULL,
    received_at timestamptz NOT NULL,
    corrected_at timestamptz,
    correction_of text REFERENCES observations(id) ON DELETE RESTRICT,
    quality text NOT NULL CHECK (quality IN ('accepted', 'suspect', 'quarantined')),
    quality_reasons jsonb NOT NULL,
    idempotency_key text NOT NULL UNIQUE,
    source_batch_id text NOT NULL,
    CHECK ((correction_of IS NULL AND corrected_at IS NULL) OR (correction_of IS NOT NULL AND corrected_at IS NOT NULL))
);

CREATE INDEX IF NOT EXISTS observations_site_time_idx
    ON observations (site_id, sampled_at DESC, id);
CREATE INDEX IF NOT EXISTS observations_device_schema_idx
    ON observations (device_id, schema_id, sampled_at DESC);
CREATE INDEX IF NOT EXISTS observations_batch_idx
    ON observations (source_batch_id, received_at DESC);
