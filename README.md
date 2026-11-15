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
| `install.sh` | One-command installer / upgrader for the Pi (shipped in each release) |
| `deploy/` | systemd unit, env file and Caddyfile for a binary install (shipped in each release) |
| `.github/workflows/` | CI on pull requests and the release pipeline |

## Install on a Raspberry Pi (no Docker)

Every release on the [Releases page](https://github.com/dgrusha/pimed/releases) contains:

| File | For |
|---|---|
| `install.sh` | The automatic installer (below) |
| `pimed_linux_arm64.tar.gz` | Raspberry Pi 5 / 4 (64-bit Raspberry Pi OS) |
| `pimed_linux_amd64.tar.gz` | Regular 64-bit x86 Linux PCs/servers |
| `checksums.txt` | SHA-256 checksums of both archives |

Each archive holds a single static `pimed` binary (frontend included — nothing else to install)
plus `deploy/` with a systemd unit, an env file and a Caddyfile.

### Automatic install (recommended)

First install Tailscale on the Pi and log in (`curl -fsSL https://tailscale.com/install.sh | sh`,
then `sudo tailscale up`), and enable **MagicDNS** and **HTTPS certificates** in the Tailscale
admin console. Then:

```sh
curl -fsSL https://github.com/dgrusha/pimed/releases/latest/download/install.sh | sudo bash
```

When it finishes it prints the address to open — `https://<pi-name>.<tailnet>.ts.net` — where
you create the first (admin) account.

What the installer does:

1. detects the CPU (arm64/amd64), downloads the matching release archive and **verifies its
   SHA-256 checksum**;
2. creates the `pimed` system user, installs the binary to `/usr/local/bin/pimed`, writes
   `/etc/pimed/pimed.env` (first install only), creates the data directory (`/data`) and the
   database directory (`/var/lib/pimed`);
3. installs the hardened systemd service, starts it and waits until `/api/health` answers;
4. if Tailscale is running: installs Caddy (apt), allows it to fetch the `*.ts.net` certificate
   (`TS_PERMIT_CERT_UID=caddy`), writes `/etc/caddy/Caddyfile` bound to the Pi's Tailscale IP only
   (an existing Caddyfile is backed up first), and makes Caddy start after Tailscale.

Options — pass them after `sudo bash -s --`:

```sh
curl -fsSL https://github.com/dgrusha/pimed/releases/latest/download/install.sh \
  | sudo bash -s -- --version v0.1.0 --data-dir /mnt/pimed
```

| Option | Meaning |
|---|---|
| `--version vX.Y.Z` | Install a specific release instead of the latest |
| `--data-dir PATH` | Files/media root on first install (default `/data`), e.g. an external disk |
| `--no-caddy` | Only install the PiMed service; set up HTTPS yourself |
| `--archive FILE` | Install from a downloaded `pimed_linux_<arch>.tar.gz` (offline) |
| `--uninstall` | Remove the service and binary — keeps data, database and settings |

**Upgrading:** run the same command again. The binary and service are replaced and restarted;
`/etc/pimed/pimed.env`, the database and your files are never overwritten.

Prefer to read scripts before running them as root (a good habit)? Download it first:
`curl -fsSLO https://github.com/dgrusha/pimed/releases/latest/download/install.sh`, review it,
then `sudo bash install.sh`.

### Manual install (optional)

The same steps by hand, if you'd rather not use the installer.

#### 1. Download and verify

On the Pi (`uname -m` should print `aarch64`):

```sh
cd /tmp
# Latest release:
curl -fLO https://github.com/dgrusha/pimed/releases/latest/download/pimed_linux_arm64.tar.gz
curl -fLO https://github.com/dgrusha/pimed/releases/latest/download/checksums.txt
# …or a specific version: replace "latest/download" with "download/v0.1.0".

sha256sum --ignore-missing -c checksums.txt     # must print: pimed_linux_arm64.tar.gz: OK
tar xzf pimed_linux_arm64.tar.gz
cd pimed_linux_arm64
./pimed --version
```

#### 2. Install as a service

```sh
sudo install -m 0755 pimed /usr/local/bin/pimed
sudo useradd --system --no-create-home --shell /usr/sbin/nologin pimed
sudo install -d -o pimed -g pimed -m 0750 /data          # files/media root
sudo install -D -m 0640 -g pimed deploy/pimed.env /etc/pimed/pimed.env
sudo install -m 0644 deploy/pimed.service /etc/systemd/system/pimed.service
sudo systemctl daemon-reload
sudo systemctl enable --now pimed
systemctl status pimed            # logs: journalctl -u pimed -f
```

The backend now listens on `127.0.0.1:8080` only. Settings live in `/etc/pimed/pimed.env`
(restart with `sudo systemctl restart pimed` after changes); the database is in `/var/lib/pimed/`.

#### 3. HTTPS over Tailscale with Caddy

```sh
sudo apt install caddy
echo 'TS_PERMIT_CERT_UID=caddy' | sudo tee -a /etc/default/tailscaled   # lets Caddy fetch the *.ts.net cert
sudo systemctl restart tailscaled
sudo install -m 0644 deploy/Caddyfile /etc/caddy/Caddyfile
sudo nano /etc/caddy/Caddyfile    # set your *.ts.net name and the Pi's IP from `tailscale ip -4`
sudo systemctl reload caddy
```

Open `https://<pi-name>.<tailnet>.ts.net` from a device on your tailnet and create the first
(admin) account. MagicDNS and HTTPS certificates must be enabled in the Tailscale admin console.

#### Updating a manual install

Download the new archive as in step 1, then:

```sh
sudo install -m 0755 pimed /usr/local/bin/pimed && sudo systemctl restart pimed
```

Your data and settings are untouched. Check `deploy/` in the new archive for changes to the unit
or env file. To move files to an external disk, point `PIMED_DATA_DIR` at the mount, `chown` it
to `pimed`, and add the path to `ReadWritePaths=` in the unit.

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

## Continuous integration

[`.github/workflows/ci.yml`](.github/workflows/ci.yml) runs on **every pull request** and on
pushes to `main`:

| Job | Checks |
|---|---|
| Frontend | `npm ci`, `svelte-check`, prettier, production build |
| Backend | `gofmt`, `go vet`, `go test -race` against the built frontend, arm64 cross-compile |
| Installer | `shellcheck install.sh` |
| Docker | builds the runtime image and smoke-tests it (`/api/health`, setup status, SPA) |

To block merging failing PRs: **Settings → Branches → Add branch ruleset** for `main` →
*Require status checks to pass* → select the four CI jobs.

## Making a release

Releases are built by [`.github/workflows/release.yml`](.github/workflows/release.yml) when a
version tag is pushed:

```sh
git checkout main && git pull
git tag v0.1.0              # semantic version; v0.2.0-rc.1 style tags become pre-releases
git push origin v0.1.0
```

The workflow then:

1. runs the full CI suite (nothing is released if tests fail);
2. builds the frontend once and cross-compiles static binaries for `linux/arm64` and
   `linux/amd64`, stamping the version (`pimed --version`);
3. packs each with `deploy/` and this README into `pimed_linux_<arch>.tar.gz`, writes
   `checksums.txt`;
4. creates the GitHub Release with auto-generated notes (from merged PRs/commits) and uploads the
   archives, `checksums.txt` and `install.sh`.

Watch it under the repository's **Actions** tab; the result appears under **Releases**. To redo a
broken release, delete the release and tag on GitHub (`git push --delete origin v0.1.0`), fix,
and tag again.

## Development without Docker

Run the backend and frontend separately with hot reload — see
[backend/README.md](backend/README.md) for the backend, then in `frontend/` run `npm install`
and `npm run dev` (it proxies `/api` to the backend on `127.0.0.1:8080`).
