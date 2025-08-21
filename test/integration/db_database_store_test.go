//go:build integration

package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/rajasatyajit/SupplyChain/config"
	"github.com/rajasatyajit/SupplyChain/internal/database"
	"github.com/rajasatyajit/SupplyChain/internal/models"
	"github.com/rajasatyajit/SupplyChain/internal/store"
)

func pgMigrationsPath(t *testing.T) string {
	t.Helper()
	root := "."
	cwd, _ := os.Getwd()
	root = cwd
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		root = filepath.Dir(root)
	}
	return filepath.Join(root, "scripts", "init.sql")
}

func TestDatabaseAndPostgresStore_Integration(t *testing.T) {
	if !containersAvailable() {
		t.Skip("container runtime not available; skipping container-based integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		Env:          map[string]string{"POSTGRES_DB": "supplychain", "POSTGRES_USER": "supplychain", "POSTGRES_PASSWORD": "password"},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60 * time.Second),
	}
	pg, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true})
	if err != nil {
		t.Fatalf("start container: %v", err)
	}
	t.Cleanup(func() { _ = pg.Terminate(context.Background()) })

	host, _ := pg.Host(ctx)
	port, _ := pg.MappedPort(ctx, "5432")
	dsn := "postgres://supplychain:password@" + host + ":" + port.Port() + "/supplychain?sslmode=disable"

	cfg := config.DatabaseConfig{URL: dsn, MaxConns: 5, MinConns: 1, MaxConnLifetime: time.Hour, MaxConnIdleTime: 30 * time.Minute}
	db, err := database.New(ctx, cfg)
	if err != nil {
		t.Fatalf("database.New: %v", err)
	}
	defer db.Close(ctx)

	// Health should pass
	if err := db.Health(ctx); err != nil {
		t.Fatalf("db health: %v", err)
	}

	// Apply migrations
	pool := dbpoolFromDB(db)
	b, err := os.ReadFile(pgMigrationsPath(t))
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	if _, err := pool.Exec(ctx, string(b)); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	// Exec
	if err := db.Exec(ctx, "SELECT 1"); err != nil {
		t.Fatalf("exec: %v", err)
	}
	// Query
	if _, err := db.Query(ctx, "SELECT 1"); err != nil {
		t.Fatalf("query: %v", err)
	}
	// QueryRow
	if r := db.QueryRow(ctx, "SELECT 1"); r == nil {
		t.Fatalf("expected non-nil row")
	}

	// Store
	st := store.New(db)
	alerts := []models.Alert{{
		ID:         "int-1",
		Source:     "itest",
		Title:      "Port Strike Disrupts",
		Summary:    "",
		URL:        "http://x/1",
		DetectedAt: time.Now().UTC(),
		Severity:   "high",
	}}
	if err := st.UpsertAlerts(ctx, alerts); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, err := st.GetAlert(ctx, "int-1")
	if err != nil || got == nil {
		t.Fatalf("get alert: %v, %+v", err, got)
	}

	list, err := st.QueryAlerts(ctx, models.AlertQuery{Sources: []string{"itest"}, Limit: 10})
	if err != nil {
		t.Fatalf("query alerts: %v", err)
	}
	if len(list) == 0 {
		t.Fatalf("expected at least one alert")
	}
}
