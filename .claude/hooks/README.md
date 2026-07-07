# Claude Code Hooks

Security and validation hooks for gcp-project-operator development.

## Overview

This repository uses **prek** (git hook manager) for quality checks and validation. Claude Code hooks integrate with prek to provide immediate feedback during development.

## Architecture

```text
┌─────────────────────────────────────┐
│   Developer / Claude Code Agent     │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│   Stop Hook (conditional)           │
│   - Default: runs only with changes │
│   - Strict: runs every turn         │
│   - Blocks if issues found          │
│   - Claude fixes automatically      │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│   Prek Hooks (CI config)            │
│   - golangci-lint (static analysis) │
│   - go build validation             │
│   - file hygiene (trailing space)  │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│   Prek Hooks (full config)          │
│   + gitleaks (secret scanning)      │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│   Git Commit                         │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│   CI/CD (Prow)                      │
└─────────────────────────────────────┘
```

## Available Hooks

### [stop-prek-validation.sh](./stop-prek-validation.sh)
**Purpose**: Run prek validation when Claude makes changes (or always, if configured)

**Triggers**: On Claude Code session stop (Stop hook)

**Behavior**:

**Default mode** (recommended):
- Only runs if there are uncommitted changes (staged, unstaged, or untracked files)
- Skips validation for read-only queries (fast iteration)
- Validates when Claude modifies code (before commit)

**Strict mode** (opt-in):
- Set environment variable: `export CLAUDE_LINT_ON_STOP=true`
- Always runs validation on every stop, regardless of changes
- Use when you want maximum quality enforcement
- Slower but catches issues immediately

**Common behavior**:
- Runs `prek run --config hack/prek.ci.toml` on changed files
- Uses CI-compatible config (skips network-dependent hooks like gitleaks)
- Blocks Claude from stopping if issues found
- Feeds errors back to Claude for automatic fixes
- Includes infinite loop guard (allows stop on retry)

**Benefits**:
- **Default**: No performance impact on read-only queries (0s when no changes)
- **Default**: Catches issues when Claude modifies code (before commit)
- **Strict**: Maximum quality enforcement on every interaction
- Fast validation (5-10s typical) - only checks changed files

**Performance**:
- Default mode, clean working directory: 0s (skipped)
- Default mode, with changes: 5-10s typical (changed files only)
- Strict mode (CLAUDE_LINT_ON_STOP=true): 5-10s every stop

**Installation**: Configured in `.claude/settings.json`

**Enable strict mode**:
```bash
# In your shell profile (~/.zshrc, ~/.bashrc)
export CLAUDE_LINT_ON_STOP=true

# Or for single session
CLAUDE_LINT_ON_STOP=true claude
```

---

### [session-start-prek-setup.sh](./session-start-prek-setup.sh)
**Purpose**: Ensure prek is installed and configured when Claude Code starts

**Triggers**: On Claude Code session start

