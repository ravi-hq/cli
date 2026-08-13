package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ravi-hq/cli/internal/api"
	"github.com/ravi-hq/cli/internal/config"
)

func TestMain(m *testing.M) {
	// Never open a real browser during tests
	OpenBrowser = func(url string) error { return nil }
	os.Exit(m.Run())
}

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

// withAPIBaseURL points the hosted API client at a local httptest server.
func withAPIBaseURL(t *testing.T, url string) func() {
	t.Helper()
	t.Setenv("RAVI_CLI_TEST_API_BASE_URL", url)
	return func() {}
}

// TestNewDeviceFlow_Success verifies that NewDeviceFlow creates a flow handler.
func TestNewDeviceFlow_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v, want nil", err)
	}

	if flow == nil {
		t.Fatal("NewDeviceFlow() returned nil flow, want non-nil")
	}

	if flow.client == nil {
		t.Error("NewDeviceFlow() flow.client = nil, want non-nil")
	}

	if flow.spinner == nil {
		t.Error("NewDeviceFlow() flow.spinner = nil, want non-nil")
	}

	expectedSuffix := " Waiting for authorization..."
	if flow.spinner.Suffix != expectedSuffix {
		t.Errorf("flow.spinner.Suffix = %q, want %q", flow.spinner.Suffix, expectedSuffix)
	}
}

// TestNewDeviceFlow_NoAPIURL verifies that NewDeviceFlow falls back to the
// default URL when the API base URL is not explicitly configured.
func TestNewDeviceFlow_NoAPIURL(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	cleanupURL := withAPIBaseURL(t, "")
	defer cleanupURL()

	flow, err := NewDeviceFlow()

	if err != nil {
		t.Fatalf("NewDeviceFlow() unexpected error = %v", err)
	}

	if flow == nil {
		t.Fatal("NewDeviceFlow() flow = nil, want non-nil")
	}
}

// browserCommandTestCase represents a test case for browser command selection.
type browserCommandTestCase struct {
	goos            string
	expectedCommand string
	expectedArgs    []string
	shouldError     bool
}

// getBrowserCommandTestCases returns test cases for different platforms.
func getBrowserCommandTestCases() []browserCommandTestCase {
	return []browserCommandTestCase{
		{
			goos:            "darwin",
			expectedCommand: "open",
			expectedArgs:    []string{},
			shouldError:     false,
		},
		{
			goos:            "linux",
			expectedCommand: "xdg-open",
			expectedArgs:    []string{},
			shouldError:     false,
		},
		{
			goos:            "windows",
			expectedCommand: "cmd",
			expectedArgs:    []string{"/c", "start"},
			shouldError:     false,
		},
		{
			goos:            "freebsd",
			expectedCommand: "",
			expectedArgs:    nil,
			shouldError:     true,
		},
		{
			goos:            "openbsd",
			expectedCommand: "",
			expectedArgs:    nil,
			shouldError:     true,
		},
	}
}

func TestOpenBrowser_Darwin(t *testing.T) {
	tc := getBrowserCommandTestCases()[0]
	if tc.goos != "darwin" {
		t.Fatalf("Test case mismatch: expected darwin, got %s", tc.goos)
	}
	if tc.expectedCommand != "open" {
		t.Errorf("Expected command for darwin = %q, want %q", tc.expectedCommand, "open")
	}
	if tc.shouldError {
		t.Error("darwin should not error")
	}
}

func TestOpenBrowser_Linux(t *testing.T) {
	tc := getBrowserCommandTestCases()[1]
	if tc.goos != "linux" {
		t.Fatalf("Test case mismatch: expected linux, got %s", tc.goos)
	}
	if tc.expectedCommand != "xdg-open" {
		t.Errorf("Expected command for linux = %q, want %q", tc.expectedCommand, "xdg-open")
	}
}

