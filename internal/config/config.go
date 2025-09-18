package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// AppConfig holds global configuration loaded from environment variables.
type AppConfig struct {
	// DSN for the application's central Postgres database where jobs and logs are stored.
	AppPostgresDSN string
	// Network address for the REST server, e.g. ":8080".
	RESTAddr string
	// Network address for the gRPC server, e.g. ":9090".
	GRPCAddr string
	// Path to a JSON file containing a map of customerId -> Postgres DSN
	CustomerMapPath string
	// Default number of worker goroutines processing jobs
	WorkerConcurrency int
	// Number of records to parse in a batch, default 0 means no batching
	ParseBatchSize int
}

// CustomerDBMap is a mapping from customerId to DSN string.
type CustomerDBMap map[string]string

// LoadConfig reads configuration from environment variables.
func LoadConfig() (AppConfig, error) {
	c := AppConfig{
		AppPostgresDSN:  os.Getenv("APP_DB_DSN"),
		RESTAddr:        valueOrDefault(os.Getenv("REST_ADDR"), ":8080"),
		GRPCAddr:        valueOrDefault(os.Getenv("GRPC_ADDR"), ":9090"),
		CustomerMapPath: valueOrDefault(os.Getenv("CUSTOMER_MAP_PATH"), "customer_map.json"),
		ParseBatchSize:  0,
	}

	if wc := os.Getenv("WORKER_CONCURRENCY"); wc != "" {
		// parse int safely
		var n int
		_, err := fmt.Sscanf(wc, "%d", &n)
		if err == nil && n > 0 {
			c.WorkerConcurrency = n
		}
	}
	if c.WorkerConcurrency == 0 {
		c.WorkerConcurrency = 4
	}

	if pbs := os.Getenv("PARSE_BATCH_SIZE"); pbs != "" {
		// parse int safely
		var n int
		_, err := fmt.Sscanf(pbs, "%d", &n)
		if err == nil && n >= 0 {
			c.ParseBatchSize = n
		}
	}

	if c.AppPostgresDSN == "" {
		return AppConfig{}, errors.New("APP_DB_DSN is required")
	}
	return c, nil
}

// LoadCustomerMap loads the map of customerId to DSN from a JSON file.
// File format: { "customer1": "postgres://...", "customer2": "postgres://..." }
func LoadCustomerMap(path string) (CustomerDBMap, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m CustomerDBMap
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func valueOrDefault(value, def string) string {
	if value == "" {
		return def
	}
	return value
}


