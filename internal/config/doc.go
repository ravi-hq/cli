// Package config handles persistent storage of user credentials and settings.
//
// Configuration is stored in ~/.ravi/config.json with restricted file
// permissions (0600) to protect sensitive token data.
//
// The package provides functions to:
//   - Load: Read existing configuration from disk
//   - Save: Write configuration to disk with proper permissions
//   - Clear: Remove stored credentials (logout)
//   - ConfigPath: Get the path to the configuration file
package config
