package server

import (
	"fmt"
	"time"

	"net/http"

	"github.com/joeblew999/plat-mjml/internal/errorx"
	"github.com/joeblew999/plat-mjml/internal/handler"
	"github.com/joeblew999/plat-mjml/internal/svc"
	"github.com/joeblew999/plat-mjml/internal/ui"
	"github.com/joeblew999/plat-mjml/pkg/db"
	"github.com/joeblew999/plat-mjml/pkg/delivery"
	"github.com/joeblew999/plat-mjml/pkg/mail"
	"github.com/joeblew999/plat-mjml/pkg/mjml"
	"github.com/joeblew999/plat-mjml/pkg/queue"
	gomjml "github.com/preslavrachev/gomjml/mjml"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"
	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/core/prometheus"
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
	// Register global error handler for proper HTTP status codes
	errorx.RegisterErrorHandler()

	// Enable go-zero prometheus metrics (required for metric.CounterVec/HistogramVec/GaugeVec to record)
	prometheus.Enable()

	// Create MCP server
	mcpServer := mcp.NewMcpServer(c.McpConf)

	// Parallel initialization: template loading and database opening are independent
	var renderer *mjml.Renderer
	var database *db.DB

	err := mr.Finish(
		func() error {
			renderer = mjml.NewRenderer(
				mjml.WithTemplateDir(c.Templates.Dir),
				mjml.WithFontDir(c.Fonts.Dir),
				mjml.WithCache(true),
			)
			return renderer.LoadTemplatesFromDir(c.Templates.Dir)
		},
		func() error {
			var e error
			database, e = db.Open(c.Database.Path)
			return e
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize: %w", err)
	}

	// Create queue using go-zero sqlx.SqlConn for circuit breaking + tracing
	conn := database.SqlConn()
	emailQueue := queue.NewQueue(conn)

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

	smtpConfig := mail.Config{
		SMTPHost:  c.SMTP.Host,
		SMTPPort:  c.SMTP.Port,
		Username:  c.SMTP.Username,
		Password:  c.SMTP.Password,
		FromEmail: c.SMTP.FromEmail,
		FromName:  c.SMTP.FromName,
	}
	deliveryEngine := delivery.NewEngine(emailQueue, renderer, smtpConfig, deliveryConfig)

	// Register MCP tools
	RegisterMCPTools(mcpServer, renderer, emailQueue)

	// Create UI rest server (Datastar web UI) with CORS
	uiServer, err := rest.NewServer(c.UI.RestConf, rest.WithCors("*"))
	if err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create UI server: %w", err)
	}

	uiHandlers := ui.NewHandlers(renderer, emailQueue)
	uiServer.AddRoutes(uiHandlers.Routes())
	uiServer.AddRoutes(uiHandlers.SSERoutes(), rest.WithSSE())

	// Create API rest server (goctl-generated JSON REST API) with CORS
	apiServer, err := rest.NewServer(c.API.RestConf, rest.WithCors("*"))
	if err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to create API server: %w", err)
	}

	apiCtx := svc.NewServiceContext(renderer, emailQueue)
	handler.RegisterHandlers(apiServer, apiCtx)

	// Expose Prometheus metrics endpoint
	apiServer.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/metrics",
		Handler: promhttp.Handler().ServeHTTP,
	})

	// Register cleanup via proc shutdown listeners
	proc.AddShutdownListener(func() {
		logx.Info("Closing database")
		database.Close()
	})
	proc.AddShutdownListener(func() {
		gomjml.StopASTCacheCleanup()
	})
	if emailQueue.Events != nil {
		proc.AddShutdownListener(func() {
			logx.Info("Flushing email events")
			emailQueue.Events.Flush()
		})
	}

	// Build service group: delivery + UI + API + MCP (stopped in reverse order)
	group := service.NewServiceGroup()
	group.Add(newDeliveryService(deliveryEngine, 2))
	group.Add(uiServer)
	group.Add(apiServer)
	group.Add(mcpServer)

	logx.Infow("plat-mjml server configured",
		logx.Field("mcp", fmt.Sprintf("http://%s:%d/sse", c.Host, c.Port)),
		logx.Field("ui", fmt.Sprintf("http://%s:%d", c.UI.Host, c.UI.Port)),
		logx.Field("api", fmt.Sprintf("http://%s:%d/api/v1", c.API.Host, c.API.Port)),
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
