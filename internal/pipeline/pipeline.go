package pipeline

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
	"github.com/rajasatyajit/SupplyChain/config"
	"github.com/rajasatyajit/SupplyChain/internal/logger"
	"github.com/rajasatyajit/SupplyChain/internal/metrics"
	"github.com/rajasatyajit/SupplyChain/internal/models"
	"github.com/rajasatyajit/SupplyChain/pkg/utils"
)

// Source defines a pluggable data source implementation
type Source interface {
	Name() string
	Fetch(ctx context.Context) ([]models.Alert, error)
	Interval() time.Duration
}

// Classifier interface for alert classification
type Classifier interface {
	Classify(alert *models.Alert)
}

// Geocoder interface for alert geocoding
type Geocoder interface {
	Geocode(alert *models.Alert) error
}

// Store interface for alert storage
type Store interface {
	UpsertAlerts(ctx context.Context, alerts []models.Alert) error
}

// Pipeline coordinates concurrent fetching, classification, geocoding, and storing
type Pipeline struct {
	store      Store
	classifier Classifier
	geocoder   Geocoder
	clients    map[string]*http.Client
	limiter    *rate.Limiter
	sources    []Source
	cfg        config.PipelineConfig
	sem        *semaphore.Weighted
	mu         sync.RWMutex
	running    bool
}

// New creates a new pipeline instance
func New(store Store, classifier Classifier, geocoder Geocoder, cfg config.PipelineConfig) *Pipeline {
	p := &Pipeline{
		store:      store,
		classifier: classifier,
		geocoder:   geocoder,
		cfg:        cfg,
		clients: map[string]*http.Client{
			"default": {
				Timeout: 30 * time.Second,
				Transport: &http.Transport{
					MaxIdleConns:        100,
					MaxIdleConnsPerHost: 10,
					IdleConnTimeout:     90 * time.Second,
				},
			},
		},
		limiter: rate.NewLimiter(rate.Limit(cfg.RateLimit), int(cfg.RateLimit)),
		sem:     semaphore.NewWeighted(int64(cfg.WorkerCount)),
	}

	// Register sources (in production, this would be configurable)
	p.sources = []Source{
		NewRSSSource("Global Shipping News", []string{
			"https://news.un.org/feed/subscribe/en/news/region/africa/feed/rss.xml",
		}),
	}

	logger.Info("Pipeline initialized",
		"sources", len(p.sources),
		"rate_limit", cfg.RateLimit,
		"workers", cfg.WorkerCount,
	)

	return p
}

// Run starts the pipeline and runs until context is cancelled
func (p *Pipeline) Run(ctx context.Context) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return fmt.Errorf("pipeline already running")
	}
	p.running = true
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
	}()

	logger.Info("Starting pipeline")

	// Fan-out per-source pollers
	var wg sync.WaitGroup
	errChan := make(chan error, len(p.sources))

	for _, src := range p.sources {
		src := src
		wg.Add(1)

		go func() {
			defer wg.Done()

			if err := p.runSourcePoller(ctx, src); err != nil {
				select {
				case errChan <- fmt.Errorf("source %s: %w", src.Name(), err):
				case <-ctx.Done():
				}
			}
		}()
	}

	// Wait for all pollers to finish
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Collect any errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
		logger.Error("Pipeline source error", "error", err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("pipeline completed with %d errors", len(errors))
	}

	logger.Info("Pipeline stopped")
	return nil
}

// runSourcePoller runs a single source poller
func (p *Pipeline) runSourcePoller(ctx context.Context, src Source) error {
	logger.Info("Starting source poller", "source", src.Name())

	ticker := time.NewTicker(src.Interval())
	defer ticker.Stop()

	// Initial immediate run
	if err := p.runOnce(ctx, src); err != nil {
		logger.Error("Initial source run failed", "source", src.Name(), "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			logger.Info("Source poller stopping", "source", src.Name())
			return ctx.Err()
		case <-ticker.C:
			if err := p.runOnce(ctx, src); err != nil {
				logger.Error("Source run failed", "source", src.Name(), "error", err)

				// Implement exponential backoff on errors
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(p.cfg.RetryDelay):
					// Continue after delay
				}
			}
		}
	}
}

