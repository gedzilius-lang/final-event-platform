// Package handler implements the auth service HTTP handlers.
// Endpoints per AUTH_MODEL.md:
//   POST /register  — email + password registration
//   POST /login     — email + password login (rate-limited)
//   POST /refresh   — access token refresh via refresh token
//   POST /logout    — revoke tokens
//   POST /pin       — venue PIN login (staff terminals)
//   GET  /jwks      — RS256 public key endpoint
package handler

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"niteos.internal/pkg/httputil"
)

const (
	bcryptCost         = 12
	accessTokenTTL     = 15 * time.Minute
	staffAccessTTL     = 12 * time.Hour
	refreshTokenBytes  = 32
	loginRateLimitIP   = 5   // per minute
	loginRateLimitEmail = 10  // per minute
	registerRateLimit  = 3   // per minute
	pinRateLimit       = 10  // per minute per device
)

// Handler holds dependencies for all auth endpoints.
type Handler struct {
	users              userStorer
	tokens             tokenStorer
	privateKey         *rsa.PrivateKey
	profilesServiceURL string
}

type userStorer interface {
	GetVenueStaffPin(ctx context.Context, venueID string) (string, error)
	GetDeviceVenueRole(ctx context.Context, deviceID string) (venueID, deviceRole string, err error)
}

type tokenStorer interface {
	SetAccessToken(ctx context.Context, jti, uid string, ttl time.Duration) error
	AccessTokenExists(ctx context.Context, jti string) (bool, error)
	DeleteAccessToken(ctx context.Context, jti string) error
	SetRefreshToken(ctx context.Context, uid, tokenHash, sessionID string) error
	GetRefreshToken(ctx context.Context, uid, tokenHash string) (string, error)
	DeleteRefreshToken(ctx context.Context, uid, tokenHash string) error
	DeleteAllRefreshTokens(ctx context.Context, uid string) error
	SetDeviceStatus(ctx context.Context, deviceID, venueID, deviceRole, status string) error
	CheckRateLimit(ctx context.Context, key string, limit int) (bool, error)
}

// New creates a new Handler.
func New(users userStorer, tokens tokenStorer, privateKey *rsa.PrivateKey, profilesServiceURL string) *Handler {
	return &Handler{
		users:              users,
		tokens:             tokens,
		privateKey:         privateKey,
		profilesServiceURL: strings.TrimRight(profilesServiceURL, "/"),
	}
}

