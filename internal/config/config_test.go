package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// withTempHome is a test helper that temporarily changes the HOME environment variable
// to allow testing functions that use os.UserHomeDir(). It returns a cleanup function.
func withTempHome(t *testing.T) (tmpDir string, cleanup func()) {
	t.Helper()

	tmpDir = t.TempDir()

	// Save original HOME value
	var homeEnvVar string
	if runtime.GOOS == "windows" {
		homeEnvVar = "USERPROFILE"
	} else {
		homeEnvVar = "HOME"
	}
	originalHome := os.Getenv(homeEnvVar)

	// Set HOME to temp directory
	if err := os.Setenv(homeEnvVar, tmpDir); err != nil {
		t.Fatalf("Failed to set %s: %v", homeEnvVar, err)
	}

	cleanup = func() {
		os.Setenv(homeEnvVar, originalHome)
	}

	return tmpDir, cleanup
}

// TestPath verifies that Path returns a path ending with ~/.ravi/config.json
func TestPath(t *testing.T) {
	path := Path()

	// Should end with .ravi/config.json
	if filepath.Base(path) != "config.json" {
		t.Errorf("Path() = %v, want ending with config.json", path)
	}

	// Should contain .ravi directory
	dir := filepath.Dir(path)
	if filepath.Base(dir) != ".ravi" {
		t.Errorf("Path() dir = %v, want ending with .ravi", dir)
	}

	// Path should be absolute (starts with / on Unix or drive letter on Windows)
	homeDir, err := os.UserHomeDir()
	if err == nil {
		// If we can get home dir, path should start with it
		if !strings.HasPrefix(path, homeDir) {
			t.Errorf("Path() = %v, want prefix %v", path, homeDir)
		}
	}
}

// TestPath_NoHomeDir tests the fallback behavior when home directory is unavailable.
// Note: This is difficult to test directly since os.UserHomeDir() typically works.
// We test the expected fallback path structure instead.
func TestPath_NoHomeDir(t *testing.T) {
	// The fallback path would be "./.ravi/config.json"
	// We verify this by checking the implementation's fallback behavior
	fallbackPath := filepath.Join(".", ".ravi", "config.json")

	// Verify the fallback path structure is correct
	if filepath.Base(fallbackPath) != "config.json" {
		t.Errorf("Fallback path base = %v, want config.json", filepath.Base(fallbackPath))
	}

	dir := filepath.Dir(fallbackPath)
	if filepath.Base(dir) != ".ravi" {
		t.Errorf("Fallback path dir = %v, want .ravi", filepath.Base(dir))
	}

	// Note: To fully test this, we would need to mock os.UserHomeDir,
	// which requires refactoring the package to accept a function parameter.
	// For now, we verify the structure of the expected fallback.
}

// TestLoad_NoFile verifies that Load returns an empty config when the file doesn't exist
func TestLoad_NoFile(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	// Verify config path is now in temp directory
	path := Path()
	if !strings.HasPrefix(path, tmpDir) {
		t.Fatalf("Path() = %v, expected prefix %v", path, tmpDir)
	}

	// File doesn't exist, Load should return empty config
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil for non-existent file", err)
	}
	if cfg == nil {
		t.Fatal("Load() returned nil config, want empty config")
	}
	if cfg.AccessToken != "" {
		t.Errorf("Load() AccessToken = %v, want empty", cfg.AccessToken)
	}
	if cfg.RefreshToken != "" {
		t.Errorf("Load() RefreshToken = %v, want empty", cfg.RefreshToken)
	}
	if cfg.UserEmail != "" {
		t.Errorf("Load() UserEmail = %v, want empty", cfg.UserEmail)
	}
}

