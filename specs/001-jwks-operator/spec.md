# Feature Specification: JWKS Rotation Operator

**Feature Branch**: `001-jwks-operator`
**Created**: 2026-04-10
**Status**: Draft
**Input**: User description: "Kubernetes operator that manages JWKS key rotation across deployments, with CRD-based configuration, automatic key generation, configurable old key retention and cleanup, and deployment restart support"

## Clarifications

### Session 2026-04-10

- Q: Should the JWKS Secret contain only public keys (verification) or private+public key pairs (signing)? → A: Private+public key pairs for signing, with the operator also deriving a separate public-only JWKS for verifiers.
- Q: What level of operational visibility should the operator provide? → A: CR status + standard operator metrics (rotation count, key age, errors, reconciliation latency) + Kubernetes Events emitted on rotation, cleanup, and errors.
- Q: How should the JWKS be stored within the Kubernetes Secret? → A: Single data key `jwks.json` in each Secret. Public Secret is automatically named `<target-secret-name>-public`.
- Q: What happens to Secrets when a JWKSRotation CR is deleted? → A: Clean up both private and public Secrets by default via finalizer. Opt-out with `spec.retainSecretsOnDelete: true` to leave Secrets in place.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Declare JWKS Rotation for a Service (Priority: P1)

A platform team member creates a JWKSRotation custom resource to declare that a specific service needs automatic JWKS key rotation. The operator picks up the resource, generates an initial JWKS key set, and stores it in the specified Kubernetes Secret.

**Why this priority**: Without the ability to declare and provision an initial key set, no other functionality is possible. This is the foundational capability.

**Independent Test**: Can be fully tested by applying a JWKSRotation CR and verifying that the target Secret is created with a valid JWKS containing one key.

**Acceptance Scenarios**:

1. **Given** a cluster with the operator running, **When** a user applies a JWKSRotation CR specifying a target Secret name, key type, and key size, **Then** the operator creates the target Secret containing a valid JWKS with one key and updates the CR status with the current key count and last rotation timestamp.
2. **Given** a JWKSRotation CR referencing a Secret that already exists, **When** the operator reconciles, **Then** it adopts the existing Secret and appends a new key without removing existing keys.
3. **Given** a JWKSRotation CR with invalid configuration (e.g., unsupported key type), **When** the operator reconciles, **Then** it sets a failure condition on the CR status with a human-readable error message and does not create or modify any Secret.

---

### User Story 2 - Automatic Key Rotation on Schedule (Priority: P1)

The operator automatically generates a new key and appends it to the JWKS at the interval specified in the JWKSRotation CR. Existing keys remain available so that tokens signed with older keys can still be verified during the transition window.

**Why this priority**: Rotation is the core value proposition of the operator. Without scheduled rotation, this is just a key provisioner.

**Independent Test**: Can be tested by applying a CR with a short rotation interval, waiting for the interval to pass, and verifying that the JWKS Secret now contains two keys.

**Acceptance Scenarios**:

1. **Given** a JWKSRotation CR with a rotation interval of 1 hour and an initial key already provisioned, **When** 1 hour elapses, **Then** the operator generates a new key, appends it to the JWKS Secret, and updates the CR status with the new key count, last rotation timestamp, and next rotation time.
2. **Given** the operator is restarted after missing a scheduled rotation, **When** it reconciles, **Then** it detects that rotation is overdue and performs an immediate rotation.

---

### User Story 3 - Old Key Cleanup After Retention Period (Priority: P1)

The operator removes keys that have exceeded the configured retention period, ensuring the JWKS does not grow unbounded while still allowing enough time for dependent services to transition.

**Why this priority**: Without cleanup, the JWKS grows indefinitely. Retention is tightly coupled with rotation and is essential for a production-ready system.

**Independent Test**: Can be tested by creating a CR with a short retention period and verifying that keys older than the retention period are removed from the JWKS Secret on the next reconciliation.

**Acceptance Scenarios**:

1. **Given** a JWKS Secret with three keys where the oldest key was created beyond the retention period, **When** the operator reconciles, **Then** it removes the expired key and updates the Secret to contain only the two remaining keys.
2. **Given** all keys in the JWKS are older than the retention period, **When** the operator reconciles, **Then** it retains at least the most recent key (never leaves the JWKS empty) and generates a new key.

---

### User Story 4 - Trigger Rolling Restart of Target Deployments (Priority: P2)

When a key rotation occurs, the operator optionally triggers a rolling restart of specified Deployments so that pods pick up the updated Secret contents.

**Why this priority**: Important for services that read the JWKS at startup rather than watching the Secret. Without this, teams must manually restart services or build their own file-watching logic.

**Independent Test**: Can be tested by applying a CR with target deployments listed, triggering a rotation, and verifying that the target Deployments receive an annotation update that initiates a rollout.

**Acceptance Scenarios**:

1. **Given** a JWKSRotation CR that lists two target Deployments, **When** a key rotation occurs, **Then** the operator patches each Deployment with a restart annotation causing a rolling restart.
2. **Given** a JWKSRotation CR with no target Deployments specified, **When** a key rotation occurs, **Then** no Deployments are restarted.
3. **Given** a target Deployment that does not exist, **When** the operator attempts to trigger a restart, **Then** it logs a warning and sets a degraded condition on the CR status but does not block rotation.

---

### User Story 5 - Auto-Discovery via Label Selector (Priority: P3)

A cluster-wide JWKSRotationPolicy resource allows platform teams to define a default rotation configuration that automatically applies to any Deployment matching a label selector, without requiring a per-service JWKSRotation CR.

