package server

import (
	"context"
	"fmt"

	"github.com/joeblew999/plat-mjml/pkg/mjml"
	"github.com/joeblew999/plat-mjml/pkg/queue"
	"github.com/zeromicro/go-zero/mcp"
)

// RegisterMCPTools registers all MCP tools for the email platform.
func RegisterMCPTools(s mcp.McpServer, renderer *mjml.Renderer, q *queue.Queue) {
	registerRenderTool(s, renderer)
	registerListTemplatesTool(s, renderer)
	registerSendEmailTool(s, q)
	registerGetEmailStatusTool(s, q)
	registerTemplatesResource(s, renderer)
}

func registerRenderTool(s mcp.McpServer, renderer *mjml.Renderer) {
	s.RegisterTool(mcp.Tool{
		Name:        "render_template",
		Description: "Render an MJML email template to HTML. Returns the rendered HTML that can be used for email sending.",
		InputSchema: mcp.InputSchema{
			Properties: map[string]any{
				"template": map[string]any{
					"type":        "string",
					"description": "Template slug (e.g., simple, welcome, reset_password, notification)",
				},
				"data": map[string]any{
					"type":        "object",
					"description": "Template variables as key-value pairs (e.g., {\"name\": \"John\", \"email\": \"john@example.com\"})",
				},
			},
			Required: []string{"template"},
		},
		Handler: func(ctx context.Context, p map[string]any) (any, error) {
			var args struct {
				Template string         `json:"template"`
				Data     map[string]any `json:"data"`
			}
			if err := mcp.ParseArguments(p, &args); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			// Use test data if none provided
			var data any = args.Data
			if data == nil {
				testData := mjml.TestData()
				if d, ok := testData[args.Template]; ok {
					data = d
				} else {
					data = testData["simple"]
				}
			}

			html, err := renderer.RenderTemplate(args.Template, data)
			if err != nil {
				return nil, fmt.Errorf("render failed: %w", err)
			}

			return map[string]any{
				"html":     html,
				"template": args.Template,
				"size":     len(html),
			}, nil
		},
	})
}

func registerListTemplatesTool(s mcp.McpServer, renderer *mjml.Renderer) {
	s.RegisterTool(mcp.Tool{
		Name:        "list_templates",
		Description: "List all available MJML email templates with their names and descriptions.",
		InputSchema: mcp.InputSchema{
			Properties: map[string]any{},
		},
		Handler: func(ctx context.Context, p map[string]any) (any, error) {
			templates := renderer.ListTemplates()

			result := make([]map[string]any, 0, len(templates))
			for _, t := range templates {
				result = append(result, map[string]any{
					"slug":        t,
					"description": getTemplateDescription(t),
				})
			}

			return map[string]any{
				"templates": result,
				"count":     len(result),
			}, nil
		},
	})
}

func registerSendEmailTool(s mcp.McpServer, q *queue.Queue) {
	s.RegisterTool(mcp.Tool{
		Name:        "send_email",
		Description: "Queue an email for delivery. The email will be rendered using the specified template and sent to the recipients.",
		InputSchema: mcp.InputSchema{
			Properties: map[string]any{
				"template": map[string]any{
					"type":        "string",
					"description": "Template slug (e.g., welcome, reset_password)",
				},
				"to": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "string"},
					"description": "List of recipient email addresses",
				},
				"subject": map[string]any{
					"type":        "string",
					"description": "Email subject line",
				},
				"data": map[string]any{
					"type":        "object",
					"description": "Template variables as key-value pairs",
				},
			},
			Required: []string{"template", "to", "subject"},
		},
		Handler: func(ctx context.Context, p map[string]any) (any, error) {
			var args struct {
				Template string         `json:"template"`
				To       []string       `json:"to"`
				Subject  string         `json:"subject"`
				Data     map[string]any `json:"data"`
			}
			if err := mcp.ParseArguments(p, &args); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			// Use test data if none provided
			data := args.Data
			if data == nil {
				testData := mjml.TestData()
				if d, ok := testData[args.Template].(map[string]any); ok {
					data = d
				}
			}

			job := queue.EmailJob{
				TemplateSlug: args.Template,
				Recipients:   args.To,
				Subject:      args.Subject,
				Data:         data,
				Priority:     queue.PriorityNormal,
			}

			id, err := q.Enqueue(ctx, job)
			if err != nil {
				return nil, fmt.Errorf("failed to queue email: %w", err)
			}

			return map[string]any{
				"id":         id,
				"status":     "queued",
				"recipients": len(args.To),
				"template":   args.Template,
			}, nil
		},
	})
}

func registerGetEmailStatusTool(s mcp.McpServer, q *queue.Queue) {
	s.RegisterTool(mcp.Tool{
		Name:        "get_email_status",
		Description: "Get the delivery status of a queued email by its ID.",
		InputSchema: mcp.InputSchema{
			Properties: map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "Email job ID returned from send_email",
				},
			},
			Required: []string{"id"},
		},
		Handler: func(ctx context.Context, p map[string]any) (any, error) {
			var args struct {
				ID string `json:"id"`
			}
			if err := mcp.ParseArguments(p, &args); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			job, err := q.GetStatus(ctx, args.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to get status: %w", err)
			}
			if job == nil {
				return nil, fmt.Errorf("email not found: %s", args.ID)
			}

			return map[string]any{
				"id":         job.ID,
				"template":   job.TemplateSlug,
				"recipients": job.Recipients,
				"subject":    job.Subject,
				"attempts":   job.Attempts,
				"error":      job.Error,
				"created_at": job.CreatedAt,
			}, nil
		},
	})
}

func registerTemplatesResource(s mcp.McpServer, renderer *mjml.Renderer) {
	s.RegisterResource(mcp.Resource{
		Name:        "templates",
		URI:         "mjml://templates",
		Description: "Available MJML email templates",
		MimeType:    "application/json",
		Handler: func(ctx context.Context) (mcp.ResourceContent, error) {
			templates := renderer.ListTemplates()

			content := "Available templates:\n"
			for _, t := range templates {
				content += fmt.Sprintf("- %s: %s\n", t, getTemplateDescription(t))
			}

			return mcp.ResourceContent{
				URI:      "mjml://templates",
				MimeType: "text/plain",
				Text:     content,
			}, nil
		},
	})
}

func getTemplateDescription(slug string) string {
	descriptions := map[string]string{
		"simple":                "Basic email template",
		"welcome":               "Welcome/activation email for new users",
		"reset_password":        "Password reset email with security info",
		"notification":          "System notification email",
		"premium_newsletter":    "Newsletter with premium fonts",
		"business_announcement": "Business announcement email",
	}
	if desc, ok := descriptions[slug]; ok {
		return desc
	}
	return "Email template"
}
