package store

import (
	"context"
	"testing"
	"time"

	"github.com/rajasatyajit/SupplyChain/internal/models"
)

func TestInMemoryStore_UpsertAlerts(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	alerts := []models.Alert{
		{
			ID:         "alert-1",
			Source:     "test-source",
			Title:      "Test Alert 1",
			DetectedAt: time.Now().UTC(),
			Severity:   "high",
		},
		{
			ID:         "alert-2",
			Source:     "test-source",
			Title:      "Test Alert 2",
			DetectedAt: time.Now().UTC(),
			Severity:   "medium",
		},
	}

	err := store.UpsertAlerts(ctx, alerts)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify alerts were stored
	if len(store.alerts) != 2 {
		t.Errorf("Expected 2 alerts, got %d", len(store.alerts))
	}

	// Test upsert (update existing)
	alerts[0].Title = "Updated Alert 1"
	err = store.UpsertAlerts(ctx, alerts[:1])
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Should still have 2 alerts
	if len(store.alerts) != 2 {
		t.Errorf("Expected 2 alerts after upsert, got %d", len(store.alerts))
	}

	// Verify update
	if store.alerts["alert-1"].Title != "Updated Alert 1" {
		t.Errorf("Expected updated title, got %s", store.alerts["alert-1"].Title)
	}
}

func TestInMemoryStore_QueryAlerts(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	// Setup test data
	alerts := []models.Alert{
		{
			ID:         "alert-1",
			Source:     "source-1",
			Title:      "High Severity Alert",
			DetectedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			Severity:   "high",
			Region:     "North America",
		},
		{
			ID:         "alert-2",
			Source:     "source-2",
			Title:      "Medium Severity Alert",
			DetectedAt: time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
			Severity:   "medium",
			Region:     "Europe",
		},
		{
			ID:         "alert-3",
			Source:     "source-1",
			Title:      "Low Severity Alert",
			DetectedAt: time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			Severity:   "low",
			Region:     "Asia",
		},
	}

	err := store.UpsertAlerts(ctx, alerts)
	if err != nil {
		t.Fatalf("Failed to setup test data: %v", err)
	}

	tests := []struct {
		name          string
		query         models.AlertQuery
		expectedCount int
		expectedFirst string
	}{
		{
			name:          "No filter - all alerts",
			query:         models.AlertQuery{},
			expectedCount: 3,
			expectedFirst: "alert-3", // Most recent first
		},
		{
			name: "Filter by severity",
			query: models.AlertQuery{
				Severities: []string{"high"},
			},
			expectedCount: 1,
			expectedFirst: "alert-1",
		},
		{
			name: "Filter by source",
			query: models.AlertQuery{
				Sources: []string{"source-1"},
			},
			expectedCount: 2,
			expectedFirst: "alert-3", // Most recent first
		},
		{
			name: "Filter by time range",
			query: models.AlertQuery{
				Since: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				Until: time.Date(2024, 1, 15, 11, 30, 0, 0, time.UTC),
			},
			expectedCount: 1,
			expectedFirst: "alert-2",
		},
		{
			name: "Limit results",
			query: models.AlertQuery{
				Limit: 2,
			},
			expectedCount: 2,
			expectedFirst: "alert-3",
		},
		{
			name: "Offset results",
			query: models.AlertQuery{
				Offset: 1,
			},
			expectedCount: 2,
			expectedFirst: "alert-2",
		},
		{
			name: "No matches",
			query: models.AlertQuery{
				Severities: []string{"critical"},
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := store.QueryAlerts(ctx, tt.query)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(results))
			}

			if tt.expectedCount > 0 && results[0].ID != tt.expectedFirst {
				t.Errorf("Expected first result ID %s, got %s", tt.expectedFirst, results[0].ID)
			}
		})
	}
}

func TestInMemoryStore_GetAlert(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	alert := models.Alert{
		ID:     "test-alert",
		Source: "test-source",
		Title:  "Test Alert",
	}

	err := store.UpsertAlerts(ctx, []models.Alert{alert})
	if err != nil {
		t.Fatalf("Failed to setup test data: %v", err)
	}

	t.Run("Existing alert", func(t *testing.T) {
		result, err := store.GetAlert(ctx, "test-alert")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result == nil {
			t.Error("Expected alert, got nil")
		} else if result.ID != "test-alert" {
			t.Errorf("Expected ID test-alert, got %s", result.ID)
		}
	})

	t.Run("Non-existent alert", func(t *testing.T) {
		result, err := store.GetAlert(ctx, "non-existent")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result != nil {
			t.Error("Expected nil, got alert")
		}
	})
}

func TestInMemoryStore_Health(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	err := store.Health(ctx)
	if err != nil {
		t.Errorf("Expected no error for in-memory store health, got %v", err)
	}
}