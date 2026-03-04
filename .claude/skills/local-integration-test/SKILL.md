---
name: local-integration-test
description: Use when testing CLI changes against the local backend before committing. Covers building with local endpoint, authenticating, and driving every command group with verification.
---

# Local Integration Test — CLI

End-to-end integration test of the CLI against a locally-running backend. Builds the binary
with `localhost` API URL, authenticates, and drives every command group to verify the full
stack works before committing.

## Prerequisites

```bash
# 1. Backend Docker containers must be running (from the backend repo)
docker ps | grep -E "ravi_backend|ravi_db|ravi_cache"
# Must see all three: ravi_backend, ravi_db, ravi_cache

# 2. Backend dev server running on localhost:8000
curl -s -o /dev/null -w "%{http_code}" http://localhost:8000/
# Should return 200 (landing page) or 301

# 3. Go installed
go version
```

## Step 1: Build with Local Endpoint

```bash
cd /path/to/cli
make build API_URL=http://localhost:8000
```

API URL is baked in at build time via ldflags (`internal/version.APIBaseURL`). The binary
at `./bin/ravi` will always talk to localhost after this.

## Step 2: Authenticate

Existing tokens in `~/.ravi/auth.json` are signed by the production server's `SECRET_KEY`.
The local server has a different key, so they're always invalid. **You must re-login.**

```bash
./bin/ravi auth login
# Opens browser → http://localhost:8000/device/...
# Complete the device flow in the browser
# Enter PIN when prompted (derived keypair saved to auth.json)
```

After login, verify:
```bash
./bin/ravi auth status --json
# Expected: {"authenticated": true, "email": "...", "identity": "...", "identity_uuid": "..."}
```

Verify bound tokens exist (identity-scoped JWT):
```bash
cat ~/.ravi/config.json | python3 -m json.tool
# Must have: bound_access_token, bound_refresh_token, identity_uuid, identity_name
```

**Key concept:** Login does two things: (1) device flow → global tokens, (2) identity bind
→ bound tokens with `identity_uuid` JWT claim. The keypair derived from the PIN is persisted
to `auth.json` so no PIN prompt is needed for any subsequent command.

## Step 3: Test Matrix

Run each group sequentially. Each step depends on prior steps being healthy.

### 3.1 Identity Commands

```bash
# List identities
./bin/ravi identity list --json
# Expected: JSON array of identity objects with uuid, name, inbox, phone

# Create identity (requires active subscription)
./bin/ravi identity create --name "Test Agent" --json
# Expected: New identity with uuid, name, inbox
# If 402: "Active paid subscription required." — billing gate working, give account a subscription

# Switch to new identity (MUST use full UUID, prefix matching not supported)
./bin/ravi identity use <full-uuid-from-create> --json
# Expected: {"identity_name": "Test Agent", "identity_uuid": "...", "status": "active"}

# Verify switch
./bin/ravi auth status --json
# identity field should now show "Test Agent"

# Switch back to original identity
./bin/ravi identity use <original-uuid> --json
```

**Verify:** `~/.ravi/config.json` should update `bound_access_token` on each identity switch
(new JWT with different `identity_uuid` claim).

### 3.2 Get Email/Phone

```bash
./bin/ravi get email --json
# Expected: {"id": N, "email": "...@local.raviapp.com", "created_dt": "..."}

./bin/ravi get phone --json
# Expected (no Twilio locally): {"error": "no phone number assigned"} with exit code 1
```

### 3.3 Inbox Commands

```bash
# Email inbox (will be empty locally unless you've sent test webhooks)
./bin/ravi inbox email --json
# Expected: [] (empty array)

./bin/ravi inbox email --unread --json
# Expected: []

# SMS inbox
./bin/ravi inbox sms --json
# Expected: []

# Flat message list
./bin/ravi message sms --json
# Expected: []

# Specific email by invalid ID (404 test)
./bin/ravi message email nonexistent-id --json
# Expected: {"error": "API error: Not found."} with exit code 1
```

### 3.4 Passwords CRUD (E2E Encrypted)

This tests the full E2E encryption round-trip: encrypt client-side → store ciphertext
on server → decrypt client-side on retrieval.

