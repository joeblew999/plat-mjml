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
	FontDir          string // Font cache directory (empty = default from config)
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

// WithFontDir sets the font cache directory. If not set, uses config default.
func WithFontDir(dir string) RendererOption {
	return func(opts *RenderOptions) {
		opts.FontDir = dir
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
		if options.FontDir != "" {
			renderer.fontManager = font.NewManagerWithDir(options.FontDir)
		} else {
			renderer.fontManager = font.NewManager()
		}
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

// ReplaceTemplatesFromDir atomically replaces all templates by loading from a
// directory. This holds the write lock for the entire operation so no requests
// see a partially-loaded state.
func (r *Renderer) ReplaceTemplatesFromDir(dir string) error {
	newTemplates := make(map[string]*template.Template)

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".mjml") {
			return nil
		}

		name := strings.TrimSuffix(filepath.Base(path), ".mjml")
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read template file %s: %w", path, err)
		}

		tmpl, err := template.New(name).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", name, err)
		}

		newTemplates[name] = tmpl
		return nil
	})
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.templates = newTemplates
	r.cache = make(map[string]string)
	r.mu.Unlock()

	return nil
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

// GetFontCSS generates @font-face CSS for a font, using CDN URLs for email compatibility.
// The font is downloaded/cached if not already available.
func (r *Renderer) GetFontCSS(family string, weight int) (string, error) {
	if !r.options.EnableFonts || r.fontManager == nil {
		return "", fmt.Errorf("font management is disabled")
	}

	// Ensure font is cached (downloads if needed)
	if _, err := r.fontManager.Get(family, weight); err != nil {
		return "", fmt.Errorf("failed to get font: %w", err)
	}

	// Get full info including CDN URL
	info, ok := r.fontManager.GetInfo(family, weight)
	if !ok {
		return "", fmt.Errorf("font %s %d not found after caching", family, weight)
	}

	return font.GetFontCSS(info), nil
}

// GetEmailSafeFontStack returns a CSS font stack with email-safe fallbacks
func (r *Renderer) GetEmailSafeFontStack(primaryFont string) string {
	lower := strings.ToLower(primaryFont)

	// Classify font type — most Google Fonts are sans-serif
	fontType := "sans" // default
	if strings.Contains(lower, "serif") && !strings.Contains(lower, "sans") {
		fontType = "serif"
	} else if strings.Contains(lower, "mono") || strings.Contains(lower, "code") || strings.Contains(lower, "courier") {
		fontType = "mono"
	}

	stack := fmt.Sprintf("'%s'", primaryFont)

	switch fontType {
	case "serif":
		stack += ", Georgia, 'Times New Roman', Times, serif"
	case "mono":
		stack += ", 'Courier New', Courier, 'Lucida Console', monospace"
	default: // sans
		stack += ", Arial, Helvetica, sans-serif"
	}

	return stack
}

// PrepareFontData generates FontCSS and FontStack for a given font family,
// ready to inject into template data. Downloads the font if not cached.
func (r *Renderer) PrepareFontData(family string, weight int) (fontCSS, fontStack string, err error) {
	fontStack = r.GetEmailSafeFontStack(family)

	css, err := r.GetFontCSS(family, weight)
	if err != nil {
		// Font CSS is optional — return stack with empty CSS
		return "", fontStack, nil
	}

	return css, fontStack, nil
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