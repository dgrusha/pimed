#!/usr/bin/env bash
# PiMed installer — installs or upgrades PiMed as a systemd service, and (if
# Tailscale is present) sets up Caddy for HTTPS on the Pi's *.ts.net name.
#
#   curl -fsSL https://github.com/dgrusha/pimed/releases/latest/download/install.sh | sudo bash
#
# Options (append after `sudo bash -s --` when piping, e.g. `| sudo bash -s -- --no-caddy`):
#   --version vX.Y.Z   install a specific release instead of the latest
#   --data-dir PATH    files/media root on first install (default: /data)
#   --no-caddy         install only the PiMed service, skip Caddy/HTTPS setup
#   --archive FILE     install from a local pimed_linux_<arch>.tar.gz (offline)
#   --uninstall        remove PiMed's service and binary (keeps data, DB, settings)
#   -h, --help         show this help
#
# Re-running the script upgrades the binary and unit; existing settings in
# /etc/pimed/pimed.env, the database and your files are never overwritten.

set -euo pipefail

REPO="dgrusha/pimed"
BIN="/usr/local/bin/pimed"
ENV_FILE="/etc/pimed/pimed.env"
UNIT_FILE="/etc/systemd/system/pimed.service"
UNIT_DROPIN_DIR="/etc/systemd/system/pimed.service.d"
CADDYFILE="/etc/caddy/Caddyfile"
CADDY_DROPIN_DIR="/etc/systemd/system/caddy.service.d"
MARKER="# Managed by the PiMed installer."

VERSION="latest"
DATA_DIR="/data"
WITH_CADDY=1
ARCHIVE=""
TMP=""

info() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33mwarning:\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }
have() { command -v "$1" >/dev/null 2>&1; }
usage() {
	cat <<'EOF'
PiMed installer. Usage: install.sh [options]
  --version vX.Y.Z   install a specific release instead of the latest
  --data-dir PATH    files/media root on first install (default: /data)
  --no-caddy         install only the PiMed service, skip Caddy/HTTPS setup
  --archive FILE     install from a local pimed_linux_<arch>.tar.gz (offline)
  --uninstall        remove PiMed's service and binary (keeps data, DB, settings)
  -h, --help         show this help
EOF
}

cleanup() { [ -n "$TMP" ] && rm -rf "$TMP"; }
trap cleanup EXIT

