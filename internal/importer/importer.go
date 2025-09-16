package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/user/importer/internal/blob"
	"github.com/user/importer/internal/db"
	"github.com/user/importer/internal/jobs"
	"github.com/user/importer/internal/parser"
	"github.com/user/importer/internal/products"
	"github.com/user/importer/internal/validate"
)

// Service orchestrates reading from blob, parsing, validating, and inserting into customer DB.
type Service struct {
	BlobReader blob.Reader
	JobRepo   *jobs.Repository
	CustMap   map[string]string
}

func NewService(br blob.Reader, jr *jobs.Repository, cust map[string]string) *Service {
	return &Service{BlobReader: br, JobRepo: jr, CustMap: cust}
}

// ProcessJob executes a single job end-to-end.
func (s *Service) ProcessJob(ctx context.Context, job *jobs.Job) error {
	if err := products.ValidateProductType(job.ProductType); err != nil {
		return err
	}
	dsn, ok := s.CustMap[job.CustomerID]
	if !ok {
		return fmt.Errorf("unknown customer id: %s", job.CustomerID)
	}
	rc, err := s.BlobReader.Open(ctx, job.BlobURI)
	if err != nil {
		return err
	}
	defer rc.Close()

	// Parse based on filename
	records, err := parser.Parse(filepath.Base(job.BlobURI), rc)
	if err != nil {
		return err
	}
	if err := validate.Records(job.ProductType, records); err != nil {
		return err
	}

	cdb, err := db.ConnectCustomerDB(ctx, dsn)
	if err != nil {
		return err
	}
	defer cdb.Pool.Close()

	table, err := products.TargetTableFor(job.ProductType)
	if err != nil {
		return err
	}
	if err := cdb.EnsureTargetTable(ctx, table); err != nil {
		return err
	}

	for _, rec := range records {
		b, err := json.Marshal(rec)
		if err != nil {
			return err
		}
		if err := cdb.InsertJSONB(ctx, table, b); err != nil {
			return err
		}
	}
	return nil
}

// Worker consumes jobs concurrently.
func (s *Service) Worker(ctx context.Context, concurrency int) {
	if concurrency <= 0 {
		concurrency = 1
	}
	jobCh := make(chan *jobs.Job)
	done := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		go func() {
			for j := range jobCh {
				if j == nil { // no job, continue polling
					continue
				}
				if err := s.ProcessJob(ctx, j); err != nil {
					_ = s.JobRepo.Fail(ctx, j.ID, err.Error())
				} else {
					_ = s.JobRepo.Complete(ctx, j.ID)
				}
			}
			done <- struct{}{}
		}()
	}

	// Poll loop
	go func() {
		defer close(jobCh)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				j, err := s.JobRepo.FetchAndStart(ctx)
				if err != nil {
					// best-effort log
					_ = s.JobRepo.Log(ctx, 0, "error", "FetchAndStart failed", []byte(fmt.Sprintf(`{"error":"%v"}`, err)))
					continue
				}
				jobCh <- j
			}
		}
	}()

	// Wait for workers to exit when context is cancelled
	for i := 0; i < concurrency; i++ {
		<-done
	}
}


