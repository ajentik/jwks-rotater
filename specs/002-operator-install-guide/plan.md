# Implementation Plan: Operator Installation Guide

**Branch**: `002-operator-install-guide` | **Date**: 2026-04-14 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/002-operator-install-guide/spec.md`

## Summary

Deliver a comprehensive installation guide for the JWKS Rotation Operator, consisting of: (1) a pre-built `dist/install.yaml` manifest referencing a published GHCR image, (2) a detailed standalone `docs/install.md` covering prerequisites, quick-start install, Kustomize install, verification, troubleshooting, uninstall, and building from source, and (3) an updated README with a concise installation summary linking to the full guide.

## Technical Context

**Language/Version**: Go 1.25+ (operator), Markdown (documentation deliverable)
**Primary Dependencies**: kustomize v5.8.1 (manifest generation), controller-gen v0.20.1 (CRD/RBAC generation)
**Storage**: N/A (documentation feature, no datastore)
**Testing**: Manual verification against a Kubernetes cluster; e2e test suite already exists for operator functionality
**Target Platform**: Kubernetes 1.28+ clusters
**Project Type**: Kubernetes operator (kubebuilder v4)
**Performance Goals**: N/A (documentation feature)
**Constraints**: Documentation must be accurate against the current codebase; `dist/install.yaml` must be valid and deployable
**Scale/Scope**: 3 deliverable files (`dist/install.yaml`, `docs/install.md`, updated `README.md`), 1 Makefile update

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Applicable? | Status | Notes |
|-----------|-------------|--------|-------|
| I. Test-Driven Development | Partially | PASS | This is a documentation feature. The generated `dist/install.yaml` must be validated by running `make build-installer` and verifying the output. The e2e test suite already tests operator deployment. No new Go code requires TDD. |
| II. Operator Pattern Discipline | No | N/A | No reconciliation logic changes. |
| III. Idiomatic Go | No | N/A | No Go code changes. |
| IV. Security by Default | Yes | PASS | `dist/install.yaml` inherits existing RBAC (least-privilege). Documentation must not expose private key material in examples. |
| V. Observability | No | N/A | No metrics/logging changes. |
| CRD Generation | Yes | PASS | `dist/install.yaml` is generated via `make build-installer` which depends on `make manifests generate`. No hand-edited CRDs. |
| RBAC Generation | Yes | PASS | RBAC in `dist/install.yaml` comes from generated `config/rbac/role.yaml`. No hand-edited RBAC. |

**Gate result**: PASS. No violations.

## Project Structure

### Documentation (this feature)

```text
specs/002-operator-install-guide/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (N/A for this feature)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
docs/
└── install.md           # NEW: Detailed installation guide (FR-011)

dist/
└── install.yaml         # NEW: Pre-built consolidated manifest (FR-002, FR-013)

README.md                # MODIFIED: Add installation summary section (FR-012)
Makefile                 # MODIFIED: Add GHCR image variable, update build-installer target
```

**Structure Decision**: This feature adds a `docs/` directory for the standalone installation guide and generates the `dist/install.yaml` manifest. The README is updated in-place. No changes to `src/`, `internal/`, `api/`, or test directories.
