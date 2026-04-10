# Implementation Plan: JWKS Rotation Operator

**Branch**: `001-jwks-operator` | **Date**: 2026-04-10 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-jwks-operator/spec.md`

## Summary

A Kubernetes operator that manages automatic JWKS key rotation. The operator watches JWKSRotation custom resources, generates RSA/ECDSA key pairs using go-jose/v4, stores them as Kubernetes Secrets (private + derived public), rotates on schedule via controller-runtime's RequeueAfter, cleans up expired keys after a retention period, and optionally restarts target Deployments. A cluster-scoped JWKSRotationPolicy CRD enables auto-discovery of Deployments via label selectors. API group: `jwks.ajentik.ai`.

## Technical Context

**Language/Version**: Go 1.22+
**Primary Dependencies**: controller-runtime v0.18+, kubebuilder v4, go-jose/v4
**Storage**: Kubernetes Secrets (native) — no external datastore
**Testing**: Go `testing` package + controller-runtime envtest; TDD mandatory
**Target Platform**: Kubernetes 1.28+
**Project Type**: Kubernetes operator (CRD + controller)
**Performance Goals**: Key available in Secret within 60s of rotation interval; reconciliation < 5s
**Constraints**: No private key material in logs/events/status; least-privilege RBAC
**Scale/Scope**: Operator manages JWKSRotation CRs across namespaces; single binary deployment

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Evidence |
|-----------|--------|----------|
| I. TDD (NON-NEGOTIABLE) | ✅ PASS | Tasks require tests written first (Red-Green-Refactor). envtest for controller tests, table-driven tests for key generation and retention logic. |
| II. Operator Pattern Discipline | ✅ PASS | Idempotent reconciliation, status conditions (Ready/Degraded/Error), finalizer for cleanup, owner references on Secrets, RequeueAfter from rotation schedule. |
| III. Idiomatic Go | ✅ PASS | Kubebuilder v4 layout (api/, internal/controller/, cmd/). go vet + golangci-lint. Error wrapping with %w. Context propagation. |
| IV. Security by Default | ✅ PASS | crypto/rand for key gen. Private keys never logged. Opaque Secrets with jwks.json only. Least-privilege RBAC via controller-gen markers. |
| V. Observability | ✅ PASS | Kubernetes Events for rotation/cleanup/errors. Prometheus metrics via controller-runtime. Structured logging via logr. |

## Project Structure

### Documentation (this feature)

```text
specs/001-jwks-operator/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── crd-schemas.md   # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
cmd/
└── main.go                          # Operator entrypoint

api/
└── v1alpha1/
    ├── jwksrotation_types.go        # JWKSRotation CRD Go types
    ├── jwksrotationpolicy_types.go  # JWKSRotationPolicy CRD Go types
    ├── groupversion_info.go         # API group registration
    └── zz_generated.deepcopy.go     # Generated

internal/
├── controller/
│   ├── jwksrotation_controller.go       # Main reconciler
│   ├── jwksrotation_controller_test.go  # envtest-based controller tests
│   ├── jwksrotationpolicy_controller.go
│   └── jwksrotationpolicy_controller_test.go
└── jwks/
    ├── keygen.go              # Key generation (RSA, ECDSA)
    ├── keygen_test.go         # Table-driven key gen tests
    ├── keystore.go            # JWKS assembly, serialization, retention
    ├── keystore_test.go       # Table-driven retention/cleanup tests
    ├── secret.go              # Secret construction helpers
    └── secret_test.go         # Secret assembly tests

config/
├── crd/                       # Generated CRD manifests
├── rbac/                      # Generated RBAC manifests
├── manager/                   # Deployment manifests
├── default/                   # Kustomize default overlay
└── samples/
    ├── jwks_v1alpha1_jwksrotation.yaml
    └── jwks_v1alpha1_jwksrotationpolicy.yaml

Dockerfile                     # Multi-stage build
Makefile                       # kubebuilder standard targets
```

**Structure Decision**: Standard kubebuilder v4 layout. Business logic isolated in `internal/jwks/` for independent unit testing. Controllers in `internal/controller/` tested via envtest.

## Complexity Tracking

No constitution violations. No complexity justifications needed.
