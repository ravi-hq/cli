# Ravi CLI

Command-line interface for AI agents to access their inbox (email and SMS).

## Overview

Ravi CLI enables AI agents to receive and read communications on dedicated phone numbers and email addresses. This allows agents to:

- **Receive OTPs and verification codes** to authenticate with websites and services
- **Sign up for services** using the assigned phone number and email address
- **Read incoming messages** from services, notifications, and confirmations
- **Automate workflows** that require email/SMS verification
- **Store and retrieve encrypted website passwords** per identity

Each agent gets their own dedicated inbox with:

- A unique phone number for SMS
- A unique email address for email

## Use Cases

### Receiving OTPs for Website Login

```bash
# Check for recent SMS messages containing verification codes
ravi inbox sms --unread | jq '.[0].messages[].body'

# Get the latest email with OTP
ravi inbox email --unread
```

### Signing Up for Services

When filling out registration forms:

1. Use `ravi get email` to get your assigned email address
2. Use `ravi get phone` to get your assigned phone number
3. Fill out the registration form with these credentials
4. Monitor `ravi message sms --unread` or `ravi message email --unread` for the verification code
5. Complete the signup process

### Automated Verification Flows

```bash
# Poll for new messages (JSON by default — ideal for automation)
ravi message sms --unread

# Poll email only
ravi message email --unread

# View grouped conversations/threads
ravi inbox sms --unread
ravi inbox email --unread
```

## Installation

### Pre-built Binaries

