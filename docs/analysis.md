# Smart Charging — Technical Architecture & Approach

## 1. Context Summary

The application must:
- Ingest real-time energy data from a **P1-meter** (production + grid consumption/injection)
- Calculate the available **excess solar energy**
- Dynamically adjust the **EV charging rate** based on that excess
- Expose a **responsive web dashboard** with dark mode (Tailwind CSS v4)
- Run on a **bare-metal Kubernetes cluster** behind Traefik + Gateway API + Let's Encrypt

Design principles: KISS, YAGNI, DRY, SOLID. Security is non-negotiable at every layer.

---

## 2. High-Level Architecture

The system decomposes into four bounded concerns:

```
┌──────────────────────────────────────────────────────────┐
│                        Browser                           │
│                  (SvelteKit Dashboard)                   │
└─────────────────────────┬────────────────────────────────┘
                          │ HTTPS / WebSocket
┌─────────────────────────▼────────────────────────────────┐
│                     API Gateway                          │
│            (Traefik + Kubernetes Gateway API)            │
└────────┬──────────────────────────────────┬──────────────┘
         │                                  │
┌────────▼────────┐               ┌─────────▼──────────────┐
│   Backend API   │               │   Charging Controller  │
│  (Go / Fiber)   │               │    (Go / goroutines)   │
└────────┬────────┘               └─────────┬──────────────┘
         │                                  │
┌────────▼────────┐               ┌─────────▼──────────────┐
│  TimescaleDB    │               │     EV Charger          │
│  (PostgreSQL)   │               │  (OCPP / Modbus / API) │
└─────────────────┘               └────────────────────────┘
         ▲
         │ MQTT / HTTP push
┌────────┴────────┐
│    P1-Meter     │
│  (DSMR reader)  │
└─────────────────┘
```

---

## 3. P1-Meter Integration

The P1-meter reads from the DSMR (Dutch Smart Meter Requirements) telegram on the physical P1 port of the smart meter.

### Options

| Approach | Description | Pros | Cons |
|---|---|---|---|
| **USB/Serial direct read** | Connect P1 cable to a Pi/NUC, parse DSMR telegram locally | No cloud dependency, raw access, free | Requires hardware near meter, serial parsing complexity |
| **HWi P1 Meter (HomeWizard)** | Commercial device with HTTP/JSON REST API | Easy integration, no parsing needed, LAN-only | Vendor lock-in, costs €40–80 |
| **MQTT broker (e.g., via ESPHome/Tasmota)** | Flash ESP8266/ESP32 with DSMR firmware, publish to MQTT | Well-supported in home automation, decoupled | Requires extra hardware flashing |

**Recommended**: HomeWizard P1 Meter with HTTP polling or its push webhook feature, falling back to a local MQTT bridge if independence from vendor is required. This satisfies KISS without over-engineering a serial parser.

### Data Points Extracted

From the DSMR telegram / HomeWizard API:
- `power_w` — net grid power (positive = consuming, negative = injecting)
- `solar_power_w` — inverter production (if inverter also exposes API, e.g., SolarEdge/Fronius)
- `tariff` (T1/T2) — for cost calculations

**Excess solar calculation**:

$$P_{\text{excess}} = P_{\text{solar}} - P_{\text{home\_consumption}}$$

Where $P_{\text{home\_consumption}} = P_{\text{solar}} + P_{\text{grid\_import}} - P_{\text{grid\_export}}$.

If the meter only exposes net grid power, then:

$$P_{\text{excess}} = -P_{\text{grid\_net}} \quad \text{(when negative = injecting)}$$

---

## 4. Backend

### Language Choice: Go vs TypeScript (Node.js) vs Python

