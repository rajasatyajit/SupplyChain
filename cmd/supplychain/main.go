package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	chi "github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rajasatyajit/SupplyChain/config"
	"github.com/rajasatyajit/SupplyChain/internal/api"
	"github.com/rajasatyajit/SupplyChain/internal/auth"
	"github.com/rajasatyajit/SupplyChain/internal/classifier"
	"github.com/rajasatyajit/SupplyChain/internal/database"
	"github.com/rajasatyajit/SupplyChain/internal/geocoder"
	"github.com/rajasatyajit/SupplyChain/internal/logger"
	"github.com/rajasatyajit/SupplyChain/internal/metrics"
	middlewares "github.com/rajasatyajit/SupplyChain/internal/middleware"
	"github.com/rajasatyajit/SupplyChain/internal/pipeline"
	"github.com/rajasatyajit/SupplyChain/internal/ratelimit"
	"github.com/rajasatyajit/SupplyChain/internal/store"
	"github.com/rajasatyajit/SupplyChain/internal/usage"
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

	// Attach DB to base context for auth lookups
	ctx = auth.WithDB(ctx, db)

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

	// API Key auth (feature-flagged)
	r.Use(middlewares.APIKeyAuth(cfg.Auth))

	// Redis-backed rate/quota if configured, else fallback to in-memory
	var rlManager *ratelimit.Manager
	if cfg.Redis.URL != "" {
		mgr, err := ratelimit.NewManager(cfg.Redis.URL)
		if err == nil {
			rlManager = mgr
			logger.Info("Redis rate limiter enabled")
		} else {
			logger.Error("Failed to init Redis limiter, using in-memory", "error", err)
		}
	}
	if rlManager != nil {
		// Inject subscription checker using DB
		middlewares.SetSubscriptionChecker(func(ctx interface{}, accountID string) bool {
			row := db.QueryRow(context.Background(), "SELECT 1 FROM subscriptions WHERE account_id=$1 AND status IN ('active','trialing') LIMIT 1", accountID)
			var one int
			if s, ok := row.(interface{ Scan(dest ...any) error }); ok {
				if err := s.Scan(&one); err == nil {
					return true
				}
			}
			return false
		})
		r.Use(middlewares.RedisRateQuotaEnforcer(rlManager))
	} else {
		r.Use(middlewares.RateQuotaEnforcer())
	}

	// Note: Admin endpoints are protected via header checks within handlers.

	// Initialize API handlers
	apiHandler := api.NewHandler(alertStore, db, cfg.Admin.AdminSecret, Version, BuildTime, GitCommit)
	// pass limiter to api package for usage endpoint access
	api.SetRateLimiter(rlManager)
	apiHandler.RegisterRoutes(r)

	// Serve admin UI static files at /admin (owner will input secret in page)
	r.Handle("GET /admin/*", http.StripPrefix("/admin/", http.FileServer(http.Dir("admin-ui"))))

	// Start usage aggregator (flush Redis to Postgres)
	if rlManager != nil {
		usage.StartAggregator(ctx, db, rlManager)
	}

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
