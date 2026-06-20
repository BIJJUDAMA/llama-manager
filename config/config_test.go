package config

import (
	"os"
	"testing"
)

func TestConfigLoadAndSave(t *testing.T) {
	// Setup a temporary directory for the test to avoid modifying the real config.json
	tempDir, err := os.MkdirTemp("", "llama-manager-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change working directory to tempDir for the duration of the test
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change working dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(origWD)
	}()

	// Load configuration (should create default config and directories)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error loading config, got %v", err)
	}

	// Verify paths are set to defaults
	if cfg.Paths.Models != "models" {
		t.Errorf("expected paths.models to be 'models', got %q", cfg.Paths.Models)
	}

	// Verify directories are created
	expectedDirs := []string{"models", "llama.cpp", "profiles", "cache", "benchmarks", "downloads"}
	for _, dir := range expectedDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("expected directory %q to be created, but it was not", dir)
		}
	}

	// Modify config and save
	cfg.Theme = "light"
	cfg.Favorites = append(cfg.Favorites, "test-model")
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Reload config and verify changes
	reloadedCfg, err := Load()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}

	if reloadedCfg.Theme != "light" {
		t.Errorf("expected theme to be 'light', got %q", reloadedCfg.Theme)
	}

	if len(reloadedCfg.Favorites) != 1 || reloadedCfg.Favorites[0] != "test-model" {
		t.Errorf("expected favorites to contain 'test-model', got %v", reloadedCfg.Favorites)
	}
}

func TestConfigHelpers(t *testing.T) {
	cfg := DefaultConfig()

	// Test Favorites helper methods
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

	// Test Recent Launches helper methods (max 5, prepends)
	models := []string{"m1", "m2", "m3", "m4", "m5", "m6"}
	for _, m := range models {
		cfg.RecordLaunch(m)
	}
	if len(cfg.RecentLaunches) != 5 {
		t.Errorf("expected RecentLaunches count to be capped at 5, got %d", len(cfg.RecentLaunches))
	}
	// "m6" was launched last, so it should be at index 0. "m1" should be discarded.
	if cfg.RecentLaunches[0] != "m6" {
		t.Errorf("expected most recent launch to be 'm6', got %q", cfg.RecentLaunches[0])
	}
	cfg.RecordLaunch("m3") // Move m3 to top
	if cfg.RecentLaunches[0] != "m3" {
		t.Errorf("expected 'm3' to move to top after recording launch again, got %q", cfg.RecentLaunches[0])
	}
}

