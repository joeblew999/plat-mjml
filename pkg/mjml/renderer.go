// Package mjml provides email template rendering using MJML (Mailjet Markup Language)
// for generating responsive HTML emails at runtime.
//
// This package combines MJML's email-optimized markup with Go's template system
// to enable dynamic email generation with responsive design and email client compatibility.
package mjml

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/preslavrachev/gomjml/mjml"
	"github.com/joeblew999/plat-mjml/pkg/config"
	"github.com/joeblew999/plat-mjml/pkg/font"
)

// Renderer handles MJML template loading, caching, and rendering
type Renderer struct {
	templates   map[string]*template.Template
	cache       map[string]string // Cache for rendered HTML
	mu          sync.RWMutex
	options     *RenderOptions
	fontManager *font.Manager
}

// RenderOptions configures the MJML renderer behavior
type RenderOptions struct {
	EnableCache      bool   // Cache rendered HTML for performance
	EnableDebug      bool   // Add debug attributes to HTML
	EnableValidation bool   // Validate MJML before rendering
	TemplateDir      string // Default directory for templates
	EnableFonts      bool   // Enable Google Fonts integration
}

// RendererOption configures the renderer
type RendererOption func(*RenderOptions)

// WithCache enables HTML output caching for performance
func WithCache(enabled bool) RendererOption {
	return func(opts *RenderOptions) {
		opts.EnableCache = enabled
	}
}

// WithDebug adds debug attributes to generated HTML
func WithDebug(enabled bool) RendererOption {
	return func(opts *RenderOptions) {
		opts.EnableDebug = enabled
	}
}

// WithValidation enables MJML validation before rendering
func WithValidation(enabled bool) RendererOption {
	return func(opts *RenderOptions) {
		opts.EnableValidation = enabled
	}
}

// WithTemplateDir sets the default template directory
func WithTemplateDir(dir string) RendererOption {
	return func(opts *RenderOptions) {
		opts.TemplateDir = dir
	}
}

// WithFonts enables Google Fonts integration for email templates
func WithFonts(enabled bool) RendererOption {
	return func(opts *RenderOptions) {
		opts.EnableFonts = enabled
	}
}

// NewRenderer creates a new MJML renderer with the specified options
func NewRenderer(opts ...RendererOption) *Renderer {
	options := &RenderOptions{
		EnableCache:      false,
		EnableDebug:      false,
		EnableValidation: true,
		TemplateDir:      config.GetMjmlTemplatePath(),
		EnableFonts:      true,
	}
	
	for _, opt := range opts {
		opt(options)
	}

	renderer := &Renderer{
		templates: make(map[string]*template.Template),
		cache:     make(map[string]string),
		options:   options,
	}

	// Initialize font manager if fonts are enabled
	if options.EnableFonts {
		renderer.fontManager = font.NewManager()
	}

	return renderer
}

// LoadTemplate loads a single MJML template with the given name
func (r *Renderer) LoadTemplate(name, content string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tmpl, err := template.New(name).Parse(content)
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	r.templates[name] = tmpl
	
	// Clear cache for this template
	if r.options.EnableCache {
		delete(r.cache, name)
	}
	
	return nil
}

// LoadTemplateFromFile loads a single MJML template from a file
func (r *Renderer) LoadTemplateFromFile(name, filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read template file %s: %w", filePath, err)
	}
	
	return r.LoadTemplate(name, string(content))
}

// LoadTemplatesFromDir loads all .mjml files from a directory
func (r *Renderer) LoadTemplatesFromDir(dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() || !strings.HasSuffix(path, ".mjml") {
			return nil
		}
		
		// Use filename without extension as template name
		name := strings.TrimSuffix(filepath.Base(path), ".mjml")
		return r.LoadTemplateFromFile(name, path)
	})
}

// RenderTemplate renders a template with the given data to HTML
func (r *Renderer) RenderTemplate(name string, data any) (string, error) {
	r.mu.RLock()
	tmpl, exists := r.templates[name]
	r.mu.RUnlock()
	
	if !exists {
		return "", fmt.Errorf("template %s not found", name)
	}

	// Create deterministic cache key based on template name and data content
	cacheKey, err := r.createCacheKey(name, data)
	if err != nil {
		return "", fmt.Errorf("failed to create cache key for template %s: %w", name, err)
	}
	
	// Check cache if enabled
	if r.options.EnableCache {
		r.mu.RLock()
		if cached, found := r.cache[cacheKey]; found {
			r.mu.RUnlock()
			return cached, nil
		}
		r.mu.RUnlock()
	}

	// Execute template to get MJML
	var mjmlBuf bytes.Buffer
	if err := tmpl.Execute(&mjmlBuf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	mjmlContent := mjmlBuf.String()

	// Convert MJML to HTML
	html, err := r.renderMJML(mjmlContent)
	if err != nil {
		return "", fmt.Errorf("failed to render MJML for template %s: %w", name, err)
	}

	// Cache result if enabled
	if r.options.EnableCache {
		r.mu.Lock()
		r.cache[cacheKey] = html
		r.mu.Unlock()
	}

	return html, nil
}

// RenderString renders MJML content directly to HTML
func (r *Renderer) RenderString(mjmlContent string) (string, error) {
	return r.renderMJML(mjmlContent)
}

// renderMJML converts MJML content to HTML using gomjml
func (r *Renderer) renderMJML(mjmlContent string) (string, error) {
	var mjmlOpts []mjml.RenderOption
	
	if r.options.EnableDebug {
		mjmlOpts = append(mjmlOpts, mjml.WithDebugTags(true))
	}
	
	if r.options.EnableCache {
		mjmlOpts = append(mjmlOpts, mjml.WithCache())
	}

	html, err := mjml.Render(mjmlContent, mjmlOpts...)
	if err != nil {
		return "", fmt.Errorf("gomjml render failed: %w", err)
	}

	return html, nil
}

// ListTemplates returns a list of loaded template names
func (r *Renderer) ListTemplates() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	names := make([]string, 0, len(r.templates))
	for name := range r.templates {
		names = append(names, name)
	}
	return names
}

