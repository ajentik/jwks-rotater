# Feature Specification: Helm Chart & GHCR CI/CD Pipeline

**Feature Branch**: `003-helm-ghcr-deploy`  
**Created**: 2026-04-16  
**Status**: Draft  
**Input**: User description: "Add a helm chart deployment strategy. To serve the deployment image, we need to add GitHub Actions to build the image and the helm chart so both can be served via ghcr.io"

## Clarifications

### Session 2026-04-16

- Q: Should GHCR packages be public or private? → A: Private (matching the private repository). Consumers must authenticate to pull.
- Q: Should the release workflow gate on CI (tests/lint) passing? → A: Gated — tests and lint must pass before publishing artifacts.
- Q: How should CRDs be managed in the Helm chart? → A: CRDs as Helm templates with lifecycle hooks, ensuring they are updated on `helm upgrade`.
- Q: Should PRs validate the Helm chart in CI? → A: Yes — run `helm lint` and `helm template` on PRs that modify chart files.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Install Operator via Helm Chart (Priority: P1)

A cluster administrator wants to install the jwks-rotater operator into their Kubernetes cluster using Helm, pulling the chart and container image from GHCR. They run a single `helm install` command referencing the GHCR-hosted chart and the operator deploys successfully.

**Why this priority**: Helm is the primary distribution mechanism requested. Without a working Helm chart, no other story delivers value.

**Independent Test**: Can be tested by running `helm install` from the published GHCR chart URL against a Kubernetes cluster and verifying the operator pod starts and reconciles CRDs.

**Acceptance Scenarios**:

1. **Given** the Helm chart is published to GHCR and the administrator has authenticated to GHCR, **When** they run `helm install jwks-rotater oci://ghcr.io/yanok/jwks-rotater/charts/jwks-rotater --version <version>`, **Then** the operator deploys with correct RBAC, CRDs, and the manager pod reaches Ready state.
2. **Given** a deployed operator via Helm, **When** the administrator inspects the release with `helm list`, **Then** the release appears with correct version and status.
3. **Given** the Helm chart is installed, **When** the administrator creates a JWKSRotation or JWKSRotationPolicy CR, **Then** the operator reconciles it successfully.

---

### User Story 2 - Automated Image Build and Push on Release (Priority: P1)

A maintainer tags a new release in the repository. GitHub Actions automatically builds the multi-architecture container image and pushes it to `ghcr.io/yanok/jwks-rotater` with the appropriate version tag.

**Why this priority**: The container image must be available on GHCR before the Helm chart can reference it. This is a prerequisite for the full deployment pipeline.

**Independent Test**: Can be tested by pushing a version tag to the repository and verifying the container image appears in GHCR with the correct tag and multi-arch manifests.

**Acceptance Scenarios**:

1. **Given** a maintainer pushes a semantic version tag (e.g., `v1.0.0`), **When** GitHub Actions runs, **Then** a multi-architecture container image is built and pushed to `ghcr.io/yanok/jwks-rotater:<version>`.
2. **Given** a push to the main branch, **When** GitHub Actions runs, **Then** a container image is pushed with the `latest` tag.
3. **Given** the image build fails, **When** the workflow completes, **Then** the maintainer receives a clear failure notification and no partial image is published.

---

### User Story 3 - Automated Helm Chart Package and Push on Release (Priority: P2)

A maintainer tags a new release and GitHub Actions automatically packages the Helm chart and pushes it as an OCI artifact to GHCR so consumers can pull it.

**Why this priority**: Builds on the image pipeline (Story 2) to complete the distribution story. Lower priority because the chart can be installed manually from source during early development.

**Independent Test**: Can be tested by pushing a version tag and verifying the Helm chart OCI artifact appears in GHCR with the correct version.

**Acceptance Scenarios**:

1. **Given** a maintainer pushes a semantic version tag, **When** GitHub Actions runs, **Then** the Helm chart is packaged and pushed to `ghcr.io/yanok/jwks-rotater/charts/jwks-rotater:<version>`.
2. **Given** the chart is pushed to GHCR, **When** a consumer runs `helm pull oci://ghcr.io/yanok/jwks-rotater/charts/jwks-rotater --version <version>`, **Then** the chart downloads successfully and contains valid templates.

---

