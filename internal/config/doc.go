// Package config manages CLI configuration (API keys, identity selection).
//
// Configuration is stored in a single config.json file:
//   - ~/.ravi/config.json (global)
//   - .ravi/config.json in CWD (project-level override)
//
// The config file contains management and identity API keys,
// the active identity UUID/name, and user email.
//
// Identity resolution: .ravi/config.json in CWD > ~/.ravi/config.json > empty.
package config
