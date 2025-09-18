package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type jsonEnv struct {
	AppPostgresDSN    string `json:"app_db_dsn"`
	RESTAddr          string `json:"rest_addr"`
	GRPCAddr          string `json:"grpc_addr"`
	CustomerMapPath   string `json:"customer_map_path"`
	WorkerConcurrency int    `json:"worker_concurrency"`
	ParseBatchSize    int    `json:"parse_batch_size"`
}

// LoadFromJSON loads configuration from a JSON file with environment sections.
// Structure options:
// 1) Top-level env objects: { "default": {...}, "docker": {...} }
// 2) Or nested: { "environments": { "default": {...}, ... } }
func LoadFromJSON(path, env string) (AppConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return AppConfig{}, err
	}
	// Try flat map first
	var flat map[string]jsonEnv
	if err := json.Unmarshal(b, &flat); err == nil && len(flat) > 0 && hasAnyEnv(flat) {
		return mergeJSONEnvs(flat, env)
	}
	// Try nested under environments
	var nested struct {
		Environments map[string]jsonEnv `json:"environments"`
	}
	if err := json.Unmarshal(b, &nested); err != nil {
		return AppConfig{}, err
	}
	if len(nested.Environments) == 0 {
		return AppConfig{}, errors.New("no environments found in config json")
	}
	return mergeJSONEnvs(nested.Environments, env)
}

func hasAnyEnv(m map[string]jsonEnv) bool {
	for range m {
		return true
	}
	return false
}

func mergeJSONEnvs(envs map[string]jsonEnv, env string) (AppConfig, error) {
	d := envs["default"]
	sel := envs[env]
	out := jsonEnv{
		AppPostgresDSN:    pickString(sel.AppPostgresDSN, d.AppPostgresDSN),
		RESTAddr:          pickString(sel.RESTAddr, d.RESTAddr, ":8080"),
		GRPCAddr:          pickString(sel.GRPCAddr, d.GRPCAddr, ":9090"),
		CustomerMapPath:   pickString(sel.CustomerMapPath, d.CustomerMapPath, "customer_map.json"),
		WorkerConcurrency: pickInt(sel.WorkerConcurrency, d.WorkerConcurrency, 4),
		ParseBatchSize:  pickInt(sel.ParseBatchSize, d.ParseBatchSize, 0),
	}
	if out.AppPostgresDSN == "" {
		return AppConfig{}, fmt.Errorf("app_db_dsn is required in JSON for env '%s'", env)
	}
	return AppConfig{
		AppPostgresDSN:    out.AppPostgresDSN,
		RESTAddr:          out.RESTAddr,
		GRPCAddr:          out.GRPCAddr,
		CustomerMapPath:   out.CustomerMapPath,
		WorkerConcurrency: out.WorkerConcurrency,
		ParseBatchSize:    out.ParseBatchSize,
	}, nil
}

func pickString(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func pickInt(values ...int) int {
	for _, v := range values {
		if v > 0 {
			return v
		}
	}
	return 0
}


