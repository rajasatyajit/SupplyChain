package models

import (
	"testing"
	"time"
)

func TestAlertQuery_Matches(t *testing.T) {
	alert := Alert{
		ID:          "test-alert-1",
		Source:      "test-source",
		Title:       "Test Alert",
		Summary:     "Test summary",
		DetectedAt:  time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		Severity:    "high",
		Disruption:  "port_status",
		Region:      "North America",
		Country:     "United States",
	}

	tests := []struct {
		name     string
		query    AlertQuery
		expected bool
	}{
		{
			name:     "Empty query matches all",
			query:    AlertQuery{},
			expected: true,
		},
		{
			name: "ID filter matches",
			query: AlertQuery{
				IDs: []string{"test-alert-1"},
			},
			expected: true,
		},
		{
			name: "ID filter doesn't match",
			query: AlertQuery{
				IDs: []string{"other-alert"},
			},
			expected: false,
		},
		{
			name: "Source filter matches",
			query: AlertQuery{
				Sources: []string{"test-source"},
			},
			expected: true,
		},
		{
			name: "Severity filter matches",
			query: AlertQuery{
				Severities: []string{"high", "medium"},
			},
			expected: true,
		},
		{
			name: "Multiple filters match",
			query: AlertQuery{
				Sources:    []string{"test-source"},
				Severities: []string{"high"},
				Regions:    []string{"North America"},
			},
			expected: true,
		},
		{
			name: "Time filter matches",
			query: AlertQuery{
				Since: time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
				Until: time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
			},
			expected: true,
		},
		{
			name: "Time filter before doesn't match",
			query: AlertQuery{
				Since: time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
			},
			expected: false,
		},
		{
			name: "Time filter after doesn't match",
			query: AlertQuery{
				Until: time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.query.Matches(alert)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "Item exists",
			slice:    []string{"a", "b", "c"},
			item:     "b",
			expected: true,
		},
		{
			name:     "Item doesn't exist",
			slice:    []string{"a", "b", "c"},
			item:     "d",
			expected: false,
		},
		{
			name:     "Empty slice",
			slice:    []string{},
			item:     "a",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}