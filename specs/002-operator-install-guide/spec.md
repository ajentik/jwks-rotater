# Feature Specification: Operator Installation Guide

**Feature Branch**: `002-operator-install-guide`  
**Created**: 2026-04-14  
**Status**: Draft  
**Input**: User description: "As a Kubernetes admin, I want to know how to install this operator on my cluster."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - First-Time Cluster Installation (Priority: P1)

A Kubernetes administrator wants to install the JWKS Rotation Operator on their cluster for the first time. They need clear, step-by-step instructions that cover prerequisites, installation commands, and verification that the operator is running correctly.

**Why this priority**: Without a successful first installation, no other operator functionality can be used. This is the gateway to all value the operator provides.

**Independent Test**: Can be fully tested by following the installation steps on a fresh Kubernetes cluster and confirming the operator pod is running, CRDs are registered, and a sample JWKSRotation resource can be created.

**Acceptance Scenarios**:

1. **Given** a Kubernetes 1.28+ cluster with no prior operator installation, **When** the admin follows the installation instructions, **Then** the operator is deployed to a dedicated namespace, CRDs are registered, and the controller pod reaches Ready state.
2. **Given** a completed installation, **When** the admin applies the provided sample resource, **Then** the operator creates the expected JWKS Secrets and reports status on the custom resource.
3. **Given** a completed installation, **When** the admin checks cluster resources, **Then** all expected RBAC roles, service accounts, and services are present.

---

### User Story 2 - Verify and Troubleshoot Installation (Priority: P2)

After installing, a Kubernetes administrator wants to verify that the operator is functioning correctly and has the information needed to diagnose common installation problems.

**Why this priority**: Verification and troubleshooting are essential follow-ups to installation. Without them, admins cannot confirm success or recover from failures.

**Independent Test**: Can be tested by providing a checklist of verification commands and common failure scenarios with resolution steps, then confirming an admin can diagnose a deliberately misconfigured installation.

**Acceptance Scenarios**:

1. **Given** a successful installation, **When** the admin runs the documented verification commands, **Then** they see confirmation that all components are healthy (pods running, CRDs registered, RBAC in place).
2. **Given** a failed installation (e.g., image pull error, insufficient RBAC), **When** the admin consults the troubleshooting section, **Then** they find the relevant failure scenario and resolution steps.
3. **Given** the operator pod is in CrashLoopBackOff, **When** the admin follows troubleshooting guidance, **Then** they can identify the root cause from logs and take corrective action.

---

### User Story 3 - Uninstall the Operator (Priority: P3)

A Kubernetes administrator wants to cleanly remove the operator and all its resources from the cluster.

**Why this priority**: Clean removal is necessary for lifecycle management, but less frequently needed than installation and verification.

**Independent Test**: Can be tested by performing an uninstall after a successful installation and confirming all operator resources (namespace, CRDs, RBAC, deployments) are removed.

**Acceptance Scenarios**:

1. **Given** a running operator installation, **When** the admin follows the uninstall instructions, **Then** the operator deployment, RBAC resources, and namespace are removed.
2. **Given** a running operator with existing JWKSRotation custom resources, **When** the admin follows the uninstall instructions, **Then** the CRDs and all custom resources are removed (with a clear warning about data loss).

---

### User Story 4 - Build and Push a Custom Operator Image (Priority: P3)

A Kubernetes administrator or developer wants to build the operator container image from source and push it to their own registry before deploying.

**Why this priority**: Required for environments without access to a pre-published image, but secondary to the core install flow.

**Independent Test**: Can be tested by building the image, pushing to a local registry, and deploying from that registry.

**Acceptance Scenarios**:

1. **Given** the operator source code and a container build tool, **When** the admin follows the build instructions, **Then** a container image is produced and can be pushed to their chosen registry.
2. **Given** a custom-built image in a private registry, **When** the admin deploys the operator referencing that image, **Then** the operator runs successfully using the custom image.

---

### Edge Cases

