/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// JWKSRotationPolicySpec defines the desired state of JWKSRotationPolicy.
type JWKSRotationPolicySpec struct {
	// selector matches Deployments by labels.
	// +required
	Selector metav1.LabelSelector `json:"selector"`

	// keyType is the default key algorithm for matched Deployments. Defaults to RSA.
	// +optional
	// +kubebuilder:default=RSA
	KeyType KeyType `json:"keyType,omitempty"`

	// keySize is the default key size in bits. Defaults to 2048.
	// +optional
	// +kubebuilder:default=2048
	KeySize int `json:"keySize,omitempty"`

	// rotationInterval is the default rotation interval for matched Deployments.
	// +required
	RotationInterval metav1.Duration `json:"rotationInterval"`

	// retentionPeriod is the default retention period for matched Deployments.
	// Must be greater than rotationInterval.
	// +required
	RetentionPeriod metav1.Duration `json:"retentionPeriod"`
}

// JWKSRotationPolicyStatus defines the observed state of JWKSRotationPolicy.
type JWKSRotationPolicyStatus struct {
	// matchedDeployments is the number of Deployments currently matched by the selector.
	// +optional
	MatchedDeployments int `json:"matchedDeployments,omitempty"`

	// managedSecrets is the number of Secrets actively managed by this policy.
	// +optional
	ManagedSecrets int `json:"managedSecrets,omitempty"`

	// conditions represent the current state of the JWKSRotationPolicy resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Matched",type=integer,JSONPath=`.status.matchedDeployments`
// +kubebuilder:printcolumn:name="Managed Secrets",type=integer,JSONPath=`.status.managedSecrets`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// JWKSRotationPolicy is the Schema for the jwksrotationpolicies API.
type JWKSRotationPolicy struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec JWKSRotationPolicySpec `json:"spec"`

	// +optional
	Status JWKSRotationPolicyStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// JWKSRotationPolicyList contains a list of JWKSRotationPolicy.
type JWKSRotationPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []JWKSRotationPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JWKSRotationPolicy{}, &JWKSRotationPolicyList{})
}
