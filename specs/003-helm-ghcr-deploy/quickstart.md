# Quickstart: Helm Chart & GHCR CI/CD Pipeline

**Date**: 2026-04-16  
**Feature**: 003-helm-ghcr-deploy

## Prerequisites

- `kubebuilder` v4.13.1 installed
- `helm` v3.x installed
- `docker` with buildx support
- Access to the GitHub repository

## Step 1: Generate Helm Chart

```bash
# Ensure dist/install.yaml is up to date
make build-installer IMG=ghcr.io/yanok/jwks-rotater:latest

# Generate the Helm chart
kubebuilder edit --plugins=helm/v2-alpha
```

This creates `dist/chart/` with the full Helm chart.

## Step 2: Customize CRD Management

Move CRDs from `dist/chart/crds/` to `dist/chart/templates/crd/` so they are updated on `helm upgrade`.

## Step 3: Add GitHub Actions Workflows

Create two new workflow files:
- `.github/workflows/release.yml` — triggered on `v*` tags and main branch pushes
- `.github/workflows/helm-validate.yml` — triggered on PRs modifying `dist/chart/**`

## Step 4: Test Locally

```bash
# Validate the chart
helm lint dist/chart/
helm template jwks-rotater dist/chart/

# Test installation on a local cluster
kind create cluster
helm install jwks-rotater dist/chart/ --namespace jwks-rotater-system --create-namespace
```

## Step 5: Create a Release

```bash
git tag v0.1.0
git push origin v0.1.0
```

The release workflow will build the image, package the chart, and push both to GHCR.

## Consuming the Chart

```bash
echo $GHCR_TOKEN | helm registry login ghcr.io -u USERNAME --password-stdin
helm install jwks-rotater oci://ghcr.io/yanok/jwks-rotater/charts/jwks-rotater \
  --version 0.1.0 \
  --namespace jwks-rotater-system \
  --create-namespace
```
