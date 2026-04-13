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

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	jwksv1alpha1 "github.com/yanok/jwks-rotater/api/v1alpha1"
	"github.com/yanok/jwks-rotater/internal/jwks"
)

const (
	finalizerName = "jwks.ajentik.ai/secret-cleanup"
)

// JWKSRotationReconciler reconciles a JWKSRotation object.
type JWKSRotationReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder EventRecorder
}

// EventRecorder provides the ability to record events. Optional — nil-safe.
type EventRecorder interface {
	Eventf(regarding runtime.Object, related runtime.Object, eventtype, reason, action, note string, args ...any)
}

// +kubebuilder:rbac:groups=jwks.ajentik.ai,resources=jwksrotations,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=jwks.ajentik.ai,resources=jwksrotations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=jwks.ajentik.ai,resources=jwksrotations/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;patch

func (r *JWKSRotationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reconcileStart := time.Now()
	defer func() {
		reconcileDuration.WithLabelValues("jwksrotation").Observe(time.Since(reconcileStart).Seconds())
	}()
	log := logf.FromContext(ctx)

	var rotation jwksv1alpha1.JWKSRotation
	if err := r.Get(ctx, req.NamespacedName, &rotation); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("fetching JWKSRotation: %w", err)
	}

	// Handle deletion with finalizer
	if !rotation.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &rotation)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&rotation, finalizerName) {
		controllerutil.AddFinalizer(&rotation, finalizerName)
		if err := r.Update(ctx, &rotation); err != nil {
			return ctrl.Result{}, fmt.Errorf("adding finalizer: %w", err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Validate spec
	if err := rotation.Spec.Validate(); err != nil {
		return r.setErrorCondition(ctx, &rotation, "ValidationFailed", err.Error())
	}

	secretName := rotation.Spec.TargetSecret.Name
	now := time.Now().UTC()

	// Load or create keystore
	ks, err := r.loadOrCreateKeyStore(ctx, &rotation, secretName)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("loading keystore: %w", err)
	}

	needsWrite := false
	rotated := false

	// Initial key generation (no keys yet)
	if ks.Len() == 0 {
		if err := r.addNewKey(ks, &rotation); err != nil {
			return r.setErrorCondition(ctx, &rotation, "KeyGenerationFailed", fmt.Sprintf("generating initial key: %v", err))
		}
		needsWrite = true
		rotated = true
		log.Info("generated initial key", "secretName", secretName)
	}

	// Rotation check — derive effective last rotation from status or keystore
	var effectiveLastRotation time.Time
	if rotation.Status.LastRotation != nil {
		effectiveLastRotation = rotation.Status.LastRotation.Time
	} else if newest := ks.NewestKey(); newest != nil {
		effectiveLastRotation = newest.CreatedAt
	}

	if !effectiveLastRotation.IsZero() {
		nextRotation := effectiveLastRotation.Add(rotation.Spec.RotationInterval.Duration)
		if now.After(nextRotation) {
			if err := r.addNewKey(ks, &rotation); err != nil {
				return r.setErrorCondition(ctx, &rotation, "KeyRotationFailed", fmt.Sprintf("rotating key: %v", err))
			}
			needsWrite = true
			rotated = true
			log.Info("rotated key", "secretName", secretName)
			r.recordEvent(&rotation, corev1.EventTypeNormal, "KeyRotated", "New key generated and appended to JWKS")
			rotationTotal.WithLabelValues(rotation.Namespace, rotation.Name).Inc()

			// Restart target deployments (clear Degraded first; restartDeployments re-sets if needed)
			clearCondition(&rotation, "Degraded")
			r.restartDeployments(ctx, &rotation)
		}
	}

	// Retention cleanup
	removed := ks.RemoveExpired(rotation.Spec.RetentionPeriod.Duration, now)
	if len(removed) > 0 {
		needsWrite = true
		for _, kid := range removed {
			log.Info("removed expired key", "kid", kid)
			r.recordEvent(&rotation, corev1.EventTypeNormal, "KeyExpired", fmt.Sprintf("Removed expired key %s", kid))
		}
	}

	// Write secrets if anything changed
	if needsWrite {
		if err := r.writeSecrets(ctx, &rotation, ks, secretName); err != nil {
			rotationErrorsTotal.WithLabelValues(rotation.Namespace, rotation.Name).Inc()
			return ctrl.Result{}, fmt.Errorf("writing secrets: %w", err)
		}
	}

	// Update status — only update LastRotation when a key was actually generated
	if rotated {
		nowMeta := metav1.NewTime(now)
		rotation.Status.LastRotation = &nowMeta
	}

	// Compute next rotation from the effective last rotation time
	var requeueAfter time.Duration
	if rotation.Status.LastRotation != nil {
		next := rotation.Status.LastRotation.Add(rotation.Spec.RotationInterval.Duration)
		nextMeta := metav1.NewTime(next)
		rotation.Status.NextRotation = &nextMeta
		until := next.Sub(now)
		if until > 0 {
			requeueAfter = until
		} else {
			requeueAfter = rotation.Spec.RotationInterval.Duration
		}
	} else {
		requeueAfter = rotation.Spec.RotationInterval.Duration
	}

	rotation.Status.ActiveKeys = ks.Len()

	setCondition(&rotation, "Ready", metav1.ConditionTrue, "ReconcileSuccessful",
		fmt.Sprintf("JWKS contains %d active keys", ks.Len()))
	clearCondition(&rotation, "Error")

	// Update metrics
	activeKeys.WithLabelValues(rotation.Namespace, rotation.Name).Set(float64(ks.Len()))
	oldestKeyAge.WithLabelValues(rotation.Namespace, rotation.Name).Set(ks.OldestKeyAge(now).Seconds())

	if err := r.Status().Update(ctx, &rotation); err != nil {
		return ctrl.Result{}, fmt.Errorf("updating status: %w", err)
	}

	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

func (r *JWKSRotationReconciler) addNewKey(ks *jwks.KeyStore, rotation *jwksv1alpha1.JWKSRotation) error {
	k, err := jwks.GenerateKey(string(rotation.Spec.KeyType), rotation.Spec.KeySize)
	if err != nil {
		return err
	}
	ks.AddKey(k)
	return nil
}

func (r *JWKSRotationReconciler) loadOrCreateKeyStore(ctx context.Context, rotation *jwksv1alpha1.JWKSRotation, secretName string) (*jwks.KeyStore, error) {
	var existing corev1.Secret
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: rotation.Namespace}, &existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return jwks.NewKeyStore(), nil
		}
		return nil, fmt.Errorf("fetching existing secret: %w", err)
	}

	data, ok := existing.Data["jwks.json"]
	if !ok {
		return jwks.NewKeyStore(), nil
	}

	ks, err := jwks.ParseKeyStore(data)
	if err != nil {
		return nil, fmt.Errorf("parsing existing JWKS: %w", err)
	}
	return ks, nil
}

