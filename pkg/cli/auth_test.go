package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ravi-hq/cli/internal/config"
	"github.com/ravi-hq/cli/internal/output"
	"github.com/spf13/cobra"
)

// withTempHome is a test helper that temporarily changes the HOME environment variable.
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

// newTestAuthStatusCmd creates a fresh auth status command for testing.
func newTestAuthStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}

			isAuthenticated := cfg.ManagementKey != "" || cfg.IdentityKey != ""

			if isAuthenticated {
				result := map[string]interface{}{
					"authenticated": true,
				}
				if cfg.UserEmail != "" {
					result["email"] = cfg.UserEmail
				}
				data, _ := json.MarshalIndent(result, "", "  ")
				cmd.Println(string(data))
			} else {
				result := map[string]interface{}{
					"authenticated": false,
				}
				data, _ := json.MarshalIndent(result, "", "  ")
				cmd.Println(string(data))
			}
			return nil
		},
	}
	return cmd
}

// newTestAuthLogoutCmd creates a fresh auth logout command for testing.
func newTestAuthLogoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Clear stored credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.ClearAll(); err != nil {
				return err
			}
			cmd.Println("Logged out successfully")
			return nil
		},
	}
	return cmd
}

// TestAuthStatus_NotAuthenticated verifies that the status command shows
// authenticated: false when there are no stored credentials.
func TestAuthStatus_NotAuthenticated(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	raviDir := config.Dir()
	if !strings.HasPrefix(raviDir, tmpDir) {
		t.Fatalf("config.Dir() = %v, expected prefix %v", raviDir, tmpDir)
	}

	cmd := newTestAuthStatusCmd()

	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	outputStr := stdout.String()

	if !strings.Contains(outputStr, "authenticated") {
		t.Errorf("Status output should contain 'authenticated', got:\n%s", outputStr)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(outputStr), &result); err != nil {
		t.Fatalf("Failed to parse output as JSON: %v\nOutput was: %s", err, outputStr)
	}

	authenticated, ok := result["authenticated"].(bool)
	if !ok {
		t.Errorf("Expected 'authenticated' to be a boolean, got: %T", result["authenticated"])
	}
	if authenticated {
		t.Errorf("Expected authenticated=false when no credentials exist, got true")
	}

	if _, hasEmail := result["email"]; hasEmail {
		t.Error("Expected no email field when not authenticated")
	}
}

// TestAuthStatus_Authenticated verifies that the status command shows
// the user's email and authenticated status when valid credentials exist.
func TestAuthStatus_Authenticated(t *testing.T) {
	_, cleanup := withTempHome(t)
	defer cleanup()

	testConfig := &config.Config{
		ManagementKey: "ravi_mgmt_test123",
		IdentityKey:   "ravi_id_test456",
		UserEmail:     "user@example.com",
	}

	if err := config.SaveGlobalConfig(testConfig); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	cmd := newTestAuthStatusCmd()

	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	outputStr := stdout.String()

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(outputStr), &result); err != nil {
		t.Fatalf("Failed to parse output as JSON: %v\nOutput was: %s", err, outputStr)
	}

	authenticated, ok := result["authenticated"].(bool)
	if !ok {
		t.Errorf("Expected 'authenticated' to be a boolean, got: %T", result["authenticated"])
	}
	if !authenticated {
		t.Errorf("Expected authenticated=true when credentials exist, got false")
	}

	email, ok := result["email"].(string)
	if !ok {
		t.Errorf("Expected 'email' to be a string, got: %T", result["email"])
	}
	if email != "user@example.com" {
		t.Errorf("Expected email='user@example.com', got %q", email)
	}
}

// TestAuthLogout_ClearsConfig verifies that the logout command removes stored credentials.
func TestAuthLogout_ClearsConfig(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	originalFormatter := output.Current
	defer func() { output.Current = originalFormatter }()

	raviDir := filepath.Join(tmpDir, ".ravi")
	if err := os.MkdirAll(raviDir, 0700); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	testConfig := &config.Config{
		ManagementKey: "ravi_mgmt_test123",
		IdentityKey:   "ravi_id_test456",
		UserEmail:     "user@example.com",
	}

	if err := config.SaveGlobalConfig(testConfig); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	configPath := filepath.Join(raviDir, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file should exist before logout")
	}

	cmd := newTestAuthLogoutCmd()

	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	outputStr := stdout.String()

	if !strings.Contains(outputStr, "Logged out") || !strings.Contains(outputStr, "successfully") {
		t.Errorf("Expected logout success message, got:\n%s", outputStr)
	}

	if _, err := os.Stat(raviDir); !os.IsNotExist(err) {
		t.Error("Ravi directory should not exist after logout")
	}

	// Verify that loading config now returns empty
	loadedConfig, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned error: %v", err)
	}
	if loadedConfig.ManagementKey != "" {
		t.Errorf("Expected empty ManagementKey after logout, got %q", loadedConfig.ManagementKey)
	}
	if loadedConfig.IdentityKey != "" {
		t.Errorf("Expected empty IdentityKey after logout, got %q", loadedConfig.IdentityKey)
	}
}
