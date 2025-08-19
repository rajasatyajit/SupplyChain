package metrics

import (
	"net/http"
	"time"
)

// Metrics interface for dependency injection
type Metrics interface {
	RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration)
	RecordAlertProcessed(source, status string)
	RecordPipelineRun(source string, duration time.Duration)
	SetDBConnectionsActive(count float64)
	RecordDBQuery(operation, status string)
	Handler() http.Handler
}

// NoOpMetrics provides a no-op implementation
type NoOpMetrics struct{}

func (m *NoOpMetrics) RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration) {
}
func (m *NoOpMetrics) RecordAlertProcessed(source, status string)              {}
func (m *NoOpMetrics) RecordPipelineRun(source string, duration time.Duration) {}
func (m *NoOpMetrics) SetDBConnectionsActive(count float64)                    {}
func (m *NoOpMetrics) RecordDBQuery(operation, status string)                  {}
func (m *NoOpMetrics) Handler() http.Handler                                   { return http.NotFoundHandler() }

// Global metrics instance
var globalMetrics Metrics = &NoOpMetrics{}

// Init initializes metrics (no-op for now, can be extended with Prometheus)
func Init() {
	// For now, keep using no-op metrics
	// In a full implementation, this would initialize Prometheus metrics
}

// Handler returns the metrics handler
func Handler() http.Handler {
	return globalMetrics.Handler()
}

// RecordHTTPRequest records HTTP request metrics
func RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration) {
	globalMetrics.RecordHTTPRequest(method, endpoint, statusCode, duration)
}

// RecordAlertProcessed records alert processing metrics
func RecordAlertProcessed(source, status string) {
	globalMetrics.RecordAlertProcessed(source, status)
}

// RecordPipelineRun records pipeline run metrics
func RecordPipelineRun(source string, duration time.Duration) {
	globalMetrics.RecordPipelineRun(source, duration)
}

// SetDBConnectionsActive sets the number of active database connections
func SetDBConnectionsActive(count float64) {
	globalMetrics.SetDBConnectionsActive(count)
}

// RecordDBQuery records database query metrics
func RecordDBQuery(operation, status string) {
	globalMetrics.RecordDBQuery(operation, status)
}
