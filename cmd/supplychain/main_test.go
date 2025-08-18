package main

import (
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/rajasatyajit/SupplyChain/internal/logger"
)

// getFreePort returns an available TCP port
func getFreePort(t *testing.T) int {
	l, err := net.Listen("tcp", ":0")
	if err != nil { t.Fatalf("listen: %v", err) }
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func TestStartMetricsServer_Smoke(t *testing.T) {
	// Initialize logger to avoid nil logger panics
	logger.Init("error", "text")
	port := getFreePort(t)
	go startMetricsServer(port, "/metrics")
	url := fmt.Sprintf("http://localhost:%d/metrics", port)

	deadline := time.Now().Add(3 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			defer resp.Body.Close()
			// NoOp handler returns 404 Not Found
			if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusMovedPermanently {
				return
			}
		}
		lastErr = err
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("metrics server not reachable: %v", lastErr)
}
