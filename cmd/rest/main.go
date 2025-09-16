package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/user/importer/internal/blob"
	"github.com/user/importer/internal/config"
	"github.com/user/importer/internal/db"
	"github.com/user/importer/internal/importer"
	"github.com/user/importer/internal/jobs"
)

func main() {
	var cfg config.AppConfig
	if yamlPath := os.Getenv("CONFIG_PATH"); yamlPath != "" {
		env := os.Getenv("CONFIG_ENV")
		if env == "" {
			env = "default"
		}
		c, err := config.LoadFromJSON(yamlPath, env)
		if err != nil {
			log.Fatalf("load json config: %v", err)
		}
		cfg = c
	} else {
		c, err := config.LoadConfig()
		if err != nil {
			log.Fatalf("load config: %v", err)
		}
		cfg = c
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	adb, err := db.ConnectAppDB(ctx, cfg.AppPostgresDSN)
	if err != nil {
		log.Fatalf("connect app db: %v", err)
	}
	defer adb.Close()

	custMap, err := config.LoadCustomerMap(cfg.CustomerMapPath)
	if err != nil {
		log.Fatalf("load customer map: %v", err)
	}

	jr := jobs.NewRepository(adb)
	imp := importer.NewService(blob.FileBlob{}, jr, custMap)

	// start worker
	go imp.Worker(ctx, cfg.WorkerConcurrency)

	// Serve swagger at /swagger and spec at /swagger/doc.json
	http.HandleFunc("/swagger", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<!DOCTYPE html>
		<html>
		<head>
		  <meta charset="utf-8"/>
		  <title>Swagger UI</title>
		  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
		</head>
		<body>
		  <div id="swagger-ui"></div>
		  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
		  <script>
		    window.onload = () => {
		      window.ui = SwaggerUIBundle({
		        url: '/swagger/doc.json',
		        dom_id: '#swagger-ui'
		      });
		    };
		  </script>
		</body>
		</html>`))
	})
	http.HandleFunc("/swagger/doc.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Prefer docker path if present, else local path
		if _, err := os.Stat("/app/api/swagger.json"); err == nil {
			http.ServeFile(w, r, "/app/api/swagger.json")
			return
		}
		http.ServeFile(w, r, "api/swagger.json")
	})

	http.HandleFunc("/enqueue", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			CustomerID  string `json:"customer_id"`
			ProductType string `json:"product_type"`
			BlobURI     string `json:"blob_uri"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		id, err := jr.Enqueue(r.Context(), req.CustomerID, req.ProductType, req.BlobURI)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"job_id": id})
	})

	log.Printf("REST listening on %s", cfg.RESTAddr)
	if err := http.ListenAndServe(cfg.RESTAddr, nil); err != nil && err != http.ErrServerClosed {
		log.Fatalf("http: %v", err)
	}
}


