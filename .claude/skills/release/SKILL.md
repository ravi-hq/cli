---
name: release
description: Guide the Ravi CLI release process — version bump, tagging, and verification
disable-model-invocation: true
---

# Release Ravi CLI

Guide the release process for a new CLI version. This is a checklist — the user drives each step.

## Arguments

The user should specify the version to release (e.g., "release 0.4.0").

## Pre-Release Checklist

### 1. Verify clean state
```bash
git status                    # Must be clean, on main branch
git log --oneline -5          # Review recent commits
```

### 2. Run full test suite
```bash
make test                     # All tests must pass
make lint                     # No lint errors
```

### 3. Verify build works
```bash
make build API_URL=https://ravi.app
./bin/ravi version            # Confirm version output
```

### 4. Check CHANGELOG-worthy commits since last tag
```bash
git log $(git describe --tags --abbrev=0)..HEAD --oneline --no-decorate
```
Review for any breaking changes that need documentation.

## Release Steps

### 5. Create and push the version tag
```bash
git tag v<VERSION>            # e.g., git tag v0.4.0
git push origin v<VERSION>    # Triggers GitHub Actions release workflow
```

### 6. Monitor the release workflow
```bash
gh run list --limit 1         # Find the triggered run
gh run watch                  # Watch it complete
```

The workflow automatically:
- Runs tests
- Cross-compiles for darwin/amd64, darwin/arm64, linux/amd64, linux/arm64
- Creates tar.gz archives with SHA256 checksums
- Creates GitHub Release with auto-generated notes
- Updates the Homebrew formula in `ravi-hq/homebrew-tap`

### 7. Verify the release
```bash
gh release view v<VERSION>    # Check release was created
```

Confirm:
- All 4 platform archives are attached
- checksums.txt is present
- Release notes look correct

### 8. Verify Homebrew tap updated
```bash
gh api repos/ravi-hq/homebrew-tap/commits --jq '.[0].commit.message'
```
Should show: `ravi <VERSION>`

## Rollback

If something goes wrong:
```bash
# Delete the tag locally and remotely
git tag -d v<VERSION>
git push origin :refs/tags/v<VERSION>

# Delete the GitHub release (if created)
gh release delete v<VERSION> --yes
```

## Platform Matrix

| OS | Arch | Binary Name |
|----|------|-------------|
| macOS | arm64 | ravi-darwin-arm64 |
| macOS | amd64 | ravi-darwin-amd64 |
| Linux | arm64 | ravi-linux-arm64 |
| Linux | amd64 | ravi-linux-amd64 |

Note: Windows (amd64) is built by the Makefile but NOT included in the release workflow.
