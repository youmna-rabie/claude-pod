# claude-pod

An event-driven gateway service written in Go. Receives webhooks from external sources (Grafana alerts, etc.), wraps them in event envelopes, forwards them to a downstream agent runtime via HTTP, and returns the response. Intentionally a **thin proxy** — no planning, skill execution, or memory management happens in the gateway itself.

## Quick Start

```bash
# Clone
git clone https://github.com/youmna-rabie/claude-pod.git
cd claude-pod

# Build
make build

# Configure
cp config.example.yaml gateway.yaml
# Edit gateway.yaml to set your agent URL, channels, etc.

# Run
./bin/gateway run --config gateway.yaml

# Test with curl
curl -X POST http://localhost:8080/webhooks/dummy \
  -H "Content-Type: application/json" \
  -d '{"test": true}'

curl http://localhost:8080/health
curl http://localhost:8080/admin/events
```

## Gas Town Setup

claude-pod is designed to run as a [Gas Town](https://github.com/youmna-rabie/gastown) rig. The repo already includes `.claude/`, `.beads/`, and `.gastown/` scaffolding.

### Prerequisites

- [Gas Town](https://github.com/youmna-rabie/gastown) (`gt` CLI)
- [Beads](https://github.com/youmna-rabie/beads) (`bd` CLI)
- [Dolt](https://www.dolthub.com/repositories) (version-controlled database for beads)

### Adopt as a Rig

```bash
gt rig add claudepod https://github.com/youmna-rabie/claude-pod.git --adopt --prefix cp
```

This registers claude-pod as a rig in your Gas Town workspace. The `cp` prefix routes beads commands (e.g., `bd show cp-xxxx`) to this rig's issue tracker.

The scaffolding files are already in the repo:
- `.gastown/config.json` — rig identity and configuration
- `.beads/config.yaml` — issue tracking prefix (`cp`)
- `.claude/settings.json` — Claude Code hooks for session management

## Configuration

Configuration is YAML-based. See [`config.example.yaml`](config.example.yaml) for a complete example.

```yaml
server:
  host: "0.0.0.0"        # Bind address
  port: 8080              # Listen port

agent:
  url: "http://localhost:3000"  # Downstream agent runtime URL
  timeout: 30s                  # Forward request timeout

channels:                 # Webhook channel adapters
  - name: dummy           # Channel name (used in URL path)
    type: dummy            # Adapter type
  - name: grafana
    type: grafana
    auth: ""               # Bearer token (supports ${ENV_VAR} expansion)

skills:
  dirs:                    # Directories to scan for SKILL.md files
    - "./skills"
  allowlist: []            # Empty = allow all discovered skills

store:
  type: memory             # Event store backend (memory only for now)
  capacity: 1000           # Ring buffer size

logging:
  level: "info"            # debug, info, warn, error
  format: "json"           # json or text
```

### Environment Variable Expansion

Use `${VAR_NAME}` syntax in config values for secrets:

```yaml
channels:
  - name: grafana
    type: grafana
    auth: "${GRAFANA_AUTH_TOKEN}"
```

## API Endpoints

| Method | Route | Description |
|--------|-------|-------------|
| `POST` | `/webhooks/{channel}` | Receive webhook, parse, store, forward to agent |
| `GET` | `/health` | Liveness check — returns `{"status":"ok"}` |
| `GET` | `/admin/events` | List recent events (up to 50, newest first) |
| `GET` | `/admin/channels` | List configured channel names |
| `GET` | `/admin/skills` | List registered skills |

### Webhook Pipeline

`POST /webhooks/{channel}` processes requests through this pipeline:

1. **Resolve** — Look up channel adapter by name (404 if unknown)
2. **Validate** — Channel-specific validation: method, content-type, auth (400 on failure)
3. **Parse** — Extract event from request body
4. **Store** — Save event to the event store
5. **Forward** — Wrap in `EventEnvelope` with skills metadata, send to agent (502 on failure)
6. **Respond** — Return agent response as JSON

## Adding a Channel

Implement the `types.Channel` interface:

```go
type Channel interface {
    Name() string
    ValidateRequest(r *http.Request) error
    ParseRequest(r *http.Request) (*Event, error)
}
```

Then register it in `internal/cli/run.go` inside `buildChannels()`. See `internal/channel/dummy.go` for a minimal example or `internal/channel/grafana.go` for one with auth and size limits.

### Built-in Channels

- **dummy** — Accepts any POST body. No auth. For testing and development.
- **grafana** — Validates Content-Type (`application/json`), optional Bearer token auth, 1MB body size limit. For Grafana alert webhooks.

## Project Structure

```
claude-pod/
├── cmd/gateway/
│   └── main.go              # Entrypoint — calls cli.Execute()
├── internal/
│   ├── agent/
│   │   ├── client.go        # Client interface (Forward)
│   │   └── stub.go          # Stub implementation for dev/test
│   ├── channel/
│   │   ├── dummy.go         # Dummy channel adapter
│   │   └── grafana.go       # Grafana webhook adapter
│   ├── cli/
│   │   ├── root.go          # Cobra root command
│   │   ├── run.go           # `gateway run` — starts the server
│   │   ├── channels.go      # `gateway list-channels`
│   │   ├── events.go        # `gateway list-events`
│   │   └── skills.go        # `gateway list-skills`
│   ├── config/
│   │   └── config.go        # YAML config loader with validation
│   ├── event/
│   │   ├── store.go         # Store interface
│   │   └── memory.go        # In-memory ring buffer implementation
│   ├── server/
│   │   ├── server.go        # HTTP server, routes, handlers
│   │   └── middleware.go     # RequestID, Logging, Recovery
│   ├── skill/
│   │   └── registry.go      # SKILL.md discovery and parsing
│   └── types/
│       ├── event.go         # Event, EventEnvelope, EventStatus
│       ├── channel.go       # Channel interface
│       └── skill.go         # Skill struct
├── docs/
│   └── PLAN.md              # Implementation plan (Phase 1-3)
├── config.example.yaml      # Example configuration
├── Makefile                  # Build automation
├── go.mod
└── go.sum
```

## Development

```bash
make build    # Build binary to bin/gateway
make test     # Run all tests
make lint     # Run golangci-lint
make run      # Build and run
make clean    # Remove build artifacts
```

### CLI Commands

```bash
gateway run --config gateway.yaml    # Start the server
gateway list-channels                # Show configured channels
gateway list-events --limit 20       # Show recent events
gateway list-skills                  # Show discovered skills
```

### Skills

Skills are discovered from `SKILL.md` files in configured directories. Each file uses YAML frontmatter:

```markdown
---
name: summarize
description: Summarize text using Claude
---

# Implementation details...
```

Skills are included as metadata in event envelopes forwarded to the agent — the gateway does not execute them.

## Architecture

```
Webhook Source (e.g. Grafana)
        │
        ▼
  POST /webhooks/{channel}
        │
        ▼
  ┌─────────────┐
  │   Channel    │  Validate request, parse body
  │   Adapter    │
  └──────┬──────┘
         │
         ▼
  ┌─────────────┐
  │   Event      │  Ring buffer, O(1) lookups by ID
  │   Store      │
  └──────┬──────┘
         │
         ▼
  ┌─────────────┐
  │   Agent      │  Forward EventEnvelope via HTTP
  │   Client     │
  └──────┬──────┘
         │
         ▼
   Agent Runtime
```

The middleware stack applies to all routes: `Recovery → Logging → RequestID → Router`.

## Roadmap

See [`docs/PLAN.md`](docs/PLAN.md) for the full implementation plan.

- **Phase 1** (complete) — Skeleton and core domain: config, types, channels, event store, skill registry, HTTP server, CLI
- **Phase 2** — Persistence and admin: SQLite event store, admin stats, channel health tracking, per-channel rate limiting
- **Phase 3** — Agent integration and hardening: real HTTP agent client with retries, structured logging, metrics endpoint, E2E tests