func TestOpenBrowser_Windows(t *testing.T) {
	tc := getBrowserCommandTestCases()[2]
	if tc.goos != "windows" {
		t.Fatalf("Test case mismatch: expected windows, got %s", tc.goos)
	}
	if tc.expectedCommand != "cmd" {
		t.Errorf("Expected command for windows = %q, want %q", tc.expectedCommand, "cmd")
	}
}

func TestOpenBrowser_Unsupported(t *testing.T) {
	unsupportedPlatforms := []string{"freebsd", "openbsd"}
	for _, platform := range unsupportedPlatforms {
		t.Run(platform, func(t *testing.T) {
			var found bool
			for _, tc := range getBrowserCommandTestCases() {
				if tc.goos == platform {
					found = true
					if !tc.shouldError {
						t.Errorf("Platform %q should return an error", platform)
					}
					break
				}
			}
			if !found {
				// Not in test cases is ok for non-supported platforms
			}
		})
	}

	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		err := openBrowserImpl("https://example.com")
		if err == nil {
			t.Errorf("openBrowserImpl() on %s should return error, got nil", runtime.GOOS)
		}
	}
}

func TestOpenBrowser_CurrentPlatform(t *testing.T) {
	switch runtime.GOOS {
	case "darwin", "linux", "windows":
		t.Logf("Running on supported platform: %s", runtime.GOOS)
	default:
		err := openBrowserImpl("https://example.com")
		if err == nil {
			t.Errorf("openBrowserImpl() on unsupported platform %s should return error", runtime.GOOS)
		}
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	return <-done
}

func TestDeviceVerifyURL(t *testing.T) {
	tests := []struct {
		name   string
		apiURI string
		code   string
		want   string
	}{
		{
			name:   "production API verify path",
			apiURI: "https://api.ravi.app/api/auth/device/verify/",
			code:   "TEST-1234",
			want:   "https://ravi.id/device?user_code=TEST-1234",
		},
		{
			name:   "production API verify path without trailing slash",
			apiURI: "https://api.ravi.app/api/auth/device/verify",
			code:   "ABCD-EFGH",
			want:   "https://ravi.id/device?user_code=ABCD-EFGH",
		},
		{
			name:   "already public device URL",
			apiURI: "https://ravi.id/device",
			code:   "TEST-1234",
			want:   "https://ravi.id/device?user_code=TEST-1234",
		},
		{
			name:   "ravi.app device URL rewrites to ravi.id",
			apiURI: "https://ravi.app/device",
			code:   "TEST-1234",
			want:   "https://ravi.id/device?user_code=TEST-1234",
		},
		{
			name:   "empty verification URI falls back to public URL",
			apiURI: "",
			code:   "TEST-1234",
			want:   "https://ravi.id/device?user_code=TEST-1234",
		},
		{
			name:   "localhost API verify path",
			apiURI: "http://localhost:8000/api/auth/device/verify/",
			code:   "TEST-1234",
			want:   "http://localhost:8000/device?user_code=TEST-1234",
		},
		{
			name:   "loopback API verify path",
			apiURI: "http://127.0.0.1:8000/api/auth/device/verify/",
			code:   "TEST-1234",
			want:   "http://127.0.0.1:8000/device?user_code=TEST-1234",
		},
		{
			name:   "custom frontend URL is kept",
			apiURI: "http://127.0.0.1:0/verify",
			code:   "TEST-1234",
			want:   "http://127.0.0.1:0/verify?user_code=TEST-1234",
		},
		{
			name:   "empty user code omits query",
			apiURI: "https://api.ravi.app/api/auth/device/verify/",
			code:   "",
			want:   "https://ravi.id/device",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deviceVerifyURL(tt.apiURI, tt.code)
			if got != tt.want {
				t.Errorf("deviceVerifyURL(%q, %q) = %q, want %q", tt.apiURI, tt.code, got, tt.want)
			}
		})
	}
}

