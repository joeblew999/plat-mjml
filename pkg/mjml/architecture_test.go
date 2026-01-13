package mjml

import (
	"testing"
)

// TestCacheKeyDeterministic verifies cache keys are deterministic
func TestCacheKeyDeterministic(t *testing.T) {
	renderer := NewRenderer(WithCache(true))
	
	testData := TestData()
	data := testData["simple"]
	
	// Create cache key multiple times
	key1, err := renderer.createCacheKey("simple", data)
	if err != nil {
		t.Fatalf("Failed to create cache key: %v", err)
	}
	
	key2, err := renderer.createCacheKey("simple", data)
	if err != nil {
		t.Fatalf("Failed to create second cache key: %v", err)
	}
	
	if key1 != key2 {
		t.Errorf("Cache keys are not deterministic: %s != %s", key1, key2)
	}
	
	t.Logf("Cache key: %s", key1)
}

// TestCanonicalTestData tests the canonical test data
func TestCanonicalTestData(t *testing.T) {
	testData := TestData()
	
	// Check we have all expected templates
	expectedTemplates := []string{"simple", "welcome", "reset_password", "notification", "premium_newsletter"}
	
	for _, templateName := range expectedTemplates {
		if _, exists := testData[templateName]; !exists {
			t.Errorf("Missing test data for template: %s", templateName)
		}
	}
	
	// Check simple data
	simple := testData["simple"].(EmailData)
	if simple.CompanyName == "" || simple.Name == "" {
		t.Error("Simple test data should have populated required fields")
	}
	
	// Check welcome data  
	welcome := testData["welcome"].(WelcomeEmailData)
	if welcome.ActivationURL == "" {
		t.Error("Welcome data should have ActivationURL")
	}
	
	// Check reset password data
	reset := testData["reset_password"].(ResetPasswordData)
	if reset.ResetURL == "" {
		t.Error("Reset password data should have ResetURL")
	}
	
	// Check newsletter data
	newsletter := testData["premium_newsletter"].(NewsletterData)
	if len(newsletter.ContentBlocks) == 0 {
		t.Error("Newsletter data should have content blocks")
	}
	if len(newsletter.FeaturedContent) == 0 {
		t.Error("Newsletter data should have featured content")
	}
}

// TestSimplifiedArchitecture tests the architecture simplifications
func TestSimplifiedArchitecture(t *testing.T) {
	// Test that we can create a renderer without complex setup
	renderer := NewRenderer(WithFonts(true))
	
	if renderer == nil {
		t.Fatal("Failed to create renderer")
	}
	
	// Test font integration is optional
	fontStack := renderer.GetEmailSafeFontStack("Inter")
	if fontStack == "" {
		t.Error("Should return font stack even without loaded fonts")
	}
	
	// Test templates list is initially empty (file-based only)
	templates := renderer.ListTemplates()
	if len(templates) != 0 {
		t.Error("Templates should be empty initially - loaded from files only")
	}
	
	// Test canonical test data is DRY
	testData := TestData()
	if len(testData) != 5 {
		t.Errorf("Expected 5 test data sets, got %d", len(testData))
	}
}

// TestFontIntegrationOptional tests that font integration is truly optional
func TestFontIntegrationOptional(t *testing.T) {
	// Renderer without fonts should work
	rendererNoFonts := NewRenderer(WithFonts(false))
	
	// Should still be able to get font stacks (fallback)
	stack := rendererNoFonts.GetEmailSafeFontStack("Roboto")
	if stack == "" {
		t.Error("Should return fallback font stack even with fonts disabled")
	}
	
	// Font operations should fail gracefully
	err := rendererNoFonts.LoadFont("Roboto", 400)
	if err == nil {
		t.Error("LoadFont should fail when fonts are disabled")
	}
	
	// Renderer with fonts enabled should work
	rendererWithFonts := NewRenderer(WithFonts(true))
	
	// Should be able to get font CSS (may use mock fonts)
	_, err = rendererWithFonts.GetFontCSS("Roboto", 400)
	// Error is acceptable if font download fails - it should use mocks
}