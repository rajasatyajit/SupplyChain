package metrics

import (
	"net/http"
	"testing"
	"time"
)

// Ensure NoOpMetrics methods do not panic and global functions delegate without error
func TestNoOpMetricsAndDelegates(t *testing.T) {
	m := &NoOpMetrics{}
	m.RecordHTTPRequest("GET", "/x", 200, time.Millisecond)
	m.RecordAlertProcessed("src", "ok")
	m.RecordPipelineRun("src", time.Millisecond)
	m.SetDBConnectionsActive(1)
	m.RecordDBQuery("exec", "ok")
	h := m.Handler()
	if h == nil {
		t.Fatalf("NoOp handler is nil")
	}

	// Delegates
	RecordHTTPRequest("GET", "/x", 200, time.Millisecond)
	RecordAlertProcessed("src", "ok")
	RecordPipelineRun("src", time.Millisecond)
	SetDBConnectionsActive(2)
	RecordDBQuery("query", "ok")

	// Handler should be NotFound
	req, _ := http.NewRequest("GET", "/metrics", nil)
	rw := httptestResponseRecorder{}
	h.ServeHTTP(&rw, req)
	if rw.status == 0 {
		t.Errorf("expected status set, got 0")
	}
}

type httptestResponseRecorder struct{
	header http.Header
	status int
}

func (w *httptestResponseRecorder) Header() http.Header {
	if w.header == nil { w.header = make(http.Header) }
	return w.header
}
func (w *httptestResponseRecorder) Write(b []byte) (int, error) { return len(b), nil }
func (w *httptestResponseRecorder) WriteHeader(code int) { w.status = code }
