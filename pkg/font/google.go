package font

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/joeblew999/plat-mjml/pkg/log"
)

// Uses GoogleFontsAPI from consts.go

// downloadGoogleFont downloads a font from Google Fonts and returns the CDN URL
func downloadGoogleFont(font Font, path string) (cdnURL string, err error) {
	// For TTF format, try direct API first
	if font.Format == "ttf" {
		if url, dlErr := tryDirectTTFDownload(font, path); dlErr == nil {
			return url, nil
		}
		log.Warn("Direct TTF API failed, trying CSS method", "family", font.Family)
	}

	// Fallback to CSS method
	return tryCSSDownload(font, path)
}

// tryDirectTTFDownload attempts to download TTF using direct API
func tryDirectTTFDownload(font Font, path string) (string, error) {
	fontURL, err := getGoogleFontDirectURL(font)
	if err != nil {
		return "", err
	}
	log.Info("Using direct TTF URL", "family", font.Family, "url", fontURL)
	return fontURL, downloadFontFile(fontURL, path)
}

// tryCSSDownload attempts to download using CSS parsing method
func tryCSSDownload(font Font, path string) (string, error) {
	cssURL := buildGoogleFontsURL(font)
	fontURL, err := getFontURL(cssURL, font.Format)
	if err != nil {
		log.Warn("Failed to get font from CSS, using mock", "family", font.Family, "error", err)
		return "", createMockFontFile(path, font)
	}
	return fontURL, downloadFontFile(fontURL, path)
}

// GoogleFontsResponse represents the Google Fonts Web API response
type GoogleFontsResponse struct {
	Items []GoogleFontItem `json:"items"`
}

type GoogleFontItem struct {
	Family string                 `json:"family"`
	Files  map[string]string      `json:"files"`
}

// getGoogleFontDirectURL gets the direct TTF URL from Google Fonts Web API
func getGoogleFontDirectURL(font Font) (string, error) {
	// Google Fonts Web API (public, no key needed for basic usage)
	apiURL := "https://www.googleapis.com/webfonts/v1/webfonts"
	
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch Google Fonts list: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Google Fonts API returned status: %s", resp.Status)
	}
	
	var fontsResponse GoogleFontsResponse
	if err := json.NewDecoder(resp.Body).Decode(&fontsResponse); err != nil {
		return "", fmt.Errorf("failed to parse Google Fonts response: %w", err)
	}
	
	// Find the font family
	for _, item := range fontsResponse.Items {
		if strings.EqualFold(item.Family, font.Family) {
			// Look for TTF file - try different weight variants
			weightKey := fmt.Sprintf("%d", font.Weight)
			if font.Weight == 400 {
				weightKey = "regular"
			}
			
			if ttfURL, exists := item.Files[weightKey]; exists {
				return ttfURL, nil
			}
			
			// Fallback to regular if specific weight not found
			if ttfURL, exists := item.Files["regular"]; exists {
				return ttfURL, nil
			}
			
			return "", fmt.Errorf("no TTF file found for %s weight %d", font.Family, font.Weight)
		}
	}
	
	return "", fmt.Errorf("font family '%s' not found in Google Fonts", font.Family)
}

// buildGoogleFontsURL creates the CSS URL for Google Fonts
func buildGoogleFontsURL(font Font) string {
	// Example: https://fonts.googleapis.com/css2?family=Roboto:wght@400
	familyName := strings.ReplaceAll(font.Family, " ", "+")
	
	url := fmt.Sprintf("%s?family=%s:wght@%d", GoogleFontsAPI, familyName, font.Weight)
	
	if font.Style == "italic" {
		url = fmt.Sprintf("%s?family=%s:ital,wght@1,%d", GoogleFontsAPI, familyName, font.Weight)
	}
	
	return url
}

// getUserAgentForFormat returns appropriate user agent for the desired font format
func getUserAgentForFormat(format string) string {
	if format == "ttf" {
		// Use extremely old user agent to force TTF format from Google Fonts
		return "Mozilla/3.0 (X11; U; SunOS 5.4 sun4c)"
	}
	// Default to modern user agent for WOFF2
	return "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
}

// getFontURL parses CSS to extract the actual font file URL
func getFontURL(cssURL, format string) (string, error) {
	req, err := http.NewRequest("GET", cssURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", getUserAgentForFormat(format))
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	css, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	
	return extractFontURLFromCSS(string(css), format)
}

// extractFontURLFromCSS extracts font URL from CSS using regex
func extractFontURLFromCSS(css, format string) (string, error) {
	// Extract font URL from CSS - Google Fonts uses different URL patterns
	re := regexp.MustCompile(`url\((https://fonts\.gstatic\.com/[^)]+)\)`)
	matches := re.FindStringSubmatch(css)
	
	if len(matches) < 2 {
		return "", fmt.Errorf("no %s font URL found in CSS", format)
	}
	
	return matches[1], nil
}

// downloadFontFile downloads a font file from a URL
func downloadFontFile(url, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download font: %s", resp.Status)
	}
	
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	
	_, err = io.Copy(file, resp.Body)
	return err
}

// createMockFontFile creates a mock font file for testing/fallback
func createMockFontFile(path string, font Font) error {
	mockData := generateMockFontData(font)
	return os.WriteFile(path, mockData, 0644)
}

// generateMockFontData creates mock font data
func generateMockFontData(font Font) []byte {
	content := fmt.Sprintf("mock font data for %s %d", font.Family, font.Weight)
	return []byte(content)
}

// ListGoogleFonts returns available Google Fonts
func ListGoogleFonts() []string {
	return DefaultFonts
}

// GetFontCSS generates @font-face CSS for embedding a font.
// When a CDN URL is available (from Google Fonts), it uses that for email compatibility.
// Falls back to the local path if no CDN URL is set.
func GetFontCSS(info FontInfo) string {
	src := info.Path
	if info.CDNURL != "" {
		src = info.CDNURL
	}
	return fmt.Sprintf(`@font-face {
  font-family: '%s';
  font-style: %s;
  font-weight: %d;
  font-display: swap;
  src: url('%s') format('%s');
}`, info.Family, info.Style, info.Weight, src, info.Format)
}

// GetEmailSafeFonts returns fonts that are widely supported in email clients
func GetEmailSafeFonts() []string {
	return []string{
		"Arial",
		"Helvetica",
		"Georgia", 
		"Times",
		"Courier",
		"Verdana",
		"Tahoma",
		"Impact",
		"Comic Sans MS",
		"Trebuchet MS",
		"Arial Black",
		"Palatino",
		"Lucida Console",
	}
}