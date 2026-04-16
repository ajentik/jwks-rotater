# Implementation Plan: Helm Chart & GHCR CI/CD Pipeline

**Branch**: `003-helm-ghcr-deploy` | **Date**: 2026-04-16 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/003-helm-ghcr-deploy/spec.md`

## Summary

Add a Helm chart distribution strategy and GitHub Actions CI/CD pipelines to build and publish both the operator container image and Helm chart as OCI artifacts to GHCR. The Helm chart will be generated via kubebuilder's helm plugin, customized to manage CRDs as templates, and the release pipeline will gate on passing tests and lint before publishing.

## Technical Context

**Language/Version**: Go 1.25+, GitHub Actions YAML, Helm chart templates  
**Primary Dependencies**: kubebuilder v4 (helm/v2-alpha plugin), docker buildx, helm CLI, GitHub Actions  
**Storage**: N/A  
**Testing**: `helm lint`, `helm template` for chart validation; existing `make test` and `make lint` for gating  
**Target Platform**: Kubernetes 1.28+ clusters, GitHub Actions runners (ubuntu-latest)  
**Project Type**: Kubernetes operator with Helm distribution and CI/CD pipelines  
**Performance Goals**: Release pipeline completes within 15 minutes  
**Constraints**: Private GHCR registry; GITHUB_TOKEN authentication; multi-arch images (amd64, arm64)  
**Scale/Scope**: Single operator chart, 2 new workflows, 1 modified workflow

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Test-Driven Development | PASS | Chart validation (helm lint/template) in CI covers the "test first" requirement for this infrastructure feature. No Go code changes requiring unit tests. |
| II. Operator Pattern Discipline | PASS | Helm chart will deploy the operator with leader election, RBAC, CRDs, and status subresource — all already implemented in the operator. |
| III. Idiomatic Go | N/A | This feature adds no Go code; only Helm templates, GitHub Actions YAML, and Makefile targets. |
| IV. Security by Default | PASS | GHCR packages are private. GITHUB_TOKEN used for auth (no long-lived secrets). RBAC in chart follows least-privilege from generated manifests. |
| V. Observability | PASS | Existing metrics server and logging configuration will be preserved in Helm chart values. |
| Development Workflow | PASS | `make test` gating preserved. Generated manifests committed. Chart validated in PR CI. |

No violations. Proceeding.

## Project Structure

### Documentation (this feature)

```text
specs/003-helm-ghcr-deploy/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
└── tasks.md
```

### Source Code (repository root)

```text
dist/chart/                          # Helm chart (kubebuilder helm plugin output)
├── Chart.yaml
├── values.yaml
├── templates/
│   ├── manager/
│   │   └── manager.yaml             # Deployment template
│   ├── rbac/
│   │   ├── role.yaml
│   │   ├── rolebinding.yaml
│   │   └── serviceaccount.yaml
│   ├── crd/
│   │   ├── jwks.ajentik.ai_jwksrotations.yaml
│   │   └── jwks.ajentik.ai_jwksrotationpolicies.yaml
│   └── _helpers.tpl
.github/workflows/
├── test.yml                         # Existing (unchanged)
├── test-e2e.yml                     # Existing (unchanged)
├── lint.yml                         # Existing (unchanged)
├── release.yml                      # NEW: build image + package/push chart on tag
└── helm-validate.yml                # NEW: helm lint + helm template on PRs
```

**Structure Decision**: Helm chart goes in `dist/chart/` (kubebuilder default). New workflows are separate files to keep concerns isolated from existing CI.
