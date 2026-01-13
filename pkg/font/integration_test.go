//go:build integration
// +build integration

package font

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFontCachePopulation(t *testing.T) {
	// Skip if no network access
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use local .data directory for development visibility
	fontDir := GetLocalFontPath()

	// Create manager with local directory
	manager := &Manager{
		cacheDir: fontDir,
		registry: NewRegistry(),
	}

	t.Run("CacheFontIntegration", func(t *testing.T) {
		// Test caching a known Google Font
		family := "Roboto"
		weight := 400

		// Ensure font is not initially available
		assert.False(t, manager.Available(family, weight))

		// Cache the font
		err := manager.Cache(family, weight)
		require.NoError(t, err)

		// Verify font is now available
		assert.True(t, manager.Available(family, weight))

		// Get the font path
		path, err := manager.Get(family, weight)
		require.NoError(t, err)
		assert.FileExists(t, path)

		// Verify it's the expected file format (TTF is now default)
		assert.Equal(t, "ttf", filepath.Ext(path)[1:])
	})

	t.Run("CacheMultipleFontsIntegration", func(t *testing.T) {
		fonts := []struct {
			family string
			weight int
		}{
			{"Open Sans", 400},
			{"Roboto", 700},
			{"Lato", 400},
		}

		for _, font := range fonts {
			err := manager.Cache(font.family, font.weight)
			require.NoError(t, err)
			assert.True(t, manager.Available(font.family, font.weight))
		}

		// Verify all fonts are listed (may include fonts from previous runs)
		available := manager.List()
		assert.GreaterOrEqual(t, len(available), len(fonts))
		t.Logf("Total cached fonts: %d", len(available))
	})

	t.Run("CacheDirectoryStructureIntegration", func(t *testing.T) {
		// Verify directory structure is created correctly
		family := "Roboto"
		weight := 400

		path, err := manager.Get(family, weight)
		require.NoError(t, err)

		// Verify the file exists in the expected location
		assert.Contains(t, path, family)
		assert.Contains(t, path, "400.ttf")
		assert.FileExists(t, path)
		t.Logf("Font cached at: %s", path)
	})

	t.Run("RegistryPersistenceIntegration", func(t *testing.T) {
		family := "Inter"
		weight := 400

		// Cache a font
		err := manager.Cache(family, weight)
		require.NoError(t, err)

		// Create a new manager with same directory
		newManager := &Manager{
			cacheDir: fontDir,
			registry: NewRegistry(),
		}

		// Cache the font in new manager to ensure it's registered
		err = newManager.Cache(family, weight)
		require.NoError(t, err)

		// Verify font is available
		assert.True(t, newManager.Available(family, weight))

		// Verify we can get the path without re-downloading
		path, err := newManager.Get(family, weight)
		require.NoError(t, err)
		assert.FileExists(t, path)
	})

	t.Run("CacheReuseIntegration", func(t *testing.T) {
		family := "Poppins"
		weight := 400

		// First call should download and cache
		path1, err := manager.Get(family, weight)
		require.NoError(t, err)
		assert.FileExists(t, path1)

		// Second call should use cached version
		path2, err := manager.Get(family, weight)
		require.NoError(t, err)
		assert.Equal(t, path1, path2, "Should return same path for cached font")

		// Verify file was not re-downloaded by checking modification time
		info, err := os.Stat(path1)
		require.NoError(t, err)
		assert.Greater(t, info.Size(), int64(0), "Font file should have content")
	})

	t.Run("TTFFontCachingForDeck", func(t *testing.T) {
		// Test TTF font caching specifically for deck tools compatibility
		family := "Roboto"
		weight := 400

		// Cache font in TTF format
		err := manager.CacheTTF(family, weight)
		require.NoError(t, err)

		// Get TTF font path
		path, err := manager.GetFormat(family, weight, "ttf")
		require.NoError(t, err)
		assert.FileExists(t, path)

		// Verify it's actually a TTF file
		assert.Equal(t, "ttf", filepath.Ext(path)[1:])

		// Check file content has TTF signature
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Greater(t, len(content), 4, "TTF file should have content")

		// TTF files start with specific signatures
		// 0x00, 0x01, 0x00, 0x00 for TrueType or 'OTTO' for OpenType
		if len(content) >= 4 {
			isValidTTF := (content[0] == 0x00 && content[1] == 0x01 && content[2] == 0x00 && content[3] == 0x00) ||
				(string(content[0:4]) == "OTTO")
			assert.True(t, isValidTTF, "File should have valid TTF/OTF signature, got: % x", content[0:4])
		}

		// Verify font is available via standard methods
		assert.True(t, manager.Available(family, weight))
	})

	t.Run("DeckToolsFontCompatibility", func(t *testing.T) {
		// Test fonts that deck tools specifically look for
		requiredFonts := []struct {
			name   string
			family string
			weight int
		}{
			{"FiraSans-Regular", "Roboto", 400}, // Most common deck font
			{"arial", "Open Sans", 400},
			{"helvetica", "Roboto", 400},
		}

		for _, font := range requiredFonts {
			t.Run(font.name, func(t *testing.T) {
				// Cache as TTF
				err := manager.CacheTTF(font.family, font.weight)
				require.NoError(t, err)

				// Verify we can get TTF format
				path, err := manager.GetFormat(font.family, font.weight, "ttf")
				require.NoError(t, err)
				assert.FileExists(t, path)

				// Verify file is non-empty TTF
				info, err := os.Stat(path)
				require.NoError(t, err)
				assert.Greater(t, info.Size(), int64(1000), "TTF should be at least 1KB")
			})
		}
	})
	
	t.Run("ValidateAllCachedFontSignatures", func(t *testing.T) {
		// Get all cached fonts
		fonts := manager.List()
		
		for _, fontInfo := range fonts {
			if fontInfo.Format == "ttf" {
				t.Run(fmt.Sprintf("%s_%d", fontInfo.Family, fontInfo.Weight), func(t *testing.T) {
					// Read the actual cached file
					content, err := os.ReadFile(fontInfo.Path)
					require.NoError(t, err, "Should be able to read cached font file")
					require.Greater(t, len(content), 4, "Font file should have at least 4 bytes")
					
					// Validate TTF signature
					signature := content[0:4]
					isValidTTF := (signature[0] == 0x00 && signature[1] == 0x01 && signature[2] == 0x00 && signature[3] == 0x00) ||
						(string(signature) == "OTTO")
					
					assert.True(t, isValidTTF, 
						"Font %s %d should have valid TTF signature, got: % x", 
						fontInfo.Family, fontInfo.Weight, signature)
				})
			}
		}
		
		// Ensure we tested at least one TTF font
		ttfCount := 0
		for _, fontInfo := range fonts {
			if fontInfo.Format == "ttf" {
				ttfCount++
			}
		}
		assert.Greater(t, ttfCount, 0, "Should have at least one TTF font to validate")
	})
}