package controller

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	jwksv1alpha1 "github.com/ajentik/jwks-rotater/api/v1alpha1"
	"github.com/ajentik/jwks-rotater/internal/jwks"
)

// TestReconcile_NoDoubleRotation simulates the race condition where reconcile A
// rotates a key and writes the Secret, but its status update hasn't landed yet.
// Reconcile B is triggered by the Secret watch and sees a stale
// Status.LastRotation from a previous rotation cycle. The Secret already
// contains the freshly rotated key. The reconciler must use the keystore
// (Secret) as the source of truth and not rotate again.
func TestReconcile_NoDoubleRotation(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(jwksv1alpha1.AddToScheme(scheme))

	now := time.Now().UTC()

	// Build a keystore with two keys: one old (2h ago) and one fresh (just rotated
	// by reconcile A). This is the Secret state after A's write.
	ks := jwks.NewKeyStore()
	oldKey, err := jwks.GenerateKey("RSA", 2048)
	if err != nil {
		t.Fatal(err)
	}
	oldKey.CreatedAt = now.Add(-2 * time.Hour)
	ks.AddKey(oldKey)

	newKey, err := jwks.GenerateKey("RSA", 2048)
	if err != nil {
		t.Fatal(err)
	}
	newKey.CreatedAt = now
	ks.AddKey(newKey)

	jwksData, err := ks.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	// Status.LastRotation is stale — set to the OLD key's creation time.
	// This is what reconcile B sees: the status from the previous rotation
	// cycle, before reconcile A's status update landed.
	staleLastRotation := metav1.NewTime(oldKey.CreatedAt)

	cr := &jwksv1alpha1.JWKSRotation{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-jwks",
			Namespace:  "default",
			Finalizers: []string{finalizerName},
		},
		Spec: jwksv1alpha1.JWKSRotationSpec{
			KeyType:          jwksv1alpha1.RSA,
			KeySize:          2048,
			RotationInterval: metav1.Duration{Duration: 1 * time.Hour},
			RetentionPeriod:  metav1.Duration{Duration: 24 * time.Hour},
			TargetSecret:     jwksv1alpha1.TargetSecretRef{Name: "test-secret"},
		},
		Status: jwksv1alpha1.JWKSRotationStatus{
			LastRotation: &staleLastRotation,
			ActiveKeys:   1,
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{"jwks.json": jwksData},
	}

	pubSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret-public",
			Namespace: "default",
		},
		Data: map[string][]byte{"jwks.json": jwksData},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cr, secret, pubSecret).
		WithStatusSubresource(cr).
		Build()

	r := &JWKSRotationReconciler{
		Client: cl,
		Scheme: scheme,
	}

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-jwks", Namespace: "default"}}

	// Reconcile B: Status.LastRotation is 2h ago (stale), but the Secret
	// already has a key created just now. Must not rotate again.
	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	var s corev1.Secret
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "test-secret", Namespace: "default"}, &s); err != nil {
		t.Fatal(err)
	}
	result, err := jwks.ParseKeyStore(s.Data["jwks.json"])
	if err != nil {
		t.Fatal(err)
	}
	if result.Len() != 2 {
		t.Fatalf("got %d keys, want 2 (spurious rotation occurred)", result.Len())
	}
}
