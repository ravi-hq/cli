package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ravi-hq/cli/internal/config"
	"github.com/ravi-hq/cli/internal/output"
	"github.com/spf13/cobra"
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

// newTestAuthStatusCmd creates a fresh auth status command for testing.
func newTestAuthStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			auth, err := config.LoadAuth()
			if err != nil {
				return err
			}

			// Check if authenticated (has both access and refresh tokens)
			isAuthenticated := auth.AccessToken != "" && auth.RefreshToken != ""

			if isAuthenticated {
				if auth.UserEmail != "" {
					result := map[string]interface{}{
						"authenticated": true,
						"email":         auth.UserEmail,
					}
					data, _ := json.MarshalIndent(result, "", "  ")
					cmd.Println(string(data))
				} else {
					result := map[string]interface{}{
						"authenticated": true,
					}
					data, _ := json.MarshalIndent(result, "", "  ")
					cmd.Println(string(data))
				}
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
// "not logged in" / authenticated: false when there are no stored credentials.
func TestAuthStatus_NotAuthenticated(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	// Verify config dir is now in temp directory
	raviDir := config.Dir()
	if !strings.HasPrefix(raviDir, tmpDir) {
		t.Fatalf("config.Dir() = %v, expected prefix %v", raviDir, tmpDir)
	}

	// No config file exists, so user is not authenticated

	cmd := newTestAuthStatusCmd()

	// Capture output
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	output := stdout.String()

	// Verify output indicates not authenticated
	if !strings.Contains(output, "authenticated") {
		t.Errorf("Status output should contain 'authenticated', got:\n%s", output)
	}

	// Parse the JSON output to verify the value
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse output as JSON: %v\nOutput was: %s", err, output)
	}

	authenticated, ok := result["authenticated"].(bool)
	if !ok {
		t.Errorf("Expected 'authenticated' to be a boolean, got: %T", result["authenticated"])
	}
	if authenticated {
		t.Errorf("Expected authenticated=false when no credentials exist, got true")
	}

	// Email should not be present
	if _, hasEmail := result["email"]; hasEmail {
		t.Error("Expected no email field when not authenticated")
	}
}

// TestAuthStatus_Authenticated verifies that the status command shows
// the user's email and authenticated status when valid credentials exist.
func TestAuthStatus_Authenticated(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	// Create a config file with valid credentials
	raviDir := filepath.Join(tmpDir, ".ravi")
	if err := os.MkdirAll(raviDir, 0700); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	testConfig := &config.AuthConfig{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
		UserEmail:    "user@example.com",
	}

	if err := config.SaveAuth(testConfig); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	cmd := newTestAuthStatusCmd()

	// Capture output
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	outputStr := stdout.String()

	// Parse the JSON output
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(outputStr), &result); err != nil {
		t.Fatalf("Failed to parse output as JSON: %v\nOutput was: %s", err, outputStr)
	}

	// Verify authenticated is true
	authenticated, ok := result["authenticated"].(bool)
	if !ok {
		t.Errorf("Expected 'authenticated' to be a boolean, got: %T", result["authenticated"])
	}
	if !authenticated {
		t.Errorf("Expected authenticated=true when credentials exist, got false")
	}

	// Verify email is present and correct
	email, ok := result["email"].(string)
	if !ok {
		t.Errorf("Expected 'email' to be a string, got: %T", result["email"])
	}
	if email != "user@example.com" {
		t.Errorf("Expected email='user@example.com', got %q", email)
	}
}

// TestAuthLogout_ClearsConfig verifies that the logout command removes
// the stored credentials from the config file.
func TestAuthLogout_ClearsConfig(t *testing.T) {
	tmpDir, cleanup := withTempHome(t)
	defer cleanup()

	// Save the original output formatter and restore after test
	originalFormatter := output.Current
	defer func() { output.Current = originalFormatter }()

	// Create a config file with credentials
	raviDir := filepath.Join(tmpDir, ".ravi")
	authPath := filepath.Join(raviDir, "auth.json")
	if err := os.MkdirAll(raviDir, 0700); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	testConfig := &config.AuthConfig{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
		UserEmail:    "user@example.com",
	}

	if err := config.SaveAuth(testConfig); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify auth file exists
	if _, err := os.Stat(authPath); os.IsNotExist(err) {
		t.Fatal("Auth file should exist before logout")
	}

	cmd := newTestAuthLogoutCmd()

	// Capture output
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	outputStr := stdout.String()

	// Verify success message
	if !strings.Contains(outputStr, "Logged out") || !strings.Contains(outputStr, "successfully") {
		t.Errorf("Expected logout success message, got:\n%s", outputStr)
	}

	// Verify ravi directory is removed
	if _, err := os.Stat(raviDir); !os.IsNotExist(err) {
		t.Error("Ravi directory should not exist after logout")
	}

	// Verify that loading auth now returns empty credentials
	loadedAuth, err := config.LoadAuth()
	if err != nil {
		t.Fatalf("LoadAuth() returned error: %v", err)
	}
	if loadedAuth.AccessToken != "" {
		t.Errorf("Expected empty AccessToken after logout, got %q", loadedAuth.AccessToken)
	}
	if loadedAuth.RefreshToken != "" {
		t.Errorf("Expected empty RefreshToken after logout, got %q", loadedAuth.RefreshToken)
	}
}
