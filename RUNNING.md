# How to run the application (development)

This document contains the minimal steps to run the backend locally.

## 1) Prerequisites
- Go 1.26+ installed
- PostgreSQL 14+ available
- (Optional) `npm` or `websocat` to test the WebSocket

## 2) Configure environment variables
Copy the example file and edit it with your credentials:

Windows CMD:
```cmd
copy .env.example .env
```
PowerShell / Bash:
```bash
cp .env.example .env
```

Edit `.env` and set at least:
- `JWT_SECRET` (secure random string)
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`

By default `DB_NAME` is typically `network_control`.

## 3) Create the database
Using `psql`:

```bash
psql -U postgres -c "CREATE DATABASE network_control;"
```

Or with `createdb`:

```bash
createdb -U postgres network_control
```

(Adjust user/host/port according to your environment.)

## 4) Install Go dependencies

```bash
go mod tidy
```

## 5) Migrations
The migrations for `users`, `devices`, `alerts` and `incidents` run automatically when the server starts. Ensure the database configured in `.env` is reachable before starting.

## 6) Run in development

```bash
go run ./cmd/server
```

The server listens by default on `http://localhost:8080`.

## 7) Build a binary (optional)

```bash
go build -o bin/server ./cmd/server
# Windows
.\bin\server
# Linux / Mac
./bin/server
```

## 8) Swagger UI
- Swagger UI: http://localhost:8080/swagger/index.html
- Regenerate the spec (if you edit annotations):

```bash
go run github.com/swaggo/swag/cmd/swag@v1.16.4 init -g cmd/server/main.go -o docs --parseDependency --parseInternal
make swagger
```

## 9) Get a JWT (quick example)
1) Register a user (if none exists):

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"operator@noc.local","password":"securepass","role":"operator"}'
```

2) Login and extract the token:

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"operator@noc.local","password":"securepass"}'
```

The response includes `token`; use it as `Authorization: Bearer <token>`.

## 10) Connect to the WebSocket (requires JWT)

- wscat:

```bash
wscat -H "Authorization: Bearer <JWT>" -c ws://localhost:8080/api/v1/monitoring/ws
```

- websocat:

```bash
websocat -H "Authorization: Bearer <JWT>" ws://localhost:8080/api/v1/monitoring/ws
```

You will receive JSON events of type `metric`, `device_status`, `alert`, and `incident` while the monitoring engine is running.

## 11) Tests
- Run all tests:

```bash
go test ./...
```

- Run only monitoring tests:

```bash
go test ./internal/monitoring -run Test
```

---