// runOnce executes a single pipeline run for a source
func (p *Pipeline) runOnce(ctx context.Context, src Source) error {
	start := time.Now()

	// Acquire semaphore to limit concurrent processing
	if err := p.sem.Acquire(ctx, 1); err != nil {
		return fmt.Errorf("acquire semaphore: %w", err)
	}
	defer p.sem.Release(1)

	// Rate limiting
	if err := p.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit: %w", err)
	}

	defer func() {
		duration := time.Since(start)
		metrics.RecordPipelineRun(src.Name(), duration)
		logger.Debug("Pipeline run completed",
			"source", src.Name(),
			"duration_ms", duration.Milliseconds(),
		)
	}()

	// Fetch alerts with retry logic
	var alerts []models.Alert
	var err error

	for attempt := 0; attempt <= p.cfg.RetryAttempts; attempt++ {
		if attempt > 0 {
			delay := time.Duration(attempt) * p.cfg.RetryDelay
			logger.Debug("Retrying fetch", "source", src.Name(), "attempt", attempt, "delay", delay)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		alerts, err = src.Fetch(ctx)
		if err == nil {
			break
		}

		logger.Warn("Fetch attempt failed",
			"source", src.Name(),
			"attempt", attempt+1,
			"error", err,
		)
	}

	if err != nil {
		metrics.RecordAlertProcessed(src.Name(), "fetch_error")
		return fmt.Errorf("%s fetch failed after %d attempts: %w", src.Name(), p.cfg.RetryAttempts+1, err)
	}

	if len(alerts) == 0 {
		logger.Debug("No alerts fetched", "source", src.Name())
		return nil
	}

	logger.Debug("Processing alerts", "source", src.Name(), "count", len(alerts))

	// Process alerts in batches
	batchSize := p.cfg.BatchSize
	if batchSize <= 0 {
		batchSize = len(alerts)
	}

	for i := 0; i < len(alerts); i += batchSize {
		end := i + batchSize
		if end > len(alerts) {
			end = len(alerts)
		}

		batch := alerts[i:end]
		if err := p.processBatch(ctx, src.Name(), batch); err != nil {
			logger.Error("Batch processing failed",
				"source", src.Name(),
				"batch_start", i,
				"batch_size", len(batch),
				"error", err,
			)
			metrics.RecordAlertProcessed(src.Name(), "process_error")
			return err
		}
	}

	metrics.RecordAlertProcessed(src.Name(), "success")
	logger.Info("Successfully processed alerts",
		"source", src.Name(),
		"count", len(alerts),
	)

	return nil
}

// processBatch processes a batch of alerts
func (p *Pipeline) processBatch(ctx context.Context, sourceName string, alerts []models.Alert) error {
	// Process each alert
	for i := range alerts {
		alert := &alerts[i]

		// Set source if not already set
		if alert.Source == "" {
			alert.Source = sourceName
		}

		// Set detection time
		if alert.DetectedAt.IsZero() {
			alert.DetectedAt = time.Now().UTC()
		}

		// Generate ID if not set
		if alert.ID == "" {
			alert.ID = utils.HashString(alert.URL + alert.Title + alert.PublishedAt.String())
		}

		// Set disruption type
		if alert.Disruption == "" {
			alert.Disruption = utils.InferDisruption(alert.Title + " " + alert.Summary)
		}

		// Classify alert
		p.classifier.Classify(alert)

		// Geocode alert
		if err := p.geocoder.Geocode(alert); err != nil {
			logger.Warn("Geocoding failed",
				"alert_id", alert.ID,
				"error", err,
			)
			// Reduce confidence but continue processing
			alert.Confidence *= 0.8
		}
	}

	// Store alerts
	return p.store.UpsertAlerts(ctx, alerts)
}

// IsRunning returns whether the pipeline is currently running
func (p *Pipeline) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

