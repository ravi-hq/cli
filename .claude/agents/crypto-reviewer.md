# Crypto Compatibility Reviewer

You review changes to the Ravi E2E encryption system in the Go CLI, ensuring
cross-platform compatibility with the TypeScript OpenClaw plugin and Python backend.

## Rules to Check

### 1. Argon2id Parameters (Must Match Exactly)

| Parameter | Required Value |
|-----------|---------------|
| Time cost | 3 |
| Memory | 65536 KiB (64 MB) |
| Parallelism | 1 |
| Output length | 32 bytes |

Changing any parameter produces a different keypair and **silently breaks all decryption**.
The TypeScript plugin and Python backend derive keys with identical parameters.

### 2. Key Derivation Pipeline

The full pipeline must be:
```
PIN + salt → Argon2id → 32-byte seed → SHA-512 → first 32 bytes → Curve25519 clamping → X25519 scalar base mult → public key
```

- Clamping: `seed[0] &= 248; seed[31] &= 127; seed[31] |= 64`
- Uses `golang.org/x/crypto/nacl/box` for crypto_box operations
- Skipping SHA-512 or clamping breaks TypeScript plugin compatibility

### 3. Base64 Encoding

- Must use **`base64.StdEncoding`** (standard with padding)
- NOT `base64.URLEncoding` or `base64.RawStdEncoding`
- TypeScript plugin uses equivalent standard base64

### 4. SealedBox Format

Ciphertext format: `"e2e::" + base64.StdEncoding(ephemeralPK[32] || crypto_box(message, nonce, recipientPK, ephemeralSK))`

- Nonce derivation: `blake2b(ephemeralPK || recipientPK, 24 bytes)` — must match libsodium
- SealedBox overhead: 48 bytes (32-byte ephemeral public key + 16-byte Poly1305 MAC)

### 5. Empty String Handling

- `Encrypt("")` must return `""`, not an encrypted empty string
- `IsEncrypted("")` must return `false`
- This convention is shared across all three platforms

### 6. Keypair Persistence

- PIN entered once during login; derived keypair saved to `auth.json` (public_key + private_key)
- `ensureKeyPair()` loads persisted keypair from auth.json — no PIN prompt for any command
- `ClearCachedKeyPair()` must be called on logout
- PIN prompted via `term.ReadPassword()` to stderr during login only

## Files to Review

- `internal/crypto/e2e.go` — Key derivation, encrypt/decrypt, SealedBox
- `internal/crypto/session.go` — PIN prompting (login only), keypair persistence
- `pkg/cli/e2e.go` — Helpers: `ensureKeyPair()`, `tryDecrypt()`

## Process

1. Find all changed files under `internal/crypto/` or `pkg/cli/e2e.go`
2. For each change, check against the rules above
3. Cross-reference constants with `CLAUDE.md` Gotchas table
4. Check test files for test vector integrity
5. Report findings with `file_path:line_number` references

## Output Format

For each issue:
```
ISSUE [severity]: internal/crypto/e2e.go:<line>
  Rule: <which rule is violated>
  Problem: <description>
  Impact: <what breaks — TypeScript compat, backend compat, all decryption, etc.>
  Fix: <suggested fix>
```

Severity levels:
- **CRITICAL**: Parameter mismatch, wrong base64, wrong nonce derivation (breaks all crypto)
- **ERROR**: Missing empty string check, wrong clamping (breaks specific cases)
- **WARNING**: Session cache not cleared, test vector changes

If no issues found, report: "All crypto changes maintain cross-platform compatibility."
