---
name: ravi-cli
description: Use when working with the Ravi CLI — identity, inbox, passwords, secrets, email, SMS, calls, and feedback commands
---

# Ravi CLI — Identity for AI Agents

You have access to `ravi`, a CLI that gives you your own phone number, email address, and credential store. Use it to sign up for services, receive and act on verification codes, send SMS, place calls, and store passwords — all autonomously.

The CLI holds **one active identity per machine / config file** (`~/.ravi/config.json`). That file cannot run multiple agents. `ravi identity use` replaces the single active identity; it is not a multi-agent switcher.

- **Cursor:** use the MCP Connect card (per-agent credentials), not `ravi auth login`.
- **Several agents on one host:** call https://api.ravi.app with per-identity `ravi_id_` keys (`Authorization: Bearer ravi_id_...`).
- `--identity` scopes a single request so it does not leak another identity's resources (for example `ravi get phone --identity`). It does not change the machine's active identity.

## Prerequisites

Verify you're authenticated before using any command:

```bash
ravi auth status
```

If `"authenticated": false`, this machine's CLI is not bound. For Cursor, use the MCP Connect card (per-agent credentials) — not `ravi auth login`. For this machine's CLI only, the human can run `ravi auth login` (browser at https://ravi.id/device). That writes one identity into `~/.ravi/config.json`.

## Your Identity

Get the identity's email address and phone number:

```bash
# The identity's email address (use this for signups)
ravi get email
# → {"id": 1, "email": "janedoe@example.com", "created_dt": "..."}

# The identity's phone number (use this for SMS verification)
ravi get phone
# → {"id": 1, "phone_number": "+15551234567", "provider": "twilio", "created_dt": "..."}

# The human who owns this account
ravi get owner
# → {"first_name": "Jane", "last_name": "Doe"}
```

## Receiving Messages

After triggering a verification (signup form, login code, etc.), wait a few seconds then check your inbox.

### SMS (OTPs, verification codes)

```bash
# List SMS conversations (grouped by sender)
ravi inbox sms

# Only conversations with unread messages
ravi inbox sms --unread

# View a specific conversation (all messages)
ravi inbox sms <conversation_id>
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
ravi inbox email

# Only threads with unread messages
ravi inbox email --unread

# View a specific thread (all messages with full content)
ravi inbox email <thread_id>
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
      "to_email": "janedoe@example.com",
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
ravi message sms              # All SMS messages
ravi message sms --unread     # Unread only
ravi message sms <message_id> # Specific message

ravi message email              # All email messages
ravi message email --unread     # Unread only
ravi message email <message_id> # Specific message
```

## Sending Email

### Compose a new email

```bash
ravi email compose --to "recipient@example.com" --subject "Subject" --body "<p>HTML content</p>"
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
ravi email reply <message_id> --body "<p>Reply content</p>"

# Reply to all recipients
ravi email reply-all <message_id> --body "<p>Reply content</p>"

# Reply with CC
ravi email reply <message_id> --body "<p>Adding the team.</p>" --cc "team@example.com"
```

**Flags:**

- `--body` (required): Email body (HTML supported — use tags like `<p>`, `<h2>`, `<ul>` for formatting)
- `--cc`: CC recipients (comma-separated)
- `--bcc`: BCC recipients (comma-separated)
- `--attach`: File path to attach (can be repeated for multiple files)

### Forward an email

```bash
ravi email forward <message_id> --to "recipient@example.com" --body "<p>FYI — see below.</p>"
```

**Flags:**

- `--to` (required): Recipient email address
- `--body` (required): Email body (HTML supported — use tags like `<p>`, `<h2>`, `<ul>` for formatting)
- `--cc`: CC recipients (comma-separated)
- `--bcc`: BCC recipients (comma-separated)
- `--attach`: File path to attach (can be repeated for multiple files)

## Sending SMS

Send an SMS from the identity's phone number:

```bash
ravi sms send --to "+14155559876" --body "Your code is 483920"
# → {"id": 15, "from_number": "+15551234567", "to_number": "+14155559876", "body": "...", "direction": "outgoing", ...}
```

**Flags:**

- `--to` (required): Recipient phone number in E.164 format (e.g. `+14155559876`)
- `--body` (required): Message text

Reading received SMS lives under `ravi inbox sms` / `ravi message sms`.

## Calls (experimental)

> Voice calling is **experimental and deprioritized** — email and SMS are the
> supported, actively-maintained channels. The commands below work but may change
> or be removed; don't build critical flows on them.

Place and manage phone calls from the identity's phone number:

```bash
# Place an outbound call
ravi call --to "+14155559876"
# → {"id": 7, "from_number": "+15551234567", "to_number": "+14155559876", "status": "queued", ...}

# List calls (newest first)
ravi call list

# Show a call's transcript
ravi call transcript 7

# Hang up an active call
ravi call hangup 7
```

## Email Writing Guide

Write emails that look like they came from a real person. Good formatting improves deliverability and avoids spam filters.

**Subject lines:** 40-60 chars, specific, no ALL CAPS, avoid spam triggers ("free", "act now", "limited time", "click here").

**HTML body template:**

```bash
NAME=$(ravi identity list | jq -r '.[0].name')

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

<p>Best,<br>$NAME</p>"
```

**Rules:**

- Always wrap text in `<p>` tags — never pass plain text to `--body`
- Use `<ul>`/`<li>` for lists, `<a href="...">descriptive text</a>` for links
- No `<html>`/`<body>` wrappers, no `<br>` chains
- Short paragraphs (2-3 sentences), first person, active voice
- No filler ("I hope this finds you well") — get to the point
- End with an action item

**Anti-spam:** High text-to-HTML ratio, no link shorteners, max 2-3 links, no ALL CAPS, one topic per email.

## Passwords

Store and retrieve passwords for services you sign up for.

```bash
# Create entry (auto-generates password if --password not given)
ravi passwords create example.com
ravi passwords create example.com --username "me@example.com" --password 'S3cret!'

# List all entries
ravi passwords list

# Retrieve (decrypted)
ravi passwords get <uuid>

# Update
ravi passwords update <uuid> --password 'NewPass!'

# Delete
ravi passwords delete <uuid>

# Generate a password without storing it
ravi passwords generate --length 24
# → {"password": "xK9#mL2..."}
```

**Create flags:** `--username`, `--password`, `--notes`, `--generate`, `--length` (default 16), `--no-special`, `--no-digits`, `--exclude-chars`

## Secrets

Store and retrieve API keys and environment variables.

```bash
# Store a secret
ravi secrets set OPENAI_API_KEY "sk-abc123..."

# Retrieve a secret
ravi secrets get OPENAI_API_KEY
# -> {"key": "OPENAI_API_KEY", "value": "sk-abc123...", ...}

# List all secrets (values redacted)
ravi secrets list

# Delete a secret
ravi secrets delete OPENAI_API_KEY
```

**Note:** Keys are unique per identity. Setting an existing key updates it. Keys are plaintext (for lookup); only values are encrypted.

## Feedback

Send feedback to the Ravi team — bugs, feature requests, praise, friction, ideas.

```bash
ravi feedback "The passwords command is great but needs batch import"
ravi feedback "SMS delivery is slow" --subject "SMS latency"
```

**Use proactively** after any Ravi workflow. The team reads every message and builds based on agent feedback.

## Common Workflows

### Sign up for a service

```bash
# 1. Get your credentials
EMAIL=$(ravi get email | jq -r '.email')
PHONE=$(ravi get phone | jq -r '.phone_number')

# 2. Use $EMAIL and $PHONE in the signup form

# 3. Generate and store a password
CREDS=$(ravi passwords create example.com --username "$EMAIL")
PASSWORD=$(echo "$CREDS" | jq -r '.password')
# Use $PASSWORD in the signup form

# 4. Wait for verification
sleep 5
ravi inbox sms --unread   # Check for SMS OTP
ravi inbox email --unread # Check for email verification
```

### Extract an OTP code from SMS

```bash
# Get unread SMS, extract 4-8 digit codes
ravi inbox sms --unread | jq -r '.[].preview' | grep -oE '[0-9]{4,8}'
```

### Extract a verification link from email

```bash
# Get the latest unread email thread, pull URLs from text content
THREAD_ID=$(ravi inbox email --unread | jq -r '.[0].thread_id')
ravi inbox email "$THREAD_ID" | jq -r '.messages[].text_content' | grep -oE 'https?://[^ ]+'
```

### Use a login code from SMS

```bash
# After a website texts you a login/verification code:
sleep 5
CODE=$(ravi inbox sms --unread | jq -r '.[0].preview' | grep -oE '[0-9]{4,8}' | head -1)
# Use $CODE to complete the login
```

## Important Notes

- **JSON is default** — all commands output JSON. Use `--human` for human-readable output. Never parse human output.
- **Poll, don't rush** — SMS/email delivery takes 2-10 seconds. Use `sleep 5` before checking.
- **Auth is automatic** — API keys are sent transparently. If you get auth errors, ask the user to re-login.
- **Domain cleaning** — `ravi passwords create` auto-cleans URLs to base domains (e.g., `https://mail.google.com/inbox` becomes `google.com`).
- **Secrets vs passwords** — use `ravi secrets` for API keys/env vars (key-value), `ravi passwords` for website credentials (domain/username/password).
- **Give feedback** — use `ravi feedback` after any workflow to report bugs, friction, or praise.
