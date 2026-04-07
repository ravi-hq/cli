# CLAUDE.md

Ravi CLI is a Go command-line client for the Ravi backend. It gives AI agents programmatic
access to their provisioned email, phone, credentials, and 2FA.

## Using Ravi CLI as an AI Agent

```bash
# Identity management
ravi identity list                # List all identities
ravi identity create --name "X"   # Create a new identity
ravi identity use <uuid>           # Switch active identity

# Identity info
ravi get email                    # Get assigned email address
ravi get phone                    # Get assigned phone number

# Check for messages
ravi inbox sms --unread           # SMS conversations (grouped)
ravi inbox email --unread         # Email threads (grouped)
ravi message sms                  # Flat SMS list
ravi message email <message_id>   # Specific email by ID

# Send emails
ravi email compose --to user@example.com --subject "Hi" --body "<p>Hello</p>"
ravi email reply <message_id> --body "<p>Reply</p>"
ravi email reply <message_id> --body "<p>Reply</p>" --cc "a@b.com"  # Reply with CC
ravi email reply-all <message_id> --body "<p>Reply all</p>"
ravi email forward <message_id> --to user@example.com --body "<p>FYI</p>"  # Forward email

# Passwords (website credentials)
ravi passwords list               # List all entries
ravi passwords get <uuid>         # Show entry
ravi passwords create example.com        # Create (auto-generates password)
ravi passwords create example.com --username me@email.com --password 'mypass'
ravi passwords update <uuid> --password 'new'  # Update fields
ravi passwords delete <uuid>             # Delete entry
ravi passwords generate --length 32      # Generate without storing

# Secrets (key-value secrets)
ravi secrets set OPENAI_API_KEY "sk-..."   # Store a secret
ravi secrets get OPENAI_API_KEY             # Retrieve a secret
ravi secrets list                           # List all secrets
ravi secrets delete <uuid>                  # Delete a secret

# Feedback
ravi feedback "Your feedback message"   # Send feedback to Ravi team

# Auth
ravi auth status                  # Check authentication
```

**Agent workflow:** Select identity (`ravi identity use`) → get email/phone → sign up for service → wait → check inbox for OTP → complete verification.

See `.claude/skills/ravi-cli.md` for detailed usage instructions.

## Commands

```bash
# Build (API_URL is REQUIRED at build time)
make build API_URL=https://ravi.id      # Build binary
make build-all API_URL=https://ravi.id  # Cross-compile all platforms
make install API_URL=https://ravi.id    # Install to $GOPATH/bin

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
│   ├── client.go      # NewClient, authenticated requests (API key header)
│   ├── types.go       # ~250 lines of API response structs
│   ├── constants.go   # API endpoint paths
│   ├── identity.go    # ListIdentities, CreateIdentity API methods
│   ├── email.go       # Compose, reply, reply-all, forward, presign API methods
│   ├── attachment.go  # UploadAttachment orchestrator (presign + upload)
│   └── validation.go  # Client-side extension blocklist + size check
├── config/            # API key config (config.json)
├── output/            # Human/JSON formatters (switched by --human flag)
└── version/           # Build-time version info (ldflags)
pkg/cli/               # Cobra commands (identity, inbox, passwords, secrets, auth, get, message, email send)
    └── identity.go    # identity list/create/use commands
```

### Identity Resolution

Active identity is stored in `config.json` with:
- `identity_uuid` + `identity_name` — which identity is active
- `management_key` — API key for account-level operations
- `identity_key` — API key scoped to the active identity

Resolution order:

1. `.ravi/config.json` in CWD (project-level override)
2. `~/.ravi/config.json` (global default)

### Key Patterns

- **Output formatting**: Default is JSON. `--human` flag switches to human-readable. Global `output.Current` switches at runtime via `PersistentPreRun`
- **Auth**: API key sent as header on every request. `management_key` for account-level ops, `identity_key` for identity-scoped ops
- **Build-time config**: API URL injected via ldflags — no runtime config needed

## Code Style

- `gofmt` formatting
- Go idioms and effective Go guidelines
- Error wrapping: `fmt.Errorf("context: %w", err)`
- Conventional commits: `feat(scope):`, `fix(scope):`, `refactor(scope):`

## Gotchas

| Gotcha | Details |
|--------|---------|
| API_URL required at build time | `make build` without `API_URL=` errors. Binary without it crashes |
| SMS conversation IDs contain `+` | Phone numbers in IDs need `url.PathEscape()` for API calls |
| JSON field name mismatches | `Identity.Email` maps to JSON `"inbox"`, `Identity.Phone` maps to JSON `"phone"` |

## Anti-Patterns

| Anti-Pattern | Why It's Bad | Do This Instead |
|--------------|--------------|-----------------|
| Hardcoding API URL | Binary won't work in other environments | Always inject via `make build API_URL=...` |
| Parsing human output | Format changes break automation | Use default JSON output (omit `--human`) |
| Skipping `url.PathEscape` | Breaks API calls with `+` in phone numbers | Always escape conversation/thread IDs |

## Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| "API URL not configured" | Built without `API_URL` | Rebuild: `make build API_URL=https://ravi.id` |
| 401 on every command | Invalid or missing API key | Re-authenticate: `ravi auth login` |
| `golangci-lint` not found | Not installed | `brew install golangci-lint` or see golangci-lint docs |
| Test fails with config error | Tests polluting `~/.ravi/` | Use `withTempHome(t)` helper to isolate |

## CI/CD

- **Release workflow** triggers on `v*` tag push
- Cross-compiles for darwin/linux (amd64+arm64)
- Creates GitHub release with SHA256 checksums
- Auto-updates Homebrew formula in `ravi-hq/homebrew-tap`
