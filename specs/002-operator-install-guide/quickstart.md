# Quickstart: Operator Installation Guide

## Prerequisites

- Access to the `jwks-rotater` repository
- `make`, `kustomize` (auto-downloaded by Makefile), and `kubectl`
- Understanding of the existing `config/` Kustomize layout

## Step 1: Update Makefile IMG Default

```bash
# In Makefile, change line 2:
# FROM: IMG ?= controller:latest
# TO:   IMG ?= ghcr.io/yanok/jwks-rotater:latest
```

## Step 2: Generate dist/install.yaml

```bash
# Generate the consolidated manifest with the GHCR image reference
make build-installer IMG=ghcr.io/yanok/jwks-rotater:latest

# Verify the output
ls -la dist/install.yaml
grep "image:" dist/install.yaml
```

## Step 3: Create docs/install.md

Create `docs/install.md` following the structure from research.md R-003:

1. Prerequisites
2. Quick Install (`kubectl apply -f` from raw GitHub URL)
3. Install with Kustomize (`git clone` + `make deploy`)
4. Verify Installation (kubectl get pods, CRDs, RBAC)
5. Try It Out (apply sample JWKSRotation CR)
6. Uninstall (delete CRs, undeploy, uninstall CRDs + warnings)
7. Building from Source (docker-build, docker-push, deploy)
8. Troubleshooting (5 scenarios from research.md R-006)

Key content to include:
- Namespace: `jwks-rotater-system`
- Image: `ghcr.io/yanok/jwks-rotater:latest`
- Sample CR from `config/samples/jwks_v1alpha1_jwksrotation.yaml`
- Verification commands: `kubectl get pods -n jwks-rotater-system`, `kubectl get crds | grep jwks`, `kubectl get clusterrole | grep jwks-rotater`

## Step 4: Update README.md

Replace the "Deploy to a cluster" subsection and "Project Distribution" section with a consolidated "Installation" section:

```markdown
## Installation

Install the operator with a single command:

    kubectl apply -f https://raw.githubusercontent.com/yanok/jwks-rotater/main/dist/install.yaml

For detailed instructions including Kustomize-based installation, verification,
troubleshooting, and building from source, see the [Installation Guide](docs/install.md).
```

## Step 5: Validate

```bash
# Verify dist/install.yaml is valid YAML
kubectl apply --dry-run=client -f dist/install.yaml

# Verify no placeholder image references remain
grep "controller:latest" dist/install.yaml  # Should return nothing

# Verify docs/install.md exists and has all required sections
head -50 docs/install.md
```
