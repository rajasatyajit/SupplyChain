package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rajasatyajit/SupplyChain/internal/models"
)

// MockStore implements the store interface for testing
type MockStore struct {
	alerts map[string]models.Alert
	health error
}

func NewMockStore() *MockStore {
	return &MockStore{
		alerts: make(map[string]models.Alert),
		health: nil,
	}
}

func (m *MockStore) UpsertAlerts(ctx context.Context, alerts []models.Alert) error {
	for _, alert := range alerts {
		m.alerts[alert.ID] = alert
	}
	return nil
}

func (m *MockStore) QueryAlerts(ctx context.Context, q models.AlertQuery) ([]models.Alert, error) {
	var results []models.Alert
	for _, alert := range m.alerts {
		if q.Matches(alert) {
			results = append(results, alert)
		}
	}
	
	// Apply limit
	if q.Limit > 0 && len(results) > q.Limit {
		results = results[:q.Limit]
	}
	
	return results, nil
}

func (m *MockStore) GetAlert(ctx context.Context, id string) (*models.Alert, error) {
	if alert, exists := m.alerts[id]; exists {
		return &alert, nil
	}
	return nil, nil
}

func (m *MockStore) Health(ctx context.Context) error {
	return m.health
}

func (m *MockStore) SetHealthError(err error) {
	m.health = err
}

func TestHandler_HealthEndpoints(t *testing.T) {
	store := NewMockStore()
	handler := NewHandler(store, "test-version", "test-build-time", "test-commit")

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	tests := []struct {
		name           string
		endpoint       string
		expectedStatus int
		checkBody      bool
	}{
		{
			name:           "Basic health check",
			endpoint:       "/health",
			expectedStatus: http.StatusOK,
			checkBody:      true,
		},
		{
			name:           "V1 health check",
			endpoint:       "/v1/health",
			expectedStatus: http.StatusOK,
			checkBody:      true,
		},
		{
			name:           "Readiness check - healthy",
			endpoint:       "/v1/health/ready",
			expectedStatus: http.StatusOK,
			checkBody:      true,
		},
		{
			name:           "Liveness check",
			endpoint:       "/v1/health/live",
			expectedStatus: http.StatusOK,
			checkBody:      true,
		},
		{
			name:           "Version endpoint",
			endpoint:       "/v1/version",
			expectedStatus: http.StatusOK,
			checkBody:      false, // Version endpoint doesn't have timestamp
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.endpoint, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkBody {
				contentType := w.Header().Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", contentType)
				}

				var response map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Errorf("Failed to decode JSON response: %v", err)
				}

				if _, exists := response["timestamp"]; !exists {
					t.Error("Expected timestamp in response")
				}
			}
		})
	}
}

func TestHandler_ReadinessCheck_Unhealthy(t *testing.T) {
	store := NewMockStore()
	store.SetHealthError(errors.New("database connection failed"))
	
	handler := NewHandler(store, "test-version", "test-build-time", "test-commit")

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/v1/health/ready", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestHandler_GetAlerts(t *testing.T) {
	store := NewMockStore()
	
	// Setup test data
	testAlerts := []models.Alert{
		{
			ID:         "alert-1",
			Source:     "test-source",
			Title:      "Test Alert 1",
			Summary:    "Test summary 1",
			DetectedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			Severity:   "high",
		},
		{
			ID:         "alert-2",
			Source:     "test-source",
			Title:      "Test Alert 2",
			Summary:    "Test summary 2",
			DetectedAt: time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
			Severity:   "medium",
		},
	}

	err := store.UpsertAlerts(context.Background(), testAlerts)
	if err != nil {
		t.Fatalf("Failed to setup test data: %v", err)
	}

	handler := NewHandler(store, "test-version", "test-build-time", "test-commit")
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "Get all alerts",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "Filter by severity",
			queryParams:    "?severity=high",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "Filter by source",
			queryParams:    "?source=test-source",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "Limit results",
			queryParams:    "?limit=1",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "Invalid limit",
			queryParams:    "?limit=invalid",
			expectedStatus: http.StatusBadRequest,
			expectedCount:  0,
		},
		{
			name:           "Limit too high",
			queryParams:    "?limit=2000",
			expectedStatus: http.StatusBadRequest,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/v1/alerts"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Errorf("Failed to decode JSON response: %v", err)
				}

				data, ok := response["data"].([]interface{})
				if !ok {
					t.Error("Expected data to be an array")
				}

				if len(data) != tt.expectedCount {
					t.Errorf("Expected %d alerts, got %d", tt.expectedCount, len(data))
				}

				// Check cache header
				cacheControl := w.Header().Get("Cache-Control")
				if cacheControl != "public, max-age=60" {
					t.Errorf("Expected Cache-Control header, got %s", cacheControl)
				}
			}
		})
	}
}

