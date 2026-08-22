CREATE TABLE IF NOT EXISTS alerts (
    id text PRIMARY KEY,
    site_id text NOT NULL REFERENCES sites(id) ON DELETE RESTRICT,
    point_id text REFERENCES monitoring_points(id) ON DELETE RESTRICT,
    device_id text NOT NULL REFERENCES devices(id) ON DELETE RESTRICT,
    kind text NOT NULL CHECK (kind IN ('environmental-exceedance', 'device-offline', 'device-drift')),
    rule_id text NOT NULL DEFAULT '',
    rule_version bigint NOT NULL DEFAULT 0 CHECK (rule_version >= 0),
    status text NOT NULL CHECK (status IN ('open', 'acknowledged', 'dispatched', 'recovering', 'recovered', 'closed')),
    started_at timestamptz NOT NULL,
    last_signal_at timestamptz NOT NULL,
    recovered_at timestamptz,
    closed_at timestamptz,
    assignee_id text NOT NULL DEFAULT '',
    merge_key text NOT NULL,
    occurrence_count integer NOT NULL CHECK (occurrence_count > 0),
    version bigint NOT NULL CHECK (version > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS alerts_open_merge_key_idx
    ON alerts (merge_key) WHERE status <> 'closed';
CREATE INDEX IF NOT EXISTS alerts_site_queue_idx
    ON alerts (site_id, status, started_at DESC);

CREATE TABLE IF NOT EXISTS work_orders (
    id text PRIMARY KEY,
    alert_id text NOT NULL REFERENCES alerts(id) ON DELETE RESTRICT,
    assignee_id text NOT NULL,
    status text NOT NULL CHECK (status IN ('assigned', 'accepted', 'processing', 'resolved', 'verified', 'cancelled')),
    description text NOT NULL,
    created_at timestamptz NOT NULL,
    due_at timestamptz NOT NULL,
    resolved_at timestamptz,
    version bigint NOT NULL CHECK (version > 0),
    CHECK (due_at > created_at)
);

CREATE INDEX IF NOT EXISTS work_orders_assignee_queue_idx
    ON work_orders (assignee_id, status, due_at);