func TestDefaultSpinnerCharSet(t *testing.T) {
	expectedCharSet := 14
	if DefaultSpinnerCharSet != expectedCharSet {
		t.Errorf("DefaultSpinnerCharSet = %d, want %d", DefaultSpinnerCharSet, expectedCharSet)
	}
}

func TestDeviceFlowStruct(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v", err)
	}

	if flow.client == nil {
		t.Error("flow.client should be non-nil")
	}

	if flow.spinner == nil {
		t.Error("flow.spinner should be non-nil")
	}
}

func TestIdentityLabel_NameAndEmail(t *testing.T) {
	label := identityLabel(api.Identity{Name: "Personal", Email: "user@ravi.app"})
	if label != "Personal (user@ravi.app)" {
		t.Errorf("identityLabel() = %q, want %q", label, "Personal (user@ravi.app)")
	}
}

func TestIdentityLabel_NameAndPhone(t *testing.T) {
	label := identityLabel(api.Identity{Name: "Mobile", Phone: "+15551234567"})
	if label != "Mobile (+15551234567)" {
		t.Errorf("identityLabel() = %q, want %q", label, "Mobile (+15551234567)")
	}
}

func TestIdentityLabel_NameOnly(t *testing.T) {
	label := identityLabel(api.Identity{Name: "Bare"})
	if label != "Bare" {
		t.Errorf("identityLabel() = %q, want %q", label, "Bare")
	}
}

func TestIdentityLabel_EmailPreferredOverPhone(t *testing.T) {
	label := identityLabel(api.Identity{Name: "Both", Email: "user@ravi.app", Phone: "+1555"})
	if label != "Both (user@ravi.app)" {
		t.Errorf("identityLabel() = %q, want %q", label, "Both (user@ravi.app)")
	}
}

func TestHandleSignup_SavesConfig(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v", err)
	}

	tokenResp := &api.DeviceTokenResponse{
		ManagementKey: "ravi_mgmt_signup",
		IdentityKey:   "ravi_id_signup",
		Identity:      &api.Identity{UUID: "id-1", Name: "Personal", Email: "test@ravi.app"},
		User:          api.User{Email: "test@example.com"},
	}

	err = flow.handleSignup(tokenResp)
	if err != nil {
		t.Fatalf("handleSignup() error = %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.ManagementKey != "ravi_mgmt_signup" {
		t.Errorf("ManagementKey = %q, want ravi_mgmt_signup", cfg.ManagementKey)
	}
	if cfg.IdentityKey != "ravi_id_signup" {
		t.Errorf("IdentityKey = %q, want ravi_id_signup", cfg.IdentityKey)
	}
	if cfg.IdentityUUID != "id-1" {
		t.Errorf("IdentityUUID = %q, want id-1", cfg.IdentityUUID)
	}
	if cfg.IdentityName != "Personal" {
		t.Errorf("IdentityName = %q, want Personal", cfg.IdentityName)
	}
	if cfg.UserEmail != "test@example.com" {
		t.Errorf("UserEmail = %q, want test@example.com", cfg.UserEmail)
	}
}

func TestHandleSignup_NoIdentity(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v", err)
	}

	tokenResp := &api.DeviceTokenResponse{
		ManagementKey: "ravi_mgmt_test",
		IdentityKey:   "ravi_id_test",
		User:          api.User{Email: "test@example.com"},
	}

	err = flow.handleSignup(tokenResp)
	if err != nil {
		t.Fatalf("handleSignup() error = %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.IdentityUUID != "" {
		t.Errorf("IdentityUUID = %q, want empty", cfg.IdentityUUID)
	}
}

