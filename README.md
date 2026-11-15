# PiMed

A private, self-hosted multimedia and file server for the Raspberry Pi 5, reachable only over
Tailscale. Go backend (hexagonal architecture) with the SvelteKit SPA embedded into a single
binary, fronted by Caddy.

| Path | What |
|---|---|
| `backend/` | Go server — see [backend/README.md](backend/README.md) |
| `frontend/` | SvelteKit SPA (builds into `backend/web/dist`, embedded into the binary) |
| `docs/api.md` | HTTP API contract and configuration reference |
| `Dockerfile`, `docker-compose*.yml`, `Caddyfile` | Docker setup (below) |

## Running with Docker

Docker runs two containers that work together:

```
 browser ──► caddy (ports 80/443) ──► pimed:8080 (private network only)
              TLS, compression,        Go API + the built frontend,
              forwards client IP       SQLite DB, your files
```

- **pimed** — one image containing both halves of the app. The [Dockerfile](Dockerfile) builds
  it in three stages: Node builds the SvelteKit frontend → Go compiles a static binary with that
  frontend embedded → the binary is copied into a minimal distroless image that runs as a
  non-root user. The backend port is **not** published; only Caddy can reach it.
- **caddy** — the reverse proxy from the plan. It serves HTTP for local use or automatic HTTPS
  with a Tailscale certificate on the Pi, and passes the real client IP to the backend (which
  only trusts Caddy's fixed address, `172.30.0.10`, for that).

### Data and persistence

| Volume | Mounted at | Contains |
|---|---|---|
| `pimed-data` (or `PIMED_DATA_PATH`) | `/data` | Your files and media |
| `pimed-db` | `/var/lib/pimed` | SQLite database (users, sessions) |
| `caddy-data`, `caddy-config` | Caddy | TLS certificates and Caddy state |

Volumes survive `docker compose down` and image rebuilds. `docker compose down -v` deletes them —
**including all users and files**.

### Try it locally

Requires Docker with Compose v2.

```sh
docker compose up -d --build
```

Open <http://localhost> — or, from another device on the same network (e.g. a laptop),
`http://<this computer's LAN IP>` (Windows: `ipconfig`, Linux/macOS: `ip addr` / `ipconfig getifaddr en0`).
On Windows the firewall must allow Docker Desktop on the network's profile (Docker Desktop
normally adds this rule itself). The first visit shows **Create the first account** — that account
becomes the admin, and setup then closes for good. The admin adds further users on the
**Users** page.

If port 80 is taken, choose another: `PIMED_HTTP_PORT=8080 docker compose up -d` (PowerShell:
`$env:PIMED_HTTP_PORT=8080; docker compose up -d`), then open <http://localhost:8080>.

The local defaults (`PIMED_SITE=:80`) use plain HTTP and therefore non-`Secure` cookies — fine on a
trusted home network for testing, never on an untrusted one. Until the first account exists,
anyone who can reach the page can create the admin, so create it right away.

### Deploy on the Raspberry Pi (Tailscale + HTTPS)

1. Install Docker and Tailscale on the Pi, join your tailnet, and in the Tailscale admin console
   enable **MagicDNS** and **HTTPS certificates**.
2. Clone the repo on the Pi and create the settings file:
   ```sh
   cp .env.example .env
   ```
   Set `PIMED_SITE` to the Pi's Tailscale name (e.g. `pimed.your-tailnet.ts.net`),
   `PIMED_COOKIE_SECURE=true`, and `PIMED_BIND_IP` to the Pi's Tailscale IP
   (`tailscale ip -4`) so the ports are open only on the tailnet, not your LAN.
3. Start it with the Tailscale add-on, which gives Caddy access to the host's `tailscaled` to
   fetch the certificate:
   ```sh
   docker compose -f docker-compose.yml -f docker-compose.tailscale.yml up -d --build
   ```
4. On your iPhone (Tailscale connected) open `https://pimed.your-tailnet.ts.net`.

The image builds natively on the Pi (arm64). To build on a PC for the Pi instead:
`docker buildx build --platform linux/arm64 -t pimed:latest .`

### Moving files to an external disk later

Set `PIMED_DATA_PATH` in `.env` to the mount point (e.g. `/mnt/pimed`). The container runs as
UID 65532, so give it the directory first:

```sh
docker compose down
sudo rsync -a /var/lib/docker/volumes/pimed_pimed-data/_data/ /mnt/pimed/   # copy existing files
sudo chown -R 65532:65532 /mnt/pimed
docker compose up -d
```

(The volume name prefix is your Compose project name — the folder name by default; check with
`docker volume ls`.)

### Settings (`.env`)

| Variable | Default | Meaning |
|---|---|---|
| `PIMED_SITE` | `:80` | Address Caddy serves. `:80` = plain HTTP for any hostname/IP; a bare hostname (e.g. `pimed.your-tailnet.ts.net`) gets automatic HTTPS. |
| `PIMED_COOKIE_SECURE` | `false` | Must be `true` whenever you use HTTPS. |
| `PIMED_BIND_IP` | `0.0.0.0` | Host IP the ports are published on. On the Pi, use its Tailscale IP. |
| `PIMED_HTTP_PORT` / `PIMED_HTTPS_PORT` | `80` / `443` | Host ports. |
| `PIMED_DATA_PATH` | `pimed-data` volume | Host directory for files/media. |
| `PIMED_SESSION_TTL` | `720h` | How long a login lasts. |
| `PIMED_LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error`. |

Other backend settings are fixed by the image (`PIMED_ADDR=:8080`, data and DB paths, trusted
proxy). The full list is in [docs/api.md](docs/api.md#configuration-environment-variables).

### Everyday commands

```sh
docker compose logs -f pimed           # backend logs
docker compose up -d --build           # rebuild and restart after pulling changes
docker compose down                    # stop (data is kept)
docker compose ps                      # status
```

## Running the tests (Docker)

All checks for the whole project run inside Docker — no local Go or Node needed, and it avoids
Windows policies that block locally built test binaries:

```sh
docker build --target test --output type=cacheonly .
```

This runs, in parallel:

- **frontend** — `npm ci`, `npm run build`, `npm run check` (svelte-check / TypeScript) and
  `npm run lint` (prettier);
- **backend** — `gofmt` check, `go vet ./...` and `go test -race -count=1 ./...` (all tests with
  the race detector, against the freshly built frontend).

The command fails if any step fails; add `--progress=plain` to see the full output.
`--output type=cacheonly` keeps the run from leaving an image behind. Unchanged steps are cached,
so add `--no-cache` to force everything to rerun.

## Development without Docker

Run the backend and frontend separately with hot reload — see
[backend/README.md](backend/README.md) for the backend, then in `frontend/` run `npm install`
and `npm run dev` (it proxies `/api` to the backend on `127.0.0.1:8080`).
