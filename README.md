# plat-mjml

MJML email template platform with MCP integration for AI assistants.

## Features

- **MCP Server** - AI assistant integration via go-zero
- **Web UI** - Datastar-based dashboard for email management
- **Email Queue** - SQLite-backed queue with goqite
- **Delivery Engine** - Retry logic with exponential backoff
- **CLI Tool** - Render, validate, and send emails
- **Go Library** - Programmatic template rendering
- **Email Validation** - Client compatibility checking
- **SMTP Sending** - Direct email delivery
- **Google Fonts** - Font integration for emails
- **Template Caching** - Performance optimization

## Quick Start

### Run MCP Server

```bash
# Start the server
task server

# Or directly
go run ./cmd/server
```

The server provides:
- MCP endpoint at `http://localhost:8080/sse`
- Web UI at `http://localhost:8081/`
- Tools: `render_template`, `list_templates`, `send_email`, `get_email_status`

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

# Send test email
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

## Project Structure

```
├── main.go              # CLI entry point
├── cmd/server/          # MCP server
├── internal/
│   ├── server/          # Server implementation
│   └── ui/              # Datastar web UI
├── pkg/
│   ├── mjml/            # Core MJML rendering
│   ├── mail/            # SMTP and validation
│   ├── db/              # SQLite database
│   ├── queue/           # Email queue (goqite)
│   ├── delivery/        # Delivery engine
│   ├── font/            # Google Fonts
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
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATA_PATH` | Base data directory | `./.data` |
| `MJML_TEMPLATE_PATH` | Template directory | `./templates` |
| `FONT_PATH` | Font cache directory | `./.data/fonts` |
| `GMAIL_USERNAME` | Gmail address | - |
| `GMAIL_APP_PASSWORD` | Gmail app password | - |

## Task Commands

```bash
task server       # Run MCP server
task list         # List templates
task render       # Render template
task validate     # Validate HTML
task send         # Send email
task clean:data   # Clean cache
```

## Templates

- `simple` - Basic email
- `welcome` - Welcome/activation
- `reset_password` - Password reset
- `notification` - System notifications
- `premium_newsletter` - Newsletter with fonts
- `business_announcement` - Business announcements

## Dependencies

- [go-zero](https://github.com/zeromicro/go-zero) - MCP server framework
- [gomjml](https://github.com/preslavrachev/gomjml) - Pure Go MJML renderer
- [goqite](https://maragu.dev/goqite) - SQLite-backed message queue
- [gomponents](https://maragu.dev/gomponents) - Go HTML components
- [gomponents-datastar](https://maragu.dev/gomponents-datastar) - Datastar integration

## Architecture

See [ADR-001](docs/adr/001-email-platform-architecture.md) for detailed architecture decisions.
