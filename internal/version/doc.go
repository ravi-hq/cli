// Package version provides build-time version information.
//
// Variables in this package are set at build time using -ldflags:
//
//	go build -ldflags "-X internal/version.Version=1.0.0"
//
// The package provides:
//   - Version: The application version string
//   - GetAPIBaseURL(): Returns the hosted Ravi API URL
//   - GetVersion(): Returns version or "dev" if not set
package version
