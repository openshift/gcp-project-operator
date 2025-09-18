# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Development Commands

### Building and Testing
- `make` - Default target: runs go-check, go-test, and go-build
- `make go-build` - Build the operator binary (outputs to build/_output/bin/gcp-project-operator)
- `make go-test` - Run unit tests with test environment setup
- `make go-check` - Run linting and static analysis with golangci-lint
- `make test` - Alias for go-test
- `make lint` - Run YAML validation and Go linting
- `make validate` - Ensure code generation is up-to-date and validate boilerplate

### Code Generation
- `make generate` - Generate all code (CRDs, Go code, OpenAPI, manifests)
- `make op-generate` - Generate CRDs and Go objects for API types
- `make go-generate` - Run go generate on all packages
- `make openapi-generate` - Generate OpenAPI specs

### Local Development
- `operator-sdk up local --namespace gcp-project-operator` - Run operator locally (requires CRDs to be applied first)
- `oc apply -f deploy/crds/gcp.managed.openshift.io_projectclaims.yaml` - Apply ProjectClaim CRD
- `oc apply -f deploy/crds/gcp.managed.openshift.io_projectreferences.yaml` - Apply ProjectReference CRD

### Testing
- `make gotest` or `go test ./...` - Run Go tests directly
- Example CRs are available in `deploy/crds/` for testing

## Architecture Overview

### Core Components
This is a Kubernetes operator that manages GCP projects and service accounts. The operator watches for custom resources and manages their lifecycle through reconciliation loops.

**Main Controllers:**
- `ProjectClaimReconciler` (controllers/projectclaim/) - Handles ProjectClaim resources
- `ProjectReferenceReconciler` (controllers/projectreference/) - Handles ProjectReference resources

**API Types:**
- `ProjectClaim` (api/v1alpha1/projectclaim_types.go) - Primary CR for requesting GCP projects
- `ProjectReference` (api/v1alpha1/projectreference_types.go) - Internal tracking of project state
- Common types in api/v1alpha1/common_types.go

**Key Packages:**
- `pkg/gcpclient/` - GCP API client wrapper
- `pkg/configmap/` - ConfigMap handling for operator configuration
- `pkg/condition/` - Condition management utilities
- `pkg/util/` - General utilities and mocks

### Workflow
1. User creates a `ProjectClaim` CR specifying region and credentials
2. ProjectClaim controller creates a `ProjectReference` and initiates GCP project creation
3. ProjectReference controller handles the actual GCP interactions (project creation, service account setup)
4. Upon success, credentials are stored in a Kubernetes secret and ProjectClaim status is updated to "Ready"
5. Project deletion is triggered when ProjectClaim is deleted (finalizer-based cleanup)

### Configuration Requirements
The operator requires:
- A Secret named `gcp-project-operator-credentials` containing GCP service account keys (key.json)
- A ConfigMap named `gcp-project-operator` with operator configuration
- Both must be in the `gcp-project-operator` namespace

### Build System
This project uses OpenShift's boilerplate build system with:
- FIPS-enabled builds (FIPS_ENABLED=true)
- Container image building with podman/docker
- Automated code generation and validation
- OLM bundle and catalog generation for operator lifecycle management

The build system is primarily controlled through boilerplate makefiles in the `boilerplate/` directory.