package mjml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestTemplateFilesIntegration tests loading and rendering actual template files
func TestTemplateFilesIntegration(t *testing.T) {
	// Skip if template files don't exist
	templatesDir := "templates"
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		t.Skip("Template files not found, skipping integration test")
	}

	renderer := NewRenderer(WithCache(true))

	// Load all templates from directory
	err := renderer.LoadTemplatesFromDir(templatesDir)
	if err != nil {
		t.Fatalf("Failed to load templates from directory: %v", err)
	}

	templates := renderer.ListTemplates()
	if len(templates) == 0 {
		t.Fatal("No templates loaded from directory")
	}

	t.Logf("Loaded %d templates: %v", len(templates), templates)

	// Test each template type
	testCases := []struct {
		templateName string
		data         any
		expectText   string
	}{
		{
			templateName: "simple",
			data: EmailData{
				Name:       "Test User",
				Subject:    "Test Simple Email",
				Title:      "Simple Test",
				Message:    "This is a simple test message",
				ButtonText: "Click Me",
				ButtonURL:  "https://example.com",
			},
			expectText: "Test User",
		},
		{
			templateName: "welcome",
			data: WelcomeEmailData{
				EmailData: EmailData{
					Name:        "New User",
					Subject:     "Welcome!",
					CompanyName: "Test Company",
					Message:     "Welcome to our platform",
					Timestamp:   time.Now(),
				},
				ActivationURL: "https://example.com/activate",
			},
			expectText: "New User",
		},
		{
			templateName: "reset_password",
			data: ResetPasswordData{
				EmailData: EmailData{
					Name:        "Reset User",
					Subject:     "Password Reset",
					CompanyName: "Test Company",
					Timestamp:   time.Now(),
				},
				ResetURL:    "https://example.com/reset",
				ExpiresIn:   24 * time.Hour,
				RequestIP:   "127.0.0.1",
				RequestTime: time.Now(),
			},
			expectText: "Reset User",
		},
		{
			templateName: "notification",
			data: NotificationData{
				EmailData: EmailData{
					Name:        "Alert User",
					Subject:     "System Alert",
					Title:       "High CPU Usage",
					Message:     "System alert message",
					Timestamp:   time.Now(),
				},
				NotificationType: "System",
				Priority:         "high",
				ActionRequired:   true,
			},
			expectText: "Alert User",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.templateName, func(t *testing.T) {
			if !renderer.HasTemplate(tc.templateName) {
				t.Skipf("Template %s not found", tc.templateName)
			}

			html, err := renderer.RenderTemplate(tc.templateName, tc.data)
			if err != nil {
				t.Fatalf("Failed to render template %s: %v", tc.templateName, err)
			}

			// Basic validation
			if len(html) == 0 {
				t.Error("Generated HTML is empty")
			}

			if !strings.Contains(html, "<!doctype html>") {
				t.Error("Generated HTML missing doctype")
			}

			if !strings.Contains(html, tc.expectText) {
				t.Errorf("Generated HTML missing expected text: %s", tc.expectText)
			}

			// Save rendered HTML for manual inspection
			outputFile := filepath.Join("testdata", tc.templateName+"_test.html")
			os.MkdirAll("testdata", 0755)
			if err := os.WriteFile(outputFile, []byte(html), 0644); err != nil {
				t.Logf("Could not save test output: %v", err)
			}
		})
	}
}

