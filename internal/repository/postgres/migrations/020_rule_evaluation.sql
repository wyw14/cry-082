CREATE TABLE IF NOT EXISTS rule_versions (
    rule_id text NOT NULL,
    version bigint NOT NULL CHECK (version > 0),
    site_id text NOT NULL REFERENCES sites(id) ON DELETE RESTRICT,
    name text NOT NULL,
    timezone text NOT NULL,
    conditions jsonb NOT NULL,
    require_all boolean NOT NULL,
    duration_ns bigint NOT NULL CHECK (duration_ns >= 0),
    merge_window_ns bigint NOT NULL CHECK (merge_window_ns >= 0),
    late_grace_ns bigint NOT NULL CHECK (late_grace_ns >= 0),
    effective_from timestamptz NOT NULL,
    status text NOT NULL CHECK (status IN ('draft', 'active', 'superseded', 'retired')),
    created_by text NOT NULL,
    created_at timestamptz NOT NULL,
    PRIMARY KEY (rule_id, version)
);

CREATE INDEX IF NOT EXISTS rule_versions_site_active_idx
    ON rule_versions (site_id, status, effective_from DESC);

CREATE TABLE IF NOT EXISTS evaluations (
    id bigserial PRIMARY KEY,
    rule_id text NOT NULL,
    rule_version bigint NOT NULL,
    timezone text NOT NULL,
    site_id text NOT NULL REFERENCES sites(id) ON DELETE RESTRICT,
    point_id text NOT NULL REFERENCES monitoring_points(id) ON DELETE RESTRICT,
    window_start timestamptz,
    window_end timestamptz,
    matched boolean NOT NULL,
    observation_ids jsonb NOT NULL,
    conclusion text NOT NULL,
    recalculation_id text NOT NULL DEFAULT '',
    evaluated_at timestamptz NOT NULL,
    FOREIGN KEY (rule_id, rule_version) REFERENCES rule_versions(rule_id, version) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS evaluations_rule_point_idx
    ON evaluations (rule_id, rule_version, point_id, evaluated_at DESC);

CREATE TABLE IF NOT EXISTS recalculations (
    id text PRIMARY KEY,
    site_id text NOT NULL REFERENCES sites(id) ON DELETE RESTRICT,
    rule_id text NOT NULL,
    from_version bigint NOT NULL CHECK (from_version > 0),
    to_version bigint NOT NULL CHECK (to_version > 0 AND to_version <> from_version),
    window_start timestamptz NOT NULL,
    window_end timestamptz NOT NULL CHECK (window_end > window_start),
    reason text NOT NULL,
    requested_by text NOT NULL,
    requested_at timestamptz NOT NULL,
    status text NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
    processed_points integer NOT NULL DEFAULT 0 CHECK (processed_points >= 0),
    failure text NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS recalculations_site_status_idx
    ON recalculations (site_id, status, requested_at);
