package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rajasatyajit/SupplyChain/config"
	"github.com/rajasatyajit/SupplyChain/internal/logger"
	"github.com/rajasatyajit/SupplyChain/internal/metrics"
)

// DB represents a database connection
type DB struct {
	pool *pgxpool.Pool
	cfg  config.DatabaseConfig
}

// Config exposes config for dependent services
func (d *DB) Config() *config.Config { return &config.Config{Database: d.cfg} }

// New creates a new database connection
func New(ctx context.Context, cfg config.DatabaseConfig) (*DB, error) {
	if cfg.URL == "" {
		logger.Info("DATABASE_URL not set; using in-memory store only")
		return &DB{pool: nil, cfg: cfg}, nil
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse db url: %w", err)
	}

	// Configure connection pool
	poolCfg.MaxConns = int32(cfg.MaxConns)
	poolCfg.MinConns = int32(cfg.MinConns)
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime

	// Add connection callbacks for metrics
	poolCfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		logger.Debug("Database connection established")
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	db := &DB{pool: pool, cfg: cfg}

	// Start metrics collection
	go db.collectMetrics(ctx)

	logger.Info("Database connection established",
		"max_conns", cfg.MaxConns,
		"min_conns", cfg.MinConns,
	)

	return db, nil
}

// Close closes the database connection
func (d *DB) Close(ctx context.Context) {
	if d.pool != nil {
		d.pool.Close()
		logger.Info("Database connection closed")
	}
}

// collectMetrics periodically collects database metrics
func (d *DB) collectMetrics(ctx context.Context) {
	if d.pool == nil {
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stat := d.pool.Stat()
			metrics.SetDBConnectionsActive(float64(stat.AcquiredConns()))
		}
	}
}

// Exec executes a statement
func (d *DB) Exec(ctx context.Context, sql string, args ...any) error {
	if d.pool == nil {
		return nil
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		logger.Debug("Database exec",
			"sql", sql,
			"duration_ms", duration.Milliseconds(),
		)
	}()

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := d.pool.Exec(ctx, sql, args...)

	status := "success"
	if err != nil {
		status = "error"
		logger.Error("Database exec failed", "error", err, "sql", sql)
	}
	metrics.RecordDBQuery("exec", status)

	return err
}

// Query executes a query and returns rows
func (d *DB) Query(ctx context.Context, sql string, args ...any) (interface{}, error) {
	if d.pool == nil {
		return nil, errors.New("db not configured")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		logger.Debug("Database query",
			"sql", sql,
			"duration_ms", duration.Milliseconds(),
		)
	}()

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rows, err := d.pool.Query(ctx, sql, args...)

	status := "success"
	if err != nil {
		status = "error"
		logger.Error("Database query failed", "error", err, "sql", sql)
	}
	metrics.RecordDBQuery("query", status)

	return rows, err
}

// QueryRow executes a query that returns a single row
func (d *DB) QueryRow(ctx context.Context, sql string, args ...any) interface{} {
	if d.pool == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	return d.pool.QueryRow(ctx, sql, args...)
}

// Health checks database connectivity
func (d *DB) Health(ctx context.Context) error {
	if d.pool == nil {
		return errors.New("database not configured")
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return d.pool.Ping(ctx)
}

// IsConfigured returns true if database is configured
func (d *DB) IsConfigured() bool {
	return d.pool != nil
}
