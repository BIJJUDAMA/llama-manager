package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Profile struct {
	Name      string `json:"name"`
	Context   uint32 `json:"context"`
	Threads   int    `json:"threads"`
	GPULayers int    `json:"gpu_layers"`
	BatchSize int    `json:"batch_size"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
}

// DefaultProfiles generates the default profiles based on system cpu count.
func DefaultProfiles() []*Profile {
	threads := runtime.NumCPU() / 2
	if threads < 1 {
		threads = 1
	}

	return []*Profile{
		{
			Name:      "Fast",
			Context:   2048,
			Threads:   threads,
			GPULayers: 999, // default to offload as much as possible
			BatchSize: 512,
			Host:      "127.0.0.1",
			Port:      50505,
		},
		{
			Name:      "Balanced",
			Context:   4096,
			Threads:   threads,
			GPULayers: 999,
			BatchSize: 512,
			Host:      "127.0.0.1",
			Port:      50505,
		},
		{
			Name:      "High",
			Context:   8192,
			Threads:   threads,
			GPULayers: 999,
			BatchSize: 512,
			Host:      "127.0.0.1",
			Port:      50505,
		},
		{
			Name:      "Long Context",
			Context:   16384,
			Threads:   threads,
			GPULayers: 999,
			BatchSize: 512,
			Host:      "127.0.0.1",
			Port:      50505,
		},
		{
			Name:      "CPU",
			Context:   2048,
			Threads:   runtime.NumCPU(),
			GPULayers: 0,
			BatchSize: 512,
			Host:      "127.0.0.1",
			Port:      50505,
		},
	}
}

// LoadAll reads all profiles from the specified profiles directory, 
// auto-generating defaults if no profile files exist.
func LoadAll(profilesDir string) ([]*Profile, error) {
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		return nil, err
	}

	// Always ensure default profiles exist in the folder
	defaults := DefaultProfiles()
	for _, p := range defaults {
		fileName := strings.ReplaceAll(strings.ToLower(p.Name), " ", "_") + ".json"
		filePath := filepath.Join(profilesDir, fileName)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			data, err := json.MarshalIndent(p, "", "  ")
			if err == nil {
				_ = os.WriteFile(filePath, data, 0644)
			}
		}
	}

	files, err := os.ReadDir(profilesDir)
	if err != nil {
		return nil, err
	}

	var profiles []*Profile
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			filePath := filepath.Join(profilesDir, file.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}
			var p Profile
			if err := json.Unmarshal(data, &p); err == nil {
				profiles = append(profiles, &p)
			}
		}
	}

	return profiles, nil
}
