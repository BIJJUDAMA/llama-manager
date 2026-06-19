package benchmark

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBenchmarkDatabase(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "llama-benchmark-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Save benchmark result
	res := &BenchmarkResult{
		ModelPath:        "models/test.gguf",
		ModelName:        "Test Model",
		RunDate:          time.Now(),
		StartupTimeMs:    1200,
		TokensPerSec:     45.2,
		PeakTokensPerSec: 48.0,
		RAMUsageMB:       2048.5,
		VRAMUsageMB:      512.0,
	}

	err = SaveResult(tempDir, res)
	if err != nil {
		t.Fatalf("failed to save result: %v", err)
	}

	// Verify history file exists
	historyFile := filepath.Join(tempDir, "history.json")
	if _, err := os.Stat(historyFile); os.IsNotExist(err) {
		t.Errorf("history.json was not created")
	}

	// Load history
	history, err := LoadHistory(tempDir)
	if err != nil {
		t.Fatalf("failed to load history: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("expected 1 result, got %d", len(history))
	}

	loaded := history[0]
	if loaded.ModelName != "Test Model" || loaded.TokensPerSec != 45.2 || loaded.StartupTimeMs != 1200 {
		t.Errorf("loaded data mismatch: %+v", loaded)
	}
}
