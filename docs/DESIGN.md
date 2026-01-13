# MJML Package Design

## Overview

The MJML package provides email template rendering using MJML (Mailjet Markup Language) for generating responsive HTML emails at runtime. This enables AI agents and MCP systems to create professional emails dynamically.

## Architecture

```
pkg/mjml/
├── renderer.go           # Core MJML rendering engine
├── service.go           # Infrastructure service wrapper
├── templates.go         # Default templates and data structures
├── templates/           # MJML template files
│   ├── simple.mjml
│   ├── welcome.mjml
│   ├── reset_password.mjml
│   ├── notification.mjml
│   └── business_announcement.mjml
├── example/             # Example application
│   ├── go.mod
│   └── main.go
├── renderer_test.go     # Unit tests
├── integration_test.go  # Template file tests
└── README.md
```

## Components

### Renderer
- **Purpose**: Core MJML template processing
- **Features**: Template loading, caching, variable substitution
- **Dependencies**: gomjml library for MJML→HTML conversion

### Service
- **Purpose**: Infrastructure integration layer
- **Features**: Lifecycle management, logging, health checks
- **Integration**: Works with pkg/config for paths

### Templates
- **Purpose**: Predefined email templates for common use cases
- **Types**: Welcome, password reset, notifications, business announcements
- **Format**: MJML XML with Go template variables

## Data Flow

```
Template Data → Go Template → MJML → gomjml → Responsive HTML
```

1. **Input**: Structured data (EmailData, WelcomeEmailData, etc.)
2. **Template Processing**: Go template engine substitutes variables
3. **MJML Compilation**: gomjml converts MJML to responsive HTML
4. **Output**: Email-ready HTML with responsive design

## AI/MCP Integration

### Use Cases
- **AI Email Generation**: AI agents create emails using structured prompts
- **MCP Services**: External tools generate emails via MCP protocol
- **Dynamic Content**: Runtime email generation based on events
- **Template Management**: Web GUI for editing templates via DataStar

### API Patterns
```go
// Service-level API
service := mjml.NewService()
html, err := service.RenderEmail("welcome", welcomeData)

// Direct rendering for AI agents
renderer := mjml.NewRenderer(mjml.WithCache(true))
html, err := renderer.RenderTemplate("notification", alertData)

// Dynamic MJML for MCP
html, err := service.RenderEmailString(aiGeneratedMJML)
```

## Template Design

### Variable Naming
- **Go Structs**: PascalCase field names (e.g., `CompanyName`)
- **Templates**: Match struct fields exactly (e.g., `{{.CompanyName}}`)
- **JSON Input**: Can use either convention, mapped automatically

### Template Structure
```xml
<mjml>
  <mj-head>
    <mj-title>{{.Subject}}</mj-title>
    <mj-preview>{{.Preview}}</mj-preview>
  </mj-head>
  <mj-body>
    <!-- Responsive email content -->
  </mj-body>
</mjml>
```

### Data Types
- **EmailData**: Base email structure
- **WelcomeEmailData**: User onboarding emails
- **ResetPasswordData**: Security-related emails
- **NotificationData**: Alert and system emails
- **Custom Data**: map[string]interface{} for flexible use

## Performance

### Caching Strategy
- **Template Cache**: Compiled Go templates cached in memory
- **Render Cache**: Generated HTML cached by data hash
- **Invalidation**: Manual cache clearing on template updates

### Optimization
- **Parallel Loading**: Templates loaded concurrently
- **Memory Efficient**: Templates reused across renders
- **Fast Rendering**: gomjml optimized for performance

## Security

### Input Validation
- **Template Safety**: Go template engine prevents code injection
- **MJML Validation**: Optional MJML syntax validation
- **Data Sanitization**: User input should be sanitized before rendering

### Best Practices
- **Template Isolation**: Templates cannot access file system
- **Content Security**: No arbitrary code execution in templates
- **Error Handling**: Detailed errors without information disclosure

## Configuration

### Service Options
```go
service := mjml.NewServiceWithOptions(
    "/custom/templates",
    mjml.WithCache(true),        // Enable HTML caching
    mjml.WithDebug(false),       // Production mode
    mjml.WithValidation(true),   // Validate MJML syntax
)
```

### Directory Structure
```
.data/email-templates/          # Default template directory
├── welcome.mjml
├── reset_password.mjml
├── notification.mjml
└── custom/                     # Custom templates
    └── branded_welcome.mjml
```

## Future Enhancements

### Planned Features
- **Template Editor**: Web-based MJML editor with live preview
- **Version Control**: Template versioning and rollback
- **A/B Testing**: Multiple template variants
- **Analytics**: Email rendering metrics and performance tracking
- **Internationalization**: Multi-language template support

### AI/MCP Enhancements
- **Template Generation**: AI creates new templates from descriptions
- **Content Optimization**: AI optimizes email content for engagement
- **Dynamic Layouts**: Runtime layout generation based on content
- **Smart Defaults**: AI suggests appropriate templates for content type

## Dependencies

### Core Dependencies
- `github.com/preslavrachev/gomjml/mjml` - MJML to HTML conversion
- `html/template` - Go template processing
- `sync` - Thread-safe operations

### Infrastructure Dependencies
- `pkg/config` - Configuration management
- `pkg/log` - Structured logging
- File system access for template loading

## Testing

### Test Categories
- **Unit Tests**: Core functionality testing
- **Integration Tests**: Template file processing
- **Performance Tests**: Cache and rendering performance
- **Example Tests**: End-to-end usage validation

### Test Data
- Generated HTML saved for manual inspection
- Template validation for MJML compliance
- Cross-platform rendering verification