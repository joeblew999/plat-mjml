# plat-mjml

MJML (Mailjet Markup Language).

MJML email template platform with MCP integration for AI assistants.

## Features

- **MCP Server** — 4 tools for Claude to render templates, send emails, and check delivery status
- **REST API** — goctl-generated JSON API with Swagger docs (`/api/v1/*`)
- **Web UI** — Datastar-based dashboard for email management
- **Email Queue** — SQLite-backed queue with retry and exponential backoff
- **Google Fonts** — CDN-based font integration for email templates
- **CLI Tool** — Render, validate, and send emails from the terminal
- **Go Library** — Embed template rendering in your own Go services
- **Docker** — goctl-generated Dockerfile for containerized deployment

## Quick Start

### Prerequisites

- Go 1.25+
- [Task](https://taskfile.dev) (optional, for convenience commands)

### 1. Start the Server

```bash
# Clone and run
git clone https://github.com/joeblew999/plat-mjml.git
cd plat-mjml
go run ./cmd/server
```

This starts four services in a single process via go-zero ServiceGroup:
- **MCP server** on `http://localhost:8080/sse`
- **Web UI** on `http://localhost:8081`
- **REST API** on `http://localhost:8082/api/v1`
- **Delivery engine** — background email processing with retry/backoff

### 2. Connect to Claude

#### Claude Code (CLI)

```bash
claude mcp add plat-mjml -- npx -y mcp-remote http://localhost:8080/sse
```

Then in any Claude Code session:

```
> List the available email templates
> Render the welcome template for a user named Alice
> Send a welcome email to alice@example.com with subject "Welcome aboard"
> Check the status of that email
```

#### Claude Desktop

Add to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "plat-mjml": {
      "command": "npx",
      "args": ["-y", "mcp-remote", "http://localhost:8080/sse"]
    }
  }
}
```

Restart Claude Desktop — the email tools will appear in the tool list.

### 3. Open the Web UI

Visit [http://localhost:8081](http://localhost:8081) to manage templates, monitor the queue, and send test emails.

| Dashboard | Templates |
|:---------:|:---------:|
| ![Dashboard](docs/screenshots/dashboard.png) | ![Templates](docs/screenshots/templates.png) |

| Queue | Send |
|:-----:|:----:|
| ![Queue](docs/screenshots/queue.png) | ![Send](docs/screenshots/send.png) |

## MCP Tools

The server exposes 4 tools via the [Model Context Protocol](https://modelcontextprotocol.io):

| Tool | Description |
|------|-------------|
| `list_templates` | List all available email templates with descriptions |
| `render_template` | Render an MJML template to HTML with provided data |
| `send_email` | Queue an email for delivery (template + recipients + subject) |
| `get_email_status` | Check delivery status of a queued email by ID |

### Example Conversation with Claude

```
You: Send a password reset email to user@example.com

Claude: I'll send that for you.
[Calls send_email with template="reset_password", to=["user@example.com"],
 subject="Reset Your Password"]

The email has been queued with ID abc123. It will be delivered shortly.

You: What's the status?

Claude: [Calls get_email_status with id="abc123"]

The email is currently in "processing" status. It's been attempted once
and is being delivered now.
```

## REST API

The REST API runs on port 8082 as a goctl-generated service. The API contract is defined in [`api/plat-mjml.api`](api/plat-mjml.api) — edit this file, run `task generate`, and goctl regenerates handlers, types, routes, and Swagger docs.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/templates` | List all templates |
| `GET` | `/api/v1/templates/:slug` | Get template info |
| `GET` | `/api/v1/templates/:slug/render` | Render template to HTML |
| `POST` | `/api/v1/emails` | Queue an email for delivery |
| `GET` | `/api/v1/emails/:id` | Get email delivery status |
| `GET` | `/api/v1/emails?status=pending&limit=50` | List queued emails |
| `GET` | `/api/v1/stats` | Get queue statistics |

### Examples

```bash
# List templates
curl http://localhost:8082/api/v1/templates

# Render a template
curl http://localhost:8082/api/v1/templates/welcome/render

# Send an email
curl -X POST http://localhost:8082/api/v1/emails \
  -H 'Content-Type: application/json' \
  -d '{"template":"welcome","to":["user@example.com"],"subject":"Hello"}'

# Check status
curl http://localhost:8082/api/v1/emails/<id>

# Queue stats
curl http://localhost:8082/api/v1/stats
```

Swagger documentation is available at [docs/swagger.json](docs/swagger.json).

### goctl Code Generation Workflow

https://go-zero.dev/en/docs/tutorials/cli/overview

```bash
# 1. Edit the API contract
vim api/plat-mjml.api

# 2. Regenerate (validates, generates code + Swagger)
task generate

# goctl regenerates:
#   internal/types/types.go    — request/response structs (DO NOT EDIT)
#   internal/handler/routes.go — route registration (DO NOT EDIT)
#   docs/swagger.json          — OpenAPI 2.0 spec
#
# goctl preserves (Safe to edit):
#   internal/handler/*/        — handler stubs (scaffolded once)
#   internal/logic/*/          — business logic (your code lives here)
#   internal/svc/              — service context (dependency injection)
```

## Email Delivery Setup

To actually send emails, configure Gmail SMTP credentials:

```bash
cp .env.example .env
```

Edit `.env`:
```
GMAIL_USERNAME=your-email@gmail.com
GMAIL_APP_PASSWORD=your-app-password
```

