# CRD Contract Schemas

## JWKSRotation CR — Example

```yaml
apiVersion: jwks.ajentik.ai/v1alpha1
kind: JWKSRotation
metadata:
  name: auth-service-jwks
  namespace: payments
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
    - name: token-validator
```

### Expected Status

```yaml
status:
  lastRotation: "2026-04-10T00:00:00Z"
  nextRotation: "2026-04-11T00:00:00Z"
  activeKeys: 3
  conditions:
    - type: Ready
      status: "True"
      lastTransitionTime: "2026-04-10T00:00:00Z"
      reason: RotationSuccessful
      message: "JWKS contains 3 active keys"
```

### Expected Secrets

Private Secret `auth-jwks`:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: auth-jwks
  namespace: payments
  labels:
    jwks.ajentik.ai/managed-by: jwks-operator
    jwks.ajentik.ai/rotation: auth-service-jwks
type: Opaque
data:
  jwks.json: <base64-encoded JWKS with private keys>
```

Public Secret `auth-jwks-public`:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: auth-jwks-public
  namespace: payments
  labels:
    jwks.ajentik.ai/managed-by: jwks-operator
    jwks.ajentik.ai/rotation: auth-service-jwks
type: Opaque
data:
  jwks.json: <base64-encoded JWKS with public keys only>
```

## JWKSRotationPolicy CR — Example

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

### Expected Behavior

A Deployment with the matching label:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-service
  namespace: default
  labels:
    jwks.ajentik.ai/rotate: "true"
```

Results in Secrets:
- `my-service-jwks` (private) in `default` namespace
- `my-service-jwks-public` (public) in `default` namespace

## RBAC Requirements

The operator ServiceAccount requires:

```yaml
rules:
  - apiGroups: ["jwks.ajentik.ai"]
    resources: ["jwksrotations", "jwksrotations/status", "jwksrotations/finalizers"]
    verbs: ["get", "list", "watch", "update", "patch"]
  - apiGroups: ["jwks.ajentik.ai"]
    resources: ["jwksrotationpolicies", "jwksrotationpolicies/status"]
    verbs: ["get", "list", "watch", "update", "patch"]
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["get", "list", "watch", "patch"]
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["create", "patch"]
  - apiGroups: ["coordination.k8s.io"]
    resources: ["leases"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```
