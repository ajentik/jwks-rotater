# Tasks: Operator Installation Guide

**Input**: Design documents from `/specs/002-operator-install-guide/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md

**Tests**: Not requested for this documentation feature. The existing e2e test suite covers operator deployment.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

## Phase 1: Setup

**Purpose**: Project initialization and directory structure

- [x] T001 Update Makefile IMG default from `controller:latest` to `ghcr.io/yanok/jwks-rotater:latest` in Makefile
- [x] T002 Run `make build-installer IMG=ghcr.io/yanok/jwks-rotater:latest` to generate dist/install.yaml
- [x] T003 Verify dist/install.yaml contains the correct image reference (`ghcr.io/yanok/jwks-rotater:latest`) and no `controller:latest` placeholder
- [x] T004 Create docs/ directory at repository root

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Create the installation guide skeleton with shared structure

- [x] T005 Create docs/install.md with document title, introduction paragraph, and table of contents linking to all 8 sections per research.md R-003

**Checkpoint**: Guide skeleton ready — user story sections can now be written in parallel

---

## Phase 3: User Story 1 — First-Time Cluster Installation (Priority: P1)

**Goal**: Deliver prerequisites, quick install, Kustomize install, and sample CR sections so an admin can install the operator from scratch.

**Independent Test**: Follow the guide on a fresh Kubernetes 1.28+ cluster; confirm the operator pod is running, CRDs are registered, and a sample JWKSRotation resource can be created.

### Implementation for User Story 1

- [x] T006 [US1] Write Prerequisites section in docs/install.md covering Kubernetes 1.28+, kubectl, cluster-admin permissions, and network access to GHCR (FR-001)
- [x] T007 [US1] Write Quick Install section in docs/install.md with single-command `kubectl apply -f` from the raw GitHub URL for dist/install.yaml (FR-002, FR-009)
- [x] T008 [US1] Write Install with Kustomize section in docs/install.md covering git clone, optional customization, and `make deploy` (FR-003, FR-009)
- [x] T009 [US1] Write Try It Out section in docs/install.md with the sample JWKSRotation CR from config/samples/jwks_v1alpha1_jwksrotation.yaml and expected output (FR-006)

**Checkpoint**: An admin can install the operator and apply a sample CR by following docs/install.md

---

## Phase 4: User Story 2 — Verify and Troubleshoot Installation (Priority: P2)

**Goal**: Deliver verification checklist and troubleshooting section so an admin can confirm a healthy installation or diagnose failures.

**Independent Test**: Run verification commands against a working installation; consult troubleshooting for a deliberately misconfigured installation (e.g., wrong image name).

### Implementation for User Story 2

- [x] T010 [US2] Write Verify Installation section in docs/install.md with kubectl commands to check pods, CRDs, RBAC, and service account in jwks-rotater-system namespace (FR-005)
- [x] T011 [US2] Write Troubleshooting section in docs/install.md covering 5 failure scenarios: ImagePullBackOff, CrashLoopBackOff, CRDs not registered, permission denied, namespace conflicts — each with diagnostic command and resolution (FR-008)

**Checkpoint**: An admin can verify a successful installation and diagnose common failures

---

## Phase 5: User Story 3 — Uninstall the Operator (Priority: P3)

**Goal**: Deliver uninstall instructions with appropriate warnings about destructive actions.

**Independent Test**: Perform uninstall after a successful installation; confirm all operator resources (namespace, CRDs, RBAC) are removed.

### Implementation for User Story 3

- [x] T012 [US3] Write Uninstall section in docs/install.md with step-by-step removal of CRs, operator deployment, CRDs, and namespace — including warnings about CRD removal deleting all custom resources (FR-007, FR-010)

**Checkpoint**: An admin can cleanly remove the operator following the documented steps

---

## Phase 6: User Story 4 — Build and Push a Custom Operator Image (Priority: P3)

**Goal**: Deliver build-from-source instructions for admins who need a custom image.

**Independent Test**: Build the image from source, push to a local registry, deploy using the custom image.

### Implementation for User Story 4

- [x] T013 [US4] Write Building from Source section in docs/install.md covering prerequisites (Go 1.25+, Docker), `make docker-build docker-push IMG=<registry>/<name>:<tag>`, and deploying with the custom image via `make deploy IMG=...` (FR-004)

**Checkpoint**: An admin can build, push, and deploy a custom operator image

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: README update, cross-references, and final validation

- [x] T014 [P] Update README.md: replace "Deploy to a cluster" subsection and "Project Distribution" section with a consolidated "Installation" section containing the single-command install and a link to docs/install.md (FR-012)
- [x] T015 [P] Review docs/install.md for internal consistency: verify all namespace references say `jwks-rotater-system`, all image references say `ghcr.io/yanok/jwks-rotater:latest`, and section cross-links work
- [x] T016 Validate dist/install.yaml with `kubectl apply --dry-run=client -f dist/install.yaml` to confirm it is valid and deployable

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on T004 (docs/ directory exists)
- **User Stories (Phases 3–6)**: All depend on T005 (guide skeleton exists)
  - US1, US2, US3, US4 can proceed in parallel after T005
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Phase 7)**: Depends on all user story phases being complete

### User Story Dependencies

- **US1 (P1)**: Depends on T005 only — no dependencies on other stories
- **US2 (P2)**: Depends on T005 only — independently testable
- **US3 (P3)**: Depends on T005 only — independently testable
- **US4 (P3)**: Depends on T005 only — independently testable

### Within Each User Story

- Sections are written sequentially within each story (content builds on prior sections)
- Each story is a complete, independently testable increment of the guide

### Parallel Opportunities

- T001 and T004 can run in parallel (different files, no dependencies)
- After T005, all four user stories (Phases 3–6) can be worked on in parallel
- T014 and T015 can run in parallel (different files)

---

## Parallel Example: User Stories

```text
# After T005 completes, launch all user stories in parallel:
Task: T006–T009 (US1: Installation sections in docs/install.md)
Task: T010–T011 (US2: Verification and troubleshooting in docs/install.md)
Task: T012 (US3: Uninstall section in docs/install.md)
Task: T013 (US4: Build from source section in docs/install.md)

# Note: Since all stories write to the same file (docs/install.md),
# parallel execution requires non-overlapping sections.
# Sequential execution (P1→P2→P3) is recommended for a single implementer.
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001–T004)
2. Complete Phase 2: Foundational (T005)
3. Complete Phase 3: User Story 1 (T006–T009)
4. **STOP and VALIDATE**: Follow the guide on a test cluster
5. The operator can be installed and a sample CR applied

### Incremental Delivery

1. Complete Setup + Foundational → Guide skeleton ready
2. Add US1 → Admin can install → Deploy/Demo (MVP!)
3. Add US2 → Admin can verify and troubleshoot
4. Add US3 + US4 → Admin can uninstall and build from source
5. Polish → README updated, consistency validated, manifest verified

---

## Notes

- All user stories write sections into the same file (docs/install.md) — sequential execution within a single implementer avoids merge conflicts
- dist/install.yaml is a generated artifact — regenerate it if any config/ files change
- The README update (T014) should preserve the existing "Run locally" developer section unchanged
- Commit after each completed user story phase for clean git history
