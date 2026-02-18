package font

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joeblew999/plat-mjml/pkg/config"
	"github.com/joeblew999/plat-mjml/pkg/log"
)

// Font represents a font with family, weight, and style
type Font struct {
	Family string
	Weight int    // 100, 200, 300, 400, 500, 600, 700, 800, 900
	Style  string // normal, italic
	Format string // woff2, woff, ttf
}

// FontInfo contains metadata about a cached font
type FontInfo struct {
	Font
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Version string `json:"version"`
	Source  string `json:"source"`  // "google", "local"
	CDNURL  string `json:"cdn_url"` // CDN URL for use in email @font-face rules
}

// Manager handles font operations
type Manager struct {
	cacheDir string
	registry *Registry
}

// GetLocalFontPath returns the font cache path (environment-aware via config)
func GetLocalFontPath() string {
	return config.GetFontPath()
}

// GetLocalFontPathForFamily returns the path for a specific font family
func GetLocalFontPathForFamily(family string) string {
	return config.GetFontPathForFamily(family)
}

// newFont creates a Font struct with defaults
func newFont(family string, weight int, format string) Font {
	return Font{
		Family: family,
		Weight: weight,
		Style:  DefaultFontStyle,
		Format: format,
	}
}

// newDefaultFont creates a Font struct with default format
func newDefaultFont(family string, weight int) Font {
	return newFont(family, weight, DefaultFontFormat)
}

// NewManager creates a new font manager using the default cache path (environment-aware via config).
func NewManager() *Manager {
	return NewManagerWithDir(GetLocalFontPath())
}

// NewManagerWithDir creates a new font manager with an explicit cache directory.
// Use this in tests with t.TempDir() to avoid polluting the source tree.
func NewManagerWithDir(cacheDir string) *Manager {
	registryPath := filepath.Join(cacheDir, RegistryFilename)
	return &Manager{
		cacheDir: cacheDir,
		registry: NewRegistryAt(registryPath),
	}
}

// Get returns the path to a cached font, downloading if necessary
func (m *Manager) Get(family string, weight int) (string, error) {
	return m.GetFormat(family, weight, DefaultFontFormat)
}

// GetFormat returns the path to a cached font in a specific format
func (m *Manager) GetFormat(family string, weight int, format string) (string, error) {
	font := newFont(family, weight, format)

	// Check if font is already cached
	if path, exists := m.registry.GetPath(font); exists {
		return path, nil
	}

	// Download and cache the font
	return m.cacheFont(font)
}

// List returns all available cached fonts
func (m *Manager) List() []FontInfo {
	return m.registry.List()
}

// Cache downloads and caches a font
func (m *Manager) Cache(family string, weight int) error {
	_, err := m.cacheFont(newDefaultFont(family, weight))
	return err
}

// Available checks if a font is cached
func (m *Manager) Available(family string, weight int) bool {
	_, exists := m.registry.GetPath(newDefaultFont(family, weight))
	return exists
}

// CacheTTF downloads and caches a font in TTF format (for deck tools)
func (m *Manager) CacheTTF(family string, weight int) error {
	_, err := m.cacheFont(newFont(family, weight, "ttf"))
	return err
}

// GetTTF returns the path to a cached TTF font, downloading if necessary
func (m *Manager) GetTTF(family string, weight int) (string, error) {
	return m.GetFormat(family, weight, "ttf")
}

// cacheFont downloads and caches a font
func (m *Manager) cacheFont(font Font) (string, error) {
	// Ensure cache directory exists and get file path
	path, err := m.prepareFontPath(font)
	if err != nil {
		return "", err
	}

	// Download from Google Fonts (returns CDN URL for email use)
	cdnURL, err := downloadGoogleFont(font, path)
	if err != nil {
		return "", fmt.Errorf("failed to download font: %w", err)
	}

	// Register in registry with CDN URL
	if err := m.registerFont(font, path, cdnURL); err != nil {
		log.Warn("Failed to register font", "error", err)
	}

	return path, nil
}

// prepareFontPath ensures directory exists and returns the full file path
func (m *Manager) prepareFontPath(font Font) (string, error) {
	familyDir := filepath.Join(m.cacheDir, font.Family)
	if err := ensureDir(familyDir); err != nil {
		return "", fmt.Errorf("failed to create font directory: %w", err)
	}

	filename := fmt.Sprintf("%d.%s", font.Weight, font.Format)
	return filepath.Join(familyDir, filename), nil
}

// registerFont adds font info to the registry
func (m *Manager) registerFont(font Font, path string, cdnURL string) error {
	info := FontInfo{
		Font:    font,
		Path:    path,
		Source:  "google",
		Version: "latest",
		CDNURL:  cdnURL,
	}
	return m.registry.Add(info)
}

// GetInfo returns the full FontInfo for a cached font
func (m *Manager) GetInfo(family string, weight int) (FontInfo, bool) {
	font := newDefaultFont(family, weight)
	return m.registry.GetInfo(font)
}

// ensureDir creates a directory if it doesn't exist
func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}
