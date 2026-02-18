# Event-Driven Gateway Service — Implementation Plan

## Context

Build an OpenClaw-inspired event-driven gateway in Go. The gateway receives webhooks (Grafana alerts initially), wraps them in event envelopes, forwards to a downstream agent runtime via HTTP, and returns the response. It is intentionally a **thin proxy** — no planning, skill execution, or memory management. Lives as a new Gas Town rig called `claudepod`.

## Setup

1. **Create GitHub repo** `youmna-rabie/claude-pod` via `gh repo create`
2. **Register rig**: `gt rig add claudepod <repo-url> --prefix cp`
3. **Work in** `~/gt/claudepod/mayor/rig/`

## Architecture

```
Webhook Source (Grafana) → [Channel Adapter] → Event Store → [Agent Client] → Agent Runtime
                                ↑                                  ↓
                           HTTP Server ←──── Admin API ──── Response back
```

## Design Decisions

| Decision | Choice | Why |
|----------|--------|-----|
| HTTP router | `go-chi/chi` | Lightweight, idiomatic, middleware support |
| Config | YAML via `gopkg.in/yaml.v3` | Human-readable, spec-recommended |
| CLI | `spf13/cobra` | Standard Go CLI framework |
| Logging | `log/slog` (stdlib) | No external dep, Go 1.21+ |
| Event IDs | `google/uuid` | Standard UUIDs |
| SQLite (Phase 2) | `modernc.org/sqlite` | Pure Go, no CGO |
| Testing | `testing` + `httptest` | Stdlib, no framework |

## File Tree

```
cmd/gateway/main.go
internal/
  config/config.go, config_test.go
  types/event.go, channel.go, skill.go
  channel/grafana.go, grafana_test.go, dummy.go, dummy_test.go
  event/store.go, memory.go, memory_test.go, sqlite.go (P2), sqlite_test.go (P2)
  skill/registry.go, registry_test.go
  agent/client.go, stub.go, client_test.go
  admin/handler.go, handler_test.go
  server/server.go, server_test.go, middleware.go
  cli/root.go, run.go, channels.go, events.go, skills.go
Makefile
gateway.yaml.example
README.md
```

## Phase 1 — Skeleton & Core Domain

Build order (each step testable independently):

### 1. Project scaffolding
- `go mod init github.com/youmna-rabie/claude-pod`
- Makefile (build, test, run, clean)
- `gateway.yaml.example` config

### 2. Config loader (`internal/config/`)
- `Config` struct: Server, Agent, Channels, Skills, Store, Logging sections
- `Load(path) (*Config, error)` — YAML parsing + defaults + validation
- Tests: valid YAML, missing fields, env var override for secrets

### 3. Core types (`internal/types/`)
- `Event`: ID, ChannelID, RawBody, Headers, Timestamp, Status
- `EventEnvelope`: Version, Event, Channel, Skills, Timestamp
- `Channel` interface: Name(), ValidateRequest(r), ParseRequest(r) → Event
- `Skill` struct: Name, Description, Path

### 4. In-memory event store (`internal/event/`)
- `Store` interface: Save, Get, List, UpdateStatus, Count
- `MemoryStore`: ring buffer, `sync.RWMutex`, map index for O(1) Get
- Tests: save/get roundtrip, eviction at capacity, list newest-first, concurrent access

### 5. Channel implementations (`internal/channel/`)
- **DummyChannel**: accepts any POST, wraps in Event (for testing)
- **GrafanaChannel**: validates Content-Type, auth token, parses JSON body, 1MB limit
- Tests: valid/invalid payloads, auth validation, size limits

### 6. Stub agent client (`internal/agent/`)
- `Client` interface: Forward(envelope) → (Response, error)
- `StubClient`: logs via slog, returns `{"status":"ok","event_id":"..."}`

### 7. Skill registry (`internal/skill/`)
- Scans directories for `SKILL.md` files, parses YAML frontmatter (name, description)
- Allowlist filtering from config
- Tests: scan, parse, filtering

### 8. HTTP server (`internal/server/`)
- chi router with middleware (RequestID, RealIP, logging, recovery)
- `POST /webhooks/{channel}` — validate → parse → store → forward → respond
- `GET /health` — liveness check
- `GET /admin/events`, `/admin/channels`, `/admin/skills` — JSON listings
- Tests: full pipeline via httptest

### 9. CLI (`internal/cli/`)
- `gateway run --config gateway.yaml` — starts server
- `gateway list-channels` — prints configured channels
- `gateway list-events --limit 20` — prints recent events
- `gateway list-skills` — prints registered skills

### 10. Entrypoint (`cmd/gateway/main.go`)
- Calls `cli.Execute()`

## Phase 2 — Persistence & Admin

- **SQLite event store**: same `Store` interface, schema with indexes on channel/timestamp/status
- **Admin stats endpoint**: `GET /admin/stats` — counts by channel/status, uptime
- **Channel health tracking**: last event time, event/error counts per channel
- **Per-channel rate limiting**: token bucket middleware, configurable in YAML

## Phase 3 — Agent Runtime Integration & Hardening

- **Real HTTP agent client**: configurable endpoint, timeout, exponential backoff retries
- **Structured logging**: slog throughout, DEBUG/INFO/WARN/ERROR levels
- **Metrics endpoint**: `GET /metrics` — request counts, latency percentiles, error rates
- **E2E acceptance tests**: mock agent runtime + gateway + real webhook payloads
- **README**: quick start, config reference, API docs, adding new channels

## Verification

After each phase:
```bash
make test                    # all unit tests pass
make build                   # binary compiles
./gateway run --config gateway.yaml  # server starts

# Phase 1 smoke test:
curl -X POST http://localhost:8080/webhooks/dummy \
  -H "Content-Type: application/json" \
  -d '{"test": true}'

curl -X POST http://localhost:8080/webhooks/grafana \
  -H "Content-Type: application/json" \
  -d '{"status":"firing","alerts":[{"labels":{"alertname":"HighCPU"}}]}'

curl http://localhost:8080/admin/events
curl http://localhost:8080/admin/channels
curl http://localhost:8080/admin/skills
```

## Execution Strategy

Sling Phase 1 steps to polecats in parallel where possible, or implement directly given this is a greenfield project. Each phase ends with a commit + push + phase summary.
