# Contributing to GCP Project Operator

Thank you for your interest in contributing to the GCP Project Operator! This guide provides comprehensive instructions for setting up your development environment, running tests, and contributing code.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Prerequisites](#prerequisites)
- [Development Environment Setup](#development-environment-setup)
- [Pre-commit Hooks with prek](#pre-commit-hooks-with-prek)
- [Validation and Linting](#validation-and-linting)
- [Testing](#testing)
- [Boilerplate Framework](#boilerplate-framework)
- [Development Workflow](#development-workflow)
- [Claude Code Integration](#claude-code-integration)
- [CI/CD Integration](#cicd-integration)
- [Finding Issues to Work On](#finding-issues-to-work-on)
- [Submitting Pull Requests](#submitting-pull-requests)

## Code of Conduct

As contributors and maintainers of this GCP Project Operator, we respect all people who contribute through reporting issues, posting feature requests, updating documentation, submitting pull requests or patches, and other activities.

We are committed to making participation in this project a harassment-free experience for everyone, regardless of level of experience, gender, gender identity and expression, sexual orientation, disability, personal appearance, body size, race, ethnicity, age, religion, or nationality. In short, be excellent to each other.

## Prerequisites

Before contributing, ensure you have the following tools installed:

### Required Tools

- **Go 1.22+**: [Installation guide](https://golang.org/doc/install)
- **golangci-lint**: Used for Go code linting
  ```bash
  # macOS
  brew install golangci-lint

  # Linux
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
  ```

- **prek**: Git hook manager for pre-commit validation
  ```bash
  # macOS
  brew install prek

  # Linux
  curl -fsSL https://prek.j178.dev/install.sh | bash
  ```

- **operator-sdk**: Required for local operator development
  - [Installation guide](https://sdk.operatorframework.io/docs/installation/)

### Optional Tools (for Claude Code integration)

- **jq**: JSON processor used by Claude Code stop hooks
  ```bash
  # macOS
  brew install jq

  # Linux
  sudo apt-get install jq  # Debian/Ubuntu
  sudo dnf install jq      # Fedora/RHEL
  ```

## Development Environment Setup

1. **Fork and clone the repository**:
   ```bash
   git clone https://github.com/YOUR_USERNAME/gcp-project-operator.git
   cd gcp-project-operator
   ```

2. **Install pre-commit hooks**:
   ```bash
   prek install
   ```

   This installs git hooks that automatically run validation checks before each commit.

3. **Verify your setup**:
   ```bash
   make go-check    # Run linting
   make go-test     # Run tests
   ```

## Pre-commit Hooks with prek

This project uses [prek](https://prek.j178.dev/) to manage pre-commit hooks. The prek configuration is defined in [`prek.toml`](prek.toml).

### prek Version

The project pins a specific prek version in [`.prek-version`](.prek-version). This ensures consistent behavior across all contributors and CI environments.

### What prek Checks

Pre-commit hooks automatically check for:

- **File hygiene**: Trailing whitespace, end-of-file newlines
- **File format validation**: JSON, YAML, TOML syntax
- **Large files**: Prevents accidentally committing large binary files
- **Merge conflicts**: Detects unresolved merge conflict markers
- **Go linting**: Runs `make go-check` when Go files are modified

### Important: Boilerplate File Exclusions

The prek configuration **excludes** the `boilerplate/` directory from auto-fixing. This is intentional:

- Boilerplate files come from upstream and may contain trailing whitespace
- The CI system has a separate `boilerplate-freeze-check` that validates these files
- If prek accidentally modifies boilerplate files, restore them with:
  ```bash
  git checkout boilerplate/
  ```

### Running Validation Manually

```bash
# Run all pre-commit checks on all files
prek run --all-files

# Run all pre-commit checks on staged files only
prek run

# Run via CI script (same as what runs in CI)
./hack/ci.sh
```

## Validation and Linting

The project uses the [boilerplate framework](#boilerplate-framework) for standardized validation and linting.

### Available Make Targets

```bash
# Run Go linting with golangci-lint
make go-check

# Run Go tests
make go-test

# Run YAML validation and Go linting
make lint

# Verify code generation is up-to-date
make validate

# Run all checks (linting, testing, building)
make
```

### What Gets Validated

- **Go code**: golangci-lint checks for code quality issues, style violations, and potential bugs
- **YAML files**: Syntax validation for Kubernetes manifests
- **Generated code**: Ensures CRDs, OpenAPI specs, and Go code are up-to-date
- **Boilerplate headers**: Verifies copyright and license headers are present

## Testing

### Running Tests

```bash
# Preferred: use make target (includes envtest setup)
make go-test

# Advanced: run go test directly only after envtest setup
# The repository requires kubebuilder test assets to be set up first.
# If you've already run make go-test once, you can use:
go test ./...                          # Run all tests
go test ./controllers/projectclaim/    # Run specific package tests
go test -v ./...                       # Run with verbose output
go test -cover ./...                   # Run with coverage
```

### Writing Tests

When contributing new features, please include tests. See the [testing documentation](./docs/testing.md) for detailed guidance.

## Boilerplate Framework

This project uses the [OpenShift boilerplate framework](https://github.com/openshift/boilerplate) for standardized build, test, and validation workflows.

### What is Boilerplate?

Boilerplate provides:
- Standardized Makefiles for common operations
- Code generation utilities
- CI/CD configuration templates
- Validation and linting rules

### Key Boilerplate Commands

```bash
# Update boilerplate to latest version
make boilerplate-update

# Generate all code (CRDs, Go types, OpenAPI)
make generate

# Validate that generated code is up-to-date
make validate

# Build the operator binary
make go-build
```

### Boilerplate Files Are Protected

The `boilerplate/` directory contains generated files from upstream. Do not modify these files manually:
- Changes will be rejected by CI (`boilerplate-freeze-check`)
- prek excludes these files from auto-fixing
- Update boilerplate via `make boilerplate-update` if needed

## Development Workflow

### Typical Development Cycle

1. **Create a feature branch**:
   ```bash
   git checkout -b feat/my-feature
   ```

2. **Make your changes** and commit frequently:
   ```bash
   git add .
   git commit -m "Add feature X"
   # Pre-commit hooks run automatically
   ```

3. **Run validation before pushing**:
   ```bash
   prek run --all-files
   make validate
   make go-test
   ```

4. **Push and create a pull request**:
   ```bash
   git push origin feat/my-feature
   ```

### Local Operator Development

See [DEVELOPMENT.md](./DEVELOPMENT.md#run-locally) for CRD setup and running the operator locally.

## Claude Code Integration

This project includes integration with [Claude Code](https://claude.ai/code), an AI-powered development tool.

### Stop Hook Validation

Claude Code users benefit from an automatic **stop hook** that runs `prek run --all-files` before Claude stops working. This catches validation issues early in the development cycle.

**How it works**:
1. When Claude Code is about to stop, the hook runs validation
2. If validation fails, Claude is blocked from stopping and shown the errors
3. Claude fixes the issues and tries again
4. Once validation passes, Claude can stop normally

**Setup for Claude Code users**:
- The stop hook is configured in [`.claude/settings.json`](.claude/settings.json)
- The hook script is at [`.claude/hooks/stop-prek-validation.sh`](.claude/hooks/stop-prek-validation.sh)
- Requires `jq` and `prek` to be installed (see [Prerequisites](#prerequisites))

**Human developers** should follow the standard setup in this guide and rely on pre-commit hooks instead.

## CI/CD Integration

The project uses [OpenShift PROW](https://docs.ci.openshift.org/docs/) for continuous integration.

### CI Checks

Every pull request runs:
- `prek` validation (via `hack/ci.sh`)
- Go linting (`make go-check`)
- Unit tests (`make go-test`)
- Code generation validation (`make validate`)
- Boilerplate freeze check

### CI Configuration

CI configuration is maintained in the [openshift/release](https://github.com/openshift/release) repository:
- **Config file**: `ci-operator/config/openshift/gcp-project-operator/openshift-gcp-project-operator-master.yaml`
- **prek runner image**: Built from the operator image with prek binary added
- **Test jobs**: Run validation, linting, and tests

## Finding Issues to Work On

* ["good-first-issue"](https://github.com/openshift/gcp-project-operator/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22) - Issues that are easy to complete even for beginners

* ["help wanted"](https://github.com/openshift/gcp-project-operator/issues?q=is%3Aopen+is%3Aissue+label%3A%22help+wanted%22) - Issues where we currently have no resources to work on them

Once you've discovered an issue to work on:
1. Add a comment mentioning that you plan to work on this issue
2. Assign the issue to yourself
3. Reference the issue in your commit messages and PR description

## Submitting Pull Requests

### PR Guidelines

1. **Include tests**: New features and bug fixes should include appropriate tests
2. **Update documentation**: Update docs if you're changing user-facing behavior
3. **Follow code style**: Run `make go-check` before submitting
4. **Write clear commit messages**: Follow [conventional commits](https://www.conventionalcommits.org/) format
5. **Reference issues**: Mention related issues in your PR description

### PR Process

1. All tests must pass in CI
2. Code review required from maintainers (`/lgtm` label)
3. Approval required from approvers (`/approve` label)
4. PRs are merged automatically by the OpenShift Merge Bot
5. Mark PRs as work-in-progress with `[WIP]` in the title to prevent accidental merges

### PR Template

See [docs/PULL_REQUEST_TEMPLATE.md](./docs/PULL_REQUEST_TEMPLATE.md) for the PR template.

## Additional Resources

- **User documentation**: [docs/userstory.md](./docs/userstory.md)
- **API documentation**: [docs/api.md](./docs/api.md)
- **Design documentation**: [docs/design.md](./docs/design.md)
- **Building guide**: [docs/building.md](./docs/building.md)
- **Testing guide**: [docs/testing.md](./docs/testing.md)
- **Troubleshooting**: [docs/troubleshooting.md](./docs/troubleshooting.md)

## Getting Help

If you need help or have questions:
- Open an issue on [GitHub](https://github.com/openshift/gcp-project-operator/issues)
- Check existing [documentation](./docs/)
- Ask in your pull request comments

Thank you for contributing to GCP Project Operator!
