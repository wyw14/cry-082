CREATE TABLE IF NOT EXISTS daily_reports (
    id text PRIMARY KEY,
    site_id text NOT NULL REFERENCES sites(id) ON DELETE RESTRICT,
    local_date text NOT NULL,
    timezone text NOT NULL,
    metrics jsonb NOT NULL,
    environmental_alerts integer NOT NULL CHECK (environmental_alerts >= 0),
    offline_alerts integer NOT NULL CHECK (offline_alerts >= 0),
    generated_at timestamptz NOT NULL,
    revision bigint NOT NULL CHECK (revision > 0),
    UNIQUE (site_id, local_date, revision)
);

CREATE TABLE IF NOT EXISTS regulatory_exports (
    id text PRIMARY KEY,
    site_id text NOT NULL REFERENCES sites(id) ON DELETE RESTRICT,
    format text NOT NULL CHECK (format IN ('csv', 'json')),
    report_ids jsonb NOT NULL,
    requested_by text NOT NULL,
    requested_at timestamptz NOT NULL,
    file_id text NOT NULL REFERENCES stored_files(id) ON DELETE RESTRICT,
    checksum text NOT NULL CHECK (length(checksum) = 64)
);

CREATE TABLE IF NOT EXISTS users (
    id text PRIMARY KEY,
    username text NOT NULL UNIQUE,
    password_hash bytea NOT NULL,
    display_name text NOT NULL,
    masked_phone text NOT NULL DEFAULT '',
    active boolean NOT NULL,
    failed_attempts integer NOT NULL DEFAULT 0 CHECK (failed_attempts >= 0),
    locked_until timestamptz,
    version bigint NOT NULL CHECK (version > 0)
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id text PRIMARY KEY,
    user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    digest text NOT NULL UNIQUE,
    issued_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL CHECK (expires_at > issued_at),
    revoked_at timestamptz,
    replaced_by text REFERENCES refresh_tokens(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS refresh_tokens_user_active_idx
    ON refresh_tokens (user_id, expires_at) WHERE revoked_at IS NULL;
