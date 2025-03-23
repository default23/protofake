package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/default23/protofake/config"
	"github.com/default23/protofake/server"
)

func main() {
	conf, err := config.Parse()
	if err != nil {
		log.Fatalf("failed to parse configuration: %v", err)
	}

	logger := NewLogger(conf.Logger)

	descriptorsDir := filepath.Join(conf.DataDir, "descriptors")
	descriptors, err := parseDescriptorFiles(conf.DescriptorExtensions, descriptorsDir)
	if err != nil {
		log.Fatalf("failed to parse descriptor files: %v", err)
	}

	logger.Info("processed descriptors dir", "count", len(descriptors), "dir", descriptorsDir)
	if len(descriptors) == 0 {
		log.Fatal("no descriptors found in the directory, protofake can't work without protobuf definitions, which should be mocked")
	}

	mappingsDir := filepath.Join(conf.DataDir, "mappings")
	mappings, err := parseMappingFiles(mappingsDir)
	if err != nil {
		log.Fatalf("failed to parse mapping files: %v", err)
	}

	logger.Info("processed mappings dir", "count", len(mappings), "dir", mappingsDir)

	srv, err := server.New(conf.GRPC)
	if err != nil {
		log.Fatalf("failed to create gRPC server: %s", err)
	}

	srv.SetMappings(mappings)
	for _, d := range descriptors {
		if err = srv.Register(d); err != nil {
			log.Fatalf("failed to register gRPC services: %s", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	if conf.WatchMappingsChanges {
		err = watchChanges(ctx, mappingsDir, func() {
			logger.Info("reloading mappings...")

			mappings, err = parseMappingFiles(mappingsDir)
			if err != nil {
				logger.Error("failed to reload mappings", "error", err)
				return
			}

			srv.SetMappings(mappings)
			logger.Info("reloaded mappings", "count", len(mappings))
		})
		if err != nil {
			log.Fatalf("failed to configure the mapping files watching: %s", err)
		}
	}

	srv.Run()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	cancel()
	logger.Info("shutting down application...")

	if err = srv.Close(); err != nil {
		log.Fatalf("API Server failed to shut down: %s", err)
	}
}
