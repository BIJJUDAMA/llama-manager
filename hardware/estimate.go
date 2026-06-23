package hardware

import (
	"fmt"
	"github.com/BIJJUDAMA/runora/model"
)

type Suitability int

const (
	SuitabilityFits Suitability = iota // Green
	SuitabilityPartial                 // Yellow
	SuitabilityExceeds                 // Red
)

type MemoryEstimate struct {
	WeightSize    uint64
	KVCacheSize   uint64
	Overhead      uint64
	TotalMemory   uint64
	Suitability   Suitability
	Reason        string
	GPUOffloadPct int // Percentage of weights offloaded to GPU
}

// EstimateMemory computes estimated memory sizes and suitability for a given model.
func EstimateMemory(meta *model.GGUFMetadata, specs *HardwareSpecs, contextLength uint32) *MemoryEstimate {
	if contextLength == 0 {
		contextLength = meta.ContextLength
		if contextLength == 0 {
			contextLength = 2048 // default fallback
		}
	}

	// Standard fallback dimensions if GGUF keys are missing
	layers := meta.Layers
	if layers == 0 {
		layers = 32
	}
	heads := meta.Heads
	if heads == 0 {
		heads = 32
	}
	headsKV := meta.HeadsKV
	if headsKV == 0 {
		headsKV = 8
	}
	embedLen := meta.EmbeddingLen
	if embedLen == 0 {
		embedLen = 4096
	}

	headDim := meta.HeadDim
	if headDim == 0 {
		headDim = embedLen / heads
		if headDim == 0 {
			headDim = 128 // default fallback
		}
	}

	// KV Cache Size: 4 bytes per token context: 2 bytes for key, 2 bytes for value per layer element
	kvCacheSize := uint64(4) * uint64(layers) * uint64(headsKV) * uint64(headDim) * uint64(contextLength)
	weightSize := uint64(meta.FileSize)
	overhead := uint64(512 * 1024 * 1024) // 512 MB general llama.cpp runtime overhead
	totalMemory := weightSize + kvCacheSize + overhead

	est := &MemoryEstimate{
		WeightSize:  weightSize,
		KVCacheSize: kvCacheSize,
		Overhead:    overhead,
		TotalMemory: totalMemory,
	}

	if specs.IsUnified {
		// macOS Apple Silicon Unified Memory
		if totalMemory <= specs.GPU.VRAM {
			est.Suitability = SuitabilityFits
			est.GPUOffloadPct = 100
			est.Reason = "Fits fully in Unified Memory (Metal accelerated)"
		} else if totalMemory <= specs.RAM.Total {
			est.Suitability = SuitabilityPartial
			est.GPUOffloadPct = int((float64(specs.GPU.VRAM) / float64(totalMemory)) * 100)
			if est.GPUOffloadPct > 99 {
				est.GPUOffloadPct = 99
			}
			est.Reason = "Fits in system RAM; partial Unified GPU offload"
		} else {
			est.Suitability = SuitabilityExceeds
			est.GPUOffloadPct = 0
			est.Reason = "Exceeds total system memory; severe performance lag expected"
		}
		return est
	}

	// Dedicated RAM / VRAM system (Windows/Linux)
	if specs.GPU.VRAM > 0 {
		if totalMemory <= specs.GPU.VRAM {
			est.Suitability = SuitabilityFits
			est.GPUOffloadPct = 100
			est.Reason = fmt.Sprintf("Fits fully in GPU VRAM (%s)", specs.GPU.Name)
		} else if specs.GPU.VRAM > overhead+100*1024*1024 {
			// Some capacity to offload to GPU
			est.Suitability = SuitabilityPartial
			vramAvailableForWeights := int64(specs.GPU.VRAM) - int64(kvCacheSize) - int64(overhead)
			if vramAvailableForWeights > 0 {
				est.GPUOffloadPct = int((float64(vramAvailableForWeights) / float64(weightSize)) * 100)
			} else {
				est.GPUOffloadPct = int((float64(specs.GPU.VRAM) / float64(totalMemory)) * 100)
			}
			if est.GPUOffloadPct > 99 {
				est.GPUOffloadPct = 99
			}
			if est.GPUOffloadPct < 5 {
				est.GPUOffloadPct = 5
			}

			// Check if it fits in total system memory
			if totalMemory > specs.GPU.VRAM+specs.RAM.Total {
				est.Suitability = SuitabilityExceeds
				est.Reason = "Exceeds combined GPU VRAM and System RAM"
			} else {
				est.Reason = "Partial GPU offload; remaining layers will run on CPU"
			}
		} else {
			// Insufficient VRAM, treat as CPU-only
			if totalMemory <= specs.RAM.Total {
				est.Suitability = SuitabilityFits
				est.GPUOffloadPct = 0
				est.Reason = "Runs on CPU-only (Insufficient VRAM)"
			} else {
				est.Suitability = SuitabilityExceeds
				est.GPUOffloadPct = 0
				est.Reason = "Exceeds system memory limits"
			}
		}
	} else {
		// CPU-only system
		if totalMemory <= specs.RAM.Total {
			est.Suitability = SuitabilityFits
			est.GPUOffloadPct = 0
			est.Reason = "Fits system RAM (Runs on CPU-only)"
		} else {
			est.Suitability = SuitabilityExceeds
			est.GPUOffloadPct = 0
			est.Reason = "Exceeds total system RAM"
		}
	}

	return est
}
