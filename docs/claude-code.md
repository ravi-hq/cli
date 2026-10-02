# Claude Code

Install the Ravi plugin so Claude Code can use `ravi`. Skills and the
Claude Code marketplace are published from
[ravi-hq/ravi-skills](https://github.com/ravi-hq/ravi-skills)
(`.claude-plugin/` on `main`). Product docs:
[docs.ravi.app](https://docs.ravi.app).

## Install

From a shell:

```bash
claude plugin marketplace add ravi-hq/ravi-skills
claude plugin install ravi@ravi-hq
```

Inside a Claude Code session, the same steps are:

```text
/plugin marketplace add ravi-hq/ravi-skills
/plugin install ravi@ravi-hq
```

`ravi-hq/ravi-skills` is the GitHub source. Claude Code names the
marketplace from `.claude-plugin/marketplace.json`, which is `ravi-hq`.
The plugin entry in that file is `ravi`, so the install target is
`ravi@ravi-hq`.

The plugin puts a `ravi` wrapper on `PATH` and installs the CLI on
first use. Then bind this machine's CLI (one identity per config file):

```bash
ravi auth login
```

That is an RFC 8628 device-code flow against https://api.ravi.id. Open
https://ravi.id/device and enter the code the CLI prints. Keys
(`ravi_mgmt_` / `ravi_id_`) go in `~/.ravi/config.json`. Auth commands
are `login` / `logout` / `status` only.

Cursor agents should use the MCP Connect card (per-agent credentials),
not this plugin install and not a shared `~/.ravi/config.json`.

## Where the skills live

Distributed skills are in
[ravi-hq/ravi-skills](https://github.com/ravi-hq/ravi-skills). The copy
under `.claude/skills/` in this repository is for developing the CLI.
