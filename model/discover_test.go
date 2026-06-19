package model

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func createMockGGUFAt(path string, name string, arch string, ctxLen uint32) error {
	var buf bytes.Buffer
	// Magic
	_, _ = buf.Write([]byte("GGUF"))
	// Version
	_ = binary.Write(&buf, binary.LittleEndian, uint32(3))
	// Tensor count
	_ = binary.Write(&buf, binary.LittleEndian, uint64(0))
	// Metadata KV count
	_ = binary.Write(&buf, binary.LittleEndian, uint64(3))

	// KV 1: general.name
	writeGGUFString(&buf, "general.name")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(TypeString))
	writeGGUFString(&buf, name)

	// KV 2: general.architecture
	writeGGUFString(&buf, "general.architecture")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(TypeString))
	writeGGUFString(&buf, arch)

	// KV 3: <arch>.context_length
	writeGGUFString(&buf, arch+".context_length")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(TypeUInt32))
	_ = binary.Write(&buf, binary.LittleEndian, ctxLen)

	return os.WriteFile(path, buf.Bytes(), 0644)
}

func TestDiscoverModels(t *testing.T) {
	// Create a temporary directory structure
	tempDir, err := os.MkdirTemp("", "llama-models-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create nested folders
	dir1 := filepath.Join(tempDir, "Gemma")
	dir2 := filepath.Join(tempDir, "Qwen", "Coder")
	if err := os.MkdirAll(dir1, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}
	if err := os.MkdirAll(dir2, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	// Create mock files
	model1Path := filepath.Join(dir1, "gemma-2b.gguf")
	model2Path := filepath.Join(dir2, "qwen-coder-7b.gguf")
	nonModelPath := filepath.Join(tempDir, "readme.txt")

	if err := createMockGGUFAt(model1Path, "Gemma 2B", "gemma", 2048); err != nil {
		t.Fatalf("failed to create mock GGUF: %v", err)
	}
	if err := createMockGGUFAt(model2Path, "Qwen Coder 7B", "qwen2", 32768); err != nil {
		t.Fatalf("failed to create mock GGUF: %v", err)
	}
	if err := os.WriteFile(nonModelPath, []byte("plain text"), 0644); err != nil {
		t.Fatalf("failed to write dummy file: %v", err)
	}

	// Discover models
	models, err := DiscoverModels(tempDir)
	if err != nil {
		t.Fatalf("DiscoverModels failed: %v", err)
	}

	// Should discover exactly 2 models
	if len(models) != 2 {
		t.Errorf("expected 2 discovered models, got %d", len(models))
	}

	// Check model contents
	foundGemma := false
	foundQwen := false
	for _, m := range models {
		if m.Name == "Gemma 2B" {
			foundGemma = true
			if m.Architecture != "gemma" {
				t.Errorf("expected gemma arch, got %q", m.Architecture)
			}
			if m.ContextLength != 2048 {
				t.Errorf("expected 2048 context length, got %d", m.ContextLength)
			}
		} else if m.Name == "Qwen Coder 7B" {
			foundQwen = true
			if m.Architecture != "qwen2" {
				t.Errorf("expected qwen2 arch, got %q", m.Architecture)
			}
			if m.ContextLength != 32768 {
				t.Errorf("expected 32768 context length, got %d", m.ContextLength)
			}
		}
	}

	if !foundGemma {
		t.Errorf("Gemma 2B was not discovered")
	}
	if !foundQwen {
		t.Errorf("Qwen Coder 7B was not discovered")
	}
}
