# Claude Code Plugin

The Ravi CLI has a Claude Code plugin that teaches Claude Code
how to use `ravi`. When the plugin is installed, Claude Code can
autonomously sign up for services, receive OTPs, and manage
credentials on behalf of the user.

## How it works

The plugin provides a **skill file** — a structured document
that tells Claude Code:

- What commands are available
  (`ravi get email`, `ravi inbox sms`, `ravi vault create`, etc.)
- What the JSON output looks like for each command
- Common workflows (signup, OTP extraction, 2FA completion)
- Important conventions (always use `--json`, poll with `sleep 5`)

The plugin does **not** include the CLI itself.
Users must install `ravi` separately.

## Installation

### 1. Install the CLI

Download the latest release from the
[releases page](https://github.com/ravi-hq/cli/releases)
or build from source:

```bash
make build API_URL=https://ravi.app
```

### 2. Install the plugin

```bash
claude plugin marketplace add ravi-hq/claude-code-plugin
claude plugin install ravi@ravi
```

### 3. Authenticate

```bash
ravi auth login
```

After these steps, any Claude Code session will know how to
use `ravi`.

## Plugin repo

The plugin is maintained at
[ravi-hq/claude-code-plugin](https://github.com/ravi-hq/claude-code-plugin).

### Structure

```text
claude-code-plugin/
├── .claude-plugin/
│   ├── plugin.json          # Plugin metadata
│   └── marketplace.json     # Marketplace index
├── skills/
│   └── ravi-cli/
│       └── SKILL.md         # Skill file Claude Code reads
├── README.md
└── LICENSE
```

### Updating the skill

The skill content lives in two places:

- `.claude/skills/ravi-cli.md` (this repo) —
  dev use inside the CLI repo
- `skills/ravi-cli/SKILL.md` (plugin repo) —
  distributed to users via `claude plugin install`

When updating CLI commands, update **both** files to keep them
in sync. The plugin repo's `SKILL.md` is identical to
`.claude/skills/ravi-cli.md` except it has YAML frontmatter:

```yaml
---
name: ravi-cli
description: >-
  Use the Ravi CLI to manage agent identities,
  receive SMS/email, and store credentials
---
```
