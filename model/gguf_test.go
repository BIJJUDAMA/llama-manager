package model

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"testing"
)

func writeGGUFString(w io.Writer, s string) {
	_ = binary.Write(w, binary.LittleEndian, uint64(len(s)))
	_, _ = w.Write([]byte(s))
}

func TestParseGGUF(t *testing.T) {
	// Create a temporary file to write mock GGUF content
	tempFile, err := os.CreateTemp("", "mock-gguf-*.gguf")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Build GGUF mock data
	var buf bytes.Buffer

	// 1. Magic
	_, _ = buf.Write([]byte("GGUF"))

	// 2. Version (uint32)
	_ = binary.Write(&buf, binary.LittleEndian, uint32(3))

	// 3. Tensor count (uint64)
	_ = binary.Write(&buf, binary.LittleEndian, uint64(0))

	// 4. Metadata KV count (uint64)
	_ = binary.Write(&buf, binary.LittleEndian, uint64(8))

	// KV 1: general.name (string)
	writeGGUFString(&buf, "general.name")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(TypeString))
	writeGGUFString(&buf, "Mock Model 123")

	// KV 2: general.architecture (string)
	writeGGUFString(&buf, "general.architecture")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(TypeString))
	writeGGUFString(&buf, "llama")

	// KV 3: llama.context_length (uint32)
	writeGGUFString(&buf, "llama.context_length")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(TypeUInt32))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(4096))

	// KV 4: general.file_type (uint32)
	writeGGUFString(&buf, "general.file_type")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(TypeUInt32))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(2)) // Q4_0

	// KV 5: llama.block_count (uint32)
	writeGGUFString(&buf, "llama.block_count")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(TypeUInt32))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(32))

	// KV 6: llama.attention.head_count (uint32)
	writeGGUFString(&buf, "llama.attention.head_count")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(TypeUInt32))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(32))

	// KV 7: llama.attention.head_count_kv (uint32)
	writeGGUFString(&buf, "llama.attention.head_count_kv")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(TypeUInt32))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(8))

	// KV 8: llama.embedding_length (uint32)
	writeGGUFString(&buf, "llama.embedding_length")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(TypeUInt32))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(4096))

	// Write to temp file
	_, err = tempFile.Write(buf.Bytes())
	if err != nil {
		t.Fatalf("failed to write mock GGUF data: %v", err)
	}
	_ = tempFile.Close()

	// Parse GGUF file
	meta, err := ParseGGUF(tempFile.Name())
	if err != nil {
		t.Fatalf("failed to parse GGUF: %v", err)
	}

	if meta.Name != "Mock Model 123" {
		t.Errorf("expected name to be 'Mock Model 123', got %q", meta.Name)
	}
	if meta.Architecture != "llama" {
		t.Errorf("expected architecture to be 'llama', got %q", meta.Architecture)
	}
	if meta.ContextLength != 4096 {
		t.Errorf("expected context length to be 4096, got %d", meta.ContextLength)
	}
	if meta.Quantization != "Q4_0" {
		t.Errorf("expected quantization to be 'Q4_0', got %q", meta.Quantization)
	}
	if meta.Layers != 32 {
		t.Errorf("expected layers to be 32, got %d", meta.Layers)
	}
	if meta.Heads != 32 {
		t.Errorf("expected heads to be 32, got %d", meta.Heads)
	}
	if meta.HeadsKV != 8 {
		t.Errorf("expected headsKV to be 8, got %d", meta.HeadsKV)
	}
	if meta.EmbeddingLen != 4096 {
		t.Errorf("expected embedding length to be 4096, got %d", meta.EmbeddingLen)
	}
}
