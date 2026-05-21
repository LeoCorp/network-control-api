# Network Control API

Backend for a **Telecom NOC (Network Operations Center) monitoring simulation platform**. The system is designed to simulate network infrastructure, telemetry, alerts, and incident workflows for learning and portfolio purposes — not for monitoring real production infrastructure.

This repository is built incrementally in phases. **Phase 3 (current)** provides a complete simulated NOC monitoring platform, including telemetry generation, alerting, real-time WebSockets, incident management, audit logging, and automated escalation workers.

---

## Core Features

- **Phases 1 & 2 (Foundation & Telemetry):**
  - Gin HTTP server with clean architecture layout and structured logging (`slog`).
  - PostgreSQL connection pool (`pgx`) with safe startup migration guards.
  - JWT Authentication middleware supporting role-based access control (`admin`, `operator`, `viewer`).
  - Device CRUD with advanced pagination, filtering, and query sorting.
  - Periodic telemetry simulator generating metrics and evaluating alert thresholds in memory.
  - Persistent alert engine with auto-linking of critical metrics to active incidents.
  - Real-time WebSocket hub broadcasting metric events, status updates, and alerts to clients.
  - Interactive Swagger/OpenAPI documentation for all endpoints.

- **Phase 3 (Incident Management & Concurrency):**
  - **Full Incident API:** Query, paginate, and detail incidents along with their historical audit trail and alerts.
  - **Audit Logs with JSONB:** Automatically log all actions (`created`, `status_changed`, `alert_linked`, `escalated`) into an audit trail table using flexible JSONB metadata storing status transition states.
  - **Auto-Escalation Worker:** Background routine polling open unacknowledged incidents. If age exceeds configured seconds, it marks them as escalated, prepends `[ESCALATED] ` to their title, and records an audit log.
  - **Dynamic Device Syncing:** Syncs PostgreSQL device status dynamically: `ONLINE` (no active incidents), `DEGRADED` (normal active incidents), or `OFFLINE` (escalated or telemetry-triggered downtime).

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
| `INCIDENT_ESCALATION_SECONDS` | `30`             | Incident response SLA time before auto-escalation (seconds) |

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
 - Database migrations now create `alerts` and `incidents` tables (auto-migrated at startup).
 - WebSocket endpoint is protected by JWT (use the same `Authorization: Bearer <token>` header when upgrading).
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
# with authentication header (wscat)
wscat -H "Authorization: Bearer <JWT>" -c ws://localhost:8080/api/v1/monitoring/ws
```

- Using websocat:

```bash
# with authentication header (websocat)
websocat -H "Authorization: Bearer <JWT>" ws://localhost:8080/api/v1/monitoring/ws
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

## Phase 3 — Incident Management & Realtime (Implemented)

Phase 3 introduces the operations workflow of a simulated Network Operations Center (NOC). This includes the RESTful API for managing incidents, persisting a full audit trail (audit logs) with `metadata JSONB` columns, syncing persistent device database statuses based on active incidents, and automated background escalations.

### Incidents API Endpoints

All incident routes require JWT authentication.

| Method  | Path                             | Roles                    | Description |
|---------|----------------------------------|--------------------------|-------------|
| `GET`   | `/api/v1/incidents`             | admin, operator, viewer   | Paginated list of incidents (filters: `status`, `device_id`) |
| `GET`   | `/api/v1/incidents/:id`         | admin, operator, viewer   | Details of a specific incident (includes linked alerts & logs) |
| `PATCH` | `/api/v1/incidents/:id/status`  | admin, operator          | Update incident status (`OPEN`, `INVESTIGATING`, `RESOLVED`) |
| `GET`   | `/api/v1/incidents/:id/logs`    | admin, operator, viewer   | Historical audit trail for a specific incident |

#### 1. List Incidents
```http
GET /api/v1/incidents?page=1&limit=10&status=OPEN
Authorization: Bearer <jwt>
```

