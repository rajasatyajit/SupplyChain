package main

import (
	"encoding/json"
	"net/http"
)

func RegisterRoutes(r Router, store Store) {
	type chiRouter interface { Route(string, func(r Router)) }
	// Expect chi.Router implementing Router; adapter below.
	r.Route("/v1", func(r Router) {
		r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})

		r.Get("/alerts", func(w http.ResponseWriter, req *http.Request) {
			q := ParseAlertQuery(req.URL.Query())
			alerts, err := store.QueryAlerts(req.Context(), q)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if q.Limit > 0 && len(alerts) > q.Limit { alerts = alerts[:q.Limit] }
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"data": alerts,
				"count": len(alerts),
			})
		})
	})
}

// Router is a tiny abstraction over chi.Router to help with testing and inversion.
type Router interface {
	Get(pattern string, h http.HandlerFunc)
	Route(pattern string, fn func(r Router))
}

// chiAdapter implements Router using chi.Mux
type chiAdapter struct{ mux *chi.Mux }

func (a chiAdapter) Get(pattern string, h http.HandlerFunc) { a.mux.Get(pattern, h) }
func (a chiAdapter) Route(pattern string, fn func(r Router)) {
	a.mux.Route(pattern, func(r chi.Router) { fn(chiAdapter{mux: a.mux}) })
}

