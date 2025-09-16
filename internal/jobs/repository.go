package jobs

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/user/importer/internal/db"
)

type Status string

const (
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
)

type Job struct {
	ID          int64
	CustomerID  string
	ProductType string
	BlobURI     string
	Status      Status
}

type Repository struct {
	DB *db.AppDB
}

func NewRepository(adb *db.AppDB) *Repository { return &Repository{DB: adb} }

func (r *Repository) Enqueue(ctx context.Context, customerID, productType, blobURI string) (int64, error) {
	row := r.DB.Pool.QueryRow(ctx, `INSERT INTO import_jobs(customer_id, product_type, blob_uri, status) VALUES($1,$2,$3,'queued') RETURNING id`, customerID, productType, blobURI)
	var id int64
	return id, row.Scan(&id)
}

func (r *Repository) FetchAndStart(ctx context.Context) (*Job, error) {
	tx, err := r.DB.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var j Job
	// Select for update skip locked to prevent contention
	row := tx.QueryRow(ctx, `
UPDATE import_jobs SET status='running', started_at = now()
WHERE id = (
  SELECT id FROM import_jobs WHERE status='queued' ORDER BY created_at ASC FOR UPDATE SKIP LOCKED LIMIT 1
) RETURNING id, customer_id, product_type, blob_uri, status
`)
	if err := row.Scan(&j.ID, &j.CustomerID, &j.ProductType, &j.BlobURI, &j.Status); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &j, nil
}

func (r *Repository) Complete(ctx context.Context, jobID int64) error {
	_, err := r.DB.Pool.Exec(ctx, `UPDATE import_jobs SET status='succeeded', finished_at=now() WHERE id=$1`, jobID)
	return err
}

func (r *Repository) Fail(ctx context.Context, jobID int64, errText string) error {
	_, err := r.DB.Pool.Exec(ctx, `UPDATE import_jobs SET status='failed', finished_at=now(), error_text=$2 WHERE id=$1`, jobID, errText)
	return err
}

func (r *Repository) Log(ctx context.Context, jobID int64, level, message string, contextJSON []byte) error {
	_, err := r.DB.Pool.Exec(ctx, `INSERT INTO import_logs(job_id, level, message, context) VALUES ($1,$2,$3,$4)`, jobID, level, message, contextJSON)
	return err
}


