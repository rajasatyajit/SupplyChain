package smoke

import (
	"net/http/httptest"
	"testing"
	"time"

	chi "github.com/go-chi/chi/v5"
	"github.com/rajasatyajit/SupplyChain/internal/api"
	"github.com/rajasatyajit/SupplyChain/internal/store"
)

func TestHealthAndAlertsSmoke(t *testing.T) {
	st := store.NewInMemoryStore()
	h := api.NewHandler(st, nil, "", "dev", time.Now().Format(time.RFC3339), "git")
	r := chi.NewRouter()
	h.RegisterRoutes(r)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("GET", "/v1/health", nil))
	if rec.Code != 200 {
		t.Fatalf("/v1/health %d", rec.Code)
	}

	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, httptest.NewRequest("GET", "/v1/alerts", nil))
	if rec2.Code != 200 {
		t.Fatalf("/v1/alerts %d", rec2.Code)
	}
}