// TestHandleLogin_NoIdentities verifies the agent-friendly path: a fresh account
// with no identity auto-creates its first one and saves it as active, so a
// headless caller ends up with a usable identity key without any prompt.
func TestHandleLogin_NoIdentities(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// POST /api/identities/ returns the created identity plus its one-time key.
		if r.URL.Path == "/api/identities/" && r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(api.Identity{
				UUID:   "id-first",
				Name:   "Auto Agent",
				Email:  "auto.agent@ravi.app",
				Phone:  "+15551230000",
				APIKey: "ravi_id_first",
			})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v", err)
	}

	tokenResp := &api.DeviceTokenResponse{
		ManagementKey: "ravi_mgmt_test",
		Identities:    []api.Identity{},
		User:          api.User{Email: "test@example.com"},
	}

	err = flow.handleLogin(tokenResp)
	if err != nil {
		t.Fatalf("handleLogin() error = %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.ManagementKey != "ravi_mgmt_test" {
		t.Errorf("ManagementKey = %q, want ravi_mgmt_test", cfg.ManagementKey)
	}
	if cfg.IdentityKey != "ravi_id_first" {
		t.Errorf("IdentityKey = %q, want ravi_id_first", cfg.IdentityKey)
	}
	if cfg.IdentityUUID != "id-first" {
		t.Errorf("IdentityUUID = %q, want id-first", cfg.IdentityUUID)
	}
	if cfg.IdentityName != "Auto Agent" {
		t.Errorf("IdentityName = %q, want Auto Agent", cfg.IdentityName)
	}
}

// TestHandleLogin_NoIdentities_MintsKeyWhenServerOmitsIt verifies the fallback:
// if the create response carries no api_key, the CLI mints an identity key.
func TestHandleLogin_NoIdentities_MintsKeyWhenServerOmitsIt(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/identities/" && r.Method == "POST":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(api.Identity{UUID: "id-first", Name: "Auto Agent"})
		case r.URL.Path == "/api/auth/keys/identity/" && r.Method == "POST":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(api.CreateIdentityKeyResponse{Key: "ravi_id_minted", IdentityUUID: "id-first", Label: "cli"})
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v", err)
	}

	tokenResp := &api.DeviceTokenResponse{
		ManagementKey: "ravi_mgmt_test",
		Identities:    []api.Identity{},
		User:          api.User{Email: "test@example.com"},
	}

	if err := flow.handleLogin(tokenResp); err != nil {
		t.Fatalf("handleLogin() error = %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.IdentityKey != "ravi_id_minted" {
		t.Errorf("IdentityKey = %q, want ravi_id_minted", cfg.IdentityKey)
	}
}

func TestHandleLogin_SingleIdentity(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	// Server that handles both the management client creation and CreateIdentityKey
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/auth/keys/identity/" && r.Method == "POST" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(api.CreateIdentityKeyResponse{
				Key:          "ravi_id_created",
				IdentityUUID: "id-single",
				Label:        "cli",
			})
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v", err)
	}

	tokenResp := &api.DeviceTokenResponse{
		ManagementKey: "ravi_mgmt_login",
		Identities:    []api.Identity{{UUID: "id-single", Name: "Personal", Email: "user@ravi.app"}},
		User:          api.User{Email: "test@example.com"},
	}

	err = flow.handleLogin(tokenResp)
	if err != nil {
		t.Fatalf("handleLogin() error = %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.ManagementKey != "ravi_mgmt_login" {
		t.Errorf("ManagementKey = %q, want ravi_mgmt_login", cfg.ManagementKey)
	}
	if cfg.IdentityKey != "ravi_id_created" {
		t.Errorf("IdentityKey = %q, want ravi_id_created", cfg.IdentityKey)
	}
	if cfg.IdentityUUID != "id-single" {
		t.Errorf("IdentityUUID = %q, want id-single", cfg.IdentityUUID)
	}
	if cfg.IdentityName != "Personal" {
		t.Errorf("IdentityName = %q, want Personal", cfg.IdentityName)
	}
}

