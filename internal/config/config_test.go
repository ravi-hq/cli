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

// --- AuthConfig tests ---

func TestLoadAuth_NoFile(t *testing.T) {
	_, cleanup := withTempHome(t)
	defer cleanup()

	cfg, err := LoadAuth()
	if err != nil {
		t.Fatalf("LoadAuth() error = %v, want nil", err)
	}
	if cfg.AccessToken != "" {
		t.Errorf("AccessToken = %v, want empty", cfg.AccessToken)
	}
}

func TestSaveAuth_CreatesFile(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	cfg := &AuthConfig{
		AccessToken:  "test-access",
		RefreshToken: "test-refresh",
		ExpiresAt:    time.Now().Add(time.Hour).Truncate(time.Second),
		UserEmail:    "test@example.com",
		PINSalt:      "salt123",
		PublicKey:    "pub123",
		PrivateKey:   "priv123",
	}

	if err := SaveAuth(cfg); err != nil {
		t.Fatalf("SaveAuth() error = %v", err)
	}

	// Verify file exists at auth.json (not config.json).
	authPath := filepath.Join(tmpDir, ".ravi", "auth.json")
	if _, err := os.Stat(authPath); os.IsNotExist(err) {
		t.Fatal("auth.json was not created")
	}

	// Verify permissions.
	info, err := os.Stat(authPath)
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("File permissions = %o, want 0600", info.Mode().Perm())
	}
}

func TestSaveAuth_LoadAuth_RoundTrip(t *testing.T) {
	_, cleanup := withTempHome(t)
	defer cleanup()

	original := &AuthConfig{
		AccessToken:  "rt-access",
		RefreshToken: "rt-refresh",
		ExpiresAt:    time.Date(2024, 12, 25, 15, 30, 45, 0, time.UTC),
		UserEmail:    "rt@example.com",
		PINSalt:      "salt",
		PublicKey:    "pub",
		PrivateKey:   "priv",
	}

	if err := SaveAuth(original); err != nil {
		t.Fatalf("SaveAuth() error = %v", err)
	}

	loaded, err := LoadAuth()
	if err != nil {
		t.Fatalf("LoadAuth() error = %v", err)
	}

	if loaded.AccessToken != original.AccessToken {
		t.Errorf("AccessToken = %v, want %v", loaded.AccessToken, original.AccessToken)
	}
	if loaded.RefreshToken != original.RefreshToken {
		t.Errorf("RefreshToken = %v, want %v", loaded.RefreshToken, original.RefreshToken)
	}
	if !loaded.ExpiresAt.Equal(original.ExpiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", loaded.ExpiresAt, original.ExpiresAt)
	}
	if loaded.UserEmail != original.UserEmail {
		t.Errorf("UserEmail = %v, want %v", loaded.UserEmail, original.UserEmail)
	}
	if loaded.PINSalt != original.PINSalt {
		t.Errorf("PINSalt = %v, want %v", loaded.PINSalt, original.PINSalt)
	}
	if loaded.PublicKey != original.PublicKey {
		t.Errorf("PublicKey = %v, want %v", loaded.PublicKey, original.PublicKey)
	}
	if loaded.PrivateKey != original.PrivateKey {
		t.Errorf("PrivateKey = %v, want %v", loaded.PrivateKey, original.PrivateKey)
	}
}

func TestLoadAuth_CorruptFile(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	raviDir := filepath.Join(tmpDir, ".ravi")
	os.MkdirAll(raviDir, 0700)
	os.WriteFile(filepath.Join(raviDir, "auth.json"), []byte("not json"), 0600)

	_, err := LoadAuth()
	if err == nil {
		t.Error("LoadAuth() error = nil, want error for corrupt file")
	}
	if !strings.Contains(err.Error(), "parsing") {
		t.Errorf("Error = %v, want 'parsing' in message", err)
	}
}

// --- Config (identity selector) tests ---

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
}

func TestSaveGlobalConfig_LoadConfig(t *testing.T) {
	_, cleanup := withTempHome(t)
	defer cleanup()

	if err := SaveGlobalConfig(&Config{IdentityUUID: "uuid-123", IdentityName: "Research"}); err != nil {
		t.Fatalf("SaveGlobalConfig() error = %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.IdentityUUID != "uuid-123" {
		t.Errorf("IdentityUUID = %v, want 'uuid-123'", cfg.IdentityUUID)
	}
	if cfg.IdentityName != "Research" {
		t.Errorf("IdentityName = %v, want 'Research'", cfg.IdentityName)
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

// --- Resolution helpers ---

func TestResolveIdentityUUID_NoConfig(t *testing.T) {
	_, cleanup := withTempHome(t)
	defer cleanup()

	uuid, err := ResolveIdentityUUID()
	if err != nil {
		t.Fatalf("ResolveIdentityUUID() error = %v", err)
	}
	if uuid != "" {
		t.Errorf("ResolveIdentityUUID() = %v, want empty", uuid)
	}
}

func TestResolveIdentityUUID_WithConfig(t *testing.T) {
	_, cleanup := withTempHome(t)
	defer cleanup()

	SaveGlobalConfig(&Config{IdentityUUID: "uuid-123"})

	uuid, err := ResolveIdentityUUID()
	if err != nil {
		t.Fatalf("ResolveIdentityUUID() error = %v", err)
	}
	if uuid != "uuid-123" {
		t.Errorf("ResolveIdentityUUID() = %v, want 'uuid-123'", uuid)
	}
}

// --- Cleanup ---

func TestClearAll(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	// Create some state.
	SaveAuth(&AuthConfig{AccessToken: "token"})
	SaveGlobalConfig(&Config{IdentityUUID: "uuid-123", IdentityName: "Test"})

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

// --- Recovery key ---

func TestSaveRecoveryKey(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	if err := SaveRecoveryKey("base64-seed-data"); err != nil {
		t.Fatalf("SaveRecoveryKey() error = %v", err)
	}

	path := filepath.Join(tmpDir, ".ravi", "recovery-key.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if strings.TrimSpace(string(data)) != "base64-seed-data" {
		t.Errorf("Recovery key = %q, want 'base64-seed-data'", strings.TrimSpace(string(data)))
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("File permissions = %o, want 0600", info.Mode().Perm())
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

func TestSaveAuth_DirectoryPermissions(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	SaveAuth(&AuthConfig{AccessToken: "test"})

	raviDir := filepath.Join(tmpDir, ".ravi")
	info, err := os.Stat(raviDir)
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	if info.Mode().Perm() != 0700 {
		t.Errorf("Directory permissions = %o, want 0700", info.Mode().Perm())
	}
}