```bash
# Create
./bin/ravi passwords create testsite.com \
  --username "user@example.com" --password "SuperSecret123!" --notes "test" --json
# Expected: Entry with e2e:: prefixed username, password, notes
# Save the UUID from the response

# Get (decrypt)
./bin/ravi passwords get <uuid> --json
# Expected: All fields decrypted — username: "user@example.com", password: "SuperSecret123!"
# This confirms E2E round-trip works WITHOUT a PIN prompt

# List
./bin/ravi passwords list --json
# Expected: Username decrypted, password still shows e2e:: (intentional for list view)

# Edit
./bin/ravi passwords edit <uuid> --password "NewPassword!" --notes "updated" --json
# Expected: Updated entry with new e2e:: ciphertext, updated_dt changed

# Verify edit
./bin/ravi passwords get <uuid> --json
# Expected: password: "NewPassword!", notes: "updated"

# Generate (no storage)
./bin/ravi passwords generate --length 32 --json
# Expected: {"password": "<random-32-char-string>"}

# Delete
./bin/ravi passwords delete <uuid> --json
# Expected: {"status": "deleted"}
```

### 3.5 Secrets CRUD (E2E Encrypted)

```bash
# Create
./bin/ravi secrets set MY_API_KEY "sk-test-12345" --json
# Expected: Entry with e2e:: prefixed value

# Get (decrypt)
./bin/ravi secrets get MY_API_KEY --json
# Expected: value: "sk-test-12345" (decrypted)

# Upsert (update existing key)
./bin/ravi secrets set MY_API_KEY "sk-updated-67890" --json
# Expected: Same UUID, updated value, updated_dt changed
# This should NOT error — the CLI checks for existing key and patches

# Verify upsert
./bin/ravi secrets get MY_API_KEY --json
# Expected: value: "sk-updated-67890"

# List
./bin/ravi secrets list --json
# Expected: Array with decrypted values

# Delete (takes UUID, not key name)
./bin/ravi secrets delete <uuid> --json
# Expected: {"status": "deleted"}
```

### 3.6 Error Cases

```bash
# Non-existent password
./bin/ravi passwords get 00000000-0000-0000-0000-000000000000 --json
# Expected: {"error": "API error: No PasswordEntry matches the given query."}

# Non-existent secret by key
./bin/ravi secrets get NONEXISTENT_KEY --json
# Expected: {"error": "secret not found: NONEXISTENT_KEY"}

# Non-existent secret delete
./bin/ravi secrets delete 00000000-0000-0000-0000-000000000000 --json
# Expected: {"error": "API error: No SecretEntry matches the given query."}
```

### 3.7 Cross-Identity Isolation

```bash
# Switch to a different identity (create one if needed)
./bin/ravi identity use <other-identity-uuid> --json

# Verify it sees no data from the first identity
./bin/ravi passwords list --json   # Expected: []
./bin/ravi secrets list --json     # Expected: []

# Switch back
./bin/ravi identity use <original-uuid> --json
```

## Step 4: Run Unit Tests

```bash
make test
# All tests must pass (currently 134 tests)
```

## Step 5: Cleanup

Delete any test identities/entries created during testing:

```bash
# Delete test identity if created
# (no CLI delete command — leave it or clean up via backend admin)

# Verify no leftover test data
./bin/ravi passwords list --json
./bin/ravi secrets list --json
```

## Limitations

| Limitation | Reason |
|-----------|--------|
| Cannot send emails | No Resend credentials locally |
| Cannot provision phones | No Twilio credentials locally |
| SSE streams may not work | Requires `REDIS_URL` set on the dev server |
| Cannot test SMS sending | No phone number provisioned locally |

## Common Failures

| Symptom | Cause | Fix |
|---------|-------|-----|
| "Token is invalid" on every command | Tokens from production, not local | Re-login: `./bin/ravi auth login` |
| "API URL not configured" | Built without `API_URL` | Rebuild: `make build API_URL=http://localhost:8000` |
| 402 on identity create | No subscription in local DB | Add subscription via backend admin/shell |
| "identity not found" on `use` | Used UUID prefix instead of full UUID | Pass the complete UUID |
| Connection refused | Backend not running on :8000 | Start dev server or check Docker |
| E2E decryption fails | Wrong PIN at login or corrupted auth.json | Delete `~/.ravi/auth.json` and re-login |
| `secrets set` fails on duplicate key | Using old CLI without upsert fix | Rebuild the CLI |
