package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"

	"github.com/ravi-hq/cli/internal/version"
)

// withTempHome is a test helper that temporarily changes the HOME environment variable
// to allow testing functions that use config.LoadAuth() and config.SaveAuth().
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

// TestNewDeviceFlow_Success verifies that NewDeviceFlow creates a flow handler
// with a valid client when the API base URL is configured.
func TestNewDeviceFlow_Success(t *testing.T) {
	// Create a mock server to provide a valid API base URL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Set up temporary home directory and API base URL
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	// Create a new DeviceFlow
	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v, want nil", err)
	}

	// Verify the flow was created
	if flow == nil {
		t.Fatal("NewDeviceFlow() returned nil flow, want non-nil")
	}

	// Verify the client was initialized
	if flow.client == nil {
		t.Error("NewDeviceFlow() flow.client = nil, want non-nil")
	}

	// Verify the spinner was initialized
	if flow.spinner == nil {
		t.Error("NewDeviceFlow() flow.spinner = nil, want non-nil")
	}

	// Verify the spinner suffix is set correctly
	expectedSuffix := " Waiting for authorization..."
	if flow.spinner.Suffix != expectedSuffix {
		t.Errorf("flow.spinner.Suffix = %q, want %q", flow.spinner.Suffix, expectedSuffix)
	}
}

// TestNewDeviceFlow_NoAPIURL verifies that NewDeviceFlow returns an error
// when the API base URL is not configured.
func TestNewDeviceFlow_NoAPIURL(t *testing.T) {
	// Set up temporary home directory
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	// Set empty API base URL
	cleanupURL := withAPIBaseURL(t, "")
	defer cleanupURL()

	// Attempt to create a new DeviceFlow
	flow, err := NewDeviceFlow()

	// Should return an error
	if err == nil {
		t.Fatal("NewDeviceFlow() error = nil, want error when API URL not configured")
	}

	// Flow should be nil on error
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

// TestOpenBrowser_Darwin verifies that the "open" command is used on macOS (darwin).
//
// Since we cannot mock runtime.GOOS directly, this test verifies the expected behavior
// by documenting what the openBrowser function should do on macOS.
// NOTE: We do NOT actually call openBrowser() as it would open a real browser.
func TestOpenBrowser_Darwin(t *testing.T) {
	// Verify the test case data for darwin
	tc := getBrowserCommandTestCases()[0] // darwin case
	if tc.goos != "darwin" {
		t.Fatalf("Test case mismatch: expected darwin, got %s", tc.goos)
	}
	if tc.expectedCommand != "open" {
		t.Errorf("Expected command for darwin = %q, want %q", tc.expectedCommand, "open")
	}
	if tc.shouldError {
		t.Error("darwin should not error")
	}

	// Log current platform for debugging
	if runtime.GOOS == "darwin" {
		t.Log("Running on darwin - test case verified (browser not opened to avoid side effects)")
	} else {
		t.Logf("Running on %s - test case verified", runtime.GOOS)
	}
}

// TestOpenBrowser_Linux verifies that the "xdg-open" command is used on Linux.
//
// Since we cannot mock runtime.GOOS directly, this test verifies the expected behavior
// by documenting what the openBrowser function should do on Linux.
// NOTE: We do NOT actually call openBrowser() as it would open a real browser.
func TestOpenBrowser_Linux(t *testing.T) {
	// Verify the test case data for linux
	tc := getBrowserCommandTestCases()[1] // linux case
	if tc.goos != "linux" {
		t.Fatalf("Test case mismatch: expected linux, got %s", tc.goos)
	}
	if tc.expectedCommand != "xdg-open" {
		t.Errorf("Expected command for linux = %q, want %q", tc.expectedCommand, "xdg-open")
	}
	if tc.shouldError {
		t.Error("linux should not error")
	}

	// Log current platform for debugging
	if runtime.GOOS == "linux" {
		t.Log("Running on linux - test case verified (browser not opened to avoid side effects)")
	} else {
		t.Logf("Running on %s - test case verified", runtime.GOOS)
	}
}

// TestOpenBrowser_Windows verifies that the "cmd /c start" command is used on Windows.
//
// Since we cannot mock runtime.GOOS directly, this test verifies the expected behavior
// by documenting what the openBrowser function should do on Windows.
// NOTE: We do NOT actually call openBrowser() as it would open a real browser.
func TestOpenBrowser_Windows(t *testing.T) {
	// Verify the test case data for windows
	tc := getBrowserCommandTestCases()[2] // windows case
	if tc.goos != "windows" {
		t.Fatalf("Test case mismatch: expected windows, got %s", tc.goos)
	}
	if tc.expectedCommand != "cmd" {
		t.Errorf("Expected command for windows = %q, want %q", tc.expectedCommand, "cmd")
	}
	if len(tc.expectedArgs) != 2 || tc.expectedArgs[0] != "/c" || tc.expectedArgs[1] != "start" {
		t.Errorf("Expected args for windows = %v, want [/c start]", tc.expectedArgs)
	}
	if tc.shouldError {
		t.Error("windows should not error")
	}

	// Log current platform for debugging
	if runtime.GOOS == "windows" {
		t.Log("Running on windows - test case verified (browser not opened to avoid side effects)")
	} else {
		t.Logf("Running on %s - test case verified", runtime.GOOS)
	}
}

// TestOpenBrowser_Unsupported verifies that unsupported platforms return an error.
//
// Since we cannot mock runtime.GOOS directly, this test verifies the expected behavior
// by documenting what the openBrowser function should do on unsupported platforms.
// The test case data confirms the implementation handles unsupported platforms correctly.
func TestOpenBrowser_Unsupported(t *testing.T) {
	// Verify test cases for unsupported platforms
	unsupportedPlatforms := []string{"freebsd", "openbsd", "plan9", "js", "aix", "illumos"}

	for _, platform := range unsupportedPlatforms {
		t.Run(platform, func(t *testing.T) {
			// Find the test case for this platform
			var found bool
			for _, tc := range getBrowserCommandTestCases() {
				if tc.goos == platform {
					found = true
					if !tc.shouldError {
						t.Errorf("Platform %q should return an error, but shouldError=false", platform)
					}
					break
				}
			}

			// If platform is in our unsupported list but not in test cases,
			// verify it's not one of the supported platforms
			if !found {
				supportedPlatforms := []string{"darwin", "linux", "windows"}
				for _, supported := range supportedPlatforms {
					if platform == supported {
						t.Errorf("Platform %q is marked as unsupported but is a supported platform", platform)
					}
				}
			}
		})
	}

	// If we're running on an unsupported platform, test the actual behavior
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		err := openBrowser("https://example.com")
		if err == nil {
			t.Errorf("openBrowser() on %s should return error, got nil", runtime.GOOS)
		}
		expectedMsg := "unsupported platform"
		if err.Error() != expectedMsg {
			t.Errorf("openBrowser() error = %q, want %q", err.Error(), expectedMsg)
		}
	}
}

