package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	configDirName       = ".ravi"
	authFileName        = "auth.json"
	configFileName      = "config.json"
	recoveryKeyFileName = "recovery-key.txt"
	configDirPerm       = 0700
	configFilePerm      = 0600
)

// AuthConfig holds tokens and encryption keys.
type AuthConfig struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	UserEmail    string `json:"user_email,omitempty"`
	PINSalt      string `json:"pin_salt,omitempty"`
	PublicKey    string `json:"public_key,omitempty"`
	PrivateKey   string `json:"private_key,omitempty"`
}

// Config holds the active identity reference.
type Config struct {
	IdentityUUID string `json:"identity_uuid,omitempty"`
	IdentityName string `json:"identity_name,omitempty"`
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

// RecoveryKeyPath returns the path to the recovery key file.
func RecoveryKeyPath() string {
	return filepath.Join(Dir(), recoveryKeyFileName)
}

// --- Auth ---

// LoadAuth reads the auth config from ~/.ravi/auth.json.
func LoadAuth() (*AuthConfig, error) {
	path := filepath.Join(Dir(), authFileName)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &AuthConfig{}, nil
		}
		return nil, fmt.Errorf("reading auth file: %w", err)
	}

	var cfg AuthConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing auth file: %w", err)
	}

	return &cfg, nil
}

// SaveAuth writes the auth config to ~/.ravi/auth.json with 0600 permissions.
func SaveAuth(cfg *AuthConfig) error {
	dir := Dir()
	if err := os.MkdirAll(dir, configDirPerm); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding auth config: %w", err)
	}

	path := filepath.Join(dir, authFileName)
	if err := os.WriteFile(path, data, configFilePerm); err != nil {
		return fmt.Errorf("writing auth file: %w", err)
	}

	return nil
}

// --- Config (identity selector) ---

// LoadConfig resolves the active identity config. Resolution order:
// 1. .ravi/config.json in CWD (project-level override)
// 2. ~/.ravi/config.json (global default)
// 3. Falls back to empty Config (unscoped)
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

	// Default: unscoped.
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

// --- Resolution helpers ---

// ResolveIdentityUUID resolves the active identity UUID from the config chain.
// Returns empty string if no identity is configured (unscoped).
func ResolveIdentityUUID() (string, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return "", err
	}
	return cfg.IdentityUUID, nil
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

// SaveRecoveryKey writes the recovery key to ~/.ravi/recovery-key.txt with 0600 permissions.
func SaveRecoveryKey(key string) error {
	dir := Dir()
	if err := os.MkdirAll(dir, configDirPerm); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	path := RecoveryKeyPath()
	if err := os.WriteFile(path, []byte(key+"\n"), configFilePerm); err != nil {
		return fmt.Errorf("writing recovery key: %w", err)
	}

	return nil
}
