# plat-mjml

MJML email template platform with MCP integration for AI assistants.

## Features

- **MCP Server** - AI assistant integration via go-zero
- **Web UI** - Datastar-based dashboard for email management
- **Email Queue** - SQLite-backed queue with goqite
- **Delivery Engine** - Retry logic with exponential backoff
- **Google Fonts** - CDN-based font integration for email templates
- **CLI Tool** - Render, validate, and send emails
- **Go Library** - Programmatic template rendering
- **Template Caching** - Performance optimization

## Web UI

The platform ships with a Datastar-powered web dashboard for managing templates, monitoring the queue, and sending emails.

| Dashboard | Templates |
|:---------:|:---------:|
| ![Dashboard](docs/screenshots/dashboard.png) | ![Templates](docs/screenshots/templates.png) |

| Queue | Send |
|:-----:|:----:|
| ![Queue](docs/screenshots/queue.png) | ![Send](docs/screenshots/send.png) |

## Quick Start

```bash
# Install dependencies
task deps

# Start the server (MCP on :8080, Web UI on :8081)
task server

# Or directly
go run ./cmd/server
```

### Add to Claude

```bash
claude mcp add plat-mjml -- npx -y mcp-remote http://localhost:8080/sse
```

Then ask Claude:
- "List the email templates"
- "Render the welcome template with name John"
- "Send a test email to user@example.com"

## CLI Usage

```bash
# List available templates
task list

# Render a template
task render TEMPLATE=welcome

# Render to file
task render TEMPLATE=welcome OUT=.data/welcome.html

# Validate HTML
task validate FILE=.data/welcome.html

# Send test email (requires GMAIL env vars)
task send TO=test@example.com FILE=.data/welcome.html
```

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

## Templates

| Template | Description |
|----------|-------------|
| `simple` | Basic email |
| `welcome` | Welcome/activation email |
| `reset_password` | Password reset with security info |
| `notification` | System notifications |
| `premium_newsletter` | Newsletter with premium fonts |
| `business_announcement` | Business announcements |

All templates support Google Fonts via `FontCSS`/`FontStack` fields with email-safe fallbacks (Inter as primary, Arial/Helvetica as fallback).

## Project Structure

```
├── main.go              # CLI entry point
├── cmd/server/          # MCP server
├── internal/
│   ├── server/          # Server implementation
│   └── ui/              # Datastar web UI
├── pkg/
│   ├── mjml/            # Core MJML rendering
│   ├── font/            # Google Fonts (CDN URLs for email)
│   ├── mail/            # SMTP and validation
│   ├── db/              # SQLite database
│   ├── queue/           # Email queue (goqite)
│   ├── delivery/        # Delivery engine
│   ├── config/          # Configuration
│   └── log/             # Logging
├── templates/           # MJML email templates
├── docs/adr/            # Architecture decisions
└── config.yaml          # Server configuration
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
| `GMAIL_USERNAME` | Gmail address | - |
| `GMAIL_APP_PASSWORD` | Gmail app password | - |

Copy `.env.example` to `.env` and fill in your credentials.

## Task Commands

```bash
task server       # Run MCP server
task list         # List templates
task render       # Render template
task validate     # Validate HTML
task send         # Send email
task test         # Run all tests
task build        # Build server binary
task build:cli    # Build CLI binary
task clean:data   # Clean cache
task kill:ports   # Kill server ports
```

## Architecture

See [ADR-001](docs/adr/001-email-platform-architecture.md) for detailed architecture decisions.

## Acknowledgements

- [gomjml](https://github.com/preslavrachev/gomjml) by [Preslav Rachev](https://github.com/preslavrachev) — Pure Go MJML renderer. No Node.js required.
- [go-zero](https://github.com/zeromicro/go-zero) by [Kevin Wan](https://github.com/kevwan) — Cloud-native Go microservices framework for MCP server.
- [goqite](https://maragu.dev/goqite) by [Markus Wüstenberg](https://github.com/maragudk) — SQLite-backed persistent message queue.
- [gomponents](https://maragu.dev/gomponents) + [gomponents-datastar](https://maragu.dev/gomponents-datastar) — Go HTML components with Datastar integration.
