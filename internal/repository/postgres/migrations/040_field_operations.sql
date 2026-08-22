CREATE TABLE IF NOT EXISTS maintenance_records (
    id text PRIMARY KEY,
    device_id text NOT NULL REFERENCES devices(id) ON DELETE RESTRICT,
    type text NOT NULL CHECK (type IN ('inspection', 'repair', 'calibration', 'replacement')),
    performed_by text NOT NULL,
    started_at timestamptz NOT NULL,
    completed_at timestamptz NOT NULL CHECK (completed_at >= started_at),
    reason text NOT NULL,
    result text NOT NULL,
    replacement_id text REFERENCES devices(id) ON DELETE RESTRICT,
    attachment_ids jsonb NOT NULL
);

CREATE INDEX IF NOT EXISTS maintenance_device_time_idx
    ON maintenance_records (device_id, completed_at DESC);

CREATE TABLE IF NOT EXISTS stored_files (
    id text PRIMARY KEY,
    site_id text NOT NULL REFERENCES sites(id) ON DELETE RESTRICT,
    display_name text NOT NULL,
    media_type text NOT NULL,
    purpose text NOT NULL CHECK (purpose IN ('maintenance-certificate', 'regulatory-export')),
    checksum text NOT NULL CHECK (length(checksum) = 64),
    size_bytes bigint NOT NULL CHECK (size_bytes > 0),
    created_by text NOT NULL,
    created_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS stored_files_site_created_idx
    ON stored_files (site_id, created_at DESC);

CREATE TABLE IF NOT EXISTS calibrations (
    id text PRIMARY KEY,
    device_id text NOT NULL REFERENCES devices(id) ON DELETE RESTRICT,
    schema_id text NOT NULL REFERENCES measurement_schemas(id) ON DELETE RESTRICT,
    reference_value double precision NOT NULL,
    observed_value double precision NOT NULL,
    offset_value double precision NOT NULL,
    performed_by text NOT NULL,
    performed_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL CHECK (expires_at > performed_at),
    certificate_id text REFERENCES stored_files(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS calibrations_device_expiry_idx
    ON calibrations (device_id, expires_at DESC);