func (r *JWKSRotationReconciler) writeSecrets(ctx context.Context, rotation *jwksv1alpha1.JWKSRotation, ks *jwks.KeyStore, secretName string) error {
	var owner *metav1.OwnerReference
	if !rotation.Spec.RetainSecretsOnDelete {
		owner = &metav1.OwnerReference{
			APIVersion:         rotation.APIVersion,
			Kind:               rotation.Kind,
			Name:               rotation.Name,
			UID:                rotation.UID,
			Controller:         boolPtr(true),
			BlockOwnerDeletion: boolPtr(true),
		}
	}

	privSecret, pubSecret, err := jwks.BuildSecrets(ks, rotation.Namespace, secretName, rotation.Name, owner)
	if err != nil {
		return fmt.Errorf("building secrets: %w", err)
	}

	for _, secret := range []*corev1.Secret{privSecret, pubSecret} {
		var existing corev1.Secret
		err := r.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, &existing)
		if apierrors.IsNotFound(err) {
			if ks.Len() > 0 {
				r.recordEvent(rotation, corev1.EventTypeWarning, "SecretRecreated",
					fmt.Sprintf("Secret %s was missing and has been recreated", secret.Name))
			}
			if err := r.Create(ctx, secret); err != nil {
				return fmt.Errorf("creating secret %s: %w", secret.Name, err)
			}
		} else if err != nil {
			return fmt.Errorf("fetching secret %s: %w", secret.Name, err)
		} else {
			existing.Data = secret.Data
			if existing.Labels == nil {
				existing.Labels = make(map[string]string)
			}
			maps.Copy(existing.Labels, secret.Labels)
			existing.OwnerReferences = secret.OwnerReferences
			if err := r.Update(ctx, &existing); err != nil {
				return fmt.Errorf("updating secret %s: %w", secret.Name, err)
			}
		}
	}

	return nil
}

