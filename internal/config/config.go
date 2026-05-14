package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Provider     string `yaml:"provider"`
	Model        string `yaml:"model"`
	APIKey       string `yaml:"api_key"`
	BaseURL      string `yaml:"base_url"`
	ContextLines int    `yaml:"context_lines"`
	Shell        string `yaml:"shell"`
	TriggerKey   string `yaml:"trigger_key"`
}

// Engagement holds mission-scoping context injected into all LLM system prompts.
type Engagement struct {
	Scope     string `yaml:"scope"`
	Type      string `yaml:"type"`
	Objective string `yaml:"objective"`
	Notes     string `yaml:"notes"`
}

// LoadEngagement reads an engagement YAML file from path.
func LoadEngagement(path string) (*Engagement, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var eng Engagement
	if err := yaml.Unmarshal(data, &eng); err != nil {
		return nil, err
	}
	return &eng, nil
}

func Default() *Config {
	return &Config{
		Provider:     "ollama",
		Model:        "llama3.2",
		APIKey:       "",
		BaseURL:      "http://localhost:11434",
		ContextLines: 500,
		Shell:        "/bin/bash",
		TriggerKey:   "ctrl+g",
	}
}

// Load reads the config file at path (or the default location if empty),
// then applies environment variable overrides.
func Load(path string) (*Config, error) {
	cfg := Default()

	if path == "" {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, ".config", "redterm", "config.yaml")
		}
	}

	if path != "" {
		data, err := os.ReadFile(path)
		if err == nil {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, err
			}
		}
		// Silently ignore missing config file — defaults are used.
	}

	if v := os.Getenv("REDTERM_PROVIDER"); v != "" {
		cfg.Provider = v
	}
	if v := os.Getenv("REDTERM_MODEL"); v != "" {
		cfg.Model = v
	}
	if v := os.Getenv("REDTERM_API_KEY"); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv("REDTERM_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}

	return cfg, nil
}

// TriggerByte returns the raw byte value for the configured trigger key.
func (c *Config) TriggerByte() byte {
	switch strings.ToLower(strings.ReplaceAll(c.TriggerKey, " ", "")) {
	case "ctrl+g":
		return 0x07
	case "ctrl+b":
		return 0x02
	case "ctrl+x":
		return 0x18
	case "ctrl+a":
		return 0x01
	default:
		return 0x07 // fallback to Ctrl+G
	}
}
