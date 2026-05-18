# Network Control API

Backend for a **Telecom NOC (Network Operations Center) monitoring simulation platform**. The system is designed to simulate network infrastructure, telemetry, alerts, and incident workflows for learning and portfolio purposes — not for monitoring real production infrastructure.

This repository is built incrementally in phases. **Phase 1 (current)** provides the core backend foundation.

---

## Features (Phase 1)

- Gin HTTP server with clean architecture layout
- Environment-based configuration (`.env` support)
- Structured logging (`slog`)
- PostgreSQL connection pool (`pgx`)
- Health check endpoint with database status
- Graceful shutdown on `SIGINT` / `SIGTERM`
- Request logging middleware
- User registration and login (JWT)
- Password hashing (bcrypt)
- JWT authentication middleware
- Auto-migration for `users` and `devices` tables
- Device CRUD with pagination, filtering, and role-based access
- Swagger/OpenAPI documentation (auth & devices)

---

## Tech Stack

| Component   | Technology        |
|------------|-------------------|
| Language   | Go 1.26+          |
| Framework  | [Gin](https://github.com/gin-gonic/gin) |
| Database   | PostgreSQL        |
| DB Driver  | [pgx](https://github.com/jackc/pgx)     |
| Config     | Environment variables + [godotenv](https://github.com/joho/godotenv) |
| Auth       | JWT ([golang-jwt](https://github.com/golang-jwt/jwt)) + bcrypt       |
| API Docs   | [Swagger](https://swagger.io/) via [swaggo](https://github.com/swaggo/gin-swagger) |

---

## Project Structure

```
Network-control-api/
├── cmd/
│   └── server/              # Application entrypoint
├── internal/
│   ├── config/              # Configuration loading and validation
│   ├── handlers/            # HTTP handlers
│   ├── infrastructure/
│   │   └── database/        # PostgreSQL connection pool
│   ├── logger/              # Structured logging setup
│   ├── middleware/          # HTTP middleware
│   ├── auth/                # JWT token service
│   ├── models/              # Domain entities
│   ├── repositories/        # Data access layer
│   ├── router/              # Route registration
│   ├── server/              # HTTP server and graceful shutdown
│   ├── services/            # Business logic / use cases
│   └── httputil/            # Shared HTTP helpers
├── .env.example             # Environment variable template
├── PROJECT_CONTEXT.md       # Full project vision and phased roadmap
└── go.mod
```

### Architecture

The project follows **clean architecture** principles:

- **Handlers** — HTTP layer only; no business logic
- **Services** — application use cases (future phases)
- **Repositories** — database access (future phases)
- **Infrastructure** — external systems (PostgreSQL, etc.)

---

## Prerequisites

- [Go](https://go.dev/dl/) 1.26 or later
- [PostgreSQL](https://www.postgresql.org/) 14+ running locally or remotely

---

## Getting Started

### 1. Clone the repository

```bash
git clone <repository-url>
cd Network-control-api
```

### 2. Configure environment variables

```bash
cp .env.example .env
```

Edit `.env` to match your local PostgreSQL credentials.

### 3. Create the database

```sql
CREATE DATABASE network_control;
```

### 4. Install dependencies

```bash
go mod tidy
```

### 5. Run the server

```bash
go run ./cmd/server
```

The API listens on `http://localhost:8080` by default.

### 6. Open Swagger UI

- **Swagger UI:** [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)
- **OpenAPI JSON:** embedded via `docs/docs.go` (also available at `/swagger/doc.json`)

Use **Authorize** in Swagger UI and enter: `Bearer <your-jwt-token>` (after login).

To regenerate the spec from code annotations:

```bash
make swagger
```

### 7. Build a binary (optional)

```bash
go build -o bin/server ./cmd/server
./bin/server
```

---

## Environment Variables

| Variable                  | Default              | Description                          |
|---------------------------|----------------------|--------------------------------------|
| `APP_ENV`                 | `development`        | Application environment              |
| `APP_NAME`                | `network-control-api`| Service name in health responses     |
| `SERVER_HOST`             | `0.0.0.0`            | HTTP bind host                       |
| `SERVER_PORT`             | `8080`               | HTTP port                            |
| `SERVER_SHUTDOWN_TIMEOUT` | `10`                 | Graceful shutdown timeout (seconds)  |
| `LOG_LEVEL`               | `info`               | Log level: `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT`              | `text`               | Log format: `text` or `json`         |
| `DB_HOST`                 | `localhost`          | PostgreSQL host                      |
| `DB_PORT`                 | `5433`               | PostgreSQL port                      |
| `DB_USER`                 | `postgres`           | PostgreSQL user                      |
| `DB_PASSWORD`             | `postgres`           | PostgreSQL password                  |
| `DB_NAME`                 | `network_control`    | PostgreSQL database name             |
| `DB_SSLMODE`              | `disable`            | PostgreSQL SSL mode                  |
| `JWT_SECRET`              | *(required)*         | Secret key for signing JWTs          |
| `JWT_EXPIRATION_HOURS`    | `24`                 | Token lifetime in hours              |

---

## API Endpoints

### Authentication

#### Register

```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "email": "operator@noc.local",
  "password": "securepass",
  "role": "operator"
}
```

`role` is optional (`admin`, `operator`, `viewer`). Defaults to `viewer`.

#### Login

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "operator@noc.local",
  "password": "securepass"
}
```

Both endpoints return:

```json
{
  "token": "<jwt>",
  "user": {
    "id": "uuid",
    "email": "operator@noc.local",
    "role": "operator",
    "created_at": "2026-05-15T12:00:00Z",
    "updated_at": "2026-05-15T12:00:00Z"
  }
}
```

#### Protected test route

```http
GET /api/v1/protected/test
Authorization: Bearer <jwt>
```

### Devices

All device routes require JWT authentication.

| Method | Path | Roles |
|--------|------|-------|
| `GET` | `/api/v1/devices` | admin, operator, viewer |
| `GET` | `/api/v1/devices/:id` | admin, operator, viewer |
| `POST` | `/api/v1/devices` | admin, operator |
| `PATCH` | `/api/v1/devices/:id` | admin, operator |
| `DELETE` | `/api/v1/devices/:id` | admin |

#### List devices (pagination & filtering)

```http
GET /api/v1/devices?page=1&limit=10&search=core&type=router&status=online&sort_by=created_at&sort_order=desc
Authorization: Bearer <jwt>
```

Query parameters:

| Parameter | Description |
|-----------|-------------|
| `page` | Page number (default: `1`) |
| `limit` | Items per page (default: `10`, max: `100`) |
| `search` | Search in name, location, IP, description |
| `type` | Filter by type: `router`, `tower`, `switch`, `core_node`, `link`, `service` |
| `status` | Filter by status: `online`, `offline`, `degraded`, `maintenance` |
| `sort_by` | Sort field: `name`, `type`, `status`, `created_at`, `updated_at` |
| `sort_order` | `asc` or `desc` (default: `desc`) |

#### Create device

```http
POST /api/v1/devices
Authorization: Bearer <jwt>
Content-Type: application/json

{
  "name": "Core Router 01",
  "type": "router",
  "status": "online",
  "location": "DC-East",
  "ip_address": "10.0.0.1",
  "description": "Primary edge router"
}
```

#### Update device

```http
PATCH /api/v1/devices/:id
Authorization: Bearer <jwt>
Content-Type: application/json

{
  "status": "maintenance"
}
```

#### Delete device

```http
DELETE /api/v1/devices/:id
Authorization: Bearer <jwt>
```

### Health Check

```http
GET /health
```

**Response (healthy):**

```json
{
  "status": "ok",
  "service": "network-control-api",
  "database": "up"
}
```

**Response (database unavailable) — HTTP 503:**

```json
{
  "status": "degraded",
  "service": "network-control-api",
  "database": "down"
}
```

---

## Phase 2 — Monitoring (Implemented)

Phase 2 has now been implemented in this repository. The monitoring subsystem adds a realtime telemetry simulator, concurrent processing, alert evaluation and persistence, incident triggering, and WebSocket-based realtime events.

### New / Updated Features (Phase 2)

- Monitoring engine that periodically generates simulated metrics for devices and keeps recent metrics in memory (no high-frequency telemetry persisted).
- Concurrent architecture using goroutines and channels to evaluate metrics and produce alerts asynchronously.
- Alert engine that evaluates metrics against configurable rules and persists alerts to PostgreSQL.
- Incident engine that receives critical alerts and triggers incident creation/processing pipelines.
- WebSocket hub and handler to broadcast realtime events (metrics, device status changes, alerts, incidents) to connected clients.
- Monitoring HTTP endpoints to inspect live runtime state and connect to the realtime feed:
  - `GET /api/v1/monitoring/live` — list current runtime state for devices
  - `GET /api/v1/monitoring/live/:id` — single device runtime state
  - `GET /api/v1/monitoring/ws` — WebSocket upgrade endpoint for realtime events

### How it works (high level)

- The monitoring engine queries devices from the configured device provider and periodically generates telemetry (latency, packet loss, CPU, etc.).
- Generated metrics are sent to the alert engine via a channel. The alert engine evaluates metrics against alert rules and emits Alert objects which are persisted to the `alerts` table.
- Critical alerts are forwarded to the incident engine which can persist incidents and manage escalation (implemented as a simple engine in Phase 2).
- Realtime events are published to the WebSocket hub which broadcasts JSON events to connected clients without blocking the main pipelines.

---

## Tests for Phase 2

Run all tests (recommended):

```bash
go test ./...
```

Run only monitoring-related tests:

```bash
go test ./internal/monitoring -run Test
```

Run a specific test file (example):

```bash
go test ./internal/monitoring -run TestAlertEvaluator
```

Notes:
- Tests include unit tests for alert evaluation and engine logic.
- Use `go test -v` to see verbose output.

---

## Examples / Quick manual checks

1) Start the server (after configuring `.env` and database):

```bash
go run ./cmd/server
```

2) Connect to the WebSocket realtime feed (using wscat or websocat):

- Using wscat (npm):

```bash
# install if needed
npm install -g wscat
wscat -c ws://localhost:8080/api/v1/monitoring/ws
```

- Using websocat:

```bash
websocat ws://localhost:8080/api/v1/monitoring/ws
```

You should receive JSON events of type `metric`, `device_status`, `alert`, and `incident` while the monitoring engine is running.

3) Inspect live monitoring state via HTTP (requires a valid JWT):

```bash
# List live states
curl -H "Authorization: Bearer $JWT" http://localhost:8080/api/v1/monitoring/live

# Single device runtime state
curl -H "Authorization: Bearer $JWT" http://localhost:8080/api/v1/monitoring/live/<device-id>
```

4) Trigger a metric that causes an alert (manual / simulated):

- You can either modify a monitoring rule in code/tests to lower thresholds, or send a crafted MetricEvent into the monitoring engine in tests. For development, the unit tests demonstrate evaluation logic and examples.

---

If you want me to add specific curl examples for creating users, obtaining JWTs, or a short demo script that wires everything together (create user, login, connect websocket, list live states), tell me which flows you prefer and I will add them to the README.
