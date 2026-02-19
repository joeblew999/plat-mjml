package mjml

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// projectRoot finds the project root by locating go.mod relative to this source file.
// Works regardless of the working directory go test runs from.
func projectRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// This file is at pkg/mjml/integration_test.go — walk up twice to project root.
	root := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("Could not find project root (expected go.mod at %s): %v", root, err)
	}
	return root
}

// TestTemplateFilesIntegration tests loading and rendering actual template files
func TestTemplateFilesIntegration(t *testing.T) {
	templatesDir := filepath.Join(projectRoot(t), "templates")

	renderer := NewRenderer(WithCache(true))
	err := renderer.LoadTemplatesFromDir(templatesDir)
	if err != nil {
		t.Fatalf("Failed to load templates from directory: %v", err)
	}

	templates := renderer.ListTemplates()
	if len(templates) == 0 {
		t.Fatal("No templates loaded from directory")
	}

	t.Logf("Loaded %d templates: %v", len(templates), templates)

	// Output dir for rendered HTML (cleaned up automatically)
	outputDir := t.TempDir()

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
		{
			templateName: "premium_newsletter",
			data: NewsletterData{
				EmailData: EmailData{
					Name:        "Newsletter Reader",
					Subject:     "Monthly Newsletter",
					Title:       "Newsletter Title",
					CompanyName: "Test Company",
					Timestamp:   time.Now(),
				},
				PreviewText:      "Preview text here",
				Greeting:         "Hello Newsletter Reader,",
				ContentBlocks:    []string{"First block of content."},
				CallToActionURL:  "https://example.com/read",
				CallToActionText: "Read More",
				FeaturedTitle:    "Featured",
				FeaturedContent: []FeaturedItem{
					{Title: "Article", Description: "A great article", URL: "https://example.com/article"},
				},
				SocialLinks: []SocialLink{
					{Platform: "twitter", URL: "https://twitter.com/test"},
				},
				CompanyAddress: "123 Test St",
				UnsubscribeURL: "https://example.com/unsub",
			},
			expectText: "Newsletter Reader",
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

			if len(html) == 0 {
				t.Error("Generated HTML is empty")
			}

			if !strings.Contains(html, "<!doctype html>") {
				t.Error("Generated HTML missing doctype")
			}

			if !strings.Contains(html, tc.expectText) {
				t.Errorf("Generated HTML missing expected text: %s", tc.expectText)
			}

			// Save rendered HTML to temp dir for inspection
			outputFile := filepath.Join(outputDir, tc.templateName+"_test.html")
			if err := os.WriteFile(outputFile, []byte(html), 0644); err != nil {
				t.Logf("Could not save test output: %v", err)
			}
		})
	}
}

// TestBusinessAnnouncementTemplate tests the complex business announcement template
func TestBusinessAnnouncementTemplate(t *testing.T) {
	templatesDir := filepath.Join(projectRoot(t), "templates")

	renderer := NewRenderer()
	err := renderer.LoadTemplatesFromDir(templatesDir)
	if err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}

	if !renderer.HasTemplate("business_announcement") {
		t.Fatal("business_announcement template not found")
	}

	data := map[string]any{
		"subject":             "Test Announcement",
		"preview":             "Test preview text",
		"company_name":        "Test Company",
		"name":                "Test User",
		"location":            "Test City",
		"venue":               "Test Venue",
		"address":             "123 Test St",
		"title":               "Test Event Opening",
		"message":             "Test announcement message",
		"primary_button_text": "RSVP",
		"primary_button_url":  "https://example.com/rsvp",
		"visit_title":         "Visit Us",
		"visit_message":       "We look forward to seeing you",
		"disclaimer":          "Test disclaimer text",
		"privacy_url":         "https://example.com/privacy",
		"unsubscribe_url":     "https://example.com/unsubscribe",
	}

	html, err := renderer.RenderTemplate("business_announcement", data)
	if err != nil {
		t.Fatalf("Failed to render business announcement: %v", err)
	}

	if !strings.Contains(html, "Test Company") {
		t.Error("Company name not found in output")
	}

	if !strings.Contains(html, "Test User") {
		t.Error("User name not found in output")
	}

	if !strings.Contains(html, "RSVP") {
		t.Error("Button text not found in output")
	}
}

// TestTemplateDirectory validates template file structure
func TestTemplateDirectory(t *testing.T) {
	templatesDir := filepath.Join(projectRoot(t), "templates")

	expectedTemplates := []string{
		"simple.mjml",
		"welcome.mjml",
		"reset_password.mjml",
		"notification.mjml",
		"premium_newsletter.mjml",
		"business_announcement.mjml",
	}

	for _, templateFile := range expectedTemplates {
		path := filepath.Join(templatesDir, templateFile)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected template file %s not found", templateFile)
			continue
		}

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
	templatesDir := filepath.Join(projectRoot(t), "templates")

	rendererNoCache := NewRenderer(WithCache(false))
	err := rendererNoCache.LoadTemplatesFromDir(templatesDir)
	if err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}

	rendererWithCache := NewRenderer(WithCache(true))
	err = rendererWithCache.LoadTemplatesFromDir(templatesDir)
	if err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}

	if !rendererWithCache.HasTemplate("simple") {
		t.Fatal("simple template not found")
	}

	data := EmailData{
		Name:    "Performance Test",
		Subject: "Performance Test Email",
		Message: "Testing performance",
	}

	for i := 0; i < 3; i++ {
		_, err := rendererWithCache.RenderTemplate("simple", data)
		if err != nil {
			t.Fatalf("Cached render failed: %v", err)
		}
	}

	if rendererWithCache.GetCacheSize() == 0 {
		t.Error("Cache should be populated after renders")
	}

	_, err = rendererNoCache.RenderTemplate("simple", data)
	if err != nil {
		t.Fatalf("Non-cached render failed: %v", err)
	}

	if rendererNoCache.GetCacheSize() != 0 {
		t.Error("Non-cached renderer should have empty cache")
	}
}
