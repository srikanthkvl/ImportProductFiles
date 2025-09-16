package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/user/importer/internal/blob"
	"github.com/user/importer/internal/config"
	"github.com/user/importer/internal/db"
	"github.com/user/importer/internal/importer"
	"github.com/user/importer/internal/jobs"
	"github.com/user/importer/internal/grpcsvc"
)

// Simple gRPC service definition without proto files using grpc.ServiceRegistrar via reflection is limited;
// For demo, we keep the server running and log that gRPC would be implemented with protobufs.

func main() {
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
	_ = importer.NewService(blob.FileBlob{}, jr, custMap)

	l, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	s := grpc.NewServer()
	// Register our manual service descriptor
	grpcServer := grpcsvc.New(jr)
	s.RegisterService(&grpcsvc.ImporterServiceDesc, grpcServer)
	reflection.Register(s)
	log.Printf("gRPC listening on %s. Service: importer.Importer/Enqueue (Struct).", cfg.GRPCAddr)
	if err := s.Serve(l); err != nil {
		log.Fatalf("grpc: %v", err)
	}
}


