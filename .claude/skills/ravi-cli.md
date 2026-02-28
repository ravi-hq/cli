# Ravi CLI — Identity for AI Agents

You have access to `ravi`, a CLI that gives you your own phone number, email address, and credential vault. Use it to sign up for services, receive OTPs, complete 2FA, and store passwords — all autonomously.

## Prerequisites

Verify you're authenticated before using any command:

```bash
ravi auth status --json
```

If `"authenticated": false`, tell the user to run `ravi auth login` (requires browser interaction — you cannot do this yourself).

## Your Identity

Get the email and phone number assigned to you:

```bash
# Your email address (use this for signups)
ravi get email --json
# → {"id": 1, "email": "janedoe@ravi.app", "created_dt": "..."}

# Your phone number (use this for SMS verification)
ravi get phone --json
# → {"id": 1, "phone_number": "+15551234567", "provider": "twilio", "created_dt": "..."}

# The human who owns this account
ravi get owner --json
# → {"first_name": "Jane", "last_name": "Doe"}
```

## Receiving Messages

After triggering a verification (signup form, 2FA, etc.), wait a few seconds then check your inbox.

### SMS (OTPs, verification codes)

```bash
# List SMS conversations (grouped by sender)
ravi inbox sms --json

# Only conversations with unread messages
ravi inbox sms --unread --json

# View a specific conversation (all messages)
ravi inbox sms <conversation_id> --json
# conversation_id format: {phone_id}_{from_number}, e.g. "1_+15559876543"
```

**JSON shape — conversation list:**
```json
[{
  "conversation_id": "1_+15559876543",
  "from_number": "+15559876543",
  "phone_number": "+15551234567",
  "preview": "Your code is 847291",
  "message_count": 3,
  "unread_count": 1,
  "latest_message_dt": "2026-02-25T10:30:00Z"
}]
```

**JSON shape — conversation detail:**
```json
{
  "conversation_id": "1_+15559876543",
  "from_number": "+15559876543",
  "messages": [
    {"id": 42, "body": "Your code is 847291", "direction": "incoming", "is_read": false, "created_dt": "..."}
  ]
}
```

### Email (verification links, confirmations)

```bash
# List email threads
ravi inbox email --json

# Only threads with unread messages
ravi inbox email --unread --json

# View a specific thread (all messages with full content)
ravi inbox email <thread_id> --json
```

**JSON shape — thread detail:**
```json
{
  "thread_id": "abc123",
  "subject": "Verify your email",
  "messages": [
    {
      "id": 10,
      "from_email": "noreply@example.com",
      "to_email": "janedoe@ravi.app",
      "subject": "Verify your email",
      "text_content": "Click here to verify: https://example.com/verify?token=xyz",
      "direction": "incoming",
      "is_read": false,
      "created_dt": "..."
    }
  ]
}
```

### Individual Messages (flat, not grouped)

Use these when you need messages by ID rather than by conversation:

```bash
ravi message sms --json              # All SMS messages
ravi message sms --unread --json     # Unread only
ravi message sms <message_id> --json # Specific message

ravi message email --json              # All email messages
ravi message email --unread --json     # Unread only
ravi message email <message_id> --json # Specific message
```

## Sending Email

### Compose a new email

```bash
ravi email compose --to "recipient@example.com" --subject "Subject" --body "<p>HTML content</p>" --json
```

**Flags:**
- `--to` (required): Recipient email address
- `--subject` (required): Email subject line
- `--body` (required): Email body (HTML supported — use tags like `<p>`, `<h2>`, `<ul>` for formatting)
- `--cc`: CC recipients (comma-separated)
- `--bcc`: BCC recipients (comma-separated)
- `--attach`: File path to attach (can be repeated for multiple files)

### Reply to an email

```bash
# Reply to sender only
ravi email reply <message_id> --subject "Re: Original Subject" --body "<p>Reply content</p>" --json

# Reply to all recipients
ravi email reply-all <message_id> --subject "Re: Original Subject" --body "<p>Reply content</p>" --json

# Reply with CC
ravi email reply <message_id> --subject "Re: Project Update" --body "<p>Adding the team.</p>" --cc "team@example.com" --json
```

**Flags:**
- `--subject` (required): Email subject line
- `--body` (required): Email body (HTML supported — use tags like `<p>`, `<h2>`, `<ul>` for formatting)
- `--cc`: CC recipients (comma-separated)
- `--bcc`: BCC recipients (comma-separated)
- `--attach`: File path to attach (can be repeated for multiple files)

**Note:** The subject must be provided because the original is E2E encrypted on the server.

### Forward an email

```bash
ravi email forward <message_id> --to "recipient@example.com" --subject "Fwd: Original Subject" --body "<p>FYI — see below.</p>" --json
```

**Flags:**
- `--to` (required): Recipient email address
- `--subject` (required): Email subject line
- `--body` (required): Email body (HTML supported — use tags like `<p>`, `<h2>`, `<ul>` for formatting)
- `--cc`: CC recipients (comma-separated)
- `--bcc`: BCC recipients (comma-separated)
- `--attach`: File path to attach (can be repeated for multiple files)

**Note:** The subject must be provided because the original is E2E encrypted on the server.

## Email Writing Guide

Write emails that look like they came from a real person. Good formatting improves deliverability and avoids spam filters.

