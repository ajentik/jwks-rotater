// Copyright (c) 2026 Volodymyr Ajentik
// SPDX-License-Identifier: MIT

package v1alpha1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KeyType specifies the cryptographic algorithm for key generation.
// +kubebuilder:validation:Enum=RSA;ECDSA
type KeyType string

const (
	RSA   KeyType = "RSA"
	ECDSA KeyType = "ECDSA"
)

// TargetSecretRef references the Secret to store the JWKS in.
type TargetSecretRef struct {
	// name is the name of the target Secret. The public Secret is automatically
	// named <name>-public.
	// +required
	Name string `json:"name"`
}

// TargetDeploymentRef references a Deployment to restart on rotation.
type TargetDeploymentRef struct {
	// name is the Deployment name in the same namespace as the JWKSRotation resource.
	// +required
	Name string `json:"name"`
}

// JWKSRotationSpec defines the desired state of JWKSRotation.
type JWKSRotationSpec struct {
	// keyType is the cryptographic key algorithm. Defaults to RSA.
	// +optional
	// +kubebuilder:default=RSA
	KeyType KeyType `json:"keyType,omitempty"`

	// keySize is the key size in bits. RSA: 2048 or 4096. ECDSA: 256 (P-256) or 384 (P-384).
	// Defaults to 2048 for RSA and 256 for ECDSA when omitted.
	// +optional
	KeySize int `json:"keySize,omitempty"`

	// rotationInterval specifies how often to generate a new key (e.g., "24h", "168h").
	// +required
	RotationInterval metav1.Duration `json:"rotationInterval"`

	// retentionPeriod specifies how long to keep old keys before cleanup.
	// Must be greater than rotationInterval.
	// +required
	RetentionPeriod metav1.Duration `json:"retentionPeriod"`

	// targetSecret references the Secret to store the JWKS in.
	// +required
	TargetSecret TargetSecretRef `json:"targetSecret"`

	// targetDeployments lists Deployments to restart on key rotation.
	// +optional
	TargetDeployments []TargetDeploymentRef `json:"targetDeployments,omitempty"`

	// retainSecretsOnDelete controls whether Secrets are deleted when this CR is removed.
	// If true, Secrets are left in place as orphans.
	// +optional
	// +kubebuilder:default=false
	RetainSecretsOnDelete bool `json:"retainSecretsOnDelete,omitempty"`
}

// JWKSRotationStatus defines the observed state of JWKSRotation.
type JWKSRotationStatus struct {
	// lastRotation is the time of the most recent key rotation.
	// +optional
	LastRotation *metav1.Time `json:"lastRotation,omitempty"`

	// nextRotation is the scheduled time for the next rotation.
	// +optional
	NextRotation *metav1.Time `json:"nextRotation,omitempty"`

	// activeKeys is the number of keys currently in the JWKS.
	// +optional
	ActiveKeys int `json:"activeKeys,omitempty"`

	// conditions represent the current state of the JWKSRotation resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// DefaultKeySize returns the default key size for the given key type.
func DefaultKeySize(kt KeyType) int {
	if kt == ECDSA {
		return 256
	}
	return 2048
}

// Validate checks that the spec fields are consistent.
// It applies default keySize when omitted (0).
func (s *JWKSRotationSpec) Validate() error {
	if s.TargetSecret.Name == "" {
		return fmt.Errorf("targetSecret.name must not be empty")
	}
	if s.KeySize == 0 {
		s.KeySize = DefaultKeySize(s.KeyType)
	}
	switch s.KeyType {
	case RSA:
		if s.KeySize != 2048 && s.KeySize != 4096 {
			return fmt.Errorf("invalid key size %d for RSA", s.KeySize)
		}
	case ECDSA:
		if s.KeySize != 256 && s.KeySize != 384 {
			return fmt.Errorf("invalid key size %d for ECDSA", s.KeySize)
		}
	default:
		return fmt.Errorf("unsupported key type %q", s.KeyType)
	}
	if s.RotationInterval.Duration <= 0 {
		return fmt.Errorf("rotationInterval must be positive")
	}
	if s.RetentionPeriod.Duration <= 0 {
		return fmt.Errorf("retentionPeriod must be positive")
	}
	if s.RetentionPeriod.Duration <= s.RotationInterval.Duration {
		return fmt.Errorf("retentionPeriod must be greater than rotationInterval")
	}
	return nil
}

// PublicSecretName returns the derived name for the public-only JWKS Secret.
func (s *JWKSRotationSpec) PublicSecretName() string {
	return s.TargetSecret.Name + "-public"
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Key Type",type=string,JSONPath=`.spec.keyType`
// +kubebuilder:printcolumn:name="Active Keys",type=integer,JSONPath=`.status.activeKeys`
// +kubebuilder:printcolumn:name="Last Rotation",type=date,JSONPath=`.status.lastRotation`
// +kubebuilder:printcolumn:name="Next Rotation",type=date,JSONPath=`.status.nextRotation`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// JWKSRotation is the Schema for the jwksrotations API.
type JWKSRotation struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec JWKSRotationSpec `json:"spec"`

	// +optional
	Status JWKSRotationStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// JWKSRotationList contains a list of JWKSRotation.
type JWKSRotationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []JWKSRotation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JWKSRotation{}, &JWKSRotationList{})
}
