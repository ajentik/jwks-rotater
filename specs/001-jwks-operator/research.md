# Research: JWKS Rotation Operator

## R-001: JWKS Key Generation Library

**Decision**: Use `go-jose/v4` (github.com/go-jose/go-jose/v4)
**Rationale**: go-jose is the maintained successor to square/go-jose, provides native JWKS serialization/deserialization, supports RSA and ECDSA key generation, and handles `kid` assignment. It's the de facto standard Go library for JOSE/JWKS.
**Alternatives considered**:
- `crypto` stdlib only: Can generate keys but requires manual JWKS JSON marshaling and `kid` generation. More code, more risk of non-standard output.
- `lestrrat-go/jwx`: Full-featured but heavier API surface. go-jose has broader adoption in the Kubernetes ecosystem.

## R-002: Key ID (`kid`) Generation Strategy

**Decision**: Use a hash-based `kid` derived from the public key thumbprint (RFC 7638)
**Rationale**: Deterministic, collision-resistant, and standard-compliant. Allows consumers to verify `kid` integrity. go-jose provides `Thumbprint()` on JWK objects.
**Alternatives considered**:
- UUID-based `kid`: Simpler but provides no relationship to key material. Cannot detect duplicate keys.
- Timestamp-based `kid`: Human-readable but not standard, risk of collision if two keys generated in same second.

## R-003: Creation Timestamp Storage in JWKS

**Decision**: Store creation timestamp as a custom claim `iat` (issued-at, Unix epoch seconds) in each JWK's additional fields
**Rationale**: `iat` is a well-known JWT claim name that maps naturally to key creation time. go-jose supports extra fields on JWK objects. The operator reads this field to determine key age for retention calculations.
**Alternatives considered**:
- Kubernetes Secret annotations with per-key timestamps: Fragile — annotations can be edited independently of the JWKS data, leading to drift.
- Separate ConfigMap tracking key metadata: Adds another resource to manage and keep in sync.

## R-004: Reconciliation Scheduling Strategy

**Decision**: Use controller-runtime's `RequeueAfter` to schedule next reconciliation at the rotation interval
**Rationale**: Native to the controller-runtime reconcile loop. No external cron needed. If the operator restarts, it reconciles immediately and detects overdue rotations by comparing `status.lastRotation` + `spec.rotationInterval` against current time.
**Alternatives considered**:
- CronJob sidecar: Adds deployment complexity, doesn't benefit from controller-runtime's leader election and watch mechanics.
- Time-based ticker in a separate goroutine: Fights against the controller-runtime model, harder to test.

## R-005: Finalizer Strategy for Secret Cleanup

**Decision**: Add a finalizer `jwks.ajentik.ai/secret-cleanup` to JWKSRotation resources
**Rationale**: Standard Kubernetes pattern for cleanup on deletion. The finalizer handler checks `retainSecretsOnDelete`; if false, deletes both private and public Secrets before removing the finalizer.
**Alternatives considered**:
- Owner references only: Would auto-delete Secrets when CR is deleted, but doesn't support the `retainSecretsOnDelete` opt-out.
- No cleanup: Leads to orphaned Secrets.

## R-006: Metrics Exposition

**Decision**: Use controller-runtime's built-in metrics server (Prometheus format on `:8080/metrics`)
**Rationale**: controller-runtime exposes a metrics endpoint by default. Custom metrics are registered via the `prometheus` client library already vendored by controller-runtime. Standard pattern for all kubebuilder operators.
**Alternatives considered**:
- OpenTelemetry: More flexible but adds a dependency and doesn't align with the kubebuilder default setup.

## R-007: Public Key Derivation

**Decision**: After each mutation of the private JWKS, strip private key material and write the public-only JWKS to `<secret-name>-public`
**Rationale**: go-jose JWK objects expose `.Public()` which returns the public-only representation. This is a simple map operation over the key set.
**Alternatives considered**:
- Serve public keys via an HTTP endpoint in the operator: Adds network exposure, TLS management, and availability concerns. Secrets are more aligned with the K8s-native consumption model specified in requirements.

## R-008: JWKSRotationPolicy Auto-Discovery

**Decision**: Watch Deployments with a label index; reconcile JWKSRotationPolicy by listing Deployments matching the selector. Create synthetic JWKSRotation-like behavior (Secret per Deployment) without creating actual JWKSRotation CRs.
**Rationale**: Creating actual CRs would clutter the namespace and confuse ownership. The policy controller manages Secrets directly using the policy's spec as the configuration source. If an explicit JWKSRotation CR exists for a Deployment's Secret, the policy controller skips it.
**Alternatives considered**:
- Auto-generate JWKSRotation CRs: Creates ownership confusion — who owns the CR? Hard to garbage-collect when labels change.
- Admission webhook that injects JWKSRotation CRs: Heavy-weight, requires webhook infrastructure.
