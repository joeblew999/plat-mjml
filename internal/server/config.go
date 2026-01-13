package server

import "github.com/zeromicro/go-zero/mcp"

// Config holds the server configuration.
type Config struct {
	mcp.McpConf

	Templates TemplatesConfig `json:",optional"`
	Database  DatabaseConfig  `json:",optional"`
	Delivery  DeliveryConfig  `json:",optional"`
}

// TemplatesConfig holds template directory settings.
type TemplatesConfig struct {
	Dir string `json:",default=./templates"`
}

// DatabaseConfig holds database settings.
type DatabaseConfig struct {
	Path string `json:",default=./.data/plat-mjml.db"`
}

// DeliveryConfig holds email delivery settings.
type DeliveryConfig struct {
	MaxRetries   int    `json:",default=3"`
	RetryBackoff string `json:",default=5m"`
	MaxBackoff   string `json:",default=4h"`
	RateLimit    int    `json:",default=60"`
}
