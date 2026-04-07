package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// withTempHome temporarily changes HOME to a temp directory.
func withTempHome(t *testing.T) (tmpDir string, cleanup func()) {
	t.Helper()

	tmpDir = t.TempDir()

	var homeEnvVar string
	if runtime.GOOS == "windows" {
		homeEnvVar = "USERPROFILE"
	} else {
		homeEnvVar = "HOME"
	}
	originalHome := os.Getenv(homeEnvVar)

	if err := os.Setenv(homeEnvVar, tmpDir); err != nil {
		t.Fatalf("Failed to set %s: %v", homeEnvVar, err)
	}

	cleanup = func() {
		os.Setenv(homeEnvVar, originalHome)
	}

	return tmpDir, cleanup
}

func TestLoadConfig_Default(t *testing.T) {
	_, cleanup := withTempHome(t)
	defer cleanup()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.IdentityUUID != "" {
		t.Errorf("IdentityUUID = %v, want empty", cfg.IdentityUUID)
	}
	if cfg.ManagementKey != "" {
		t.Errorf("ManagementKey = %v, want empty", cfg.ManagementKey)
	}
}

func TestSaveGlobalConfig_LoadConfig(t *testing.T) {
	_, cleanup := withTempHome(t)
	defer cleanup()

	if err := SaveGlobalConfig(&Config{
		ManagementKey: "ravi_mgmt_test123",
		IdentityKey:   "ravi_id_test456",
		IdentityUUID:  "uuid-123",
		IdentityName:  "Research",
		UserEmail:     "user@example.com",
	}); err != nil {
		t.Fatalf("SaveGlobalConfig() error = %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.ManagementKey != "ravi_mgmt_test123" {
		t.Errorf("ManagementKey = %v, want 'ravi_mgmt_test123'", cfg.ManagementKey)
	}
	if cfg.IdentityKey != "ravi_id_test456" {
		t.Errorf("IdentityKey = %v, want 'ravi_id_test456'", cfg.IdentityKey)
	}
	if cfg.IdentityUUID != "uuid-123" {
		t.Errorf("IdentityUUID = %v, want 'uuid-123'", cfg.IdentityUUID)
	}
	if cfg.IdentityName != "Research" {
		t.Errorf("IdentityName = %v, want 'Research'", cfg.IdentityName)
	}
	if cfg.UserEmail != "user@example.com" {
		t.Errorf("UserEmail = %v, want 'user@example.com'", cfg.UserEmail)
	}
}

func TestLoadConfig_CWDOverridesGlobal(t *testing.T) {
	_, cleanup := withTempHome(t)
	defer cleanup()

	// Set global config.
	if err := SaveGlobalConfig(&Config{IdentityUUID: "global-uuid", IdentityName: "Global"}); err != nil {
		t.Fatalf("SaveGlobalConfig() error = %v", err)
	}

	// Create a CWD override.
	tmpProject := t.TempDir()
	localDir := filepath.Join(tmpProject, ".ravi")
	os.MkdirAll(localDir, 0700)
	data, _ := json.Marshal(Config{IdentityUUID: "project-uuid", IdentityName: "Project"})
	os.WriteFile(filepath.Join(localDir, "config.json"), data, 0600)

	// Change CWD.
	origWD, _ := os.Getwd()
	os.Chdir(tmpProject)
	defer os.Chdir(origWD)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.IdentityUUID != "project-uuid" {
		t.Errorf("IdentityUUID = %v, want 'project-uuid'", cfg.IdentityUUID)
	}
	if cfg.IdentityName != "Project" {
		t.Errorf("IdentityName = %v, want 'Project'", cfg.IdentityName)
	}
}

func TestLoadConfig_EmptyJSON(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	// Write a config.json with empty JSON.
	raviDir := filepath.Join(tmpDir, ".ravi")
	os.MkdirAll(raviDir, 0700)
	os.WriteFile(filepath.Join(raviDir, "config.json"), []byte(`{}`), 0600)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.IdentityUUID != "" {
		t.Errorf("IdentityUUID = %v, want empty", cfg.IdentityUUID)
	}
}

func TestLoadConfig_CorruptFile(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	raviDir := filepath.Join(tmpDir, ".ravi")
	os.MkdirAll(raviDir, 0700)
	os.WriteFile(filepath.Join(raviDir, "config.json"), []byte("not json"), 0600)

	_, err := LoadConfig()
	if err == nil {
		t.Error("LoadConfig() error = nil, want error for corrupt file")
	}
}

// --- SaveConfig tests ---

func TestSaveConfig_WritesCWD_WhenLocalExists(t *testing.T) {
	tmpHome, cleanup := withTempHome(t)
	defer cleanup()

	// Set up a project directory with an existing .ravi/config.json.
	tmpProject := t.TempDir()
	localDir := filepath.Join(tmpProject, ".ravi")
	os.MkdirAll(localDir, 0700)
	os.WriteFile(filepath.Join(localDir, "config.json"), []byte(`{}`), 0600)

	origWD, _ := os.Getwd()
	os.Chdir(tmpProject)
	defer os.Chdir(origWD)

	cfg := &Config{
		ManagementKey: "ravi_mgmt_local",
		IdentityKey:   "ravi_id_local",
		IdentityUUID:  "local-uuid",
		IdentityName:  "Local",
	}
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Verify it was written to the CWD path.
	data, err := os.ReadFile(filepath.Join(localDir, "config.json"))
	if err != nil {
		t.Fatalf("ReadFile(local config) error = %v", err)
	}
	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}
	if loaded.IdentityUUID != "local-uuid" {
		t.Errorf("IdentityUUID = %v, want 'local-uuid'", loaded.IdentityUUID)
	}

	// Verify nothing was written to the global path.
	globalPath := filepath.Join(tmpHome, ".ravi", "config.json")
	if _, err := os.Stat(globalPath); !os.IsNotExist(err) {
		t.Error("Expected global config.json to NOT exist when CWD config was present")
	}
}

