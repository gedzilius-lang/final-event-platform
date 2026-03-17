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

	"niteos.internal/profiles/internal/handler"
	"niteos.internal/profiles/internal/store"
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

	userStore := store.New(db)
	h := handler.New(userStore)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("GET /metrics", metrics.Handler("profiles"))

	// Internal routes (called by auth service)
	mux.HandleFunc("POST /users", h.CreateUser)
	mux.HandleFunc("GET /users", h.ListUsers)
	mux.HandleFunc("GET /users/{user_id}", h.GetUser)
	mux.HandleFunc("GET /users/by-email/{email}", h.GetUserByEmail)
	mux.HandleFunc("GET /users/by-nfc-uid/{nfc_uid}", h.GetUserByNFCUID)
	mux.HandleFunc("PATCH /users/{user_id}/venue", h.PatchUserVenue)
	mux.HandleFunc("POST /users/{user_id}/xp", h.AddXP)

	// NiteTap routes (called by sessions, orders services)
	mux.HandleFunc("GET /nitetaps/{nfc_uid}", h.GetNiteTap)
	mux.HandleFunc("POST /nitetaps/{nfc_uid}/link", h.LinkNiteTap)

	// VenueProfile routes (called by sessions service)
	mux.HandleFunc("GET /venue-profiles/{user_id}/{venue_id}", h.GetVenueProfile)
	mux.HandleFunc("POST /venue-profiles", h.UpsertVenueProfile)

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("profiles service listening", "addr", cfg.ListenAddr)
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
		ListenAddr:  envOrDefault("PROFILES_LISTEN_ADDR", ":8020"),
		DatabaseURL: envOrDefault("DATABASE_URL", "postgres://niteos:devpassword@localhost:5432/niteos?sslmode=disable"),
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
