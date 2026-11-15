# PiMed HTTP API — v0 (base + authentication)

Shared contract between the Go backend (`backend/`) and the SvelteKit SPA (`frontend/`).
All endpoints live under `/api`. Bodies are JSON, field names are **camelCase**.

## Conventions

- Errors: non-2xx responses carry `{"error": "<human readable message>"}`.
- Every non-GET request **must** send `Content-Type: application/json` (body may be `{}`);
  otherwise the server answers `415`. Together with the `SameSite=Strict` cookie this is the
  CSRF defence.
- Request bodies are limited to 1 MiB (`413`); unknown fields are rejected (`400`). A JSON
  request body must arrive within 10 s, otherwise `408`.
- A known `/api` path called with the wrong method answers
  `405 {"error":"method not allowed"}` with an `Allow` header (this check comes before the
  `415` Content-Type check). Unknown `/api` paths answer `404`.
- `503 {"error":"server busy, try again"}` (with `Retry-After`) means the server is
  temporarily overloaded with password hashing (login, setup, user creation); retry shortly.
- Security headers: every response carries `X-Content-Type-Options: nosniff`,
  `X-Frame-Options: DENY`, `Referrer-Policy: same-origin`. API responses add
  `Cache-Control: no-store` and `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'`.
  SPA responses carry `Content-Security-Policy: default-src 'self'; script-src 'self' 'sha256-…';
  style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; media-src 'self' blob:;
  connect-src 'self'; font-src 'self'; object-src 'none'; base-uri 'none'; form-action 'self';
  frame-ancestors 'none'`, where the `sha256-…` hashes cover the inline bootstrap script of the
  built `index.html` (computed by the server at startup; SvelteKit also emits the same policy as a
  `<meta>` tag, see `kit.csp` in `frontend/svelte.config.js`).
- Auth is a session cookie named `pimed_session` (`HttpOnly`, `SameSite=Strict`, `Path=/`,
  `Secure` unless `PIMED_COOKIE_SECURE=false`). The browser handles it; the SPA never reads it.

### User object

```json
{ "id": 1, "username": "alice", "role": "admin", "createdAt": "2026-11-14T19:40:00Z" }
```

- `role` is `"admin"` or `"user"`.
- `username`: 3–32 chars, `[a-zA-Z0-9_.-]`, unique **case-insensitively** (stored lowercased).
- `password`: 8–128 characters (any characters). Never returned.

## First-run setup

The very first account is created freely (no auth) and becomes **admin**. Once any user
exists, setup is closed forever; further accounts are created by an admin.

| Method | Path | Auth | Request | Success | Errors |
|---|---|---|---|---|---|
| GET | `/api/setup/status` | none | – | `200 {"needsSetup": true}` | – |
| POST | `/api/setup` | none | `{"username","password"}` | `201 {"user": User}` + session cookie (logged in) | `400` validation, `409` setup already completed, `503` busy |

Setup must be race-safe: two concurrent requests can never both create a first admin. Once
setup is closed, `POST /api/setup` always answers `409` (checked before input validation).

## Authentication

| Method | Path | Auth | Request | Success | Errors |
|---|---|---|---|---|---|
| POST | `/api/auth/login` | none | `{"username","password"}` | `200 {"user": User}` + session cookie | `400` malformed, `401 {"error":"invalid username or password"}` (same message for unknown user and wrong password), `429` too many attempts (with `Retry-After` seconds), `503` busy |
| POST | `/api/auth/logout` | session (optional) | `{}` | `204`, cookie cleared, session deleted server-side | – |
| GET | `/api/auth/me` | session | – | `200 {"user": User}` | `401` |

Login rate limit: at most 10 failed logins per 15 minutes per **client IP** and, separately,
per **username** (case-insensitive); if either is exhausted the request gets `429` with
`Retry-After`. Attempts still in progress count against the limit, so concurrent bursts cannot
exceed it. A successful login clears only that username's counter, never the IP's. The client
IP is the TCP peer, or — if the peer is in `PIMED_TRUSTED_PROXIES` — the right-most
`X-Forwarded-For` address that is not itself a trusted proxy.

If a login request carries a session cookie, that old session is deleted server-side and
replaced by the new one.

## Users (admin only)

| Method | Path | Auth | Request | Success | Errors |
|---|---|---|---|---|---|
| GET | `/api/users` | admin | – | `200 {"users": [User, ...]}` ordered by id | `401`, `403` |
| POST | `/api/users` | admin | `{"username","password","role"?}` (omitted `role` defaults to `"user"`; `"role": ""` or any value other than `"admin"`/`"user"` is `400`) | `201 {"user": User}` | `400`, `401`, `403`, `409` username taken, `503` busy |

## Misc

| Method | Path | Auth | Success |
|---|---|---|---|
| GET | `/api/health` | none | `200 {"status":"ok"}` |

A known `/api/*` path with the wrong method → `405 {"error":"method not allowed"}` + `Allow`.
Any other `/api/*` path → `404 {"error":"not found"}`. Non-`/api` paths serve the SPA
(unknown paths fall back to `index.html`).

## Configuration (environment variables)

| Variable | Default | Meaning |
|---|---|---|
| `PIMED_ADDR` | `127.0.0.1:8080` | Listen address. Loopback by default: the backend is only meant to be reached through Caddy on the same host. In Docker (Caddy as a docker-compose sibling) the container sets `PIMED_ADDR=:8080`. |
| `PIMED_DATA_DIR` | `/data` | Files/media root (created if missing) |
| `PIMED_DB_PATH` | `/var/lib/pimed/pimed.db` | SQLite file (parent dir created if missing) |
| `PIMED_COOKIE_SECURE` | `true` | Set `false` for plain-HTTP local development |
| `PIMED_SESSION_TTL` | `720h` | Session lifetime (Go duration) |
| `PIMED_TRUSTED_PROXIES` | `127.0.0.1/32,::1/128` | Comma-separated CIDRs/IPs of reverse proxies whose `X-Forwarded-For` is trusted for the client IP. Set to an empty string to trust none. With docker-compose, set it to the compose network (e.g. `172.18.0.0/16`). |
| `PIMED_LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