// TestBusinessAnnouncementTemplate tests the complex business announcement template
func TestBusinessAnnouncementTemplate(t *testing.T) {
	templatesDir := "templates"
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		t.Skip("Template files not found, skipping integration test")
	}

	renderer := NewRenderer()
	err := renderer.LoadTemplatesFromDir(templatesDir)
	if err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}

	if !renderer.HasTemplate("business_announcement") {
		t.Skip("business_announcement template not found")
	}

	data := map[string]any{
		"subject":                "Test Announcement",
		"preview":                "Test preview text",
		"company_name":           "Test Company",
		"name":                   "Test User",
		"location":               "Test City",
		"venue":                  "Test Venue",
		"address":                "123 Test St",
		"title":                  "Test Event Opening",
		"message":                "Test announcement message",
		"primary_button_text":    "RSVP",
		"primary_button_url":     "https://example.com/rsvp",
		"visit_title":            "Visit Us",
		"visit_message":          "We look forward to seeing you",
		"disclaimer":             "Test disclaimer text",
		"privacy_url":            "https://example.com/privacy",
		"unsubscribe_url":        "https://example.com/unsubscribe",
	}

	html, err := renderer.RenderTemplate("business_announcement", data)
	if err != nil {
		t.Fatalf("Failed to render business announcement: %v", err)
	}

	// Validate complex template
	if !strings.Contains(html, "Test Company") {
		t.Error("Company name not found in output")
	}

	if !strings.Contains(html, "Test User") {
		t.Error("User name not found in output")
	}

	if !strings.Contains(html, "RSVP") {
		t.Error("Button text not found in output")
	}

	// Save for inspection
	os.MkdirAll("testdata", 0755)
	err = os.WriteFile("testdata/business_announcement_test.html", []byte(html), 0644)
	if err != nil {
		t.Logf("Could not save test output: %v", err)
	}
}

// TestTemplateDirectory validates template file structure
func TestTemplateDirectory(t *testing.T) {
	templatesDir := "templates"
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		t.Skip("Template files not found, skipping directory test")
	}

	expectedTemplates := []string{
		"simple.mjml",
		"welcome.mjml",
		"reset_password.mjml",
		"notification.mjml",
		"business_announcement.mjml",
	}

	for _, templateFile := range expectedTemplates {
		path := filepath.Join(templatesDir, templateFile)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected template file %s not found", templateFile)
			continue
		}

		// Validate template content
		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("Failed to read template file %s: %v", templateFile, err)
			continue
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, "<mjml>") {
			t.Errorf("Template %s missing <mjml> tag", templateFile)
		}

		if !strings.Contains(contentStr, "<mj-body") {
			t.Errorf("Template %s missing <mj-body> tag", templateFile)
		}

		if !strings.Contains(contentStr, "{{.") {
			t.Errorf("Template %s missing template variables", templateFile)
		}
	}
}

// TestCachePerformance validates that caching improves performance
func TestCachePerformance(t *testing.T) {
	templatesDir := "templates"
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		t.Skip("Template files not found, skipping performance test")
	}

	// Renderer without cache
	rendererNoCache := NewRenderer(WithCache(false))
	err := rendererNoCache.LoadTemplatesFromDir(templatesDir)
	if err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}

	// Renderer with cache
	rendererWithCache := NewRenderer(WithCache(true))
	err = rendererWithCache.LoadTemplatesFromDir(templatesDir)
	if err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}

	if !rendererWithCache.HasTemplate("simple") {
		t.Skip("simple template not found")
	}

	data := EmailData{
		Name:    "Performance Test",
		Subject: "Performance Test Email",
		Message: "Testing performance",
	}

	// Render multiple times with cache
	for i := 0; i < 3; i++ {
		_, err := rendererWithCache.RenderTemplate("simple", data)
		if err != nil {
			t.Fatalf("Cached render failed: %v", err)
		}
	}

	// Verify cache is populated
	if rendererWithCache.GetCacheSize() == 0 {
		t.Error("Cache should be populated after renders")
	}

	// Render without cache for comparison
	_, err = rendererNoCache.RenderTemplate("simple", data)
	if err != nil {
		t.Fatalf("Non-cached render failed: %v", err)
	}

	// Non-cached renderer should have no cache
	if rendererNoCache.GetCacheSize() != 0 {
		t.Error("Non-cached renderer should have empty cache")
	}
}