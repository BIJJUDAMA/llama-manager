package model

import (
	"os"
	"path/filepath"
	"strings"
)

// DiscoverModels recursively scans the given root directory for GGUF files and parses their metadata, utilizing a metadata cache.
func DiscoverModels(root string) ([]*GGUFMetadata, error) {
	var models []*GGUFMetadata

	// Clean the root path
	root = filepath.Clean(root)

	// Check if directory exists
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil, nil
	}

	cachePath := filepath.Join("cache", "metadata_cache.json")
	cache, _ := LoadCache(cachePath)
	if cache == nil {
		cache = NewMetadataCache()
	}

	cacheUpdated := false

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".gguf") {
			info, statErr := d.Info()
			if statErr != nil {
				return nil
			}

			cacheKey := filepath.ToSlash(path)

			var meta *GGUFMetadata
			entry, exists := cache.Entries[cacheKey]
			if exists && entry.ModTime == info.ModTime().Unix() && entry.Size == info.Size() {
				meta = entry.Metadata
				meta.FilePath = path
			} else {
				var parseErr error
				meta, parseErr = ParseGGUF(path)
				if parseErr == nil {
					cache.Entries[cacheKey] = &GGUFCacheEntry{
						Metadata: meta,
						ModTime:  info.ModTime().Unix(),
						Size:     info.Size(),
					}
					cacheUpdated = true
				}
			}

			if meta != nil {
				models = append(models, meta)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	if cacheUpdated {
		_ = cache.Save(cachePath)
	}

	return models, nil
}
