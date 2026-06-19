package model

import (
	"os"
	"path/filepath"
	"strings"
)

// DiscoverModels recursively scans the given root directory for GGUF files and parses their metadata.
func DiscoverModels(root string) ([]*GGUFMetadata, error) {
	var models []*GGUFMetadata

	// Clean the root path
	root = filepath.Clean(root)

	// Check if directory exists
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil, nil
	}

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Skip files/directories that generate errors (e.g. permission issues)
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".gguf") {
			meta, parseErr := ParseGGUF(path)
			if parseErr == nil {
				models = append(models, meta)
			}
			// If parsing fails, we skip it (or could log it, but standard TUI ignores broken files)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return models, nil
}