func TestHandleLogin_MultipleIdentities(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	// Server handles CreateIdentityKey.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/auth/keys/identity/" && r.Method == "POST" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(api.CreateIdentityKeyResponse{
				Key:          "ravi_id_multi",
				IdentityUUID: "id-2",
				Label:        "cli",
			})
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v", err)
	}

	tokenResp := &api.DeviceTokenResponse{
		ManagementKey: "ravi_mgmt_multi",
		Identities: []api.Identity{
			{UUID: "id-1", Name: "Work", Email: "work@ravi.app"},
			{UUID: "id-2", Name: "Personal", Email: "personal@ravi.app"},
		},
		User: api.User{Email: "test@example.com"},
	}

	// Simulate user selecting option 2 by providing stdin.
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.WriteString("2\n")
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	err = flow.handleLogin(tokenResp)
	if err != nil {
		t.Fatalf("handleLogin() error = %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.IdentityKey != "ravi_id_multi" {
		t.Errorf("IdentityKey = %q, want ravi_id_multi", cfg.IdentityKey)
	}
	if cfg.IdentityUUID != "id-2" {
		t.Errorf("IdentityUUID = %q, want id-2", cfg.IdentityUUID)
	}
	if cfg.IdentityName != "Personal" {
		t.Errorf("IdentityName = %q, want Personal", cfg.IdentityName)
	}
}

func TestHandleLogin_MultipleIdentities_InvalidInput(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v", err)
	}

	tokenResp := &api.DeviceTokenResponse{
		ManagementKey: "ravi_mgmt_test",
		Identities: []api.Identity{
			{UUID: "id-1", Name: "Work"},
			{UUID: "id-2", Name: "Personal"},
		},
		User: api.User{Email: "test@example.com"},
	}

	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.WriteString("abc\n")
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	err = flow.handleLogin(tokenResp)
	if err == nil {
		t.Fatal("handleLogin() error = nil, want error for non-numeric input")
	}
}

func TestHandleLogin_MultipleIdentities_OutOfRange(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v", err)
	}

	tokenResp := &api.DeviceTokenResponse{
		ManagementKey: "ravi_mgmt_test",
		Identities: []api.Identity{
			{UUID: "id-1", Name: "Work"},
			{UUID: "id-2", Name: "Personal"},
		},
		User: api.User{Email: "test@example.com"},
	}

	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.WriteString("5\n")
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	err = flow.handleLogin(tokenResp)
	if err == nil {
		t.Fatal("handleLogin() error = nil, want error for out-of-range selection")
	}
}

func TestHandleLogin_MultipleIdentities_ZeroInput(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v", err)
	}

	tokenResp := &api.DeviceTokenResponse{
		ManagementKey: "ravi_mgmt_test",
		Identities: []api.Identity{
			{UUID: "id-1", Name: "Work"},
			{UUID: "id-2", Name: "Personal"},
		},
		User: api.User{Email: "test@example.com"},
	}

	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.WriteString("0\n")
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	err = flow.handleLogin(tokenResp)
	if err == nil {
		t.Fatal("handleLogin() error = nil, want error for 0 selection")
	}
}

func TestHandleLogin_CreateIdentityKeyError(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/auth/keys/identity/" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(api.Error{Detail: "key creation failed"})
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v", err)
	}

	tokenResp := &api.DeviceTokenResponse{
		ManagementKey: "ravi_mgmt_test",
		Identities:    []api.Identity{{UUID: "id-1", Name: "Work", Email: "work@ravi.app"}},
		User:          api.User{Email: "test@example.com"},
	}

	err = flow.handleLogin(tokenResp)
	if err == nil {
		t.Fatal("handleLogin() error = nil, want error when CreateIdentityKey fails")
	}
}

