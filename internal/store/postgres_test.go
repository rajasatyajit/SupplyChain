package store

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/rajasatyajit/SupplyChain/internal/models"
)

type mockDB struct{
	ExecFn    func(ctx context.Context, sql string, args ...any) error
	QueryFn   func(ctx context.Context, sql string, args ...any) (interface{}, error)
	QueryRowFn func(ctx context.Context, sql string, args ...any) interface{}
	HealthFn  func(ctx context.Context) error
	IsConfiguredFn func() bool
}

func (m *mockDB) Exec(ctx context.Context, sql string, args ...any) error {
	if m.ExecFn != nil { return m.ExecFn(ctx, sql, args...) }
	return nil
}
func (m *mockDB) Query(ctx context.Context, sql string, args ...any) (interface{}, error) {
	if m.QueryFn != nil { return m.QueryFn(ctx, sql, args...) }
	return nil, nil
}
func (m *mockDB) QueryRow(ctx context.Context, sql string, args ...any) interface{} {
	if m.QueryRowFn != nil { return m.QueryRowFn(ctx, sql, args...) }
	return nil
}
func (m *mockDB) Health(ctx context.Context) error { if m.HealthFn!=nil { return m.HealthFn(ctx) }; return nil }
func (m *mockDB) IsConfigured() bool { if m.IsConfiguredFn!=nil { return m.IsConfiguredFn() }; return true }

func TestPostgresStore_UpsertAlerts_Empty(t *testing.T) {
	s := NewPostgresStore(&mockDB{})
	err := s.UpsertAlerts(context.Background(), []models.Alert{})
	if err != nil { t.Fatalf("expected nil, got %v", err) }
}

func TestPostgresStore_UpsertAlerts_BuildsQueryAndPropagatesError(t *testing.T) {
	called := 0
	var gotSQL string
	db := &mockDB{ExecFn: func(ctx context.Context, sql string, args ...any) error {
		called++
		gotSQL = sql
		if called == 1 {
			return errors.New("exec failure")
		}
		return nil
	}}
	s := NewPostgresStore(db)
	alerts := []models.Alert{{ID:"id1", Source:"s", Title:"t"}}
	err := s.UpsertAlerts(context.Background(), alerts)
	if err == nil { t.Fatalf("expected error, got nil") }
	if !strings.Contains(gotSQL, "INSERT INTO alerts") || !strings.Contains(gotSQL, "ON CONFLICT") {
		t.Errorf("unexpected SQL: %s", gotSQL)
	}
}

func TestPostgresStore_QueryAlerts_ErrorFromDB(t *testing.T) {
	db := &mockDB{QueryFn: func(ctx context.Context, sql string, args ...any) (interface{}, error) { return nil, errors.New("db error") }}
	s := NewPostgresStore(db)
	_, err := s.QueryAlerts(context.Background(), models.AlertQuery{})
	if err == nil { t.Fatalf("expected error, got nil") }
	if !strings.Contains(err.Error(), "query alerts") { t.Errorf("wrap missing: %v", err) }
}

func TestPostgresStore_QueryAlerts_InvalidRowsType(t *testing.T) {
	db := &mockDB{QueryFn: func(ctx context.Context, sql string, args ...any) (interface{}, error) { return 123, nil }}
	s := NewPostgresStore(db)
	_, err := s.QueryAlerts(context.Background(), models.AlertQuery{})
	if err == nil { t.Fatalf("expected error, got nil") }
	if !strings.Contains(err.Error(), "invalid rows type") { t.Errorf("got %v", err) }
}

type fakeRow struct{ err error }
func (r fakeRow) Scan(dest ...any) error { return r.err }

func TestPostgresStore_GetAlert_InvalidRowType(t *testing.T) {
	db := &mockDB{QueryRowFn: func(ctx context.Context, sql string, args ...any) interface{} { return 123 }}
	s := NewPostgresStore(db)
	_, err := s.GetAlert(context.Background(), "x")
	if err == nil { t.Fatalf("expected error, got nil") }
	if !strings.Contains(err.Error(), "invalid row type") { t.Errorf("got %v", err) }
}

func TestPostgresStore_GetAlert_NoRows(t *testing.T) {
	db := &mockDB{QueryRowFn: func(ctx context.Context, sql string, args ...any) interface{} { return fakeRow{err: pgx.ErrNoRows} }}
	s := NewPostgresStore(db)
	res, err := s.GetAlert(context.Background(), "missing")
	if err != nil { t.Fatalf("unexpected err: %v", err) }
	if res != nil { t.Fatalf("expected nil, got %+v", res) }
}
