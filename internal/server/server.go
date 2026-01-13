package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/joeblew999/plat-mjml/internal/ui"
	"github.com/joeblew999/plat-mjml/pkg/db"
	"github.com/joeblew999/plat-mjml/pkg/delivery"
	"github.com/joeblew999/plat-mjml/pkg/mail"
	"github.com/joeblew999/plat-mjml/pkg/mjml"
	"github.com/joeblew999/plat-mjml/pkg/queue"
	"github.com/zeromicro/go-zero/mcp"
)

// Server wraps the MCP server and email platform services.
type Server struct {
	config   Config
	mcp      mcp.McpServer
	renderer *mjml.Renderer
	db       *db.DB
	queue    *queue.Queue
	delivery *delivery.Engine
	ui       *ui.Handlers
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

	// Create UI handlers
	uiHandlers := ui.NewHandlers(renderer, emailQueue)

	s := &Server{
		config:   c,
		mcp:      mcpServer,
		renderer: renderer,
		db:       database,
		queue:    emailQueue,
		delivery: deliveryEngine,
		ui:       uiHandlers,
	}

	// Register MCP tools
	RegisterMCPTools(mcpServer, renderer, emailQueue)

	return s, nil
}

// Start starts the server.
func (s *Server) Start() {
	fmt.Printf("Starting plat-mjml server on %s:%d\n", s.config.Host, s.config.Port)
	fmt.Printf("MCP endpoint: http://%s:%d/sse\n", s.config.Host, s.config.Port)
	fmt.Printf("Web UI: http://%s:%d/\n", s.config.Host, s.config.Port)
	fmt.Printf("Templates loaded from: %s\n", s.config.Templates.Dir)
	fmt.Printf("Database: %s\n", s.config.Database.Path)
	fmt.Println()
	fmt.Println("To add to Claude:")
	fmt.Printf("  claude mcp add plat-mjml -- npx -y mcp-remote http://localhost:%d/sse\n", s.config.Port)

	// Start delivery workers
	s.delivery.Start(2)

	// Start UI server on a separate port
	uiPort := s.config.Port + 1
	mux := http.NewServeMux()
	s.ui.RegisterRoutes(mux)

	go func() {
		addr := fmt.Sprintf("%s:%d", s.config.Host, uiPort)
		fmt.Printf("\nWeb UI running at http://%s\n", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			fmt.Printf("UI server error: %v\n", err)
		}
	}()

	s.mcp.Start()
}

// Stop stops the server.
func (s *Server) Stop() {
	s.delivery.Stop()
	s.mcp.Stop()
	s.db.Close()
}
