package store

import (
	"context"

	"github.com/rajasatyajit/SupplyChain/internal/models"
)

// Store defines the interface for alert storage
type Store interface {
	UpsertAlerts(ctx context.Context, alerts []models.Alert) error
	QueryAlerts(ctx context.Context, q models.AlertQuery) ([]models.Alert, error)
	GetAlert(ctx context.Context, id string) (*models.Alert, error)
	Health(ctx context.Context) error
}

// Database interface for dependency injection
type Database interface {
	Exec(ctx context.Context, sql string, args ...any) error
	Query(ctx context.Context, sql string, args ...any) (interface{}, error)
	QueryRow(ctx context.Context, sql string, args ...any) interface{}
	Health(ctx context.Context) error
	IsConfigured() bool
}

// New creates a new store instance
func New(db Database) Store {
	if db.IsConfigured() {
		return NewPostgresStore(db)
	}
	// Fallback to in-memory store if no database
	return NewInMemoryStore()
}
