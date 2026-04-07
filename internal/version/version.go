package version

import (
	"errors"
	"fmt"
)

// Build-time information injected via ldflags.
// Example:
//
//	go build -ldflags "-X github.com/.../version.Version=1.0.0 -X github.com/.../version.APIBaseURL=https://ravi.id"
var (
	Version    = "dev"
	Commit     = "unknown"
	BuildDate  = "unknown"
	APIBaseURL = "" // Required, no default - must be set at build time
)

// Info returns formatted version information for display.
func Info() string {
	return fmt.Sprintf("ravi version %s (commit: %s, built: %s)", Version, Commit, BuildDate)
}

// GetAPIBaseURL returns the configured API base URL.
// Returns an error if the URL was not set at build time.
func GetAPIBaseURL() (string, error) {
	if APIBaseURL == "" {
		return "", errors.New("API URL not configured. Binary must be built with: make build API_URL=<url>")
	}
	return APIBaseURL, nil
}
