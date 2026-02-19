package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/joeblew999/plat-mjml/pkg/mjml"
	"github.com/joeblew999/plat-mjml/pkg/queue"
	"github.com/zeromicro/go-zero/mcp"
)

// Typed argument structs — the SDK auto-generates JSON schema from these.

type renderTemplateArgs struct {
	Template string         `json:"template" jsonschema:"template slug, e.g. simple, welcome, reset_password, notification"`
	Data     map[string]any `json:"data,omitempty" jsonschema:"template variables as key-value pairs"`
}

type listTemplatesArgs struct{}

type sendEmailArgs struct {
	Template string         `json:"template" jsonschema:"template slug, e.g. welcome, reset_password"`
	To       []string       `json:"to" jsonschema:"list of recipient email addresses"`
	Subject  string         `json:"subject" jsonschema:"email subject line"`
	Data     map[string]any `json:"data,omitempty" jsonschema:"template variables as key-value pairs"`
}

type getEmailStatusArgs struct {
	ID string `json:"id" jsonschema:"email job ID returned from send_email"`
}

// RegisterMCPTools registers all MCP tools for the email platform.
func RegisterMCPTools(s mcp.McpServer, renderer *mjml.Renderer, q *queue.Queue) {
	registerRenderTool(s, renderer)
	registerListTemplatesTool(s, renderer)
	registerSendEmailTool(s, q)
	registerGetEmailStatusTool(s, q)
}

func registerRenderTool(s mcp.McpServer, renderer *mjml.Renderer) {
	tool := &mcp.Tool{
		Name:        "render_template",
		Description: "Render an MJML email template to HTML. Returns the rendered HTML that can be used for email sending.",
	}

	mcp.AddTool(s, tool, func(ctx context.Context, req *mcp.CallToolRequest, args renderTemplateArgs) (*mcp.CallToolResult, any, error) {
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
			return nil, nil, fmt.Errorf("render failed: %w", err)
		}

		result := map[string]any{
			"html":     html,
			"template": args.Template,
			"size":     len(html),
		}
		resultJSON, err := json.Marshal(result)
		if err != nil {
			return nil, nil, fmt.Errorf("marshal result: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(resultJSON)},
			},
		}, nil, nil
	})
}

func registerListTemplatesTool(s mcp.McpServer, renderer *mjml.Renderer) {
	tool := &mcp.Tool{
		Name:        "list_templates",
		Description: "List all available MJML email templates with their names and descriptions.",
	}

	mcp.AddTool(s, tool, func(ctx context.Context, req *mcp.CallToolRequest, args listTemplatesArgs) (*mcp.CallToolResult, any, error) {
		templates := renderer.ListTemplates()

		templateList := make([]map[string]any, 0, len(templates))
		for _, t := range templates {
			templateList = append(templateList, map[string]any{
				"slug":        t,
				"description": mjml.TemplateDescription(t),
			})
		}

		result := map[string]any{
			"templates": templateList,
			"count":     len(templateList),
		}
		resultJSON, err := json.Marshal(result)
		if err != nil {
			return nil, nil, fmt.Errorf("marshal result: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(resultJSON)},
			},
		}, nil, nil
	})
}

func registerSendEmailTool(s mcp.McpServer, q *queue.Queue) {
	tool := &mcp.Tool{
		Name:        "send_email",
		Description: "Queue an email for delivery. The email will be rendered using the specified template and sent to the recipients.",
	}

	mcp.AddTool(s, tool, func(ctx context.Context, req *mcp.CallToolRequest, args sendEmailArgs) (*mcp.CallToolResult, any, error) {
		// Use test data if none provided
		data := args.Data
		if data == nil {
			testData := mjml.TestData()
			td := testData[args.Template]
			if td == nil {
				td = testData["simple"]
			}
			// Convert struct test data to map[string]any via JSON round-trip
			b, err := json.Marshal(td)
			if err == nil {
				var m map[string]any
				if err := json.Unmarshal(b, &m); err == nil {
					data = m
				}
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
			return nil, nil, fmt.Errorf("failed to queue email: %w", err)
		}

		result := map[string]any{
			"id":         id,
			"status":     "queued",
			"recipients": len(args.To),
			"template":   args.Template,
		}
		resultJSON, err := json.Marshal(result)
		if err != nil {
			return nil, nil, fmt.Errorf("marshal result: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(resultJSON)},
			},
		}, nil, nil
	})
}

func registerGetEmailStatusTool(s mcp.McpServer, q *queue.Queue) {
	tool := &mcp.Tool{
		Name:        "get_email_status",
		Description: "Get the delivery status of a queued email by its ID.",
	}

	mcp.AddTool(s, tool, func(ctx context.Context, req *mcp.CallToolRequest, args getEmailStatusArgs) (*mcp.CallToolResult, any, error) {
		job, err := q.GetStatus(ctx, args.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get status: %w", err)
		}
		if job == nil {
			return nil, nil, fmt.Errorf("email not found: %s", args.ID)
		}

		result := map[string]any{
			"id":         job.ID,
			"template":   job.TemplateSlug,
			"recipients": job.Recipients,
			"subject":    job.Subject,
			"status":     job.Status,
			"attempts":   job.Attempts,
			"error":      job.Error,
			"created_at": job.CreatedAt,
		}
		resultJSON, err := json.Marshal(result)
		if err != nil {
			return nil, nil, fmt.Errorf("marshal result: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(resultJSON)},
			},
		}, nil, nil
	})
}

