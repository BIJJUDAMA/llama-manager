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
	Paths             Paths               `json:"paths"`
	Favorites         []string            `json:"favorites"`
	RecentLaunches    []string            `json:"recent_launches"`
	Collections       map[string][]string `json:"collections"`
	LastSelectedModel string              `json:"last_selected_model"`
	Theme             string              `json:"theme"`
	ModelProfiles     map[string]string   `json:"model_profiles"`
	HFToken           string              `json:"hf_token"`
	ModelTags         map[string][]string `json:"model_tags"`
	ModelNotes        map[string]string   `json:"model_notes"`
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
	if cfg.ModelTags == nil {
		cfg.ModelTags = make(map[string][]string)
	}
	if cfg.ModelNotes == nil {
		cfg.ModelNotes = make(map[string]string)
	}
	if cfg.Favorites == nil {
		cfg.Favorites = []string{}
	}
	if cfg.RecentLaunches == nil {
		cfg.RecentLaunches = []string{}
	}
	if cfg.Collections == nil {
		cfg.Collections = make(map[string][]string)
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
		Favorites:         []string{},
		RecentLaunches:    []string{},
		Collections:       make(map[string][]string),
		LastSelectedModel: "",
		Theme:             "dark",
		ModelProfiles:     make(map[string]string),
		ModelTags:         make(map[string][]string),
		ModelNotes:        make(map[string]string),
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

// AddToCollection adds a model path to the specified collection.
func (c *Config) AddToCollection(collection string, modelPath string) {
	if c.Collections == nil {
		c.Collections = make(map[string][]string)
	}
	list := c.Collections[collection]
	for _, p := range list {
		if p == modelPath {
			return
		}
	}
	c.Collections[collection] = append(list, modelPath)
}

// RemoveFromCollection removes a model path from the specified collection.
func (c *Config) RemoveFromCollection(collection string, modelPath string) {
	if c.Collections == nil {
		return
	}
	list, ok := c.Collections[collection]
	if !ok {
		return
	}
	for i, p := range list {
		if p == modelPath {
			c.Collections[collection] = append(list[:i], list[i+1:]...)
			break
		}
	}
	// Clean up empty collections
	if len(c.Collections[collection]) == 0 {
		delete(c.Collections, collection)
	}
}

// AddTag adds a tag to the model path if it doesn't already exist.
func (c *Config) AddTag(modelPath string, tag string) {
	if c.ModelTags == nil {
		c.ModelTags = make(map[string][]string)
	}
	tags := c.ModelTags[modelPath]
	for _, t := range tags {
		if t == tag {
			return
		}
	}
	c.ModelTags[modelPath] = append(tags, tag)
}

// RemoveTag removes a tag from the model path.
func (c *Config) RemoveTag(modelPath string, tag string) {
	if c.ModelTags == nil {
		return
	}
	tags, ok := c.ModelTags[modelPath]
	if !ok {
		return
	}
	for i, t := range tags {
		if t == tag {
			c.ModelTags[modelPath] = append(tags[:i], tags[i+1:]...)
			break
		}
	}
	if len(c.ModelTags[modelPath]) == 0 {
		delete(c.ModelTags, modelPath)
	}
}

// SetNotes sets the notes for the model path.
func (c *Config) SetNotes(modelPath string, notes string) {
	if c.ModelNotes == nil {
		c.ModelNotes = make(map[string]string)
	}
	if notes == "" {
		delete(c.ModelNotes, modelPath)
	} else {
		c.ModelNotes[modelPath] = notes
	}
}

// GetNotes gets the notes for the model path.
func (c *Config) GetNotes(modelPath string) string {
	if c.ModelNotes == nil {
		return ""
	}
	return c.ModelNotes[modelPath]
}

