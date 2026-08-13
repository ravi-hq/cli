# Claude Code Plugin

The Ravi CLI has a Claude Code plugin that teaches Claude Code
how to use `ravi`. When the plugin is installed, Claude Code can
autonomously sign up for services, receive OTPs, and manage
credentials on behalf of the user.

## How it works

The plugin provides a **skill file** — a structured document
that tells Claude Code:

- What commands are available
  (`ravi get email`, `ravi inbox sms`, `ravi sms send`, `ravi call`, `ravi passwords create`, etc.)
- What the JSON output looks like for each command
- Common workflows (signup, OTP extraction, using login codes)
- Important conventions (JSON is default output, poll with `sleep 5`)

The plugin does **not** include the CLI itself.
Users must install `ravi` separately.

## Installation

### Cursor (first-class)

Use the **MCP Connect card** (per-agent credentials). That is the supported
path for Cursor agents — not `ravi auth login`, which binds one identity into
a shared `~/.ravi/config.json` that cannot run multiple agents.

Several agents on one host should call https://api.ravi.app with per-identity
`ravi_id_` keys.

### Claude Code plugin

1. Install the CLI from the
   [releases page](https://github.com/ravi-hq/cli/releases)
   or `make build`.

2. Install the plugin:

   ```bash
   claude plugin marketplace add ravi-hq/claude-code-plugin
   claude plugin install ravi@ravi
   ```

3. Optional — bind this machine's CLI (one identity per config file):

   ```bash
   ravi auth login
   ```

   RFC 8628 device-code against https://api.ravi.app. Open https://ravi.id/device
   and enter the code the CLI prints. Keys (`ravi_mgmt_` / `ravi_id_`) go in
   `~/.ravi/config.json`. Auth commands are `login` / `logout` / `status` only.

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
