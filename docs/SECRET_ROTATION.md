# SECRET_ROTATION.md

Procedures for rotating every secret in the NiteOS system. Follow exactly. Do not improvise.

---

## JWT RS256 Key Pair

**Location:** `infra/secrets/jwt_private_key.pem` and `jwt_public_key.pem` (never committed to git)

**Rotation trigger:** Annually, or immediately if the private key is compromised or suspected compromised.

**Impact of rotation:** All active access tokens (15-min JWTs) become invalid immediately. All active refresh tokens (stored in Redis) are NOT affected by key rotation — they are opaque strings, not signed JWTs. After rotation, users need to log in again to get a new access token.

### Rotation procedure

```bash
# 1. Generate new key pair
openssl genrsa -out jwt_private_key_NEW.pem 2048
openssl rsa -in jwt_private_key_NEW.pem -pubout -out jwt_public_key_NEW.pem

# 2. Deploy new keys to VPS A (copy to /run/secrets/ on the server)
# The auth service reads the private key; the gateway reads the public key.
# Both are passed as Docker secrets — not environment variables.

# 3. Restart auth service (now issues tokens signed with new key)
docker compose restart auth

# 4. Restart gateway service (now validates tokens with new public key)
docker compose restart gateway

# 5. At this point, all existing access tokens (signed with old key) are rejected by gateway.
# Users will receive 401 and must log in again. This is expected.
# Refresh tokens are unaffected — users who refresh get a new access token signed with the new key.

# 6. Remove old key files from VPS A
rm /run/secrets/jwt_private_key_OLD.pem
rm /run/secrets/jwt_public_key_OLD.pem

# 7. Update infra/secrets/ on the deployment machine with the new key files
mv jwt_private_key_NEW.pem jwt_private_key.pem
mv jwt_public_key_NEW.pem jwt_public_key.pem
```

**Verify:** After restart, attempt login and confirm new access token validates through gateway.

---

## Postgres Password (niteos_app role)

**Location:** `.env` on VPS A (never in git). Passed as environment variable to all service containers.

**Rotation trigger:** Annually, or on staff departure with database access.

### Rotation procedure

```bash
# 1. Generate new password
NEW_PW=$(openssl rand -base64 32)

# 2. Update Postgres role
psql -U postgres -c "ALTER ROLE niteos_app PASSWORD '$NEW_PW';"

# 3. Update .env on VPS A with new DATABASE_URL
# DATABASE_URL=postgres://niteos_app:${NEW_PW}@db:5432/niteos?sslmode=require

# 4. Restart all services
docker compose down && docker compose up -d

# 5. Verify all services connect cleanly (check /healthz on each service)
```

---

## Redis Password

**Location:** `.env` on VPS A. Passed to Redis container and all services that use Redis.

**Rotation trigger:** Annually, or on staff departure.

### Rotation procedure

```bash
# 1. Generate new password
NEW_PW=$(openssl rand -base64 32)

# 2. Update Redis config (redis.conf) and .env
# redis.conf: requirepass <NEW_PW>
# .env: REDIS_URL=redis://:${NEW_PW}@redis:6379

# 3. Restart Redis, then restart all services that use Redis (auth, gateway)
docker compose restart redis auth gateway

# 4. Verify auth token validation still works
```

---

## Stripe Webhook Secret

**Location:** Docker secret on VPS A — `/run/secrets/stripe_webhook_secret`

**Rotation trigger:** When Stripe rotates it (in Stripe dashboard), or on suspicion of compromise.

### Rotation procedure

```bash
# 1. In Stripe Dashboard: generate new webhook signing secret
# 2. Update Docker secret on VPS A
# 3. Restart payments service
docker compose restart payments
```

---

## TWINT API Key

**Location:** Docker secret on VPS A — `/run/secrets/twint_api_key`

**Rotation trigger:** Per PSP/SIX Payment Services policy, or on suspicion of compromise.

### Rotation procedure

Follow PSP provider's key rotation procedure. Update Docker secret on VPS A and restart payments service.

---

## Edge Device Token (per venue)

**Location:** On the Master Tablet at `/etc/niteos-edge/config.toml`

**Rotation trigger:** Device reported lost/stolen, or annually.

### Rotation procedure

```
1. Nitecore HQ: revoke old device token via admin console (devices service marks device as revoked)
2. Redis key dev:{device_id} deleted → old token immediately invalid
3. Re-enroll the Master Tablet:
   POST /devices/enroll  →  new device record (pending)
   POST /devices/{id}/approve  →  new device JWT issued
4. Update config.toml on the Master Tablet with new token
5. Restart edge service on Master Tablet
```

---

## Key Inventory

| Secret | Location (production) | Rotation cadence |
|--------|-----------------------|-----------------|
| JWT private key | `/run/secrets/jwt_private_key` on VPS A | Annual / on compromise |
| JWT public key | `/run/secrets/jwt_public_key` on VPS A | With private key |
| Postgres password | VPS A `.env` (POSTGRES_PASSWORD) | Annual |
| Redis password | VPS A `.env` (REDIS_PASSWORD) | Annual |
| Stripe webhook secret | `/run/secrets/stripe_webhook_secret` | Per Stripe |
| TWINT API key | `/run/secrets/twint_api_key` | Per PSP |
| Edge device tokens | Master Tablet config.toml | Annual / on loss |
| Cloudflare API token (DNS-01 TLS) | VPS A `.env` (CF_DNS_API_TOKEN) | Annual |
