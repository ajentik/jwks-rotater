# JWKS Rotation Operator

A Kubernetes operator that automates JSON Web Key Set (JWKS) lifecycle management — key generation, rotation, retention, and distribution — for services that sign or verify JWTs.

## Features

- **Automatic key rotation** on a configurable schedule (e.g., every 24h)
- **Retention-based cleanup** removes expired keys after a grace period
- **Dual Secret output** — private JWKS for signing, public JWKS for verification
- **Deployment restart** on rotation to pick up new keys
- **Policy-based auto-discovery** — match Deployments by label selector
- **RSA (2048/4096) and ECDSA (P-256/P-384)** key types
- **Prometheus metrics** for rotation counts, active keys, and reconcile duration
- **Finalizer-based cleanup** with optional secret retention on CR deletion

## Prerequisites

- Go 1.25+
- Docker 17.03+
- kubectl v1.28+
- Access to a Kubernetes v1.28+ cluster

## Quick Start

### Run locally (against current kubeconfig)

```sh
make install   # Install CRDs
make run       # Run the controller locally
```

In another terminal:

```sh
kubectl apply -f config/samples/jwks_v1alpha1_jwksrotation.yaml
kubectl get jwksrotation -o wide
kubectl get secret | grep jwks
```

### Deploy to a cluster

```sh
# Build and push the image
make docker-build docker-push IMG=<registry>/jwks-rotater:latest

# Deploy CRDs, RBAC, and the controller
make deploy IMG=<registry>/jwks-rotater:latest

# Apply sample CRs
kubectl apply -k config/samples/
```

### Uninstall

```sh
kubectl delete -k config/samples/   # Delete CRs
make undeploy                        # Remove controller and RBAC
make uninstall                       # Remove CRDs
```

## Custom Resources

### JWKSRotation (namespace-scoped)

Manages JWKS for a specific service:

```yaml
apiVersion: jwks.ajentik.ai/v1alpha1
kind: JWKSRotation
metadata:
  name: auth-service-jwks
spec:
  keyType: RSA
  keySize: 2048
  rotationInterval: 24h
  retentionPeriod: 72h
  targetSecret:
    name: auth-jwks
  targetDeployments:
    - name: auth-service
```

### JWKSRotationPolicy (cluster-scoped)

Auto-discovers Deployments by label and manages JWKS for each:

```yaml
apiVersion: jwks.ajentik.ai/v1alpha1
kind: JWKSRotationPolicy
metadata:
  name: default-rotation
spec:
  selector:
    matchLabels:
      jwks.ajentik.ai/rotate: "true"
  keyType: RSA
  keySize: 2048
  rotationInterval: 24h
  retentionPeriod: 72h
```

## Development

```sh
make generate    # Regenerate deepcopy methods
make manifests   # Regenerate CRD and RBAC manifests
make test        # Run unit and integration tests (envtest)
make lint        # Run golangci-lint
```

## Project Distribution

### Single YAML installer

```sh
make build-installer IMG=<registry>/jwks-rotater:latest
kubectl apply -f dist/install.yaml
```

## License

Copyright 2026. Licensed under the Apache License, Version 2.0.