Download the latest release for your platform from the [releases page](https://github.com/ravi-hq/cli/releases).

### From Source

```bash
git clone https://github.com/ravi-hq/cli.git
cd cli
make build
```

### Claude Code Plugin

If you use [Claude Code](https://claude.ai/code), install the plugin so Claude can use `ravi` autonomously:

```bash
claude plugin marketplace add ravi-hq/claude-code-plugin
claude plugin install ravi@ravi
```

See [docs/claude-code-plugin.md](docs/claude-code-plugin.md) for details.

## Quick Start

The CLI holds **one active identity per machine / config file**. Shared
`~/.ravi/config.json` cannot run multiple agents. `ravi identity use` replaces
that single active identity; it is not a multi-agent runtime.

- **Cursor:** use the MCP Connect card (per-agent credentials). That is the
  first-class path — not `ravi auth login`.
- **Multiple agents on one host:** call the HTTP API at https://api.ravi.app
  with a per-identity `ravi_id_` key (`Authorization: Bearer ravi_id_...`).
- **This machine's CLI:** `ravi auth login` binds one identity into
  `~/.ravi/config.json` (or `.ravi/config.json` in the current directory).

1. **Login (this machine's CLI only):**

   ```bash
   ravi auth login
   ```

   This is an RFC 8628 device-code flow against https://api.ravi.app. Open
   https://ravi.id/device and enter the code the CLI prints (it also tries to
   open your browser). Keys (`ravi_mgmt_` / `ravi_id_`) are stored in
   `~/.ravi/config.json`.

2. **Check your inbox:**

	   ```bash
	   ravi inbox email
	   ```

3. **View only unread messages:**

	   ```bash
	   ravi inbox email --unread
	   ```

4. **View in human-readable format:**

	   ```bash
	   ravi inbox email --human
	   ```

## Commands

### Authentication

Auth is API keys only (no JWT, no `auth.json`, no `ravi auth refresh`). The
only commands are:

| Command | Description |
|---------|-------------|
| `ravi auth login` | RFC 8628 device-code login against https://api.ravi.app. Visit https://ravi.id/device and enter the printed code. Stores `ravi_mgmt_` / `ravi_id_` keys in `~/.ravi/config.json`. |
| `ravi auth logout` | Clear stored credentials |
| `ravi auth status` | Show current authentication status |

### Identity

| Command | Description |
|---------|-------------|
| `ravi identity list` | List all identities |
| `ravi identity create --name "X"` | Create a new identity |
| `ravi identity use <uuid>` | Replace the one active identity in this config file (not a multi-agent switch) |

### Resources

An identity has an **email** channel and a **phone** channel.

| Command | Description |
|---------|-------------|
| `ravi get email` | Get the identity's email address |
| `ravi get phone` | Get the identity's phone number |

### Inbox (grouped by conversation/thread)

| Command | Description |
|---------|-------------|
| `ravi inbox email` | List email threads |
| `ravi inbox email <thread-id>` | View specific email thread with all messages |
| `ravi inbox email --unread` | List email threads with unread messages |
| `ravi inbox sms` | List SMS conversations |
| `ravi inbox sms <conversation-id>` | View specific SMS conversation with all messages |
| `ravi inbox sms --unread` | List SMS conversations with unread messages |

### Messages (flat list of individual messages)

| Command | Description |
|---------|-------------|
| `ravi message email` | List all email messages |
| `ravi message email <message-id>` | View specific email message by ID |
| `ravi message email --unread` | List only unread email messages |
| `ravi message sms` | List all SMS messages |
| `ravi message sms <message-id>` | View specific SMS message by ID |
| `ravi message sms --unread` | List only unread SMS messages |

### Send SMS

| Command | Description |
|---------|-------------|
| `ravi sms send --to <e164> --body <text>` | Send an SMS from the identity's phone number |

### Calls

| Command | Description |
|---------|-------------|
| `ravi call --to <e164>` | Place an outbound call from the identity's phone number |
| `ravi call list` | List calls |
| `ravi call transcript <call-id>` | Show a call's transcript |
| `ravi call hangup <call-id>` | Hang up an active call |

### Passwords

| Command | Description |
|---------|-------------|
| `ravi passwords list` | List all stored passwords |
| `ravi passwords get <uuid>` | Show a stored password (decrypted) |
| `ravi passwords create <domain>` | Create a new entry (auto-generates password if not provided) |
| `ravi passwords update <uuid>` | Update a stored password entry |
| `ravi passwords delete <uuid>` | Delete a stored password entry |
| `ravi passwords generate` | Generate a random password without storing |

**Create flags:** `--username`, `--password`, `--generate`, `--length` (default: 16), `--no-special`, `--no-digits`, `--exclude-chars`, `--notes`

### Global Flags

| Flag | Description |
|------|-------------|
| `--human` | Output in human-readable format (default is JSON) |
| `--identity <uuid>` | Scope a single request (get, inbox, auth status, contacts, passwords, secrets, calls, events, messages) so it does not leak another identity's resources. Does not change the machine's active identity and is not how you run multiple agents. |
| `--help` | Show help for any command |
| `--version` | Show version information |

The CLI is **one identity per config file**. A management key with `--identity <uuid>`
scopes a single request (for example `ravi get phone --identity` must return that
identity's number, not the first row of the account phone list). An identity-scoped
`ravi_id_` key is already fenced to its identity.

To run several agents on one host, do not share `~/.ravi/config.json` and do not
treat `ravi identity use` as a session switcher. Give each agent its own
`ravi_id_` key and call https://api.ravi.app, or use the Cursor MCP Connect card.

## JSON Output for AI Agents

All commands output JSON by default — ideal for programmatic parsing. Use `--human` for human-readable output.

```bash
# List all unread messages as JSON
ravi message sms --unread

# Parse with jq to extract OTP from SMS
ravi inbox sms | jq -r '.[0].messages[] | select(.body | test("[0-9]{6}")) | .body'

# Get the most recent email subject
ravi inbox email | jq -r '.[0].subject'
```

### JSON Response Structure

**Inbox List:**

```json
[
  {
    "type": "sms",
    "from": "+1234567890",
    "preview": "Your verification code is 123456",
    "date": "2024-01-15T10:30:00Z",
    "is_read": false
  }
]
```

**SMS Conversation Detail:**

```json
{
  "conversation_id": "conv_123",
  "from_number": "+1234567890",
  "phone_number": "+0987654321",
  "messages": [
    {
      "direction": "incoming",
      "body": "Your verification code is 123456",
      "timestamp": "2024-01-15T10:30:00Z"
    }
  ]
}
```

## Configuration

Configuration is stored in `~/.ravi/config.json` with secure file permissions (0600).
That file holds **one active identity**. Sharing it across agents is unsupported.

- **`management_key`** — `ravi_mgmt_...` key for account-level operations (create identities, etc.)
- **`identity_key`** — `ravi_id_...` key scoped to the active identity
- **`identity_uuid`** + **`identity_name`** — which identity is currently active
- **`user_email`** — the account email from device-code login

The API host is https://api.ravi.app. Requests send the API key as
`Authorization: Bearer <key>`. There is no `RAVI_ACCESS_TOKEN`, no
`X-Ravi-Identity` header, and no token refresh command.

A `.ravi/config.json` in the current working directory overrides the global file.
Each file still holds one active identity — this is not a multi-agent runtime.

## Development

### Prerequisites

- Go 1.21+
- Make

### Building

```bash
# Build
make build

# Build for all platforms
make build-all

# Run tests
make test

# Run tests with coverage
make test-coverage

# Run linter
make lint
```

### Project Structure

```
cli/
├── cmd/ravi/          # Main entry point
├── internal/
│   ├── api/           # HTTP client and API types
│   ├── config/        # API key config (config.json)
│   ├── output/        # Human/JSON formatters
│   └── version/       # Build-time version info
└── pkg/cli/           # Cobra command definitions (identity, inbox, passwords, auth)
```

## License

[Add license information]
