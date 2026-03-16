// Package proxy implements the NiteOS API gateway reverse proxy.
// Every request is authenticated before being forwarded to the upstream service.
// Unauthenticated routes (/auth/*) bypass token validation.
//
// Authentication flow per AUTH_MODEL.md:
//  1. Extract Bearer token from Authorization header
//  2. Verify RS256 signature using cached public key
//  3. Check token is not expired
//  4. Check Redis tok:{jti} exists (revocation check)
//  5. On success: inject X-User-Id, X-User-Role, X-Venue-Id, X-Device-Id headers
//  6. Forward to upstream service
package proxy

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"niteos.internal/pkg/jwtutil"
)

// Config holds all upstream service URLs and dependencies.
type Config struct {
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
	Redis               *redis.Client
	JWKSRefreshInterval time.Duration
}

// Proxy is the gateway reverse proxy with authentication middleware.
type Proxy struct {
	cfg     Config
	routes  map[string]*httputil.ReverseProxy
	rdb     *redis.Client
	pubKey  *rsa.PublicKey
	pubKeyMu sync.RWMutex
}

// New creates and initialises the gateway proxy.
// Fetches the JWKS public key on startup.
func New(cfg Config) (*Proxy, error) {
	p := &Proxy{
		cfg:    cfg,
		rdb:    cfg.Redis,
		routes: buildRoutes(cfg),
	}

	// Fetch initial JWKS
	if err := p.refreshJWKS(); err != nil {
		return nil, fmt.Errorf("fetch initial JWKS: %w", err)
	}

	// Background JWKS refresh
	go func() {
		ticker := time.NewTicker(cfg.JWKSRefreshInterval)
		defer ticker.Stop()
		for range ticker.C {
			if err := p.refreshJWKS(); err != nil {
				slog.Error("JWKS refresh failed", "err", err)
			}
		}
	}()

	return p, nil
}

func buildRoutes(cfg Config) map[string]*httputil.ReverseProxy {
	routes := map[string]*httputil.ReverseProxy{}
	add := func(prefix, target string) {
		u, err := url.Parse(target)
		if err != nil {
			panic(fmt.Sprintf("invalid upstream URL %q: %v", target, err))
		}
		rp := &httputil.ReverseProxy{
			// Strip the gateway prefix before forwarding to upstream.
			// e.g. /catalog/venues → /venues at the catalog service.
			// Services register routes without their gateway prefix.
			Director: func(req *http.Request) {
				req.URL.Scheme = u.Scheme
				req.URL.Host = u.Host
				stripped := strings.TrimPrefix(req.URL.Path, prefix)
				if stripped == "" {
					stripped = "/"
				}
				req.URL.Path = stripped
				req.Header.Set("X-Forwarded-Host", req.Host)
				req.Header.Del("X-Forwarded-For") // gateway sets this itself if needed
			},
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				slog.Error("upstream error", "path", r.URL.Path, "err", err)
				http.Error(w, `{"error":"upstream unavailable"}`, http.StatusBadGateway)
			},
		}
		routes[prefix] = rp
	}

	add("/auth", cfg.AuthServiceURL)
	add("/profiles", cfg.ProfilesServiceURL)
	add("/ledger", cfg.LedgerServiceURL)
	add("/wallet", cfg.WalletServiceURL)
	add("/payments", cfg.PaymentsServiceURL)
	add("/ticketing", cfg.TicketingServiceURL)
	add("/orders", cfg.OrdersServiceURL)
	add("/catalog", cfg.CatalogServiceURL)
	add("/devices", cfg.DevicesServiceURL)
	add("/sessions", cfg.SessionsServiceURL)
	add("/reporting", cfg.ReportingServiceURL)
	add("/sync", cfg.SyncServiceURL)

	return routes
}

