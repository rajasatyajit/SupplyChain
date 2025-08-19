package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/rajasatyajit/SupplyChain/internal/models"
)

// PostgresStore implements Store using PostgreSQL
type PostgresStore struct {
	db Database
}

// NewPostgresStore creates a new PostgreSQL store
func NewPostgresStore(db Database) *PostgresStore {
	return &PostgresStore{db: db}
}

// UpsertAlerts inserts or updates alerts in the database
func (s *PostgresStore) UpsertAlerts(ctx context.Context, alerts []models.Alert) error {
	if len(alerts) == 0 {
		return nil
	}

	// Use UPSERT (INSERT ... ON CONFLICT DO UPDATE)
	query := `
		INSERT INTO alerts (
			id, source, title, summary, url, detected_at, published_at,
			region, country, location, latitude, longitude, disruption,
			severity, sentiment, confidence, raw
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			summary = EXCLUDED.summary,
			url = EXCLUDED.url,
			detected_at = EXCLUDED.detected_at,
			published_at = EXCLUDED.published_at,
			region = EXCLUDED.region,
			country = EXCLUDED.country,
			location = EXCLUDED.location,
			latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			disruption = EXCLUDED.disruption,
			severity = EXCLUDED.severity,
			sentiment = EXCLUDED.sentiment,
			confidence = EXCLUDED.confidence,
			raw = EXCLUDED.raw,
			updated_at = NOW()
	`

	for _, alert := range alerts {
		err := s.db.Exec(ctx, query,
			alert.ID, alert.Source, alert.Title, alert.Summary, alert.URL,
			alert.DetectedAt, alert.PublishedAt, alert.Region, alert.Country,
			alert.Location, alert.Latitude, alert.Longitude, alert.Disruption,
			alert.Severity, alert.Sentiment, alert.Confidence, alert.Raw,
		)
		if err != nil {
			return fmt.Errorf("upsert alert %s: %w", alert.ID, err)
		}
	}

	return nil
}

// QueryAlerts retrieves alerts based on query parameters
func (s *PostgresStore) QueryAlerts(ctx context.Context, q models.AlertQuery) ([]models.Alert, error) {
	query := `
		SELECT id, source, title, summary, url, detected_at, published_at,
			   region, country, location, latitude, longitude, disruption,
			   severity, sentiment, confidence, raw, created_at, updated_at
		FROM alerts
		WHERE 1=1
	`

	var args []interface{}
	argIndex := 1

	// Build WHERE conditions
	if len(q.IDs) > 0 {
		query += fmt.Sprintf(" AND id = ANY($%d)", argIndex)
		args = append(args, q.IDs)
		argIndex++
	}

	if len(q.Sources) > 0 {
		query += fmt.Sprintf(" AND source = ANY($%d)", argIndex)
		args = append(args, q.Sources)
		argIndex++
	}

	if len(q.Severities) > 0 {
		query += fmt.Sprintf(" AND severity = ANY($%d)", argIndex)
		args = append(args, q.Severities)
		argIndex++
	}

	if len(q.Disruptions) > 0 {
		query += fmt.Sprintf(" AND disruption = ANY($%d)", argIndex)
		args = append(args, q.Disruptions)
		argIndex++
	}

	if len(q.Regions) > 0 {
		query += fmt.Sprintf(" AND region = ANY($%d)", argIndex)
		args = append(args, q.Regions)
		argIndex++
	}

	if len(q.Countries) > 0 {
		query += fmt.Sprintf(" AND country = ANY($%d)", argIndex)
		args = append(args, q.Countries)
		argIndex++
	}

	if !q.Since.IsZero() {
		query += fmt.Sprintf(" AND detected_at >= $%d", argIndex)
		args = append(args, q.Since)
		argIndex++
	}

	if !q.Until.IsZero() {
		query += fmt.Sprintf(" AND detected_at <= $%d", argIndex)
		args = append(args, q.Until)
		argIndex++
	}

	// Add ordering
	query += " ORDER BY detected_at DESC"

	// Add limit and offset
	if q.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, q.Limit)
		argIndex++
	}

	if q.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, q.Offset)
	}

	rowsInterface, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query alerts: %w", err)
	}

	rows, ok := rowsInterface.(pgx.Rows)
	if !ok {
		return nil, fmt.Errorf("invalid rows type")
	}
	defer rows.Close()

	var alerts []models.Alert
	for rows.Next() {
		var alert models.Alert
		err := rows.Scan(
			&alert.ID, &alert.Source, &alert.Title, &alert.Summary, &alert.URL,
			&alert.DetectedAt, &alert.PublishedAt, &alert.Region, &alert.Country,
			&alert.Location, &alert.Latitude, &alert.Longitude, &alert.Disruption,
			&alert.Severity, &alert.Sentiment, &alert.Confidence, &alert.Raw,
			&alert.CreatedAt, &alert.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan alert: %w", err)
		}
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// GetAlert retrieves a single alert by ID
func (s *PostgresStore) GetAlert(ctx context.Context, id string) (*models.Alert, error) {
	query := `
		SELECT id, source, title, summary, url, detected_at, published_at,
			   region, country, location, latitude, longitude, disruption,
			   severity, sentiment, confidence, raw, created_at, updated_at
		FROM alerts
		WHERE id = $1
	`

	rowInterface := s.db.QueryRow(ctx, query, id)
	row, ok := rowInterface.(pgx.Row)
	if !ok {
		return nil, fmt.Errorf("invalid row type")
	}

	var alert models.Alert
	err := row.Scan(
		&alert.ID, &alert.Source, &alert.Title, &alert.Summary, &alert.URL,
		&alert.DetectedAt, &alert.PublishedAt, &alert.Region, &alert.Country,
		&alert.Location, &alert.Latitude, &alert.Longitude, &alert.Disruption,
		&alert.Severity, &alert.Sentiment, &alert.Confidence, &alert.Raw,
		&alert.CreatedAt, &alert.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan alert: %w", err)
	}

	return &alert, nil
}

// Health checks the database connection
func (s *PostgresStore) Health(ctx context.Context) error {
	return s.db.Health(ctx)
}
