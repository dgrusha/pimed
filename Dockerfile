# syntax=docker/dockerfile:1

# PiMed image: SvelteKit SPA built and embedded into a static Go binary.
# Multi-arch: `docker buildx build --platform linux/arm64,linux/amd64 .`
# The build stages run on the build machine's native platform and Go
# cross-compiles, so building an arm64 image on an amd64 PC is fast (no QEMU).

# ── 1. Frontend: SvelteKit static build → /src/backend/web/dist ───────────────
FROM --platform=$BUILDPLATFORM node:22-alpine AS frontend
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/ ./
# svelte.config.js writes the build to ../backend/web/dist
RUN npm run build

# ── 2. Backend: static Go binary with the SPA embedded ───────────────────────
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS backend
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend /src/backend/web/dist ./web/dist
ARG TARGETOS=linux
ARG TARGETARCH
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w -X main.version=$VERSION" -o /out/pimed ./cmd/pimed
# Writable dirs for the volumes, owned by the distroless "nonroot" user (65532).
RUN mkdir -p /out/data /out/db

# ── Tests (only built with `docker build --target test .`) ───────────────────
# Frontend: type-check (svelte-check) + formatting (prettier). The build itself
# already ran in the `frontend` stage.
FROM frontend AS frontend-test
RUN npm run check && npm run lint

# Backend: gofmt, go vet, and all tests with the race detector (needs cgo, so
# this uses the Debian-based Go image that ships gcc).
FROM --platform=$BUILDPLATFORM golang:1.27 AS backend-test
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend /src/backend/web/dist ./web/dist
RUN unformatted="$(gofmt -l .)" && [ -z "$unformatted" ] || { echo "gofmt needed:"; echo "$unformatted"; exit 1; }
RUN go vet ./...
RUN CGO_ENABLED=1 go test -race -count=1 ./...

# Succeeds only if both test stages passed.
FROM scratch AS test
COPY --from=frontend-test /src/frontend/package.json /passed/frontend
COPY --from=backend-test /src/backend/go.mod /passed/backend

# ── 3. Runtime: distroless, non-root, no shell ───────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=backend /out/pimed /usr/local/bin/pimed
COPY --from=backend --chown=65532:65532 /out/data /data
COPY --from=backend --chown=65532:65532 /out/db /var/lib/pimed

# Inside a container the server must listen on all interfaces so Caddy (a
# sibling container) can reach it. The port is NOT published to the host.
ENV PIMED_ADDR=:8080 \
    PIMED_DATA_DIR=/data \
    PIMED_DB_PATH=/var/lib/pimed/pimed.db

USER nonroot:nonroot
EXPOSE 8080
VOLUME ["/data", "/var/lib/pimed"]
ENTRYPOINT ["/usr/local/bin/pimed"]
