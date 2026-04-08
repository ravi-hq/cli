package cli

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ravi-hq/cli/internal/api"
)

// TestSSOCmdRegistration verifies that the sso command and its subcommands are registered.
func TestSSOCmdRegistration(t *testing.T) {
	// Verify sso is registered on rootCmd.
	subNames := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		subNames[cmd.Name()] = true
	}

	if !subNames["sso"] {
		t.Error("rootCmd missing subcommand 'sso'")
	}

	// Verify sso token is registered on ssoCmd.
	ssoSubNames := make(map[string]bool)
	for _, cmd := range ssoCmd.Commands() {
		ssoSubNames[cmd.Name()] = true
	}

	if !ssoSubNames["token"] {
		t.Error("ssoCmd missing subcommand 'token'")
	}
}

// TestSSOTokenCmd_Success verifies that ssoTokenCmd calls the API and prints the token.
func TestSSOTokenCmd_Success(t *testing.T) {
	server, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != api.PathSSOToken {
			t.Errorf("Expected path %s, got %s", api.PathSSOToken, r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(api.SSOTokenResponse{
			Token:     "rvt_test_abc123",
			ExpiresAt: "2026-04-07T12:05:00Z",
		})
	}))
	_ = server
	defer cleanup()

	err := ssoTokenCmd.RunE(ssoTokenCmd, nil)
	if err != nil {
		t.Fatalf("ssoTokenCmd.RunE() error = %v", err)
	}
}

// TestSSOTokenCmd_APIError verifies that ssoTokenCmd propagates API errors.
func TestSSOTokenCmd_APIError(t *testing.T) {
	server, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPaymentRequired)
		json.NewEncoder(w).Encode(map[string]string{"detail": "Active subscription required."})
	}))
	_ = server
	defer cleanup()

	err := ssoTokenCmd.RunE(ssoTokenCmd, nil)
	if err == nil {
		t.Fatal("ssoTokenCmd.RunE() expected error for 402, got nil")
	}
}
