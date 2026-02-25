# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Project Overview

Ravi CLI is a Go command-line client for the Ravi backend service. It provides programmatic access to inbox (emails and SMS) for AI agents.

**Purpose:** AI agents get their own dedicated phone number and email address through Ravi. They can use these to:

- Sign up for websites and services (use the email/phone in registration forms)
- Receive OTPs and verification codes
- Complete 2FA authentication flows
- Read confirmation emails and SMS messages
- Store and retrieve E2E-encrypted passwords per identity

## Using Ravi CLI as an AI Agent

### Quick Reference

```bash
# Get your assigned Ravi email and phone
ravi get email --json
ravi get phone --json

# Check authentication status
ravi auth status --json

# Check for SMS messages (e.g., OTP codes)
ravi inbox sms --unread --json

# Check for emails (e.g., verification links)
ravi inbox email --unread --json

# View specific SMS conversation
ravi inbox sms <conversation_id> --json

# View specific email thread
ravi inbox email <thread_id> --json

# Access individual messages (flat list, not grouped)
ravi message sms --json              # List all SMS messages
ravi message sms <message_id> --json # Get specific SMS by ID
ravi message email --json            # List all email messages
ravi message email <message_id> --json # Get specific email by ID

# Credential vault (E2E encrypted)
ravi vault list --json                     # List all entries
ravi vault get <uuid> --json               # Show entry (decrypted)
ravi vault create example.com              # Create (auto-generates password)
ravi vault create example.com --username me@email.com --password 'mypass'
ravi vault edit <uuid> --password 'new'    # Edit fields
ravi vault delete <uuid>                   # Delete entry
ravi vault generate --length 32            # Generate without storing
```

### Workflow: Signing Up for a Service

1. Get your Ravi email: `ravi get email --json | jq -r '.email'`
2. Get your Ravi phone: `ravi get phone --json | jq -r '.phone_number'`
3. Fill out the signup form using these credentials
4. Wait for verification: `sleep 5 && ravi inbox sms --unread --json`
5. Extract OTP or verification link from the message
6. Complete the verification

### Workflow: Receiving 2FA Codes

```bash
# After triggering 2FA, wait and check inbox
sleep 5
ravi inbox sms --unread --json   # For SMS-based 2FA
ravi inbox email --unread --json # For email-based 2FA
```

See `.claude/skills/ravi-cli.md` for detailed usage instructions, or install the [Claude Code plugin](docs/claude-code-plugin.md) to use `ravi` from any Claude Code session.

## Commands

```bash
# Development
make build API_URL=https://ravi.app   # Build binary (API_URL required)
make test                              # Run tests
make lint                              # Check with golangci-lint
make lint-fix                          # Auto-fix lint issues
make clean                             # Remove build artifacts

# Cross-compilation
make build-all API_URL=https://ravi.app  # Build for all platforms
```

## Architecture

```
cmd/ravi/             # Entry point
internal/
├── api/              # HTTP client and API types
├── auth/             # Device code flow orchestration
├── config/           # Token/config file management
├── crypto/           # E2E encryption (Argon2id + NaCl SealedBox)
├── output/           # Human/JSON formatters
└── version/          # Build-time version info
pkg/cli/              # Cobra commands (inbox, vault, auth, etc.)
```

### Key Patterns

- **Output formatting**: All commands support `--json` flag for AI agent consumption
- **Token refresh**: API client automatically refreshes expired tokens
- **Build-time config**: API URL injected via ldflags (no runtime config needed)
- **E2E encryption**: `internal/crypto/` handles client-side encrypt/decrypt (Argon2id key derivation + NaCl SealedBox). Password fields are encrypted before API calls and decrypted after retrieval.

## Code Style

- Use `gofmt` formatting
- Follow Go idioms and effective Go guidelines
- Error wrapping with `fmt.Errorf("context: %w", err)`
- Conventional commits: `feat(scope):`, `fix(scope):`, `refactor(scope):`
