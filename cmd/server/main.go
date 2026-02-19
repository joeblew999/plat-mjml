package main

import (
	"flag"
	"os"
	"time"

	"github.com/joeblew999/plat-mjml/internal/server"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
)

func main() {
	configFile := flag.String("f", "config.yaml", "config file path")
	flag.Parse()

	logx.DisableStat()

	var c server.Config
	if err := conf.Load(*configFile, &c, conf.UseEnv()); err != nil {
		if os.IsNotExist(err) {
			c = defaultConfig()
		} else {
			logx.Must(err)
		}
	}

	s, err := server.New(c)
	logx.Must(err)

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
	c.UI.Host = "0.0.0.0"
	c.UI.Port = 8081
	c.UI.Name = "plat-mjml-ui"
	c.API.Host = "0.0.0.0"
	c.API.Port = 8082
	c.API.Name = "plat-mjml-api"
	c.Templates = server.TemplatesConfig{Dir: "./templates"}
	c.Fonts = server.FontsConfig{Dir: "./.data/fonts"}
	c.Database = server.DatabaseConfig{Path: "./.data/plat-mjml.db"}
	c.Delivery = server.DeliveryConfig{
		MaxRetries:   3,
		RetryBackoff: "5m",
		MaxBackoff:   "4h",
		RateLimit:    60,
	}
	c.SMTP = server.SMTPConfig{
		Host:     "smtp.gmail.com",
		Port:     "587",
		FromName: "MJML Email",
	}
	return c
}
