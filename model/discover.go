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
		extLower := strings.ToLower(filepath.Ext(d.Name()))
		if !d.IsDir() && (extLower == ".gguf" || extLower == ".onnx") {
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
				if meta.ID == "" {
					meta.ID = filepath.Base(path)
				}
				if meta.Runtime == "" {
					if extLower == ".gguf" {
						meta.Runtime = "llama.cpp"
					} else if extLower == ".onnx" {
						meta.Runtime = "ONNX Runtime"
					}
				}
				if meta.Task == "" {
					if meta.EmbeddingLen > 0 {
						meta.Task = "EMBEDDING"
					} else {
						meta.Task = "TEXT_GENERATION"
					}
				}
			} else {
				if extLower == ".gguf" {
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
				} else if extLower == ".onnx" {
					meta = &GGUFMetadata{
						ID:           filepath.Base(path),
						Name:         filepath.Base(path),
						FilePath:     path,
						FileSize:     info.Size(),
						Runtime:      "ONNX Runtime",
						Task:         "TEXT_GENERATION",
						Architecture: "ONNX",
						Quantization: "Float32",
					}
					cache.Entries[cacheKey] = &GGUFCacheEntry{
						Metadata: meta,
						ModTime:  info.ModTime().Unix(),
						Size:     info.Size(),
					}
					cacheUpdated = true
				}
			}

			if meta != nil {
				// Apply automatic task heuristics based on folder name
				dirLower := strings.ToLower(filepath.ToSlash(filepath.Dir(path)))
				if strings.Contains(dirLower, "/embedding") {
					meta.Task = "EMBEDDING"
				} else if strings.Contains(dirLower, "/speech") {
					meta.Task = "SPEECH_TO_TEXT"
				} else if strings.Contains(dirLower, "/vision") {
					meta.Task = "VISION"
				} else if strings.Contains(dirLower, "/diffusion") {
					meta.Task = "IMAGE_GENERATION"
				} else if strings.Contains(dirLower, "/reranker") {
					meta.Task = "RERANKING"
				}
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