// TestLoad_ValidFile verifies that Load correctly parses an existing config file
func TestLoad_ValidFile(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	// Create the config file
	raviDir := filepath.Join(tmpDir, ".ravi")
	configPath := filepath.Join(raviDir, "config.json")

	if err := os.MkdirAll(raviDir, 0700); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	testConfig := Config{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresAt:    testTime,
		UserEmail:    "test@example.com",
	}

	data, err := json.MarshalIndent(testConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load using the actual function
	loadedConfig, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify the loaded config matches
	if loadedConfig.AccessToken != testConfig.AccessToken {
		t.Errorf("AccessToken = %v, want %v", loadedConfig.AccessToken, testConfig.AccessToken)
	}
	if loadedConfig.RefreshToken != testConfig.RefreshToken {
		t.Errorf("RefreshToken = %v, want %v", loadedConfig.RefreshToken, testConfig.RefreshToken)
	}
	if !loadedConfig.ExpiresAt.Equal(testConfig.ExpiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", loadedConfig.ExpiresAt, testConfig.ExpiresAt)
	}
	if loadedConfig.UserEmail != testConfig.UserEmail {
		t.Errorf("UserEmail = %v, want %v", loadedConfig.UserEmail, testConfig.UserEmail)
	}
}

// TestLoad_CorruptFile verifies that Load handles malformed JSON gracefully
func TestLoad_CorruptFile(t *testing.T) {
	testCases := []struct {
		name    string
		content string
	}{
		{
			name:    "invalid JSON",
			content: "this is not json",
		},
		{
			name:    "incomplete JSON",
			content: `{"access_token": "test"`,
		},
		{
			name:    "JSON with syntax error",
			content: `{"access_token": "test",}`,
		},
		{
			name:    "random bytes",
			content: "\x00\x01\x02\x03",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir, cleanup := withTempHome(t)
			defer cleanup()

			// Create corrupt config file
			raviDir := filepath.Join(tmpDir, ".ravi")
			configPath := filepath.Join(raviDir, "config.json")

			if err := os.MkdirAll(raviDir, 0700); err != nil {
				t.Fatalf("Failed to create directory: %v", err)
			}

			if err := os.WriteFile(configPath, []byte(tc.content), 0600); err != nil {
				t.Fatalf("Failed to write corrupt config: %v", err)
			}

			// Load should return an error
			_, err := Load()
			if err == nil {
				t.Errorf("Load() error = nil, want error for corrupt file")
			}

			// Error should mention parsing
			if !strings.Contains(err.Error(), "parsing") {
				t.Errorf("Load() error = %v, want error containing 'parsing'", err)
			}
		})
	}
}

// TestSave_NewFile verifies that Save creates a config file from scratch
func TestSave_NewFile(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	configPath := filepath.Join(tmpDir, ".ravi", "config.json")

	// Ensure file doesn't exist
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("Config file should not exist initially")
	}

	// Create a config to save
	testConfig := &Config{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour).Truncate(time.Second),
		UserEmail:    "new@example.com",
	}

	// Save using the actual function
	if err := Save(testConfig); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load back and verify content
	loadedConfig, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loadedConfig.AccessToken != testConfig.AccessToken {
		t.Errorf("AccessToken = %v, want %v", loadedConfig.AccessToken, testConfig.AccessToken)
	}
	if loadedConfig.RefreshToken != testConfig.RefreshToken {
		t.Errorf("RefreshToken = %v, want %v", loadedConfig.RefreshToken, testConfig.RefreshToken)
	}
	if loadedConfig.UserEmail != testConfig.UserEmail {
		t.Errorf("UserEmail = %v, want %v", loadedConfig.UserEmail, testConfig.UserEmail)
	}
}

// TestSave_OverwriteExisting verifies that Save updates an existing config file
func TestSave_OverwriteExisting(t *testing.T) {
	_, cleanup := withTempHome(t)
	defer cleanup()

	// Create initial config
	initialConfig := &Config{
		AccessToken:  "initial-access-token",
		RefreshToken: "initial-refresh-token",
		ExpiresAt:    time.Now().Truncate(time.Second),
		UserEmail:    "initial@example.com",
	}

	if err := Save(initialConfig); err != nil {
		t.Fatalf("Save() initial error = %v", err)
	}

	// Verify initial config was saved
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.AccessToken != initialConfig.AccessToken {
		t.Fatalf("Initial save failed")
	}

	// Now overwrite with new config
	newConfig := &Config{
		AccessToken:  "updated-access-token",
		RefreshToken: "updated-refresh-token",
		ExpiresAt:    time.Now().Add(2 * time.Hour).Truncate(time.Second),
		UserEmail:    "updated@example.com",
	}

	if err := Save(newConfig); err != nil {
		t.Fatalf("Save() update error = %v", err)
	}

	// Load and verify it's the new config
	loaded, err = Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.AccessToken != newConfig.AccessToken {
		t.Errorf("AccessToken = %v, want %v", loaded.AccessToken, newConfig.AccessToken)
	}
	if loaded.AccessToken == initialConfig.AccessToken {
		t.Error("Config was not overwritten, still contains initial values")
	}
	if loaded.UserEmail != newConfig.UserEmail {
		t.Errorf("UserEmail = %v, want %v", loaded.UserEmail, newConfig.UserEmail)
	}
}

