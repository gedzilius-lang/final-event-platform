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

	"niteos.internal/ledger/internal/handler"
	"niteos.internal/ledger/internal/store"
	"niteos.internal/pkg/metrics"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := configFromEnv()

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer db.Close()
	if err := db.PingContext(context.Background()); err != nil {
		log.Fatalf("db ping: %v", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)

	ledgerStore := store.New(db)
	h := handler.New(ledgerStore)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("GET /metrics", metrics.Handler("ledger"))

	// Balance queries (used by wallet service, orders service)
	mux.HandleFunc("GET /balance/{user_id}", h.GetBalance)
	mux.HandleFunc("GET /balance/{user_id}/venue/{venue_id}", h.GetBalanceForVenue)

	// Event history (used by wallet service, reporting service)
	mux.HandleFunc("GET /events/{user_id}", h.GetUserEvents)
	mux.HandleFunc("GET /events/venue/{venue_id}", h.GetVenueEvents)

	// Write event (called by payments, orders, sessions, ticketing services)
	mux.HandleFunc("POST /events", h.WriteEvent)

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("ledger service listening", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

type config struct {
	ListenAddr  string
	DatabaseURL string
}

func configFromEnv() config {
	return config{
		ListenAddr:  envOrDefault("LEDGER_LISTEN_ADDR", ":8030"),
		DatabaseURL: envOrDefault("DATABASE_URL", "postgres://niteos:devpassword@localhost:5432/niteos?sslmode=disable"),
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
