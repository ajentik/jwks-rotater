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
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	jwksv1alpha1 "github.com/yanok/jwks-rotater/api/v1alpha1"
	"github.com/yanok/jwks-rotater/internal/jwks"
)

// JWKSRotationPolicyReconciler reconciles a JWKSRotationPolicy object.
type JWKSRotationPolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=jwks.ajentik.ai,resources=jwksrotationpolicies,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=jwks.ajentik.ai,resources=jwksrotationpolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=jwks.ajentik.ai,resources=jwksrotationpolicies/finalizers,verbs=update

func (r *JWKSRotationPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var policy jwksv1alpha1.JWKSRotationPolicy
	if err := r.Get(ctx, req.NamespacedName, &policy); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("fetching JWKSRotationPolicy: %w", err)
	}

	if err := policy.Spec.Validate(); err != nil {
		setPolicyCondition(&policy, "Error", metav1.ConditionTrue, "ValidationFailed", err.Error())
		if statusErr := r.Status().Update(ctx, &policy); statusErr != nil {
			return ctrl.Result{}, fmt.Errorf("updating error status: %w", statusErr)
		}
		return ctrl.Result{}, nil
	}

	selector, err := metav1.LabelSelectorAsSelector(&policy.Spec.Selector)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("parsing label selector: %w", err)
	}

	var deployments appsv1.DeploymentList
	if err := r.List(ctx, &deployments, &client.ListOptions{
		LabelSelector: selector,
	}); err != nil {
		return ctrl.Result{}, fmt.Errorf("listing deployments: %w", err)
	}

	// Build a set of secret names targeted by explicit JWKSRotation CRs
	explicitSecrets, err := r.buildExplicitSecretSet(ctx, deployments.Items)
	if err != nil {
		return ctrl.Result{}, err
	}

	managedSecrets := 0
	now := time.Now().UTC()

	for i := range deployments.Items {
		dep := &deployments.Items[i]
		secretName := dep.Name + "-jwks"

		// Skip if a dedicated JWKSRotation CR already targets this Deployment's Secret
		if explicitSecrets[types.NamespacedName{Name: secretName, Namespace: dep.Namespace}] {
			log.V(1).Info("skipping deployment with explicit JWKSRotation CR", "deployment", dep.Name)
			continue
		}

		ks, err := r.loadKeyStore(ctx, dep.Namespace, secretName)
		if err != nil {
			log.Error(err, "loading keystore for deployment", "deployment", dep.Name)
			continue
		}

		needsWrite := false

		if ks.Len() == 0 {
			k, err := jwks.GenerateKey(string(policy.Spec.KeyType), policy.Spec.KeySize)
			if err != nil {
				log.Error(err, "generating key for deployment", "deployment", dep.Name)
				continue
			}
			ks.AddKey(k)
			needsWrite = true
		}

		// Check rotation
		oldest := ks.OldestKeyAge(now)
		if oldest > 0 {
			newest := ks.NewestKey()
			if newest != nil && now.Sub(newest.CreatedAt) >= policy.Spec.RotationInterval.Duration {
				k, err := jwks.GenerateKey(string(policy.Spec.KeyType), policy.Spec.KeySize)
				if err != nil {
					log.Error(err, "rotating key for deployment", "deployment", dep.Name)
					continue
				}
				ks.AddKey(k)
				needsWrite = true
			}
		}

		// Retention cleanup
		removed := ks.RemoveExpired(policy.Spec.RetentionPeriod.Duration, now)
		if len(removed) > 0 {
			needsWrite = true
		}

		if needsWrite {
			if err := r.writeSecrets(ctx, &policy, dep.Namespace, secretName, ks); err != nil {
				log.Error(err, "writing secrets for deployment", "deployment", dep.Name)
				continue
			}
		}

		managedSecrets++
	}

	policy.Status.MatchedDeployments = len(deployments.Items)
	policy.Status.ManagedSecrets = managedSecrets

	setPolicyCondition(&policy, "Ready", metav1.ConditionTrue, "ReconcileSuccessful",
		fmt.Sprintf("Managing %d secrets for %d deployments", managedSecrets, len(deployments.Items)))

	if err := r.Status().Update(ctx, &policy); err != nil {
		return ctrl.Result{}, fmt.Errorf("updating policy status: %w", err)
	}

	return ctrl.Result{RequeueAfter: policy.Spec.RotationInterval.Duration}, nil
}

