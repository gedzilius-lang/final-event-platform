package store

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// TokenStore manages JWT and refresh token records in Redis.
// Redis key conventions per AUTH_MODEL.md:
//   - tok:{jti}          — access token revocation record (TTL: 15 min)
//   - ref:{uid}:{hash}   — refresh token record (TTL: 30 days)
//   - dev:{device_id}    — device token status (TTL: 90 days)
//   - rate:login:{ip}    — rate limit counter
//   - rate:login:{email} — rate limit counter
//   - rate:register:{ip} — rate limit counter
//   - rate:pin:{did}     — rate limit counter
type TokenStore struct {
	rdb *redis.Client
}

func NewTokenStore(rdb *redis.Client) *TokenStore {
	return &TokenStore{rdb: rdb}
}

const (
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 30 * 24 * time.Hour
	StaffTokenTTL   = 12 * time.Hour
	DeviceTokenTTL  = 90 * 24 * time.Hour
)

// SetAccessToken records an access token in Redis with the standard TTL.
func (s *TokenStore) SetAccessToken(ctx context.Context, jti, uid string, ttl time.Duration) error {
	return s.rdb.Set(ctx, "tok:"+jti, uid, ttl).Err()
}

// AccessTokenExists returns true if the access token record is still in Redis.
func (s *TokenStore) AccessTokenExists(ctx context.Context, jti string) (bool, error) {
	n, err := s.rdb.Exists(ctx, "tok:"+jti).Result()
	return n > 0, err
}

// DeleteAccessToken removes an access token record (on logout/revocation).
func (s *TokenStore) DeleteAccessToken(ctx context.Context, jti string) error {
	return s.rdb.Del(ctx, "tok:"+jti).Err()
}

// SetRefreshToken stores a refresh token hash in Redis.
func (s *TokenStore) SetRefreshToken(ctx context.Context, uid, tokenHash, sessionID string) error {
	val := fmt.Sprintf("%s:%d", sessionID, time.Now().Unix())
	return s.rdb.Set(ctx, "ref:"+uid+":"+tokenHash, val, RefreshTokenTTL).Err()
}

// GetRefreshToken retrieves the session payload for a refresh token hash.
// Returns ("", ErrNotFound) if the token does not exist or has expired.
func (s *TokenStore) GetRefreshToken(ctx context.Context, uid, tokenHash string) (sessionID string, err error) {
	val, err := s.rdb.Get(ctx, "ref:"+uid+":"+tokenHash).Result()
	if err == redis.Nil {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get refresh token: %w", err)
	}
	// val is "sessionID:issuedAtUnix"
	if len(val) > 0 {
		for i, c := range val {
			if c == ':' {
				return val[:i], nil
			}
		}
		return val, nil
	}
	return "", nil
}

// DeleteRefreshToken removes a single refresh token (logout from one device).
func (s *TokenStore) DeleteRefreshToken(ctx context.Context, uid, tokenHash string) error {
	return s.rdb.Del(ctx, "ref:"+uid+":"+tokenHash).Err()
}

// DeleteAllRefreshTokens removes all refresh tokens for a user (logout everywhere).
func (s *TokenStore) DeleteAllRefreshTokens(ctx context.Context, uid string) error {
	pattern := "ref:" + uid + ":*"
	var cursor uint64
	for {
		keys, cur, err := s.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			s.rdb.Del(ctx, keys...)
		}
		cursor = cur
		if cursor == 0 {
			break
		}
	}
	return nil
}

// SetDeviceStatus sets or updates the device status in Redis.
func (s *TokenStore) SetDeviceStatus(ctx context.Context, deviceID, venueID, deviceRole, status string) error {
	val := fmt.Sprintf("%s:%s:%s", venueID, deviceRole, status)
	return s.rdb.Set(ctx, "dev:"+deviceID, val, DeviceTokenTTL).Err()
}

// GetDeviceStatus retrieves the cached device status from Redis.
// Returns ("", "", "", ErrNotFound) if not cached.
func (s *TokenStore) GetDeviceStatus(ctx context.Context, deviceID string) (venueID, deviceRole, status string, err error) {
	val, err := s.rdb.Get(ctx, "dev:"+deviceID).Result()
	if err == redis.Nil {
		return "", "", "", ErrNotFound
	}
	if err != nil {
		return "", "", "", fmt.Errorf("get device status: %w", err)
	}
	// val is "venueID:deviceRole:status"
	parts := splitN(val, ':', 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("malformed device status: %q", val)
	}
	return parts[0], parts[1], parts[2], nil
}

// CheckRateLimit checks a sliding-window counter. Returns (allowed, current_count, err).
// Window is 1 minute. limit is max requests per window.
func (s *TokenStore) CheckRateLimit(ctx context.Context, key string, limit int) (bool, error) {
	pipe := s.rdb.Pipeline()
	incrCmd := pipe.Incr(ctx, "rate:"+key)
	pipe.Expire(ctx, "rate:"+key, time.Minute)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("rate limit check: %w", err)
	}
	count := incrCmd.Val()
	return count <= int64(limit), nil
}

func splitN(s string, sep rune, n int) []string {
	result := make([]string, 0, n)
	start := 0
	count := 0
	for i, c := range s {
		if c == sep {
			result = append(result, s[start:i])
			start = i + 1
			count++
			if count == n-1 {
				break
			}
		}
	}
	result = append(result, s[start:])
	return result
}
