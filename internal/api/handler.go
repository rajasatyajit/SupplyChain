package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"fmt"
	"github.com/go-chi/chi/v5"
	middlewares "github.com/rajasatyajit/SupplyChain/internal/middleware"

	"github.com/rajasatyajit/SupplyChain/internal/database"
	"github.com/rajasatyajit/SupplyChain/internal/logger"
	"github.com/rajasatyajit/SupplyChain/internal/models"
	"github.com/rajasatyajit/SupplyChain/internal/store"
)

// Handler handles HTTP requests for the API
type Handler struct {
	store       store.Store
	db          *database.DB
	version     string
	buildTime   string
	gitCommit   string
	startTime   time.Time
	adminSecret string
}

// NewHandler creates a new API handler
func NewHandler(store store.Store, db *database.DB, adminSecret, version, buildTime, gitCommit string) *Handler {
	return 6Handler{
		store:       store,
		db:          db,
		version:     version,
		buildTime:   buildTime,
		gitCommit:   gitCommit,
		startTime:   time.Now(),
		adminSecret: adminSecret,
	}
}

// RegisterRoutes registers all API routes
func (h *Handler) RegisterRoutes(r *chi.Mux) {
	r.Route("/v1", func(r chi.Router) {
		// Health check endpoints
		r.Get("/health", h.healthHandler)
		r.Get("/health/ready", h.readinessHandler)
		r.Get("/health/live", h.livenessHandler)

		// API endpoints
		r.Get("/alerts", h.getAlertsHandler)
		r.Get("/alerts/{id}", h.getAlertHandler)

		// Account visibility endpoints (non-billable)
		r.Get("/me", h.meHandler)
		r.Get("/limits", h.limitsHandler)
		r.Get("/usage", h.usageHandler)
		r.Get("/usage/timeseries", h.usageTimeseriesHandler)

		// Billing endpoints (will be implemented with Stripe)
		r.Post("/billing/checkout-session", h.createCheckoutSession)
		r.Post("/billing/portal-session", h.createPortalSession)
		r.Post("/billing/webhook", h.stripeWebhook)

		// System info
		r.Get("/version", h.versionHandler)
	})

	// Admin routes (protected by shared secret middleware)
	r.Route("/v1/admin", func(r chi.Router) {
		r.With(middlewares.AdminSecret(h.adminSecret)).Group(func(r chi.Router) {
			r.Post("/accounts", h.adminCreateAccount)
			r.Post("/accounts/{account_id}/keys", h.adminCreateKey)
			r.Get("/accounts/{account_id}/keys", h.adminListKeys)
			r.Post("/keys/{key_id}/revoke", h.adminRevokeKey)
			r.Get("/usage", h.adminUsage)
		})
	})

	// Root health check
	r.Get("/health", h.healthHandler)
}

// healthHandler provides basic health check
func (h *Handler) healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC(),
		"version":   h.version,
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// readinessHandler checks if the application is ready to serve traffic
func (h *Handler) readinessHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	checks := map[string]string{
		"store": "ok",
	}

	statusCode := http.StatusOK

	// Check store health
	if err := h.store.Health(ctx); err != nil {
		checks["store"] = "error: " + err.Error()
		statusCode = http.StatusServiceUnavailable
	}

	response := map[string]interface{}{
		"status":    "ready",
		"timestamp": time.Now().UTC(),
		"checks":    checks,
	}

	h.writeJSONResponse(w, statusCode, response)
}

// livenessHandler checks if the application is alive
func (h *Handler) livenessHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "alive",
		"timestamp": time.Now().UTC(),
		"uptime":    time.Since(h.startTime).String(),
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// versionHandler returns version information
func (h *Handler) versionHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"version":    h.version,
		"build_time": h.buildTime,
		"git_commit": h.gitCommit,
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// getAlertsHandler handles GET /alerts
func (h *Handler) getAlertsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	q, err := h.parseAlertQuery(r)
	if err != nil {
		h.writeErrorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}

	alerts, err := h.store.QueryAlerts(ctx, q)
	if err != nil {
		logger.WithContext(ctx).Error("Failed to query alerts", "error", err)
		h.writeErrorResponse(w, r, http.StatusInternalServerError, "Internal server error")
		return
	}

	response := map[string]interface{}{
		"data":      alerts,
		"count":     len(alerts),
		"timestamp": time.Now().UTC(),
	}

	w.Header().Set("Cache-Control", "public, max-age=60")
	h.writeJSONResponse(w, http.StatusOK, response)
}

// getAlertHandler handles GET /alerts/{id}
func (h *Handler) getAlertHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	alertID := chi.URLParam(r, "id")

	if alertID == "" {
		h.writeErrorResponse(w, r, http.StatusBadRequest, "alert ID is required")
		return
	}

	alert, err := h.store.GetAlert(ctx, alertID)
	if err != nil {
		logger.WithContext(ctx).Error("Failed to get alert", "error", err, "alert_id", alertID)
		h.writeErrorResponse(w, r, http.StatusInternalServerError, "Internal server error")
		return
	}

	if alert == nil {
		h.writeErrorResponse(w, r, http.StatusNotFound, "Alert not found")
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=300")
	h.writeJSONResponse(w, http.StatusOK, alert)
}

// parseAlertQuery parses query parameters into AlertQuery
func (h *Handler) parseAlertQuery(r *http.Request) (models.AlertQuery, error) {
	q := models.AlertQuery{}

	// Parse limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			return q, fmt.Errorf("invalid limit: %s", limitStr)
		}
		if limit < 0 || limit > 1000 {
			return q, fmt.Errorf("limit must be between 0 and 1000")
		}
		q.Limit = limit
	}

	// Parse offset
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			return q, fmt.Errorf("invalid offset: %s", offsetStr)
		}
		if offset < 0 {
			return q, fmt.Errorf("offset must be non-negative")
		}
		q.Offset = offset
	}

	// Parse time filters
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		since, err := time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			return q, fmt.Errorf("invalid since format: %s", sinceStr)
		}
		q.Since = since
	}

	if untilStr := r.URL.Query().Get("until"); untilStr != "" {
		until, err := time.Parse(time.RFC3339, untilStr)
		if err != nil {
			return q, fmt.Errorf("invalid until format: %s", untilStr)
		}
		q.Until = until
	}

	// Parse array filters
	q.Sources = r.URL.Query()["source"]
	q.Severities = r.URL.Query()["severity"]
	q.Disruptions = r.URL.Query()["disruption"]
	q.Regions = r.URL.Query()["region"]
	q.Countries = r.URL.Query()["country"]

	return q, nil
}

// writeJSONResponse writes a JSON response
func (h *Handler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// writeErrorResponse writes a standardized error response
func (h *Handler) writeErrorResponse(w http.ResponseWriter, r *http.Request, statusCode int, message string) {
	response := ErrorResponse{
		Error:     http.StatusText(statusCode),
		Message:   message,
		Timestamp: time.Now().UTC(),
		RequestID: r.Header.Get("X-Request-ID"),
	}

	h.writeJSONResponse(w, statusCode, response)
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error     string    `json:"error"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	RequestID string    `json:"request_id,omitempty"`
}