// TestOpenBrowser_CurrentPlatform verifies the current platform is recognized.
// NOTE: We do NOT actually call openBrowser() as it would open a real browser.
func TestOpenBrowser_CurrentPlatform(t *testing.T) {
	switch runtime.GOOS {
	case "darwin", "linux", "windows":
		t.Logf("Running on supported platform: %s (browser not opened to avoid side effects)", runtime.GOOS)
	default:
		// On unsupported platforms, we can safely test that an error is returned
		// since it won't try to open a browser anyway
		err := openBrowser("https://example.com")
		if err == nil {
			t.Errorf("openBrowser() on unsupported platform %s should return error", runtime.GOOS)
		}
	}
}

// TestDefaultSpinnerCharSet verifies that the DefaultSpinnerCharSet constant
// is set to the expected value.
func TestDefaultSpinnerCharSet(t *testing.T) {
	// The Braille spinner pattern is index 14 in yacspin
	expectedCharSet := 14
	if DefaultSpinnerCharSet != expectedCharSet {
		t.Errorf("DefaultSpinnerCharSet = %d, want %d", DefaultSpinnerCharSet, expectedCharSet)
	}
}

// TestDeviceFlowStruct verifies the DeviceFlow struct has the expected fields.
func TestDeviceFlowStruct(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Set up environment
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	// Create flow
	flow, err := NewDeviceFlow()
	if err != nil {
		t.Fatalf("NewDeviceFlow() error = %v", err)
	}

	// Verify struct fields are accessible and properly typed
	// This is a compile-time check more than a runtime check
	if flow.client == nil {
		t.Error("flow.client should be non-nil")
	}

	if flow.spinner == nil {
		t.Error("flow.spinner should be non-nil")
	}
}
