package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AppDB wraps the application's central Postgres connection pool.
type AppDB struct {
	Pool *pgxpool.Pool
}

// ConnectAppDB connects to the central application Postgres using the provided DSN.
func ConnectAppDB(ctx context.Context, dsn string) (*AppDB, error) {
	if dsn == "" {
		return nil, errors.New("empty DSN")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 10
	cfg.MinConns = 0
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	adb := &AppDB{Pool: pool}
	if err := adb.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return adb, nil
}

// Close closes the pool.
func (a *AppDB) Close() {
	if a == nil || a.Pool == nil {
		return
	}
	a.Pool.Close()
}

// migrate creates required tables if not present.
func (a *AppDB) migrate(ctx context.Context) error {
	// import_jobs: id, customer_id, product_type, blob_uri, status, created_at, updated_at, started_at, finished_at, error_text
	// import_logs: id, job_id, level, message, created_at, context JSONB
	_, err := a.Pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS import_jobs (
	id BIGSERIAL PRIMARY KEY,
	customer_id TEXT NOT NULL,
	product_type TEXT NOT NULL,
	blob_uri TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'queued',
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	started_at TIMESTAMPTZ,
	finished_at TIMESTAMPTZ,
	error_text TEXT
);

CREATE INDEX IF NOT EXISTS idx_import_jobs_status ON import_jobs(status);

CREATE TABLE IF NOT EXISTS import_logs (
	id BIGSERIAL PRIMARY KEY,
	job_id BIGINT REFERENCES import_jobs(id) ON DELETE CASCADE,
	level TEXT NOT NULL,
	message TEXT NOT NULL,
	context JSONB,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- trigger to auto-update updated_at
DO $$
BEGIN
IF NOT EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'update_updated_at') THEN
  CREATE OR REPLACE FUNCTION update_updated_at()
  RETURNS TRIGGER AS $trg$
  BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
  END;
  $trg$ LANGUAGE plpgsql;
END IF;
END $$;

DROP TRIGGER IF EXISTS trg_update_updated_at ON import_jobs;
CREATE TRIGGER trg_update_updated_at BEFORE UPDATE ON import_jobs
FOR EACH ROW EXECUTE FUNCTION update_updated_at();

`)
	return err
}
