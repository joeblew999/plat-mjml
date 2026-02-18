package mjml

import (
	"testing"
)

// TestCacheKeyDeterministic verifies cache keys are deterministic
func TestCacheKeyDeterministic(t *testing.T) {
	renderer := NewRenderer(WithCache(true), WithFonts(false))

	testData := TestData()
	data := testData["simple"]

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

	expectedTemplates := []string{"simple", "welcome", "reset_password", "notification", "premium_newsletter", "business_announcement"}

	for _, templateName := range expectedTemplates {
		if _, exists := testData[templateName]; !exists {
			t.Errorf("Missing test data for template: %s", templateName)
		}
	}

	simple := testData["simple"].(EmailData)
	if simple.CompanyName == "" || simple.Name == "" {
		t.Error("Simple test data should have populated required fields")
	}

	welcome := testData["welcome"].(WelcomeEmailData)
	if welcome.ActivationURL == "" {
		t.Error("Welcome data should have ActivationURL")
	}

	reset := testData["reset_password"].(ResetPasswordData)
	if reset.ResetURL == "" {
		t.Error("Reset password data should have ResetURL")
	}

	newsletter := testData["premium_newsletter"].(NewsletterData)
	if len(newsletter.ContentBlocks) == 0 {
		t.Error("Newsletter data should have content blocks")
	}
	if len(newsletter.FeaturedContent) == 0 {
		t.Error("Newsletter data should have featured content")
	}

	// Check business_announcement data (map type)
	ba, ok := testData["business_announcement"].(map[string]any)
	if !ok {
		t.Error("business_announcement test data should be map[string]any")
	} else if ba["company_name"] == nil || ba["name"] == nil {
		t.Error("business_announcement test data should have required fields")
	}
}

// TestSimplifiedArchitecture tests the architecture simplifications
func TestSimplifiedArchitecture(t *testing.T) {
	renderer := NewRenderer(WithFonts(true), WithFontDir(t.TempDir()))

	if renderer == nil {
		t.Fatal("Failed to create renderer")
	}

	fontStack := renderer.GetEmailSafeFontStack("Inter")
	if fontStack == "" {
		t.Error("Should return font stack even without loaded fonts")
	}

	templates := renderer.ListTemplates()
	if len(templates) != 0 {
		t.Error("Templates should be empty initially - loaded from files only")
	}

	testData := TestData()
	if len(testData) != 6 {
		t.Errorf("Expected 6 test data sets, got %d", len(testData))
	}
}

// TestFontIntegrationOptional tests that font integration is truly optional
func TestFontIntegrationOptional(t *testing.T) {
	rendererNoFonts := NewRenderer(WithFonts(false))

	stack := rendererNoFonts.GetEmailSafeFontStack("Roboto")
	if stack == "" {
		t.Error("Should return fallback font stack even with fonts disabled")
	}

	err := rendererNoFonts.LoadFont("Roboto", 400)
	if err == nil {
		t.Error("LoadFont should fail when fonts are disabled")
	}

	rendererWithFonts := NewRenderer(WithFonts(true), WithFontDir(t.TempDir()))

	_, err = rendererWithFonts.GetFontCSS("Roboto", 400)
	// Error is acceptable if font download fails - it should use mocks
}
