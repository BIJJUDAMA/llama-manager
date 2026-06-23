package hardware

type CPUSpecs struct {
	Model   string
	Threads int
}

type RAMSpecs struct {
	Total     uint64 // in bytes
	Available uint64 // in bytes
}

type GPUSpecs struct {
	Name        string
	VRAM        uint64 // in bytes
	Type        string // e.g. CUDA, Metal, ROCm, CPU-only
	CudaVersion string
}

type HardwareSpecs struct {
	CPU       CPUSpecs
	RAM       RAMSpecs
	GPU       GPUSpecs
	OS        string
	IsUnified bool // True if system has unified memory (Apple Silicon Mac)
}
