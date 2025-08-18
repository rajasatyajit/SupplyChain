package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rajasatyajit/SupplyChain/internal/logger"
)

func TestLogging(t *testing.T) {
	// Initialize logger to avoid nil logger in middleware
	logger.Init("error", "text")
	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with logging middleware
	wrappedHandler := Logging(handler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	
	// Add request ID to context (simulating chi middleware)
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.RequestIDKey, "test-request-id")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Execute request
	wrappedHandler.ServeHTTP(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got %s", w.Body.String())
	}
}

func TestMetrics(t *testing.T) {
	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with metrics middleware
	wrappedHandler := Metrics(handler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Execute request
	wrappedHandler.ServeHTTP(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got %s", w.Body.String())
	}
}

func TestSecurity(t *testing.T) {
	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with security middleware
	wrappedHandler := Security(handler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Execute request
	wrappedHandler.ServeHTTP(w, req)

	// Check security headers
	expectedHeaders := map[string]string{
		"X-Content-Type-Options":   "nosniff",
		"X-Frame-Options":          "DENY",
		"X-XSS-Protection":         "1; mode=block",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"Content-Security-Policy":  "default-src 'self'",
		"Referrer-Policy":          "strict-origin-when-cross-origin",
	}

	for header, expectedValue := range expectedHeaders {
		actualValue := w.Header().Get(header)
		if actualValue != expectedValue {
			t.Errorf("Expected header %s: %s, got %s", header, expectedValue, actualValue)
		}
	}

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRateLimit(t *testing.T) {
	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with rate limit middleware (2 requests per minute)
	wrappedHandler := RateLimit(2)(handler)

	// Create test requests from same IP
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.1:12346"
	
	req3 := httptest.NewRequest("GET", "/test", nil)
	req3.RemoteAddr = "192.168.1.1:12347"

	// First request should succeed
	w1 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Errorf("Expected first request to succeed, got status %d", w1.Code)
	}

	// Second request should succeed
	w2 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("Expected second request to succeed, got status %d", w2.Code)
	}

	// Third request should be rate limited
	w3 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w3, req3)
	if w3.Code != http.StatusTooManyRequests {
		t.Errorf("Expected third request to be rate limited, got status %d", w3.Code)
	}

	// Check retry-after header
	retryAfter := w3.Header().Get("Retry-After")
	if retryAfter != "60" {
		t.Errorf("Expected Retry-After header '60', got %s", retryAfter)
	}
}

func TestCORS(t *testing.T) {
	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	allowedOrigins := []string{"https://example.com", "https://app.example.com"}
	wrappedHandler := CORS(allowedOrigins)(handler)

	tests := []struct {
		name           string
		origin         string
		method         string
		expectedStatus int
		expectOrigin   bool
	}{
		{
			name:           "Allowed origin",
			origin:         "https://example.com",
			method:         "GET",
			expectedStatus: http.StatusOK,
			expectOrigin:   true,
		},
		{
			name:           "Disallowed origin",
			origin:         "https://malicious.com",
			method:         "GET",
			expectedStatus: http.StatusOK,
			expectOrigin:   false,
		},
		{
			name:           "OPTIONS request",
			origin:         "https://example.com",
			method:         "OPTIONS",
			expectedStatus: http.StatusOK,
			expectOrigin:   true,
		},
		{
			name:           "Wildcard origin",
			origin:         "https://any.com",
			method:         "GET",
			expectedStatus: http.StatusOK,
			expectOrigin:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/test", nil)
			req.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check CORS headers
			allowMethods := w.Header().Get("Access-Control-Allow-Methods")
			if !strings.Contains(allowMethods, "GET") {
				t.Error("Expected Access-Control-Allow-Methods to contain GET")
			}

			allowHeaders := w.Header().Get("Access-Control-Allow-Headers")
			if !strings.Contains(allowHeaders, "Content-Type") {
				t.Error("Expected Access-Control-Allow-Headers to contain Content-Type")
			}

			maxAge := w.Header().Get("Access-Control-Max-Age")
			if maxAge != "86400" {
				t.Errorf("Expected Access-Control-Max-Age '86400', got %s", maxAge)
			}

			// Check origin header
			allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if tt.expectOrigin && allowOrigin != tt.origin {
				t.Errorf("Expected Access-Control-Allow-Origin %s, got %s", tt.origin, allowOrigin)
			}
			if !tt.expectOrigin && allowOrigin == tt.origin {
				t.Errorf("Did not expect Access-Control-Allow-Origin to be set to %s", tt.origin)
			}
		})
	}

	// Test wildcard origin
	t.Run("Wildcard origin", func(t *testing.T) {
		wildcardHandler := CORS([]string{"*"})(handler)
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://any.com")
		w := httptest.NewRecorder()

		wildcardHandler.ServeHTTP(w, req)

		allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
		if allowOrigin != "https://any.com" {
			t.Errorf("Expected wildcard to allow any origin, got %s", allowOrigin)
		}
	})
}