package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"niteos.internal/gateway/internal/proxy"
	"niteos.internal/pkg/metrics"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := configFromEnv()

	// Redis for token validation
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
	})
	defer rdb.Close()
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("redis ping: %v", err)
	}

	// Build reverse proxy handler
	p, err := proxy.New(proxy.Config{
		AuthServiceURL:      cfg.AuthServiceURL,
		ProfilesServiceURL:  cfg.ProfilesServiceURL,
		LedgerServiceURL:    cfg.LedgerServiceURL,
		WalletServiceURL:    cfg.WalletServiceURL,
		PaymentsServiceURL:  cfg.PaymentsServiceURL,
		TicketingServiceURL: cfg.TicketingServiceURL,
		OrdersServiceURL:    cfg.OrdersServiceURL,
		CatalogServiceURL:   cfg.CatalogServiceURL,
		DevicesServiceURL:   cfg.DevicesServiceURL,
		SessionsServiceURL:  cfg.SessionsServiceURL,
		ReportingServiceURL: cfg.ReportingServiceURL,
		SyncServiceURL:      cfg.SyncServiceURL,
		Redis:               rdb,
		JWKSRefreshInterval: 5 * time.Minute,
	})
	if err != nil {
		log.Fatalf("build proxy: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("GET /metrics", metrics.Handler("gateway"))
	// All other routes go through the proxy
	mux.Handle("/", p)

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		slog.Info("gateway service listening", "addr", cfg.ListenAddr)
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
	RedisAddr           string
	RedisPassword       string
	AuthServiceURL      string
	ProfilesServiceURL  string
	LedgerServiceURL    string
	WalletServiceURL    string
	PaymentsServiceURL  string
	TicketingServiceURL string
	OrdersServiceURL    string
	CatalogServiceURL   string
	DevicesServiceURL   string
	SessionsServiceURL  string
	ReportingServiceURL string
	SyncServiceURL      string
}

func configFromEnv() config {
	return config{
		ListenAddr:          envOrDefault("GATEWAY_LISTEN_ADDR", ":8000"),
		RedisAddr:           envOrDefault("REDIS_ADDR", "localhost:6379"),
		RedisPassword:       envOrDefault("REDIS_PASSWORD", "devpassword"),
		AuthServiceURL:      envOrDefault("AUTH_SERVICE_URL", "http://localhost:8010"),
		ProfilesServiceURL:  envOrDefault("PROFILES_SERVICE_URL", "http://localhost:8020"),
		LedgerServiceURL:    envOrDefault("LEDGER_SERVICE_URL", "http://localhost:8030"),
		WalletServiceURL:    envOrDefault("WALLET_SERVICE_URL", "http://localhost:8040"),
		PaymentsServiceURL:  envOrDefault("PAYMENTS_SERVICE_URL", "http://localhost:8060"),
		TicketingServiceURL: envOrDefault("TICKETING_SERVICE_URL", "http://localhost:8050"),
		OrdersServiceURL:    envOrDefault("ORDERS_SERVICE_URL", "http://localhost:8090"),
		CatalogServiceURL:   envOrDefault("CATALOG_SERVICE_URL", "http://localhost:8080"),
		DevicesServiceURL:   envOrDefault("DEVICES_SERVICE_URL", "http://localhost:8070"),
		SessionsServiceURL:  envOrDefault("SESSIONS_SERVICE_URL", "http://localhost:8100"),
		ReportingServiceURL: envOrDefault("REPORTING_SERVICE_URL", "http://localhost:8120"),
		SyncServiceURL:      envOrDefault("SYNC_SERVICE_URL", "http://localhost:8110"),
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
