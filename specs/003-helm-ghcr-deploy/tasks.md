# Tasks: Helm Chart & GHCR CI/CD Pipeline

**Input**: Design documents from `/specs/003-helm-ghcr-deploy/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Chart validation via `helm lint` and `helm template` is included as part of CI workflow tasks (constitution requires test coverage).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: Generate the Helm chart scaffold and ensure prerequisites are in place

- [x] T001 Run `make build-installer IMG=ghcr.io/yanok/jwks-rotater:latest` to regenerate dist/install.yaml
- [x] T002 Run `kubebuilder edit --plugins=helm/v2-alpha` to scaffold Helm chart into dist/chart/
- [x] T003 Verify scaffolded chart structure in dist/chart/ (Chart.yaml, values.yaml, templates/)

---

## Phase 2: Foundational (Helm Chart Customization)

**Purpose**: Customize the scaffolded chart to meet requirements before any workflow can reference it

**CRITICAL**: No workflow tasks can begin until the chart is correctly configured

- [x] T004 Move CRD files from dist/chart/crds/ to dist/chart/templates/crd/ so CRDs are managed as templates and updated on helm upgrade
- [x] T005 Update dist/chart/values.yaml to expose customization parameters: manager.image.repository (default: ghcr.io/yanok/jwks-rotater), manager.image.tag, manager.image.pullPolicy (default: IfNotPresent), manager.replicas (default: 1), manager.resources.limits.cpu (default: 500m), manager.resources.limits.memory (default: 128Mi), manager.resources.requests.cpu (default: 10m), manager.resources.requests.memory (default: 64Mi)
- [x] T006 Update dist/chart/templates/manager/manager.yaml to reference values for image, replicas, and resources instead of hardcoded values
- [x] T007 Verify chart renders correctly: run `helm lint dist/chart/` and `helm template jwks-rotater dist/chart/`
- [x] T008 Add Helm deploy/status/uninstall/history/rollback Makefile targets to Makefile (matching kubebuilder helm plugin conventions)

**Checkpoint**: Helm chart is fully configured and validates locally

---

## Phase 3: User Story 1 - Install Operator via Helm Chart (Priority: P1) MVP

**Goal**: A cluster administrator can install the operator from the local chart with correct RBAC, CRDs, and a running manager pod

**Independent Test**: Run `helm install jwks-rotater dist/chart/ --namespace jwks-rotater-system --create-namespace` against a Kind cluster and verify the operator pod reaches Ready state and reconciles CRs

### Implementation for User Story 1

- [ ] T009 [US1] Test local chart installation on a Kind cluster: create cluster, run `helm install jwks-rotater dist/chart/ --namespace jwks-rotater-system --create-namespace`, verify manager pod is Running
- [ ] T010 [US1] Verify CRDs are installed by running `kubectl get crd jwksrotations.jwks.ajentik.ai jwksrotationpolicies.jwks.ajentik.ai`
- [ ] T011 [US1] Verify helm upgrade updates CRDs: modify a CRD annotation, run `helm upgrade`, confirm the change is applied
- [ ] T012 [US1] Verify values overrides work: run `helm install` with `--set manager.replicas=2` and confirm 2 replicas are running

**Checkpoint**: Operator installs and runs correctly from local Helm chart

---

## Phase 4: User Story 2 - Automated Image Build and Push on Release (Priority: P1)

**Goal**: Pushing a semver tag triggers GitHub Actions to build a multi-arch image and push it to GHCR; pushing to main pushes the latest tag

**Independent Test**: Push a `v*` tag and verify the multi-arch image appears in GHCR with correct tags

### Implementation for User Story 2

- [x] T013 [US2] Create .github/workflows/release.yml with trigger on push of tags matching `v*` and push to `main` branch
- [x] T014 [US2] Add `test` job to .github/workflows/release.yml that runs `make test` and `make lint` as a gate (runs on tag push only)
- [x] T015 [US2] Add `build-and-push-image` job to .github/workflows/release.yml that depends on `test` job, uses docker/setup-buildx-action and docker/build-push-action to build linux/amd64,linux/arm64 and push to ghcr.io/yanok/jwks-rotater with semver tag
- [x] T016 [US2] Configure the `build-and-push-image` job to push with `latest` tag when triggered by push to main branch (no test gate for main pushes)
- [x] T017 [US2] Add GHCR authentication step using docker/login-action with GITHUB_TOKEN and `packages: write` permission in .github/workflows/release.yml

**Checkpoint**: Image build and push pipeline is functional

---

## Phase 5: User Story 3 - Automated Helm Chart Package and Push on Release (Priority: P2)

**Goal**: On semver tag push, after the image is built, the Helm chart is packaged and pushed as an OCI artifact to GHCR

**Independent Test**: Push a `v*` tag and verify the chart OCI artifact appears in GHCR at ghcr.io/yanok/jwks-rotater/charts/jwks-rotater

### Implementation for User Story 3

- [x] T018 [US3] Add `package-and-push-chart` job to .github/workflows/release.yml that depends on `build-and-push-image` job
- [x] T019 [US3] In the `package-and-push-chart` job, add a step to update dist/chart/Chart.yaml version and appVersion fields from the git tag (strip `v` prefix)
- [x] T020 [US3] In the `package-and-push-chart` job, add a step to update dist/chart/values.yaml image tag to match the release version
- [x] T021 [US3] In the `package-and-push-chart` job, add steps to run `helm package dist/chart/` and `helm push` the packaged chart to oci://ghcr.io/yanok/jwks-rotater/charts using GITHUB_TOKEN auth via `helm registry login`

**Checkpoint**: Full release pipeline (test → image → chart) is functional

---

## Phase 6: User Story 4 - Customize Operator Deployment via Helm Values (Priority: P2)

**Goal**: Values overrides (replicas, resources, image tag, pull policy) work correctly at install time

**Independent Test**: Run `helm install` with various `--set` overrides and verify the deployment reflects each customization

### Implementation for User Story 4

- [x] T022 [US4] Verify dist/chart/values.yaml documents all customizable parameters with inline comments explaining each value
- [x] T023 [US4] Verify `helm template jwks-rotater dist/chart/ --set manager.replicas=3 --set manager.image.pullPolicy=Always` renders correct values in the output YAML

**Checkpoint**: All values customization works as documented

---

## Phase 7: PR Chart Validation (Cross-Cutting)

**Purpose**: Add CI validation for chart changes on pull requests

- [x] T024 [P] Create .github/workflows/helm-validate.yml triggered on pull_request with path filter `dist/chart/**`
- [x] T025 In .github/workflows/helm-validate.yml, add job that installs Helm, runs `helm lint dist/chart/` and `helm template jwks-rotater dist/chart/`

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Final validation and documentation

- [x] T026 [P] Commit all generated Helm chart files in dist/chart/
- [x] T027 [P] Verify all workflows have correct permissions block (`contents: read`, `packages: write`)
- [x] T028 Run full local validation: `helm lint dist/chart/` and `helm template jwks-rotater dist/chart/` pass cleanly
- [ ] T029 Run quickstart.md validation: follow steps end-to-end on a fresh Kind cluster

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational - validates chart locally
- **User Story 2 (Phase 4)**: Depends on Foundational - can run in parallel with US1
- **User Story 3 (Phase 5)**: Depends on User Story 2 (chart push needs image job to exist)
- **User Story 4 (Phase 6)**: Depends on Foundational - can run in parallel with US1/US2
- **PR Validation (Phase 7)**: Depends on Foundational (needs chart to exist) - can run in parallel with US2/US3
- **Polish (Phase 8)**: Depends on all previous phases

### User Story Dependencies

- **US1 (P1)**: After Foundational → independently testable
- **US2 (P1)**: After Foundational → independently testable (creates release.yml)
- **US3 (P2)**: After US2 → adds chart job to release.yml created in US2
- **US4 (P2)**: After Foundational → independently testable (validates values)

### Parallel Opportunities

- US1 and US2 can run in parallel after Foundational
- US4 can run in parallel with US1/US2
- T024 (helm-validate workflow) can run in parallel with US2/US3
- T026 and T027 (polish) can run in parallel

---

## Parallel Example: After Foundational Phase

```bash
# These can all start simultaneously after Phase 2:
Task: T009 [US1] Test local chart installation on Kind cluster
Task: T013 [US2] Create .github/workflows/release.yml
Task: T022 [US4] Verify values.yaml documentation
Task: T024 Create .github/workflows/helm-validate.yml
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (scaffold chart)
2. Complete Phase 2: Foundational (customize chart)
3. Complete Phase 3: User Story 1 (local install works)
4. **STOP and VALIDATE**: Install on Kind cluster, verify operator runs
5. Deploy/demo if ready

### Incremental Delivery

1. Setup + Foundational → Chart scaffold ready
2. Add US1 → Local install works → Validate (MVP!)
3. Add US2 → Image auto-builds on tag → Validate
4. Add US3 → Chart auto-publishes on tag → Validate
5. Add US4 → Values customization documented and verified
6. Add PR validation → Chart changes caught early
7. Polish → Everything clean

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Commit after each task or logical group
- The Helm chart is scaffolded by kubebuilder — do not create chart files manually
- CRDs moved to templates/ is a post-scaffold customization step
- Release workflow version injection happens at build time, not in committed files
