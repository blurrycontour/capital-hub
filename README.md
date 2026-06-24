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
CH_DATA_DIR=./.devdata go run ./cmd/server
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

| Variable                      | Default                  | Notes                                                          |
| ----------------------------- | ------------------------ | -------------------------------------------------------------- |
| `CH_ADDR`                     | `:8080`                  | Listen address                                                 |
| `CH_BASE_URL`                 | `http://localhost:8080`  | External URL (OIDC redirects, email links)                     |
| `CH_DATA_DIR`                 | `./data`                 | SQLite database + uploads                                      |
| `CH_LOG_LEVEL`                | `info`                   | `debug`/`info`/`warn`/`error`                                  |
| `CH_SESSION_SECRET`           | —                        | Signs session cookies                                          |
| `CH_TRUSTED_PROXIES`          | —                        | Comma-separated CIDRs to trust `X-Forwarded-*`                 |
| `CH_OIDC_ENABLED`             | `false`                  | Enable OIDC/SSO login                                          |
| `CH_OIDC_ISSUER_URL`          | —                        | Provider base URL                                              |
| `CH_OIDC_CLIENT_ID`           | —                        | OIDC client ID                                                 |
| `CH_OIDC_CLIENT_SECRET`       | —                        | OIDC client secret                                             |
| `CH_OIDC_REDIRECT_URL`        | —                        | Must be `https://<host>/api/v1/auth/oidc/callback`             |
| `CH_OIDC_ADMIN_GROUP`         | —                        | Group claim that grants Administrator role                     |
| `CH_OIDC_PROVIDER_NAME`       | `OIDC`                   | Label on the login button ("Sign in with …")                   |
| `CH_OIDC_ALLOW_REGISTRATION`  | `true`                   | `false` to prevent new account creation via OIDC              |

OIDC can also be configured entirely through the admin UI (Settings → OIDC /
SSO). Environment variables always take priority. See [docs/oidc.md](docs/oidc.md)
for a full setup guide.

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
