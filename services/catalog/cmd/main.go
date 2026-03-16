package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"niteos.internal/catalog/internal/handler"
	"niteos.internal/catalog/internal/store"
	"niteos.internal/pkg/metrics"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	cfg := configFromEnv()
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil { log.Fatalf("db open: %v", err) }
	defer db.Close()
	if err := db.PingContext(context.Background()); err != nil { log.Fatalf("db ping: %v", err) }
	db.SetMaxOpenConns(10); db.SetMaxIdleConns(5)
	s := store.New(db)
	h := handler.New(s)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json"); fmt.Fprint(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("GET /metrics", metrics.Handler("catalog"))
	mux.HandleFunc("POST /venues", h.CreateVenue)
	mux.HandleFunc("GET /venues/{venue_id}", h.GetVenue)
	mux.HandleFunc("GET /venues", h.ListVenues)
	mux.HandleFunc("PATCH /venues/{venue_id}", h.UpdateVenue)
	mux.HandleFunc("GET /venues/{venue_id}/items", h.ListItems)
	mux.HandleFunc("POST /venues/{venue_id}/items", h.CreateItem)
	mux.HandleFunc("GET /venues/{venue_id}/items/{item_id}", h.GetItem)
	mux.HandleFunc("PATCH /venues/{venue_id}/items/{item_id}", h.UpdateItem)
	mux.HandleFunc("DELETE /venues/{venue_id}/items/{item_id}", h.DeleteItem)
	mux.HandleFunc("GET /venues/{venue_id}/events", h.ListEvents)
	mux.HandleFunc("POST /venues/{venue_id}/events", h.CreateEvent)
	mux.HandleFunc("GET /venues/{venue_id}/happy-hours", h.ListHappyHours)
	mux.HandleFunc("POST /venues/{venue_id}/happy-hours", h.CreateHappyHour)
	srv := &http.Server{Addr: cfg.ListenAddr, Handler: mux, ReadTimeout: 10*time.Second, WriteTimeout: 10*time.Second}
	go func() {
		slog.Info("catalog service listening", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed { log.Fatalf("listen: %v", err) }
	}()
	quit := make(chan os.Signal, 1); signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM); <-quit
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second); defer cancel(); _ = srv.Shutdown(ctx)
}

type config struct{ ListenAddr, DatabaseURL string }

func configFromEnv() config {
	return config{
		ListenAddr:  envOrDefault("CATALOG_LISTEN_ADDR", ":8080"),
		DatabaseURL: envOrDefault("DATABASE_URL", "postgres://niteos:devpassword@localhost:5432/niteos?sslmode=disable"),
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" { return v }; return def
}
