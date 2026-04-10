# Implementation Plan: JWKS Rotation Operator

**Branch**: `001-jwks-operator` | **Date**: 2026-04-10 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-jwks-operator/spec.md`

## Summary

Build a Kubernetes operator that manages JWKS key rotation via CRDs. The operator watches `JWKSRotation` (namespace-scoped) and `JWKSRotationPolicy` (cluster-scoped) resources, generates RSA/ECDSA key pairs on a configurable schedule, stores them in Kubernetes Secrets (private + derived public), cleans up expired keys after a retention period, and optionally triggers rolling restarts on target Deployments. Built with Go and controller-runtime (kubebuilder).

## Technical Context

**Language/Version**: Go 1.22+
**Primary Dependencies**: controller-runtime v0.18+, kubebuilder v4, go-jose/v4, controller-runtime/pkg/metrics (Prometheus)
**Storage**: Kubernetes Secrets (native)
**Testing**: Go test + envtest (controller-runtime test harness), ginkgo/gomega for integration tests
**Target Platform**: Kubernetes 1.27+
**Project Type**: Kubernetes Operator (controller binary + CRDs + RBAC manifests)
**Performance Goals**: Key available within 60s of rotation interval; Deployment restart triggered within 30s; recovery from missed rotations within 2 minutes
**Constraints**: Single-leader via Lease; namespace-scoped JWKSRotation; cluster-scoped JWKSRotationPolicy
**Scale/Scope**: Hundreds of JWKSRotation resources per cluster (typical operator workload)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Constitution has not been configured for this project (template placeholders only). No gates to evaluate. Proceeding.

**Post-Phase 1 re-check**: N/A — no constitution gates defined.

## Project Structure

### Documentation (this feature)

```text
specs/001-jwks-operator/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── crd-schemas.md
└── tasks.md
```

### Source Code (repository root)

```text
api/
├── v1alpha1/
│   ├── jwksrotation_types.go
│   ├── jwksrotationpolicy_types.go
│   ├── groupversion_info.go
│   └── zz_generated.deepcopy.go

internal/
├── controller/
│   ├── jwksrotation_controller.go
│   ├── jwksrotation_controller_test.go
│   ├── jwksrotationpolicy_controller.go
│   └── jwksrotationpolicy_controller_test.go
├── jwks/
│   ├── keygen.go
│   ├── keygen_test.go
│   ├── jwks.go
│   └── jwks_test.go
└── metrics/
    └── metrics.go

config/
├── crd/
│   └── bases/
├── rbac/
├── manager/
├── default/
└── samples/

cmd/
└── main.go

test/
└── e2e/
    └── e2e_test.go

Dockerfile
Makefile
go.mod
go.sum
```

**Structure Decision**: Standard kubebuilder project layout. CRD types in `api/v1alpha1/`, controllers in `internal/controller/`, JWKS key generation logic isolated in `internal/jwks/` for independent unit testing.

## Complexity Tracking

No constitution violations to justify.
