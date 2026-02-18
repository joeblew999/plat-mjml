package mjml

import (
	"fmt"

	"github.com/joeblew999/plat-mjml/pkg/config"
	"github.com/zeromicro/go-zero/core/logx"
)

// Service provides email template rendering capabilities for the infrastructure
type Service struct {
	renderer    *Renderer
	templateDir string
}

// NewService creates a new MJML service with default configuration
func NewService() *Service {
	templateDir := config.GetMjmlTemplatePath()
	
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
	logx.Infow("Starting MJML email service", logx.Field("template_dir", s.templateDir))
	
	// Load templates from directory
	if err := s.renderer.LoadTemplatesFromDir(s.templateDir); err != nil {
		return fmt.Errorf("failed to load templates from directory %s: %w", s.templateDir, err)
	}
	
	templates := s.renderer.ListTemplates()
	logx.Infow("MJML service started", logx.Field("templates_loaded", len(templates)), logx.Field("templates", templates))
	
	return nil
}

// Stop gracefully shuts down the MJML service
func (s *Service) Stop() error {
	logx.Info("Stopping MJML email service")
	s.renderer.ClearCache()
	return nil
}

// RenderEmail renders an email template with the provided data
func (s *Service) RenderEmail(templateName string, data any) (string, error) {
	html, err := s.renderer.RenderTemplate(templateName, data)
	if err != nil {
		logx.Errorw("Failed to render email template", logx.Field("template", templateName), logx.Field("error", err))
		return "", fmt.Errorf("failed to render email template %s: %w", templateName, err)
	}
	
	logx.Debugw("Email template rendered successfully", logx.Field("template", templateName), logx.Field("html_size", len(html)))
	return html, nil
}

// RenderEmailString renders MJML content directly to HTML
func (s *Service) RenderEmailString(mjmlContent string) (string, error) {
	html, err := s.renderer.RenderString(mjmlContent)
	if err != nil {
		logx.Errorw("Failed to render MJML string", logx.Field("error", err))
		return "", fmt.Errorf("failed to render MJML string: %w", err)
	}
	
	logx.Debugw("MJML string rendered successfully", logx.Field("html_size", len(html)))
	return html, nil
}

// LoadTemplate loads a new template into the service
func (s *Service) LoadTemplate(name, content string) error {
	if err := s.renderer.LoadTemplate(name, content); err != nil {
		logx.Errorw("Failed to load template", logx.Field("name", name), logx.Field("error", err))
		return fmt.Errorf("failed to load template %s: %w", name, err)
	}
	
	logx.Infow("Template loaded successfully", logx.Field("name", name))
	return nil
}

// LoadTemplateFromFile loads a template from a file
func (s *Service) LoadTemplateFromFile(name, filePath string) error {
	if err := s.renderer.LoadTemplateFromFile(name, filePath); err != nil {
		logx.Errorw("Failed to load template from file", logx.Field("name", name), logx.Field("file", filePath), logx.Field("error", err))
		return fmt.Errorf("failed to load template %s from file %s: %w", name, filePath, err)
	}
	
	logx.Infow("Template loaded from file", logx.Field("name", name), logx.Field("file", filePath))
	return nil
}

// ReloadTemplates reloads all templates from the template directory.
// It clears and reloads atomically under a single lock to avoid serving
// requests while templates are partially loaded.
func (s *Service) ReloadTemplates() error {
	logx.Infow("Reloading templates", logx.Field("dir", s.templateDir))

	// Clear and reload atomically
	if err := s.renderer.ReplaceTemplatesFromDir(s.templateDir); err != nil {
		logx.Errorw("Failed to reload templates", logx.Field("error", err))
		return fmt.Errorf("failed to reload templates: %w", err)
	}

	newTemplates := s.renderer.ListTemplates()
	logx.Infow("Templates reloaded", logx.Field("count", len(newTemplates)), logx.Field("templates", newTemplates))

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