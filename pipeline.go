package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Source defines a pluggable data source implementation
// that can fetch raw events and transform them into Alerts.
type Source interface {
	Name() string
	Fetch(ctx context.Context) ([]Alert, error)
	Interval() time.Duration // suggested polling interval
}

// Pipeline coordinates concurrent fetching, classification, geocoding, and storing.
type Pipeline struct {
	store      Store
	classifier *Classifier
	geo        *Geocoder
	clients    map[string]*http.Client
	limiter    *rate.Limiter
	sources    []Source
	mu         sync.Mutex
}

func NewPipeline(store Store, classifier *Classifier, geo *Geocoder) *Pipeline {
	p := &Pipeline{
		store:      store,
		classifier: classifier,
		geo:        geo,
		clients:    map[string]*http.Client{"default": {Timeout: 10 * time.Second}},
		limiter:    rate.NewLimiter(rate.Every(200*time.Millisecond), 1), // 5 rps global
	}
	// Register MVP sources
	p.sources = []Source{
		NewRSSSource("Global Shipping News", []string{
			"https://news.un.org/feed/subscribe/en/news/region/africa/feed/rss.xml", // example placeholder feeds
		}),
	}
	return p
}

func (p *Pipeline) Run(ctx context.Context) {
	// Fan-out per-source pollers
	wg := sync.WaitGroup{}
	for _, src := range p.sources {
		src := src
		wg.Add(1)
		go func() {
			defer wg.Done()
			ticker := time.NewTicker(src.Interval())
			defer ticker.Stop()
			// initial immediate tick
			for {
				if err := p.runOnce(ctx, src); err != nil {
					// log error; in production use structured logs/metrics
				}
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
			}
		}()
	}
	wg.Wait()
}

func (p *Pipeline) runOnce(ctx context.Context, src Source) error {
	if err := p.limiter.Wait(ctx); err != nil { return err }
	alerts, err := src.Fetch(ctx)
	if err != nil { return fmt.Errorf("%s fetch: %w", src.Name(), err) }
	// Classify, geocode
	for i := range alerts {
		p.classifier.Classify(&alerts[i])
		if err := p.geo.Geocode(&alerts[i]); err != nil {
			// keep going but lower confidence
			alerts[i].Confidence *= 0.8
		}
	}
	return p.store.UpsertAlerts(ctx, alerts)
}

