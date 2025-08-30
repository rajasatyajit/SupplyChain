package api

import (
	"net/http"
	"strconv"
	"time"
)

// usageTimeseriesHandler returns a minimal timeseries structure for the current account
// Query: bucket (hour|day), start, end (RFC3339). Currently returns empty dataset placeholder.
func (h *Handler) usageTimeseriesHandler(w http.ResponseWriter, r *http.Request) {
	// Parse optional params; default last 7 days daily buckets
	bucket := r.URL.Query().Get("bucket")
	if bucket == "" {
		bucket = "day"
	}
	_ = bucket
	end := time.Now().UTC()
	start := end.AddDate(0, 0, -7)
	if v := r.URL.Query().Get("start"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			start = t
		}
	}
	if v := r.URL.Query().Get("end"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			end = t
		}
	}

	// Placeholder generation of N buckets with zero usage
	buckets := []map[string]any{}
	// Choose step
	step := 24 * time.Hour
	if bucket == "hour" {
		step = time.Hour
	}
	for ts := start; !ts.After(end); ts = ts.Add(step) {
		buckets = append(buckets, map[string]any{
			"ts":    ts,
			"total": 0,
		})
	}

	resp := map[string]any{
		"bucket": bucket,
		"start":  start,
		"end":    end,
		"data":   buckets,
	}
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// adminUsage returns a placeholder usage summary. In production, this should aggregate from Redis/DB.
// Query: limit, offset (unused for now)
func (h *Handler) adminUsage(w http.ResponseWriter, r *http.Request) {
	_, _ = strconv.Atoi(r.URL.Query().Get("limit"))
	_, _ = strconv.Atoi(r.URL.Query().Get("offset"))
	resp := map[string]any{
		"total_accounts": 0,
		"total_usage":    0,
		"by_account":     []any{},
	}
	h.writeJSONResponse(w, http.StatusOK, resp)
}
