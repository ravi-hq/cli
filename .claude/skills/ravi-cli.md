# Ravi CLI - AI Agent Inbox Access

This skill teaches you how to use the Ravi CLI to access your dedicated email and SMS inbox.

## What is Ravi CLI?

Ravi CLI provides AI agents with their own phone number and email address. You can:
- Receive SMS messages (including OTPs and verification codes)
- Receive emails (including signup confirmations and verification links)
- Use these credentials to sign up for websites and services
- Authenticate to services that require phone/email verification
- Store and retrieve E2E-encrypted passwords per identity

## Prerequisites

Before using Ravi CLI, ensure you are authenticated:
```bash
ravi auth status
```

If not authenticated, run:
```bash
ravi auth login
```

## Common Workflows

### 1. Checking for OTP/Verification Codes

When you need to receive a verification code (e.g., after signing up for a service):

```bash
# Check for new SMS messages
ravi inbox sms --unread --json

# Check for new emails
ravi inbox email --unread --json

# Check unified inbox for all new messages
ravi inbox list --unread --json
```

**Extracting OTP from SMS:**
```bash
# Get all unread SMS and look for 6-digit codes
ravi inbox sms --unread --json | jq -r '.[] | .messages[] | select(.body | test("[0-9]{6}")) | .body'
```

### 2. Signing Up for Services

When filling out a registration form that requires email/phone:

1. **Get your Ravi email address:**
   ```bash
   ravi get email --json | jq -r '.email'
   ```

2. **Get your Ravi phone number:**
   ```bash
   ravi get phone --json | jq -r '.phone_number'
   ```

3. **Use these credentials in the registration form**

4. **After submitting the form, check for verification:**
   ```bash
   # Wait a few seconds, then check for the verification message
   ravi inbox list --unread --json
   ```

### 3. Two-Factor Authentication (2FA)

When logging into a website that sends 2FA codes:

```bash
# For SMS-based 2FA
ravi inbox sms --unread --json

# For email-based 2FA
ravi inbox email --unread --json
```

### 4. Viewing Message Details

**View a specific SMS conversation:**
```bash
# List conversations first
ravi inbox sms --json

# Then view specific conversation
ravi inbox sms <conversation_id> --json
```

**View a specific email thread:**
```bash
# List threads first
ravi inbox email --json

# Then view specific thread
ravi inbox email <thread_id> --json
```

## Command Reference

### Authentication Commands
| Command | Description |
|---------|-------------|
| `ravi auth login` | Authenticate (opens browser) |
| `ravi auth logout` | Clear credentials |
| `ravi auth status` | Show auth status and account email |
| `ravi auth status --json` | Get auth info as JSON |

### Resource Commands
| Command | Description |
|---------|-------------|
| `ravi get phone` | Get your assigned Ravi phone number |
| `ravi get email` | Get your assigned Ravi email address |
| `ravi get phone --json` | Get phone as JSON |
| `ravi get email --json` | Get email as JSON |

### Inbox Commands (grouped by conversation/thread)
| Command | Description |
|---------|-------------|
| `ravi inbox list` | List all messages (SMS + email) |
| `ravi inbox list --unread` | Only unread messages |
| `ravi inbox list --type sms` | Only SMS messages |
| `ravi inbox list --type email` | Only email messages |
| `ravi inbox sms` | List SMS conversations |
| `ravi inbox sms <id>` | View SMS conversation |
| `ravi inbox email` | List email threads |
| `ravi inbox email <id>` | View email thread |

### Message Commands (individual messages)
| Command | Description |
|---------|-------------|
| `ravi message sms` | List all SMS messages (flat) |
| `ravi message sms <id>` | View specific SMS message by ID |
| `ravi message sms --unread` | Only unread SMS messages |
| `ravi message email` | List all email messages (flat) |
| `ravi message email <id>` | View specific email message by ID |
| `ravi message email --unread` | Only unread email messages |

### Important Flags
- `--json` - Output as JSON (always use this for parsing)
- `--unread` - Filter to unread messages only

## Best Practices

1. **Always use `--json` flag** when you need to parse the output programmatically

2. **Poll for new messages** after triggering a verification:
   ```bash
   # Wait a moment, then check
   sleep 5 && ravi inbox list --unread --json
   ```

3. **Use specific filters** to reduce noise:
   ```bash
   # If expecting SMS OTP, filter to SMS only
   ravi inbox list --type sms --unread --json
   ```

4. **Check both SMS and email** - some services send to either:
   ```bash
   ravi inbox list --unread --json
   ```

## Example: Complete Signup Flow

```bash
# 1. Get your Ravi email and phone
EMAIL=$(ravi get email --json | jq -r '.email')
PHONE=$(ravi get phone --json | jq -r '.phone_number')
echo "Use this email for signup: $EMAIL"
echo "Use this phone for signup: $PHONE"

# 2. [Fill out the signup form with these credentials]

# 3. Wait for verification email/SMS
sleep 10

# 4. Check for verification
ravi inbox list --unread --json

# 5. Extract verification link or code from the email
ravi inbox email <thread_id> --json | jq -r '.messages[].text_content'

# Or extract OTP code from SMS
ravi inbox sms <conversation_id> --json | jq -r '.messages[].body'
```

### 5. Managing Passwords

Store credentials for services you've signed up for:

```bash
# After signing up for a service, store the credentials
ravi passwords create example.com --username "$EMAIL" --password 'the-password-used'

# Or auto-generate a password during signup
ravi passwords create example.com
# Outputs: Generated password: xK9#mL2...  (use this in the signup form)

# Retrieve stored credentials later
ravi passwords list --json
ravi passwords get <uuid> --json

# Update a password
ravi passwords edit <uuid> --password 'new-password'

# Generate a password without storing it
ravi passwords generate --length 24 --json | jq -r '.password'
```

**Note:** URL inputs are automatically cleaned to domains (e.g. `https://mail.google.com/inbox` → `google.com`). Username defaults to your identity email if not specified. Password is auto-generated if not provided.

### Password Commands
| Command | Description |
|---------|-------------|
| `ravi passwords list` | List all stored passwords |
| `ravi passwords get <uuid>` | Show a stored password (decrypted) |
| `ravi passwords create <domain>` | Create a new password entry |
| `ravi passwords edit <uuid>` | Edit a stored password entry |
| `ravi passwords delete <uuid>` | Delete a stored password entry |
| `ravi passwords generate` | Generate a random password |

**Create flags:** `--username`, `--password`, `--generate`, `--length` (default: 16), `--no-special`, `--no-digits`, `--exclude-chars`, `--notes`

## Troubleshooting

**Not authenticated:**
```bash
ravi auth login
```

**No messages appearing:**
- Verify the correct email/phone was used
- Wait a few more seconds for delivery
- Check spam filters on the service side

**Token expired:**
The CLI automatically refreshes tokens. If issues persist:
```bash
ravi auth logout
ravi auth login
```
