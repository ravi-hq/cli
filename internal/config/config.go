package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	configDirName  = ".ravi"
	configFileName = "config.json"
	configDirPerm  = 0700
	configFilePerm = 0600
)

// Config holds API keys and active identity info.
// Single file at ~/.ravi/config.json (or .ravi/config.json in CWD for project overrides).
type Config struct {
	ManagementKey string `json:"management_key,omitempty"`
	IdentityKey   string `json:"identity_key,omitempty"`
	IdentityUUID  string `json:"identity_uuid,omitempty"`
	IdentityName  string `json:"identity_name,omitempty"`
	UserEmail     string `json:"user_email,omitempty"`
}

// Dir returns the global config directory (~/.ravi/).
func Dir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: $HOME not set, using ./.ravi/ for config\n")
		return filepath.Join(".", configDirName)
	}
	return filepath.Join(homeDir, configDirName)
}

// LoadConfig resolves the active config. Resolution order:
// 1. .ravi/config.json in CWD (project-level override)
// 2. ~/.ravi/config.json (global default)
// 3. Falls back to empty Config
func LoadConfig() (*Config, error) {
	// Try CWD first.
	if cwd, err := os.Getwd(); err == nil {
		localPath := filepath.Join(cwd, configDirName, configFileName)
		cfg, err := loadConfigFile(localPath)
		if err == nil {
			return cfg, nil
		}
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("reading project config %s: %w", localPath, err)
		}
	}

	// Try global config.
	globalPath := filepath.Join(Dir(), configFileName)
	cfg, err := loadConfigFile(globalPath)
	if err == nil {
		return cfg, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading global config %s: %w", globalPath, err)
	}

	// Default: empty.
	return &Config{}, nil
}

func loadConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err // Pass through raw error so caller can check os.IsNotExist
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	return &cfg, nil
}

// SaveGlobalConfig writes the global config to ~/.ravi/config.json.
func SaveGlobalConfig(cfg *Config) error {
	dir := Dir()
	if err := os.MkdirAll(dir, configDirPerm); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	path := filepath.Join(dir, configFileName)
	if err := os.WriteFile(path, data, configFilePerm); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

// SaveConfig writes the config to the appropriate location:
// - If .ravi/config.json exists in CWD, write there (project-level)
// - Otherwise, write to ~/.ravi/config.json (global)
// This mirrors LoadConfig() which reads CWD first.
func SaveConfig(cfg *Config) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	localPath := filepath.Join(cwd, configDirName, configFileName)
	if _, err := os.Stat(localPath); err == nil {
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return fmt.Errorf("encoding config: %w", err)
		}
		if err := os.WriteFile(localPath, data, configFilePerm); err != nil {
			return fmt.Errorf("writing project config: %w", err)
		}
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("checking project config %s: %w", localPath, err)
	}

	return SaveGlobalConfig(cfg)
}

// --- Cleanup ---

// ClearAll removes the entire ~/.ravi/ directory.
func ClearAll() error {
	dir := Dir()
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("removing config directory: %w", err)
	}
	return nil
}
