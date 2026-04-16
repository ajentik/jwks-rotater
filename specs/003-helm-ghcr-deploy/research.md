# Research: Helm Chart & GHCR CI/CD Pipeline

**Date**: 2026-04-16  
**Feature**: 003-helm-ghcr-deploy

## R1: Helm Chart Generation Strategy

**Decision**: Use `kubebuilder edit --plugins=helm/v2-alpha` to scaffold the Helm chart into `dist/chart/`.

**Rationale**: Kubebuilder v4.13.1 includes the `helm.kubebuilder.io/v2-alpha` plugin. It generates the chart from `dist/install.yaml`, producing proper templates with values.yaml for customization. This follows the project's convention of using kubebuilder tooling and avoids manually creating chart templates.

**Alternatives considered**:
- Manual Helm chart creation: More control but high maintenance burden and error-prone when CRDs/RBAC change.
- Third-party tools (helmify): Extra dependency, not aligned with kubebuilder ecosystem.

## R2: CRDs as Helm Templates

**Decision**: After scaffolding, move CRD YAML files from `dist/chart/crds/` into `dist/chart/templates/crd/` so they are managed as templates and updated on `helm upgrade`.

**Rationale**: Helm's built-in `crds/` directory only installs CRDs on first install and never updates them. By placing CRDs in `templates/`, they participate in the normal Helm lifecycle. This is the standard pattern for operators that evolve their CRD schemas.

**Alternatives considered**:
- Keep CRDs in `crds/` directory: Simpler but breaks upgrades when CRD schemas change.
- Separate CRD chart: Adds complexity; unnecessary for a single-operator project.

## R3: GitHub Actions Release Workflow Design

**Decision**: Single `release.yml` workflow triggered on `push` of `v*` tags. Jobs:
1. `test` — run `make test` and `make lint` (gate)
2. `build-and-push-image` — multi-arch build via docker buildx, push to `ghcr.io/yanok/jwks-rotater:<version>` (depends on test)
3. `package-and-push-chart` — `helm package` + `helm push` to `ghcr.io/yanok/jwks-rotater/charts` (depends on build-and-push-image)

A separate trigger on `push` to `main` builds and pushes the `latest` image tag only.

**Rationale**: Single workflow with dependent jobs ensures atomicity — if tests fail, nothing is published. Chart job depends on image job to guarantee the referenced image exists. Using `GITHUB_TOKEN` with `packages: write` permission handles GHCR auth.

**Alternatives considered**:
- Separate workflows for image and chart: Risk of chart referencing a non-existent image if image build fails.
- GitHub Releases trigger: Adds manual step; tag-based trigger is simpler for automation.

## R4: Helm Chart Validation in PR CI

**Decision**: New `helm-validate.yml` workflow triggered on PRs that modify `dist/chart/**` paths. Runs `helm lint dist/chart/` and `helm template jwks-rotater dist/chart/`.

**Rationale**: Path-filtered triggers avoid running helm validation on unrelated PRs. `helm lint` catches structural issues; `helm template` catches rendering errors.

**Alternatives considered**:
- Add to existing lint.yml: Would require helm installation in that workflow and runs on all PRs unnecessarily.
- Only validate at release time: Delays error detection.

## R5: Multi-Architecture Image Build

**Decision**: Use `docker/build-push-action` with `docker buildx` in GitHub Actions. Platforms: `linux/amd64,linux/arm64`. The existing Dockerfile already supports cross-compilation via `TARGETOS`/`TARGETARCH` build args.

**Rationale**: The Dockerfile is already configured for multi-arch builds. The Makefile's `docker-buildx` target builds for 4 platforms, but amd64+arm64 covers the vast majority of production clusters and keeps build times reasonable in CI.

**Alternatives considered**:
- All 4 platforms (add s390x, ppc64le): Significantly longer build times with minimal adoption benefit.
- Single platform: Doesn't meet FR-003.

## R6: GHCR Authentication and Visibility

**Decision**: Use `GITHUB_TOKEN` with `packages: write` permission for pushing. Packages are private by default (matching the private repository). Document that consumers need a PAT with `read:packages` scope.

**Rationale**: `GITHUB_TOKEN` is automatically available in GitHub Actions and requires no secret management. Private visibility matches the private repo requirement from clarification.

**Alternatives considered**:
- Personal Access Token stored as secret: Less secure, requires rotation.
- Public packages: Contradicts the private repository decision.

## R7: Chart Version Management

**Decision**: The release workflow will update `Chart.yaml` version and `appVersion` fields from the git tag before packaging. The `values.yaml` image tag will also be set to match.

**Rationale**: Ensures FR-009 (chart version matches release tag). Updating at build time avoids requiring developers to manually bump versions before tagging.

**Alternatives considered**:
- Manual version bumps in Chart.yaml: Error-prone, easy to forget.
- Separate version file: Unnecessary indirection.
