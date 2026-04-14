# Installing the JWKS Rotation Operator

This guide covers installing, verifying, and managing the JWKS Rotation Operator on a Kubernetes cluster.

**Contents**

- [Prerequisites](#prerequisites)
- [Quick Install](#quick-install)
- [Install with Kustomize](#install-with-kustomize)
- [Verify Installation](#verify-installation)
- [Try It Out](#try-it-out)
- [Uninstall](#uninstall)
- [Building from Source](#building-from-source)
- [Troubleshooting](#troubleshooting)

## Prerequisites

Before installing the operator, ensure you have:

- **Kubernetes cluster** version 1.28 or later
- **kubectl** configured to communicate with your cluster
- **Cluster-admin privileges** (or equivalent) — the installer creates CRDs, ClusterRoles, and a dedicated namespace
- **Network access** to pull container images from `ghcr.io` (GitHub Container Registry)

To verify your cluster version:

```bash
kubectl version --short
```

## Quick Install

Install the operator with a single command:

```bash
kubectl apply -f https://raw.githubusercontent.com/yanok/jwks-rotater/main/dist/install.yaml
```

This creates:

- A `jwks-rotater-system` namespace
- Two CRDs: `JWKSRotation` (namespace-scoped) and `JWKSRotationPolicy` (cluster-scoped)
- A controller Deployment running the operator
- RBAC resources (ClusterRoles, RoleBindings, ServiceAccount) with least-privilege permissions
- A metrics Service on port 8443

The operator image `ghcr.io/yanok/jwks-rotater:latest` is pulled automatically.

## Install with Kustomize

For more control over the installation, clone the repository and use Kustomize:

```bash
git clone https://github.com/yanok/jwks-rotater.git
cd jwks-rotater
```

Review and optionally customize the manifests in `config/`:

- `config/default/kustomization.yaml` — main overlay (namespace, name prefix, patches)
- `config/manager/manager.yaml` — Deployment spec (replicas, resources, probes)
- `config/rbac/` — RBAC definitions

Deploy with:

```bash
make deploy IMG=ghcr.io/yanok/jwks-rotater:latest
```

To use a different image, replace the `IMG` value:

```bash
make deploy IMG=myregistry.example.com/jwks-rotater:v1.0.0
```

The operator is deployed to the `jwks-rotater-system` namespace.

## Verify Installation

After installing, confirm the operator is running correctly.

**Check the controller pod:**

```bash
kubectl get pods -n jwks-rotater-system
```

Expected output:

```
NAME                                                READY   STATUS    RESTARTS   AGE
jwks-rotater-controller-manager-xxxxxxxxxx-xxxxx    1/1     Running   0          30s
```

**Check CRDs are registered:**

```bash
kubectl get crds | grep jwks
```

Expected output:

```
jwksrotationpolicies.jwks.ajentik.ai   2026-04-14T00:00:00Z
jwksrotations.jwks.ajentik.ai          2026-04-14T00:00:00Z
```

**Check RBAC resources:**

```bash
kubectl get clusterrole | grep jwks-rotater
```

Expected output:

```
jwks-rotater-manager-role              2026-04-14T00:00:00Z
jwks-rotater-metrics-auth-role         2026-04-14T00:00:00Z
jwks-rotater-metrics-reader            2026-04-14T00:00:00Z
```

**Check the ServiceAccount:**

```bash
kubectl get serviceaccount -n jwks-rotater-system
```

Expected output includes `jwks-rotater-controller-manager`.

**Check controller logs:**

```bash
kubectl logs -n jwks-rotater-system deployment/jwks-rotater-controller-manager -f
```

You should see startup messages including `Starting Controller` and `Starting workers`.

## Try It Out

Apply a sample `JWKSRotation` resource to verify the operator is working:

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
  retainSecretsOnDelete: false
  targetSecret:
    name: auth-jwks
  targetDeployments:
    - name: auth-service
```

Save this as `sample-jwksrotation.yaml` and apply it:

```bash
kubectl apply -f sample-jwksrotation.yaml
```

Or apply the bundled sample directly:

```bash
kubectl apply -f https://raw.githubusercontent.com/yanok/jwks-rotater/main/config/samples/jwks_v1alpha1_jwksrotation.yaml
```

Check the resource status:

```bash
kubectl get jwksrotation -o wide
```

Expected output shows the resource with key type, active key count, and rotation timestamps.

Verify the operator created the JWKS Secrets:

```bash
kubectl get secrets | grep auth-jwks
```

You should see two Secrets: `auth-jwks` (private JWKS) and `auth-jwks-public` (public JWKS).

## Uninstall

> **Warning**: Removing CRDs deletes **all** JWKSRotation and JWKSRotationPolicy resources across all namespaces. Ensure you have backed up any data you need before proceeding.

**Step 1: Delete custom resources**

Remove all JWKSRotation and JWKSRotationPolicy resources first to allow the operator's finalizers to clean up owned Secrets:

```bash
kubectl delete jwksrotation --all --all-namespaces
kubectl delete jwksrotationpolicy --all
```

Wait for the resources to be fully deleted before proceeding.

**Step 2: Remove the operator**

If you installed with the quick install method:

```bash
kubectl delete -f https://raw.githubusercontent.com/yanok/jwks-rotater/main/dist/install.yaml
```

If you installed with Kustomize:

```bash
make undeploy
```

This removes the controller Deployment, RBAC resources, and the `jwks-rotater-system` namespace.

**Step 3: Remove CRDs (optional)**

If you want to remove the CRDs as well:

```bash
kubectl delete crd jwksrotations.jwks.ajentik.ai jwksrotationpolicies.jwks.ajentik.ai
```

> **Warning**: This permanently deletes all JWKSRotation and JWKSRotationPolicy resources that may still exist in the cluster.

## Building from Source

To build and deploy the operator from source code:

**Prerequisites for building:**

- Go 1.25 or later
- Docker (or Podman)
- Access to a container registry

**Build the image:**

```bash
git clone https://github.com/yanok/jwks-rotater.git
cd jwks-rotater
make docker-build IMG=<your-registry>/jwks-rotater:<tag>
```

**Push to your registry:**

```bash
make docker-push IMG=<your-registry>/jwks-rotater:<tag>
```

**Deploy using the custom image:**

```bash
make deploy IMG=<your-registry>/jwks-rotater:<tag>
```

For multi-platform builds (linux/arm64, linux/amd64):

```bash
make docker-buildx IMG=<your-registry>/jwks-rotater:<tag>
```

To generate a standalone install manifest with your custom image:

```bash
make build-installer IMG=<your-registry>/jwks-rotater:<tag>
kubectl apply -f dist/install.yaml
```

## Troubleshooting

### ImagePullBackOff

**Symptom**: The controller pod is stuck in `ImagePullBackOff` or `ErrImagePull`.

**Diagnose:**

```bash
kubectl describe pod -n jwks-rotater-system -l control-plane=controller-manager
```

Look for the `Events` section at the bottom for pull error details.

**Common causes and fixes:**

- **Wrong image reference**: Verify the image exists at the expected registry. If using a custom image, confirm you ran `make deploy IMG=<correct-image>`.
- **Private registry**: Create an image pull secret and patch the Deployment's ServiceAccount:
  ```bash
  kubectl create secret docker-registry regcred \
    --docker-server=<registry> \
    --docker-username=<user> \
    --docker-password=<token> \
    -n jwks-rotater-system
  kubectl patch serviceaccount jwks-rotater-controller-manager \
    -n jwks-rotater-system \
    -p '{"imagePullSecrets": [{"name": "regcred"}]}'
  ```
- **Network issues**: Ensure cluster nodes can reach `ghcr.io` (or your custom registry).

### CrashLoopBackOff

**Symptom**: The controller pod repeatedly restarts.

**Diagnose:**

```bash
kubectl logs -n jwks-rotater-system deployment/jwks-rotater-controller-manager --previous
```

**Common causes and fixes:**

- **Leader election failure**: If running multiple replicas without proper configuration, the controller may fail to acquire the leader lease. Check for `failed to acquire lease` in logs.
- **Insufficient RBAC**: The controller needs permissions to manage Secrets, Deployments, and its CRDs. Verify the ClusterRole was created:
  ```bash
  kubectl get clusterrole jwks-rotater-manager-role -o yaml
  ```
- **Malformed arguments**: Check the Deployment spec for incorrect command-line flags. The default flags are `--leader-elect` and `--health-probe-bind-address=:8081`.

### CRDs Not Registered

**Symptom**: `kubectl get jwksrotation` returns `error: the server doesn't have a resource type "jwksrotation"`.

**Diagnose:**

```bash
kubectl get crds | grep jwks
```

**Common causes and fixes:**

- **CRDs not applied**: If you used `make deploy`, CRDs are included automatically. If applying manifests manually, ensure CRDs are applied first:
  ```bash
  kubectl apply -f config/crd/bases/
  ```
- **API server version incompatibility**: The CRDs use `apiextensions.k8s.io/v1`, which requires Kubernetes 1.16+. Verify your cluster version with `kubectl version`.
- **CRD validation errors**: Check API server logs for CRD registration failures.

### Permission Denied Errors

**Symptom**: `kubectl apply` fails with `Error from server (Forbidden)`.

**Diagnose:**

```bash
kubectl auth can-i create clusterrole --all-namespaces
kubectl auth can-i create crd --all-namespaces
kubectl auth can-i create namespace
```

**Common causes and fixes:**

- **Insufficient privileges**: The installer creates ClusterRoles, CRDs, and a namespace. You need cluster-admin privileges or an equivalent role. Contact your cluster administrator for elevated access.
- **RBAC restrictions**: Some clusters restrict who can create cluster-scoped resources. Ask your admin to apply the install manifest on your behalf.

### Namespace Conflicts

**Symptom**: Installation fails because the `jwks-rotater-system` namespace already exists with conflicting resources.

**Diagnose:**

```bash
kubectl get all -n jwks-rotater-system
```

**Common causes and fixes:**

- **Previous installation**: If a previous installation exists, uninstall it first (see [Uninstall](#uninstall)) before re-installing.
- **Namespace exists but is empty**: This is harmless — `kubectl apply` will create resources in the existing namespace without issues.
- **Conflicting resource names**: If other resources in the namespace conflict with the operator's resources, delete the conflicting resources or use a different namespace by customizing `config/default/kustomization.yaml` before installing with Kustomize.
