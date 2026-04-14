# Smart Charging

A smart EV charging tool that uses real-time solar energy data from a HomeWizard P1 meter to maximize self-consumption of solar power. The application monitors energy production and grid flow, then provides a live dashboard with insights to help charge your electric vehicle with excess solar energy.

## Architecture

The system consists of three components:

| Component | Technology | Purpose |
|---|---|---|
| API | Go | Polls P1 meter, persists readings to TimescaleDB, streams live data via WebSocket |
| Web | SvelteKit (Svelte 5) | Responsive dashboard with real-time energy chart and dark mode |
| Database | TimescaleDB (PostgreSQL 16) | Time-series storage for energy readings |

Traffic is routed through Traefik using the Kubernetes Gateway API. TLS certificates are managed by Cert-Manager with Let's Encrypt.

## Repository Structure

```
repo/
├── .github/          # GitHub workflows and instructions
├── argo-cd/          # Argo CD Application manifests (GitOps)
├── docs/             # Documentation (architecture, deployment, local dev)
├── helm/             # Helm charts for each component
│   ├── api/
│   ├── db/
│   └── web/
├── src/              # Application source code
│   ├── api/          # Go API server
│   └── web/          # SvelteKit frontend
├── tests/            # Integration / build tests
├── docker-compose.yml
└── Makefile
```

## Prerequisites

| Tool | Minimum version |
|------|----------------|
| Go | 1.23 |
| Node.js | 22 |
| Docker | 24 |

## Quick Start

```sh
# Copy env files and install frontend dependencies
make setup

# Start the database
make dev-db

# Start the P1 meter mock (or use a real HomeWizard P1 meter)
make mock-p1

# Start the API server
make dev-api

# Start the frontend dev server
make dev-web
```

Open [http://localhost:5173](http://localhost:5173) to view the dashboard.

## Running Tests

```sh
make test        # all tests
make test-api    # Go unit tests
make test-web    # frontend unit tests (Vitest)
```

## Deployment

The application is deployed on a bare-metal Kubernetes cluster via Argo CD using a GitOps workflow. See [docs/deployment.md](docs/deployment.md) for the full guide.

## Documentation

- [Local Development Guide](docs/local-development.md)
- [Deployment Guide](docs/deployment.md)
- [Technical Architecture](docs/analysis.md)
