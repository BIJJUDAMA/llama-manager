package runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

type ServerStatus int

const (
	StatusStopped ServerStatus = iota
	StatusRunning
	StatusFailed
)

type ServerInstance struct {
	Port       int
	ModelPath  string
	PID        int
	Cmd        *exec.Cmd
	LaunchTime time.Time
	LogFile    string
	cancelFunc context.CancelFunc
}

type InstanceInfo struct {
	Port      int
	ModelPath string
	PID       int
	Uptime    time.Duration
	LogFile   string
}

type ServerRunner struct {
	mu        sync.Mutex
	logDir    string
	instances map[int]*ServerInstance
}

func NewServerRunner(logDir string) *ServerRunner {
	return &ServerRunner{
		logDir:    logDir,
		instances: make(map[int]*ServerInstance),
	}
}

// Start launches the llama-server on the specified port.
func (sr *ServerRunner) Start(llamaCppDir string, modelPath string, ctxSize uint32, threads int, gpuLayers int, batchSize int, host string, port int) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	// Check if already running on this port
	if _, exists := sr.instances[port]; exists {
		return fmt.Errorf("a server is already running on port %d", port)
	}

	// Resolve binary name
	binaryName := "llama-server"
	if runtime.GOOS == "windows" {
		binaryName = "llama-server.exe"
	}
	binaryPath := filepath.Join(llamaCppDir, binaryName)

	// Check if binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("llama-server binary not found at %s", binaryPath)
	}

	// Prepare arguments
	args := []string{
		"--model", modelPath,
		"--host", host,
		"--port", fmt.Sprintf("%d", port),
	}
	if ctxSize > 0 {
		args = append(args, "--ctx-size", fmt.Sprintf("%d", ctxSize))
	}
	if threads > 0 {
		args = append(args, "--threads", fmt.Sprintf("%d", threads))
	}
	if gpuLayers >= 0 {
		args = append(args, "--n-gpu-layers", fmt.Sprintf("%d", gpuLayers))
	}
	if batchSize > 0 {
		args = append(args, "--batch-size", fmt.Sprintf("%d", batchSize))
	}

	// Open log file specific to this port
	logFileName := fmt.Sprintf("llama-server-%d.log", port)
	logFilePath := filepath.Join(sr.logDir, logFileName)
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", logFilePath, err)
	}

	// Setup context for cancellation
	ctx, cancel := context.WithCancel(context.Background())

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Configure system process attributes
	configureSysProcAttr(cmd)

	if err := cmd.Start(); err != nil {
		logFile.Close()
		cancel()
		return fmt.Errorf("failed to start process: %w", err)
	}

	pid := 0
	if cmd.Process != nil {
		pid = cmd.Process.Pid
	}

	inst := &ServerInstance{
		Port:       port,
		ModelPath:  modelPath,
		PID:        pid,
		Cmd:        cmd,
		LaunchTime: time.Now(),
		LogFile:    logFilePath,
		cancelFunc: cancel,
	}
	sr.instances[port] = inst

	// Monitor termination in goroutine
	go func(p int, lf *os.File) {
		defer lf.Close()
		_ = cmd.Wait()

		sr.mu.Lock()
		defer sr.mu.Unlock()
		delete(sr.instances, p)
	}(port, logFile)

	return nil
}

// Stop terminates ALL running servers.
func (sr *ServerRunner) Stop() error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	for port, inst := range sr.instances {
		if inst.cancelFunc != nil {
			inst.cancelFunc()
		}
		if inst.Cmd != nil && inst.Cmd.Process != nil {
			_ = inst.Cmd.Process.Kill()
		}
		delete(sr.instances, port)
	}
	return nil
}

// StopInstance terminates the server running on the specified port.
func (sr *ServerRunner) StopInstance(port int) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	inst, exists := sr.instances[port]
	if !exists {
		return nil
	}

	if inst.cancelFunc != nil {
		inst.cancelFunc()
	}
	if inst.Cmd != nil && inst.Cmd.Process != nil {
		_ = inst.Cmd.Process.Kill()
	}
	delete(sr.instances, port)
	return nil
}

// GetStatus returns the status, running model path, and port of the primary running server (8080 or first found).
func (sr *ServerRunner) GetStatus() (ServerStatus, string, int) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	if len(sr.instances) == 0 {
		return StatusStopped, "", 8080
	}

	// Prefer default 8080 if running
	if inst, exists := sr.instances[8080]; exists {
		return StatusRunning, inst.ModelPath, 8080
	}

	// Fallback to first active one
	for port, inst := range sr.instances {
		return StatusRunning, inst.ModelPath, port
	}

	return StatusStopped, "", 8080
}

// GetAllInstances returns status information for all active servers.
func (sr *ServerRunner) GetAllInstances() []InstanceInfo {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	var list []InstanceInfo
	for port, inst := range sr.instances {
		pid := 0
		if inst.Cmd != nil && inst.Cmd.Process != nil {
			pid = inst.Cmd.Process.Pid
		}
		list = append(list, InstanceInfo{
			Port:      port,
			ModelPath: inst.ModelPath,
			PID:       pid,
			Uptime:    time.Since(inst.LaunchTime),
			LogFile:   inst.LogFile,
		})
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Port < list[j].Port
	})
	return list
}

// GetMemoryUsage queries physical memory usage (RSS) of a process in MB.
func GetMemoryUsage(pid int) (float64, error) {
	if pid <= 0 {
		return 0, fmt.Errorf("invalid pid")
	}
	if runtime.GOOS == "windows" {
		cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH")
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			return 0, err
		}
		lines := strings.Split(out.String(), "\n")
		for _, line := range lines {
			if strings.Contains(line, fmt.Sprintf(`"%d"`, pid)) || strings.Contains(line, fmt.Sprintf(`,%d,`, pid)) {
				parts := strings.Split(line, ",")
				if len(parts) >= 5 {
					memStr := strings.Trim(parts[4], ` "`)
					memStr = strings.ReplaceAll(memStr, ",", "")
					memStr = strings.ReplaceAll(memStr, ".", "")
					memStr = strings.TrimSuffix(memStr, " K")
					memStr = strings.TrimSuffix(memStr, " KB")
					var kb float64
					if _, err := fmt.Sscanf(memStr, "%f", &kb); err == nil {
						return kb / 1024.0, nil
					}
				}
			}
		}
		return 0, fmt.Errorf("process not found in tasklist")
	} else {
		cmd := exec.Command("ps", "-p", fmt.Sprintf("%d", pid), "-o", "rss=")
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			return 0, err
		}
		var kb float64
		if _, err := fmt.Sscanf(strings.TrimSpace(out.String()), "%f", &kb); err == nil {
			return kb / 1024.0, nil
		}
		return 0, fmt.Errorf("failed to parse ps output")
	}
}

// QueryServerRequests queries total completion requests processed via the /metrics endpoint.
func QueryServerRequests(port int) (int, error) {
	client := http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/metrics", port))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "llama_requests_total") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				var count int
				if _, err := fmt.Sscanf(parts[len(parts)-1], "%d", &count); err == nil {
					return count, nil
				}
			}
		}
	}
	return 0, nil
}
