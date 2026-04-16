# Data Model: Helm Chart & GHCR CI/CD Pipeline

**Date**: 2026-04-16  
**Feature**: 003-helm-ghcr-deploy

## Overview

This feature has no runtime data model — it adds build/distribution infrastructure. The "entities" are CI/CD artifacts and configuration files.

## Artifacts

### Helm Chart (`dist/chart/`)

| File | Purpose |
|------|---------|
| `Chart.yaml` | Chart metadata: name, version, appVersion, description |
| `values.yaml` | Default configuration: image repo/tag, replicas, resources, service accounts |
| `templates/_helpers.tpl` | Template helper functions (labels, names, selectors) |
| `templates/manager/manager.yaml` | Operator Deployment template |
| `templates/rbac/*.yaml` | ServiceAccount, ClusterRole, ClusterRoleBinding templates |
| `templates/crd/*.yaml` | CRD templates (managed as templates for upgrade support) |

### Key Configuration Values (`values.yaml`)

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `manager.image.repository` | string | `ghcr.io/yanok/jwks-rotater` | Container image repository |
| `manager.image.tag` | string | Chart appVersion | Image tag |
| `manager.image.pullPolicy` | string | `IfNotPresent` | Image pull policy |
| `manager.replicas` | int | `1` | Deployment replica count |
| `manager.resources.limits.cpu` | string | `500m` | CPU limit |
| `manager.resources.limits.memory` | string | `128Mi` | Memory limit |
| `manager.resources.requests.cpu` | string | `10m` | CPU request |
| `manager.resources.requests.memory` | string | `64Mi` | Memory request |

### GitHub Actions Workflows

| Workflow | Trigger | Jobs |
|----------|---------|------|
| `release.yml` | `push` tag `v*` + `push` to `main` | test → build-and-push-image → package-and-push-chart |
| `helm-validate.yml` | PR modifying `dist/chart/**` | lint-and-template |

### OCI Artifacts (GHCR)

| Artifact | GHCR Path | Tags |
|----------|-----------|------|
| Container image | `ghcr.io/yanok/jwks-rotater` | `<semver>`, `latest` (main) |
| Helm chart | `ghcr.io/yanok/jwks-rotater/charts/jwks-rotater` | `<semver>` |
