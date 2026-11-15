# PiMed backend

Go server for PiMed: JSON API (see [`../docs/api.md`](../docs/api.md)) plus the embedded
SvelteKit SPA. Pure Go (SQLite via `modernc.org/sqlite`), no CGO, single static binary.

## Layout (hexagonal, dependencies point inward)

```
cmd/pimed/                     composition root (config, wiring, graceful shutdown)
internal/domain/user/          User entity, Role, validation, errors, Repository/PasswordHasher ports
internal/domain/session/       Session entity, Repository/TokenGenerator ports, token hashing
internal/app/                  use cases: setup, login/logout, current user, create/list users
internal/adapters/inbound/http/       net/http router, handlers, DTOs, middleware, login rate limiter (package httpapi)
internal/adapters/outbound/sqlite/    repositories + embedded SQL migrations
internal/adapters/outbound/argon2id/  argon2id password hasher
internal/adapters/outbound/token/     crypto/rand session tokens
internal/platform/             config (env), logging, http server, SPA handler
web/                           go:embed of the frontend build output (web/dist/, gitignored)
```

## Run locally

`tmp/` is gitignored, so it is a convenient place for local data.

Linux / macOS / Git Bash:

```sh
PIMED_DATA_DIR=./tmp/data PIMED_DB_PATH=./tmp/pimed.db PIMED_COOKIE_SECURE=false go run ./cmd/pimed
```

Windows PowerShell:

```powershell
$env:PIMED_DATA_DIR = "./tmp/data"; $env:PIMED_DB_PATH = "./tmp/pimed.db"; $env:PIMED_COOKIE_SECURE = "false"
go run ./cmd/pimed
```

Then open <http://127.0.0.1:8080>. On first run the SPA (or `POST /api/setup`) creates the
admin account. If the frontend has not been built into `web/dist/`, non-API paths show a
placeholder page; rebuild the server after building the frontend, since files are embedded.

`PIMED_COOKIE_SECURE=false` is required for plain HTTP, otherwise browsers drop the cookie.

### Configuration

| Variable | Default | Meaning |
|---|---|---|
| `PIMED_ADDR` | `127.0.0.1:8080` | Listen address; loopback by default so only Caddy on the same host can reach it. Docker sets `:8080` (Caddy as a compose sibling) |
| `PIMED_DATA_DIR` | `/data` | Files/media root (created if missing) |
| `PIMED_DB_PATH` | `/var/lib/pimed/pimed.db` | SQLite file (parent dir created if missing) |
| `PIMED_COOKIE_SECURE` | `true` | `false` for plain-HTTP local development |
| `PIMED_SESSION_TTL` | `720h` | Session lifetime (Go duration, at least `1m`) |
| `PIMED_LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `PIMED_TRUSTED_PROXIES` | `127.0.0.1/32,::1/128` | CIDRs/IPs whose `X-Forwarded-For` is trusted; empty string = trust none. In docker-compose use the compose network CIDR |

## Test

```sh
go vet ./...
go test ./...
```

## Build for the Raspberry Pi

```sh
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/pimed-linux-arm64 ./cmd/pimed
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/pimed-linux-amd64 ./cmd/pimed
```

PowerShell: `$env:GOOS="linux"; $env:GOARCH="arm64"; $env:CGO_ENABLED="0"; go build -o bin/pimed-linux-arm64 ./cmd/pimed`
(then `Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED`).

## Security notes

- Passwords: argon2id, m=64 MiB, t=3, p=4 (RFC 9106 second recommendation), at most two
  hashes computed concurrently to bound memory on a 2 GB Pi; at most 8 callers may queue
  (honouring request cancellation), further ones get `503 server busy`. Parameters are stored in each
  hash, so they can be changed later without breaking existing accounts.
- Sessions: 32 random bytes (base64url) in an `HttpOnly`, `SameSite=Strict` cookie; only
  the SHA-256 of the token is stored. Expired sessions are rejected, deleted on lookup and
  pruned hourly.
- Login rate limit: 10 failed attempts per 15 minutes per client IP and, separately, per
  username, then `429`. Attempts are reserved before the password is verified, so concurrent
  bursts cannot exceed the limit; a success clears only the username's counter. The client IP
  is the TCP peer unless that peer is in `PIMED_TRUSTED_PROXIES` (default: loopback, i.e.
  Caddy on the same host), in which case the right-most untrusted `X-Forwarded-For` entry is
  used.
- Headers: `Content-Security-Policy` on every response (API: `default-src 'none'`; SPA: a
  strict same-origin policy whose `script-src` includes the SHA-256 of SvelteKit's inline
  bootstrap script, computed from the embedded `index.html` at startup), plus `nosniff`,
  `X-Frame-Options: DENY`, `Referrer-Policy: same-origin`.
- Slow bodies: JSON endpoints set a 10 s read deadline for the request body (`408`). The
  server has no global read timeout so future upload/streaming routes are not affected.
- CSRF: non-GET API requests must be `application/json` (else `415`) plus `SameSite=Strict`.
