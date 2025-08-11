package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type RSSSource struct {
	name  string
	feeds []string
	cli   *http.Client
}

type rssFeed struct {
	Channel struct {
		Title string    `xml:"title"`
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func NewRSSSource(name string, feeds []string) *RSSSource {
	return &RSSSource{name: name, feeds: feeds, cli: &http.Client{Timeout: 10 * time.Second}}
}

func (r *RSSSource) Name() string       { return r.name }
func (r *RSSSource) Interval() time.Duration { return 2 * time.Minute }

func (r *RSSSource) Fetch(ctx context.Context) ([]Alert, error) {
	var all []Alert
	for _, url := range r.feeds {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := r.cli.Do(req)
		if err != nil { continue }
		io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // read to buffer for parsing
		resp.Body.Close()
		// Re-fetch for parsing since we discarded; in prod, parse the first read bytes
		resp2, err := r.cli.Get(url)
		if err != nil { continue }
		defer resp2.Body.Close()
		var feed rssFeed
		if err := xml.NewDecoder(io.LimitReader(resp2.Body, 2<<20)).Decode(&feed); err != nil { continue }
		for _, it := range feed.Channel.Items {
			pub := time.Now()
			if t, err := parsePubDate(it.PubDate); err == nil { pub = t }
			alert := Alert{
				ID:         hashString(it.Link),
				Source:     r.name,
				Title:      strings.TrimSpace(it.Title),
				Summary:    strings.TrimSpace(it.Description),
				URL:        it.Link,
				DetectedAt: time.Now().UTC(),
				PublishedAt: pub,
				Disruption: inferDisruption(it.Title+" "+it.Description),
				Confidence: 0.6,
				Raw:        fmt.Sprintf("%s\n%s", it.Title, it.Description),
			}
			all = append(all, alert)
		}
	}
	return all, nil
}

func parsePubDate(s string) (time.Time, error) {
	layouts := []string{time.RFC1123Z, time.RFC1123, time.RFC822, time.RFC822Z, time.RFC3339}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil { return t, nil }
	}
	return time.Time{}, fmt.Errorf("parse pubdate: %s", s)
}

