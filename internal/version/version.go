package version

import "fmt"

// Build-time information injected via ldflags.
// Example:
//
//	go build -ldflags "-X github.com/.../version.Version=1.0.0 -X github.com/.../version.APIBaseURL=https://ravi.id"
var (
	Version    = "dev"
	Commit     = "unknown"
	BuildDate  = "unknown"
	APIBaseURL = "" // Overridden at build time; defaults to https://ravi.id
)

// Info returns formatted version information for display.
func Info() string {
	return fmt.Sprintf("ravi version %s (commit: %s, built: %s)", Version, Commit, BuildDate)
}

// GetAPIBaseURL returns the configured API base URL.
// Falls back to https://ravi.id if not set at build time.
func GetAPIBaseURL() (string, error) {
	if APIBaseURL == "" {
		return "https://ravi.id", nil
	}
	return APIBaseURL, nil
}