// HasTemplate checks if a template is loaded
func (r *Renderer) HasTemplate(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	_, exists := r.templates[name]
	return exists
}

// RemoveTemplate removes a template from the renderer
func (r *Renderer) RemoveTemplate(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	delete(r.templates, name)
	
	// Clear cache entries for this template
	if r.options.EnableCache {
		for key := range r.cache {
			if strings.HasPrefix(key, name+"_") {
				delete(r.cache, key)
			}
		}
	}
}

// ClearCache clears all cached rendered HTML
func (r *Renderer) ClearCache() {
	if !r.options.EnableCache {
		return
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.cache = make(map[string]string)
}

// GetCacheSize returns the number of cached HTML entries
func (r *Renderer) GetCacheSize() int {
	if !r.options.EnableCache {
		return 0
	}
	
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	return len(r.cache)
}

// createCacheKey creates a deterministic cache key based on template name and data content
func (r *Renderer) createCacheKey(name string, data any) (string, error) {
	// Serialize data to JSON for consistent hashing
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to serialize data for caching: %w", err)
	}
	
	// Create hash of template name + data content
	hasher := sha256.New()
	hasher.Write([]byte(name))
	hasher.Write(dataBytes)
	hash := fmt.Sprintf("%x", hasher.Sum(nil))
	
	return fmt.Sprintf("%s_%s", name, hash[:16]), nil // Use first 16 chars of hash
}

// LoadFont downloads and caches a Google Font for use in email templates
func (r *Renderer) LoadFont(family string, weight int) error {
	if !r.options.EnableFonts || r.fontManager == nil {
		return fmt.Errorf("font management is disabled")
	}
	
	return r.fontManager.Cache(family, weight)
}

// GetFontCSS generates CSS for embedding a cached font in email templates
func (r *Renderer) GetFontCSS(family string, weight int) (string, error) {
	if !r.options.EnableFonts || r.fontManager == nil {
		return "", fmt.Errorf("font management is disabled")
	}
	
	// Get font path
	path, err := r.fontManager.Get(family, weight)
	if err != nil {
		return "", fmt.Errorf("failed to get font: %w", err)
	}
	
	// Create font object for CSS generation
	fontObj := font.Font{
		Family: family,
		Weight: weight,
		Style:  "normal",
		Format: "woff2",
	}
	
	return font.GetFontCSS(fontObj, path), nil
}

// GetEmailSafeFontStack returns a CSS font stack with email-safe fallbacks
func (r *Renderer) GetEmailSafeFontStack(primaryFont string) string {
	emailSafeFonts := font.GetEmailSafeFonts()
	
	// Build font stack: primary font + email-safe fallbacks
	stack := fmt.Sprintf("'%s'", primaryFont)
	for _, safeFont := range emailSafeFonts {
		// Add relevant fallbacks based on font type
		switch {
		case strings.Contains(strings.ToLower(primaryFont), "sans"):
			if safeFont == "Arial" || safeFont == "Helvetica" || safeFont == "Verdana" {
				stack += fmt.Sprintf(", '%s'", safeFont)
			}
		case strings.Contains(strings.ToLower(primaryFont), "serif"):
			if safeFont == "Georgia" || safeFont == "Times" {
				stack += fmt.Sprintf(", '%s'", safeFont)
			}
		case strings.Contains(strings.ToLower(primaryFont), "mono"):
			if safeFont == "Courier" || safeFont == "Lucida Console" {
				stack += fmt.Sprintf(", '%s'", safeFont)
			}
		}
	}
	
	// Add generic fallback
	if strings.Contains(strings.ToLower(primaryFont), "sans") {
		stack += ", sans-serif"
	} else if strings.Contains(strings.ToLower(primaryFont), "serif") {
		stack += ", serif"
	} else if strings.Contains(strings.ToLower(primaryFont), "mono") {
		stack += ", monospace"
	} else {
		stack += ", sans-serif" // Default fallback
	}
	
	return stack
}

// ListCachedFonts returns all currently cached fonts
func (r *Renderer) ListCachedFonts() []font.FontInfo {
	if !r.options.EnableFonts || r.fontManager == nil {
		return nil
	}
	
	return r.fontManager.List()
}

// IsFontCached checks if a specific font is already cached
func (r *Renderer) IsFontCached(family string, weight int) bool {
	if !r.options.EnableFonts || r.fontManager == nil {
		return false
	}
	
	return r.fontManager.Available(family, weight)
}