**Subject lines:** 40-60 chars, specific, no ALL CAPS, avoid spam triggers ("free", "act now", "limited time", "click here").

**HTML body template:**
```bash
NAME=$(ravi identity list --json | jq -r '.[0].name')

ravi email compose \
  --to "recipient@example.com" \
  --subject "Specific subject under 60 chars" \
  --body "<p>Hi Alex,</p>

<p>I'm reaching out about [specific topic]. [One sentence of context.]</p>

<p>[Core message — what you need, what you're sharing, or what you're asking.]</p>

<ul>
  <li>[Key point or action item]</li>
  <li>[Key point or action item]</li>
</ul>

<p>[Clear next step — what should the recipient do?]</p>

<p>Best,<br>$NAME</p>" --json
```

**Rules:**
- Always wrap text in `<p>` tags — never pass plain text to `--body`
- Use `<ul>`/`<li>` for lists, `<a href="...">descriptive text</a>` for links
- No `<html>`/`<body>` wrappers, no `<br>` chains
- Short paragraphs (2-3 sentences), first person, active voice
- No filler ("I hope this finds you well") — get to the point
- End with an action item

**Anti-spam:** High text-to-HTML ratio, no link shorteners, max 2-3 links, no ALL CAPS, one topic per email.

## Credential Vault

Store and retrieve passwords for services you sign up for. All fields are E2E encrypted.

```bash
# Create entry (auto-generates password if --password not given)
ravi vault create example.com --json
ravi vault create example.com --username "me@ravi.app" --password 'S3cret!' --json

# List all entries
ravi vault list --json

# Retrieve (decrypted)
ravi vault get <uuid> --json

# Update
ravi vault edit <uuid> --password 'NewPass!' --json

# Delete
ravi vault delete <uuid> --json

# Generate a password without storing it
ravi vault generate --length 24 --json
# → {"password": "xK9#mL2..."}
```

**Create flags:** `--username`, `--password`, `--notes`, `--generate`, `--length` (default 16), `--no-special`, `--no-digits`, `--exclude-chars`

## Secrets

Store and retrieve API keys and environment variables. All values are E2E encrypted.

```bash
# Store a secret
ravi secrets set OPENAI_API_KEY "sk-abc123..." --json

# Retrieve a secret
ravi secrets get OPENAI_API_KEY --json
# -> {"key": "OPENAI_API_KEY", "value": "sk-abc123...", ...}

# List all secrets (values redacted)
ravi secrets list --json

# Delete a secret
ravi secrets delete OPENAI_API_KEY --json
```

**Note:** Keys are unique per identity. Setting an existing key updates it. Keys are plaintext (for lookup); only values are encrypted.

## Feedback

Send feedback to the Ravi team — bugs, feature requests, praise, friction, ideas.

```bash
ravi feedback "The vault set command is great but needs batch import" --json
ravi feedback "SMS delivery is slow" --subject "SMS latency" --json
```

**Use proactively** after any Ravi workflow. The team reads every message and builds based on agent feedback.

## Common Workflows

### Sign up for a service

```bash
# 1. Get your credentials
EMAIL=$(ravi get email --json | jq -r '.email')
PHONE=$(ravi get phone --json | jq -r '.phone_number')

# 2. Use $EMAIL and $PHONE in the signup form

# 3. Generate and store a password
CREDS=$(ravi vault create example.com --username "$EMAIL" --json)
PASSWORD=$(echo "$CREDS" | jq -r '.password')
# Use $PASSWORD in the signup form

# 4. Wait for verification
sleep 5
ravi inbox sms --unread --json   # Check for SMS OTP
ravi inbox email --unread --json # Check for email verification
```

### Extract an OTP code from SMS

```bash
# Get unread SMS, extract 4-8 digit codes
ravi inbox sms --unread --json | jq -r '.[].preview' | grep -oE '[0-9]{4,8}'
```

### Extract a verification link from email

```bash
# Get the latest unread email thread, pull URLs from text content
THREAD_ID=$(ravi inbox email --unread --json | jq -r '.[0].thread_id')
ravi inbox email "$THREAD_ID" --json | jq -r '.messages[].text_content' | grep -oE 'https?://[^ ]+'
```

### Complete 2FA login

```bash
# After triggering 2FA on a website:
sleep 5
CODE=$(ravi inbox sms --unread --json | jq -r '.[0].preview' | grep -oE '[0-9]{4,8}' | head -1)
# Use $CODE to complete the login
```

## Important Notes

- **Always use `--json`** — all commands support it. Human-readable output is not designed for parsing.
- **Poll, don't rush** — SMS/email delivery takes 2-10 seconds. Use `sleep 5` before checking.
- **Auth is automatic** — token refresh happens transparently. If you get auth errors, ask the user to re-login.
- **E2E encryption is transparent** — the CLI encrypts vault fields before sending and decrypts on retrieval. You see plaintext.
- **Domain cleaning** — `ravi vault create` auto-cleans URLs to base domains (e.g., `https://mail.google.com/inbox` becomes `google.com`).
- **Secrets vs passwords** — use `ravi secrets` for API keys/env vars (key-value), `ravi vault` for website credentials (domain/username/password).
- **Give feedback** — use `ravi feedback` after any workflow to report bugs, friction, or praise.