#### 2. Get Incident Details (with Linked Alerts & Audit Trail)
```http
GET /api/v1/incidents/7b949c8f-dc11-4775-9276-f84dbcb2ccfa
Authorization: Bearer <jwt>
```
**Response Schema:**
```json
{
  "incident": {
    "id": "7b949c8f-dc11-4775-9276-f84dbcb2ccfa",
    "device_id": "8c37d896-189f-431c-b26a-9a99266abfa9",
    "device_name": "router-east-01",
    "title": "Critical latency on router-east-01",
    "description": "Latency metric has exceeded threshold",
    "status": "OPEN",
    "escalated": true,
    "created_at": "2026-05-20T12:00:00Z",
    "updated_at": "2026-05-20T12:05:00Z"
  },
  "alerts": [
    {
      "id": "2b9921da-085c-4f7f-ba7d-a128fb2cc1fa",
      "device_id": "8c37d896-189f-431c-b26a-9a99266abfa9",
      "device_name": "router-east-01",
      "severity": "CRITICAL",
      "metric": "latency",
      "message": "high latency detected: 420ms",
      "value": 420.0,
      "threshold": 300.0,
      "created_at": "2026-05-20T12:00:00Z"
    }
  ],
  "logs": [
    {
      "id": "ea3c3d52-fa22-4889-8d26-ccfae32a229a",
      "incident_id": "7b949c8f-dc11-4775-9276-f84dbcb2ccfa",
      "action": "created",
      "message": "Incident automatically created from critical alert.",
      "created_at": "2026-05-20T12:00:00Z"
    },
    {
      "id": "4dfba429-122e-4b68-b769-dcbbaee91fca",
      "incident_id": "7b949c8f-dc11-4775-9276-f84dbcb2ccfa",
      "action": "escalated",
      "message": "Incident automatically escalated due to response delay.",
      "metadata": {
        "age_seconds": 30,
        "escalation_seconds": 30
      },
      "created_at": "2026-05-20T12:00:30Z"
    }
  ]
}
```

#### 3. Update Incident Status (Acknowledge / Resolve)
Operators can manually transition status:
- **`OPEN` ➔ `INVESTIGATING`:** Acknowledges the incident. Logs audit trail and updates device status in DB to `degraded`.
- **`INVESTIGATING` / `OPEN` ➔ `RESOLVED`:** Closes the incident. Checks if other active incidents exist for the device. If none, transitions device back to `online` in PostgreSQL.
```http
PATCH /api/v1/incidents/7b949c8f-dc11-4775-9276-f84dbcb2ccfa/status
Authorization: Bearer <jwt>
Content-Type: application/json

{
  "status": "INVESTIGATING"
}
```

---

## How to Test and Verify Phase 3 Features

You can fully test Phase 3 either using the **Swagger UI** or **Postman/cURL**.

### A. Testing via Swagger UI

1. **Start the API:** Run `go run ./cmd/server` or `npm run dev`.
2. **Access Swagger UI:** Navigate to [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html).
3. **Register/Login:**
   - Execute `/api/v1/auth/register` or `/api/v1/auth/login` to create a user and obtain a JWT token.
4. **Authorize:**
   - Click the **Authorize** lock button in the top right.
   - Enter `Bearer <your-token>` (replacing `<your-token>` with the JWT string you copied) and click **Authorize**.
5. **Inspect the Incident Management endpoints:**
   - Under the `incidents` section, try calling `GET /api/v1/incidents` to list simulated incidents.
   - To manually transition a status, select `PATCH /api/v1/incidents/{id}/status`, specify an Incident UUID, and provide a JSON body such as `{"status": "INVESTIGATING"}`.
   - To inspect audit trails, use `GET /api/v1/incidents/{id}/logs`.

### B. Observing Automated Escalations (Telemetry Simulation)

1. Set `INCIDENT_ESCALATION_SECONDS=10` in your `.env` file to accelerate verification.
2. Establish a WebSocket connection using a client tool like `wscat` or `websocat`:
   ```bash
   wscat -H "Authorization: Bearer <JWT>" -c ws://localhost:8080/api/v1/monitoring/ws
   ```
3. When device metrics trigger a critical alert, you'll receive a websocket event of type `incident` with `"action": "created"`.
4. Observe the database `devices` status change to `degraded` for that device.
5. Wait **10 seconds** without acknowledging it:
   - You will see a new `incident` event with `"action": "escalated"` and `"escalated": true`.
   - The title is updated to start with `[ESCALATED]`.
   - The device status in the database is automatically set to `offline` to simulate critical downtime.
6. Acknowledge and resolve the incident via the Swagger `PATCH` endpoint, then observe:
   - The device status automatically restores to `online` in the database once the incident is resolved.
   - The audit log correctly maps the transition states (`{"old_status": "OPEN", "new_status": "RESOLVED"}`) inside the `incident_logs` table.

---

If you want me to add specific curl examples for creating users, obtaining JWTs, or a short demo script that wires everything together, tell me which flows you prefer and I will add them to the README.

