# Research: Operator Installation Guide

## R-001: Container Image Registry and Naming

**Decision**: Use `ghcr.io/yanok/jwks-rotater` as the published image name on GitHub Container Registry.

**Rationale**: GHCR is free for public repositories, integrates natively with GitHub Actions for CI/CD publishing, and follows the convention used by most GitHub-hosted Kubernetes operators (e.g., cert-manager, Argo CD). The image name mirrors the GitHub repository path (`yanok/jwks-rotater`).

**Alternatives considered**:
- Docker Hub (`docker.io/yanok/jwks-rotater`) — requires a separate Docker Hub account and has rate limits for anonymous pulls. Rejected for friction.
- Custom/self-hosted registry — unnecessary for a public open-source project.

## R-002: Image Tagging Strategy for dist/install.yaml

**Decision**: Use a `latest` tag in the committed `dist/install.yaml` for the initial release. The Makefile `IMG` default will be updated from `controller:latest` to `ghcr.io/yanok/jwks-rotater:latest`.

**Rationale**: The project does not yet have a release process or semantic versioning for images. Using `latest` provides an immediately functional install manifest. When releases are introduced, `dist/install.yaml` should be regenerated per release with a pinned version tag (e.g., `v0.1.0`), but that is out of scope for this feature.

**Alternatives considered**:
- Pinned version tag (e.g., `v0.1.0`) — premature since no release pipeline exists yet. The guide will document how to override the image tag.
- SHA-based tags — poor UX for documentation; admins expect human-readable tags.

## R-003: Installation Guide Structure

**Decision**: The standalone `docs/install.md` will follow this section order:
1. Prerequisites
2. Quick Install (single-command via `dist/install.yaml`)
3. Install with Kustomize
4. Verify Installation
5. Try It Out (apply sample CR)
6. Uninstall
7. Building from Source
8. Troubleshooting

**Rationale**: This structure follows the "happy path first" pattern used by well-established Kubernetes operator documentation (cert-manager, Prometheus Operator, Argo CD). Prerequisites come first, the simplest install path is presented immediately, progressively more advanced options follow, and troubleshooting is last (consulted only on failure).

**Alternatives considered**:
- Building from source first, then install — rejected because most admins want to install from a pre-built image, not compile from source.
- Separate troubleshooting document — rejected; keeping it in one file reduces context-switching for admins debugging issues.

## R-004: README Update Strategy

**Decision**: Replace the existing "Quick Start > Deploy to a cluster" and "Project Distribution" sections with a concise "Installation" section that provides the single-command install and links to `docs/install.md` for the full guide.

**Rationale**: The current README has installation content scattered across "Quick Start" and "Project Distribution" sections. Consolidating into a single "Installation" section with a link to the full guide reduces duplication and keeps the README concise. The existing "Quick Start > Run locally" section for developers is preserved as-is.

**Alternatives considered**:
- Keep both existing sections and add a third — creates redundancy and confusion about which instructions to follow.
- Remove all install content from README — too aggressive; the README should provide the simplest install path for discoverability.

## R-005: Makefile IMG Default Update

**Decision**: Update the Makefile `IMG` default from `controller:latest` to `ghcr.io/yanok/jwks-rotater:latest`.

**Rationale**: The placeholder `controller:latest` is a kubebuilder scaffold default that doesn't work for actual deployments. Updating the default ensures `make build-installer`, `make deploy`, and `make docker-build` all reference the correct registry by default, reducing the chance of user error.

**Alternatives considered**:
- Keep `controller:latest` and require explicit `IMG=` override — error-prone; admins who forget the override get a broken deployment. Rejected.
- Use an environment variable without a default — rejected for the same reason; a sensible default reduces friction.

## R-006: Troubleshooting Scenarios

**Decision**: Cover these 5 installation failure scenarios in the troubleshooting section:
1. **ImagePullBackOff** — wrong image reference or private registry without pull secrets
2. **CrashLoopBackOff** — leader election failure, insufficient RBAC, or malformed arguments
3. **CRDs not registered** — CRDs not applied or API server version incompatibility
4. **Permission denied errors** — admin lacks cluster-admin privileges
5. **Namespace conflicts** — target namespace already exists with conflicting resources

**Rationale**: These are the most commonly reported issues when deploying Kubernetes operators, based on patterns from cert-manager, OPA Gatekeeper, and Prometheus Operator issue trackers. Each scenario has a clear diagnostic command and resolution.

**Alternatives considered**:
- Include network policy issues — deferred since network policies are not part of the default deployment.
- Include webhook TLS issues — not applicable; webhooks are disabled in the default configuration.
