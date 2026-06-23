# syntax=docker/dockerfile:1

# --- Stage 1: build the SvelteKit frontend ---------------------------------
FROM node:22-alpine AS frontend
WORKDIR /app/frontend

# Install dependencies using the lockfile for reproducible builds.
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build


# --- Stage 2: build the Go backend (with frontend embedded) ----------------
FROM golang:1.26-alpine AS backend
WORKDIR /app/backend

# Cache module downloads.
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./

# Embed the compiled frontend into the binary.
COPY --from=frontend /app/frontend/build ./internal/web/dist

# Pure-Go SQLite => CGO can stay off for a fully static binary.
ARG VERSION=dev
RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags "-s -w -X main.version=${VERSION}" \
    -o /out/capital-hub ./cmd/server

# Pre-create the data dir so it can be COPYed with nonroot ownership below.
RUN mkdir -p /data


# --- Stage 3: minimal runtime image ----------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

# CA certificates are included in distroless/static for outbound TLS
# (OIDC discovery, SMTP over TLS).
COPY --from=backend /out/capital-hub /usr/local/bin/capital-hub

# Writable data directory owned by the nonroot user (uid 65532). A fresh named
# or anonymous volume inherits this ownership on first mount.
COPY --from=backend /data /data

ENV CH_ENV=prod \
    CH_ADDR=:8080 \
    CH_DATA_DIR=/data

VOLUME ["/data"]
EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["/usr/local/bin/capital-hub"]
