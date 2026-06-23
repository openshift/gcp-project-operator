# Development Guide

Quick reference for developing GCP Project Operator.

## Prerequisites

- **Go**: 1.22.7 or later
- **operator-sdk**: v1.21.0
- **kubectl**: For cluster interaction
- **prek**: Git hook manager (`brew install prek` or see [prek docs](https://prek.j178.dev/))

## Initial Setup

```bash
# Clone repository
git clone https://github.com/openshift/gcp-project-operator.git
cd gcp-project-operator

# Install pre-commit hooks
prek install
```

## Common Commands

### Build
```bash
make go-build                 # Build operator binary
make docker-build             # Build container image
```

### Test
```bash
make go-test                  # Run all unit tests
go test ./controllers/projectclaim/...  # Test specific package
ginkgo -r ./controllers/      # Run controller tests with Ginkgo
```

### Lint
```bash
make go-check                 # Full linting (golangci-lint)
prek run --all-files          # Run all prek hooks
```

### Code Generation
```bash
# After modifying API types (api/v1alpha1/*.go)
# or interfaces requiring mocks
boilerplate/_lib/container-make generate

# What this generates:
# - Deepcopy methods (zz_generated.deepcopy.go)
# - OpenAPI schemas
# - Mock interfaces for testing
```

### Run Locally
```bash
# Apply CRDs first
oc apply -f deploy/crds/gcp.managed.openshift.io_projectclaims.yaml
oc apply -f deploy/crds/gcp.managed.openshift.io_projectreferences.yaml

# Run operator locally against cluster in ~/.kube/config
operator-sdk run local --namespace gcp-project-operator
```

### Container-based Build
```bash
# Run make targets inside boilerplate container
# (ensures consistent environment with CI)
boilerplate/_lib/container-make go-test
boilerplate/_lib/container-make generate
```

## Fast Local Iteration

**Minimal validation loop:**
```bash
# After code changes
go build ./...                # Fast compile check (~5s)
go test ./pkg/mypackage       # Run affected tests
prek run                      # Lint staged files
```

**Full validation (pre-PR):**
```bash
prek run --all-files          # All hooks (~15-30s)
make validate                 # Generated code / manifest / boilerplate checks
make go-test                  # Full test suite
```

## Targeted Testing

```bash
# Run specific test
ginkgo -focus="NetworkPolicy" ./controllers/projectclaim/

# Run tests for one package
go test -v ./controllers/projectclaim/

# Skip slow tests during development
ginkgo -skip="E2E" -r ./...
```

## Debugging

```bash
# Print specific package logs
go test -v ./pkg/... 2>&1 | grep "MyFunction"

# Ginkgo verbose output
ginkgo -v ./...
```

## Dependency Management

```bash
# Add new dependency
go get github.com/some/package@v1.2.3

# Update dependency
go get -u github.com/some/package

# Tidy (removes unused, adds missing)
go mod tidy

# Verify checksums
go mod verify
```

**Note**: `go.sum` changes automatically trigger validation in prek hooks.

## Architecture Pointers

- **API Types**: `api/v1alpha1/` - CRD definitions
- **Controllers**: `controllers/{projectclaim,projectreference}/` - Reconciliation logic
- **Business Logic**: `controllers/projectclaim/` - Resource management
- **Tests**: `*_test.go` alongside source, `*_suite_test.go` for Ginkgo
- **Mocks**: `pkg/util/test/generated/` - Generated mocks
- **Config**: `config/` - Operator deployment configuration

## CI Parity

Local prek hooks mirror Tekton CI checks:
- **go-check** ↔ Tekton lint job
- **go-build** ↔ Compilation in CI
- **go-test** ↔ Unit test job
- **gitleaks** ↔ Security scanning
- **go-mod-tidy** ↔ Dependency consistency checks
- **rbac-wildcard-check** ↔ RBAC policy checks

Run `prek run --all-files` before pushing to catch CI failures early.

## Boilerplate Integration

This repo uses OpenShift boilerplate:
- Centralized Makefiles: `boilerplate/openshift/golang-osd-operator/`
- Standard targets: `go-build`, `go-check`, `go-test`
- Container builds: `boilerplate/_lib/container-make`
- Update boilerplate: `make boilerplate-update`

## Troubleshooting

**Mock generation fails:**
```bash
# Use container-make for consistency with CI
boilerplate/_lib/container-make generate
```

**Prek hook timeout:**
```bash
# macOS: Install GNU timeout
brew install coreutils

# Linux: timeout is built-in
```

**go.sum checksum mismatch:**
```bash
export GOPROXY="https://proxy.golang.org"
go mod tidy
```

**Tests fail locally but pass in CI:**
```bash
# Use container environment
boilerplate/_lib/container-make go-test
```

## Further Reading

- [Testing Guide](./TESTING.md)
- [Design Documentation](./docs/design.md)
- [Testing Documentation](./docs/testing.md)
- [Operator SDK Docs](https://sdk.operatorframework.io/)
