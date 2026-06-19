//go:build windows

package hardware

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

type memoryStatusEx struct {
	dwLength                uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

func DetectHardware() (*HardwareSpecs, error) {
	specs := &HardwareSpecs{
		OS:        "Windows",
		IsUnified: false, // Unified memory is macOS Apple Silicon specific
	}

	// 1. CPU Detection
	specs.CPU.Threads = runtime.NumCPU()
	specs.CPU.Model = getWindowsCPUModel()

	// 2. RAM Detection
	totalRAM, availRAM, err := getWindowsRAM()
	if err == nil {
		specs.RAM.Total = totalRAM
		specs.RAM.Available = availRAM
	} else {
		// Fallback to safe defaults if DLL call fails
		specs.RAM.Total = 8 * 1024 * 1024 * 1024
		specs.RAM.Available = 4 * 1024 * 1024 * 1024
	}

	// 3. GPU & VRAM Detection
	gpuName, gpuVRAM, gpuType := getWindowsGPU()
	specs.GPU.Name = gpuName
	specs.GPU.VRAM = gpuVRAM
	specs.GPU.Type = gpuType

	return specs, nil
}

func getWindowsCPUModel() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `HARDWARE\DESCRIPTION\System\CentralProcessor\0`, registry.QUERY_VALUE)
	if err != nil {
		return "Unknown CPU"
	}
	defer k.Close()

	model, _, err := k.GetStringValue("ProcessorNameString")
	if err != nil {
		return "Unknown CPU"
	}
	return strings.TrimSpace(model)
}

func getWindowsRAM() (uint64, uint64, error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	globalMemoryStatusEx := kernel32.NewProc("GlobalMemoryStatusEx")

	var memStatus memoryStatusEx
	memStatus.dwLength = uint32(unsafe.Sizeof(memStatus))

	ret, _, err := globalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&memStatus)))
	if ret == 0 {
		return 0, 0, err
	}

	return memStatus.ullTotalPhys, memStatus.ullAvailPhys, nil
}

func getWindowsGPU() (string, uint64, string) {
	// First, check if nvidia-smi is available
	nvSmiPath := "nvidia-smi"
	// Check standard location if not in PATH
	stdNvSmi := filepath.Join(os.Getenv("ProgramFiles"), "NVIDIA Corporation", "NVSMI", "nvidia-smi.exe")
	if _, err := os.Stat(stdNvSmi); err == nil {
		nvSmiPath = stdNvSmi
	}

	cmd := exec.Command(nvSmiPath, "--query-gpu=name,memory.total", "--format=csv,noheader,nounits")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil {
		parts := strings.Split(strings.TrimSpace(out.String()), ",")
		if len(parts) >= 2 {
			name := strings.TrimSpace(parts[0])
			vramMb, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64)
			if err == nil {
				return name, vramMb * 1024 * 1024, "CUDA"
			}
		}
	}

	// Fallback to wmic for general GPU detection
	cmdWmic := exec.Command("wmic", "path", "win32_VideoController", "get", "Name,AdapterRAM", "/value")
	var outWmic bytes.Buffer
	cmdWmic.Stdout = &outWmic
	if err := cmdWmic.Run(); err == nil {
		lines := strings.Split(outWmic.String(), "\n")
		var name string
		var vramBytes uint64
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Name=") {
				name = strings.TrimPrefix(line, "Name=")
			} else if strings.HasPrefix(line, "AdapterRAM=") {
				vramStr := strings.TrimPrefix(line, "AdapterRAM=")
				// WMIC sometimes reports RAM as negative or signed integer on some systems, parse absolute val
				val, err := strconv.ParseInt(vramStr, 10, 64)
				if err == nil {
					if val < 0 {
						val = -val
					}
					vramBytes = uint64(val)
				}
			}
		}
		if name != "" {
			gpuType := "CPU"
			// Check if name implies NVIDIA or AMD
			lowerName := strings.ToLower(name)
			if strings.Contains(lowerName, "nvidia") || strings.Contains(lowerName, "geforce") || strings.Contains(lowerName, "quadro") || strings.Contains(lowerName, "tesla") || strings.Contains(lowerName, "rtx") {
				gpuType = "CUDA"
			} else if strings.Contains(lowerName, "amd") || strings.Contains(lowerName, "radeon") {
				gpuType = "ROCm"
			} else if strings.Contains(lowerName, "intel") {
				gpuType = "Intel"
			}
			return name, vramBytes, gpuType
		}
	}

	return "Integrated Graphics / CPU", 0, "CPU"
}
