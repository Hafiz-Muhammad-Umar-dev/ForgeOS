# Authentication — ForgeOS

## Architecture

```
┌──────────┐     ┌──────────────┐     ┌──────────┐
│ Frontend │────▶│ Auth Service │────▶│ Gateway  │
│ (5173)   │     │ (8081)       │     │ (8080)   │
└──────────┘     └──────────────┘     └──────────┘
                       │
                       ▼
                  ┌──────────┐
                  │ JWT      │
                  │ HMAC-    │
                  │ SHA256   │
                  └──────────┘
```

The authentication service (`apps/auth/`) issues HS256 JWTs compatible with the API Gateway's `JWTAdapter` (`apps/gateway/auth/jwt.go`). The Gateway verifies tokens using the same shared secret; it does not call the Auth Service.

## JWT Flow

1. User sends `POST /auth/login` with username/password.
2. Auth Service validates credentials against the hardcoded dev account.
3. On success, Auth Service signs an HS256 JWT using `DEVOS_JWT_SECRET`.
4. Frontend stores the token in `localStorage` under key `forge_token`.
5. Every API request includes `Authorization: Bearer <token>`.
6. Gateway verifies the token's HMAC-SHA256 signature and extracts claims.
7. On 401/403, frontend clears the token and redirects to `/login`.

## API Endpoints

### POST /auth/login

**Request:**
```json
{
  "username": "admin",
  "password": "admin123"
}
```

**Response (200):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiJ9...",
  "token_type": "Bearer",
  "expires_in": 86400,
  "user": {
    "id": "dev-admin",
    "name": "Administrator",
    "role": "admin"
  }
}
```

### GET /auth/me

**Headers:**
```
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "id": "dev-admin",
  "name": "Administrator",
  "role": "admin"
}
```

### POST /auth/refresh

Re-issues a token for the dev user.

**Response (200):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiJ9...",
  "token_type": "Bearer",
  "expires_in": 86400,
  "user": { ... }
}
```

## Token Lifetime

- **Expiry:** 24 hours (86400 seconds)
- **Claim:** `exp` in JWT payload
- **Automatic refresh:** Frontend can call `POST /auth/refresh` to get a new token
- **On expiry:** Gateway returns 401; frontend redirects to `/login`

## JWT Structure

| Component | Format | Example |
|-----------|--------|---------|
| Header | `{"alg":"HS256"}` | Base64URL-encoded |
| Payload | `{"sub":"dev-admin","org_id":"org-1","role":"admin","exp":...}` | Base64URL-encoded |
| Signature | HMAC-SHA256 of `header.payload` | Base64URL-encoded |

Claims:

| Claim | Value | Description |
|-------|-------|-------------|
| `sub` | User ID | e.g. `dev-admin` |
| `org_id` | Organization | e.g. `org-1` |
| `role` | User role | e.g. `admin` |
| `scopes` | Permission scopes | `["admin"]` |
| `exp` | Expiration timestamp | Unix seconds |
| `iat` | Issued at timestamp | Unix seconds |

## Development Credentials

| Field | Value |
|-------|-------|
| Username | `admin` |
| Password | `admin123` |
| Role | `admin` |
| User ID | `dev-admin` |

These are hardcoded in `apps/auth/authservice/service.go` for development only.

## Running the Auth Service

```bash
DEVOS_JWT_SECRET=dev-secret-k8s-switch PORT=8081 go run ./apps/auth/
```

The auth service starts on port 8081 by default. The frontend dev server proxies `/auth/*` to `http://localhost:8081`.

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DEVOS_JWT_SECRET` | Yes | — | HMAC-SHA256 secret for JWT signing (must match Gateway) |
| `PORT` | No | `8081` | HTTP listen port |

## Security Notes

1. **Development only.** The hardcoded admin account is not suitable for production.
2. **Shared secret.** `DEVOS_JWT_SECRET` must be identical between Auth Service and Gateway.
3. **No HTTPS.** For development; production requires TLS termination at the load balancer.
4. **CORS.** Only `http://localhost:5173` is allowed by default.
5. **Token storage.** The frontend stores the JWT in `localStorage`. For production, consider httpOnly cookies.
