package hardware

import (
	"testing"
	"llama-manager/model"
)

func TestEstimateMemoryUnified(t *testing.T) {
	// macOS Apple Silicon profile: Total RAM 16GB, Unified VRAM limit 10.67GB (2/3)
	specs := &HardwareSpecs{
		OS:        "macOS",
		IsUnified: true,
		RAM: RAMSpecs{
			Total:     16 * 1024 * 1024 * 1024,
			Available: 10 * 1024 * 1024 * 1024,
		},
		GPU: GPUSpecs{
			Name: "Apple Silicon Integrated GPU",
			VRAM: 16 * 1024 * 1024 * 1024 * 2 / 3,
			Type: "Metal",
		},
	}

	// Case 1: Model fits fully (e.g. 4GB file size)
	meta1 := &model.GGUFMetadata{
		FileSize:     4 * 1024 * 1024 * 1024,
		Layers:       32,
		Heads:        32,
		HeadsKV:      8,
		EmbeddingLen: 4096,
	}

	est1 := EstimateMemory(meta1, specs, 2048)
	if est1.Suitability != SuitabilityFits {
		t.Errorf("expected Fits suitability, got %d. Reason: %s", est1.Suitability, est1.Reason)
	}
	if est1.GPUOffloadPct != 100 {
		t.Errorf("expected 100%% offload, got %d%%", est1.GPUOffloadPct)
	}

	// Case 2: Partial offload (e.g. 11GB file size)
	meta2 := &model.GGUFMetadata{
		FileSize:     11 * 1024 * 1024 * 1024,
		Layers:       32,
		Heads:        32,
		HeadsKV:      8,
		EmbeddingLen: 4096,
	}
	est2 := EstimateMemory(meta2, specs, 2048)
	if est2.Suitability != SuitabilityPartial {
		t.Errorf("expected Partial suitability, got %d. Reason: %s", est2.Suitability, est2.Reason)
	}

	// Case 3: Exceeds total RAM (e.g. 18GB file size)
	meta3 := &model.GGUFMetadata{
		FileSize:     18 * 1024 * 1024 * 1024,
		Layers:       32,
		Heads:        32,
		HeadsKV:      8,
		EmbeddingLen: 4096,
	}
	est3 := EstimateMemory(meta3, specs, 2048)
	if est3.Suitability != SuitabilityExceeds {
		t.Errorf("expected Exceeds suitability, got %d. Reason: %s", est3.Suitability, est3.Reason)
	}
}

func TestEstimateMemoryDedicated(t *testing.T) {
	// Windows/Linux PC profile: System RAM 16GB, GPU VRAM 8GB
	specs := &HardwareSpecs{
		OS:        "Windows",
		IsUnified: false,
		RAM: RAMSpecs{
			Total:     16 * 1024 * 1024 * 1024,
			Available: 12 * 1024 * 1024 * 1024,
		},
		GPU: GPUSpecs{
			Name: "NVIDIA GeForce RTX 4070",
			VRAM: 8 * 1024 * 1024 * 1024,
			Type: "CUDA",
		},
	}

	// Case 1: Model fits fully (e.g. 5GB file size)
	meta1 := &model.GGUFMetadata{
		FileSize:     5 * 1024 * 1024 * 1024,
		Layers:       32,
		Heads:        32,
		HeadsKV:      8,
		EmbeddingLen: 4096,
	}
	est1 := EstimateMemory(meta1, specs, 2048)
	if est1.Suitability != SuitabilityFits {
		t.Errorf("expected Fits suitability, got %d. Reason: %s", est1.Suitability, est1.Reason)
	}

	// Case 2: Partial offload (e.g. 10GB file size)
	meta2 := &model.GGUFMetadata{
		FileSize:     10 * 1024 * 1024 * 1024,
		Layers:       32,
		Heads:        32,
		HeadsKV:      8,
		EmbeddingLen: 4096,
	}
	est2 := EstimateMemory(meta2, specs, 2048)
	if est2.Suitability != SuitabilityPartial {
		t.Errorf("expected Partial suitability, got %d. Reason: %s", est2.Suitability, est2.Reason)
	}

	// Case 3: Exceeds total system memory (e.g. 26GB file size)
	meta3 := &model.GGUFMetadata{
		FileSize:     26 * 1024 * 1024 * 1024,
		Layers:       32,
		Heads:        32,
		HeadsKV:      8,
		EmbeddingLen: 4096,
	}
	est3 := EstimateMemory(meta3, specs, 2048)
	if est3.Suitability != SuitabilityExceeds {
		t.Errorf("expected Exceeds suitability, got %d. Reason: %s", est3.Suitability, est3.Reason)
	}
}