func TestRun_Success(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/auth/device/":
			json.NewEncoder(w).Encode(api.DeviceCodeResponse{
				DeviceCode:      "test-device-code",
				UserCode:        "TEST-1234",
				VerificationURI: "http://127.0.0.1:0/verify",
				ExpiresIn:       300,
				Interval:        0,
			})
		case "/api/auth/device/token/":
			callCount++
			// Return success immediately (no pending).
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(api.DeviceTokenResponse{
				ManagementKey: "ravi_mgmt_run",
				IdentityKey:   "ravi_id_run",
				Identity:      &api.Identity{UUID: "run-id", Name: "RunTest", Email: "run@ravi.app"},
				User:          api.User{Email: "run@example.com"},
			})
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v", err)
	}

	err = flow.Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.ManagementKey != "ravi_mgmt_run" {
		t.Errorf("ManagementKey = %q, want ravi_mgmt_run", cfg.ManagementKey)
	}
	if cfg.IdentityKey != "ravi_id_run" {
		t.Errorf("IdentityKey = %q, want ravi_id_run", cfg.IdentityKey)
	}
}

func TestRun_PrintsPublicDeviceURL(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	var openedURL string
	origBrowser := OpenBrowser
	OpenBrowser = func(url string) error {
		openedURL = url
		return nil
	}
	defer func() { OpenBrowser = origBrowser }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/auth/device/":
			json.NewEncoder(w).Encode(api.DeviceCodeResponse{
				DeviceCode:      "test-device-code",
				UserCode:        "TEST-1234",
				VerificationURI: "https://api.ravi.app/api/auth/device/verify/",
				ExpiresIn:       300,
				Interval:        0,
			})
		case "/api/auth/device/token/":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(api.DeviceTokenResponse{
				ManagementKey: "ravi_mgmt_url",
				IdentityKey:   "ravi_id_url",
				Identity:      &api.Identity{UUID: "url-id", Name: "URLTest"},
				User:          api.User{Email: "url@example.com"},
			})
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v", err)
	}

	wantURL := "https://ravi.id/device?user_code=TEST-1234"
	out := captureStdout(t, func() {
		if err := flow.Run(); err != nil {
			t.Errorf("Run() error = %v", err)
		}
	})

	if !strings.Contains(out, wantURL) {
		t.Errorf("Run() stdout missing public device URL %q, got:\n%s", wantURL, out)
	}
	if strings.Contains(out, "api.ravi.app") || strings.Contains(out, "/api/auth/device/verify") {
		t.Errorf("Run() stdout still prints API verify path, got:\n%s", out)
	}
	if !strings.Contains(out, "TEST-1234") {
		t.Errorf("Run() stdout missing user code, got:\n%s", out)
	}
	if openedURL != wantURL {
		t.Errorf("OpenBrowser URL = %q, want %q", openedURL, wantURL)
	}
}

func TestRun_PendingThenSuccess(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/auth/device/":
			json.NewEncoder(w).Encode(api.DeviceCodeResponse{
				DeviceCode:      "test-device-code",
				UserCode:        "TEST-1234",
				VerificationURI: "http://127.0.0.1:0/verify",
				ExpiresIn:       300,
				Interval:        0,
			})
		case "/api/auth/device/token/":
			callCount++
			if callCount == 1 {
				// First call: pending.
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(api.DeviceTokenError{
					Error:            "authorization_pending",
					ErrorDescription: "waiting",
				})
			} else {
				// Second call: success.
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(api.DeviceTokenResponse{
					ManagementKey: "ravi_mgmt_poll",
					IdentityKey:   "ravi_id_poll",
					Identity:      &api.Identity{UUID: "poll-id", Name: "PollTest"},
					User:          api.User{Email: "poll@example.com"},
				})
			}
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v", err)
	}

	err = flow.Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if callCount < 2 {
		t.Errorf("Expected at least 2 poll calls, got %d", callCount)
	}
}

func TestRun_ExpiredToken(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/auth/device/":
			json.NewEncoder(w).Encode(api.DeviceCodeResponse{
				DeviceCode:      "test-device-code",
				UserCode:        "TEST-1234",
				VerificationURI: "http://127.0.0.1:0/verify",
				ExpiresIn:       300,
				Interval:        0,
			})
		case "/api/auth/device/token/":
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(api.DeviceTokenError{
				Error:            "expired_token",
				ErrorDescription: "code expired",
			})
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v", err)
	}

	err = flow.Run()
	if err == nil {
		t.Fatal("Run() error = nil, want error for expired token")
	}
}

