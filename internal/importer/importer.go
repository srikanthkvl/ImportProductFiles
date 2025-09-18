package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

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
	JobRepo    *jobs.Repository
	CustMap    map[string]string
}

func NewService(br blob.Reader, jr *jobs.Repository, cust map[string]string) *Service {
	return &Service{BlobReader: br, JobRepo: jr, CustMap: cust}
}

// ProcessJob executes a single job end-to-end.
func (s *Service) ProcessJob(ctx context.Context, job *jobs.Job) error {
	startedAt := time.Now()
	logMessage := fmt.Sprintf(`{
		"customer_id": %v,
		"product_type": %v,
		"blob_uri": %v,
		"started_at": %v,
	}`, job.CustomerID, job.ProductType, job.BlobURI, startedAt)

	logMsgBytes, err := json.Marshal(logMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal log message: %w", err)
	}

	errs := s.JobRepo.Log(ctx, job.ID, "info", "job started", logMsgBytes)
	if errs != nil {
		fmt.Println("Failed to log job start:", errs)
		return fmt.Errorf("failed to log job start: %w", errs)
	}

	if err := products.ValidateProductType(job.ProductType); err != nil {
		return err
	}
	dsn, ok := s.CustMap[job.CustomerID]
	if !ok {
		return fmt.Errorf("unknown customer id: %s", job.CustomerID)
	}
	rc, err := s.BlobReader.Open(ctx, job.BlobURI)
	if err != nil {
		return fmt.Errorf("failed to open blob %s: %w", job.BlobURI, err)
	}
	defer rc.Close()

	cdb, err := db.ConnectCustomerDB(ctx, dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to customer db: %w", err)
	}
	defer cdb.Pool.Close()

	table, err := products.TargetTableFor(job.ProductType)
	if err != nil {
		return fmt.Errorf("failed to get target table for product type %s: %w", job.ProductType, err)
	}
	if err := cdb.EnsureTargetTable(ctx, table); err != nil {
		return fmt.Errorf("failed to ensure target table %s: %w", table, err)
	}

	s.JobRepo.Log(ctx, job.ID, "info", "target table ensured", []byte(fmt.Sprintf(`{"table": %s}`, table)))

	batchSize := 1000
	if v := ctx.Value("parse_batch_size"); v != nil {
		if n, ok := v.(int); ok && n > 0 {
			batchSize = n
		}
	}

	processed := 0
	handler := func(records []parser.Record) error {
		if err := validate.Records(job.ProductType, records); err != nil {
			return err
		}
		fmt.Printf("Inserting batch of %d records into customer: %s, table: %s\n", len(records), job.CustomerID, table)

		for _, rec := range records {
			b, err := json.Marshal(rec)
			if err != nil {
				return err
			}
			if err := cdb.InsertJSONB(ctx, table, b); err != nil {
				return err
			}
			processed++
		}
		return nil
	}

	fmt.Printf("Starting to parse blob %s for customer %s, product type %s\n", job.BlobURI, job.CustomerID, job.ProductType)

	if err := parser.ParseBatches(filepath.Base(job.BlobURI), rc, batchSize, handler); err != nil {
		s.JobRepo.Log(ctx, job.ID, "error", "job failed during parsing/processing", []byte(fmt.Sprintf(`{"error":"%v"}`, err.Error())))
		return fmt.Errorf("failed to parse/process blob %s: %w", job.BlobURI, err)
	}

	completedAt := time.Now()
	logMsg := fmt.Sprintf(`{"processed_records":"` + strconv.Itoa(processed) + `", "completed_at": "` + completedAt.Format(time.RFC3339) + `", duration_sec: "` + strconv.FormatFloat(completedAt.Sub(startedAt).Seconds(), 'f', 2, 64) + `"}`)
	logMsgBytes, err = json.Marshal(logMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal completion log message: %w", err)
	}

	err = s.JobRepo.Log(ctx, job.ID, "info", "Batch job completed successfully", logMsgBytes)
	if err != nil {
		return fmt.Errorf("failed to log completion: %w", err)
	}

	fmt.Printf("Job completed successfully, processed %d records\n", processed)
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
					time.Sleep(60 * time.Second)
					continue
				}
				if err := s.ProcessJob(ctx, j); err != nil {
					_ = s.JobRepo.Fail(ctx, j.ID, err.Error())
				} else {
					_ = s.JobRepo.Complete(ctx, j.ID)
				}
			}

			fmt.Println("worker done")
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
					_ = s.JobRepo.Log(ctx, 0, "error", "FetchAndStart failed", []byte(`{"error":"`+err.Error()+`"}`))
					continue
				}

				// got a job
				fmt.Println("Fetched job:", j)

				jobCh <- j
			}
		}
	}()

	// Wait for workers to exit when context is cancelled
	for i := 0; i < concurrency; i++ {
		<-done
	}
}
