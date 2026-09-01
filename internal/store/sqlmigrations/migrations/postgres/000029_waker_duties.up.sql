-- Waker duties ledger (Sprint DX12).
CREATE TABLE IF NOT EXISTS waker_duties (
    id           TEXT PRIMARY KEY,
    space_id     TEXT NOT NULL DEFAULT 'local',
    kind         TEXT NOT NULL,
    enabled      BOOLEAN NOT NULL DEFAULT TRUE,
    interval_ms  BIGINT NOT NULL DEFAULT 300000,
    config_json  TEXT NOT NULL DEFAULT '{}',
    next_run_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (space_id, kind)
);
CREATE INDEX IF NOT EXISTS idx_waker_duties_next ON waker_duties (enabled, next_run_at);

CREATE TABLE IF NOT EXISTS waker_duty_runs (
    id          TEXT PRIMARY KEY,
    space_id    TEXT NOT NULL DEFAULT 'local',
    duty_id     TEXT NOT NULL,
    kind        TEXT NOT NULL,
    status      TEXT NOT NULL,
    matched     INTEGER NOT NULL DEFAULT 0,
    flagged     INTEGER NOT NULL DEFAULT 0,
    canceled    INTEGER NOT NULL DEFAULT 0,
    summary     TEXT NOT NULL DEFAULT '',
    started_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_waker_duty_runs_space_started ON waker_duty_runs (space_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_waker_duty_runs_duty_started ON waker_duty_runs (duty_id, started_at DESC);

DO $rls$
DECLARE
    tbl text := 'waker_duties';
BEGIN
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', tbl);
    EXECUTE format('DROP POLICY IF EXISTS %I ON %I', 'ash_space_' || tbl, tbl);
    EXECUTE format(
        'CREATE POLICY %I ON %I USING (ash_rls_space_visible(space_id)) WITH CHECK (ash_rls_space_visible(space_id))',
        'ash_space_' || tbl,
        tbl
    );
END
$rls$;

DO $rls$
DECLARE
    tbl text := 'waker_duty_runs';
BEGIN
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', tbl);
    EXECUTE format('DROP POLICY IF EXISTS %I ON %I', 'ash_space_' || tbl, tbl);
    EXECUTE format(
        'CREATE POLICY %I ON %I USING (ash_rls_space_visible(space_id)) WITH CHECK (ash_rls_space_visible(space_id))',
        'ash_space_' || tbl,
        tbl
    );
END
$rls$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ash_app') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON waker_duties TO ash_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON waker_duty_runs TO ash_app;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ash_rls_tester') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON waker_duties TO ash_rls_tester;
        GRANT SELECT, INSERT, UPDATE, DELETE ON waker_duty_runs TO ash_rls_tester;
    END IF;
END
$$;