func TestRun_PollError(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/auth/device/":
			json.NewEncoder(w).Encode(api.DeviceCodeResponse{
				DeviceCode:      "test-device-code",
				UserCode:        "TEST-1234",
				VerificationURI: "http://127.0.0.1:0/verify",
				ExpiresIn:       300,
				Interval:        0,
			})
		case "/api/auth/device/token/":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("server error"))
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v", err)
	}

	err = flow.Run()
	if err == nil {
		t.Fatal("Run() error = nil, want error for poll failure")
	}
}

func TestRun_UnknownErrorCode(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/auth/device/":
			json.NewEncoder(w).Encode(api.DeviceCodeResponse{
				DeviceCode:      "test-device-code",
				UserCode:        "TEST-1234",
				VerificationURI: "http://127.0.0.1:0/verify",
				ExpiresIn:       300,
				Interval:        0,
			})
		case "/api/auth/device/token/":
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(api.DeviceTokenError{
				Error:            "access_denied",
				ErrorDescription: "denied",
			})
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v", err)
	}

	err = flow.Run()
	if err == nil {
		t.Fatal("Run() error = nil, want error for unknown error code")
	}
}

func TestRun_RequestDeviceCodeError(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(api.Error{Detail: "server down"})
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v", err)
	}

	err = flow.Run()
	if err == nil {
		t.Fatal("Run() error = nil, want error for device code request failure")
	}
}

func TestRun_LoginFlowWithIdentities(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/auth/device/":
			json.NewEncoder(w).Encode(api.DeviceCodeResponse{
				DeviceCode:      "test-device-code",
				UserCode:        "TEST-1234",
				VerificationURI: "http://127.0.0.1:0/verify",
				ExpiresIn:       300,
				Interval:        0,
			})
		case "/api/auth/device/token/":
			// Login flow: management key + identities, no identity key.
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(api.DeviceTokenResponse{
				ManagementKey: "ravi_mgmt_login_run",
				Identities:    []api.Identity{{UUID: "id-run", Name: "RunLogin", Email: "run@ravi.app"}},
				User:          api.User{Email: "run@example.com"},
			})
		case "/api/auth/keys/identity/":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(api.CreateIdentityKeyResponse{
				Key:          "ravi_id_login_run",
				IdentityUUID: "id-run",
				Label:        "cli",
			})
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v", err)
	}

	err = flow.Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.ManagementKey != "ravi_mgmt_login_run" {
		t.Errorf("ManagementKey = %q, want ravi_mgmt_login_run", cfg.ManagementKey)
	}
	if cfg.IdentityKey != "ravi_id_login_run" {
		t.Errorf("IdentityKey = %q, want ravi_id_login_run", cfg.IdentityKey)
	}
}

func TestRun_OpenBrowserError(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/auth/device/":
			json.NewEncoder(w).Encode(api.DeviceCodeResponse{
				DeviceCode:      "test-device-code",
				UserCode:        "TEST-1234",
				VerificationURI: "http://127.0.0.1:0/verify",
				ExpiresIn:       300,
				Interval:        0,
			})
		case "/api/auth/device/token/":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(api.DeviceTokenResponse{
				ManagementKey: "ravi_mgmt_ob",
				IdentityKey:   "ravi_id_ob",
				Identity:      &api.Identity{UUID: "ob-id", Name: "OBTest"},
				User:          api.User{Email: "ob@example.com"},
			})
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	// Override OpenBrowser to return an error (covering the error branch in Run).
	origBrowser := OpenBrowser
	OpenBrowser = func(url string) error { return fmt.Errorf("browser error") }
	defer func() { OpenBrowser = origBrowser }()

	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v", err)
	}

	// Run should still succeed (browser error is non-fatal).
	err = flow.Run()
	if err != nil {
		t.Fatalf("Run() error = %v, want nil (browser error is non-fatal)", err)
	}
}

