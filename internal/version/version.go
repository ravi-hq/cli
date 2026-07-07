package version

import (
	"fmt"
	"os"
	"strings"
)

// Build-time information injected via ldflags.
// Example:
//
//	go build -ldflags "-X github.com/.../version.Version=1.0.0"
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"

	// APIBaseURL is the hosted Ravi API host. It is a var (not a const) so the
	// release build can inject it via `-ldflags "-X .../version.APIBaseURL=..."`.
	APIBaseURL = "https://api.ravi.app"
)

const (
	// apiBaseURLEnv overrides the API host in any build (useful for debugging
	// against a staging host or a local server).
	apiBaseURLEnv = "RAVI_API_URL"
	// testAPIBaseURLEnv overrides the API host only under `go test` binaries.
	testAPIBaseURLEnv = "RAVI_CLI_TEST_API_BASE_URL"
)

// Info returns formatted version information for display.
func Info() string {
	return fmt.Sprintf("ravi version %s (commit: %s, built: %s)", Version, Commit, BuildDate)
}

// GetAPIBaseURL returns the configured API base URL. It prefers the RAVI_API_URL
// env override, then the test-only override under `go test`, and otherwise the
// hosted default (https://api.ravi.app).
func GetAPIBaseURL() (string, error) {
	if value := os.Getenv(apiBaseURLEnv); value != "" {
		return value, nil
	}
	if strings.HasSuffix(os.Args[0], ".test") {
		if value := os.Getenv(testAPIBaseURLEnv); value != "" {
			return value, nil
		}
	}
	return APIBaseURL, nil
}
