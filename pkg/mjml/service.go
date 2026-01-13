package mjml

import (
	"fmt"
	"path/filepath"

	"github.com/joeblew999/plat-mjml/pkg/config"
	"github.com/joeblew999/plat-mjml/pkg/log"
)

// Service provides email template rendering capabilities for the infrastructure
type Service struct {
	renderer    *Renderer
	templateDir string
}

// NewService creates a new MJML service with default configuration
func NewService() *Service {
	templateDir := filepath.Join(config.GetDataPath(), "email-templates")
	
	renderer := NewRenderer(
		WithCache(true),
		WithDebug(false),
		WithValidation(true),
		WithTemplateDir(templateDir),
	)

	return &Service{
		renderer:    renderer,
		templateDir: templateDir,
	}
}

// NewServiceWithOptions creates a new MJML service with custom options
func NewServiceWithOptions(templateDir string, opts ...RendererOption) *Service {
	renderer := NewRenderer(opts...)
	
	return &Service{
		renderer:    renderer,
		templateDir: templateDir,
	}
}

// Start initializes the MJML service and loads templates
func (s *Service) Start() error {
	log.Info("Starting MJML email service", "template_dir", s.templateDir)
	
	// Load templates from directory
	if err := s.renderer.LoadTemplatesFromDir(s.templateDir); err != nil {
		return fmt.Errorf("failed to load templates from directory %s: %w", s.templateDir, err)
	}
	
	templates := s.renderer.ListTemplates()
	log.Info("MJML service started", "templates_loaded", len(templates), "templates", templates)
	
	return nil
}

// Stop gracefully shuts down the MJML service
func (s *Service) Stop() error {
	log.Info("Stopping MJML email service")
	s.renderer.ClearCache()
	return nil
}

// RenderEmail renders an email template with the provided data
func (s *Service) RenderEmail(templateName string, data any) (string, error) {
	html, err := s.renderer.RenderTemplate(templateName, data)
	if err != nil {
		log.Error("Failed to render email template", "template", templateName, "error", err)
		return "", fmt.Errorf("failed to render email template %s: %w", templateName, err)
	}
	
	log.Debug("Email template rendered successfully", "template", templateName, "html_size", len(html))
	return html, nil
}

// RenderEmailString renders MJML content directly to HTML
func (s *Service) RenderEmailString(mjmlContent string) (string, error) {
	html, err := s.renderer.RenderString(mjmlContent)
	if err != nil {
		log.Error("Failed to render MJML string", "error", err)
		return "", fmt.Errorf("failed to render MJML string: %w", err)
	}
	
	log.Debug("MJML string rendered successfully", "html_size", len(html))
	return html, nil
}

// LoadTemplate loads a new template into the service
func (s *Service) LoadTemplate(name, content string) error {
	if err := s.renderer.LoadTemplate(name, content); err != nil {
		log.Error("Failed to load template", "name", name, "error", err)
		return fmt.Errorf("failed to load template %s: %w", name, err)
	}
	
	log.Info("Template loaded successfully", "name", name)
	return nil
}

// LoadTemplateFromFile loads a template from a file
func (s *Service) LoadTemplateFromFile(name, filePath string) error {
	if err := s.renderer.LoadTemplateFromFile(name, filePath); err != nil {
		log.Error("Failed to load template from file", "name", name, "file", filePath, "error", err)
		return fmt.Errorf("failed to load template %s from file %s: %w", name, filePath, err)
	}
	
	log.Info("Template loaded from file", "name", name, "file", filePath)
	return nil
}

// ReloadTemplates reloads all templates from the template directory
func (s *Service) ReloadTemplates() error {
	log.Info("Reloading templates", "dir", s.templateDir)
	
	// Clear existing templates
	templates := s.renderer.ListTemplates()
	for _, tmpl := range templates {
		s.renderer.RemoveTemplate(tmpl)
	}
	
	// Reload from directory
	if err := s.renderer.LoadTemplatesFromDir(s.templateDir); err != nil {
		log.Error("Failed to reload templates", "error", err)
		return fmt.Errorf("failed to reload templates: %w", err)
	}
	
	// Clear cache to ensure fresh renders
	s.renderer.ClearCache()
	
	newTemplates := s.renderer.ListTemplates()
	log.Info("Templates reloaded", "count", len(newTemplates), "templates", newTemplates)
	
	return nil
}

// ListTemplates returns the names of all loaded templates
func (s *Service) ListTemplates() []string {
	return s.renderer.ListTemplates()
}

// HasTemplate checks if a template is loaded
func (s *Service) HasTemplate(name string) bool {
	return s.renderer.HasTemplate(name)
}

// GetCacheStats returns cache statistics
func (s *Service) GetCacheStats() map[string]any {
	return map[string]any{
		"cache_size":    s.renderer.GetCacheSize(),
		"cache_enabled": s.renderer.options.EnableCache,
		"templates":     len(s.renderer.templates),
	}
}

// Health checks the health of the MJML service
func (s *Service) Health() map[string]any {
	templates := s.renderer.ListTemplates()
	
	status := "healthy"
	if len(templates) == 0 {
		status = "unhealthy"
	}
	
	return map[string]any{
		"status":         status,
		"templates":      len(templates),
		"template_names": templates,
		"cache_size":     s.renderer.GetCacheSize(),
		"template_dir":   s.templateDir,
	}
}