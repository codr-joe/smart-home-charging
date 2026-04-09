# Local Development Guide

This guide explains how to run and test the Smart Charging application on your local machine without a physical P1 meter or Kubernetes cluster.

## Prerequisites

Make sure the following tools are installed before you begin.

| Tool | Minimum version | Why |
|------|----------------|-----|
| [Go](https://go.dev/dl/) | 1.23 | API server and P1 mock |
| [Node.js](https://nodejs.org/) | 22 | SvelteKit frontend |
| [Docker](https://docs.docker.com/get-docker/) | 24 | TimescaleDB container |

## First-time setup

Run the following command from the **repository root** once to copy the example environment files and install frontend dependencies:

```sh
make setup
```

This creates `src/api/.env` and `src/web/.env` from the provided `.env.example` templates. Review each file and adjust any values if needed (for example, if a port is already in use on your machine).

## Running the stack

Open four terminal windows from the repository root and run one command per window in the order listed below.

### 1. Database

```sh
make dev-db
```

Starts a TimescaleDB container (PostgreSQL 16 with the TimescaleDB extension) on port **5432**. The command waits until the database is healthy before returning.

### 2. P1 meter mock

```sh
make mock-p1
```

Starts a lightweight HTTP server on port **8090** that simulates the HomeWizard P1 meter API. It serves `GET /api/v1/data` with a realistic sine-wave solar production profile — peaking at roughly 3 000 W around "solar noon" — so the dashboard displays meaningful, varying data.

> If you have a real HomeWizard P1 meter on your local network, set `P1_METER_URL` in `src/api/.env` to its LAN address (e.g. `http://192.168.1.100`) and skip this step.

### 3. API server

```sh
make dev-api
```

Builds and starts the Go backend on port **8080**. On startup it runs the database migrations automatically. The server polls the P1 meter (real or mock) every 10 seconds, persists readings to TimescaleDB, and broadcasts live updates to connected WebSocket clients.

### 4. Frontend

```sh
make dev-web
```

Starts the SvelteKit development server with hot module replacement. Open [http://localhost:5173](http://localhost:5173) in your browser to view the dashboard.

## Running the tests

### All tests

```sh
make test
```

### API (Go) tests only

```sh
make test-api
```

The Go tests do not require a running database or P1 meter; external dependencies are replaced by in-memory mocks.

### Frontend (Vitest) tests only

```sh
make test-web
```

## Linting

```sh
make lint        # run all linters
make lint-api    # go vet
make lint-web    # prettier + eslint
```

## Environment variables reference

### `src/api/.env`

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://smartcharging:smartcharging@localhost:5432/smartcharging` | PostgreSQL connection string |
| `P1_METER_URL` | `http://localhost:8090` | URL of the HomeWizard P1 meter (or mock) |
| `LISTEN_ADDR` | `:8080` | Address the API server binds to |

### `src/web/.env`

| Variable | Default | Description |
|----------|---------|-------------|
| `API_BASE_URL` | `http://localhost:8080` | API base URL used by SvelteKit server-side load functions |

## Stopping the stack

Press `Ctrl+C` in each terminal to stop the frontend, API, and mock. To stop and remove the database container:

```sh
make dev-db-stop
```

## Port overview

| Port | Service |
|------|---------|
| 5432 | TimescaleDB |
| 8080 | Go API |
| 8090 | P1 meter mock |
| 5173 | SvelteKit dev server |
