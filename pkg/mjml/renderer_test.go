package mjml

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestNewRenderer(t *testing.T) {
	renderer := NewRenderer()
	if renderer == nil {
		t.Fatal("NewRenderer returned nil")
	}
}

func TestRendererOptions(t *testing.T) {
	renderer := NewRenderer(
		WithCache(true),
		WithDebug(true),
		WithValidation(false),
		WithFonts(true),
	)
	
	if renderer == nil {
		t.Fatal("NewRenderer with options returned nil")
	}
	
	if !renderer.options.EnableCache {
		t.Error("Cache option not set")
	}
	
	if !renderer.options.EnableDebug {
		t.Error("Debug option not set")
	}
	
	if renderer.options.EnableValidation {
		t.Error("Validation should be disabled")
	}
	
	if !renderer.options.EnableFonts {
		t.Error("Fonts option not set")
	}
}

func TestLoadTemplateFromFile(t *testing.T) {
	renderer := NewRenderer()
	
	// Create a temporary MJML template
	content := `<mjml>
		<mj-head>
			<mj-title>{{.Subject}}</mj-title>
		</mj-head>
		<mj-body>
			<mj-section>
				<mj-column>
					<mj-text>Hello {{.Name}}</mj-text>
				</mj-column>
			</mj-section>
		</mj-body>
	</mjml>`
	
	tmpFile, err := os.CreateTemp("", "test_template_*.mjml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(content)
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	
	err = renderer.LoadTemplateFromFile("test", tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadTemplateFromFile failed: %v", err)
	}
	
	if !renderer.HasTemplate("test") {
		t.Error("Template was not loaded")
	}
}

func TestLoadTemplatesFromFiles(t *testing.T) {
	renderer := NewRenderer()
	
	// Test loading from template directory
	err := renderer.LoadTemplatesFromDir("templates")
	if err != nil {
		t.Logf("Template directory not found (expected in tests): %v", err)
		// This is expected in unit tests - templates are in pkg/mjml/templates
		return
	}
	
	templates := renderer.ListTemplates()
	if len(templates) == 0 {
		t.Error("No templates loaded from directory")
	}
}

func TestRenderTemplate(t *testing.T) {
	renderer := NewRenderer()
	
	// Load a simple template
	renderer.LoadTemplate("simple", `<mjml>
		<mj-head><mj-title>{{.Subject}}</mj-title></mj-head>
		<mj-body>
			<mj-section>
				<mj-column>
					<mj-text>Hello {{.Name}}</mj-text>
				</mj-column>
			</mj-section>
		</mj-body>
	</mjml>`)
	
	testData := TestData()
	data := testData["simple"]
	
	html, err := renderer.RenderTemplate("simple", data)
	if err != nil {
		t.Fatalf("RenderTemplate failed: %v", err)
	}
	
	if html == "" {
		t.Error("RenderTemplate returned empty HTML")
	}
	
	if !strings.Contains(html, "<!doctype html>") {
		t.Error("Generated HTML doesn't contain DOCTYPE")
	}
}

func TestTemplateCache(t *testing.T) {
	renderer := NewRenderer(WithCache(true))
	
	renderer.LoadTemplate("cached", `<mjml>
		<mj-head><mj-title>{{.Subject}}</mj-title></mj-head>
		<mj-body>
			<mj-section>
				<mj-column>
					<mj-text>{{.Message}}</mj-text>
				</mj-column>
			</mj-section>
		</mj-body>
	</mjml>`)
	
	testData := TestData()
	data := testData["simple"]
	
	// First render
	_, err := renderer.RenderTemplate("cached", data)
	if err != nil {
		t.Fatal(err)
	}
	
	// Cache should have one entry
	if renderer.GetCacheSize() != 1 {
		t.Errorf("Expected cache size 1, got %d", renderer.GetCacheSize())
	}
	
	// Second render should use cache
	_, err = renderer.RenderTemplate("cached", data)
	if err != nil {
		t.Fatal(err)
	}
	
	// Cache size should still be 1
	if renderer.GetCacheSize() != 1 {
		t.Errorf("Expected cache size 1, got %d", renderer.GetCacheSize())
	}
}

func TestEmailDataStructs(t *testing.T) {
	now := time.Now()
	
	emailData := EmailData{
		Name:        "Test User",
		Email:       "test@example.com",
		Subject:     "Test Email",
		Timestamp:   now,
		CompanyName: "Test Company",
		Title:       "Test Title",
		Message:     "Test message",
	}
	
	if emailData.Name != "Test User" {
		t.Error("EmailData fields not set correctly")
	}
	
	welcomeData := WelcomeEmailData{
		EmailData:     emailData,
		ActivationURL: "https://example.com/activate",
	}
	
	if welcomeData.ActivationURL != "https://example.com/activate" {
		t.Error("WelcomeEmailData fields not set correctly")
	}
}

func TestCanonicalTestDataStructures(t *testing.T) {
	testData := TestData()
	
	expectedTemplates := []string{"welcome", "reset_password", "notification", "simple", "premium_newsletter"}
	
	for _, name := range expectedTemplates {
		data, exists := testData[name]
		if !exists {
			t.Errorf("Test data for %s not found", name)
		}
		
		if data == nil {
			t.Errorf("Test data for %s is nil", name)
		}
		
		// Check that data has expected structure based on type
		switch v := data.(type) {
		case EmailData:
			if v.Name == "" || v.CompanyName == "" {
				t.Errorf("EmailData for %s missing required fields", name)
			}
		case WelcomeEmailData:
			if v.Name == "" || v.ActivationURL == "" {
				t.Errorf("WelcomeEmailData for %s missing required fields", name)
			}
		case ResetPasswordData:
			if v.Name == "" || v.ResetURL == "" {
				t.Errorf("ResetPasswordData for %s missing required fields", name)
			}
		case NewsletterData:
			if v.Name == "" || len(v.ContentBlocks) == 0 {
				t.Errorf("NewsletterData for %s missing required fields", name)
			}
		}
	}
}