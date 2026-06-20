package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// loadWithDir loads config rooted at dir, bypassing the platform AppDataDir.
func loadWithDir(dir string) (*Config, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	configPath := filepath.Join(dir, configFileName)

	var cfg *Config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg = defaultConfig(dir)
		if err := cfg.Save(); err != nil {
			return nil, err
		}
	} else {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, err
		}
		cfg = defaultConfig(dir)
		_ = json.Unmarshal(data, cfg)
	}

	cfg.configPath = configPath

	if cfg.ModelProfiles == nil {
		cfg.ModelProfiles = make(map[string]string)
	}
	if cfg.Favorites == nil {
		cfg.Favorites = []string{}
	}
	if cfg.RecentLaunches == nil {
		cfg.RecentLaunches = []string{}
	}

	if err := cfg.CreateDirectories(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func TestConfigLoadAndSave(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "llmgr-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg, err := loadWithDir(tempDir)
	if err != nil {
		t.Fatalf("expected no error loading config, got %v", err)
	}

	// Paths should be absolute and rooted under tempDir
	expectedModels := filepath.Join(tempDir, "models")
	if cfg.Paths.Models != expectedModels {
		t.Errorf("expected paths.models to be %q, got %q", expectedModels, cfg.Paths.Models)
	}

	// All data directories should have been created under tempDir
	for _, sub := range []string{"models", "llama.cpp", "profiles", "cache", "benchmarks", "downloads"} {
		full := filepath.Join(tempDir, sub)
		if _, err := os.Stat(full); os.IsNotExist(err) {
			t.Errorf("expected directory %q to be created, but it was not", full)
		}
	}

	// Modify and save
	cfg.Theme = "light"
	cfg.Favorites = append(cfg.Favorites, "test-model")
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Reload and verify
	reloaded, err := loadWithDir(tempDir)
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	if reloaded.Theme != "light" {
		t.Errorf("expected theme to be 'light', got %q", reloaded.Theme)
	}
	if len(reloaded.Favorites) != 1 || reloaded.Favorites[0] != "test-model" {
		t.Errorf("expected favorites to contain 'test-model', got %v", reloaded.Favorites)
	}
}

func TestConfigHelpers(t *testing.T) {
	cfg := defaultConfig("")

	// Favorites
	modelPath := "models/Qwen/qwen2.5.gguf"
	if cfg.IsFavorite(modelPath) {
		t.Errorf("expected model to not be favorite initially")
	}
	cfg.ToggleFavorite(modelPath)
	if !cfg.IsFavorite(modelPath) {
		t.Errorf("expected model to be favorite after toggling")
	}
	cfg.ToggleFavorite(modelPath)
	if cfg.IsFavorite(modelPath) {
		t.Errorf("expected model to not be favorite after toggling again")
	}

	// RecentLaunches capped at 5
	for _, m := range []string{"m1", "m2", "m3", "m4", "m5", "m6"} {
		cfg.RecordLaunch(m)
	}
	if len(cfg.RecentLaunches) != 5 {
		t.Errorf("expected RecentLaunches capped at 5, got %d", len(cfg.RecentLaunches))
	}
	if cfg.RecentLaunches[0] != "m6" {
		t.Errorf("expected most recent launch to be 'm6', got %q", cfg.RecentLaunches[0])
	}
	cfg.RecordLaunch("m3")
	if cfg.RecentLaunches[0] != "m3" {
		t.Errorf("expected 'm3' to move to top, got %q", cfg.RecentLaunches[0])
	}
}