**Why this priority**: Reduces operational overhead at scale, but teams can function with per-service CRs. This is a convenience layer.

**Independent Test**: Can be tested by applying a JWKSRotationPolicy with a label selector, labeling a Deployment, and verifying that a JWKS Secret is created for it using a conventional naming pattern.

**Acceptance Scenarios**:

1. **Given** a JWKSRotationPolicy selecting Deployments with label `jwks.ajentik.ai/rotate: "true"`, **When** a Deployment with that label exists, **Then** the operator creates a Secret named `<deployment-name>-jwks` in the same namespace with a valid JWKS.
2. **Given** a Deployment that matches a JWKSRotationPolicy and also has a dedicated JWKSRotation CR, **When** the operator reconciles, **Then** the explicit JWKSRotation CR takes precedence over the policy defaults.

---

### Edge Cases

- What happens when the target Secret is deleted externally? The operator should detect the missing Secret on next reconciliation and recreate it with a new key.
- What happens when two JWKSRotation CRs target the same Secret? The operator should reject the second CR with a conflict status condition.
- What happens when the operator loses leader election during a rotation? The new leader should detect the incomplete state and complete or retry the rotation.
- What happens when the cluster clock skews? Rotation and retention calculations should use the key's embedded creation timestamp, not wall clock comparisons alone.
- What happens when a JWKSRotation CR is deleted? The operator uses a finalizer to delete both Secrets by default. If `retainSecretsOnDelete` is true, Secrets are left in place as orphans.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a JWKSRotation custom resource definition that accepts key type, key size, rotation interval, retention period, target Secret reference, optional target Deployment references, and an optional `retainSecretsOnDelete` flag (default: false).
- **FR-002**: System MUST generate cryptographic key pairs (RSA and ECDSA), store the private+public JWKS in a Kubernetes Secret, and derive a separate public-only JWKS Secret for use by token verifiers.
- **FR-003**: System MUST assign a unique key ID (`kid`) to each generated key.
- **FR-004**: System MUST embed a creation timestamp in each key's metadata to support retention calculations.
- **FR-005**: System MUST automatically rotate keys at the configured interval by generating a new key and appending it to the JWKS.
- **FR-006**: System MUST remove keys that have exceeded the configured retention period, while never leaving the JWKS empty.
- **FR-007**: System MUST optionally trigger rolling restarts of specified Deployments when rotation occurs.
- **FR-008**: System MUST report rotation status (last rotation time, next rotation time, active key count, error conditions) on the CR status subresource.
- **FR-009**: System MUST support namespace-scoped JWKSRotation resources (operator manages Secrets within the same namespace as the CR).
- **FR-010**: System MUST provide a JWKSRotationPolicy CRD for cluster-wide auto-discovery of Deployments via label selectors.
- **FR-011**: System MUST use leader election to ensure only one operator instance performs rotations at a time.
- **FR-012**: System MUST recreate the target Secret if it is deleted externally.
- **FR-013**: System MUST expose operator metrics including rotation event count, active key age, error count, and reconciliation latency.
- **FR-014**: System MUST emit Kubernetes Events on the JWKSRotation resource for key rotation, key cleanup, rotation failures, and Secret recreation.
- **FR-015**: System MUST store the JWKS under the data key `jwks.json` within each Secret.
- **FR-016**: System MUST automatically create a public-only Secret named `<target-secret-name>-public` alongside the private Secret, kept in sync on every rotation and cleanup.
- **FR-017**: System MUST use a finalizer to delete both private and public Secrets when a JWKSRotation CR is deleted, unless `retainSecretsOnDelete` is set to true.

### Key Entities

- **JWKSRotation**: A namespace-scoped custom resource that declares the desired rotation configuration for a single JWKS Secret. Attributes include key type, key size, rotation interval, retention period, target Secret name, and optional target Deployments.
- **JWKSRotationPolicy**: A cluster-scoped custom resource that applies default rotation configuration to Deployments matching a label selector.
- **JWKS Private Secret**: A Kubernetes Secret containing the full JSON Web Key Set with private+public key pairs under the data key `jwks.json`. Used by signing services. Named as specified in the CR's `targetSecret.name`. Each key includes a `kid`, full key material, and creation metadata.
- **JWKS Public Secret**: A derived Kubernetes Secret containing only the public keys under the data key `jwks.json`. Used by token verifiers. Named `<target-secret-name>-public`. Automatically kept in sync with the private Secret.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A new JWKS key is available in the target Secret within 60 seconds of the configured rotation interval elapsing.
- **SC-002**: Expired keys are removed within one reconciliation cycle after exceeding the retention period.
- **SC-003**: Target Deployments begin a rolling restart within 30 seconds of a key rotation event.
- **SC-004**: The operator recovers from missed rotations (e.g., due to downtime) within 2 minutes of regaining availability.
- **SC-005**: Services consuming the JWKS experience zero verification failures during key rotation (old keys remain valid for the full retention window).

## Assumptions

- The operator runs inside the Kubernetes cluster with appropriate RBAC permissions to manage Secrets, read Deployments, and patch Deployment annotations.
- Services consuming the JWKS read the Secret directly (mounted as a volume or fetched via the Kubernetes API) rather than via an external JWKS endpoint.
- The cluster has a functioning leader election mechanism (Kubernetes Lease objects).
- Key sizes and types follow standard cryptographic practices (RSA 2048+ or ECDSA P-256/P-384).
- The retention period is always longer than the rotation interval.
