//go:build !windows

package hardware

import (
	"runtime"
)

func DetectHardware() (*HardwareSpecs, error) {
	// TODO: Implement macOS Apple Silicon unified memory detection and GPU properties
	// TODO: Implement Linux /proc/cpuinfo, /proc/meminfo, and nvidia-smi GPU parsing
	
	// Fallback/Default Specifications for Unix/macOS build targets (to be implemented in future phases)
	specs := &HardwareSpecs{
		OS:        runtime.GOOS,
		IsUnified: false,
	}

	specs.CPU.Threads = runtime.NumCPU()
	specs.CPU.Model = "Unix-class CPU (TODO: parse model)"

	// Fallback to safe default memory specifications
	specs.RAM.Total = 8 * 1024 * 1024 * 1024       // 8 GB
	specs.RAM.Available = 4 * 1024 * 1024 * 1024   // 4 GB
	
	specs.GPU.Name = "Unix Video Controller (TODO: query GPU info)"
	specs.GPU.VRAM = 0
	specs.GPU.Type = "CPU"

	return specs, nil
}