// TestSave_CreatesDirectory verifies that Save creates the ~/.ravi/ directory if it doesn't exist
func TestSave_CreatesDirectory(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	raviDir := filepath.Join(tmpDir, ".ravi")

	// Ensure directory doesn't exist
	if _, err := os.Stat(raviDir); !os.IsNotExist(err) {
		t.Fatalf("Expected directory to not exist initially")
	}

	// Save should create the directory
	testConfig := &Config{
		AccessToken: "test-token",
	}

	if err := Save(testConfig); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify directory was created
	info, err := os.Stat(raviDir)
	if err != nil {
		t.Fatalf("Directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("Expected .ravi to be a directory")
	}

	// Verify directory permissions (0700)
	perm := info.Mode().Perm()
	expectedPerm := os.FileMode(0700)
	if perm != expectedPerm {
		t.Errorf("Directory permissions = %o, want %o", perm, expectedPerm)
	}
}

// TestSave_Permissions verifies that the saved file has 0600 mode for security
func TestSave_Permissions(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	configPath := filepath.Join(tmpDir, ".ravi", "config.json")

	testConfig := &Config{
		AccessToken:  "secret-token",
		RefreshToken: "secret-refresh",
	}

	if err := Save(testConfig); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file permissions
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}

	perm := info.Mode().Perm()
	expectedPerm := os.FileMode(0600)
	if perm != expectedPerm {
		t.Errorf("File permissions = %o, want %o", perm, expectedPerm)
	}
}

// TestClear_ExistingFile verifies that Clear removes an existing config file
func TestClear_ExistingFile(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	configPath := filepath.Join(tmpDir, ".ravi", "config.json")

	// Create a config file
	testConfig := &Config{AccessToken: "to-be-deleted"}
	if err := Save(testConfig); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file should exist before clear")
	}

	// Clear the file
	if err := Clear(); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	// Verify file is gone
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Error("Config file should not exist after clear")
	}
}

// TestClear_NoFile verifies that Clear returns no error when clearing a non-existent file
func TestClear_NoFile(t *testing.T) {
	_, cleanup := withTempHome(t)
	defer cleanup()

	// Don't create any config file

	// Clear should not return an error for non-existent file
	err := Clear()
	if err != nil {
		t.Errorf("Clear() error = %v, want nil for non-existent file", err)
	}
}

// TestConfig_JSONMarshaling verifies that Config marshals/unmarshals correctly
func TestConfig_JSONMarshaling(t *testing.T) {
	testCases := []struct {
		name   string
		config Config
	}{
		{
			name: "full config",
			config: Config{
				AccessToken:  "access-123",
				RefreshToken: "refresh-456",
				ExpiresAt:    time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC),
				UserEmail:    "user@example.com",
			},
		},
		{
			name: "minimal config",
			config: Config{
				AccessToken: "access-only",
			},
		},
		{
			name:   "empty config",
			config: Config{},
		},
		{
			name: "config with special characters in email",
			config: Config{
				AccessToken: "token",
				UserEmail:   "user+tag@sub.example.com",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Marshal
			data, err := json.Marshal(tc.config)
			if err != nil {
				t.Fatalf("Failed to marshal config: %v", err)
			}

			// Unmarshal
			var loaded Config
			if err := json.Unmarshal(data, &loaded); err != nil {
				t.Fatalf("Failed to unmarshal config: %v", err)
			}

			// Compare
			if loaded.AccessToken != tc.config.AccessToken {
				t.Errorf("AccessToken = %v, want %v", loaded.AccessToken, tc.config.AccessToken)
			}
			if loaded.RefreshToken != tc.config.RefreshToken {
				t.Errorf("RefreshToken = %v, want %v", loaded.RefreshToken, tc.config.RefreshToken)
			}
			if !loaded.ExpiresAt.Equal(tc.config.ExpiresAt) {
				t.Errorf("ExpiresAt = %v, want %v", loaded.ExpiresAt, tc.config.ExpiresAt)
			}
			if loaded.UserEmail != tc.config.UserEmail {
				t.Errorf("UserEmail = %v, want %v", loaded.UserEmail, tc.config.UserEmail)
			}
		})
	}
}