// ── Register ─────────────────────────────────────────────────────────────────

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type authResponse struct {
	UserID       string `json:"user_id"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // seconds
	DisplayName  string `json:"display_name,omitempty"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	// Rate limit by IP
	ip := clientIP(r)
	ok, err := h.tokens.CheckRateLimit(r.Context(), "register:"+ip, registerRateLimit)
	if err != nil {
		slog.Error("rate limit check", "err", err)
	}
	if !ok {
		httputil.RespondError(w, http.StatusTooManyRequests, "too many registration attempts")
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		httputil.RespondError(w, http.StatusBadRequest, "email and password are required")
		return
	}
	if len(req.Password) < 8 {
		httputil.RespondError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Create user via profiles service
	userID, err := h.createProfilesUser(r.Context(), req.Email, string(hash), req.DisplayName)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "duplicate") {
			httputil.RespondError(w, http.StatusConflict, "email already registered")
			return
		}
		slog.Error("create profiles user", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Issue tokens
	resp, err := h.issueTokens(r.Context(), userID, "guest", "", "")
	if err != nil {
		slog.Error("issue tokens", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	httputil.Respond(w, http.StatusCreated, resp)
}

// ── Login ─────────────────────────────────────────────────────────────────────

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)

	// Rate limit by IP and email
	okIP, _ := h.tokens.CheckRateLimit(r.Context(), "login:"+ip, loginRateLimitIP)
	if !okIP {
		httputil.RespondError(w, http.StatusTooManyRequests, "too many attempts")
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Rate limit by email (checked after decode so we have the email)
	if req.Email != "" {
		okEmail, _ := h.tokens.CheckRateLimit(r.Context(), "login:email:"+req.Email, loginRateLimitEmail)
		if !okEmail {
			httputil.RespondError(w, http.StatusTooManyRequests, "too many attempts")
			return
		}
	}

	// Fetch user from profiles service
	user, err := h.fetchProfilesUser(r.Context(), req.Email)
	if err != nil {
		// Do not reveal which field is wrong
		httputil.RespondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Compare password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		httputil.RespondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	resp, err := h.issueTokens(r.Context(), user.UserID, user.Role, user.VenueID, "")
	if err != nil {
		slog.Error("issue tokens on login", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	resp.DisplayName = user.DisplayName

	httputil.Respond(w, http.StatusOK, resp)
}

// ── Refresh ───────────────────────────────────────────────────────────────────

type refreshRequest struct {
	UserID       string `json:"user_id"`
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tokenHash := hashToken(req.RefreshToken)
	sessionID, err := h.tokens.GetRefreshToken(r.Context(), req.UserID, tokenHash)
	if err != nil {
		httputil.RespondError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	// Fetch user role from profiles
	user, err := h.fetchProfilesUserByID(r.Context(), req.UserID)
	if err != nil {
		httputil.RespondError(w, http.StatusUnauthorized, "user not found")
		return
	}

	// Issue new access token (same refresh token, rotate if desired)
	jti := newJTI()
	accessToken, err := h.mintToken(req.UserID, user.Role, "", "", sessionID, jti, accessTokenTTL)
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := h.tokens.SetAccessToken(r.Context(), jti, req.UserID, accessTokenTTL); err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	httputil.Respond(w, http.StatusOK, map[string]any{
		"user_id":      req.UserID,
		"access_token": accessToken,
		"expires_in":   int(accessTokenTTL.Seconds()),
	})
}

// ── Logout ────────────────────────────────────────────────────────────────────

type logoutRequest struct {
	UserID       string `json:"user_id"`
	JTI          string `json:"jti"`
	RefreshToken string `json:"refresh_token"`
	AllDevices   bool   `json:"all_devices"`
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.JTI != "" {
		_ = h.tokens.DeleteAccessToken(r.Context(), req.JTI)
	}

	if req.AllDevices && req.UserID != "" {
		_ = h.tokens.DeleteAllRefreshTokens(r.Context(), req.UserID)
	} else if req.RefreshToken != "" && req.UserID != "" {
		tokenHash := hashToken(req.RefreshToken)
		_ = h.tokens.DeleteRefreshToken(r.Context(), req.UserID, tokenHash)
	}

	httputil.Respond(w, http.StatusOK, map[string]string{"status": "logged out"})
}

// ── PIN Login ─────────────────────────────────────────────────────────────────

type pinLoginRequest struct {
	VenueID  string `json:"venue_id"`
	VenuePIN string `json:"venue_pin"`
	DeviceID string `json:"device_id"`
}

func (h *Handler) PINLogin(w http.ResponseWriter, r *http.Request) {
	// Validate device token from Authorization header
	deviceID, err := h.validateDeviceToken(r)
	if err != nil {
		httputil.RespondError(w, http.StatusUnauthorized, "invalid device token")
		return
	}

	var req pinLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Rate limit by device
	if ok, _ := h.tokens.CheckRateLimit(r.Context(), "pin:"+deviceID, pinRateLimit); !ok {
		httputil.RespondError(w, http.StatusTooManyRequests, "too many PIN attempts")
		return
	}

	// Get device's venue and role from DB
	venueID, deviceRole, err := h.users.GetDeviceVenueRole(r.Context(), deviceID)
	if err != nil {
		httputil.RespondError(w, http.StatusUnauthorized, "device not enrolled")
		return
	}
	if venueID != req.VenueID {
		httputil.RespondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Validate venue PIN
	staffPinHash, err := h.users.GetVenueStaffPin(r.Context(), venueID)
	if err != nil {
		httputil.RespondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(staffPinHash), []byte(req.VenuePIN)); err != nil {
		httputil.RespondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Derive staff role from device role
	staffRole := "bartender"
	if deviceRole == "terminal" {
		staffRole = "door_staff"
	}

	// Issue staff access token (12 hour shift token, no refresh)
	jti := newJTI()
	// Use device_id as synthetic user identifier for staff tokens
	staffUserID := "staff:" + deviceID
	accessToken, err := h.mintToken(staffUserID, staffRole, venueID, deviceID, "", jti, staffAccessTTL)
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := h.tokens.SetAccessToken(r.Context(), jti, staffUserID, staffAccessTTL); err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	httputil.Respond(w, http.StatusOK, map[string]any{
		"access_token": accessToken,
		"role":         staffRole,
		"venue_id":     venueID,
		"device_id":    deviceID,
		"expires_in":   int(staffAccessTTL.Seconds()),
	})
}

// ── JWKS ──────────────────────────────────────────────────────────────────────

func (h *Handler) JWKS(w http.ResponseWriter, r *http.Request) {
	pub := &h.privateKey.PublicKey

	// Encode to PEM for display
	pubBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "key encoding error")
		return
	}
	pemBlock := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})

	// Build JWK (RS256, public key components)
	eBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(eBytes, uint32(pub.E))
	// Trim leading zeros
	for len(eBytes) > 1 && eBytes[0] == 0 {
		eBytes = eBytes[1:]
	}

	jwk := map[string]any{
		"kty": "RSA",
		"use": "sig",
		"alg": "RS256",
		"kid": "niteos-auth-key-1",
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(eBytes),
	}

	_ = pemBlock // PEM available if needed

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"keys": []any{jwk},
	})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// mintToken creates a signed RS256 JWT per AUTH_MODEL.md.
func (h *Handler) mintToken(uid, role, venueID, deviceID, sessionID, jti string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"jti": jti,
		"uid": uid,
		"role": role,
		"iat": now.Unix(),
		"exp": now.Add(ttl).Unix(),
	}
	if venueID != "" {
		claims["venue_id"] = venueID
	}
	if deviceID != "" {
		claims["device_id"] = deviceID
	}
	if sessionID != "" {
		claims["session_id"] = sessionID
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(h.privateKey)
}

// issueTokens creates both access and refresh tokens and stores them in Redis.
func (h *Handler) issueTokens(ctx context.Context, uid, role, venueID, deviceID string) (*authResponse, error) {
	jti := newJTI()
	sessionID := newJTI() // session scoped to this login

	accessToken, err := h.mintToken(uid, role, venueID, deviceID, sessionID, jti, accessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("mint access token: %w", err)
	}
	if err := h.tokens.SetAccessToken(ctx, jti, uid, accessTokenTTL); err != nil {
		return nil, fmt.Errorf("store access token: %w", err)
	}

	// Generate refresh token
	rawRefresh := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(rawRefresh); err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	refreshToken := base64.RawURLEncoding.EncodeToString(rawRefresh)
	tokenHash := hashToken(refreshToken)
	if err := h.tokens.SetRefreshToken(ctx, uid, tokenHash, sessionID); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &authResponse{
		UserID:       uid,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(accessTokenTTL.Seconds()),
	}, nil
}

// validateDeviceToken checks the Authorization: Bearer header contains a valid device JWT.
// Returns the device_id claim.
func (h *Handler) validateDeviceToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", fmt.Errorf("missing bearer token")
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

	// Parse without validation first to get claims (gateway already validates for normal requests;
	// this is the PIN endpoint which is unauthenticated at gateway level)
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return &h.privateKey.PublicKey, nil
	})
	if err != nil {
		return "", fmt.Errorf("parse device token: %w", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid device token claims")
	}

	deviceID, _ := claims["device_id"].(string)
	if deviceID == "" {
		return "", fmt.Errorf("device_id not in token")
	}

	return deviceID, nil
}

// createProfilesUser calls the profiles service to create a user.
func (h *Handler) createProfilesUser(ctx context.Context, email, passwordHash, displayName string) (string, error) {
	body := map[string]string{
		"email":         email,
		"password_hash": passwordHash,
		"display_name":  displayName,
	}
	data, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.profilesServiceURL+"/users", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Service", "auth")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("profiles service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		return "", fmt.Errorf("email already exists")
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("profiles service error %d: %s", resp.StatusCode, body)
	}

	var result struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode profiles response: %w", err)
	}
	return result.UserID, nil
}

