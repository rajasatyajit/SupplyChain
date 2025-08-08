package main

import (
	"crypto/sha1"
	"encoding/hex"
	"regexp"
	"strings"
)

// Classifier MVP: keyword scoring
type Classifier struct{}

func NewClassifier() *Classifier { return &Classifier{} }

func (c *Classifier) Classify(a *Alert) {
	text := strings.ToLower(a.Title + " " + a.Summary)
	severity := "low"
	if containsAny(text, []string{"strike", "shutdown", "closure", "blocked", "riot", "earthquake", "hurricane"}) {
		severity = "high"
	} else if containsAny(text, []string{"delay", "congestion", "backlog", "maintenance"}) {
		severity = "medium"
	}
	a.Severity = severity
	// naive sentiment
	if containsAny(text, []string{"disrupt", "risk", "shortage", "warning"}) {
		a.Sentiment = "negative"
	} else {
		a.Sentiment = "neutral"
	}
}

// Geocoder MVP: regex extract simple locations and leave lat/lon zero
type Geocoder struct{ reCity *regexp.Regexp }

func NewGeocoder() *Geocoder { return &Geocoder{reCity: regexp.MustCompile(`(?i)\b(port of [a-z]+|[A-Z][a-z]+,?\s*[A-Z]{2})\b`)} }

func (g *Geocoder) Geocode(a *Alert) error {
	text := a.Title + " " + a.Summary
	if loc := g.reCity.FindString(text); loc != "" {
		a.Location = loc
	}
	return nil
}

func containsAny(s string, words []string) bool {
	for _, w := range words {
		if strings.Contains(s, w) { return true }
	}
	return false
}

func inferDisruption(s string) string {
	s = strings.ToLower(s)
	switch {
	case strings.Contains(s, "port"):
		return "port_status"
	case strings.Contains(s, "rail"):
		return "rail"
	case strings.Contains(s, "truck") || strings.Contains(s, "road"):
		return "road"
	case strings.Contains(s, "air") || strings.Contains(s, "airport"):
		return "air"
	default:
		return "general"
	}
}

func hashString(s string) string {
	h := sha1.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

