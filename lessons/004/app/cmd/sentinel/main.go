// Package main is the entry point for posadev-go-sentinel.
// This application watches Kubernetes deployments and publishes
// deployment information to S3 for service discovery.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/posadev/go-sentinel/internal/config"
	"github.com/posadev/go-sentinel/internal/discovery"
	"github.com/posadev/go-sentinel/internal/storage"
)

func main() {
	// -------------------------------------------------------------------------
	// Initialize Structured Logging
	// -------------------------------------------------------------------------

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("application failed", "error", err)
		os.Exit(1)
	}
}

// run contains the main application logic, separated from main() for testability.
func run(logger *slog.Logger) error {
	// -------------------------------------------------------------------------
	// Load Configuration
	// -------------------------------------------------------------------------

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	logger.Info("posadev-go-sentinel starting",
		"bucket", cfg.S3Bucket,
		"file", cfg.S3FileName,
		"ignored_namespaces", cfg.IgnoredNamespaces,
	)

	// -------------------------------------------------------------------------
	// Initialize Dependencies
	// -------------------------------------------------------------------------

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize S3 storage client
	s3Client, err := storage.NewS3Client(ctx, cfg.S3Bucket, cfg.S3FileName)
	if err != nil {
		return fmt.Errorf("creating s3 client: %w", err)
	}

	// Initialize Kubernetes discovery client
	disco, err := discovery.New(logger, cfg.IgnoredNamespaces)
	if err != nil {
		return fmt.Errorf("creating discovery client: %w", err)
	}

	// -------------------------------------------------------------------------
	// Setup Graceful Shutdown
	// -------------------------------------------------------------------------

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 1)

	// -------------------------------------------------------------------------
	// Start Watching Deployments
	// -------------------------------------------------------------------------

	go func() {
		if err := disco.Watch(ctx, func(state discovery.WorldState) error {
			logger.Info("deployment state changed", "namespaces", len(state))

			if err := s3Client.Upload(ctx, state); err != nil {
				logger.Error("failed to upload state to S3", "error", err)
				return err
			}

			logger.Info("successfully uploaded discovery.json to S3")
			return nil
		}); err != nil {
			errChan <- fmt.Errorf("watching deployments: %w", err)
		}
	}()

	// -------------------------------------------------------------------------
	// Block Until Shutdown Signal or Error
	// -------------------------------------------------------------------------

	select {
	case sig := <-shutdown:
		logger.Info("shutdown signal received", "signal", sig)
	case err := <-errChan:
		return err
	}

	logger.Info("posadev-go-sentinel stopped gracefully")
	return nil
}
