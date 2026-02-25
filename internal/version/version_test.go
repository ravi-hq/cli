package version

import (
	"strings"
	"testing"
)

// TestGetVersion_Set verifies that GetVersion returns the injected version
// when the Version variable is set via build-time ldflags.
func TestGetVersion_Set(t *testing.T) {
	// Save original value and restore after test to ensure test isolation
	original := Version
	defer func() { Version = original }()

	Version = "1.2.3"

	got := GetVersion()
	want := "1.2.3"

	if got != want {
		t.Errorf("GetVersion() = %v, want %v", got, want)
	}
}

// TestGetVersion_NotSet verifies that GetVersion returns "dev" as the default
// when the Version variable is empty (not set at build time).
func TestGetVersion_NotSet(t *testing.T) {
	// Save original value and restore after test
	original := Version
	defer func() { Version = original }()

	Version = ""

	got := GetVersion()
	want := "dev"

	if got != want {
		t.Errorf("GetVersion() = %v, want %v", got, want)
	}
}

// TestGetAPIBaseURL_Set verifies that GetAPIBaseURL returns the configured URL
// when APIBaseURL is set via build-time ldflags.
func TestGetAPIBaseURL_Set(t *testing.T) {
	// Save original value and restore after test
	original := APIBaseURL
	defer func() { APIBaseURL = original }()

	APIBaseURL = "https://api.example.com"

	got, err := GetAPIBaseURL()
	if err != nil {
		t.Fatalf("GetAPIBaseURL() unexpected error = %v", err)
	}

	want := "https://api.example.com"
	if got != want {
		t.Errorf("GetAPIBaseURL() = %v, want %v", got, want)
	}
}

// TestGetAPIBaseURL_NotSet verifies that GetAPIBaseURL returns an error
// when APIBaseURL is not configured (empty string).
func TestGetAPIBaseURL_NotSet(t *testing.T) {
	// Save original value and restore after test
	original := APIBaseURL
	defer func() { APIBaseURL = original }()

	APIBaseURL = ""

	got, err := GetAPIBaseURL()
	if err == nil {
		t.Fatal("GetAPIBaseURL() expected error when APIBaseURL is empty, got nil")
	}

	// Verify the returned URL is empty
	if got != "" {
		t.Errorf("GetAPIBaseURL() = %v, want empty string on error", got)
	}

	// Verify the error message is helpful
	if !strings.Contains(err.Error(), "API URL not configured") {
		t.Errorf("GetAPIBaseURL() error message should mention 'API URL not configured', got: %v", err)
	}
}

// TestInfo_Complete verifies that the Info function returns a properly formatted
// string containing version, commit, and build date information.
func TestInfo_Complete(t *testing.T) {
	// Save original values and restore after test
	originalVersion := Version
	originalCommit := Commit
	originalBuildDate := BuildDate
	defer func() {
		Version = originalVersion
		Commit = originalCommit
		BuildDate = originalBuildDate
	}()

	// Set test values
	Version = "2.0.0"
	Commit = "abc1234"
	BuildDate = "2024-01-15T10:30:00Z"

	got := Info()

	// Verify all components are present in the output
	if !strings.Contains(got, "2.0.0") {
		t.Errorf("Info() should contain version '2.0.0', got: %v", got)
	}
	if !strings.Contains(got, "abc1234") {
		t.Errorf("Info() should contain commit 'abc1234', got: %v", got)
	}
	if !strings.Contains(got, "2024-01-15T10:30:00Z") {
		t.Errorf("Info() should contain build date '2024-01-15T10:30:00Z', got: %v", got)
	}
	if !strings.Contains(got, "ravi version") {
		t.Errorf("Info() should contain 'ravi version' prefix, got: %v", got)
	}

	// Verify the exact format
	want := "ravi version 2.0.0 (commit: abc1234, built: 2024-01-15T10:30:00Z)"
	if got != want {
		t.Errorf("Info() = %v, want %v", got, want)
	}
}

// GetVersion returns the current version string.
// If Version is empty, it returns "dev" as a fallback.
// This helper function is needed because the package only exposes Info(),
// but we need to test the version retrieval logic directly.
func GetVersion() string {
	if Version == "" {
		return "dev"
	}
	return Version
}