### User Story 4 - Customize Operator Deployment via Helm Values (Priority: P2)

A cluster administrator wants to customize the operator deployment (replica count, resource limits, image pull policy, namespace) by overriding Helm values at install time.

**Why this priority**: Customization is expected for production deployments but the default values should work out-of-the-box for most users.

**Independent Test**: Can be tested by running `helm install` with `--set` overrides and verifying the resulting deployment reflects the customized values.

**Acceptance Scenarios**:

1. **Given** the Helm chart is available, **When** the administrator installs with `--set manager.replicas=2`, **Then** the operator deployment runs 2 replicas.
2. **Given** the Helm chart is available, **When** the administrator installs with custom resource limits, **Then** the manager pod runs with the specified resource constraints.

---

### Edge Cases

- What happens when the GHCR registry is temporarily unavailable during `helm install`?
- What happens when a user attempts to install a chart version that references a container image version not yet published?
- How does the system handle concurrent release workflows triggered by rapid successive tags?
- What happens when the Helm chart is installed in a cluster that already has CRDs from a previous kustomize-based install?
- What happens when a consumer attempts to pull from GHCR without proper authentication?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The project MUST include a Helm chart that deploys the jwks-rotater operator with all required resources (Deployment, ServiceAccount, RBAC, CRDs).
- **FR-002**: The Helm chart MUST be publishable as an OCI artifact to GHCR.
- **FR-003**: The container image MUST be built for multiple architectures (at minimum amd64 and arm64) and pushed to GHCR.
- **FR-004**: A GitHub Actions workflow MUST automatically build and push the container image when a semantic version tag is pushed, only after tests and lint pass successfully.
- **FR-005**: A GitHub Actions workflow MUST automatically package and push the Helm chart to GHCR when a semantic version tag is pushed, only after tests and lint pass successfully.
- **FR-006**: The Helm chart MUST use the container image from GHCR by default (`ghcr.io/yanok/jwks-rotater`).
- **FR-007**: The Helm chart MUST allow customization of common deployment parameters (replicas, resources, image tag, namespace) through values.
- **FR-008**: The container image MUST also be pushed with the `latest` tag on pushes to the main branch.
- **FR-009**: The Helm chart version MUST match the release tag version.
- **FR-010**: The GitHub Actions workflows MUST authenticate to GHCR using the built-in `GITHUB_TOKEN`.
- **FR-011**: GHCR packages (image and chart) MUST be private, requiring consumers to authenticate before pulling.
- **FR-012**: CRDs MUST be managed as Helm templates with lifecycle hooks so they are created on install and updated on upgrade.
- **FR-013**: A CI workflow MUST validate the Helm chart (`helm lint`, `helm template`) on pull requests that modify chart files.

### Key Entities

- **Helm Chart**: The packaged deployment artifact containing Kubernetes manifests and configurable values for the operator.
- **Container Image**: The multi-architecture OCI image containing the compiled operator binary.
- **Release Pipeline**: The automated CI/CD process triggered by git tags that builds and publishes both artifacts.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A new user can install the operator using a single `helm install` command in under 2 minutes.
- **SC-002**: A tagged release produces both a published container image and Helm chart on GHCR within 15 minutes of the tag push.
- **SC-003**: The container image supports at least 2 CPU architectures (amd64, arm64).
- **SC-004**: All Helm chart values have sensible defaults that result in a working operator deployment without any overrides.
- **SC-005**: The release pipeline succeeds on the first attempt for valid releases 95% of the time.

## Assumptions

- The GitHub repository is hosted at `github.com/yanok/jwks-rotater` and has GHCR access enabled.
- The existing Makefile targets (`docker-build`, `docker-buildx`, `build-installer`) will be leveraged where possible in CI workflows.
- Semantic versioning (e.g., `v1.0.0`) is used for release tags.
- The Helm chart will be generated using the kubebuilder Helm plugin (`kubebuilder edit --plugins=helm/v2-alpha`) as the baseline.
- CRDs will be included in the Helm chart (standard for operator distributions).
- The `GITHUB_TOKEN` has sufficient permissions to push packages to GHCR (this is the default for public repositories).
- The repository is private; GHCR packages inherit private visibility and consumers must use a PAT or service account token to pull.
- Existing CI workflows (test, lint, e2e) will continue to run independently and are not modified by this feature.
