package pipeline

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rajasatyajit/SupplyChain/config"
	"github.com/rajasatyajit/SupplyChain/internal/logger"
	"github.com/rajasatyajit/SupplyChain/internal/models"
)

// MockStore for testing
type MockStore struct {
	alerts []models.Alert
	err    error
}

func (m *MockStore) UpsertAlerts(ctx context.Context, alerts []models.Alert) error {
	if m.err != nil {
		return m.err
	}
	m.alerts = append(m.alerts, alerts...)
	return nil
}

// MockClassifier for testing
type MockClassifier struct{}

func (m *MockClassifier) Classify(alert *models.Alert) {
	alert.Severity = "medium"
	alert.Sentiment = "neutral"
	alert.Confidence = 0.8
}

// MockGeocoder for testing
type MockGeocoder struct {
	err error
}

func (m *MockGeocoder) Geocode(alert *models.Alert) error {
	if m.err != nil {
		return m.err
	}
	alert.Location = "Test Location"
	alert.Country = "Test Country"
	alert.Region = "Test Region"
	return nil
}

// MockSource for testing
type MockSource struct {
	name     string
	alerts   []models.Alert
	err      error
	interval time.Duration
}

func (m *MockSource) Name() string {
	return m.name
}

func (m *MockSource) Fetch(ctx context.Context) ([]models.Alert, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.alerts, nil
}

func (m *MockSource) Interval() time.Duration {
	if m.interval == 0 {
		return time.Millisecond * 100 // Fast for testing
	}
	return m.interval
}

func TestNew(t *testing.T) {
	// Initialize logger for tests
	logger.Init("error", "text")

	store := &MockStore{}
	classifier := &MockClassifier{}
	geocoder := &MockGeocoder{}
	cfg := config.PipelineConfig{
		RateLimit:     5.0,
		WorkerCount:   2,
		BatchSize:     10,
		RetryAttempts: 3,
		RetryDelay:    time.Millisecond * 100,
	}

	pipeline := New(store, classifier, geocoder, cfg)

	if pipeline == nil {
		t.Error("Expected pipeline instance, got nil")
	}

	if pipeline.store != store {
		t.Error("Store not set correctly")
	}

	if pipeline.classifier != classifier {
		t.Error("Classifier not set correctly")
	}

	if pipeline.geocoder != geocoder {
		t.Error("Geocoder not set correctly")
	}

	if len(pipeline.sources) == 0 {
		t.Error("Expected sources to be initialized")
	}
}

func TestPipeline_ProcessBatch(t *testing.T) {
	store := &MockStore{}
	classifier := &MockClassifier{}
	geocoder := &MockGeocoder{}
	cfg := config.PipelineConfig{
		RateLimit:     5.0,
		WorkerCount:   2,
		BatchSize:     10,
		RetryAttempts: 3,
		RetryDelay:    time.Millisecond * 100,
	}

	pipeline := New(store, classifier, geocoder, cfg)

	alerts := []models.Alert{
		{
			Title:   "Test Alert 1",
			Summary: "Test summary 1",
			URL:     "http://example.com/1",
		},
		{
			Title:   "Test Alert 2",
			Summary: "Test summary 2",
			URL:     "http://example.com/2",
		},
	}

	ctx := context.Background()
	err := pipeline.processBatch(ctx, "test-source", alerts)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check that alerts were processed
	if len(store.alerts) != 2 {
		t.Errorf("Expected 2 alerts in store, got %d", len(store.alerts))
	}

	// Check that alerts were enhanced
	for _, alert := range store.alerts {
		if alert.Source != "test-source" {
			t.Errorf("Expected source 'test-source', got %s", alert.Source)
		}

		if alert.ID == "" {
			t.Error("Expected ID to be generated")
		}

		if alert.DetectedAt.IsZero() {
			t.Error("Expected DetectedAt to be set")
		}

		if alert.Disruption == "" {
			t.Error("Expected Disruption to be inferred")
		}

		if alert.Severity != "medium" {
			t.Errorf("Expected severity 'medium', got %s", alert.Severity)
		}

		if alert.Location != "Test Location" {
			t.Errorf("Expected location 'Test Location', got %s", alert.Location)
		}
	}
}

func TestPipeline_ProcessBatch_GeocodingError(t *testing.T) {
	store := &MockStore{}
	classifier := &MockClassifier{}
	geocoder := &MockGeocoder{err: errors.New("geocoding failed")}
	cfg := config.PipelineConfig{
		RateLimit:     5.0,
		WorkerCount:   2,
		BatchSize:     10,
		RetryAttempts: 3,
		RetryDelay:    time.Millisecond * 100,
	}

	pipeline := New(store, classifier, geocoder, cfg)

	alerts := []models.Alert{
		{
			Title:      "Test Alert",
			Summary:    "Test summary",
			URL:        "http://example.com/1",
			Confidence: 1.0,
		},
	}

	ctx := context.Background()
	err := pipeline.processBatch(ctx, "test-source", alerts)
	if err != nil {
		t.Errorf("Expected no error despite geocoding failure, got %v", err)
	}

	// Check that confidence was reduced due to geocoding error
	if len(store.alerts) != 1 {
		t.Fatalf("Expected 1 alert in store, got %d", len(store.alerts))
	}

	if store.alerts[0].Confidence >= 1.0 {
		t.Errorf("Expected confidence to be reduced, got %f", store.alerts[0].Confidence)
	}
}