- What happens when the admin's cluster version is below 1.28?
- How does the system handle installation when CRDs from a previous version already exist?
- What happens if the operator image cannot be pulled (private registry without image pull secrets)?
- How does the admin install when they lack cluster-admin privileges?
- What happens if the target namespace already exists?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The installation guide MUST list all prerequisites (cluster version, tools, permissions) before any installation steps
- **FR-002**: The installation guide MUST provide a single-command installation path using a pre-committed consolidated manifest file (`dist/install.yaml`) that admins can apply directly from the repository URL without cloning
- **FR-003**: The installation guide MUST provide a Kustomize-based installation path for admins who want more control
- **FR-004**: The installation guide MUST include image build and push instructions for admins deploying from source
- **FR-005**: The installation guide MUST include post-installation verification steps that confirm CRDs, RBAC, and the controller pod are correctly deployed
- **FR-006**: The installation guide MUST include a sample custom resource that admins can apply immediately to test the operator
- **FR-007**: The installation guide MUST provide uninstall instructions that cleanly remove all operator resources
- **FR-008**: The installation guide MUST include a troubleshooting section covering common installation failures (image pull errors, RBAC issues, pod crash loops)
- **FR-009**: The installation guide MUST document the namespace the operator deploys to and the RBAC permissions it requires
- **FR-010**: The installation guide MUST warn about destructive actions (CRD removal deletes all custom resources)
- **FR-011**: The installation guide MUST be delivered as a detailed standalone document in the repository (e.g., `docs/install.md`)
- **FR-012**: The README MUST include a concise installation summary section that links to the full standalone guide
- **FR-013**: The pre-committed `dist/install.yaml` MUST reference a published container image on a public registry as the default operator image

### Key Entities

- **Operator Deployment**: The controller manager that runs reconciliation loops, deployed as a single-replica Deployment in a dedicated namespace
- **Custom Resource Definitions (CRDs)**: JWKSRotation (namespace-scoped) and JWKSRotationPolicy (cluster-scoped), which define the operator's API surface
- **RBAC Resources**: ClusterRoles, ClusterRoleBindings, Roles, and RoleBindings that grant the operator permissions to manage Secrets, Deployments, and its own custom resources
- **Container Image**: The operator's packaged runtime artifact, published to a public container registry (GHCR) and embedded in the install manifest

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An administrator with Kubernetes experience can complete the installation in under 10 minutes by following the guide
- **SC-002**: 95% of administrators can verify a successful installation on the first attempt using the documented verification steps
- **SC-003**: The troubleshooting section covers at least the 5 most common installation failure scenarios
- **SC-004**: The guide supports both quick-start (single command) and customizable (Kustomize) installation paths
- **SC-005**: An administrator can cleanly uninstall the operator and confirm all resources are removed in under 5 minutes

## Clarifications

### Session 2026-04-14

- Q: Where should the installation guide be delivered? → A: Both a detailed standalone document (e.g., `docs/install.md`) and a concise summary section in the README that links to the full guide.
- Q: Should the install manifest be pre-committed or built by admins? → A: Pre-generate and commit `dist/install.yaml` to the repo so admins can install via a single `kubectl apply -f <URL>` command without cloning the repo.
- Q: Should the guide reference a published container image or require admins to build their own? → A: Reference a published image on GHCR (`ghcr.io/yanok/jwks-rotater`) as the default. The pre-committed `dist/install.yaml` must embed this image reference.

## Assumptions

- The target audience is Kubernetes administrators with working knowledge of kubectl, namespaces, and RBAC concepts
- The admin has cluster-admin or equivalent privileges sufficient to create CRDs, ClusterRoles, and namespaces
- The cluster has outbound network access to pull container images from GHCR (or the admin has a pre-loaded image in a local registry)
- The installation guide covers the operator itself; configuring individual JWKSRotation or JWKSRotationPolicy resources beyond a basic sample is out of scope
- No Helm chart is available; installation is via consolidated YAML manifest or Kustomize
- cert-manager is not required for the default installation
- The operator container image is published to `ghcr.io/yanok/jwks-rotater` and referenced in the committed `dist/install.yaml`
