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

	"niteos.internal/sessions/internal/handler"
	"niteos.internal/sessions/internal/store"
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
	h := handler.New(s, cfg.ProfilesServiceURL, cfg.LedgerServiceURL)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json"); fmt.Fprint(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("GET /metrics", metrics.Handler("sessions"))
	mux.HandleFunc("POST /checkin", h.Checkin)
	mux.HandleFunc("POST /{session_id}/checkout", h.Checkout)
	mux.HandleFunc("GET /{session_id}", h.GetSession)
	mux.HandleFunc("GET /venues/{venue_id}/active", h.ListActive)
	mux.HandleFunc("POST /{session_id}/spend", h.IncrementSpend)
	srv := &http.Server{Addr: cfg.ListenAddr, Handler: mux, ReadTimeout: 10*time.Second, WriteTimeout: 10*time.Second}
	go func() {
		slog.Info("sessions service listening", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed { log.Fatalf("listen: %v", err) }
	}()
	quit := make(chan os.Signal, 1); signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM); <-quit
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second); defer cancel(); _ = srv.Shutdown(ctx)
}

type config struct{ ListenAddr, DatabaseURL, ProfilesServiceURL, LedgerServiceURL string }

func configFromEnv() config {
	return config{
		ListenAddr:         envOrDefault("SESSIONS_LISTEN_ADDR", ":8100"),
		DatabaseURL:        envOrDefault("DATABASE_URL", "postgres://niteos:devpassword@localhost:5432/niteos?sslmode=disable"),
		ProfilesServiceURL: envOrDefault("PROFILES_SERVICE_URL", "http://localhost:8020"),
		LedgerServiceURL:   envOrDefault("LEDGER_SERVICE_URL", "http://localhost:8030"),
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" { return v }; return def
}
