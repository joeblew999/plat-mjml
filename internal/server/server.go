package server

import (
	"fmt"
	"time"

	"github.com/joeblew999/plat-mjml/internal/ui"
	"github.com/joeblew999/plat-mjml/pkg/db"
	"github.com/joeblew999/plat-mjml/pkg/delivery"
	"github.com/joeblew999/plat-mjml/pkg/mail"
	"github.com/joeblew999/plat-mjml/pkg/mjml"
	"github.com/joeblew999/plat-mjml/pkg/queue"
	gomjml "github.com/preslavrachev/gomjml/mjml"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/mcp"
	"github.com/zeromicro/go-zero/rest"
)

// Server wraps the MCP server and email platform services.
type Server struct {
	config Config
	group  *service.ServiceGroup
}

// New creates a new server instance.
func New(c Config) (*Server, error) {
	// Create MCP server
	mcpServer := mcp.NewMcpServer(c.McpConf)

	// Create MJML renderer
	renderer := mjml.NewRenderer(
		mjml.WithTemplateDir(c.Templates.Dir),
		mjml.WithCache(true),
	)

	// Load templates
	if err := renderer.LoadTemplatesFromDir(c.Templates.Dir); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	// Open database
	database, err := db.Open(c.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create queue
	emailQueue, err := queue.NewQueue(database.DB, "emails", 2)
	if err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create queue: %w", err)
	}

	// Parse delivery config
	retryBackoff, _ := time.ParseDuration(c.Delivery.RetryBackoff)
	if retryBackoff == 0 {
		retryBackoff = 5 * time.Minute
	}
	maxBackoff, _ := time.ParseDuration(c.Delivery.MaxBackoff)
	if maxBackoff == 0 {
		maxBackoff = 4 * time.Hour
	}

	// Create delivery engine
	deliveryConfig := delivery.Config{
		MaxRetries:   c.Delivery.MaxRetries,
		RetryBackoff: retryBackoff,
		MaxBackoff:   maxBackoff,
		RateLimit:    c.Delivery.RateLimit,
	}

	smtpConfig := mail.GmailConfig()
	deliveryEngine := delivery.NewEngine(emailQueue, renderer, smtpConfig, deliveryConfig)

	// Register MCP tools
	RegisterMCPTools(mcpServer, renderer, emailQueue)

	// Create UI rest server
	uiServer, err := rest.NewServer(c.UI.RestConf)
	if err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create UI server: %w", err)
	}

	uiHandlers := ui.NewHandlers(renderer, emailQueue)
	uiServer.AddRoutes(uiHandlers.Routes())
	uiServer.AddRoutes(uiHandlers.SSERoutes(), rest.WithSSE())

	// Register cleanup via proc shutdown listeners
	proc.AddShutdownListener(func() {
		logx.Info("Closing database")
		database.Close()
	})
	proc.AddShutdownListener(func() {
		gomjml.StopASTCacheCleanup()
	})

	// Build service group: delivery + UI + MCP (stopped in reverse order)
	group := service.NewServiceGroup()
	group.Add(newDeliveryService(deliveryEngine, 2))
	group.Add(uiServer)
	group.Add(mcpServer)

	logx.Infow("plat-mjml server configured",
		logx.Field("mcp", fmt.Sprintf("http://%s:%d/sse", c.Host, c.Port)),
		logx.Field("ui", fmt.Sprintf("http://%s:%d", c.UI.Host, c.UI.Port)),
		logx.Field("templates", c.Templates.Dir),
		logx.Field("database", c.Database.Path),
	)
	logx.Infof("To add to Claude: claude mcp add plat-mjml -- npx -y mcp-remote http://localhost:%d/sse", c.Port)

	return &Server{config: c, group: group}, nil
}

// Start starts all services. Blocks until shutdown signal.
func (s *Server) Start() {
	s.group.Start()
}

// Stop stops all services.
func (s *Server) Stop() {
	s.group.Stop()
}