> **Note:** Generate an [App Password](https://myaccount.google.com/apppasswords) in your Google account settings. Regular passwords won't work with 2FA enabled.

Without credentials, emails are queued but delivery will fail (useful for testing the queue UI).

## CLI Usage

```bash
# List available templates
go run . list

# Render a template to stdout
go run . render -template=welcome

# Render to file
go run . render -template=welcome -out=welcome.html

# Validate rendered HTML for email client compatibility
go run . validate -file=welcome.html

# Send a rendered HTML file
go run . send -to=test@example.com -file=welcome.html
```

Or with Task:

```bash
task list
task render TEMPLATE=welcome OUT=./.data/welcome.html
task validate FILE=./.data/welcome.html
task send TO=test@example.com FILE=.data/welcome.html
```

## Templates

| Template | Description |
|----------|-------------|
| `simple` | Basic email |
| `welcome` | Welcome/activation email |
| `reset_password` | Password reset with security info |
| `notification` | System notifications |
| `premium_newsletter` | Newsletter with premium fonts |
| `business_announcement` | Business announcements |

All templates use Google Fonts (Inter) with email-safe fallbacks (Arial, Helvetica, sans-serif). Font CSS uses CDN URLs so it works in email clients that support `@font-face` (Apple Mail, iOS Mail, Thunderbird).

## Library Usage

```go
import "github.com/joeblew999/plat-mjml/pkg/mjml"

renderer := mjml.NewRenderer(
    mjml.WithCache(true),
    mjml.WithTemplateDir("./templates"),
)

renderer.LoadTemplatesFromDir("./templates")

html, err := renderer.RenderTemplate("welcome", map[string]any{
    "name":  "John Doe",
    "email": "john@example.com",
})
```

## Project Structure

```
├── main.go              # CLI entry point
├── cmd/server/          # Server startup (MCP + UI + API + delivery)
├── api/
│   └── plat-mjml.api    # goctl API definition (source of truth)
├── internal/
│   ├── config/          # goctl-generated API config
│   ├── handler/         # goctl-generated HTTP handlers + routes
│   ├── logic/           # Business logic (safe to edit)
│   ├── svc/             # Service context — dependency injection
│   ├── types/           # goctl-generated request/response types
│   ├── server/          # MCP tools, config, startup, ServiceGroup
│   └── ui/              # Datastar web UI (gomponents + SSE)
├── pkg/
│   ├── mjml/            # MJML rendering, templates, font integration
│   ├── font/            # Google Fonts download + CDN URL capture
│   ├── mail/            # SMTP sending + HTML validation
│   ├── db/              # SQLite (auto-migrating)
│   ├── queue/           # Email queue (goqite)
│   ├── delivery/        # Delivery engine with retry/backoff
│   └── config/          # Path configuration
├── templates/           # MJML email templates
├── config.yaml          # Server configuration
├── Dockerfile           # goctl-generated Docker build
└── docs/                # ADRs, Swagger, screenshots
```

## Configuration

```yaml
# config.yaml
name: plat-mjml
host: 0.0.0.0
port: 8080

ui:
  name: plat-mjml-ui
  host: 0.0.0.0
  port: 8081

api:
  name: plat-mjml-api
  host: 0.0.0.0
  port: 8082

mcp:
  name: mjml-server
  version: "1.0.0"
  messageTimeout: 30s

templates:
  dir: ./templates

database:
  path: ./.data/plat-mjml.db

delivery:
  maxRetries: 3
  retryBackoff: 5m
  maxBackoff: 4h
  rateLimit: 60
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATA_PATH` | Base data directory | `./.data` |
| `MJML_TEMPLATE_PATH` | Template directory | `./templates` |
| `FONT_PATH` | Font cache directory | `./.data/fonts` |
| `GMAIL_USERNAME` | Gmail address for sending | — |
| `GMAIL_APP_PASSWORD` | Gmail app password | — |

## Task Commands

```bash
task server     # Start server (kills stale ports first)
task build      # Build all binaries (skips if up-to-date)
task test       # Run all tests
task generate   # Regenerate API code + Swagger from .api file
task deps       # Install tools (skips if already installed)
task list       # List templates
task render     # Render template
task validate   # Validate HTML
task send       # Send email
task clean      # Remove build artifacts + data cache
task kill-ports # Kill processes on 8080/8081/8082
```

## Architecture

See [ADR-001](docs/adr/001-email-platform-architecture.md) for detailed architecture decisions.

## Acknowledgements

- [gomjml](https://github.com/preslavrachev/gomjml) by [Preslav Rachev](https://github.com/preslavrachev) — Pure Go MJML renderer. No Node.js required.
- [go-zero](https://github.com/zeromicro/go-zero) by [Kevin Wan](https://github.com/kevwan) — Cloud-native Go microservices framework. Powers MCP server, REST API (goctl-generated handler/logic pattern + Swagger), Web UI (rest.Server), service lifecycle (ServiceGroup), graceful shutdown, structured logging (logx), and Docker builds.
- [goctl](https://github.com/zeromicro/go-zero/tree/master/tools/goctl) — go-zero code generator. Generates REST API handlers, types, routes, and Swagger docs from `.api` file definition.
- [goqite](https://maragu.dev/goqite) by [Markus Wüstenberg](https://github.com/maragudk) — SQLite-backed persistent message queue.
- [gomponents](https://maragu.dev/gomponents) + [gomponents-datastar](https://maragu.dev/gomponents-datastar) — Go HTML components with Datastar integration.
