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
	"github.com/redis/go-redis/v9"

	"niteos.internal/auth/internal/handler"
	"niteos.internal/auth/internal/store"
	"niteos.internal/pkg/metrics"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := configFromEnv()

	// Postgres
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

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
	})
	defer rdb.Close()
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("redis ping: %v", err)
	}

	// Load RS256 private key
	privKey, err := store.LoadPrivateKey(cfg.JWTPrivateKeyPath)
	if err != nil {
		log.Fatalf("load jwt private key: %v", err)
	}

	// Stores and handler
	userStore := store.NewUserStore(db)
	tokenStore := store.NewTokenStore(rdb)
	h := handler.New(userStore, tokenStore, privKey, cfg.ProfilesServiceURL)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("GET /metrics", metrics.Handler("auth"))
	mux.HandleFunc("POST /register", h.Register)
	mux.HandleFunc("POST /login", h.Login)
	mux.HandleFunc("POST /refresh", h.Refresh)
	mux.HandleFunc("POST /logout", h.Logout)
	mux.HandleFunc("POST /pin", h.PINLogin)
	mux.HandleFunc("GET /jwks", h.JWKS)

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("auth service listening", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down auth service")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

type config struct {
	ListenAddr         string
	DatabaseURL        string
	RedisAddr          string
	RedisPassword      string
	JWTPrivateKeyPath  string
	ProfilesServiceURL string
}

func configFromEnv() config {
	return config{
		ListenAddr:         envOrDefault("AUTH_LISTEN_ADDR", ":8010"),
		DatabaseURL:        envOrDefault("DATABASE_URL", "postgres://niteos:devpassword@localhost:5432/niteos?sslmode=disable"),
		RedisAddr:          envOrDefault("REDIS_ADDR", "localhost:6379"),
		RedisPassword:      envOrDefault("REDIS_PASSWORD", "devpassword"),
		JWTPrivateKeyPath:  envOrDefault("JWT_PRIVATE_KEY_PATH", "infra/secrets/jwt_private_key.pem"),
		ProfilesServiceURL: envOrDefault("PROFILES_SERVICE_URL", "http://localhost:8020"),
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
