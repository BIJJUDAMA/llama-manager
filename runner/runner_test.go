package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunnerBinaryNotFound(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "llama-runner-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	runner := NewLlamaCppRuntime(tempDir)

	// Try to start a server with a non-existent llama.cpp dir
	err = runner.Start("some-model.gguf", StartOptions{
		LlamaCppDir: filepath.Join(tempDir, "missing-dir"),
		ContextSize: 2048,
		Threads:     4,
		GPULayers:   999,
		BatchSize:   512,
		Host:        "127.0.0.1",
		Port:        50505,
	})
	if err == nil {
		t.Errorf("expected error starting server with missing directory, got nil")
	}
}

func TestMultiInstanceTracking(t *testing.T) {
	runner := NewLlamaCppRuntime("")

	// Initially, it should have no active instances
	if len(runner.GetAllInstances()) != 0 {
		t.Errorf("expected 0 running instances initially")
	}

	status, model, port := runner.GetStatus()
	if status != StatusStopped || model != "" || port != 50505 {
		t.Errorf("incorrect stopped status values: %d, %q, %d", status, model, port)
	}
}
