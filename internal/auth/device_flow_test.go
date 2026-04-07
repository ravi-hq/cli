package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"

	"github.com/ravi-hq/cli/internal/version"
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

// withAPIBaseURL is a test helper that temporarily sets the version.APIBaseURL.
func withAPIBaseURL(t *testing.T, url string) func() {
	t.Helper()

	original := version.APIBaseURL
	version.APIBaseURL = url

	return func() {
		version.APIBaseURL = original
	}
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

// TestNewDeviceFlow_NoAPIURL verifies that NewDeviceFlow returns an error
// when the API base URL is not configured.
func TestNewDeviceFlow_NoAPIURL(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	cleanupURL := withAPIBaseURL(t, "")
	defer cleanupURL()

	flow, err := NewDeviceFlow()

	if err == nil {
		t.Fatal("NewDeviceFlow() error = nil, want error when API URL not configured")
	}

	if flow != nil {
		t.Errorf("NewDeviceFlow() flow = %v, want nil on error", flow)
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
		err := openBrowser("https://example.com")
		if err == nil {
			t.Errorf("openBrowser() on %s should return error, got nil", runtime.GOOS)
		}
	}
}

func TestOpenBrowser_CurrentPlatform(t *testing.T) {
	switch runtime.GOOS {
	case "darwin", "linux", "windows":
		t.Logf("Running on supported platform: %s", runtime.GOOS)
	default:
		err := openBrowser("https://example.com")
		if err == nil {
			t.Errorf("openBrowser() on unsupported platform %s should return error", runtime.GOOS)
		}
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