parse_args() {
	while [ $# -gt 0 ]; do
		case "$1" in
			--version) VERSION="${2:?--version needs a value}"; shift 2 ;;
			--data-dir) DATA_DIR="${2:?--data-dir needs a value}"; shift 2 ;;
			--no-caddy) WITH_CADDY=0; shift ;;
			--archive) ARCHIVE="${2:?--archive needs a file}"; shift 2 ;;
			--uninstall) uninstall; exit 0 ;;
			-h|--help) usage; exit 0 ;;
			*) die "unknown option: $1 (see --help)" ;;
		esac
	done
	case "$DATA_DIR" in /*) ;; *) die "--data-dir must be an absolute path" ;; esac
}

preflight() {
	[ "$(id -u)" -eq 0 ] || die "run as root, e.g.: curl -fsSL … | sudo bash"
	[ "$(uname -s)" = "Linux" ] || die "PiMed's installer supports Linux only"
	have systemctl || die "systemd is required (use the Docker setup otherwise)"
	have tar || die "tar is required"
	case "$(uname -m)" in
		aarch64|arm64) ARCH="arm64" ;;
		x86_64|amd64) ARCH="amd64" ;;
		*) die "unsupported CPU '$(uname -m)': PiMed needs a 64-bit OS (arm64 or amd64)" ;;
	esac
}

download() {
	TMP="$(mktemp -d)"
	local name="pimed_linux_${ARCH}"
	if [ -n "$ARCHIVE" ]; then
		[ -f "$ARCHIVE" ] || die "archive not found: $ARCHIVE"
		info "Using local archive $ARCHIVE (checksum not verified)"
		cp "$ARCHIVE" "$TMP/$name.tar.gz"
	else
		have curl || die "curl is required"
		have sha256sum || die "sha256sum is required"
		local base="https://github.com/$REPO/releases/latest/download"
		[ "$VERSION" = "latest" ] || base="https://github.com/$REPO/releases/download/$VERSION"
		info "Downloading PiMed ($VERSION, linux/$ARCH)"
		curl -fsSL --retry 3 -o "$TMP/$name.tar.gz" "$base/$name.tar.gz" ||
			die "download failed — does release '$VERSION' exist at https://github.com/$REPO/releases ?"
		curl -fsSL --retry 3 -o "$TMP/checksums.txt" "$base/checksums.txt" ||
			die "could not download checksums.txt"
		(cd "$TMP" && sha256sum --ignore-missing --status -c checksums.txt) ||
			die "checksum mismatch for $name.tar.gz — aborting"
		info "Checksum OK"
	fi
	tar -xzf "$TMP/$name.tar.gz" -C "$TMP"
	PKG="$TMP/$name"
	[ -x "$PKG/pimed" ] || die "archive does not contain the pimed binary"
}

install_pimed() {
	local upgrading=0
	[ -x "$BIN" ] && upgrading=1

	if ! id pimed >/dev/null 2>&1; then
		info "Creating system user 'pimed'"
		useradd --system --no-create-home --home-dir /var/lib/pimed --shell /usr/sbin/nologin pimed
	fi

	info "Installing $("$PKG/pimed" --version) to $BIN"
	# Stop first so the running binary isn't replaced underneath the service.
	systemctl stop pimed 2>/dev/null || true
	install -m 0755 "$PKG/pimed" "$BIN"

	if [ -f "$ENV_FILE" ]; then
		info "Keeping existing settings in $ENV_FILE"
		DATA_DIR="$(sed -n 's/^PIMED_DATA_DIR=//p' "$ENV_FILE" | tail -n1)"
		DATA_DIR="${DATA_DIR:-/data}"
	else
		info "Writing settings to $ENV_FILE"
		install -d -m 0755 /etc/pimed
		sed "s#^PIMED_DATA_DIR=.*#PIMED_DATA_DIR=$DATA_DIR#" "$PKG/deploy/pimed.env" >"$ENV_FILE"
		if [ "$WITH_CADDY" -eq 0 ]; then
			warn "--no-caddy: cookies stay Secure, so log in through your own HTTPS proxy"
		fi
		chown root:pimed "$ENV_FILE"
		chmod 0640 "$ENV_FILE"
	fi

	if [ ! -d "$DATA_DIR" ]; then
		info "Creating data directory $DATA_DIR"
		install -d -o pimed -g pimed -m 0750 "$DATA_DIR"
	elif [ "$(stat -c %U "$DATA_DIR")" != "pimed" ]; then
		warn "$DATA_DIR exists and is not owned by 'pimed' — PiMed needs write access (sudo chown -R pimed:pimed $DATA_DIR)"
	fi

	install -m 0644 "$PKG/deploy/pimed.service" "$UNIT_FILE"
	if [ "$DATA_DIR" != "/data" ]; then
		install -d -m 0755 "$UNIT_DROPIN_DIR"
		printf '%s\n[Service]\nReadWritePaths=%s\n' "$MARKER" "$DATA_DIR" >"$UNIT_DROPIN_DIR/data-dir.conf"
	else
		rm -f "$UNIT_DROPIN_DIR/data-dir.conf"
	fi

	systemctl daemon-reload
	systemctl enable pimed >/dev/null 2>&1
	systemctl restart pimed
	if [ "$upgrading" -eq 1 ]; then
		info "PiMed upgraded and restarted"
	else
		info "PiMed service installed and started"
	fi
}

wait_healthy() {
	have curl || return 0
	local addr
	addr="$(sed -n 's/^PIMED_ADDR=//p' "$ENV_FILE" | tail -n1)"
	addr="${addr:-127.0.0.1:8080}"
	case "$addr" in :*) addr="127.0.0.1$addr" ;; esac
	for _ in $(seq 1 15); do
		if curl -fsS "http://$addr/api/health" >/dev/null 2>&1; then
			info "Backend is healthy on $addr"
			return 0
		fi
		sleep 1
	done
	warn "backend did not answer on $addr yet — check: journalctl -u pimed -n 50"
}

tailscale_info() {
	TS_NAME=""
	TS_IP=""
	have tailscale || return 1
	TS_IP="$(tailscale ip -4 2>/dev/null | head -n1 || true)"
	TS_NAME="$(tailscale status --json 2>/dev/null |
		sed -n '/"Self"/,/}/p' | sed -n 's/.*"DNSName": *"\([^"]*\)".*/\1/p' | head -n1 || true)"
	TS_NAME="${TS_NAME%.}"
	[ -n "$TS_IP" ] && [ -n "$TS_NAME" ]
}

