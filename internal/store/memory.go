package store

import (
	"context"
	"sort"
	"sync"

	"github.com/rajasatyajit/SupplyChain/internal/models"
)

// InMemoryStore implements Store using in-memory storage
type InMemoryStore struct {
	mu     sync.RWMutex
	alerts map[string]models.Alert
}

// NewInMemoryStore creates a new in-memory store
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		alerts: make(map[string]models.Alert),
	}
}

// UpsertAlerts stores alerts in memory
func (s *InMemoryStore) UpsertAlerts(ctx context.Context, alerts []models.Alert) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, alert := range alerts {
		s.alerts[alert.ID] = alert
	}

	return nil
}

// QueryAlerts retrieves alerts from memory based on query parameters
func (s *InMemoryStore) QueryAlerts(ctx context.Context, q models.AlertQuery) ([]models.Alert, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []models.Alert
	for _, alert := range s.alerts {
		if q.Matches(alert) {
			result = append(result, alert)
		}
	}

	// Sort by DetectedAt descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].DetectedAt.After(result[j].DetectedAt)
	})

	// Apply limit and offset
	if q.Offset > 0 && q.Offset < len(result) {
		result = result[q.Offset:]
	} else if q.Offset >= len(result) {
		result = []models.Alert{}
	}

	if q.Limit > 0 && q.Limit < len(result) {
		result = result[:q.Limit]
	}

	return result, nil
}

// GetAlert retrieves a single alert by ID
func (s *InMemoryStore) GetAlert(ctx context.Context, id string) (*models.Alert, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if alert, exists := s.alerts[id]; exists {
		return &alert, nil
	}

	return nil, nil
}

// Health always returns nil for in-memory store
func (s *InMemoryStore) Health(ctx context.Context) error {
	return nil
}
