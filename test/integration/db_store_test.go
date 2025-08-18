//go:build integration

package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/rajasatyajit/SupplyChain/config"
	"github.com/rajasatyajit/SupplyChain/internal/database"
	"github.com/rajasatyajit/SupplyChain/internal/models"
	"github.com/rajasatyajit/SupplyChain/internal/store"
)

// applyMigrations reads scripts/init.sql and executes it against the provided pool
func applyMigrations(ctx context.Context, pool *pgxpool.Pool, t *testing.T) {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil { t.Fatalf("getwd: %v", err) }
	// tests run from the package dir; locate repo root by walking up to find go.mod
	root := cwd
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		root = filepath.Dir(root)
	}
	path := filepath.Join(root, "scripts", "init.sql")
	b, err := os.ReadFile(path)
	if err != nil { t.Fatalf("read init.sql: %v", err) }
	// Execute as a single batch
	_, err = pool.Exec(ctx, string(b))
	if err != nil { t.Fatalf("apply migrations: %v", err) }
}

func TestPostgresStore_WithContainer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		Env: map[string]string{
			"POSTGRES_DB":       "supplychain",
			"POSTGRES_USER":     "supplychain",
			"POSTGRES_PASSWORD": "password",
		},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60 * time.Second),
	}
	pg, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil { t.Fatalf("start container: %v", err) }
	t.Cleanup(func() { _ = pg.Terminate(context.Background()) })

	host, err := pg.Host(ctx)
	if err != nil { t.Fatalf("host: %v", err) }
	port, err := pg.MappedPort(ctx, "5432")
	if err != nil { t.Fatalf("mapped port: %v", err) }

	dsn := "postgres://supplychain:password@" + host + ":" + port.Port() + "/supplychain?sslmode=disable"

	cfg := config.DatabaseConfig{
		URL:             dsn,
		MaxConns:        5,
		MinConns:        1,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
	}

	db, err := database.New(ctx, cfg)
	if err != nil { t.Fatalf("database.New: %v", err) }
	defer db.Close(ctx)

	// Apply migrations
	pool := dbpoolFromDB(db)
	applyMigrations(ctx, pool, t)

	st := store.New(db)

	// Upsert and query
	alerts := []models.Alert{{
		ID:         "int-alert-1",
		Source:     "integration",
		Title:      "Integration Test Alert",
		Summary:    "Inserted via integration test",
		URL:        "https://example.com/alert/1",
		DetectedAt: time.Now().UTC(),
		Severity:   "medium",
	}}
	if err := st.UpsertAlerts(ctx, alerts); err != nil {
		t.Fatalf("UpsertAlerts: %v", err)
	}

	res, err := st.QueryAlerts(ctx, models.AlertQuery{Sources: []string{"integration"}, Limit: 10})
	if err != nil { t.Fatalf("QueryAlerts: %v", err) }
	if len(res) == 0 { t.Fatalf("expected at least 1 alert, got 0") }

	one, err := st.GetAlert(ctx, "int-alert-1")
	if err != nil { t.Fatalf("GetAlert: %v", err) }
	if one == nil || one.ID != "int-alert-1" { t.Fatalf("unexpected alert: %+v", one) }
}

// dbpoolFromDB is a small helper to access the underlying pool for migrations in tests only
func dbpoolFromDB(d *database.DB) *pgxpool.Pool {
	return dpoolAccessor(d)
}
