<!--
Sync Impact Report
- Version change: N/A → 1.0.0 (initial ratification)
- Added principles:
  - I. Test-Driven Development (NON-NEGOTIABLE)
  - II. Operator Pattern Discipline
  - III. Idiomatic Go
  - IV. Security by Default
  - V. Observability
- Added sections:
  - Technology Constraints
  - Development Workflow
  - Governance
- Templates requiring updates:
  - .specify/templates/plan-template.md ✅ no changes needed (generic)
  - .specify/templates/spec-template.md ✅ no changes needed (generic)
  - .specify/templates/tasks-template.md ✅ no changes needed (supports test-first)
- Follow-up TODOs: none
-->

# JWKS Rotation Operator Constitution

## Core Principles

### I. Test-Driven Development (NON-NEGOTIABLE)

Every feature MUST follow the Red-Green-Refactor cycle:
1. Write a failing test that captures the requirement
2. Implement the minimum code to make the test pass
3. Refactor while keeping tests green

- Unit tests MUST cover all reconciliation logic, key generation, retention calculations, and Secret assembly
- Integration tests MUST use envtest (controller-runtime's test harness) to validate CRD behavior, Secret lifecycle, and status updates against a real API server
- Table-driven tests MUST be used for functions with multiple input/output variations
- Test files MUST live alongside the code they test (`_test.go` in the same package)
- Mocks are permitted only for external cryptographic randomness; all Kubernetes API interactions MUST use envtest

### II. Operator Pattern Discipline

- Every reconciliation MUST be idempotent: applying the same CR twice produces the same cluster state
- The controller MUST use status subresource conditions (Ready, Degraded, Error) following standard Kubernetes conventions
- Finalizers MUST be used for cleanup of owned resources
- The controller MUST set owner references on created Secrets so garbage collection works if finalizer logic fails
- Requeue intervals MUST be derived from rotation/retention schedules, not hardcoded polling
- Leader election MUST be enabled for all production deployments

### III. Idiomatic Go

- Code MUST pass `go vet`, `staticcheck`, and `golangci-lint` with no suppressions unless explicitly justified
- Errors MUST be wrapped with `fmt.Errorf("context: %w", err)` to preserve error chains
- Exported types MUST have GoDoc comments
- Package layout MUST follow kubebuilder conventions: `api/`, `internal/controller/`, `cmd/`
- Dependencies MUST be kept minimal; prefer standard library and controller-runtime utilities over third-party packages
- Context propagation MUST be used throughout; no background contexts in reconciliation paths

### IV. Security by Default

- Private key material MUST never appear in logs, events, metrics, or status fields
- Secrets MUST be created with `type: Opaque` and contain only the `jwks.json` data key
- Key generation MUST use `crypto/rand` exclusively; no deterministic or seeded randomness
- RBAC manifests MUST follow least-privilege: only the permissions the operator actually needs
- The operator MUST NOT read or cache private key material beyond the scope of a single reconciliation

### V. Observability

- The operator MUST emit Kubernetes Events for rotation, cleanup, recreation, and error conditions
- Prometheus metrics MUST be exposed via controller-runtime's metrics server: rotation count, key age, error count, reconciliation duration
- Structured logging MUST use controller-runtime's `logr` interface with consistent key-value pairs
- Log levels: Info for lifecycle events, Error for failures, Debug (V=1) for reconciliation details

## Technology Constraints

- **Language**: Go 1.22+
- **Framework**: controller-runtime v0.18+, kubebuilder v4 scaffolding
- **JWKS library**: go-jose/v4 for JWK construction and serialization
- **Testing**: Go's built-in `testing` package + controller-runtime envtest
- **Metrics**: controller-runtime/pkg/metrics (Prometheus)
- **Target platform**: Kubernetes 1.28+
- **Build**: Standard `go build`; container image via multi-stage Dockerfile
- **No external datastore**: all state lives in Kubernetes resources (CRs and Secrets)

## Development Workflow

- Every PR MUST include tests that fail without the change and pass with it
- `make test` MUST pass before any code is merged
- CRD changes MUST be generated via `controller-gen` from Go type markers, never hand-edited
- RBAC manifests MUST be generated via `controller-gen` from kubebuilder markers
- All generated code MUST be committed (manifests, DeepCopy methods, RBAC)
- Commits MUST be atomic: one logical change per commit with a clear message

## Governance

This constitution is the authoritative source of development standards for the JWKS Rotation Operator project. All code contributions, reviews, and architectural decisions MUST comply with these principles.

- Amendments require: (1) a written proposal describing the change and rationale, (2) review and approval, (3) a migration plan if the change affects existing code
- Version follows semantic versioning: MAJOR for principle removals or redefinitions, MINOR for new principles or material expansions, PATCH for clarifications
- Compliance is verified during code review; reviewers MUST check adherence to TDD, idiomatic Go, and security principles

**Version**: 1.0.0 | **Ratified**: 2026-04-10 | **Last Amended**: 2026-04-10
