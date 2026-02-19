package config

import (
	"github.com/zeromicro/go-zero/mcp"
	"github.com/zeromicro/go-zero/rest"
)

// Config holds the server configuration.
type Config struct {
	mcp.McpConf

	UI        UIConfig        `json:",optional"`
	API       APIConfig       `json:",optional"`
	Templates TemplatesConfig `json:",optional"`
	Fonts     FontsConfig     `json:",optional"`
	Database  DatabaseConfig  `json:",optional"`
	Delivery  DeliveryConfig  `json:",optional"`
	SMTP      SMTPConfig      `json:",optional"`
}

// UIConfig holds the Web UI server settings.
type UIConfig struct {
	rest.RestConf
}

// APIConfig holds the REST API server settings.
type APIConfig struct {
	rest.RestConf
}

// TemplatesConfig holds template directory settings.
type TemplatesConfig struct {
	Dir string `json:",default=./templates"`
}

// DatabaseConfig holds database settings.
type DatabaseConfig struct {
	Path string `json:",default=./.data/plat-mjml.db"`
}

// FontsConfig holds font cache settings.
type FontsConfig struct {
	Dir string `json:",default=./.data/fonts"`
}

// DeliveryConfig holds email delivery settings.
type DeliveryConfig struct {
	MaxRetries   int    `json:",default=3"`
	RetryBackoff string `json:",default=5m"`
	MaxBackoff   string `json:",default=4h"`
	RateLimit    int    `json:",default=60"`
}

// SMTPConfig holds SMTP email delivery settings.
type SMTPConfig struct {
	Host      string `json:",default=smtp.gmail.com"`
	Port      string `json:",default=587"`
	Username  string `json:",optional"`
	Password  string `json:",optional"`
	FromEmail string `json:",optional"`
	FromName  string `json:",optional"`
}
