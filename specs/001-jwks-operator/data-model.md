# Data Model: JWKS Rotation Operator

## JWKSRotation (namespace-scoped CRD)

**API Group**: `jwks.ajentik.ai/v1alpha1`
**Kind**: `JWKSRotation`

### Spec Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `keyType` | enum: `RSA`, `ECDSA` | No | `RSA` | Cryptographic key algorithm |
| `keySize` | int | No | `2048` (RSA), `256` (ECDSA P-256) | Key size in bits. RSA: 2048/4096. ECDSA: 256 (P-256), 384 (P-384) |
| `rotationInterval` | duration | Yes | — | How often to generate a new key (e.g., `24h`, `168h`) |
| `retentionPeriod` | duration | Yes | — | How long to keep old keys before cleanup (e.g., `72h`) |
| `targetSecret` | object | Yes | — | Reference to the target Secret |
| `targetSecret.name` | string | Yes | — | Name of the private JWKS Secret (public Secret auto-derived as `<name>-public`) |
| `targetDeployments` | []object | No | `[]` | Deployments to restart on rotation |
| `targetDeployments[].name` | string | Yes | — | Deployment name (same namespace) |
| `retainSecretsOnDelete` | bool | No | `false` | If true, Secrets are not deleted when this CR is removed |

### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `lastRotation` | timestamp | Time of the most recent key rotation |
| `nextRotation` | timestamp | Scheduled time for the next rotation |
| `activeKeys` | int | Number of keys currently in the JWKS |
| `conditions` | []Condition | Standard Kubernetes conditions (Ready, Degraded, Error) |

### Validation Rules

- `retentionPeriod` MUST be greater than `rotationInterval`
- `keySize` must be valid for the chosen `keyType` (RSA: 2048 or 4096; ECDSA: 256 or 384)
- `targetSecret.name` must be a valid Kubernetes Secret name
- `targetDeployments[].name` must be a valid Kubernetes Deployment name

### State Transitions

```
CR Created → Initial key generated → Secret created → Status: Ready
                                                          ↓
                                               Rotation interval elapsed
                                                          ↓
                                               New key appended → Status: Ready (activeKeys++)
                                                          ↓
                                               Retention period exceeded for old key
                                                          ↓
                                               Old key removed → Status: Ready (activeKeys--)
                                                          ↓
                                               CR Deleted → Finalizer runs
                                                          ↓
                                    retainSecretsOnDelete?  ─── true ──→ Secrets kept, finalizer removed
                                              │
                                            false
                                              ↓
                                    Secrets deleted, finalizer removed
```

## JWKSRotationPolicy (cluster-scoped CRD)

**API Group**: `jwks.ajentik.ai/v1alpha1`
**Kind**: `JWKSRotationPolicy`

### Spec Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `selector` | LabelSelector | Yes | — | Matches Deployments across all namespaces |
| `keyType` | enum: `RSA`, `ECDSA` | No | `RSA` | Default key algorithm for matched Deployments |
| `keySize` | int | No | `2048` | Default key size |
| `rotationInterval` | duration | Yes | — | Default rotation interval |
| `retentionPeriod` | duration | Yes | — | Default retention period |

### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `matchedDeployments` | int | Number of Deployments currently matched |
| `managedSecrets` | int | Number of Secrets actively managed by this policy |
| `conditions` | []Condition | Standard Kubernetes conditions |

### Behavior

- Secret naming convention: `<deployment-name>-jwks` (private), `<deployment-name>-jwks-public` (public)
- If a Deployment already has a dedicated JWKSRotation CR targeting a Secret, the policy skips it
- When a Deployment's label is removed (no longer matches), the policy stops managing its Secret but does not delete it

## JWKS Secret Data Format

### Private Secret (`<name>`)

```json
{
  "keys": [
    {
      "kty": "RSA",
      "kid": "<RFC7638-thumbprint>",
      "n": "...",
      "e": "...",
      "d": "...",
      "p": "...",
      "q": "...",
      "dp": "...",
      "dq": "...",
      "qi": "...",
      "iat": 1744300800
    }
  ]
}
```

### Public Secret (`<name>-public`)

```json
{
  "keys": [
    {
      "kty": "RSA",
      "kid": "<RFC7638-thumbprint>",
      "n": "...",
      "e": "...",
      "iat": 1744300800
    }
  ]
}
```

Both stored under the data key `jwks.json` in the Secret.

## Kubernetes Events

| Reason | Type | Emitted When |
|--------|------|-------------|
| `KeyRotated` | Normal | New key successfully generated and added |
| `KeyExpired` | Normal | Old key removed after retention period |
| `RotationFailed` | Warning | Error during key generation or Secret update |
| `SecretRecreated` | Warning | Target Secret was missing and rebuilt |
| `InvalidConfig` | Warning | CR spec validation failed |

## Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `jwks_rotation_total` | Counter | `namespace`, `name` | Total rotations performed |
| `jwks_rotation_errors_total` | Counter | `namespace`, `name` | Total rotation failures |
| `jwks_active_keys` | Gauge | `namespace`, `name` | Current number of keys in the JWKS |
| `jwks_oldest_key_age_seconds` | Gauge | `namespace`, `name` | Age of the oldest key in seconds |
| `jwks_reconcile_duration_seconds` | Histogram | `controller` | Reconciliation loop duration |