**Behavior**:
- Checks if prek is installed
- If installed, ensures git hooks are wired up (`prek install`)
- Provides helpful guidance if prek is missing
- Non-blocking (won't prevent session start)

**Installation**: Configured in `.claude/settings.json`

---

### [pre-edit.sh](./pre-edit.sh)
**Purpose**: Prevent editing generated files and warn about high-risk changes

**Status**: Available for standalone use (not configured as Claude Code hook)

**Checks**:
- Generated files (`zz_generated.*.go`)
- Generated mocks (`**/generated/mock_*.go`)
- Generated CRD manifests (`deploy/crds/*.yaml`)
- Vendored code (`vendor/`)
- Boilerplate files (managed upstream)
- High-risk security files (RBAC, auth, NetworkPolicy)
- CI/CD pipelines (`.tekton/*.yaml`)
- Dockerfiles

**Manual Usage**:
```bash
.claude/hooks/pre-edit.sh path/to/file.go
```

---

## Prek Configuration

This repository maintains **two prek configurations**:

### 1. **prek.toml** (Full validation)
Used for local development.

**Hooks**:
- File hygiene (trailing whitespace, EOF, syntax checks)
- **gitleaks**: Secret detection (configured via `.gitleaks.toml`)
- **golangci-lint**: Static analysis

**Usage**:
```bash
prek run  # Uses prek.toml by default
```

### 2. **hack/prek.ci.toml** (CI-compatible)
Used by Claude Code stop hook and CI environments.

**Excludes**:
- `gitleaks` (may have network dependencies)

**Usage**:
```bash
hack/ci.sh
# or
prek run --config hack/prek.ci.toml --all-files
```

**Why two configs?**
The CI-compatible config allows Claude Code and external CI systems to run quality checks without network-dependent hooks that may fail in isolated environments.

## Setup

### Prerequisites
```bash
# Install prek (choose one)
uv tool install prek      # recommended
pipx install prek         # alternative
pip install --user prek   # fallback
```

### Install Git Hooks
```bash
prek install
```

This sets up pre-commit hooks that run validation automatically.

## Usage

### Automatic Validation
Prek runs automatically:
- **Stop hook (default mode)**: Runs `prek run --config hack/prek.ci.toml` only when changes are present (staged, unstaged, or untracked files)
- **Stop hook (strict mode)**: Set `CLAUDE_LINT_ON_STOP=true` to run on every turn regardless of changes
- **On commit**: Pre-commit hook runs relevant checks

### Manual Validation
```bash
# Run all checks
prek run --all-files

# Run CI-compatible checks
prek run --config hack/prek.ci.toml --all-files

# Run specific check
prek run gitleaks
prek run golangci-lint
```

## Hook Categories

### Stop Hooks
**Purpose**: Validate before Claude Code stops

**Current**:
- `stop-prek-validation.sh`: Run prek checks

**Benefits**:
- Immediate feedback (not delayed until commit)
- Automatic fixes by Claude
- Prevents accumulation of violations

### Session Start Hooks
**Purpose**: Set up development environment when Claude Code starts

**Current**:
- `session-start-prek-setup.sh`: Ensure prek is installed and configured

### Pre-commit Hooks
**Purpose**: Validate before git commit

**Managed by**: prek (configured in `prek.toml`)

**Checks**:
- File hygiene and syntax
- Security scanning (gitleaks)
- Static analysis (golangci-lint)

## Security Guardrails

### Secret Prevention
**Implementation**: gitleaks via prek

**Configuration**: `.gitleaks.toml`

**Detects**:
- GCP service account keys
- GCP API keys
- GitHub tokens
- AWS credentials (for comparison testing)
- Private keys
- Database connection strings
- High-entropy secrets

**Action**: BLOCK commit

### File Edit Protection
**Implementation**: pre-edit.sh (standalone)

**Detects**:
- Generated files (`zz_generated.*.go`)
- Generated mocks (`**/generated/mock_*.go`)
- CRD manifests (`deploy/crds/*.yaml`)
- Vendored code (`vendor/`)
- Boilerplate files (managed upstream)
- High-risk files (RBAC, auth, NetworkPolicy, `.tekton/*.yaml`, Dockerfiles)

**Action**: BLOCK edit on generated/vendored files, WARN on high-risk files

## Hook Performance

**Targets:**
- Stop hook: <30s for full validation
- Pre-commit hook: <30s on typical changeset
- Individual checks: <10s each

**Optimization:**
- Prek runs hooks in parallel where possible
- Hooks only check changed files (where applicable)
- Build artifacts cached between runs

## Troubleshooting

### Hook Not Running
```bash
# Verify prek is installed
prek --version

# Reinstall git hooks
prek install

# Check hook configuration
cat prek.toml
```

### Hook Fails Incorrectly
```bash
# Run hook manually for debugging
prek run <hook-id> --verbose

# Check hook configuration
cat prek.toml

# Update prek
uv tool upgrade prek  # or pipx upgrade prek
```

### Hook Failures (DO NOT Bypass)

**NEVER bypass hooks:**
```bash
# FORBIDDEN - bypasses all validation
git commit --no-verify

# FORBIDDEN - bypasses specific hooks
SKIP=hook-id git commit
```

**If hooks are blocking your commit:**
1. **Investigate and fix the root issue** - hooks catch real problems
2. **If the hook or config is broken:**
   - Fix the hook/config first
   - Open an issue documenting the problem
   - Request reviewer approval before merge
3. **Re-run full validation:**
   - `prek run --all-files` locally
   - Ensure all required CI checks pass
   - Get explicit code review approval

**Security hooks (gitleaks) must NEVER be bypassed under any circumstances.**

## Version Management

### Prek Version
Pinned in `.prek-version` for CI consistency:
```bash
cat .prek-version  # v0.4.1
```

Update when new prek releases are available.

### Hook Dependencies
Defined in `prek.toml` with immutable refs:
- `v8.18.0` (gitleaks)
- `v2.0.2` (golangci-lint)

## References

- [Prek Documentation](https://prek.j178.dev/)
- [Gitleaks](https://github.com/gitleaks/gitleaks)
- [golangci-lint](https://golangci-lint.run/)
- [CLAUDE.md](../../CLAUDE.md) - Development guidelines
- [CONTRIBUTING.md](../../CONTRIBUTING.md) - Contribution workflow
