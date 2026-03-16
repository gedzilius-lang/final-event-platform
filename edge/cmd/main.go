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

	"niteos.internal/edge/internal/catalog"
	edgedb "niteos.internal/edge/internal/db"
	"niteos.internal/edge/internal/handler"
	"niteos.internal/edge/internal/ledger"
	edgesync "niteos.internal/edge/internal/sync"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	cfg := configFromEnv()

	db, err := edgedb.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	ledgerStore := ledger.New(db)
	catalogCache := catalog.New(db, cfg.CatalogServiceURL, cfg.VenueID)
	h := handler.New(ledgerStore, catalogCache, cfg.VenueID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start background catalog sync (every 15 minutes)
	catalogCache.StartSyncLoop(ctx, 15*time.Minute)

	// Start background event sync agent (every 30 seconds)
	syncAgent := edgesync.NewAgent(ledgerStore, cfg.SyncServiceURL, cfg.DeviceID, cfg.VenueID)
	go syncAgent.Run(ctx, 30*time.Second)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("GET /balance/{user_id}", h.GetBalance)
	mux.HandleFunc("POST /orders", h.CreateOrder)
	mux.HandleFunc("GET /catalog/items", h.ListItems)
	mux.HandleFunc("GET /sync/status", h.SyncStatus)

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("edge service listening", "addr", cfg.ListenAddr, "venue_id", cfg.VenueID)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancel()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutCancel()
	_ = srv.Shutdown(shutCtx)
}

type config struct {
	ListenAddr        string
	DBPath            string
	VenueID           string
	DeviceID          string
	CatalogServiceURL string
	SyncServiceURL    string
}

func configFromEnv() config {
	return config{
		ListenAddr:        envOrDefault("EDGE_LISTEN_ADDR", ":9000"),
		DBPath:            envOrDefault("EDGE_DB_PATH", "./edge.db"),
		VenueID:           mustEnv("EDGE_VENUE_ID"),
		DeviceID:          mustEnv("EDGE_DEVICE_ID"),
		CatalogServiceURL: envOrDefault("CATALOG_SERVICE_URL", "http://localhost:8080"),
		SyncServiceURL:    envOrDefault("SYNC_SERVICE_URL", "http://localhost:8110"),
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}