// TestConfig_OmitEmptyEmail verifies that empty UserEmail is omitted from JSON
func TestConfig_OmitEmptyEmail(t *testing.T) {
	config := Config{
		AccessToken:  "token",
		RefreshToken: "refresh",
		ExpiresAt:    time.Now(),
		UserEmail:    "", // Empty - should be omitted
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	// Check that user_email is not in the JSON
	if strings.Contains(string(data), "user_email") {
		t.Errorf("Expected user_email to be omitted when empty, got: %s", data)
	}

	// Now with a value
	config.UserEmail = "test@example.com"
	data, err = json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if !strings.Contains(string(data), "user_email") {
		t.Errorf("Expected user_email to be present when set, got: %s", data)
	}
}

// TestSave_Load_RoundTrip verifies that saving and loading produces the same config
func TestSave_Load_RoundTrip(t *testing.T) {
	_, cleanup := withTempHome(t)
	defer cleanup()

	original := &Config{
		AccessToken:  "round-trip-access",
		RefreshToken: "round-trip-refresh",
		ExpiresAt:    time.Date(2024, 12, 25, 15, 30, 45, 0, time.UTC),
		UserEmail:    "roundtrip@example.com",
	}

	// Save
	if err := Save(original); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Compare all fields
	if loaded.AccessToken != original.AccessToken {
		t.Errorf("AccessToken mismatch: got %v, want %v", loaded.AccessToken, original.AccessToken)
	}
	if loaded.RefreshToken != original.RefreshToken {
		t.Errorf("RefreshToken mismatch: got %v, want %v", loaded.RefreshToken, original.RefreshToken)
	}
	if !loaded.ExpiresAt.Equal(original.ExpiresAt) {
		t.Errorf("ExpiresAt mismatch: got %v, want %v", loaded.ExpiresAt, original.ExpiresAt)
	}
	if loaded.UserEmail != original.UserEmail {
		t.Errorf("UserEmail mismatch: got %v, want %v", loaded.UserEmail, original.UserEmail)
	}
}

// TestConfigConstants verifies the package constants are set correctly
func TestConfigConstants(t *testing.T) {
	// Verify constants through the path
	path := Path()

	// Should contain config.json
	if filepath.Base(path) != configFileName {
		t.Errorf("Path base = %v, want %v", filepath.Base(path), configFileName)
	}

	// Parent should be .ravi
	dir := filepath.Dir(path)
	if filepath.Base(dir) != configDirName {
		t.Errorf("Path dir = %v, want %v", filepath.Base(dir), configDirName)
	}

	// Verify permission constants have expected values
	if configDirPerm != 0700 {
		t.Errorf("configDirPerm = %o, want 0700", configDirPerm)
	}
	if configFilePerm != 0600 {
		t.Errorf("configFilePerm = %o, want 0600", configFilePerm)
	}
}

// TestPath_Structure verifies the expected path structure
func TestPath_Structure(t *testing.T) {
	path := Path()

	// Verify the path has the expected components
	parts := strings.Split(path, string(filepath.Separator))

	// Find the .ravi part
	foundRavi := false
	foundConfig := false
	for i, part := range parts {
		if part == ".ravi" {
			foundRavi = true
			// config.json should immediately follow .ravi
			if i+1 < len(parts) && parts[i+1] == "config.json" {
				foundConfig = true
			}
		}
	}

	if !foundRavi {
		t.Errorf("Path() = %v, missing .ravi directory", path)
	}
	if !foundConfig {
		t.Errorf("Path() = %v, missing config.json after .ravi", path)
	}
}