func TestHandler_GetAlert(t *testing.T) {
	store := NewMockStore()
	
	testAlert := models.Alert{
		ID:      "test-alert-1",
		Source:  "test-source",
		Title:   "Test Alert",
		Summary: "Test summary",
	}

	err := store.UpsertAlerts(context.Background(), []models.Alert{testAlert})
	if err != nil {
		t.Fatalf("Failed to setup test data: %v", err)
	}

	handler := NewHandler(store, "test-version", "test-build-time", "test-commit")
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	tests := []struct {
		name           string
		alertID        string
		expectedStatus int
	}{
		{
			name:           "Get existing alert",
			alertID:        "test-alert-1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Get non-existent alert",
			alertID:        "non-existent",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Empty alert ID",
			alertID:        "",
			expectedStatus: http.StatusNotFound, // Chi router behavior
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := "/v1/alerts/" + tt.alertID
			req := httptest.NewRequest("GET", endpoint, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var alert models.Alert
				err := json.NewDecoder(w.Body).Decode(&alert)
				if err != nil {
					t.Errorf("Failed to decode JSON response: %v", err)
				}

				if alert.ID != tt.alertID {
					t.Errorf("Expected alert ID %s, got %s", tt.alertID, alert.ID)
				}

				// Check cache header
				cacheControl := w.Header().Get("Cache-Control")
				if cacheControl != "public, max-age=300" {
					t.Errorf("Expected Cache-Control header, got %s", cacheControl)
				}
			}
		})
	}
}

func TestHandler_ParseAlertQuery(t *testing.T) {
	handler := NewHandler(NewMockStore(), "test", "test", "test")

	tests := []struct {
		name        string
		queryString string
		expectError bool
		checkFields func(models.AlertQuery) error
	}{
		{
			name:        "Empty query",
			queryString: "",
			expectError: false,
			checkFields: func(q models.AlertQuery) error {
				if q.Limit != 0 {
					return fmt.Errorf("expected limit 0, got %d", q.Limit)
				}
				return nil
			},
		},
		{
			name:        "Valid limit",
			queryString: "limit=50",
			expectError: false,
			checkFields: func(q models.AlertQuery) error {
				if q.Limit != 50 {
					return fmt.Errorf("expected limit 50, got %d", q.Limit)
				}
				return nil
			},
		},
		{
			name:        "Invalid limit",
			queryString: "limit=invalid",
			expectError: true,
		},
		{
			name:        "Limit too high",
			queryString: "limit=2000",
			expectError: true,
		},
		{
			name:        "Valid time filter",
			queryString: "since=2024-01-15T10:00:00Z",
			expectError: false,
			checkFields: func(q models.AlertQuery) error {
				expected := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
				if !q.Since.Equal(expected) {
					return fmt.Errorf("expected since %v, got %v", expected, q.Since)
				}
				return nil
			},
		},
		{
			name:        "Invalid time format",
			queryString: "since=invalid-time",
			expectError: true,
		},
		{
			name:        "Multiple filters",
			queryString: "source=test&severity=high&limit=10",
			expectError: false,
			checkFields: func(q models.AlertQuery) error {
				if len(q.Sources) != 1 || q.Sources[0] != "test" {
					return fmt.Errorf("expected sources [test], got %v", q.Sources)
				}
				if len(q.Severities) != 1 || q.Severities[0] != "high" {
					return fmt.Errorf("expected severities [high], got %v", q.Severities)
				}
				if q.Limit != 10 {
					return fmt.Errorf("expected limit 10, got %d", q.Limit)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/?"+tt.queryString, nil)
			
			query, err := handler.parseAlertQuery(req)
			
			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			
			if !tt.expectError && tt.checkFields != nil {
				if err := tt.checkFields(query); err != nil {
					t.Error(err)
				}
			}
		})
	}
}