# ADR-001: Email Platform Architecture

## Status
Accepted

## Context
plat-mjml is currently a CLI tool for rendering MJML email templates. To make it a product, we need:
- HTTP API for integration
- MCP server for AI assistant access
- Reliable email delivery with queueing
- Management UI for templates

## Decision

### Server Framework: go-zero

Use go-zero because it provides:
- Built-in MCP server support (SSE transport)
- REST API routing
- Configuration management
- Logging and observability

```go
// Single server handles MCP + REST + UI
s := mcp.NewMcpServer(c)
s.RegisterTool(mcp.Tool{...})  // AI assistant tools
s.Start()
```

### Database: SQLite (modernc.org/sqlite)

- Zero external dependencies
- Single file backup
- Pure Go (no CGO)
- Sufficient for email platform scale

### Queue: goqite

SQLite-backed queue providing:
- 12,500+ messages/second
- Scheduled delivery
- No Redis/external dependency
- Shares SQLite database

### UI: Datastar + Templ

- Real-time updates via SSE
- Type-safe Go templates (Templ)
- XML-based (aligns with MJML)
- Future WASM compilation path

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    go-zero Server                        │
│                                                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │ MCP (SSE)   │  │  REST API   │  │  UI (SSE)   │     │
│  │ /sse        │  │  /api/v1    │  │  /ui        │     │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘     │
│         └────────────────┼────────────────┘            │
│                          │                              │
│  ┌───────────────────────┼───────────────────────┐     │
│  │                       │                       │     │
│  │  ┌─────────┐   ┌──────▼──────┐   ┌─────────┐ │     │
│  │  │Template │   │   Queue     │   │Delivery │ │     │
│  │  │ Store   │   │  (goqite)   │   │ Engine  │ │     │
│  │  └────┬────┘   └──────┬──────┘   └────┬────┘ │     │
│  │       └───────────────┼───────────────┘      │     │
│  │                ┌──────▼──────┐               │     │
│  │                │   SQLite    │               │     │
│  │                └─────────────┘               │     │
│  └───────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────┘
```

## MCP Tools

### render_template
Render MJML template to HTML with data.

```go
s.RegisterTool(mcp.Tool{
    Name:        "render_template",
    Description: "Render MJML template to HTML",
    InputSchema: mcp.InputSchema{
        Properties: map[string]any{
            "template": {"type": "string", "description": "Template slug (e.g., welcome, reset_password)"},
            "data":     {"type": "object", "description": "Template variables"},
        },
        Required: []string{"template"},
    },
    Handler: func(ctx context.Context, p map[string]any) (any, error) {
        // Render template and return HTML
    },
})
```

### send_email
Queue email for delivery.

```go
s.RegisterTool(mcp.Tool{
    Name:        "send_email",
    Description: "Queue email for delivery",
    InputSchema: mcp.InputSchema{
        Properties: map[string]any{
            "template": {"type": "string", "description": "Template slug"},
            "to":       {"type": "array", "items": {"type": "string"}, "description": "Recipient emails"},
            "subject":  {"type": "string", "description": "Email subject (optional, uses template default)"},
            "data":     {"type": "object", "description": "Template variables"},
        },
        Required: []string{"template", "to"},
    },
    Handler: func(ctx context.Context, p map[string]any) (any, error) {
        // Queue email and return job ID
    },
})
```

### list_templates
List available templates.

```go
s.RegisterTool(mcp.Tool{
    Name:        "list_templates",
    Description: "List available email templates",
    Handler: func(ctx context.Context, p map[string]any) (any, error) {
        // Return template list
    },
})
```

### get_email_status
Check email delivery status.

```go
s.RegisterTool(mcp.Tool{
    Name:        "get_email_status",
    Description: "Get status of a queued email",
    InputSchema: mcp.InputSchema{
        Properties: map[string]any{
            "id": {"type": "string", "description": "Email job ID"},
        },
        Required: []string{"id"},
    },
    Handler: func(ctx context.Context, p map[string]any) (any, error) {
        // Return email status
    },
})
```

## REST API

### Templates
```
GET    /api/v1/templates           List templates
POST   /api/v1/templates           Create template
GET    /api/v1/templates/{slug}    Get template
PUT    /api/v1/templates/{slug}    Update template
DELETE /api/v1/templates/{slug}    Delete template
POST   /api/v1/templates/{slug}/preview  Preview render
```

### Emails
```
POST   /api/v1/emails/send         Send immediately
POST   /api/v1/emails/queue        Queue for delivery
GET    /api/v1/emails/{id}         Get status
GET    /api/v1/emails              List emails
```

### System
```
GET    /api/v1/health              Health check
GET    /api/v1/stats               Queue statistics
```

## Database Schema

```sql
-- Templates
CREATE TABLE templates (
    id TEXT PRIMARY KEY,
    slug TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    content TEXT NOT NULL,
    version INTEGER DEFAULT 1,
    status TEXT DEFAULT 'draft',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Template versions (history)
CREATE TABLE template_versions (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL REFERENCES templates(id),
    version INTEGER NOT NULL,
    content TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Email queue
CREATE TABLE emails (
    id TEXT PRIMARY KEY,
    template_slug TEXT NOT NULL,
    recipients TEXT NOT NULL,
    subject TEXT NOT NULL,
    data TEXT,
    status TEXT DEFAULT 'pending',
    priority INTEGER DEFAULT 1,
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    scheduled_at DATETIME,
    sent_at DATETIME,
    error TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_emails_status ON emails(status);
CREATE INDEX idx_emails_scheduled ON emails(scheduled_at);

-- SMTP providers
CREATE TABLE smtp_providers (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    host TEXT NOT NULL,
    port INTEGER NOT NULL,
    username TEXT,
    password TEXT,
    from_email TEXT NOT NULL,
    from_name TEXT,
    is_default INTEGER DEFAULT 0
);
```

## Package Structure

```
plat-mjml/
├── main.go                  # CLI entry (existing)
├── cmd/
│   └── server/
│       └── main.go          # go-zero server
├── internal/
│   └── server/
│       ├── config.go        # Config struct
│       ├── server.go        # Server setup
│       ├── mcp.go           # MCP tool handlers
│       ├── api.go           # REST handlers
│       └── ui.go            # UI handlers
├── pkg/
│   ├── mjml/                # Existing - core renderer
│   ├── mail/                # Existing - SMTP
│   ├── font/                # Existing - fonts
│   ├── config/              # Existing - config utils
│   ├── log/                 # Existing - logging
│   ├── db/                  # NEW - SQLite wrapper
│   │   ├── db.go
│   │   └── migrations.go
│   ├── queue/               # NEW - goqite wrapper
│   │   └── queue.go
│   ├── delivery/            # NEW - delivery engine
│   │   └── engine.go
│   └── template/            # NEW - template store
│       └── store.go
├── config.yaml              # Server config
└── docs/
    └── adr/
        └── 001-email-platform-architecture.md
```

## Configuration

```yaml
# config.yaml
name: plat-mjml
host: 0.0.0.0
port: 8080

mcp:
  name: mjml-server
  messageTimeout: 30s
  cors:
    - http://localhost:3000

database:
  path: ./.data/plat-mjml.db

templates:
  dir: ./templates

delivery:
  maxRetries: 3
  retryBackoff: 5m
  maxBackoff: 4h
  rateLimit: 60

smtp:
  default: gmail
  providers:
    gmail:
      host: smtp.gmail.com
      port: 587
```

## Delivery Engine

### Queue Processing
```go
func (e *Engine) processJob(job EmailJob) error {
    // 1. Rate limit
    if err := e.rateLimiter.Wait(ctx); err != nil {
        return err
    }

    // 2. Render template
    html, err := e.renderer.RenderTemplate(job.TemplateSlug, job.Data)
    if err != nil {
        return e.handleError(job, err)
    }

    // 3. Send via SMTP
    if err := e.smtp.Send(job.Recipients, job.Subject, html); err != nil {
        return e.handleError(job, err)
    }

    // 4. Mark complete
    return e.markSent(job)
}
```

### Retry Strategy
- Exponential backoff: 5m → 10m → 20m → 40m
- Max 3 attempts by default
- Permanent failures (5xx SMTP) stop retries

## Implementation Phases

### Phase 1: MCP Server
- [ ] go-zero server setup
- [ ] render_template tool
- [ ] list_templates tool
- [ ] Test with Claude

### Phase 2: Database
- [ ] SQLite setup
- [ ] Template store CRUD
- [ ] Migrations

### Phase 3: Queue & Delivery
- [ ] goqite integration
- [ ] send_email tool
- [ ] Delivery engine
- [ ] Retry logic

### Phase 4: UI
- [ ] Templ setup
- [ ] Dashboard
- [ ] Template editor
- [ ] Queue monitor

## Testing

```bash
# Start server
go run cmd/server/main.go

# Add to Claude
claude mcp add plat-mjml -- npx -y mcp-remote http://localhost:8080/sse

# Test in Claude
"List the email templates"
"Render the welcome template with name: John"
"Send a test email to test@example.com using the welcome template"
"Check the status of email abc123"
```

## Consequences

### Positive
- Single binary deployment
- No external dependencies (Redis, PostgreSQL)
- AI-native via MCP
- Real-time UI via SSE

### Negative
- SQLite limits horizontal scaling
- go-zero learning curve
- goqite is less battle-tested than Redis

### Mitigations
- SQLite handles ~100k emails/day easily
- go-zero has good documentation
- goqite is simple enough to debug/replace

## References
- [go-zero MCP](https://github.com/zeromicro/go-zero)
- [goqite](https://github.com/maragudk/goqite)
- [Datastar](https://data-star.dev/)
- [Templ](https://templ.guide/)
