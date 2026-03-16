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

	"niteos.internal/orders/internal/handler"
	"niteos.internal/orders/internal/store"
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
	db.SetMaxOpenConns(15)
	db.SetMaxIdleConns(5)

	s := store.New(db)
	h := handler.New(s, cfg.LedgerServiceURL, cfg.SessionsServiceURL)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("GET /metrics", metrics.Handler("orders"))
	mux.HandleFunc("POST /", h.CreateOrder)
	mux.HandleFunc("GET /{order_id}", h.GetOrder)
	mux.HandleFunc("POST /{order_id}/finalize", h.FinalizeOrder)
	mux.HandleFunc("POST /{order_id}/void", h.VoidOrder)

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("orders service listening", "addr", cfg.ListenAddr)
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
	ListenAddr         string
	DatabaseURL        string
	LedgerServiceURL   string
	SessionsServiceURL string
}

func configFromEnv() config {
	return config{
		ListenAddr:         envOrDefault("ORDERS_LISTEN_ADDR", ":8090"),
		DatabaseURL:        envOrDefault("DATABASE_URL", "postgres://niteos:devpassword@localhost:5432/niteos?sslmode=disable"),
		LedgerServiceURL:   envOrDefault("LEDGER_SERVICE_URL", "http://localhost:8030"),
		SessionsServiceURL: envOrDefault("SESSIONS_SERVICE_URL", "http://localhost:8100"),
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
