package store

import (
	"context"
	"testing"
)

type cfgDB struct{ configured bool }

func (d *cfgDB) Exec(ctx context.Context, sql string, args ...any) error { return nil }
func (d *cfgDB) Query(ctx context.Context, sql string, args ...any) (interface{}, error) {
	return nil, nil
}
func (d *cfgDB) QueryRow(ctx context.Context, sql string, args ...any) interface{} { return nil }
func (d *cfgDB) Health(ctx context.Context) error                                  { return nil }
func (d *cfgDB) IsConfigured() bool                                                { return d.configured }

func TestNew_ReturnsPostgresWhenConfigured(t *testing.T) {
	db := &cfgDB{configured: true}
	s := New(db)
	if _, ok := s.(*PostgresStore); !ok {
		t.Fatalf("expected PostgresStore when db is configured, got %T", s)
	}
}

func TestNew_ReturnsInMemoryWhenNotConfigured(t *testing.T) {
	db := &cfgDB{configured: false}
	s := New(db)
	if _, ok := s.(*InMemoryStore); !ok {
		t.Fatalf("expected InMemoryStore when db is not configured, got %T", s)
	}
}
