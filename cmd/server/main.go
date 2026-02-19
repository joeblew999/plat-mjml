package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/joeblew999/plat-mjml/internal/config"
	"github.com/joeblew999/plat-mjml/internal/errorx"
	"github.com/joeblew999/plat-mjml/internal/handler"
	"github.com/joeblew999/plat-mjml/internal/server"
	"github.com/joeblew999/plat-mjml/internal/svc"
	"github.com/joeblew999/plat-mjml/internal/ui"
	"github.com/joeblew999/plat-mjml/pkg/delivery"
	gomjml "github.com/preslavrachev/gomjml/mjml"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/core/prometheus"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/mcp"
	"github.com/zeromicro/go-zero/rest"
)

func main() {
	configFile := flag.String("f", "etc/plat-mjml.yaml", "config file path")
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	logx.DisableStat()
	errorx.RegisterErrorHandler()
	prometheus.Enable()

	ctx := svc.NewServiceContext(c)

	// MCP server
	mcpServer := mcp.NewMcpServer(c.McpConf)
	server.RegisterMCPTools(mcpServer, ctx.Renderer, ctx.Queue)

	// UI server (Datastar web UI)
	uiServer := rest.MustNewServer(c.UI.RestConf, rest.WithCors("*"))
	uiHandlers := ui.NewHandlers(ctx.Renderer, ctx.Queue)
	uiServer.AddRoutes(uiHandlers.Routes())
	uiServer.AddRoutes(uiHandlers.SSERoutes(), rest.WithSSE())

	// API server (goctl-generated REST API)
	apiServer := rest.MustNewServer(c.API.RestConf, rest.WithCors("*"))
	handler.RegisterHandlers(apiServer, ctx)
	apiServer.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/metrics",
		Handler: promhttp.Handler().ServeHTTP,
	})

	// Shutdown hooks
	proc.AddShutdownListener(ctx.Close)
	proc.AddShutdownListener(gomjml.StopASTCacheCleanup)

	// Service group: delivery + UI + API + MCP
	group := service.NewServiceGroup()
	group.Add(newDeliveryService(ctx.DeliveryEngine, 2))
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

	group.Start()
}

// deliveryService adapts delivery.Engine to service.Service interface.
type deliveryService struct {
	engine  *delivery.Engine
	workers int
}

func newDeliveryService(engine *delivery.Engine, workers int) *deliveryService {
	return &deliveryService{engine: engine, workers: workers}
}

func (s *deliveryService) Start() { s.engine.Start(s.workers) }
func (s *deliveryService) Stop()  { s.engine.Stop() }
