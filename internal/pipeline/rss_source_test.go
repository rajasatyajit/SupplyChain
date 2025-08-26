package pipeline

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRSSSource_Name(t *testing.T) {
	source := NewRSSSource("Test Source", []string{"http://example.com/rss"})

	if source.Name() != "Test Source" {
		t.Errorf("Expected name 'Test Source', got %s", source.Name())
	}
}

func TestRSSSource_Interval(t *testing.T) {
	source := NewRSSSource("Test Source", []string{"http://example.com/rss"})

	expected := 15 * time.Minute
	if source.Interval() != expected {
		t.Errorf("Expected interval %v, got %v", expected, source.Interval())
	}
}

func TestRSSSource_Fetch(t *testing.T) {
	// Mock RSS feed
	rssContent := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test RSS Feed</title>
    <description>Test feed for unit tests</description>
    <link>http://example.com</link>
    <item>
      <title>Port Strike Disrupts Operations</title>
      <description>Major strike affecting port operations</description>
      <link>http://example.com/news/1</link>
      <pubDate>Mon, 15 Jan 2024 10:00:00 GMT</pubDate>
      <guid>http://example.com/news/1</guid>
    </item>
    <item>
      <title>Traffic Delays on Highway</title>
      <description>Heavy congestion reported</description>
      <link>http://example.com/news/2</link>
      <pubDate>Mon, 15 Jan 2024 11:00:00 GMT</pubDate>
      <guid>http://example.com/news/2</guid>
    </item>
  </channel>
</rss>`

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(rssContent))
	}))
	defer server.Close()

	source := NewRSSSource("Test Source", []string{server.URL})
	ctx := context.Background()

	alerts, err := source.Fetch(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(alerts) != 2 {
		t.Errorf("Expected 2 alerts, got %d", len(alerts))
	}

	// Check first alert
	alert1 := alerts[0]
	if alert1.Source != "Test Source" {
		t.Errorf("Expected source 'Test Source', got %s", alert1.Source)
	}

	if alert1.Title != "Port Strike Disrupts Operations" {
		t.Errorf("Expected title 'Port Strike Disrupts Operations', got %s", alert1.Title)
	}

	if alert1.Summary != "Major strike affecting port operations" {
		t.Errorf("Expected summary 'Major strike affecting port operations', got %s", alert1.Summary)
	}

	if alert1.URL != "http://example.com/news/1" {
		t.Errorf("Expected URL 'http://example.com/news/1', got %s", alert1.URL)
	}

	if alert1.Confidence != 0.7 {
		t.Errorf("Expected confidence 0.7, got %f", alert1.Confidence)
	}

	// Check that published date was parsed
	if alert1.PublishedAt.IsZero() {
		t.Error("Expected published date to be parsed")
	}
}

func TestRSSSource_FetchError(t *testing.T) {
	// Test with invalid URL
	source := NewRSSSource("Test Source", []string{"http://invalid-url-that-does-not-exist.com/rss"})
	ctx := context.Background()

	alerts, err := source.Fetch(ctx)
	if err != nil {
		t.Errorf("Expected no error (should continue with other URLs), got %v", err)
	}

	// Should return empty slice when all URLs fail
	if len(alerts) != 0 {
		t.Errorf("Expected 0 alerts when fetch fails, got %d", len(alerts))
	}
}

func TestRSSSource_FetchHTTPError(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	source := NewRSSSource("Test Source", []string{server.URL})
	ctx := context.Background()

	alerts, err := source.Fetch(ctx)
	if err != nil {
		t.Errorf("Expected no error (should continue with other URLs), got %v", err)
	}

	if len(alerts) != 0 {
		t.Errorf("Expected 0 alerts when HTTP error occurs, got %d", len(alerts))
	}
}

func TestRSSSource_FetchInvalidXML(t *testing.T) {
	// Create test server with invalid XML
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("invalid xml content"))
	}))
	defer server.Close()

	source := NewRSSSource("Test Source", []string{server.URL})
	ctx := context.Background()

	alerts, err := source.Fetch(ctx)
	if err != nil {
		t.Errorf("Expected no error (should continue with other URLs), got %v", err)
	}

	if len(alerts) != 0 {
		t.Errorf("Expected 0 alerts when XML is invalid, got %d", len(alerts))
	}
}

func TestRSSSource_ConvertToAlerts(t *testing.T) {
	source := NewRSSSource("Test Source", []string{})

	rss := RSS{
		Channel: Channel{
			Title:       "Test Channel",
			Description: "Test Description",
			Link:        "http://example.com",
			Items: []Item{
				{
					Title:       "Test Item 1",
					Description: "Test Description 1",
					Link:        "http://example.com/1",
					PubDate:     "Mon, 15 Jan 2024 10:00:00 GMT",
					GUID:        "guid-1",
				},
				{
					Title:       "Test Item 2",
					Description: "Test Description 2",
					Link:        "http://example.com/2",
					PubDate:     "invalid date",
					GUID:        "guid-2",
				},
			},
		},
	}

	alerts := source.convertToAlerts(rss)

	if len(alerts) != 2 {
		t.Errorf("Expected 2 alerts, got %d", len(alerts))
	}

	// Check first alert with valid date
	alert1 := alerts[0]
	if alert1.Source != "Test Source" {
		t.Errorf("Expected source 'Test Source', got %s", alert1.Source)
	}

	if alert1.Title != "Test Item 1" {
		t.Errorf("Expected title 'Test Item 1', got %s", alert1.Title)
	}

	if alert1.PublishedAt.IsZero() {
		t.Error("Expected published date to be parsed for valid date")
	}

	// Check second alert with invalid date
	alert2 := alerts[1]
	if !alert2.PublishedAt.IsZero() {
		t.Error("Expected published date to be zero for invalid date")
	}

	// Check that raw field contains item data
	if !strings.Contains(alert1.Raw, "Test Item 1") {
		t.Error("Expected raw field to contain item data")
	}
}
