package font

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

)

// Registry manages font metadata and lookups
type Registry struct {
	path string
	data map[string]FontInfo
}

// NewRegistry creates a new font registry using the default font path.
func NewRegistry() *Registry {
	return NewRegistryAt(filepath.Join(GetLocalFontPath(), RegistryFilename))
}

// NewRegistryAt creates a new font registry at the specified path.
func NewRegistryAt(path string) *Registry {
	r := &Registry{
		path: path,
		data: make(map[string]FontInfo),
	}
	r.load()
	return r
}

// GetPath returns the path for a font if it exists
func (r *Registry) GetPath(font Font) (string, bool) {
	info, exists := r.GetInfo(font)
	if !exists {
		return "", false
	}
	return info.Path, true
}

// GetInfo returns the full FontInfo for a font if it exists
func (r *Registry) GetInfo(font Font) (FontInfo, bool) {
	key := r.key(font)
	info, exists := r.data[key]
	if !exists {
		return FontInfo{}, false
	}

	// Verify file still exists
	if !r.fileExists(info.Path) {
		r.removeStaleEntry(key)
		return FontInfo{}, false
	}

	return info, true
}

// fileExists checks if a file exists
func (r *Registry) fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// removeStaleEntry removes a registry entry for a non-existent file
func (r *Registry) removeStaleEntry(key string) {
	delete(r.data, key)
	r.save() // Note: ignoring error here as it's cleanup
}

// Add registers a font in the registry
func (r *Registry) Add(info FontInfo) error {
	key := r.key(info.Font)
	r.data[key] = info
	return r.save()
}

// List returns all registered fonts
func (r *Registry) List() []FontInfo {
	fonts := make([]FontInfo, 0, len(r.data))
	for _, info := range r.data {
		fonts = append(fonts, info)
	}
	return fonts
}

// Remove removes a font from the registry
func (r *Registry) Remove(font Font) error {
	key := r.key(font)
	delete(r.data, key)
	return r.save()
}

// key generates a unique key for a font
func (r *Registry) key(font Font) string {
	return fmt.Sprintf("%s-%d-%s-%s", 
		strings.ToLower(font.Family), 
		font.Weight, 
		font.Style, 
		font.Format)
}

// load reads the registry from disk
func (r *Registry) load() {
	if !r.fileExists(r.path) {
		return // Registry doesn't exist yet
	}
	
	registryData := r.readRegistryFile()
	if registryData != nil {
		r.data = registryData
	}
}

// readRegistryFile reads and parses the registry file
func (r *Registry) readRegistryFile() map[string]FontInfo {
	data, err := os.ReadFile(r.path)
	if err != nil {
		return nil // Failed to read, start fresh
	}
	
	var registryData map[string]FontInfo
	if err := json.Unmarshal(data, &registryData); err != nil {
		return nil // Failed to parse, start fresh
	}
	
	return registryData
}

// save writes the registry to disk
func (r *Registry) save() error {
	if err := r.ensureRegistryDir(); err != nil {
		return err
	}
	
	data, err := r.marshalRegistry()
	if err != nil {
		return err
	}
	
	return os.WriteFile(r.path, data, 0644)
}

// ensureRegistryDir creates the registry directory if it doesn't exist
func (r *Registry) ensureRegistryDir() error {
	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create font directory: %w", err)
	}
	return nil
}

// marshalRegistry converts the registry data to JSON
func (r *Registry) marshalRegistry() ([]byte, error) {
	data, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal registry: %w", err)
	}
	return data, nil
}