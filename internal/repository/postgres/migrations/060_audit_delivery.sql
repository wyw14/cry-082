CREATE TABLE IF NOT EXISTS audit_entries (
    id text PRIMARY KEY,
    site_id text REFERENCES sites(id) ON DELETE RESTRICT,
    actor_id text NOT NULL,
    source text NOT NULL,
    action text NOT NULL,
    resource text NOT NULL,
    resource_id text NOT NULL,
    before_state jsonb NOT NULL,
    after_state jsonb NOT NULL,
    reason text NOT NULL,
    request_id text NOT NULL,
    occurred_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS audit_resource_idx
    ON audit_entries (resource, resource_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS audit_site_time_idx
    ON audit_entries (site_id, occurred_at DESC);

CREATE TABLE IF NOT EXISTS outbox_events (
    id text PRIMARY KEY,
    topic text NOT NULL,
    aggregate_id text NOT NULL,
    payload jsonb NOT NULL,
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    available_at timestamptz NOT NULL,
    published_at timestamptz,
    last_error text NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS outbox_pending_idx
    ON outbox_events (available_at, attempts) WHERE published_at IS NULL;
