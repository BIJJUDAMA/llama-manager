package config

import (
	"encoding/json"
	"os"
)

type Paths struct {
	Models     string `json:"models"`
	LlamaCPP   string `json:"llama_cpp"`
	Profiles   string `json:"profiles"`
	Cache      string `json:"cache"`
	Benchmarks string `json:"benchmarks"`
	Downloads  string `json:"downloads"`
}

type Config struct {
	Paths               Paths             `json:"paths"`
	Favorites           []string          `json:"favorites"`
	RecentLaunches      []string          `json:"recent_launches"`
	LastSelectedModel   string            `json:"last_selected_model"`
	Theme               string            `json:"theme"`
	ModelProfiles       map[string]string `json:"model_profiles"`
	HFToken             string            `json:"hf_token"`
	OnboardingCompleted bool              `json:"onboarding_completed"`
}

const ConfigFileName = "config.json"

// Load loads the configuration from file, or creates a default one if it doesn't exist.
func Load() (*Config, error) {
	var cfg *Config
	if _, err := os.Stat(ConfigFileName); os.IsNotExist(err) {
		cfg = DefaultConfig()
		if err := cfg.Save(); err != nil {
			return nil, err
		}
	} else {
		data, err := os.ReadFile(ConfigFileName)
		if err != nil {
			return nil, err
		}
		cfg = &Config{}
		if err := json.Unmarshal(data, cfg); err != nil {
			// If JSON is malformed, fallback to default
			cfg = DefaultConfig()
			_ = cfg.Save()
		}
	}

	if cfg.ModelProfiles == nil {
		cfg.ModelProfiles = make(map[string]string)
	}
	if cfg.Favorites == nil {
		cfg.Favorites = []string{}
	}
	if cfg.RecentLaunches == nil {
		cfg.RecentLaunches = []string{}
	}

	// Ensure all directories specified in Paths exist
	if err := cfg.CreateDirectories(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func DefaultConfig() *Config {
	return &Config{
		Paths: Paths{
			Models:     "models",
			LlamaCPP:   "llama.cpp",
			Profiles:   "profiles",
			Cache:      "cache",
			Benchmarks: "benchmarks",
			Downloads:  "downloads",
		},
		Favorites:           []string{},
		RecentLaunches:      []string{},
		LastSelectedModel:   "",
		Theme:               "dark",
		ModelProfiles:       make(map[string]string),
		OnboardingCompleted: false,
	}
}

func (c *Config) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigFileName, data, 0600)
}

func (c *Config) CreateDirectories() error {
	dirs := []string{
		c.Paths.Models,
		c.Paths.LlamaCPP,
		c.Paths.Profiles,
		c.Paths.Cache,
		c.Paths.Benchmarks,
		c.Paths.Downloads,
	}
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

// ToggleFavorite adds or removes a model path to/from Favorites.
func (c *Config) ToggleFavorite(modelPath string) {
	for i, f := range c.Favorites {
		if f == modelPath {
			c.Favorites = append(c.Favorites[:i], c.Favorites[i+1:]...)
			return
		}
	}
	c.Favorites = append(c.Favorites, modelPath)
}

// IsFavorite returns true if the model path is in Favorites.
func (c *Config) IsFavorite(modelPath string) bool {
	for _, f := range c.Favorites {
		if f == modelPath {
			return true
		}
	}
	return false
}

// RecordLaunch prepends the model path to RecentLaunches, maintaining a max limit of 5 unique items.
func (c *Config) RecordLaunch(modelPath string) {
	// Remove if already exists to move it to the top
	for i, r := range c.RecentLaunches {
		if r == modelPath {
			c.RecentLaunches = append(c.RecentLaunches[:i], c.RecentLaunches[i+1:]...)
			break
		}
	}

	// Prepend
	c.RecentLaunches = append([]string{modelPath}, c.RecentLaunches...)

	// Limit to 5
	if len(c.RecentLaunches) > 5 {
		c.RecentLaunches = c.RecentLaunches[:5]
	}
}

