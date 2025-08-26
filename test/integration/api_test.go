package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	chi "github.com/go-chi/chi/v5"
	"github.com/rajasatyajit/SupplyChain/internal/api"
	"github.com/rajasatyajit/SupplyChain/internal/models"
	"github.com/rajasatyajit/SupplyChain/internal/store"
)

func TestHealthEndpoints(t *testing.T) {
	// Setup
	store := store.NewInMemoryStore()
	handler := api.NewHandler(store, nil, "", "test", "test-time", "test-commit")

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	tests := []struct {
		name           string
		endpoint       string
		expectedStatus int
	}{
		{"Health Check", "/health", http.StatusOK},
		{"Readiness Check", "/v1/health/ready", http.StatusOK},
		{"Liveness Check", "/v1/health/live", http.StatusOK},
		{"Version Info", "/v1/version", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.endpoint, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check content type
			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
			}
		})
	}
}

func TestAlertsEndpoint(t *testing.T) {
	// Setup
	store := store.NewInMemoryStore()
	handler := api.NewHandler(store, nil, "", "test", "test-time", "test-commit")

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	// Add test data
	testAlert := models.Alert{
		ID:          "test-alert-1",
		Source:      "test-source",
		Title:       "Test Alert",
		Summary:     "This is a test alert",
		URL:         "https://example.com/alert/1",
		DetectedAt:  time.Now().UTC(),
		PublishedAt: time.Now().UTC(),
		Severity:    "medium",
		Sentiment:   "neutral",
		Confidence:  0.8,
	}

	ctx := context.Background()
	err := store.UpsertAlerts(ctx, []models.Alert{testAlert})
	if err != nil {
		t.Fatalf("Failed to insert test alert: %v", err)
	}

	t.Run("Get Alerts", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/alerts", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		data, ok := response["data"].([]interface{})
		if !ok {
			t.Errorf("Expected data to be an array")
		}

		if len(data) != 1 {
			t.Errorf("Expected 1 alert, got %d", len(data))
		}
	})

	t.Run("Get Single Alert", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/alerts/test-alert-1", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var alert models.Alert
		err := json.NewDecoder(w.Body).Decode(&alert)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if alert.ID != "test-alert-1" {
			t.Errorf("Expected alert ID test-alert-1, got %s", alert.ID)
		}
	})

	t.Run("Get Non-existent Alert", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/alerts/non-existent", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})
}
