# JWKS Rotation Operator

[![Tests](https://github.com/ajentik/jwks-rotater/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/ajentik/jwks-rotater/actions/workflows/test.yml)
[![Lint](https://github.com/ajentik/jwks-rotater/actions/workflows/lint.yml/badge.svg?branch=main)](https://github.com/ajentik/jwks-rotater/actions/workflows/lint.yml)
[![Release](https://img.shields.io/github/v/release/ajentik/jwks-rotater?sort=semver)](https://github.com/ajentik/jwks-rotater/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/ajentik/jwks-rotater.svg)](https://pkg.go.dev/github.com/ajentik/jwks-rotater)
[![Go Report Card](https://goreportcard.com/badge/github.com/ajentik/jwks-rotater)](https://goreportcard.com/report/github.com/ajentik/jwks-rotater)

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

## Installation

**Cluster prerequisites:**

- Kubernetes v1.28+ cluster
- kubectl v1.28+ configured to communicate with your cluster
- Cluster-admin privileges (or equivalent)

Install the operator with a single command:

```sh
kubectl apply -f https://raw.githubusercontent.com/ajentik/jwks-rotater/main/dist/install.yaml
```

For detailed instructions including Kustomize-based installation, verification,
troubleshooting, and building from source, see the [Installation Guide](docs/install.md).

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

**Developer prerequisites:**

- Go 1.25+
- Docker 17.03+

```sh
make generate    # Regenerate deepcopy methods
make manifests   # Regenerate CRD and RBAC manifests
make test        # Run unit and integration tests (envtest)
make lint        # Run golangci-lint
```

## License

Licensed under the MIT License. See [LICENSE](LICENSE) for the full text.
