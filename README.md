# Family Finance

Self-hosted family finance tracker for Thai banking (SCB joint checking, KTC joint credit). Track partial spending per person, settle reimbursements, reconcile effective balance, and manage savings goals.

## Stack

- **Backend**: Go 1.23, Gin, PostgreSQL, golang-migrate
- **Web**: React, TypeScript, AG Grid (Excel-like editing)
- **iOS**: SwiftUI (opt-in receipt capture, Phase 2)
- **Deploy**: Docker Compose → Coolify

## Quick start (local)

### Prerequisites

- Go 1.23+
- [Air](https://github.com/air-verse/air) for hot reload
- Node.js 22+
- Docker & Docker Compose

### 1. Start Postgres only

```bash
docker compose up db -d
```

(`docker-compose.override.yml` is auto-merged — db exposes port 5432 locally.)

### 2. Run API with hot reload

**Option A — Air on host (fastest):**

```bash
cd backend
export DATABASE_URL="postgres://familyfinance:familyfinance@localhost:5432/familyfinance?sslmode=disable"
export API_KEY="dev-api-key"
air
```

**Option B — everything in Docker (override runs Air + Vite for you):**

```bash
docker compose up --build   # once; then just edit files — no container restarts
```

- Web: http://localhost:5173 (Vite HMR, file-watch polling in Docker)
- API: http://localhost:8080 (Air rebuilds on `.go` changes)
- Postgres: localhost:5432

`docker-compose.override.yml` is **auto-loaded** for local dev. Only restart containers when changing `docker-compose*.yml`, `Dockerfile*`, `package.json`, or `go.mod`.

### 3. Web + API on host (alternative)

```bash
docker compose up db -d
make dev-api   # terminal 1
make dev-web   # terminal 2
```

Web: http://localhost:5173 · API: http://localhost:8080/health

## Auth (web)

Local Docker (`docker-compose.override.yml`) sets `SKIP_AUTH=true` and `VITE_SKIP_AUTH=true`. The UI auto-calls **dev login** so you can test without LINE.

Production uses **LINE Login**. Set `LINE_LOGIN_CHANNEL_ID`, `LINE_LOGIN_CHANNEL_SECRET`, `LINE_LOGIN_CALLBACK_URL` (must match LINE console), `WEB_BASE_URL` (where users land after login), and `JWT_SECRET`. **Never set `SKIP_AUTH=true` on Coolify.**

Callback URL: `{WEB_BASE_URL}/callback` (e.g. `https://finance.diverti.se/callback`). Register that exact URL in the LINE Login channel. The web container proxies `/callback` to the API.

## LINE Messaging bot

Webhook: `POST {WEB_BASE_URL}/webhooks/line`  
(e.g. `https://finance.diverti.se/webhooks/line`). The web nginx proxies that path to the API, so you do not need a public API hostname.  
Set this in the LINE Official Account / Messaging API console.

Env: `LINE_CHANNEL_SECRET`, `LINE_CHANNEL_ACCESS_TOKEN`, `OPENAI_API_KEY`.

In a **group**, the bot can bind on join (`/bind`). Unmapped users reply `/iam 1` or `/iam 2`. Send a receipt photo or text such as `spent 350 on 7-Eleven (joint)` then reply `yes` to confirm.

## API

`/api/v1/*` requires `Authorization: Bearer <jwt>` (or `X-API-Key` if configured). Exceptions: `/api/v1/auth/line/*`, `/api/v1/auth/dev-login`, `/webhooks/line`.

| Endpoint | Description |
|----------|-------------|
| `GET /transactions` | List transactions with splits |
| `PATCH /transactions/bulk` | Spreadsheet bulk save |
| `POST /transactions` | Create (`is_joint` skips reimbursement) |
| `POST /receipts/parse` | OpenAI vision draft (confirm in UI) |
| `GET /reports/settlement?month=2026-07` | Who owes whom |
| `GET /reports/effective-balance` | SCB effective balance |
| `GET/POST /caregivers`, `/caregiver-hours` | Extra nanny hours |
| `GET/POST /lists` | Household lists |
| `GET/POST /pantry-items` | Expiring items |

## Testing (TDD)

```bash
cd backend
go test ./...
```

Domain logic tests cover splits, settlement math, and effective balance before HTTP layer.

## Coolify deployment

1. Push repo to GitHub
2. In Coolify: **New Resource → Docker Compose**
3. Point at this repo; Coolify uses [`docker-compose.yml`](docker-compose.yml) only
4. **Do not** deploy with `docker-compose.override.yml` merged — Coolify should not load the dev override (use the compose file path setting, not `docker compose up` from a dev machine's full tree if that merges overrides)
4. Set secrets:
   - `POSTGRES_PASSWORD`
   - `JWT_SECRET`
   - `LINE_LOGIN_*`, `LINE_CHANNEL_*`, `OPENAI_API_KEY`
   - `WEB_BASE_URL` and `LINE_LOGIN_CALLBACK_URL`
   - Do **not** set `SKIP_AUTH=true`
5. Ensure volumes persist: `postgres_data`, `receipts_data`
6. Attach the public domain to the **web** service, container port **80**. Do not publish host port 3000 — Coolify's proxy reaches the container on the Compose network.

Health checks are configured on `db`, `api`, and `web` services.

### Production checklist

- [ ] Change `API_KEY` and `POSTGRES_PASSWORD` from defaults
- [ ] Point the domain at **web** (port 80), not **api**
- [ ] Back up `postgres_data` volume regularly

## iOS app (opt-in capture)

See [`ios/README.md`](ios/README.md). Receipt capture is **user-initiated only** — Share Extension, Photos picker, or camera. OCR prefills a form; user must confirm before save.

## Project structure

```
familyfinance/
├── backend/          # Go API
├── web/              # React + AG Grid UI
├── ios/              # SwiftUI capture app
├── docker-compose.yml
└── docker-compose.override.yml   # auto-loaded for local dev
```
