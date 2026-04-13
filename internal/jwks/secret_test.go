package jwks

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestBuildSecrets(t *testing.T) {
	ks := NewKeyStore()
	k, err := GenerateKey("RSA", 2048)
	if err != nil {
		t.Fatal(err)
	}
	ks.AddKey(k)

	owner := metav1.OwnerReference{
		APIVersion: "jwks.ajentik.ai/v1alpha1",
		Kind:       "JWKSRotation",
		Name:       "test-cr",
		UID:        types.UID("test-uid"),
	}

	privSecret, pubSecret, err := BuildSecrets(ks, "my-ns", "auth-jwks", "test-cr", &owner)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("private secret", func(t *testing.T) {
		if privSecret.Name != "auth-jwks" {
			t.Errorf("Name = %q, want %q", privSecret.Name, "auth-jwks")
		}
		if privSecret.Namespace != "my-ns" {
			t.Errorf("Namespace = %q, want %q", privSecret.Namespace, "my-ns")
		}
		if privSecret.Type != corev1.SecretTypeOpaque {
			t.Errorf("Type = %v, want Opaque", privSecret.Type)
		}
		if _, ok := privSecret.Data["jwks.json"]; !ok {
			t.Error("missing jwks.json data key")
		}
		if privSecret.Labels["jwks.ajentik.ai/managed-by"] != "jwks-operator" {
			t.Error("missing managed-by label")
		}
		if len(privSecret.OwnerReferences) != 1 {
			t.Error("expected one owner reference")
		}
	})

	t.Run("public secret", func(t *testing.T) {
		if pubSecret.Name != "auth-jwks-public" {
			t.Errorf("Name = %q, want %q", pubSecret.Name, "auth-jwks-public")
		}
		if pubSecret.Namespace != "my-ns" {
			t.Errorf("Namespace = %q, want %q", pubSecret.Namespace, "my-ns")
		}
		if _, ok := pubSecret.Data["jwks.json"]; !ok {
			t.Error("missing jwks.json data key")
		}
	})

	t.Run("private contains more data than public", func(t *testing.T) {
		if len(privSecret.Data["jwks.json"]) <= len(pubSecret.Data["jwks.json"]) {
			t.Error("private JWKS should be larger than public JWKS (contains private key material)")
		}
	})
}

func TestBuildSecrets_Labels(t *testing.T) {
	ks := NewKeyStore()
	k, _ := GenerateKey("RSA", 2048)
	k.CreatedAt = time.Now().UTC()
	ks.AddKey(k)

	owner := metav1.OwnerReference{
		APIVersion: "jwks.ajentik.ai/v1alpha1",
		Kind:       "JWKSRotation",
		Name:       "my-rotation",
		UID:        types.UID("uid-123"),
	}

	priv, pub, err := BuildSecrets(ks, "default", "test-secret", "my-rotation", &owner)
	if err != nil {
		t.Fatal(err)
	}

	for _, s := range []*corev1.Secret{priv, pub} {
		if s.Labels["jwks.ajentik.ai/managed-by"] != "jwks-operator" {
			t.Errorf("secret %s: missing managed-by label", s.Name)
		}
		if s.Labels["jwks.ajentik.ai/rotation"] != "my-rotation" {
			t.Errorf("secret %s: missing rotation label", s.Name)
		}
	}
}
