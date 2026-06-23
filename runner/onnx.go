package runner

import (
	"fmt"
	"sync"
	"time"
)

type OnnxInstance struct {
	Port       int
	ModelPath  string
	PID        int
	LaunchTime time.Time
	LogFile    string
}

type OnnxRuntime struct {
	mu        sync.Mutex
	logDir    string
	instances map[int]*OnnxInstance
}

func NewOnnxRuntime(logDir string) *OnnxRuntime {
	return &OnnxRuntime{
		logDir:    logDir,
		instances: make(map[int]*OnnxInstance),
	}
}

func (or *OnnxRuntime) Start(modelPath string, opts StartOptions) error {
	or.mu.Lock()
	defer or.mu.Unlock()

	// Check if already running on this port
	if _, exists := or.instances[opts.Port]; exists {
		return fmt.Errorf("an ONNX server is already running on port %d", opts.Port)
	}

	// TODO: Integrate actual ONNX Runtime execution script/binary here.
	// For now, we mock a successful launch.
	inst := &OnnxInstance{
		Port:       opts.Port,
		ModelPath:  modelPath,
		PID:        99999 + opts.Port, // Mock PID
		LaunchTime: time.Now(),
		LogFile:    "",
	}
	or.instances[opts.Port] = inst

	return nil
}

func (or *OnnxRuntime) Stop() error {
	or.mu.Lock()
	defer or.mu.Unlock()

	for port := range or.instances {
		delete(or.instances, port)
	}
	return nil
}

func (or *OnnxRuntime) StopInstance(port int) error {
	or.mu.Lock()
	defer or.mu.Unlock()

	delete(or.instances, port)
	return nil
}

func (or *OnnxRuntime) GetStatus() (ServerStatus, string, int) {
	or.mu.Lock()
	defer or.mu.Unlock()

	if len(or.instances) == 0 {
		return StatusStopped, "", 50505
	}

	if inst, exists := or.instances[50505]; exists {
		return StatusRunning, inst.ModelPath, 50505
	}

	for port, inst := range or.instances {
		return StatusRunning, inst.ModelPath, port
	}

	return StatusStopped, "", 50505
}

func (or *OnnxRuntime) GetAllInstances() []InstanceInfo {
	or.mu.Lock()
	defer or.mu.Unlock()

	var list []InstanceInfo
	for port, inst := range or.instances {
		list = append(list, InstanceInfo{
			Port:      port,
			ModelPath: inst.ModelPath,
			PID:       inst.PID,
			Uptime:    time.Since(inst.LaunchTime),
			LogFile:   inst.LogFile,
		})
	}
	return list
}

func (or *OnnxRuntime) Capabilities() []TaskType {
	return []TaskType{TaskEmbedding, TaskReranking, TaskSpeechToText, TaskVision, TaskImageGeneration}
}