// fetchProfilesUser calls profiles service to get user by email (for login).
func (h *Handler) fetchProfilesUser(ctx context.Context, email string) (*profilesUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		h.profilesServiceURL+"/users/by-email/"+email, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Internal-Service", "auth")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("profiles service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("user not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("profiles service error: %d", resp.StatusCode)
	}

	var u profilesUser
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, err
	}
	return &u, nil
}

// fetchProfilesUserByID fetches a user record by ID (for refresh token).
func (h *Handler) fetchProfilesUserByID(ctx context.Context, userID string) (*profilesUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		h.profilesServiceURL+"/users/"+userID, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Internal-Service", "auth")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user not found: %d", resp.StatusCode)
	}

	var u profilesUser
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, err
	}
	return &u, nil
}

type profilesUser struct {
	UserID       string `json:"user_id"`
	Email        string `json:"email"`
	PasswordHash string `json:"password_hash"`
	Role         string `json:"role"`
	DisplayName  string `json:"display_name"`
	VenueID      string `json:"venue_id"`
}

// newJTI generates a random JWT ID (jti claim).
func newJTI() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("tok_%x", b)
}

// hashToken SHA256-hashes a token for storage.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// clientIP extracts the real client IP from the request.
func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.SplitN(fwd, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	if real := r.Header.Get("X-Real-Ip"); real != "" {
		return real
	}
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		return host[:idx]
	}
	return host
}

