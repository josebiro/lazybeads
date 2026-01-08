package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	configContent := `customCommands:
  - key: "D"
    description: "Test command"
    context: "list"
    command: "echo hello"
  - key: "C"
    description: "Another command"
    context: "detail"
    command: "echo {{.ID}}"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Override config path for testing
	originalUserConfigDir := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", originalUserConfigDir)

	// Create the lazybeads subdirectory
	if err := os.MkdirAll(filepath.Join(tmpDir, "lazybeads"), 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "lazybeads", "config.yml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(cfg.CustomCommands) != 2 {
		t.Errorf("expected 2 custom commands, got %d", len(cfg.CustomCommands))
	}

	if cfg.CustomCommands[0].Key != "D" {
		t.Errorf("expected first command key to be 'D', got '%s'", cfg.CustomCommands[0].Key)
	}

	if cfg.CustomCommands[0].Context != "list" {
		t.Errorf("expected first command context to be 'list', got '%s'", cfg.CustomCommands[0].Context)
	}

	if cfg.CustomCommands[1].Context != "detail" {
		t.Errorf("expected second command context to be 'detail', got '%s'", cfg.CustomCommands[1].Context)
	}
}

func TestLoadNoConfig(t *testing.T) {
	// Point to a nonexistent config directory
	tmpDir := t.TempDir()
	originalUserConfigDir := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", originalUserConfigDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error for missing config, got: %v", err)
	}

	if len(cfg.CustomCommands) != 0 {
		t.Errorf("expected empty custom commands for missing config, got %d", len(cfg.CustomCommands))
	}
}

func TestLoadFromEnvVar(t *testing.T) {
	// Create a temporary config file in a non-standard location (simulating dotfiles)
	tmpDir := t.TempDir()
	dotfilesDir := filepath.Join(tmpDir, "dotfiles", "lazybeads")
	if err := os.MkdirAll(dotfilesDir, 0755); err != nil {
		t.Fatalf("failed to create dotfiles dir: %v", err)
	}

	configPath := filepath.Join(dotfilesDir, "config.yml")
	configContent := `customCommands:
  - key: "Z"
    description: "Dotfiles command"
    context: "list"
    command: "echo from dotfiles"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Set LAZYBEADS_CONFIG to point to the dotfiles config
	originalLazybeadsConfig := os.Getenv("LAZYBEADS_CONFIG")
	os.Setenv("LAZYBEADS_CONFIG", configPath)
	defer os.Setenv("LAZYBEADS_CONFIG", originalLazybeadsConfig)

	// Also set XDG_CONFIG_HOME to ensure we're using LAZYBEADS_CONFIG, not XDG
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "xdg-config"))
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(cfg.CustomCommands) != 1 {
		t.Errorf("expected 1 custom command, got %d", len(cfg.CustomCommands))
	}

	if cfg.CustomCommands[0].Key != "Z" {
		t.Errorf("expected command key to be 'Z', got '%s'", cfg.CustomCommands[0].Key)
	}

	if cfg.CustomCommands[0].Description != "Dotfiles command" {
		t.Errorf("expected description to be 'Dotfiles command', got '%s'", cfg.CustomCommands[0].Description)
	}
}

func TestConfigPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Test that LAZYBEADS_CONFIG takes precedence
	customPath := filepath.Join(tmpDir, "custom", "config.yml")
	originalLazybeadsConfig := os.Getenv("LAZYBEADS_CONFIG")
	os.Setenv("LAZYBEADS_CONFIG", customPath)
	defer os.Setenv("LAZYBEADS_CONFIG", originalLazybeadsConfig)

	got := ConfigPath()
	if got != customPath {
		t.Errorf("expected ConfigPath() to return '%s', got '%s'", customPath, got)
	}

	// Test that it falls back to default when LAZYBEADS_CONFIG is not set
	os.Unsetenv("LAZYBEADS_CONFIG")
	got = ConfigPath()
	if got == customPath {
		t.Error("ConfigPath() should not return custom path when LAZYBEADS_CONFIG is unset")
	}
}

func TestDefaultContext(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "lazybeads"), 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	// Config with no context specified
	configContent := `customCommands:
  - key: "X"
    description: "No context"
    command: "echo test"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "lazybeads", "config.yml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	originalUserConfigDir := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", originalUserConfigDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(cfg.CustomCommands) != 1 {
		t.Fatalf("expected 1 custom command, got %d", len(cfg.CustomCommands))
	}

	// Should default to "list"
	if cfg.CustomCommands[0].Context != "list" {
		t.Errorf("expected default context to be 'list', got '%s'", cfg.CustomCommands[0].Context)
	}
}
