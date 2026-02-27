# CLAUDE.md

Ravi CLI is a Go command-line client for the Ravi backend. It gives AI agents programmatic
access to their provisioned email, phone, credential vault, and 2FA — with E2E encryption
handled client-side.

## Using Ravi CLI as an AI Agent

```bash
# Identity management
ravi identity list --json                # List all identities
ravi identity create --name "X" --json   # Create a new identity
ravi identity use <name-or-uuid>         # Switch active identity

# Identity info
ravi get email --json                    # Get assigned email address
ravi get phone --json                    # Get assigned phone number

# Check for messages
ravi inbox sms --unread --json           # SMS conversations (grouped)
ravi inbox email --unread --json         # Email threads (grouped)
ravi message sms --json                  # Flat SMS list
ravi message email <message_id> --json   # Specific email by ID

# Send emails
ravi email compose --to user@example.com --subject "Hi" --body "<p>Hello</p>" --json
ravi email reply <message_id> --subject "Re: Hi" --body "<p>Reply</p>" --json
ravi email reply-all <message_id> --subject "Re: Hi" --body "<p>Reply all</p>" --json

# Passwords (E2E encrypted website credentials)
ravi passwords list --json               # List all entries
ravi passwords get <uuid> --json         # Show entry (decrypted)
ravi passwords create example.com        # Create (auto-generates password)
ravi passwords create example.com --username me@email.com --password 'mypass'
ravi passwords edit <uuid> --password 'new'  # Edit fields
ravi passwords delete <uuid>             # Delete entry
ravi passwords generate --length 32      # Generate without storing

# Secrets (E2E encrypted key-value secrets)
ravi secrets set OPENAI_API_KEY "sk-..." --json   # Store a secret
ravi secrets get OPENAI_API_KEY --json             # Retrieve a secret
ravi secrets list --json                           # List all secrets
ravi secrets delete <uuid> --json                  # Delete a secret

# Feedback
ravi feedback "Your feedback message" --json   # Send feedback to Ravi team

# Auth
ravi auth status --json                  # Check authentication
```

**Agent workflow:** Select identity (`ravi identity use`) → get email/phone → sign up for service → wait → check inbox for OTP → complete verification.

See `.claude/skills/ravi-cli.md` for detailed usage instructions.

## Commands

```bash
# Build (API_URL is REQUIRED at build time)
make build API_URL=https://ravi.app      # Build binary
make build-all API_URL=https://ravi.app  # Cross-compile all platforms
make install API_URL=https://ravi.app    # Install to $GOPATH/bin

# Development
make test              # Run tests
make test-coverage     # Generate HTML coverage report
make lint              # Check with golangci-lint
make lint-fix          # Auto-fix lint issues
make deps              # Download and tidy dependencies
make clean             # Remove build artifacts
```

## Architecture

```text
cmd/ravi/              # Entry point (calls cli.Execute())
internal/
├── api/               # HTTP client, API types, endpoint constants
│   ├── client.go      # NewClient, auto token refresh, authenticated requests
│   ├── types.go       # ~250 lines of API response structs
│   ├── constants.go   # API endpoint paths
│   ├── identity.go    # ListIdentities, CreateIdentity API methods
│   ├── email.go       # Compose, reply, reply-all, presign API methods
│   ├── attachment.go  # UploadAttachment orchestrator (presign + upload)
│   └── validation.go  # Client-side extension blocklist + size check
├── auth/              # OAuth device code flow orchestration
├── config/            # Auth (auth.json) + identity config (config.json)
├── crypto/            # E2E encryption (Argon2id + NaCl SealedBox)
│   ├── e2e.go         # Key derivation, encrypt/decrypt
│   └── session.go     # PIN prompting, keypair caching (per-process)
├── output/            # Human/JSON formatters (switched by --json flag)
└── version/           # Build-time version info (ldflags)
pkg/cli/               # Cobra commands (identity, inbox, vault, secrets, auth, get, message, email send)
    ├── identity.go    # identity list/create/use commands
    └── e2e.go         # Helpers: ensureKeyPair(), tryDecrypt(), encodePublicKey()
```

### Identity Resolution

Active identity is stored as `IdentityUUID` + `IdentityName` in `config.json`. Resolution order:

1. `.ravi/config.json` in CWD (project-level override)
2. `~/.ravi/config.json` (global default)
3. Unscoped (no identity header sent)

### Key Patterns

- **Output formatting**: All commands support `--json` flag. Global `output.Current` switches at runtime via `PersistentPreRun`
- **Token refresh**: API client auto-refreshes on expiry or 401 (retry once, save to disk)
- **Build-time config**: API URL injected via ldflags — no runtime config needed
- **E2E encryption**: `internal/crypto/` handles client-side encrypt/decrypt.
  Empty strings never encrypted. `IsEncrypted()` checks `"e2e::"` prefix
- **PIN caching**: Keypair derived once per process, cached in package-level variable. `ClearCachedKeyPair()` on logout

## Code Style

- `gofmt` formatting
- Go idioms and effective Go guidelines
- Error wrapping: `fmt.Errorf("context: %w", err)`
- Conventional commits: `feat(scope):`, `fix(scope):`, `refactor(scope):`

## Gotchas

| Gotcha | Details |
|--------|---------|
| API_URL required at build time | `make build` without `API_URL=` errors. Binary without it crashes |
| Keypair cached per-process | PIN prompted once, then cached. No re-prompt within same CLI invocation |
| SMS conversation IDs contain `+` | Phone numbers in IDs need `url.PathEscape()` for API calls |
| PIN prompted to stderr | Uses `term.ReadPassword()` — works even when stdout is redirected |
| 3 PIN attempts max | After 3 wrong PINs, CLI aborts. No lockout on server side |
| Argon2id params must match libsodium | Time=3, Mem=64MB, Threads=1 — changing breaks server compat |
| JSON field name mismatches | `Identity.Email` maps to JSON `"inbox"`, `Identity.Phone` maps to JSON `"phone"` |

## Anti-Patterns

| Anti-Pattern | Why It's Bad | Do This Instead |
|--------------|--------------|-----------------|
| Hardcoding API URL | Binary won't work in other environments | Always inject via `make build API_URL=...` |
| Parsing human output | Format changes break automation | Always use `--json` flag |
| Re-prompting for PIN | Bad UX, keypair already cached | Use `ensureKeyPair()` which reads cache first |
| Encrypting empty strings | Wastes space, decrypt returns empty anyway | Check before encrypting (`""` → `""`) |
| Skipping `url.PathEscape` | Breaks API calls with `+` in phone numbers | Always escape conversation/thread IDs |

## Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| "API URL not configured" | Built without `API_URL` | Rebuild: `make build API_URL=https://ravi.app` |
| "encryption not set up" | No keypair in config | Run `ravi auth login` and complete PIN setup |
| 401 after token refresh | Refresh token also expired | Re-authenticate: `ravi auth login` |
| Decryption fails | Wrong PIN or corrupted config | Delete `~/.ravi/auth.json` and re-login |
| `golangci-lint` not found | Not installed | `brew install golangci-lint` or see golangci-lint docs |
| Test fails with config error | Tests polluting `~/.ravi/` | Use `withTempHome(t)` helper to isolate |

## CI/CD

- **Release workflow** triggers on `v*` tag push
- Cross-compiles for darwin/linux (amd64+arm64)
- Creates GitHub release with SHA256 checksums
- Auto-updates Homebrew formula in `ravi-hq/homebrew-tap`
