package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/user/importer/internal/blob"
	"github.com/user/importer/internal/config"
	"github.com/user/importer/internal/db"
	"github.com/user/importer/internal/importer"
	"github.com/user/importer/internal/jobs"
)

func main() {
	var customerID, productType, blobURI string
	flag.StringVar(&customerID, "customer", "", "customer id")
	flag.StringVar(&productType, "product", "", "product type (users|organizations|courses)")
	flag.StringVar(&blobURI, "file", "", "file path or file:// URI")
	flag.Parse()

	var cfg config.AppConfig
	if yamlPath := os.Getenv("CONFIG_PATH"); yamlPath != "" {
		env := os.Getenv("CONFIG_ENV")
		if env == "" { env = "default" }
		c, err := config.LoadFromJSON(yamlPath, env)
		if err != nil { log.Fatalf("load json config: %v", err) }
		cfg = c
	} else {
		c, err := config.LoadConfig()
		if err != nil { log.Fatalf("load config: %v", err) }
		cfg = c
	}
	ctx := context.Background()
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

	// Enqueue and run worker in foreground until context cancelled; or if flags omitted, enqueue only
	if customerID != "" && productType != "" && blobURI != "" {
		if _, err := jr.Enqueue(ctx, customerID, productType, blobURI); err != nil {
			log.Fatalf("enqueue: %v", err)
		}
	}

	imp.Worker(ctx, cfg.WorkerConcurrency)
}


