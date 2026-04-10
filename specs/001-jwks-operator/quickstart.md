# Quickstart: JWKS Rotation Operator

## Prerequisites

- Go 1.22+
- Docker (for building container images)
- kubectl with access to a Kubernetes 1.28+ cluster
- kubebuilder v4 (for scaffolding and code generation)

## Bootstrap the Project

```bash
# Initialize kubebuilder project
kubebuilder init --domain ajentik.ai --repo github.com/yanok/jwks-rotater

# Create the JWKSRotation API (namespace-scoped)
kubebuilder create api --group jwks --version v1alpha1 --kind JWKSRotation --resource --controller

# Create the JWKSRotationPolicy API (cluster-scoped)
kubebuilder create api --group jwks --version v1alpha1 --kind JWKSRotationPolicy --resource --controller
```

## Key Dependencies

```bash
go get github.com/go-jose/go-jose/v4
```

## Development Workflow

```bash
# Generate CRD manifests and deep copy methods
make generate
make manifests

# Run tests (unit + envtest)
make test

# Run the operator locally against the current kubeconfig cluster
make run

# Build and push the container image
make docker-build docker-push IMG=<registry>/jwks-rotater:latest

# Deploy to cluster
make deploy IMG=<registry>/jwks-rotater:latest
```

## Verify It Works

```bash
# Apply a sample JWKSRotation
kubectl apply -f config/samples/jwks_v1alpha1_jwksrotation.yaml

# Check status
kubectl get jwksrotation -o wide

# Verify Secrets were created
kubectl get secret | grep jwks

# Watch events
kubectl get events --field-selector involvedObject.kind=JWKSRotation
```

## Run Tests

```bash
# Unit tests (fast, no cluster needed)
go test ./internal/jwks/...

# Controller tests with envtest (simulated API server)
go test ./internal/controller/...

# E2E tests (requires a running cluster)
go test ./test/e2e/... -v
```
