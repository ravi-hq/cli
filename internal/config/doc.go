// Package config manages CLI configuration (API keys, identity selection).
//
// The CLI holds one active identity per config file:
//   - ~/.ravi/config.json (global)
//   - .ravi/config.json in CWD (overrides global; still one identity)
//
// Shared ~/.ravi/config.json cannot run multiple agents. Several agents on
// one host should call the HTTP API with per-identity ravi_id_ keys.
//
// The config file contains management and identity API keys,
// the active identity UUID/name, and user email.
//
// Identity resolution: .ravi/config.json in CWD > ~/.ravi/config.json > empty.
package config