| | Go | TypeScript (Node.js) | Python |
|---|---|---|---|
| **Performance** | Excellent, low memory footprint on k8s | Good, event loop model | Moderate |
| **Concurrency** | Native goroutines — ideal for polling loops + API serving | Async/await, single-threaded | asyncio, GIL limitations |
| **Ecosystem for IoT/energy** | Limited but sufficient | Moderate | Rich (paho-mqtt, pymodbus) |
| **Full-stack consistency** | No (frontend is JS) | Yes (shared types possible) | No |
| **Binary size / k8s pod size** | Very small (~10 MB image) | Larger (Node runtime) | Large |
| **Developer velocity** | Moderate (verbose) | High | High |

**Recommendation: Go** with the [Fiber](https://gofiber.io/) HTTP framework.

**Rationale**: The charging controller runs a tight polling/adjustment loop (every ~5–15 seconds). Go's goroutines handle this cleanly alongside the REST API without needing a separate process. The resulting container image is minimal, which matters on a bare-metal cluster with limited resources. The verbosity cost is acceptable given the limited surface area of this application.

**Pros of Go here**:
- Single static binary, tiny Docker image (~15 MB with scratch base)
- Native goroutines make the polling loop + HTTP server co-exist trivially
- Strong type system catches integration bugs at compile time

**Cons of Go here**:
- No shared type definitions with the SvelteKit frontend (mitigated with OpenAPI code generation)
- More boilerplate for simple CRUD than TypeScript or Python

### API Design

REST over WebSocket for CRUD (energy records, configuration). WebSocket or SSE for live dashboard streaming.

| Endpoint pattern | Protocol | Purpose |
|---|---|---|
| `GET /api/v1/energy/current` | REST | Current P1 reading |
| `GET /api/v1/energy/history` | REST | Historical data with time range params |
| `GET /api/v1/charging/status` | REST | Current charger state and power |
| `PUT /api/v1/charging/config` | REST | Update charging config (min/max current, mode) |
| `ws://api/v1/stream` | WebSocket | Live energy + charger telemetry push |

**Pros of REST + WebSocket split**:
- REST is cacheable, well-understood, easy to test
- WebSocket only where real-time push is genuinely needed

**Cons**:
- Two connection types to manage on the frontend
- Requires WebSocket support in Traefik (supported via Gateway API `BackendTLSPolicy`)

### Security

- JWT-based authentication (short-lived access tokens + refresh tokens stored in HttpOnly cookies)
- No API keys in query strings
- Input validation on all REST endpoints using Go struct tags + a validation library (e.g., `go-playground/validator`)
- Rate limiting via Traefik middleware to prevent brute force on the auth endpoint

---

## 5. Database

### Time-Series Data is the Core Problem

Energy readings come in at high frequency (every 5–15 seconds). Naive Postgres will degrade with millions of rows without partitioning.

### Options

| | TimescaleDB | InfluxDB | Plain PostgreSQL | SQLite |
|---|---|---|---|---|
| **Time-series optimized** | Yes (hypertables = auto partitioning) | Yes (purpose-built) | No | No |
| **SQL support** | Full PostgreSQL SQL | Flux/InfluxQL (proprietary) | Full SQL | Full SQL |
| **Kubernetes deployment** | Standard Postgres image + extension | Separate Helm chart | Standard Postgres image | File-based, no k8s complexity |
| **Operational complexity** | Low (it's just Postgres) | Moderate | Low | Very low |
| **Data retention policies** | Built-in continuous aggregates + retention | Built-in | Manual partitioning | Manual |
| **Horizontal scaling** | Via Timescale Cloud (not bare-metal friendly) | Yes | Via Citus | No |

**Recommendation: TimescaleDB**

**Pros**:
- It is PostgreSQL — the same tooling, drivers, and SQL knowledge apply
- Hypertables automatically partition by time, keeping query performance predictable
- Continuous aggregates (materialized views that auto-refresh) enable fast hourly/daily rollup queries without separate ETL jobs
- Data retention policies can automatically drop raw data older than, say, 90 days while keeping aggregates

**Cons**:
- Requires the TimescaleDB extension, which means a custom Postgres image or the official `timescale/timescaledb` image
- Slightly more complex initial schema setup compared to plain Postgres

### Schema Sketch

```sql
-- Hypertable partitioned by time
CREATE TABLE energy_readings (
  time        TIMESTAMPTZ         NOT NULL,
  power_w     DOUBLE PRECISION    NOT NULL,  -- net grid power
  solar_w     DOUBLE PRECISION,              -- solar production if available
  tariff      CHAR(2)                        -- T1 or T2
);
SELECT create_hypertable('energy_readings', 'time');

-- Charger sessions
CREATE TABLE charging_sessions (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  started_at  TIMESTAMPTZ NOT NULL,
  ended_at    TIMESTAMPTZ,
  energy_kwh  DOUBLE PRECISION,
  avg_power_w DOUBLE PRECISION
);
```

---

## 6. EV Charger Integration

### Protocol Options

| Protocol | Description | Pros | Cons |
|---|---|---|---|
| **OCPP 1.6 / 2.0.1** | Open standard for EV charger communication | Vendor-agnostic, widely supported (Easee, Alfen, Zaptec) | Requires running an OCPP server (Central System), more complex |
| **Modbus TCP/RTU** | Industrial register-based protocol | Low-level control, deterministic | Charger must expose Modbus, wiring complexity |
| **Proprietary REST API** | Vendor-specific (e.g., Easee API, go-e API) | Simple HTTP calls | Vendor lock-in, cloud dependency for some |
| **go-e Charger local API** | HTTP API on local network | No cloud dependency, simple | go-e specific |

**Recommendation**: Abstract behind a `ChargerAdapter` interface in Go:

```go
type ChargerAdapter interface {
    GetStatus(ctx context.Context) (ChargerStatus, error)
    SetChargingCurrent(ctx context.Context, amperes int) error
    StopCharging(ctx context.Context) error
}
```

Implement one concrete adapter for the charger you own (e.g., `EaseeAdapter`, `GoEAdapter`). This follows the Interface Segregation and Dependency Inversion principles from SOLID, and means swapping chargers requires only a new adapter.

**Pros of this abstraction**:
- Clean separation of concerns
- Easy to test with a `MockChargerAdapter`
- Supports future charger changes without touching the controller logic

**Cons**:
- Slightly more upfront design work (justified by SOLID)

---

## 7. Charging Control Algorithm

The core loop runs as a goroutine:

```
Every N seconds:
  1. Fetch latest P1 reading (excess power in watts)
  2. Fetch current charger status (current in A, phases)
  3. Calculate target current:
     - excess_amps = excess_watts / (phases × 230V)
     - clamp to [min_current, max_current] (e.g., 6A–16A for single-phase)
     - apply hysteresis: only change if delta > 1A (avoid oscillation)
  4. If target_current != current_current:
     - Call charger adapter SetChargingCurrent(target)
  5. Persist reading to TimescaleDB
```

**Key considerations**:
- **Hysteresis**: Without it, cloud cover causes rapid oscillation of the charging current, shortening charger relay life.
- **Minimum current**: Most chargers require at least 6A to remain in a charging state. Below this the session should be paused, not set to 0A.
- **Phase awareness**: Single-phase vs three-phase changes the watt-to-amp conversion factor.

---

## 8. Frontend

### Framework Choice: SvelteKit vs Next.js vs Astro

| | SvelteKit | Next.js | Astro |
|---|---|---|---|
| **Bundle size** | Tiny (no virtual DOM) | Larger | Minimal (island architecture) |
| **Real-time WebSocket** | Native, straightforward | Requires client component boilerplate | Limited, designed for static content |
| **SSR** | Yes | Yes | Yes (but static-first) |
| **Tailwind CSS v4 support** | Full | Full | Full |
| **Learning curve** | Low-moderate | Moderate | Low-moderate |
| **Dashboard suitability** | Excellent | Good | Poor (static-first) |

**Recommendation: SvelteKit**

**Pros**:
- Svelte compiles to vanilla JS — no runtime framework overhead, ideal for a dashboard that must feel fast
- WebSocket stores fit naturally into Svelte's reactive store pattern
- Smaller bundle means faster initial load, which matters if the dashboard is accessed from a phone at the car
- Full SSR with `+page.server.ts` for the initial data load, then WebSocket takes over

**Cons**:
- Smaller ecosystem than React/Next.js
- Fewer ready-made UI component libraries (though Tailwind CSS eliminates most of this need)

### Tailwind CSS v4

Tailwind v4 drops the `tailwind.config.js` file in favor of CSS-first configuration:

```css
/* app.css */
@import "tailwindcss";

@theme {
  --color-primary: oklch(0.6 0.2 250);
  --color-surface: oklch(0.98 0 0);
}
```

Dark mode is handled via the `dark` variant which in v4 defaults to `@media (prefers-color-scheme: dark)` or can be switched to class-based for a manual toggle:

```css
@variant dark (&:where(.dark, .dark *));
```

### Real-Time Dashboard

The dashboard receives live data over WebSocket. Svelte stores bridge the connection:

```typescript
// src/lib/stores/energy.ts
import { readable } from 'svelte/store';

export const energyStream = readable<EnergyReading | null>(null, (set) => {
  const ws = new WebSocket('wss://api.example.com/v1/stream');
  ws.onmessage = (e) => set(JSON.parse(e.data));
  return () => ws.close();
});
```

Reconnection logic (exponential backoff) must be implemented to handle WebSocket drops gracefully.

---

## 9. Repository & Monorepo Structure

Following the required folder structure extended for the actual content:

```
repo/
├── .github/
│   ├── workflows/
│   │   ├── ci.yml           # lint, test, build on PR
│   │   └── release.yml      # build & push images on main
│   └── instructions/
├── src/
│   ├── api/                 # Go backend
│   │   ├── cmd/server/
│   │   ├── internal/
│   │   │   ├── energy/      # P1 ingestion, domain logic
│   │   │   ├── charging/    # controller + adapters
│   │   │   ├── auth/
│   │   │   └── db/
│   │   ├── go.mod
│   │   └── Dockerfile
│   └── web/                 # SvelteKit frontend
│       ├── src/
│       ├── package.json
│       └── Dockerfile
├── infra/
│   ├── k8s/
│   │   ├── namespace.yaml
│   │   ├── api/             # Deployment, Service, HPA
│   │   ├── web/             # Deployment, Service
│   │   ├── db/              # StatefulSet, PVC
│   │   └── gateway/         # HTTPRoute, TLSRoute, Certificate
│   └── helm/                # Optional Helm chart wrappers
├── docs/
│   ├── architecture.md
│   ├── p1-meter-setup.md
│   └── charger-adapters.md
└── README.md
```

Note: `infra/` is not explicitly listed in the general instructions folder structure, but is implied by the infrastructure instructions. It is kept separate from `src/` to maintain a clean separation of application code and deployment manifests.

---

## 10. Infrastructure (Kubernetes)

### Traefik + Kubernetes Gateway API

The Gateway API (`gateway.networking.k8s.io`) is the successor to Ingress. Traefik v3 supports it natively.

```yaml
# infra/k8s/gateway/gateway.yaml
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: main-gateway
  namespace: smart-charging
spec:
  gatewayClassName: traefik
  listeners:
    - name: https
      port: 443
      protocol: HTTPS
      tls:
        mode: Terminate
        certificateRefs:
          - name: smart-charging-tls
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: api-route
spec:
  parentRefs:
    - name: main-gateway
  hostnames:
    - api.yourdomain.com
  rules:
    - backendRefs:
        - name: api-service
          port: 8080
```

```yaml
# infra/k8s/gateway/certificate.yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: smart-charging-tls
  namespace: smart-charging
spec:
  secretName: smart-charging-tls
  issuerRef:
    name: letsencrypt-production
    kind: ClusterIssuer
  dnsNames:
    - yourdomain.com
    - api.yourdomain.com
```

### Persistent Storage for TimescaleDB

On bare-metal, a `StorageClass` backed by local NVMe or a distributed storage solution (Longhorn, Rook/Ceph) must exist:

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: timescaledb
spec:
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        accessModes: [ReadWriteOnce]
        storageClassName: longhorn  # or local-path for single-node
        resources:
          requests:
            storage: 20Gi
```

**Pros of Longhorn on bare-metal**:
- Provides replicated block storage without a full Ceph cluster
- Web UI, snapshots, backups to S3

**Cons of Longhorn**:
- Adds operational complexity; for a single-node homelab, `local-path-provisioner` is simpler but has no replication

---

## 11. Testing Strategy

From the testing instructions: all CRUD operations must have test coverage. Tests must be automated and repeatable.

### Backend (Go)

- **Unit tests**: Test the charging algorithm logic in isolation using `MockChargerAdapter` and mock P1 readings
- **Integration tests**: Spin up a real TimescaleDB instance via Docker in CI, run CRUD operations against it
- **Table-driven tests**: Standard Go pattern — define input/expected output slices, range over them. Keeps tests DRY.

```go
func TestCalculateTargetCurrent(t *testing.T) {
    cases := []struct {
        name        string
        excessWatts float64
        phases      int
        want        int
    }{
        {"excess below minimum", 500, 1, 6},  // clamp to min
        {"single phase 1500W", 1500, 1, 6},
        {"single phase 3000W", 3000, 1, 13},
        {"three phase 6000W", 6000, 3, 8},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            got := calculateTargetCurrent(tc.excessWatts, tc.phases)
            assert.Equal(t, tc.want, got)
        })
    }
}
```

### Frontend (SvelteKit)

- **Component tests**: [Vitest](https://vitest.dev/) + [Testing Library](https://testing-library.com/docs/svelte-testing-library/intro/) for unit testing Svelte components
- **E2E tests**: [Playwright](https://playwright.dev/) for critical user flows (login, view dashboard, change charging config)

### CI Pipeline

```yaml
# .github/workflows/ci.yml
jobs:
  test-api:
    runs-on: ubuntu-latest
    services:
      timescaledb:
        image: timescale/timescaledb:latest-pg16
        env:
          POSTGRES_PASSWORD: test
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: go test ./... -race -coverprofile=coverage.out
        working-directory: src/api

  test-web:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
      - run: pnpm install && pnpm test
        working-directory: src/web
