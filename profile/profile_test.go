package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestProfileDefaultsAndLoading(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "llama-profiles-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// LoadAll on empty folder should populate default JSON files
	profiles, err := LoadAll(tempDir)
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(profiles) != 5 {
		t.Errorf("expected 5 default profiles, got %d", len(profiles))
	}

	// Verify that files are written
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("failed to read temp dir: %v", err)
	}
	if len(files) != 5 {
		t.Errorf("expected 5 JSON files in directory, got %d", len(files))
	}

	// Verify custom profile loading
	customProfile := &Profile{
		Name:      "Coding Special",
		Context:   8192,
		Threads:   4,
		GPULayers: 50,
		BatchSize: 256,
		Host:      "127.0.0.1",
		Port:      9090,
	}

	customData, err := json.Marshal(customProfile)
	if err != nil {
		t.Fatalf("failed to marshal custom profile: %v", err)
	}

	err = os.WriteFile(filepath.Join(tempDir, "coding_special.json"), customData, 0644)
	if err != nil {
		t.Fatalf("failed to write custom profile file: %v", err)
	}

	// Re-load all profiles (should load 5 defaults + 1 custom)
	reloaded, err := LoadAll(tempDir)
	if err != nil {
		t.Fatalf("second LoadAll failed: %v", err)
	}

	if len(reloaded) != 6 {
		t.Errorf("expected 6 profiles (5 defaults + 1 custom), got %d", len(reloaded))
	}

	foundCustom := false
	for _, p := range reloaded {
		if p.Name == "Coding Special" {
			foundCustom = true
			if p.Context != 8192 || p.Port != 9090 || p.GPULayers != 50 {
				t.Errorf("incorrect custom profile content parsed: %+v", p)
			}
		}
	}
	if !foundCustom {
		t.Errorf("custom profile 'Coding Special' was not loaded")
	}
}
