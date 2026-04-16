# Contracts: Helm Chart & GHCR CI/CD Pipeline

**Date**: 2026-04-16  
**Feature**: 003-helm-ghcr-deploy

## Helm Chart Install Contract

Consumer-facing interface for installing the operator:

```bash
# Authenticate to GHCR (required for private packages)
echo $GHCR_TOKEN | helm registry login ghcr.io -u USERNAME --password-stdin

# Install
helm install jwks-rotater oci://ghcr.io/yanok/jwks-rotater/charts/jwks-rotater \
  --version <semver> \
  --namespace jwks-rotater-system \
  --create-namespace

# Upgrade
helm upgrade jwks-rotater oci://ghcr.io/yanok/jwks-rotater/charts/jwks-rotater \
  --version <new-semver> \
  --namespace jwks-rotater-system

# Uninstall
helm uninstall jwks-rotater --namespace jwks-rotater-system
```

## Release Trigger Contract

Maintainer-facing interface for creating releases:

```bash
# Tag and push to trigger release pipeline
git tag v1.0.0
git push origin v1.0.0
```

Expected outcome: Within 15 minutes, both artifacts are published to GHCR.

## Values Override Contract

Customization points exposed via `values.yaml`:

```yaml
manager:
  image:
    repository: ghcr.io/yanok/jwks-rotater
    tag: ""        # defaults to appVersion
    pullPolicy: IfNotPresent
  replicas: 1
  resources:
    limits:
      cpu: 500m
      memory: 128Mi
    requests:
      cpu: 10m
      memory: 64Mi
```