func (r *JWKSRotationReconciler) handleDeletion(ctx context.Context, rotation *jwksv1alpha1.JWKSRotation) (ctrl.Result, error) { //nolint:unparam
	if !controllerutil.ContainsFinalizer(rotation, finalizerName) {
		return ctrl.Result{}, nil
	}

	if !rotation.Spec.RetainSecretsOnDelete {
		secretName := rotation.Spec.TargetSecret.Name
		for _, name := range []string{secretName, secretName + "-public"} {
			var secret corev1.Secret
			if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: rotation.Namespace}, &secret); err != nil {
				if !apierrors.IsNotFound(err) {
					return ctrl.Result{}, fmt.Errorf("fetching secret %s for deletion: %w", name, err)
				}
			} else {
				if err := r.Delete(ctx, &secret); err != nil && !apierrors.IsNotFound(err) {
					return ctrl.Result{}, fmt.Errorf("deleting secret %s: %w", name, err)
				}
			}
		}
	}

	controllerutil.RemoveFinalizer(rotation, finalizerName)
	if err := r.Update(ctx, rotation); err != nil {
		return ctrl.Result{}, fmt.Errorf("removing finalizer: %w", err)
	}
	return ctrl.Result{}, nil
}

func (r *JWKSRotationReconciler) restartDeployments(ctx context.Context, rotation *jwksv1alpha1.JWKSRotation) {
	log := logf.FromContext(ctx)
	for _, ref := range rotation.Spec.TargetDeployments {
		var dep appsv1.Deployment
		key := types.NamespacedName{Name: ref.Name, Namespace: rotation.Namespace}
		if err := r.Get(ctx, key, &dep); err != nil {
			if apierrors.IsNotFound(err) {
				log.Info("target Deployment not found, skipping restart", "name", ref.Name)
				setCondition(rotation, "Degraded", metav1.ConditionTrue, "DeploymentNotFound",
					fmt.Sprintf("Target Deployment %s not found", ref.Name))
				continue
			}
			log.Error(err, "failed to get target Deployment", "name", ref.Name)
			continue
		}
		patch := map[string]any{
			"spec": map[string]any{
				"template": map[string]any{
					"metadata": map[string]any{
						"annotations": map[string]string{
							"kubectl.kubernetes.io/restartedAt": time.Now().UTC().Format(time.RFC3339),
						},
					},
				},
			},
		}
		patchBytes, _ := json.Marshal(patch)
		if err := r.Patch(ctx, &dep, client.RawPatch(types.MergePatchType, patchBytes)); err != nil {
			log.Error(err, "failed to restart Deployment", "name", ref.Name)
			continue
		}
		log.Info("restarted deployment", "name", ref.Name)
	}
}

func (r *JWKSRotationReconciler) setErrorCondition(ctx context.Context, rotation *jwksv1alpha1.JWKSRotation, reason, message string) (ctrl.Result, error) { //nolint:unparam
	rotationErrorsTotal.WithLabelValues(rotation.Namespace, rotation.Name).Inc()
	setCondition(rotation, "Error", metav1.ConditionTrue, reason, message)
	clearCondition(rotation, "Ready")
	r.recordEvent(rotation, corev1.EventTypeWarning, reason, message)
	if err := r.Status().Update(ctx, rotation); err != nil {
		return ctrl.Result{}, fmt.Errorf("updating error status: %w", err)
	}
	return ctrl.Result{}, nil
}

func (r *JWKSRotationReconciler) recordEvent(obj runtime.Object, eventtype, reason, message string) {
	if r.Recorder != nil {
		r.Recorder.Eventf(obj, nil, eventtype, reason, reason, message)
	}
}

func setCondition(rotation *jwksv1alpha1.JWKSRotation, condType string, status metav1.ConditionStatus, reason, message string) {
	now := metav1.Now()
	for i, c := range rotation.Status.Conditions {
		if c.Type == condType {
			if c.Status != status {
				rotation.Status.Conditions[i].LastTransitionTime = now
			}
			rotation.Status.Conditions[i].Status = status
			rotation.Status.Conditions[i].Reason = reason
			rotation.Status.Conditions[i].Message = message
			rotation.Status.Conditions[i].ObservedGeneration = rotation.Generation
			return
		}
	}
	rotation.Status.Conditions = append(rotation.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: rotation.Generation,
	})
}

func clearCondition(rotation *jwksv1alpha1.JWKSRotation, condType string) {
	for i, c := range rotation.Status.Conditions {
		if c.Type == condType {
			rotation.Status.Conditions = append(rotation.Status.Conditions[:i], rotation.Status.Conditions[i+1:]...)
			return
		}
	}
}

func boolPtr(b bool) *bool { return &b }

// SetupWithManager sets up the controller with the Manager.
func (r *JWKSRotationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&jwksv1alpha1.JWKSRotation{}).
		Owns(&corev1.Secret{}).
		Named("jwksrotation").
		Complete(r)
}
