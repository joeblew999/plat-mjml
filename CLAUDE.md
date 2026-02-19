# plat-mjml — AI Context

## Project

MJML email template platform built on **go-zero** framework. Three servers (MCP + Web UI + REST API) running as a `service.ServiceGroup`.

## go-zero AI Resources

- **zero-skills**: `.claude/skills/zero-skills/` — comprehensive go-zero patterns (auto-loaded by Claude Code)
- **ai-context**: `.claude/ai-context/` — quick workflows, patterns, and tool docs

When working with this project, follow go-zero conventions:
- Handler → Logic → Model three-layer architecture
- Use `goctl` for code generation (never hand-write handlers/types)
- Config via `conf.MustLoad` with YAML files
- Pass `ctx` through all layers
- Use `errorx` for API errors, not `fmt.Errorf`
- ServiceContext for dependency injection

## Key Commands

```bash
task dev          # Start server (MCP + UI + API)
task gen          # Regenerate handlers/types from .api file
task gen:model    # Regenerate DB models from schema
task check        # lint + test + build
task build        # Build server + CLI binaries
```

## Architecture

```
api/plat-mjml.api          # API spec (source of truth for REST endpoints)
etc/plat-mjml.yaml         # Config file (go-zero standard)
cmd/
  server/main.go           # Server entry point (MCP + UI + API orchestration)
  server/mcp.go            # MCP tool registration
  cli/main.go              # CLI tool entry point
internal/
  config/config.go         # Config struct (go-zero standard location)
  handler/                 # HTTP handlers (goctl generated — DO NOT EDIT)
  logic/                   # Business logic (goctl generated — safe to edit)
  model/                   # DB models (goctl generated from schema/)
  svc/servicecontext.go    # Dependency injection (holds Config + deps)
  types/types.go           # Request/response types (goctl generated)
  ui/                      # Datastar web UI handlers
  errorx/                  # Error handling
pkg/
  mjml/                    # MJML template renderer
  delivery/                # Email delivery engine
  queue/                   # Job queue (SQLite-backed)
  mail/                    # SMTP sender
  font/                    # Google Fonts manager
  db/                      # Database wrapper
schema/                    # SQL table definitions
templates/                 # MJML email templates
```

## Conventions

- `.api` file is the source of truth — edit it, then `task gen`
- `internal/handler/` and `internal/types/` are generated — never edit manually
- `internal/logic/` is generated once then safe to edit
- SQLite via `modernc.org/sqlite` (no CGO)
- go-zero `core/metric` for Prometheus metrics
- go-zero `core/syncx.SingleFlight` for render dedup
- go-zero `core/collection.Cache` for LRU+TTL caching
