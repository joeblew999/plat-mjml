package font

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFontManager(t *testing.T) {
	manager := NewManager()

	t.Run("ManagerCreation", func(t *testing.T) {
		assert.NotNil(t, manager)
		assert.NotNil(t, manager.registry)
		assert.Equal(t, GetLocalFontPath(), manager.cacheDir)
		t.Logf("📁 Cache directory: %s", manager.cacheDir)
		t.Logf("📋 Registry path: %s", manager.registry.path)
	})

	t.Run("FontCaching", func(t *testing.T) {
		family := "Roboto"
		weight := 400
		
		// Cache the font
		err := manager.Cache(family, weight)
		require.NoError(t, err)

		// Verify it's now available
		assert.True(t, manager.Available(family, weight))

		// Get the path
		path, err := manager.Get(family, weight)
		require.NoError(t, err)
		assert.FileExists(t, path)
		assert.Contains(t, path, family)
		assert.Contains(t, path, "400.ttf")
		t.Logf("✅ Font cached: %s", path)
	})

	t.Run("CacheHitBehavior", func(t *testing.T) {
		family := "Inter"
		weight := 400
		
		// First call - cache miss, downloads
		path1, err := manager.Get(family, weight)
		require.NoError(t, err)
		
		// Second call - cache hit, same path
		path2, err := manager.Get(family, weight)
		require.NoError(t, err)
		assert.Equal(t, path1, path2, "Cache hit should return same path")
		t.Logf("✅ Cache hit - reused path: %s", path1)
	})

	t.Run("DifferentFormats", func(t *testing.T) {
		family := "Open Sans"
		weight := 400
		
		// TTF format (default)
		ttfPath, err := manager.GetFormat(family, weight, "ttf")
		require.NoError(t, err)
		assert.Contains(t, ttfPath, ".ttf")
		assert.FileExists(t, ttfPath)
		t.Logf("✅ TTF format cached: %s", ttfPath)
	})

	t.Run("RegistryPersistence", func(t *testing.T) {
		// Add a font to registry
		family := "Lato"
		weight := 400
		err := manager.Cache(family, weight)
		require.NoError(t, err)

		// Create new manager (simulates restart)
		newManager := NewManager()
		
		// Should find the cached font
		assert.True(t, newManager.Available(family, weight))
		
		path, err := newManager.Get(family, weight)
		require.NoError(t, err)
		assert.FileExists(t, path)
		t.Logf("✅ Registry persistence verified: %s", path)
	})
}

func TestRegistryKeyGeneration(t *testing.T) {
	registry := NewRegistry()
	
	font := Font{
		Family: "Roboto",
		Weight: 400,
		Style:  "normal",
		Format: "ttf",
	}
	
	key := registry.key(font)
	assert.Equal(t, "roboto-400-normal-ttf", key)
	
	// Test different weights/styles produce different keys
	boldFont := Font{Family: "Roboto", Weight: 700, Style: "normal", Format: "ttf"}
	italicFont := Font{Family: "Roboto", Weight: 400, Style: "italic", Format: "ttf"}
	
	assert.NotEqual(t, registry.key(font), registry.key(boldFont))
	assert.NotEqual(t, registry.key(font), registry.key(italicFont))
}

func TestFontFormats(t *testing.T) {
	// Test font struct creation
	font := Font{
		Family: "Test",
		Weight: 400,
		Style:  "italic",
		Format: "ttf",
	}

	assert.Equal(t, "Test", font.Family)
	assert.Equal(t, 400, font.Weight)
	assert.Equal(t, "italic", font.Style)
	assert.Equal(t, "ttf", font.Format)
}

func TestHelperFunctions(t *testing.T) {
	t.Run("newFont", func(t *testing.T) {
		font := newFont("Roboto", 400, "ttf")
		assert.Equal(t, "Roboto", font.Family)
		assert.Equal(t, 400, font.Weight)
		assert.Equal(t, "normal", font.Style) // Default
		assert.Equal(t, "ttf", font.Format)
	})
	
	t.Run("newDefaultFont", func(t *testing.T) {
		font := newDefaultFont("Inter", 700)
		assert.Equal(t, "Inter", font.Family)
		assert.Equal(t, 700, font.Weight)
		assert.Equal(t, "normal", font.Style)
		assert.Equal(t, DefaultFontFormat, font.Format) // Should be TTF
	})
}

func TestTTFValidation(t *testing.T) {
	t.Run("ValidTTFSignatures", func(t *testing.T) {
		validSignatures := [][]byte{
			{0x00, 0x01, 0x00, 0x00}, // Standard TrueType
			{'O', 'T', 'T', 'O'},     // OpenType with PostScript
		}
		
		for i, sig := range validSignatures {
			isValid := isValidTTFSignature(sig)
			assert.True(t, isValid, "Valid signature %d (%x) should be recognized as TTF", i, sig)
		}
	})
	
	t.Run("InvalidTTFSignatures", func(t *testing.T) {
		invalidSignatures := [][]byte{
			{0xc9, 0xe1, 0x00, 0x00}, // WOFF2
			{0x77, 0x4f, 0x46, 0x46}, // WOFF
			{0x00, 0x00, 0x01, 0x00}, // Wrong TrueType order
			{0x45, 0x4f, 0x54, 0x00}, // EOT format
		}
		
		for i, sig := range invalidSignatures {
			isValid := isValidTTFSignature(sig)
			assert.False(t, isValid, "Invalid signature %d (%x) should not be recognized as TTF", i, sig)
		}
	})
}

// isValidTTFSignature validates TTF/OTF file signatures
func isValidTTFSignature(signature []byte) bool {
	if len(signature) < 4 {
		return false
	}
	// TTF: 0x00, 0x01, 0x00, 0x00
	// OTF: 'OTTO'
	return (signature[0] == 0x00 && signature[1] == 0x01 && signature[2] == 0x00 && signature[3] == 0x00) ||
		   (string(signature[0:4]) == "OTTO")
}

func TestDefaults(t *testing.T) {
	assert.Equal(t, "ttf", DefaultFontFormat, "Default format should be TTF for deck compatibility")
	assert.Equal(t, 400, DefaultFontWeight, "Default weight should be regular/400")
	assert.Equal(t, "normal", DefaultFontStyle, "Default style should be normal")
	assert.Greater(t, len(DefaultFonts), 0, "Should have default fonts defined")
}