func TestSaveConfig_WritesGlobal_WhenNoCWDConfig(t *testing.T) {
	tmpHome, cleanup := withTempHome(t)
	defer cleanup()

	// Use a project directory WITHOUT .ravi/config.json.
	tmpProject := t.TempDir()
	origWD, _ := os.Getwd()
	os.Chdir(tmpProject)
	defer os.Chdir(origWD)

	cfg := &Config{
		ManagementKey: "ravi_mgmt_global",
		IdentityKey:   "ravi_id_global",
		IdentityUUID:  "global-uuid",
		IdentityName:  "Global",
	}
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Verify it was written to the global path.
	globalPath := filepath.Join(tmpHome, ".ravi", "config.json")
	data, err := os.ReadFile(globalPath)
	if err != nil {
		t.Fatalf("ReadFile(global config) error = %v", err)
	}
	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}
	if loaded.IdentityUUID != "global-uuid" {
		t.Errorf("IdentityUUID = %v, want 'global-uuid'", loaded.IdentityUUID)
	}
}

func TestSaveConfig_RoundTrip(t *testing.T) {
	_, cleanup := withTempHome(t)
	defer cleanup()

	original := &Config{
		ManagementKey: "ravi_mgmt_rt",
		IdentityKey:   "ravi_id_rt",
		IdentityUUID:  "rt-uuid",
		IdentityName:  "RoundTrip",
		UserEmail:     "rt@example.com",
	}

	// Save via SaveConfig (falls to global since no CWD config).
	if err := SaveConfig(original); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Load it back via LoadConfig.
	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if loaded.ManagementKey != original.ManagementKey {
		t.Errorf("ManagementKey = %v, want %v", loaded.ManagementKey, original.ManagementKey)
	}
	if loaded.IdentityKey != original.IdentityKey {
		t.Errorf("IdentityKey = %v, want %v", loaded.IdentityKey, original.IdentityKey)
	}
	if loaded.IdentityUUID != original.IdentityUUID {
		t.Errorf("IdentityUUID = %v, want %v", loaded.IdentityUUID, original.IdentityUUID)
	}
	if loaded.IdentityName != original.IdentityName {
		t.Errorf("IdentityName = %v, want %v", loaded.IdentityName, original.IdentityName)
	}
	if loaded.UserEmail != original.UserEmail {
		t.Errorf("UserEmail = %v, want %v", loaded.UserEmail, original.UserEmail)
	}
}

// --- Cleanup ---

func TestClearAll(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	// Create some state.
	SaveGlobalConfig(&Config{ManagementKey: "key", IdentityUUID: "uuid-123", IdentityName: "Test"})

	raviDir := filepath.Join(tmpDir, ".ravi")
	if _, err := os.Stat(raviDir); os.IsNotExist(err) {
		t.Fatal("Expected .ravi to exist before ClearAll")
	}

	if err := ClearAll(); err != nil {
		t.Fatalf("ClearAll() error = %v", err)
	}

	if _, err := os.Stat(raviDir); !os.IsNotExist(err) {
		t.Error("Expected .ravi to be removed after ClearAll")
	}
}

// --- Dir/path tests ---

func TestDir(t *testing.T) {
	dir := Dir()
	if !strings.HasSuffix(dir, ".ravi") {
		t.Errorf("Dir() = %v, want suffix .ravi", dir)
	}
}

// --- Directory permissions ---

func TestSaveGlobalConfig_DirectoryPermissions(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	SaveGlobalConfig(&Config{ManagementKey: "test"})

	raviDir := filepath.Join(tmpDir, ".ravi")
	info, err := os.Stat(raviDir)
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	if info.Mode().Perm() != 0700 {
		t.Errorf("Directory permissions = %o, want 0700", info.Mode().Perm())
	}
}

func TestSaveGlobalConfig_FilePermissions(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	SaveGlobalConfig(&Config{ManagementKey: "test"})

	configPath := filepath.Join(tmpDir, ".ravi", "config.json")
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("File permissions = %o, want 0600", info.Mode().Perm())
	}
}
