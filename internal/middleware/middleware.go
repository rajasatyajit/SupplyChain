package middleware

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rajasatyajit/SupplyChain/config"
	"github.com/rajasatyajit/SupplyChain/internal/auth"
	"github.com/rajasatyajit/SupplyChain/internal/logger"
	"github.com/rajasatyajit/SupplyChain/internal/metrics"
)

// Logging provides structured logging for HTTP requests
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Add request ID to context
		requestID := middleware.GetReqID(r.Context())
		ctx := context.WithValue(r.Context(), "request_id", requestID) //nolint:staticcheck // string context key used intentionally for cross-package simplicity
		r = r.WithContext(ctx)

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		defer func() {
			duration := time.Since(start)

			logger.WithContext(ctx).Info("HTTP request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"duration_ms", duration.Milliseconds(),
				"bytes", ww.BytesWritten(),
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
			)
		}()

		next.ServeHTTP(ww, r)
	})
}

// Metrics records HTTP metrics
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		defer func() {
			duration := time.Since(start)
			metrics.RecordHTTPRequest(
				r.Method,
				r.URL.Path,
				ww.Status(),
				duration,
			)
		}()

		next.ServeHTTP(ww, r)
	})
}

// Security adds security headers
func Security(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		next.ServeHTTP(w, r)
	})
}

// APIKeyAuth enforces API key authentication when enabled via configuration.
// It expects Authorization: Bearer <api_key> by default. Optionally enforces an agent/human header.
func APIKeyAuth(cfg config.AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.RequireAPIKeys {
				// Pass-through when disabled
				next.ServeHTTP(w, r)
				return
			}

			// Check header presence
			raw := r.Header.Get(cfg.KeyHeader)
			if raw == "" {
				http.Error(w, "Missing API key", http.StatusUnauthorized)
				return
			}
			var key string
			if strings.HasPrefix(strings.ToLower(raw), "bearer ") {
				key = strings.TrimSpace(raw[len("Bearer "):])
			} else {
				// Allow raw key in header for flexibility
				key = strings.TrimSpace(raw)
			}
			if key == "" {
				http.Error(w, "Invalid API key", http.StatusUnauthorized)
				return
			}

			// Optional client type header enforcement
			if cfg.EnableAgentHeader {
				ct := r.Header.Get(cfg.AgentHeaderName)
				if ct != "agent" && ct != "human" {
					http.Error(w, "Invalid client type", http.StatusUnauthorized)
					return
				}
			}

			// Verify key and attach principal (placeholder implementation)
			clientType := r.Header.Get(cfg.AgentHeaderName)
			principal, err := auth.VerifyAPIKey(r, key, clientType)
			if err != nil || principal == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := auth.WithPrincipal(r.Context(), principal)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

// write429 writes Too Many Requests
func write429(w http.ResponseWriter) {
	http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
}

// RateLimit provides rate limiting (basic implementation)
func RateLimit(requestsPerMinute int) func(http.Handler) http.Handler {
	// This is a simple in-memory rate limiter
	// For production, consider using Redis-based rate limiting
	clients := make(map[string][]time.Time)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := r.RemoteAddr
			if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
				clientIP = host
			}
			now := time.Now()

			// Clean old entries
			if timestamps, exists := clients[clientIP]; exists {
				var validTimestamps []time.Time
				for _, ts := range timestamps {
					if now.Sub(ts) < time.Minute {
						validTimestamps = append(validTimestamps, ts)
					}
				}
				clients[clientIP] = validTimestamps
			}

			// Check rate limit
			if len(clients[clientIP]) >= requestsPerMinute {
				w.Header().Set("Retry-After", "60")
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			// Add current request
			clients[clientIP] = append(clients[clientIP], now)

			next.ServeHTTP(w, r)
		})
	}
}

// AdminSecret protects admin routes via a simple shared secret
func AdminSecret(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if secret == "" {
				http.Error(w, "admin not configured", http.StatusForbidden)
				return
			}
			if r.Header.Get("X-Admin-Secret") != secret {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CORS handles CORS headers
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
