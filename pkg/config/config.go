// Package config provides configuration utilities for the MJML package.
package config

import (
	"os"
	"path/filepath"
)

// GetDataPath returns the data directory path.
// It checks for DATA_PATH environment variable, otherwise uses a default.
func GetDataPath() string {
	if path := os.Getenv("DATA_PATH"); path != "" {
		return path
	}

	// Default to current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return filepath.Join(cwd, ".data")
}

// GetMjmlTemplatePath returns the path to MJML templates.
// It checks for MJML_TEMPLATE_PATH environment variable, otherwise uses a default.
func GetMjmlTemplatePath() string {
	if path := os.Getenv("MJML_TEMPLATE_PATH"); path != "" {
		return path
	}

	return filepath.Join(GetDataPath(), "templates")
}

// GetFontPath returns the font cache directory path.
// It checks for FONT_PATH environment variable, otherwise uses a default.
func GetFontPath() string {
	if path := os.Getenv("FONT_PATH"); path != "" {
		return path
	}

	return filepath.Join(GetDataPath(), "fonts")
}

// GetFontPathForFamily returns the path for a specific font family.
func GetFontPathForFamily(family string) string {
	return filepath.Join(GetFontPath(), family)
}
