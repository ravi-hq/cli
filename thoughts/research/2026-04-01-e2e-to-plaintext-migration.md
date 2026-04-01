---
date: 2026-04-01T11:00:00-04:00
researcher: Claude Code (team-research skill)
git_commit: eacb20879225b9a593b86aff908e08425b464e9b
branch: main
repository: ravi-hq/cli
topic: "E2E to plaintext migration: handling 404 from encryption metadata endpoint"
tags: [research, team-research, encryption, auth, migration]
status: complete
method: agent-team
team_size: 4
tracks: [login-flow, command-usage, api-errors, config-state]
last_updated: 2026-04-01
last_updated_by: Claude Code
---

# Research: E2E to Plaintext Migration

**Date**: 2026-04-01T11:00:00-04:00
**Researcher**: Claude Code (team-research)
**Git Commit**: [`eacb208`](https://github.com/ravi-hq/cli/commit/eacb20879225b9a593b86aff908e08425b464e9b)
**Branch**: `main`
**Repository**: ravi-hq/cli
**Method**: Agent team (4 specialist researchers)

## Research Question

The API has been updated to migrate away from E2E encryption. Users who don't use E2E now get a 404 when fetching encryption metadata (`GET /api/encryption/`). The CLI must handle this 404 gracefully during login and in all commands, while still supporting existing E2E users.

## Summary

The CLI currently treats encryption as mandatory — login fails entirely if `GetEncryptionMeta()` errors, and all 13 command paths that touch encrypted data hard-fail when no keypair exists. The fix requires changes at 3 layers: (1) add a `NotFoundError` type to the API client so callers can detect 404s, (2) make `setupEncryption()` skip encryption on 404 and flag the user as plaintext in `auth.json`, and (3) make `ensureKeyPair()` return `(nil, nil)` for plaintext users so commands can proceed with unencrypted data. The existing `tryDecrypt()` and `crypto.DecryptField()` functions already handle plaintext values correctly — the only blocker is the hard error in `ensureKeyPair()`.

## Research Tracks

### Track 1: Login Flow & Encryption Setup
**Researcher**: login-flow-researcher
**Scope**: `internal/auth/device_flow.go`, `internal/api/encryption.go`, `internal/api/client.go`

#### Findings:
1. **setupEncryption is mandatory and blocking** — After successful OAuth token exchange, `Run()` calls `setupEncryption(auth)` unconditionally at `device_flow.go:113`. If it errors, login fails with `"encryption setup failed: %w"`. (`device_flow.go:113-115`)
2. **setupEncryption assumes encryption always exists** — It calls `GetEncryptionMeta()`, then branches on `meta.PublicKey == ""` (first-time setup) vs non-empty (unlock existing). There is no "no encryption" path. (`device_flow.go:138-151`)
3. **404 is an opaque error** — `parseResponse` at `client.go:174-179` converts 404 into `fmt.Errorf("API error (status %d): ...")` — callers cannot detect it without string parsing.
4. **GetEncryptionMeta returns no status info** — The function signature `(*EncryptionMeta, error)` gives no way to signal "not found" vs "server error". (`encryption.go:6-12`)
5. **Auth config already accommodates empty encryption fields** — `PINSalt`, `PublicKey`, `PrivateKey` are all `omitempty` in `AuthConfig`. Plaintext users just have these empty. (`config.go:21-29`)

### Track 2: Command-Level E2E Usage
**Researcher**: command-researcher
**Scope**: `pkg/cli/` — `e2e.go`, `passwords.go`, `secrets.go`, `inbox_email.go`, `inbox_sms.go`, `message.go`

#### Findings:
1. **13 command paths call ensureKeyPair()** — Spread across 5 files: `passwords.go` (4 calls), `secrets.go` (3), `inbox_email.go` (2), `inbox_sms.go` (2), `message.go` (2). All fail if it returns error. (`pkg/cli/passwords.go:45,93,128,196`, `pkg/cli/secrets.go:31,83,116`, etc.)
2. **Read paths already handle plaintext gracefully** — `tryDecrypt()` calls `crypto.DecryptField()` which checks for `"e2e::"` prefix and returns value as-is if absent. Plaintext data flows through untouched. (`crypto/e2e.go:86-88`, `pkg/cli/e2e.go:55-62`)
3. **Write paths encrypt before storing** — `passwords create/edit` and `secrets set` call `crypto.Encrypt()` with the public key. For plaintext users, these must skip encryption and store raw values. (`pkg/cli/passwords.go:128`, `pkg/cli/secrets.go:116`)
4. **ensureKeyPair() is the single gate** — The hard error at `e2e.go:23` ("encryption not set up") blocks all 13 paths. Changing this one function unblocks everything. (`pkg/cli/e2e.go:21-26`)
5. **tryDecrypt with nil keypair needs a guard** — If `ensureKeyPair()` returns `(nil, nil)`, `tryDecrypt()` must handle nil `kp`. Since plaintext data won't have `"e2e::"` prefix, `DecryptField` would return it unchanged — but passing nil kp to `Decrypt()` would panic on encrypted data. A nil check is needed. (`pkg/cli/e2e.go:55-62`)

### Track 3: API Error Propagation & 404 Detection
**Researcher**: api-error-researcher
**Scope**: `internal/api/client.go`, `internal/api/types.go`

#### Findings:
1. **parseResponse discards HTTP status codes** — All 4xx/5xx errors are collapsed into plain `fmt.Errorf` strings. No typed errors except `RateLimitError` (429). (`client.go:159-179`)
2. **RateLimitError is the pattern to follow** — It's a struct with `Error() string` method, checked before the generic `>= 400` block. A `NotFoundError` should use the same pattern. (`types.go:297-307`)
3. **Recommended: Add NotFoundError typed error** — Handle `http.StatusNotFound` in `parseResponse` before the generic `>= 400` block. Callers use `errors.As(err, &api.NotFoundError{})` to detect 404. (`client.go:174`)
4. **No other endpoints currently need 404 handling** — `GetEncryptionMeta` is the only call where 404 signals a valid state (plaintext user) rather than a bug.
5. **Alternative: return (nil, nil) from GetEncryptionMeta** — Instead of a typed error, `GetEncryptionMeta` could use a lower-level method that checks the response status and returns `(nil, nil)` on 404. This is simpler but less reusable.

### Track 4: Config & State for Plaintext Users
**Researcher**: config-researcher
**Scope**: `internal/config/config.go`, `pkg/cli/e2e.go`, `pkg/cli/e2e_test.go`

#### Findings:
1. **AuthConfig already represents plaintext users** — Empty `PINSalt`/`PublicKey`/`PrivateKey` with `omitempty` tags means plaintext users just have a smaller `auth.json`. No schema change strictly required. (`config.go:21-29`)
2. **ensureKeyPair conflates "no encryption" with "broken setup"** — When `AccessToken` is set but keys are empty, it returns "encryption not set up — complete PIN setup on the dashboard first". This is correct for e2e users who haven't finished setup, but wrong for plaintext users. (`e2e.go:21-26`)
3. **Test explicitly asserts the wrong behavior for plaintext** — `TestEnsureKeyPair_LoggedInButNoPIN` asserts the "encryption not set up" error. This test will need updating. (`e2e_test.go:167-185`)
4. **Recommend adding PlaintextMode flag to AuthConfig** — `PlaintextMode bool json:"plaintext_mode,omitempty"` distinguishes "plaintext user" from "e2e user who hasn't set up PIN yet". Zero value (false) means backward-compatible for existing e2e users. (`config.go:21-29`)
5. **Backward compatibility confirmed** — Existing e2e users have keys in auth.json; nothing changes for them. `LoadAuth` returns empty struct for missing file; `SaveAuth` has no encryption-specific logic. (`config.go:55-92`)

## Cross-Track Discoveries

- **tryDecrypt is already plaintext-safe** — Track 2 confirmed that `crypto.DecryptField()` (Track 1's domain) returns plaintext unchanged when no `"e2e::"` prefix is present. The only blocker is `ensureKeyPair()` erroring before `tryDecrypt` is reached.
- **The PlaintextMode flag resolves Track 2 + Track 4's shared problem** — Track 2 found that `ensureKeyPair()` can't distinguish plaintext users from broken e2e setup. Track 4's `PlaintextMode` flag solves this cleanly.
- **NotFoundError (Track 3) feeds directly into Track 1** — The typed error approach lets `setupEncryption` detect 404 with `errors.As` and set `PlaintextMode = true` in the auth config.

## Code References

| File | Tracks | Findings | Description |
|------|--------|----------|-------------|
| `internal/api/client.go:174-179` | 1, 3 | parseResponse 404 handling | Add NotFoundError before generic 400+ block |
| `internal/api/types.go:297-307` | 3 | RateLimitError pattern | Follow this for NotFoundError |
| `internal/auth/device_flow.go:113-115` | 1 | setupEncryption call | Must handle 404 gracefully |
| `internal/auth/device_flow.go:138-151` | 1 | setupEncryption body | Add nil-meta / 404 branch |
| `internal/api/encryption.go:6-12` | 1, 3 | GetEncryptionMeta | Error propagation point |
| `internal/config/config.go:21-29` | 4 | AuthConfig struct | Add PlaintextMode field |
| `pkg/cli/e2e.go:15-26` | 2, 4 | ensureKeyPair | Return (nil, nil) for plaintext |
| `pkg/cli/e2e.go:55-62` | 2 | tryDecrypt | Add nil kp guard |
| `pkg/cli/e2e_test.go:167-185` | 4 | TestEnsureKeyPair_LoggedInButNoPIN | Update for plaintext |
| `pkg/cli/passwords.go:128,196` | 2 | Write paths | Skip encryption if kp nil |
| `pkg/cli/secrets.go:116` | 2 | secrets set | Skip encryption if kp nil |

## Architecture Insights

The encryption layer has clean separation:
- **Crypto layer** (`internal/crypto/`): Pure functions, no state. Already handles plaintext correctly via `IsEncrypted()` prefix check.
- **CLI layer** (`pkg/cli/e2e.go`): Gate functions (`ensureKeyPair`, `tryDecrypt`). This is where the plaintext bypass belongs.
- **Auth layer** (`internal/auth/`): Login flow. This is where plaintext mode gets detected and flagged.
- **Config layer** (`internal/config/`): State persistence. Stores the flag.

The migration is clean because plaintext data simply lacks the `"e2e::"` prefix, and the existing decrypt functions already handle that case. The only real work is removing the hard gates that assume encryption is always present.

## Implementation Plan

### Layer 1: API Client (internal/api/)
1. Add `NotFoundError` struct to `types.go` (follow `RateLimitError` pattern)
2. Add `http.StatusNotFound` check in `parseResponse` before the `>= 400` block

### Layer 2: Config (internal/config/)
3. Add `PlaintextMode bool json:"plaintext_mode,omitempty"` to `AuthConfig`

### Layer 3: Login Flow (internal/auth/)
4. In `setupEncryption`: catch `NotFoundError` from `GetEncryptionMeta()` → set `auth.PlaintextMode = true`, return nil
5. Keep `initialEncryptionSetup` and `unlockExistingEncryption` intact for existing e2e users

### Layer 4: CLI Commands (pkg/cli/)
6. In `ensureKeyPair`: if `PlaintextMode == true`, return `(nil, nil)` (no error)
7. In `tryDecrypt`: add nil `kp` guard — return value as-is
8. In write paths (passwords create/edit, secrets set): if `kp == nil`, store raw value without encrypting
9. Update `TestEnsureKeyPair_LoggedInButNoPIN` and add `TestEnsureKeyPair_PlaintextMode`

## Decisions

1. **Remove `initialEncryptionSetup`** — New users will never use e2e. The PIN setup flow should be deleted entirely. Only `unlockExistingEncryption` remains for legacy e2e users.
2. **Existing e2e users migrating to plaintext** — When the backend removes a user's encryption metadata (404 on re-login), the CLI should clear local encryption state (`PINSalt`, `PublicKey`, `PrivateKey` in auth.json) and set `PlaintextMode = true`. Trust the server entirely for data — no local decryption needed.

## Open Questions

1. **encodePublicKey helper** — Referenced in passwords/secrets write paths but not found in the searched files. Need to verify its location and whether it needs a nil guard.
