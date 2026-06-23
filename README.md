# Capital Hub

Self-hosted, multi-user asset management for tracking items (land, apartments,
houses, vehicles, etc.) along with their transactions, maintenance, paperwork,
receipts, attachments, people, and locations.

> Status: early development. Phase 1 (foundations) is in place — runnable
> backend, embedded SPA, SQLite with migrations, and a containerized build.

## Tech stack

| Layer    | Choice                                                        |
| -------- | ------------------------------------------------------------- |
| Backend  | Go (chi, slog, goose migrations), pure-Go SQLite (modernc)    |
| Frontend | SvelteKit (static SPA), Tailwind CSS                          |
| Database | SQLite (WAL mode), single file                                |
| Storage  | Local filesystem (mounted volume)                             |
| Runtime  | Single static binary with the frontend embedded; distroless  |

## Repository layout

```
backend/    Go API + core logic + database (embeds the built frontend)
frontend/   SvelteKit single-page app (PWA)
deploy/     docker-compose + Caddy reverse proxy
Dockerfile  Multi-stage build -> single static image
```

## Development

Requires Go 1.26+ and Node 22+.

Run the backend (terminal 1):

```bash
cd backend
CH_ENV=dev CH_DATA_DIR=./.devdata go run ./cmd/server
# serves on http://localhost:8080
```

Run the frontend dev server (terminal 2):

```bash
cd frontend
npm install
npm run dev
# serves on http://localhost:5173, proxies /api to the backend
```

Or use the Makefile:

```bash
make dev-backend     # run the Go server
make dev-frontend    # run the Vite dev server
make build           # build frontend, embed it, build the binary
make docker          # build the container image
```

## Configuration

All configuration is via environment variables (prefix `CH_`):

| Variable             | Default                  | Notes                                          |
| -------------------- | ------------------------ | ---------------------------------------------- |
| `CH_ENV`             | `prod`                   | `dev` or `prod`                                |
| `CH_ADDR`            | `:8080`                  | Listen address                                 |
| `CH_BASE_URL`        | `http://localhost:8080`  | External URL (OIDC redirects, email links)     |
| `CH_DATA_DIR`        | `./data`                 | SQLite database + uploads                       |
| `CH_LOG_LEVEL`       | `info`                   | `debug`/`info`/`warn`/`error`                  |
| `CH_SESSION_SECRET`  | —                        | Required in prod; signs session cookies        |
| `CH_TRUSTED_PROXIES` | —                        | Comma-separated CIDRs to trust `X-Forwarded-*` |

## Self-hosting (Docker Compose + Caddy)

```bash
cd deploy
cp .env.example .env
# edit .env: set CH_SESSION_SECRET (openssl rand -hex 32) and CH_DOMAIN
docker compose up -d
```

Caddy terminates TLS and proxies to the app. For local testing keep
`CH_DOMAIN=localhost` (served over HTTP on port 80).

## Backups

Back up the `/data` volume — it contains `capital-hub.db` (plus `-wal`/`-shm`)
and the `uploads/` directory. SQLite WAL mode allows safe online copies.

## License

See [LICENSE](LICENSE).
