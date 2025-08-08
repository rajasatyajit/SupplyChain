package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct { pool *pgxpool.Pool }

func NewDB(ctx context.Context) (*DB, error) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		log.Printf("DATABASE_URL not set; using in-memory store only")
		return &DB{pool: nil}, nil
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil { return nil, fmt.Errorf("parse db url: %w", err) }
	cfg.MaxConns = 10
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil { return nil, fmt.Errorf("connect db: %w", err) }
	return &DB{pool: pool}, nil
}

func (d *DB) Close(ctx context.Context) {
	if d.pool != nil { d.pool.Close() }
}

// exec runs a statement if DB exists
func (d *DB) exec(ctx context.Context, sql string, args ...any) error {
	if d.pool == nil { return nil }
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := d.pool.Exec(ctx, sql, args...)
	return err
}

// query runs a query if DB exists; returns error if DB not configured and query is required
func (d *DB) query(ctx context.Context, sql string, args ...any) (pgxRows, error) {
	if d.pool == nil { return nil, errors.New("db not configured") }
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := d.pool.Query(ctx, sql, args...)
	return rows, err
}

type pgxRows interface { Close(); Next() bool; Scan(...any) error }

