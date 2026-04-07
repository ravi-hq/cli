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

API keys from the production server will not work with the local server. **You must re-login.**

```bash
./bin/ravi auth login
# Opens browser → http://localhost:8000/device/...
# Complete the device flow in the browser
```

After login, verify:
```bash
./bin/ravi auth status
# Expected: {"authenticated": true, "email": "...", "identity": "...", "identity_uuid": "..."}
```

Verify config:
```bash
cat ~/.ravi/config.json | python3 -m json.tool
# Must have: management_key, identity_key, identity_uuid, identity_name
```

**Key concept:** Login does two things: (1) device flow → management key, (2) identity bind
→ identity key scoped to the active identity.

## Step 3: Test Matrix

Run each group sequentially. Each step depends on prior steps being healthy.

### 3.1 Identity Commands

```bash
# List identities
./bin/ravi identity list
# Expected: JSON array of identity objects with uuid, name, inbox, phone

# Create identity (requires active subscription)
./bin/ravi identity create --name "Test Agent"
# Expected: New identity with uuid, name, inbox
# If 402: "Active paid subscription required." — billing gate working, give account a subscription

# Switch to new identity (MUST use full UUID, prefix matching not supported)
./bin/ravi identity use <full-uuid-from-create>
# Expected: {"identity_name": "Test Agent", "identity_uuid": "...", "status": "active"}

# Verify switch
./bin/ravi auth status
# identity field should now show "Test Agent"

# Switch back to original identity
./bin/ravi identity use <original-uuid>
```

**Verify:** `~/.ravi/config.json` should update `identity_key` on each identity switch
(new key scoped to the new identity).

### 3.2 Get Email/Phone

```bash
./bin/ravi get email
# Expected: {"id": N, "email": "...@local.ravi.id", "created_dt": "..."}

./bin/ravi get phone
# Expected (no Twilio locally): {"error": "no phone number assigned"} with exit code 1
```

### 3.3 Inbox Commands

```bash
# Email inbox (will be empty locally unless you've sent test webhooks)
./bin/ravi inbox email
# Expected: [] (empty array)

./bin/ravi inbox email --unread
# Expected: []

# SMS inbox
./bin/ravi inbox sms
# Expected: []

# Flat message list
./bin/ravi message sms
# Expected: []

# Specific email by invalid ID (404 test)
./bin/ravi message email nonexistent-id
# Expected: {"error": "API error: Not found."} with exit code 1
```

### 3.4 Passwords CRUD

```bash
# Create
./bin/ravi passwords create testsite.com \
  --username "user@example.com" --password "SuperSecret123!" --notes "test"
# Save the UUID from the response

# Get
./bin/ravi passwords get <uuid>
# Expected: All fields readable — username: "user@example.com", password: "SuperSecret123!"

# List
./bin/ravi passwords list

# Update
./bin/ravi passwords update <uuid> --password "NewPassword!" --notes "updated"
# Expected: Updated entry with updated_dt changed

# Verify edit
./bin/ravi passwords get <uuid>
# Expected: password: "NewPassword!", notes: "updated"

# Generate (no storage)
./bin/ravi passwords generate --length 32
# Expected: {"password": "<random-32-char-string>"}

# Delete
./bin/ravi passwords delete <uuid>
# Expected: {"status": "deleted"}
```

### 3.5 Secrets CRUD

```bash
# Create
./bin/ravi secrets set MY_API_KEY "sk-test-12345"
# Expected: Entry stored

# Get
./bin/ravi secrets get MY_API_KEY
# Expected: value: "sk-test-12345"

# Upsert (update existing key)
./bin/ravi secrets set MY_API_KEY "sk-updated-67890"
# Expected: Same UUID, updated value, updated_dt changed
# This should NOT error — the CLI checks for existing key and patches

# Verify upsert
./bin/ravi secrets get MY_API_KEY
# Expected: value: "sk-updated-67890"

# List
./bin/ravi secrets list
# Expected: Array of secrets

# Delete (takes UUID, not key name)
./bin/ravi secrets delete <uuid>
# Expected: {"status": "deleted"}
```

### 3.6 Error Cases

```bash
# Non-existent password
./bin/ravi passwords get 00000000-0000-0000-0000-000000000000
# Expected: {"error": "API error: No PasswordEntry matches the given query."}

# Non-existent secret by key
./bin/ravi secrets get NONEXISTENT_KEY
# Expected: {"error": "secret not found: NONEXISTENT_KEY"}

# Non-existent secret delete
./bin/ravi secrets delete 00000000-0000-0000-0000-000000000000
# Expected: {"error": "API error: No SecretEntry matches the given query."}
```

### 3.7 Cross-Identity Isolation

```bash
# Switch to a different identity (create one if needed)
./bin/ravi identity use <other-identity-uuid>

# Verify it sees no data from the first identity
./bin/ravi passwords list   # Expected: []
./bin/ravi secrets list     # Expected: []

# Switch back
./bin/ravi identity use <original-uuid>
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
./bin/ravi passwords list
./bin/ravi secrets list
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
| "401 Unauthorized" on every command | API keys from production, not local | Re-login: `./bin/ravi auth login` |
| "API URL not configured" | Built without `API_URL` | Rebuild: `make build API_URL=http://localhost:8000` |
| 402 on identity create | No subscription in local DB | Add subscription via backend admin/shell |
| "identity not found" on `use` | Used UUID prefix instead of full UUID | Pass the complete UUID |
| Connection refused | Backend not running on :8000 | Start dev server or check Docker |
| `secrets set` fails on duplicate key | Using old CLI without upsert fix | Rebuild the CLI |
