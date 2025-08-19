package models

import "time"

// Alert represents a supply chain disruption alert
type Alert struct {
	ID          string    `json:"id" db:"id"`
	Source      string    `json:"source" db:"source"`
	Title       string    `json:"title" db:"title"`
	Summary     string    `json:"summary" db:"summary"`
	URL         string    `json:"url" db:"url"`
	DetectedAt  time.Time `json:"detected_at" db:"detected_at"`
	PublishedAt time.Time `json:"published_at" db:"published_at"`
	Region      string    `json:"region" db:"region"`
	Country     string    `json:"country" db:"country"`
	Location    string    `json:"location" db:"location"`
	Latitude    float64   `json:"latitude" db:"latitude"`
	Longitude   float64   `json:"longitude" db:"longitude"`
	Disruption  string    `json:"disruption" db:"disruption"`
	Severity    string    `json:"severity" db:"severity"`
	Sentiment   string    `json:"sentiment" db:"sentiment"`
	Confidence  float64   `json:"confidence" db:"confidence"`
	Raw         string    `json:"raw" db:"raw"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// AlertQuery represents query parameters for filtering alerts
type AlertQuery struct {
	IDs         []string  `json:"ids"`
	Sources     []string  `json:"sources"`
	Severities  []string  `json:"severities"`
	Disruptions []string  `json:"disruptions"`
	Regions     []string  `json:"regions"`
	Countries   []string  `json:"countries"`
	Since       time.Time `json:"since"`
	Until       time.Time `json:"until"`
	Limit       int       `json:"limit"`
	Offset      int       `json:"offset"`
}

// Matches checks if an alert matches the query criteria
func (q AlertQuery) Matches(alert Alert) bool {
	if len(q.IDs) > 0 && !contains(q.IDs, alert.ID) {
		return false
	}
	if len(q.Sources) > 0 && !contains(q.Sources, alert.Source) {
		return false
	}
	if len(q.Severities) > 0 && !contains(q.Severities, alert.Severity) {
		return false
	}
	if len(q.Disruptions) > 0 && !contains(q.Disruptions, alert.Disruption) {
		return false
	}
	if len(q.Regions) > 0 && !contains(q.Regions, alert.Region) {
		return false
	}
	if len(q.Countries) > 0 && !contains(q.Countries, alert.Country) {
		return false
	}
	if !q.Since.IsZero() && alert.DetectedAt.Before(q.Since) {
		return false
	}
	if !q.Until.IsZero() && alert.DetectedAt.After(q.Until) {
		return false
	}
	return true
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
