package jwks

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildSecrets creates the private and public Kubernetes Secrets from a KeyStore.
// If owner is nil, no OwnerReferences are set (e.g. when retainSecretsOnDelete is true).
func BuildSecrets(ks *KeyStore, namespace, secretName string, owner *metav1.OwnerReference) (*corev1.Secret, *corev1.Secret, error) {
	privData, err := ks.MarshalJSON()
	if err != nil {
		return nil, nil, fmt.Errorf("marshaling private JWKS: %w", err)
	}

	pubData, err := ks.MarshalPublicJSON()
	if err != nil {
		return nil, nil, fmt.Errorf("marshaling public JWKS: %w", err)
	}

	labels := map[string]string{
		"jwks.ajentik.ai/managed-by": "jwks-operator",
	}
	if owner != nil {
		labels["jwks.ajentik.ai/rotation"] = owner.Name
	}

	var ownerRefs []metav1.OwnerReference
	if owner != nil {
		ownerRefs = []metav1.OwnerReference{*owner}
	}

	privSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            secretName,
			Namespace:       namespace,
			Labels:          labels,
			OwnerReferences: ownerRefs,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"jwks.json": privData,
		},
	}

	pubSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            secretName + "-public",
			Namespace:       namespace,
			Labels:          labels,
			OwnerReferences: ownerRefs,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"jwks.json": pubData,
		},
	}

	return privSecret, pubSecret, nil
}
