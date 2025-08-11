package main

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

type Alert struct {
	ID            string    `json:"id"`
	Source        string    `json:"source"`
	Title         string    `json:"title"`
	Summary       string    `json:"summary"`
	URL           string    `json:"url"`
	DetectedAt    time.Time `json:"detected_at"`
	PublishedAt   time.Time `json:"published_at"`
	Region        string    `json:"region"`
	Country       string    `json:"country"`
	Location      string    `json:"location"`
	Latitude      float64   `json:"latitude"`
	Longitude     float64   `json:"longitude"`
	Disruption    string    `json:"disruption"`
	Severity      string    `json:"severity"`
	Sentiment     string    `json:"sentiment"`
	Confidence    float64   `json:"confidence"`
	Raw           string    `json:"raw"`
}

type Store interface {
	UpsertAlerts(ctx context.Context, alerts []Alert) error
	QueryAlerts(ctx context.Context, q AlertQuery) ([]Alert, error)
}

type InMemoryStore struct {
	mu     sync.RWMutex
	alerts []Alert
}

func NewStore(db *DB) Store {
	// MVP: use in-memory store. A PostgresStore can be added later.
	return &InMemoryStore{}
}

func (s *InMemoryStore) UpsertAlerts(ctx context.Context, alerts []Alert) error {
	s.mu.Lock(); defer s.mu.Unlock()
	// naive: append all; real impl would de-dup by ID/URL
	s.alerts = append(s.alerts, alerts...)
	return nil
}

func (s *InMemoryStore) QueryAlerts(ctx context.Context, q AlertQuery) ([]Alert, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	res := make([]Alert, 0, len(s.alerts))
	for _, a := range s.alerts {
		if !q.Matches(a) { continue }
		res = append(res, a)
	}
	// sort newest first by DetectedAt
	sort.Slice(res, func(i, j int) bool { return res[i].DetectedAt.After(res[j].DetectedAt) })
	return res, nil
}

// Placeholder error for not implemented parts in future
var ErrNotImplemented = errors.New("not implemented")

