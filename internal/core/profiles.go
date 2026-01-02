package core

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"
)

//go:embed assets/profiles/*.profile
var profileFS embed.FS

// GetEmbeddedProfileNames returns a sorted list of available profile names.
func GetEmbeddedProfileNames() []string {
	entries, err := profileFS.ReadDir("assets/profiles")
	if err != nil {
		return []string{"jquery.profile"} // Fallback
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".profile") {
			names = append(names, entry.Name())
		}
	}
	return names
}

// GetProfileContent retrieves the byte content of an embedded profile by name.
func GetProfileContent(name string) ([]byte, error) {
	path := filepath.Join("assets/profiles", name)
	content, err := profileFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("profile not found: %s", name)
	}
	return content, nil
}