```

---

## 12. Key Architectural Trade-offs Summary

| Decision | Choice | Main Alternative | Why This Choice |
|---|---|---|---|
| Backend language | Go | TypeScript | Goroutines fit the polling loop, tiny k8s footprint |
| Frontend framework | SvelteKit | Next.js | Lighter bundle, reactive stores suit real-time dashboard |
| Database | TimescaleDB | InfluxDB | It's Postgres — standard tooling, SQL, familiar ops |
| P1 integration | HomeWizard HTTP API | Serial DSMR parser | KISS — avoid serial parsing complexity |
| Charger protocol | Abstracted adapter | OCPP Central System | YAGNI — start simple, promote to OCPP if needed |
| Real-time transport | WebSocket | SSE | Bidirectional (server push + client commands) |
| Kubernetes storage | Longhorn | local-path-provisioner | Replication on bare-metal without full Ceph |
| Ingress | Traefik Gateway API | Traefik IngressRoute | Gateway API is the upstream standard, future-proof |

---

## 13. Phased Delivery

To honour YAGNI and deliver value incrementally:

**Phase 1 — Monitor**: P1 ingestion → TimescaleDB → Dashboard showing real-time and historical energy data. No charger control yet.

**Phase 2 — Manual Control**: Add charger adapter + REST endpoint to manually set current. Dashboard shows charger status.

**Phase 3 — Automatic Control**: Add the charging controller loop (solar excess → target current). Configurable thresholds.

**Phase 4 — Analytics**: Cost savings calculations, CO₂ avoided, charging session history, continuous aggregate rollups.

This approach ensures a working, observable system at the end of each phase rather than a big-bang delivery.