// ServeHTTP is the main entry point for all gateway traffic.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Strip /api prefix if present (Traefik strips, but defensive)
	path = strings.TrimPrefix(path, "/api")

	// Identify upstream by path prefix
	prefix := extractPrefix(path)
	upstream, ok := p.routes[prefix]
	if !ok {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	// Auth routes (/auth/*) bypass token validation — they ARE the auth endpoints
	// Webhook endpoints bypass auth too (they're validated by signature internally)
	if prefix == "/auth" || isWebhookPath(path) {
		upstream.ServeHTTP(w, r)
		return
	}

	// Token validation for all other routes
	claims, err := p.validateToken(r)
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Inject identity headers for upstream services
	r = r.Clone(r.Context())
	r.Header.Set("X-User-Id", claims.UID)
	r.Header.Set("X-User-Role", claims.Role)
	if claims.VenueID != "" {
		r.Header.Set("X-Venue-Id", claims.VenueID)
	}
	if claims.DeviceID != "" {
		r.Header.Set("X-Device-Id", claims.DeviceID)
	}
	if claims.SessionID != "" {
		r.Header.Set("X-Session-Id", claims.SessionID)
	}
	// Internal routing marker
	r.Header.Set("X-Internal-Service", "gateway")

	// Strip external Authorization header from forwarded request
	// (upstream services must not trust raw JWTs — they trust the X-User-* headers)
	r.Header.Del("Authorization")

	upstream.ServeHTTP(w, r)
}

// validateToken verifies the JWT and the Redis revocation record.
func (p *Proxy) validateToken(r *http.Request) (*jwtutil.Claims, error) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, fmt.Errorf("missing bearer token")
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

	p.pubKeyMu.RLock()
	pubKey := p.pubKey
	p.pubKeyMu.RUnlock()

	if pubKey == nil {
		return nil, fmt.Errorf("public key not loaded")
	}

	claims, err := jwtutil.Parse(tokenStr, pubKey)
	if err != nil {
		return nil, fmt.Errorf("token parse: %w", err)
	}

	if jwtutil.IsExpired(claims) {
		return nil, fmt.Errorf("token expired")
	}

	// Redis revocation check
	jti := claims.ID // jwt.RegisteredClaims.ID = jti
	if jti == "" {
		return nil, fmt.Errorf("token missing jti")
	}
	exists, err := p.rdb.Exists(r.Context(), "tok:"+jti).Result()
	if err != nil {
		// Redis unavailable — fail open? No: fail closed per architecture.
		return nil, fmt.Errorf("redis check failed: %w", err)
	}
	if exists == 0 {
		return nil, fmt.Errorf("token revoked or not found in redis")
	}

	return claims, nil
}

// refreshJWKS fetches the RS256 public key from the auth service's /jwks endpoint.
func (p *Proxy) refreshJWKS() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.cfg.AuthServiceURL+"/jwks", nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint returned %d", resp.StatusCode)
	}

	var jwks struct {
		Keys []struct {
			Kty string `json:"kty"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("decode JWKS: %w", err)
	}

	if len(jwks.Keys) == 0 {
		return fmt.Errorf("JWKS contains no keys")
	}

	key := jwks.Keys[0]
	if key.Kty != "RSA" {
		return fmt.Errorf("JWKS key type is not RSA: %s", key.Kty)
	}

	nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return fmt.Errorf("decode JWKS N: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return fmt.Errorf("decode JWKS E: %w", err)
	}

	e := 0
	for _, b := range eBytes {
		e = e<<8 | int(b)
	}

	pub := &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: e,
	}

	p.pubKeyMu.Lock()
	p.pubKey = pub
	p.pubKeyMu.Unlock()

	slog.Info("JWKS refreshed successfully")
	return nil
}

// extractPrefix returns the first path segment (e.g. "/auth" from "/auth/login").
func extractPrefix(path string) string {
	if path == "/" || path == "" {
		return "/"
	}
	// Remove leading slash, find next slash
	trimmed := strings.TrimPrefix(path, "/")
	if idx := strings.Index(trimmed, "/"); idx != -1 {
		return "/" + trimmed[:idx]
	}
	return "/" + trimmed
}

// isWebhookPath returns true for payment provider webhook endpoints.
// These are validated by provider signature internally, not by JWT.
func isWebhookPath(path string) bool {
	return strings.HasPrefix(path, "/payments/topup/webhook/")
}
