// Package config handles persistent storage of authentication and identity settings.
//
// Files are stored in ~/.ravi/ with restricted permissions (0600/0700):
//   - auth.json: tokens and encryption keys (LoadAuth/SaveAuth)
//   - config.json: active identity reference (LoadConfig/SaveConfig/SaveGlobalConfig)
//   - recovery-key.txt: encryption recovery key (SaveRecoveryKey)
//
// Identity resolution: .ravi/config.json in CWD > ~/.ravi/config.json > unscoped.
package config