func TestPipeline_ProcessBatch_StoreError(t *testing.T) {
	store := &MockStore{err: errors.New("store error")}
	classifier := &MockClassifier{}
	geocoder := &MockGeocoder{}
	cfg := config.PipelineConfig{
		RateLimit:     5.0,
		WorkerCount:   2,
		BatchSize:     10,
		RetryAttempts: 3,
		RetryDelay:    time.Millisecond * 100,
	}

	pipeline := New(store, classifier, geocoder, cfg)

	alerts := []models.Alert{
		{
			Title:   "Test Alert",
			Summary: "Test summary",
			URL:     "http://example.com/1",
		},
	}

	ctx := context.Background()
	err := pipeline.processBatch(ctx, "test-source", alerts)
	if err == nil {
		t.Error("Expected error from store, got nil")
	}
}

func TestPipeline_RunOnce(t *testing.T) {
	store := &MockStore{}
	classifier := &MockClassifier{}
	geocoder := &MockGeocoder{}
	cfg := config.PipelineConfig{
		RateLimit:     100.0, // High rate limit for testing
		WorkerCount:   2,
		BatchSize:     10,
		RetryAttempts: 1,
		RetryDelay:    time.Millisecond * 10,
	}

	pipeline := New(store, classifier, geocoder, cfg)

	// Replace sources with mock
	mockSource := &MockSource{
		name: "test-source",
		alerts: []models.Alert{
			{
				Title:   "Test Alert",
				Summary: "Test summary",
				URL:     "http://example.com/1",
			},
		},
	}
	pipeline.sources = []Source{mockSource}

	ctx := context.Background()
	err := pipeline.runOnce(ctx, mockSource)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(store.alerts) != 1 {
		t.Errorf("Expected 1 alert in store, got %d", len(store.alerts))
	}
}

func TestPipeline_RunOnce_FetchError(t *testing.T) {
	store := &MockStore{}
	classifier := &MockClassifier{}
	geocoder := &MockGeocoder{}
	cfg := config.PipelineConfig{
		RateLimit:     100.0,
		WorkerCount:   2,
		BatchSize:     10,
		RetryAttempts: 1,
		RetryDelay:    time.Millisecond * 10,
	}

	pipeline := New(store, classifier, geocoder, cfg)

	mockSource := &MockSource{
		name: "test-source",
		err:  errors.New("fetch error"),
	}

	ctx := context.Background()
	err := pipeline.runOnce(ctx, mockSource)
	if err == nil {
		t.Error("Expected error from fetch, got nil")
	}
}

func TestPipeline_RunOnce_NoAlerts(t *testing.T) {
	store := &MockStore{}
	classifier := &MockClassifier{}
	geocoder := &MockGeocoder{}
	cfg := config.PipelineConfig{
		RateLimit:     100.0,
		WorkerCount:   2,
		BatchSize:     10,
		RetryAttempts: 1,
		RetryDelay:    time.Millisecond * 10,
	}

	pipeline := New(store, classifier, geocoder, cfg)

	mockSource := &MockSource{
		name:   "test-source",
		alerts: []models.Alert{}, // No alerts
	}

	ctx := context.Background()
	err := pipeline.runOnce(ctx, mockSource)
	if err != nil {
		t.Errorf("Expected no error when no alerts, got %v", err)
	}

	if len(store.alerts) != 0 {
		t.Errorf("Expected 0 alerts in store, got %d", len(store.alerts))
	}
}

func TestPipeline_IsRunning(t *testing.T) {
	store := &MockStore{}
	classifier := &MockClassifier{}
	geocoder := &MockGeocoder{}
	cfg := config.PipelineConfig{
		RateLimit:     5.0,
		WorkerCount:   2,
		BatchSize:     10,
		RetryAttempts: 3,
		RetryDelay:    time.Millisecond * 100,
	}

	pipeline := New(store, classifier, geocoder, cfg)

	// Initially not running
	if pipeline.IsRunning() {
		t.Error("Expected pipeline not to be running initially")
	}

	// Start pipeline in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		pipeline.Run(ctx)
	}()

	// Give it a moment to start
	time.Sleep(time.Millisecond * 50)

	if !pipeline.IsRunning() {
		t.Error("Expected pipeline to be running")
	}

	// Cancel and wait for it to stop
	cancel()
	time.Sleep(time.Millisecond * 100)

	if pipeline.IsRunning() {
		t.Error("Expected pipeline to stop running after context cancellation")
	}
}

func TestPipeline_Run_AlreadyRunning(t *testing.T) {
	store := &MockStore{}
	classifier := &MockClassifier{}
	geocoder := &MockGeocoder{}
	cfg := config.PipelineConfig{
		RateLimit:     5.0,
		WorkerCount:   2,
		BatchSize:     10,
		RetryAttempts: 3,
		RetryDelay:    time.Millisecond * 100,
	}

	pipeline := New(store, classifier, geocoder, cfg)

	// Manually set running state
	pipeline.mu.Lock()
	pipeline.running = true
	pipeline.mu.Unlock()

	ctx := context.Background()
	err := pipeline.Run(ctx)
	if err == nil {
		t.Error("Expected error when pipeline already running, got nil")
	}
}
