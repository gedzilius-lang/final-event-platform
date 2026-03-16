package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"niteos.internal/wallet/internal/handler"
	"niteos.internal/pkg/metrics"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	cfg := configFromEnv()
	h := handler.New(cfg.LedgerServiceURL)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json"); fmt.Fprint(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("GET /metrics", metrics.Handler("wallet"))
	mux.HandleFunc("GET /{user_id}", h.GetWallet)
	mux.HandleFunc("GET /{user_id}/history", h.GetHistory)
	srv := &http.Server{Addr: cfg.ListenAddr, Handler: mux, ReadTimeout: 10*time.Second, WriteTimeout: 10*time.Second}
	go func() {
		slog.Info("wallet service listening", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed { log.Fatalf("listen: %v", err) }
	}()
	quit := make(chan os.Signal, 1); signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM); <-quit
	log.Println("shutting down")
}

type config struct{ ListenAddr, LedgerServiceURL string }

func configFromEnv() config {
	return config{
		ListenAddr:       envOrDefault("WALLET_LISTEN_ADDR", ":8040"),
		LedgerServiceURL: envOrDefault("LEDGER_SERVICE_URL", "http://localhost:8030"),
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" { return v }; return def
}
