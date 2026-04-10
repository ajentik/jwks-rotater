# Tasks: JWKS Rotation Operator

**Input**: Design documents from `/specs/001-jwks-operator/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: TDD is NON-NEGOTIABLE per constitution. All tests MUST be written first and MUST fail before implementation.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: Project initialization and kubebuilder scaffolding

- [ ] T001 Initialize kubebuilder project with `kubebuilder init --domain yanok.io --repo github.com/yanok/jwks-rotater`
- [ ] T002 Scaffold JWKSRotation API with `kubebuilder create api --group jwks --version v1alpha1 --kind JWKSRotation --resource --controller`
- [ ] T003 Scaffold JWKSRotationPolicy API with `kubebuilder create api --group jwks --version v1alpha1 --kind JWKSRotationPolicy --resource --controller`
- [ ] T004 Add go-jose/v4 dependency with `go get github.com/go-jose/go-jose/v4`
- [ ] T005 [P] Configure golangci-lint with `.golangci.yml`
- [ ] T006 [P] Add sample CR manifests in `config/samples/jwks_v1alpha1_jwksrotation.yaml` and `config/samples/jwks_v1alpha1_jwksrotationpolicy.yaml`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: CRD types and core JWKS library that ALL user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### CRD Type Definitions

- [ ] T007 Write tests for JWKSRotation type validation (keyType enum, keySize constraints, retentionPeriod > rotationInterval) in `api/v1alpha1/jwksrotation_types_test.go`
- [ ] T008 Define JWKSRotation spec and status types with kubebuilder markers in `api/v1alpha1/jwksrotation_types.go` — fields: keyType, keySize, rotationInterval, retentionPeriod, targetSecret, targetDeployments, retainSecretsOnDelete; status: lastRotation, nextRotation, activeKeys, conditions
- [ ] T009 Write tests for JWKSRotationPolicy type validation in `api/v1alpha1/jwksrotationpolicy_types_test.go`
- [ ] T010 Define JWKSRotationPolicy spec and status types with kubebuilder markers in `api/v1alpha1/jwksrotationpolicy_types.go` — fields: selector, keyType, keySize, rotationInterval, retentionPeriod; status: matchedDeployments, managedSecrets, conditions
- [ ] T011 Run `make generate` and `make manifests` to generate DeepCopy methods and CRD YAML

### JWKS Key Generation Library

- [ ] T012 [P] Write table-driven tests for RSA key generation (2048, 4096) in `internal/jwks/keygen_test.go` — verify key material, kid via RFC7638 thumbprint, iat timestamp
- [ ] T013 [P] Write table-driven tests for ECDSA key generation (P-256, P-384) in `internal/jwks/keygen_test.go`
- [ ] T014 Implement key generation for RSA and ECDSA with kid and iat in `internal/jwks/keygen.go` — use crypto/rand, go-jose/v4 for JWK construction

### JWKS Keystore Library

- [ ] T015 Write table-driven tests for JWKS assembly (add key, serialize to JSON, deserialize) in `internal/jwks/keystore_test.go`
- [ ] T016 [P] Write table-driven tests for public key derivation (strip private material) in `internal/jwks/keystore_test.go`
- [ ] T017 Implement JWKS keystore: add key, serialize/deserialize, public-only derivation in `internal/jwks/keystore.go`

### Secret Construction Helpers

- [ ] T018 Write tests for Secret construction (Opaque type, jwks.json data key, labels, owner reference) in `internal/jwks/secret_test.go`
- [ ] T019 Implement Secret builder: construct private and public Secrets from JWKS in `internal/jwks/secret.go`

**Checkpoint**: Foundation ready — CRD types defined, key generation and JWKS assembly independently tested, Secret helpers ready

---

## Phase 3: User Story 1 — Declare JWKS Rotation for a Service (Priority: P1) 🎯 MVP

**Goal**: A user applies a JWKSRotation CR and the operator creates a Secret with an initial JWKS key

**Independent Test**: Apply a JWKSRotation CR → verify target Secret created with valid JWKS containing one key, public Secret created, CR status updated

### Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T020 [P] [US1] Write envtest test: applying a JWKSRotation CR creates private Secret with one key and public Secret in `internal/controller/jwksrotation_controller_test.go`
- [ ] T021 [P] [US1] Write envtest test: CR with invalid config (bad keyType) sets Error condition on status in `internal/controller/jwksrotation_controller_test.go`
- [ ] T022 [P] [US1] Write envtest test: CR referencing existing Secret adopts it and appends a key in `internal/controller/jwksrotation_controller_test.go`
- [ ] T023 [P] [US1] Write envtest test: CR status reflects activeKeys count and lastRotation timestamp in `internal/controller/jwksrotation_controller_test.go`

### Implementation for User Story 1

- [ ] T024 [US1] Implement JWKSRotation controller Reconcile: initial key generation path in `internal/controller/jwksrotation_controller.go` — on new CR: generate key, create private+public Secrets with owner reference, set Ready condition
- [ ] T025 [US1] Implement validation logic: reject invalid keyType/keySize, set Error condition in `internal/controller/jwksrotation_controller.go`
- [ ] T026 [US1] Implement existing Secret adoption: detect existing Secret, parse JWKS, append new key in `internal/controller/jwksrotation_controller.go`
- [ ] T027 [US1] Implement status update: set lastRotation, nextRotation, activeKeys on CR status in `internal/controller/jwksrotation_controller.go`
- [ ] T028 [US1] Add RBAC markers for Secrets (get, list, watch, create, update, patch, delete) and Events (create, patch) in `internal/controller/jwksrotation_controller.go`
- [ ] T029 [US1] Emit Kubernetes Events: KeyRotated on initial creation, InvalidConfig on validation failure in `internal/controller/jwksrotation_controller.go`
- [ ] T030 [US1] Run `make manifests` to regenerate RBAC from markers

**Checkpoint**: User Story 1 fully functional — applying a CR creates a JWKS Secret with one key

---

## Phase 4: User Story 2 — Automatic Key Rotation on Schedule (Priority: P1)

**Goal**: The operator generates a new key at the configured interval and appends it to the JWKS

**Independent Test**: Apply CR with short rotation interval → wait → verify JWKS contains two keys, status updated

### Tests for User Story 2

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T031 [P] [US2] Write envtest test: after rotation interval elapses, new key is appended to JWKS Secret in `internal/controller/jwksrotation_controller_test.go`
- [ ] T032 [P] [US2] Write envtest test: after rotation, CR status shows updated activeKeys, lastRotation, nextRotation in `internal/controller/jwksrotation_controller_test.go`
- [ ] T033 [P] [US2] Write envtest test: operator detects overdue rotation after restart and rotates immediately in `internal/controller/jwksrotation_controller_test.go`
- [ ] T034 [P] [US2] Write envtest test: public Secret is updated in sync after rotation in `internal/controller/jwksrotation_controller_test.go`

### Implementation for User Story 2

- [ ] T035 [US2] Implement rotation scheduling: compare lastRotation + rotationInterval vs now, generate new key if due in `internal/controller/jwksrotation_controller.go`
- [ ] T036 [US2] Implement RequeueAfter: return time until next rotation from Reconcile in `internal/controller/jwksrotation_controller.go`
- [ ] T037 [US2] Update public Secret after each rotation in `internal/controller/jwksrotation_controller.go`
- [ ] T038 [US2] Emit KeyRotated event on scheduled rotation in `internal/controller/jwksrotation_controller.go`

**Checkpoint**: User Stories 1 AND 2 work — CR creates initial key, then rotates on schedule

---

## Phase 5: User Story 3 — Old Key Cleanup After Retention Period (Priority: P1)

**Goal**: Keys older than the retention period are removed, JWKS never left empty

**Independent Test**: Create CR with short retention → verify expired keys removed, at least one key always remains

### Tests for User Story 3

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T039 [P] [US3] Write table-driven tests for retention calculation (key age vs retention period) in `internal/jwks/keystore_test.go`
- [ ] T040 [P] [US3] Write envtest test: keys older than retention period are removed from JWKS Secret in `internal/controller/jwksrotation_controller_test.go`
- [ ] T041 [P] [US3] Write envtest test: when all keys are expired, most recent is retained and new key generated in `internal/controller/jwksrotation_controller_test.go`
- [ ] T042 [P] [US3] Write envtest test: public Secret updated after cleanup in `internal/controller/jwksrotation_controller_test.go`

### Implementation for User Story 3

- [ ] T043 [US3] Implement retention filtering in keystore: remove keys with iat older than retention period, keep at least one in `internal/jwks/keystore.go`
- [ ] T044 [US3] Integrate retention cleanup into Reconcile: run after rotation check, update both Secrets in `internal/controller/jwksrotation_controller.go`
- [ ] T045 [US3] Emit KeyExpired event for each removed key in `internal/controller/jwksrotation_controller.go`
- [ ] T046 [US3] Update status activeKeys after cleanup in `internal/controller/jwksrotation_controller.go`

**Checkpoint**: Full rotation lifecycle works — create, rotate, cleanup

---

## Phase 6: User Story 4 — Trigger Rolling Restart of Target Deployments (Priority: P2)

**Goal**: On rotation, optionally restart specified Deployments via annotation patch

**Independent Test**: Apply CR with targetDeployments → trigger rotation → verify Deployments get restart annotation

### Tests for User Story 4

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T047 [P] [US4] Write envtest test: after rotation, target Deployments receive restart annotation in `internal/controller/jwksrotation_controller_test.go`
- [ ] T048 [P] [US4] Write envtest test: CR with no targetDeployments does not patch any Deployments in `internal/controller/jwksrotation_controller_test.go`
- [ ] T049 [P] [US4] Write envtest test: missing target Deployment logs warning and sets Degraded condition without blocking rotation in `internal/controller/jwksrotation_controller_test.go`

### Implementation for User Story 4

- [ ] T050 [US4] Implement Deployment restart: patch annotation `kubectl.kubernetes.io/restartedAt` on each target Deployment in `internal/controller/jwksrotation_controller.go`
- [ ] T051 [US4] Handle missing Deployment: log warning, set Degraded condition, continue in `internal/controller/jwksrotation_controller.go`
- [ ] T052 [US4] Add RBAC marker for Deployments (get, list, watch, patch) and run `make manifests` in `internal/controller/jwksrotation_controller.go`

**Checkpoint**: Rotation triggers rolling restart of target Deployments

---

## Phase 7: Finalizer & Deletion (Cross-cutting, required by US1-US4)

**Goal**: Clean up Secrets on CR deletion, respect retainSecretsOnDelete

### Tests

- [ ] T053 [P] Write envtest test: deleting JWKSRotation CR with retainSecretsOnDelete=false deletes both Secrets in `internal/controller/jwksrotation_controller_test.go`
- [ ] T054 [P] Write envtest test: deleting JWKSRotation CR with retainSecretsOnDelete=true leaves Secrets in place in `internal/controller/jwksrotation_controller_test.go`
- [ ] T055 [P] Write envtest test: externally deleted Secret is recreated on next reconciliation in `internal/controller/jwksrotation_controller_test.go`

### Implementation

- [ ] T056 Implement finalizer `jwks.yanok.io/secret-cleanup`: add on CR creation, handle on deletion in `internal/controller/jwksrotation_controller.go`
- [ ] T057 Implement Secret recreation: detect missing Secret, regenerate key, emit SecretRecreated event in `internal/controller/jwksrotation_controller.go`

**Checkpoint**: Full lifecycle — create, rotate, cleanup, delete, recreate

---

## Phase 8: User Story 5 — Auto-Discovery via Label Selector (Priority: P3)

**Goal**: JWKSRotationPolicy applies default rotation to Deployments matching a label selector

**Independent Test**: Apply JWKSRotationPolicy → label a Deployment → verify Secret `<deployment-name>-jwks` created

### Tests for User Story 5

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T058 [P] [US5] Write envtest test: Deployment with matching label gets a JWKS Secret created in `internal/controller/jwksrotationpolicy_controller_test.go`
- [ ] T059 [P] [US5] Write envtest test: Deployment with explicit JWKSRotation CR is skipped by policy in `internal/controller/jwksrotationpolicy_controller_test.go`
- [ ] T060 [P] [US5] Write envtest test: policy status reflects matchedDeployments and managedSecrets counts in `internal/controller/jwksrotationpolicy_controller_test.go`

### Implementation for User Story 5

- [ ] T061 [US5] Implement JWKSRotationPolicy controller: watch Deployments, list by label selector in `internal/controller/jwksrotationpolicy_controller.go`
- [ ] T062 [US5] Implement Secret creation per matched Deployment using `<deployment-name>-jwks` naming in `internal/controller/jwksrotationpolicy_controller.go`
- [ ] T063 [US5] Implement conflict detection: skip Deployment if a JWKSRotation CR already targets its Secret in `internal/controller/jwksrotationpolicy_controller.go`
- [ ] T064 [US5] Implement rotation and retention for policy-managed Secrets (reuse keystore logic) in `internal/controller/jwksrotationpolicy_controller.go`
- [ ] T065 [US5] Update policy status: matchedDeployments, managedSecrets, conditions in `internal/controller/jwksrotationpolicy_controller.go`
- [ ] T066 [US5] Add RBAC markers and run `make manifests` in `internal/controller/jwksrotationpolicy_controller.go`

**Checkpoint**: Policy auto-discovers Deployments and manages JWKS Secrets for them

---

## Phase 9: Observability & Metrics

**Purpose**: Prometheus metrics and structured logging across all controllers

- [ ] T067 [P] Write tests for metric registration and increment in `internal/controller/metrics_test.go`
- [ ] T068 Register Prometheus metrics (jwks_rotation_total, jwks_rotation_errors_total, jwks_active_keys, jwks_oldest_key_age_seconds, jwks_reconcile_duration_seconds) in `internal/controller/metrics.go`
- [ ] T069 Instrument JWKSRotation controller Reconcile with metric updates in `internal/controller/jwksrotation_controller.go`
- [ ] T070 Instrument JWKSRotationPolicy controller Reconcile with metric updates in `internal/controller/jwksrotationpolicy_controller.go`
- [ ] T071 Verify structured logging uses logr with consistent key-value pairs across all controllers

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Final validation and production readiness

- [ ] T072 [P] Write Dockerfile (multi-stage build) at repository root
- [ ] T073 [P] Verify leader election is configured in `cmd/main.go`
- [ ] T074 Run `make test` — all unit and envtest tests pass
- [ ] T075 Run `go vet ./...` and `golangci-lint run` — zero issues
- [ ] T076 Verify all generated manifests are committed (CRDs, RBAC, DeepCopy)
- [ ] T077 Run quickstart.md validation: apply sample CR to cluster, verify Secret creation and rotation

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup — BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational
- **User Story 2 (Phase 4)**: Depends on User Story 1 (builds on reconciler)
- **User Story 3 (Phase 5)**: Depends on User Story 2 (cleanup runs after rotation)
- **User Story 4 (Phase 6)**: Depends on User Story 2 (restart triggered by rotation)
- **Finalizer (Phase 7)**: Can start after User Story 1, independent of US2-US4
- **User Story 5 (Phase 8)**: Depends on Foundational, independent of US1-US4 (separate controller)
- **Observability (Phase 9)**: Depends on all controllers being implemented
- **Polish (Phase 10)**: Depends on all phases complete

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Keystore/library changes before controller changes
- Controller logic before RBAC/event emission
- Status updates after core logic

### Parallel Opportunities

- T005, T006: Setup linting and samples (parallel)
- T012, T013: RSA and ECDSA key gen tests (parallel)
- T015, T016: Keystore assembly and public derivation tests (parallel)
- T020-T023: All US1 controller tests (parallel)
- T031-T034: All US2 controller tests (parallel)
- T039-T042: All US3 tests (parallel)
- T047-T049: All US4 tests (parallel)
- T053-T055: All finalizer tests (parallel)
- T058-T060: All US5 tests (parallel)
- Phase 7 (Finalizer) and Phase 8 (US5) can run in parallel after Phase 3

---

## Implementation Strategy

### MVP First (User Stories 1-3)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL — blocks all stories)
3. Complete Phase 3: User Story 1 → **STOP and VALIDATE**
4. Complete Phase 4: User Story 2 → **STOP and VALIDATE**
5. Complete Phase 5: User Story 3 → **STOP and VALIDATE**
6. At this point: full rotation lifecycle works (create → rotate → cleanup)

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. Add US1 → Test independently → Secret provisioning works (MVP!)
3. Add US2 → Test independently → Scheduled rotation works
4. Add US3 → Test independently → Retention cleanup works
5. Add Finalizer → Test independently → Deletion cleanup works
6. Add US4 → Test independently → Deployment restarts work
7. Add US5 → Test independently → Auto-discovery works
8. Add Observability → Metrics and logging complete
9. Polish → Production ready

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- TDD is NON-NEGOTIABLE: write tests first, verify they fail, then implement
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
