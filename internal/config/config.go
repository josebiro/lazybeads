package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	CustomCommands []CustomCommand `yaml:"customCommands"`
}

// CustomCommand represents a user-defined command
type CustomCommand struct {
	Key         string `yaml:"key"`
	Description string `yaml:"description"`
	Context     string `yaml:"context"` // list, detail, or global
	Command     string `yaml:"command"`
}

// Load reads the configuration from the default location
func Load() (*Config, error) {
	configPath := ConfigPath()

	// If config file doesn't exist, return empty config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &Config{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Set defaults for context if not specified
	for i := range cfg.CustomCommands {
		if cfg.CustomCommands[i].Context == "" {
			cfg.CustomCommands[i].Context = "list"
		}
	}

	return &cfg, nil
}

// ConfigPath returns the config file path to use.
// It checks in order:
//  1. BB_CONFIG environment variable (direct path to config file)
//  2. Default XDG config location (~/.config/bb/config.yml)
func ConfigPath() string {
	// First, check for explicit config file path
	if configFile := os.Getenv("BB_CONFIG"); configFile != "" {
		return configFile
	}

	return DefaultConfigPath()
}

// DefaultConfigPath returns the default config file path
func DefaultConfigPath() string {
	// Check XDG_CONFIG_HOME first
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "bb", "config.yml")
	}

	// Always use ~/.config for CLI tools (cross-platform convention)
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "bb", "config.yml")
}