setup_caddy() {
	if ! tailscale_info; then
		warn "Tailscale is not installed or not logged in — skipping Caddy/HTTPS."
		warn "Install Tailscale (https://tailscale.com/download/linux), run 'sudo tailscale up', then re-run this installer."
		CADDY_DONE=0
		return
	fi
	info "Tailscale: $TS_NAME ($TS_IP)"

	if ! have caddy; then
		have apt-get || die "Caddy is not installed and apt-get is unavailable — install Caddy, then re-run (or use --no-caddy)"
		info "Installing Caddy"
		DEBIAN_FRONTEND=noninteractive apt-get update -qq
		DEBIAN_FRONTEND=noninteractive apt-get install -y -qq caddy >/dev/null
	fi

	# Allow Caddy (running as user 'caddy') to obtain the *.ts.net certificate.
	if ! grep -qs '^TS_PERMIT_CERT_UID=caddy' /etc/default/tailscaled; then
		info "Allowing Caddy to request Tailscale certificates"
		echo 'TS_PERMIT_CERT_UID=caddy' >>/etc/default/tailscaled
		systemctl restart tailscaled || warn "could not restart tailscaled — restart it manually so Caddy can get certificates"
		sleep 2
	fi

	if [ -f "$CADDYFILE" ] && ! grep -qF "$MARKER" "$CADDYFILE"; then
		local backup
		backup="$CADDYFILE.bak.$(date +%Y%m%d%H%M%S)"
		cp "$CADDYFILE" "$backup"
		warn "Existing $CADDYFILE saved as $backup"
	fi
	install -d -m 0755 /etc/caddy
	cat >"$CADDYFILE" <<EOF
$MARKER Re-running it rewrites this file.
$TS_NAME {
	# Listen only on the Tailscale IP, not on the LAN.
	bind $TS_IP
	encode zstd gzip
	reverse_proxy 127.0.0.1:8080
}
EOF

	# Caddy binds the Tailscale IP, so start it after tailscaled and retry until it's up.
	install -d -m 0755 "$CADDY_DROPIN_DIR"
	cat >"$CADDY_DROPIN_DIR/pimed.conf" <<EOF
$MARKER
[Unit]
After=tailscaled.service
Wants=tailscaled.service

[Service]
Restart=on-failure
RestartSec=5
EOF
	systemctl daemon-reload
	systemctl enable caddy >/dev/null 2>&1
	systemctl restart caddy
	info "Caddy configured for https://$TS_NAME"
	CADDY_DONE=1
}

uninstall() {
	[ "$(id -u)" -eq 0 ] || die "run as root"
	info "Removing PiMed service and binary"
	systemctl disable --now pimed 2>/dev/null || true
	rm -f "$UNIT_FILE" "$BIN"
	rm -rf "$UNIT_DROPIN_DIR"
	if grep -qsF "$MARKER" "$CADDYFILE"; then
		warn "$CADDYFILE was written by the PiMed installer — remove or edit it, then: sudo systemctl reload caddy"
	fi
	systemctl daemon-reload
	info "Done. Kept: $ENV_FILE, /var/lib/pimed (database), your data directory and the 'pimed' user."
	info "Delete them manually for a full removal."
}

main() {
	parse_args "$@"
	preflight
	download
	install_pimed
	wait_healthy
	CADDY_DONE=0
	[ "$WITH_CADDY" -eq 1 ] && setup_caddy

	echo
	info "PiMed is installed."
	if [ "$CADDY_DONE" -eq 1 ]; then
		echo "    Open https://$TS_NAME from a device on your tailnet and create the first (admin) account."
		echo "    (MagicDNS and HTTPS certificates must be enabled in the Tailscale admin console.)"
	else
		echo "    The backend listens on 127.0.0.1:8080 — put an HTTPS proxy in front (see README)."
	fi
	echo "    Settings: $ENV_FILE   Logs: journalctl -u pimed -f   Upgrade: re-run this installer."
}

# Everything runs from main, so a partially downloaded script never executes half-way.
main "$@"
