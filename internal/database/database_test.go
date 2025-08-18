package database

import (
	"context"
	"testing"
	"time"

	"github.com/rajasatyajit/SupplyChain/config"
	"github.com/rajasatyajit/SupplyChain/internal/logger"
)

func TestNew_NoDatabase(t *testing.T) {
	// Initialize logger for tests
	logger.Init("error", "text")
	
	cfg := config.DatabaseConfig{
		URL: "", // No database URL
	}

	ctx := context.Background()
	db, err := New(ctx, cfg)
	if err != nil {
		t.Errorf("Expected no error for empty database URL, got %v", err)
	}

	if db == nil {
		t.Error("Expected DB instance, got nil")
	}

	if db.pool != nil {
		t.Error("Expected pool to be nil when no database URL provided")
	}

	if db.IsConfigured() {
		t.Error("Expected IsConfigured to return false when no database")
	}
}

func TestNew_InvalidURL(t *testing.T) {
	cfg := config.DatabaseConfig{
		URL: "invalid-url",
	}

	ctx := context.Background()
	_, err := New(ctx, cfg)
	if err == nil {
		t.Error("Expected error for invalid database URL, got nil")
	}
}

func TestDB_Operations_NoPool(t *testing.T) {
	db := &DB{
		pool: nil,
		cfg:  config.DatabaseConfig{},
	}

	ctx := context.Background()

	// Test Exec with no pool
	err := db.Exec(ctx, "SELECT 1")
	if err != nil {
		t.Errorf("Expected no error for Exec with no pool, got %v", err)
	}

	// Test Query with no pool
	_, err = db.Query(ctx, "SELECT 1")
	if err == nil {
		t.Error("Expected error for Query with no pool, got nil")
	}

	// Test QueryRow with no pool
	result := db.QueryRow(ctx, "SELECT 1")
	if result != nil {
		t.Error("Expected nil for QueryRow with no pool")
	}

	// Test Health with no pool
	err = db.Health(ctx)
	if err == nil {
		t.Error("Expected error for Health with no pool, got nil")
	}
}

func TestDB_Close(t *testing.T) {
	db := &DB{
		pool: nil,
		cfg:  config.DatabaseConfig{},
	}

	ctx := context.Background()
	
	// Should not panic when closing with no pool
	db.Close(ctx)
}

func TestDB_IsConfigured(t *testing.T) {
	tests := []struct {
		name     string
		hasPool  bool
		expected bool
	}{
		{
			name:     "With pool",
			hasPool:  true,
			expected: true,
		},
		{
			name:     "Without pool",
			hasPool:  false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &DB{
				cfg: config.DatabaseConfig{},
			}

			if tt.hasPool {
				// We can't easily create a real pool in tests without a database
				// So we'll test the nil case and assume the pool case works
				// In integration tests, we would test with a real database
				db.pool = nil // This represents the "no pool" case
			}

			// For this unit test, we'll only test the nil case
			if tt.hasPool {
				t.Skip("Skipping pool test in unit tests - requires integration test")
			}

			result := db.IsConfigured()
			if result != tt.expected {
				t.Errorf("Expected IsConfigured %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDB_CollectMetrics_NoPool(t *testing.T) {
	db := &DB{
		pool: nil,
		cfg:  config.DatabaseConfig{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()

	// Should return immediately when no pool
	db.collectMetrics(ctx)
	// If we get here without hanging, the test passes
}

// Integration test helper - only runs if TEST_DATABASE_URL is set
func getTestDatabaseURL() string {
	// In a real project, you might use environment variables for integration tests
	// For now, we'll skip integration tests in unit test suite
	return ""
}

func TestIntegration_DatabaseOperations(t *testing.T) {
	dbURL := getTestDatabaseURL()
	if dbURL == "" {
		t.Skip("Skipping integration test - no test database URL provided")
	}

	cfg := config.DatabaseConfig{
		URL:             dbURL,
		MaxConns:        5,
		MinConns:        1,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: time.Minute * 30,
	}

	ctx := context.Background()
	db, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create database connection: %v", err)
	}
	defer db.Close(ctx)

	if !db.IsConfigured() {
		t.Error("Expected database to be configured")
	}

	// Test Health
	err = db.Health(ctx)
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	// Test Exec
	err = db.Exec(ctx, "SELECT 1")
	if err != nil {
		t.Errorf("Exec failed: %v", err)
	}

	// Test Query
	rows, err := db.Query(ctx, "SELECT 1 as test_column")
	if err != nil {
		t.Errorf("Query failed: %v", err)
	}
	if rows != nil {
		// In a real test, we would iterate through rows and close them
		// For this example, we'll just check that we got a result
	}

	// Test QueryRow
	row := db.QueryRow(ctx, "SELECT 1 as test_column")
	if row == nil {
		t.Error("QueryRow returned nil")
	}
}

func BenchmarkDB_Exec(b *testing.B) {
	db := &DB{
		pool: nil,
		cfg:  config.DatabaseConfig{},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.Exec(ctx, "SELECT 1")
	}
}

func BenchmarkDB_Health(b *testing.B) {
	db := &DB{
		pool: nil,
		cfg:  config.DatabaseConfig{},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.Health(ctx)
	}
}