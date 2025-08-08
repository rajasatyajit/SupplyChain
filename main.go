package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	ctx := context.Background()

	// Load config from env (simple)
	addr := getEnv("HTTP_ADDR", ":8080")

	// Initialize subsystems
	db, err := NewDB(ctx)
	if err != nil {
		log.Fatalf("db init: %v", err)
	}
	defer db.Close(ctx)

	store := NewStore(db)
	classifier := NewClassifier()
	geo := NewGeocoder()

	pipeline := NewPipeline(store, classifier, geo)
	go pipeline.Run(ctx)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	RegisterRoutes(r, store)

	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}

func getEnv(k, def string) string { if v := os.Getenv(k); v != "" { return v }; return def }

