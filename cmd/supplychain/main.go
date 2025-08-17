package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rajasatyajit/SupplyChain/config"
	"github.com/rajasatyajit/SupplyChain/internal/api"
	"github.com/rajasatyajit/SupplyChain/internal/classifier"
	"github.com/rajasatyajit/SupplyChain/internal/database"
	"github.com/rajasatyajit/SupplyChain/internal/geocoder"
	"github.com/rajasatyajit/SupplyChain/internal/logger"
	"github.com/rajasatyajit/SupplyChain/internal/metrics"
	middlewares "github.com/rajasatyajit/SupplyChain/internal/middleware"
	"github.com/rajasatyajit/SupplyChain/internal/pipeline"
	"github.com/rajasatyajit/SupplyChain/internal/store"
)

// Version information (set by build)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger.Init(cfg.Logging.Level, cfg.Logging.Format)
	logger.Info("Starting SupplyChain application",
		"version", Version,
		"build_time", BuildTime,
		"git_commit", GitCommit,
	)

	// Initialize metrics
	if cfg.Metrics.Enabled {
		metrics.Init()
		logger.Info("Metrics enabled", "port", cfg.Metrics.Port)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database
	db, err := database.New(ctx, cfg.Database)
	if err != nil {
		logger.Fatal("Failed to initialize database", "error", err)
	}
	defer db.Close(ctx)

	// Initialize store
	alertStore := store.New(db)

	// Initialize AI components
	alertClassifier := classifier.New()
	geo := geocoder.New()

	// Initialize pipeline
	alertPipeline := pipeline.New(alertStore, alertClassifier, geo, cfg.Pipeline)
	
	// Start pipeline in background
	go func() {
		if err := alertPipeline.Run(ctx); err != nil {
			logger.Error("Pipeline error", "error", err)
		}
	}()

	// Setup HTTP server
	r := chi.NewRouter()
	
	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middlewares.Logging)
	r.Use(middlewares.Metrics)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(cfg.Server.ReadTimeout))
	r.Use(middlewares.Security)

	// Initialize API handlers
	apiHandler := api.NewHandler(alertStore, Version, BuildTime, GitCommit)
	apiHandler.RegisterRoutes(r)

	// Metrics endpoint
	if cfg.Metrics.Enabled {
		go startMetricsServer(cfg.Metrics.Port, cfg.Metrics.Path)
	}

	// HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Starting HTTP server", "address", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", "error", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.GracefulShutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}

	logger.Info("Server exited")
}

func startMetricsServer(port int, path string) {
	mux := http.NewServeMux()
	mux.Handle(path, metrics.Handler())
	
	addr := fmt.Sprintf(":%d", port)
	logger.Info("Starting metrics server", "address", addr, "path", path)
	
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Error("Metrics server failed", "error", err)
	}
}