package pipeline

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"github.com/rajasatyajit/SupplyChain/internal/models"
)

// RSSSource implements Source for RSS feeds
type RSSSource struct {
	name     string
	urls     []string
	interval time.Duration
	client   *http.Client
}

// NewRSSSource creates a new RSS source
func NewRSSSource(name string, urls []string) *RSSSource {
	return &RSSSource{
		name:     name,
		urls:     urls,
		interval: 15 * time.Minute, // Default polling interval
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the source name
func (r *RSSSource) Name() string {
	return r.name
}

// Interval returns the polling interval
func (r *RSSSource) Interval() time.Duration {
	return r.interval
}

// Fetch fetches alerts from RSS feeds
func (r *RSSSource) Fetch(ctx context.Context) ([]models.Alert, error) {
	var allAlerts []models.Alert

	for _, url := range r.urls {
		alerts, err := r.fetchFromURL(ctx, url)
		if err != nil {
			// Log error but continue with other URLs
			continue
		}
		allAlerts = append(allAlerts, alerts...)
	}

	return allAlerts, nil
}

// fetchFromURL fetches and parses RSS from a single URL
func (r *RSSSource) fetchFromURL(ctx context.Context, url string) ([]models.Alert, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "SupplyChain-Monitor/1.0")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch RSS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	var rss RSS
	if err := xml.NewDecoder(resp.Body).Decode(&rss); err != nil {
		return nil, fmt.Errorf("parse RSS: %w", err)
	}

	return r.convertToAlerts(rss), nil
}

// convertToAlerts converts RSS items to Alert models
func (r *RSSSource) convertToAlerts(rss RSS) []models.Alert {
	var alerts []models.Alert

	for _, item := range rss.Channel.Items {
		alert := models.Alert{
			Source:      r.name,
			Title:       item.Title,
			Summary:     item.Description,
			URL:         item.Link,
			DetectedAt:  time.Now().UTC(),
			Confidence:  0.7, // Default confidence for RSS feeds
			Raw:         fmt.Sprintf("%+v", item),
		}

		// Parse published date
		if item.PubDate != "" {
			if pubDate, err := time.Parse(time.RFC1123Z, item.PubDate); err == nil {
				alert.PublishedAt = pubDate
			} else if pubDate, err := time.Parse(time.RFC1123, item.PubDate); err == nil {
				alert.PublishedAt = pubDate
			}
		}

		alerts = append(alerts, alert)
	}

	return alerts
}

// RSS represents the RSS feed structure
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

// Channel represents the RSS channel
type Channel struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
	Items       []Item `xml:"item"`
}

// Item represents an RSS item
type Item struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}