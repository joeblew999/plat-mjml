package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/joeblew999/plat-mjml/internal/server"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/mcp"
)

func main() {
	configFile := flag.String("f", "config.yaml", "config file path")
	flag.Parse()

	// Disable go-zero's stat logging
	logx.DisableStat()

	var c server.Config
	if err := conf.Load(*configFile, &c); err != nil {
		// If config file doesn't exist, use defaults
		if os.IsNotExist(err) {
			c = defaultConfig()
		} else {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}
	}

	s, err := server.New(c)
	if err != nil {
		fmt.Printf("Error creating server: %v\n", err)
		os.Exit(1)
	}
	defer s.Stop()

	s.Start()
}

func defaultConfig() server.Config {
	c := server.Config{}
	c.Name = "plat-mjml"
	c.Host = "0.0.0.0"
	c.Port = 8080
	c.Mcp.Name = "mjml-server"
	c.Mcp.Version = "1.0.0"
	c.Mcp.MessageTimeout = 30 * time.Second
	c.Templates = server.TemplatesConfig{Dir: "./templates"}
	c.Database = server.DatabaseConfig{Path: "./.data/plat-mjml.db"}
	c.Delivery = server.DeliveryConfig{
		MaxRetries:   3,
		RetryBackoff: "5m",
		MaxBackoff:   "4h",
		RateLimit:    60,
	}
	return c
}

// Ensure mcp package is imported for the type embedding
var _ = mcp.McpConf{}