func (r *JWKSRotationPolicyReconciler) buildExplicitSecretSet(ctx context.Context, deployments []appsv1.Deployment) (map[types.NamespacedName]bool, error) {
	result := make(map[types.NamespacedName]bool)
	namespaces := make(map[string]bool)
	for _, dep := range deployments {
		namespaces[dep.Namespace] = true
	}
	for ns := range namespaces {
		var rotations jwksv1alpha1.JWKSRotationList
		if err := r.List(ctx, &rotations, &client.ListOptions{Namespace: ns}); err != nil {
			return nil, fmt.Errorf("listing JWKSRotations in namespace %s: %w", ns, err)
		}
		for _, rot := range rotations.Items {
			result[types.NamespacedName{Name: rot.Spec.TargetSecret.Name, Namespace: ns}] = true
		}
	}
	return result, nil
}

func (r *JWKSRotationPolicyReconciler) loadKeyStore(ctx context.Context, namespace, secretName string) (*jwks.KeyStore, error) {
	var existing corev1.Secret
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, &existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return jwks.NewKeyStore(), nil
		}
		return nil, fmt.Errorf("fetching secret: %w", err)
	}
	data, ok := existing.Data["jwks.json"]
	if !ok {
		return jwks.NewKeyStore(), nil
	}
	return jwks.ParseKeyStore(data)
}

func (r *JWKSRotationPolicyReconciler) writeSecrets(ctx context.Context, policy *jwksv1alpha1.JWKSRotationPolicy, namespace, secretName string, ks *jwks.KeyStore) error {
	owner := &metav1.OwnerReference{
		APIVersion:         policy.APIVersion,
		Kind:               policy.Kind,
		Name:               policy.Name,
		UID:                policy.UID,
		Controller:         boolPtr(true),
		BlockOwnerDeletion: boolPtr(true),
	}

	privSecret, pubSecret, err := jwks.BuildSecrets(ks, namespace, secretName, owner)
	if err != nil {
		return err
	}

	privSecret.Labels["jwks.ajentik.ai/policy"] = policy.Name
	pubSecret.Labels["jwks.ajentik.ai/policy"] = policy.Name

	for _, secret := range []*corev1.Secret{privSecret, pubSecret} {
		var existing corev1.Secret
		err := r.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, &existing)
		if apierrors.IsNotFound(err) {
			if err := r.Create(ctx, secret); err != nil {
				return fmt.Errorf("creating secret %s: %w", secret.Name, err)
			}
		} else if err != nil {
			return fmt.Errorf("fetching secret %s: %w", secret.Name, err)
		} else {
			existing.Data = secret.Data
			existing.Labels = secret.Labels
			existing.OwnerReferences = secret.OwnerReferences
			if err := r.Update(ctx, &existing); err != nil {
				return fmt.Errorf("updating secret %s: %w", secret.Name, err)
			}
		}
	}

	return nil
}

func setPolicyCondition(policy *jwksv1alpha1.JWKSRotationPolicy, condType string, status metav1.ConditionStatus, reason, message string) {
	now := metav1.Now()
	for i, c := range policy.Status.Conditions {
		if c.Type == condType {
			if c.Status != status {
				policy.Status.Conditions[i].LastTransitionTime = now
			}
			policy.Status.Conditions[i].Status = status
			policy.Status.Conditions[i].Reason = reason
			policy.Status.Conditions[i].Message = message
			policy.Status.Conditions[i].ObservedGeneration = policy.Generation
			return
		}
	}
	policy.Status.Conditions = append(policy.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: policy.Generation,
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *JWKSRotationPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&jwksv1alpha1.JWKSRotationPolicy{}).
		Watches(&appsv1.Deployment{}, handler.EnqueueRequestsFromMapFunc(r.mapDeploymentToPolicy)).
		Named("jwksrotationpolicy").
		Complete(r)
}

func (r *JWKSRotationPolicyReconciler) mapDeploymentToPolicy(ctx context.Context, obj client.Object) []ctrl.Request {
	var policies jwksv1alpha1.JWKSRotationPolicyList
	if err := r.List(ctx, &policies); err != nil {
		return nil
	}

	var requests []ctrl.Request
	for _, policy := range policies.Items {
		selector, err := metav1.LabelSelectorAsSelector(&policy.Spec.Selector)
		if err != nil {
			continue
		}
		if selector.Matches(labels.Set(obj.GetLabels())) {
			requests = append(requests, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: policy.Name},
			})
		}
	}
	return requests
}
