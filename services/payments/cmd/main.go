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

	"niteos.internal/payments/internal/domain"
	"niteos.internal/payments/internal/handler"
	"niteos.internal/payments/internal/provider"
	"niteos.internal/payments/internal/store"
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
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	providers := map[string]provider.Provider{
		domain.ProviderMock: provider.NewMock(),
	}
	if cfg.StripeAPIKey != "" {
		providers[domain.ProviderStripe] = provider.NewStripe(cfg.StripeAPIKey, cfg.StripeWebhookSecret)
	}

	s := store.New(db)
	h := handler.New(s, providers, cfg.LedgerServiceURL)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("GET /metrics", metrics.Handler("payments"))
	mux.HandleFunc("POST /topup/intent", h.CreateIntent)
	mux.HandleFunc("GET /topup/{topup_id}", h.GetTopup)
	mux.HandleFunc("POST /topup/webhook/stripe", h.WebhookStripe)
	mux.HandleFunc("POST /topup/webhook/mock", h.WebhookMock)

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("payments service listening", "addr", cfg.ListenAddr)
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
	ListenAddr          string
	DatabaseURL         string
	LedgerServiceURL    string
	StripeAPIKey        string
	StripeWebhookSecret string
}

func configFromEnv() config {
	return config{
		ListenAddr:          envOrDefault("PAYMENTS_LISTEN_ADDR", ":8060"),
		DatabaseURL:         envOrDefault("DATABASE_URL", "postgres://niteos:devpassword@localhost:5432/niteos?sslmode=disable"),
		LedgerServiceURL:    envOrDefault("LEDGER_SERVICE_URL", "http://localhost:8030"),
		StripeAPIKey:        os.Getenv("STRIPE_API_KEY"),
		StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