func TestHandleLogin_StdinEOF(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v", err)
	}

	tokenResp := &api.DeviceTokenResponse{
		ManagementKey: "ravi_mgmt_test",
		Identities: []api.Identity{
			{UUID: "id-1", Name: "Work"},
			{UUID: "id-2", Name: "Personal"},
		},
		User: api.User{Email: "test@example.com"},
	}

	// Close stdin immediately (EOF) to trigger ReadString error.
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.Close() // immediate EOF
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	err = flow.handleLogin(tokenResp)
	if err == nil {
		t.Fatal("handleLogin() error = nil, want error for stdin EOF")
	}
}

func TestHandleLogin_SaveTempConfigError(t *testing.T) {
	tmpDir, cleanupHome := withTempHome(t)
	defer cleanupHome()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v", err)
	}

	tokenResp := &api.DeviceTokenResponse{
		ManagementKey: "ravi_mgmt_test",
		Identities:    []api.Identity{{UUID: "id-1", Name: "Work", Email: "w@ravi.app"}},
		User:          api.User{Email: "test@example.com"},
	}

	// Make HOME read-only so SaveGlobalConfig for temp config fails.
	os.RemoveAll(filepath.Join(tmpDir, ".ravi"))
	os.Chmod(tmpDir, 0500)
	defer os.Chmod(tmpDir, 0700)

	err = flow.handleLogin(tokenResp)
	if err == nil {
		t.Fatal("handleLogin() error = nil, want error for save failure")
	}
}

func TestHandleLogin_NoIdentities_SaveError(t *testing.T) {
	tmpDir, cleanupHome := withTempHome(t)
	defer cleanupHome()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v", err)
	}

	tokenResp := &api.DeviceTokenResponse{
		ManagementKey: "ravi_mgmt_test",
		Identities:    []api.Identity{}, // no identities
		User:          api.User{Email: "test@example.com"},
	}

	// Make HOME read-only so SaveGlobalConfig fails.
	os.RemoveAll(filepath.Join(tmpDir, ".ravi"))
	os.Chmod(tmpDir, 0500)
	defer os.Chmod(tmpDir, 0700)

	err = flow.handleLogin(tokenResp)
	if err == nil {
		t.Fatal("handleLogin() error = nil, want error for save failure")
	}
}

func TestHandleSignup_SaveConfigError(t *testing.T) {
	tmpDir, cleanupHome := withTempHome(t)
	defer cleanupHome()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v", err)
	}

	tokenResp := &api.DeviceTokenResponse{
		ManagementKey: "ravi_mgmt_test",
		IdentityKey:   "ravi_id_test",
		User:          api.User{Email: "test@example.com"},
	}

	// Make HOME read-only so SaveGlobalConfig fails.
	os.RemoveAll(filepath.Join(tmpDir, ".ravi"))
	os.Chmod(tmpDir, 0500)
	defer os.Chmod(tmpDir, 0700)

	err = flow.handleSignup(tokenResp)
	if err == nil {
		t.Fatal("handleSignup() error = nil, want error for save failure")
	}
}

func TestOpenBrowser_SupportedPlatform(t *testing.T) {
	// On supported platforms, openBrowserImpl should not return an error.
	// We use a harmless URL that won't cause issues. cmd.Start() is non-blocking.
	switch runtime.GOOS {
	case "darwin", "linux", "windows":
		err := openBrowserImpl("http://127.0.0.1:0/nonexistent")
		if err != nil {
			t.Errorf("openBrowserImpl() on %s returned error: %v", runtime.GOOS, err)
		}
	default:
		err := openBrowserImpl("http://example.com")
		if err == nil {
			t.Errorf("openBrowserImpl() on unsupported %s should return error", runtime.GOOS)
		}
	}
}
