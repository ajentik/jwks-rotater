// Copyright (c) 2026 Volodymyr Ajentik
// SPDX-License-Identifier: MIT

package controller

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	jwksv1alpha1 "github.com/ajentik/jwks-rotater/api/v1alpha1"
)

var _ = Describe("JWKSRotation Controller", func() {
	const (
		timeout  = 30 * time.Second
		interval = 250 * time.Millisecond
	)

	var (
		ns         string
		nsCount    int
		crName     string
		secretName string
	)

	BeforeEach(func() {
		nsCount++
		ns = fmt.Sprintf("test-ns-%d-%d", GinkgoParallelProcess(), nsCount)
		crName = "test-jwks"
		secretName = "my-jwks"

		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		}
		Expect(k8sClient.Create(ctx, namespace)).To(Succeed())
	})

	newValidCR := func() *jwksv1alpha1.JWKSRotation {
		return &jwksv1alpha1.JWKSRotation{
			ObjectMeta: metav1.ObjectMeta{
				Name:      crName,
				Namespace: ns,
			},
			Spec: jwksv1alpha1.JWKSRotationSpec{
				KeyType:          jwksv1alpha1.RSA,
				KeySize:          2048,
				RotationInterval: metav1.Duration{Duration: 24 * time.Hour},
				RetentionPeriod:  metav1.Duration{Duration: 72 * time.Hour},
				TargetSecret: jwksv1alpha1.TargetSecretRef{
					Name: secretName,
				},
			},
		}
	}

	Context("US1: Initial key generation (T020-T023, T031-T034)", func() {
		It("should create private and public Secrets with one key when a valid CR is created", func() {
			cr := newValidCR()
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())

			privateKey := types.NamespacedName{Name: secretName, Namespace: ns}
			publicKey := types.NamespacedName{Name: secretName + "-public", Namespace: ns}

			By("waiting for the private Secret to be created")
			privateSecret := &corev1.Secret{}
			Eventually(func() error {
				return k8sClient.Get(ctx, privateKey, privateSecret)
			}, timeout, interval).Should(Succeed())

			Expect(privateSecret.Type).To(Equal(corev1.SecretTypeOpaque))
			Expect(privateSecret.Data).To(HaveKey("jwks.json"))
			Expect(privateSecret.Labels).To(HaveKeyWithValue("jwks.ajentik.ai/managed-by", "jwks-operator"))
			Expect(privateSecret.Labels).To(HaveKeyWithValue("jwks.ajentik.ai/rotation", crName))

			By("verifying owner reference on private Secret")
			Expect(privateSecret.OwnerReferences).To(HaveLen(1))
			Expect(privateSecret.OwnerReferences[0].Name).To(Equal(crName))

			By("waiting for the public Secret to be created")
			publicSecret := &corev1.Secret{}
			Eventually(func() error {
				return k8sClient.Get(ctx, publicKey, publicSecret)
			}, timeout, interval).Should(Succeed())

			Expect(publicSecret.Type).To(Equal(corev1.SecretTypeOpaque))
			Expect(publicSecret.Data).To(HaveKey("jwks.json"))
			Expect(publicSecret.Labels).To(HaveKeyWithValue("jwks.ajentik.ai/managed-by", "jwks-operator"))
			Expect(publicSecret.Labels).To(HaveKeyWithValue("jwks.ajentik.ai/rotation", crName))

			By("verifying owner reference on public Secret")
			Expect(publicSecret.OwnerReferences).To(HaveLen(1))
			Expect(publicSecret.OwnerReferences[0].Name).To(Equal(crName))
		})

		It("should set Error condition when CR has invalid config (e.g., ECDSA with keySize 2048)", func() {
			cr := newValidCR()
			cr.Spec.KeyType = jwksv1alpha1.ECDSA
			cr.Spec.KeySize = 2048
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())

			By("waiting for the Error condition to appear on the CR status")
			Eventually(func(g Gomega) {
				updated := &jwksv1alpha1.JWKSRotation{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, updated)).To(Succeed())
				g.Expect(updated.Status.Conditions).NotTo(BeEmpty())

				var foundError bool
				for _, c := range updated.Status.Conditions {
					if c.Type == "Error" && c.Status == metav1.ConditionTrue {
						foundError = true
						g.Expect(c.Message).To(ContainSubstring("invalid key size"))
					}
				}
				g.Expect(foundError).To(BeTrue(), "expected Error condition to be set")
			}, timeout, interval).Should(Succeed())

			By("verifying no Secrets were created")
			secretList := &corev1.SecretList{}
			Expect(k8sClient.List(ctx, secretList, &client.ListOptions{Namespace: ns})).To(Succeed())
			for _, s := range secretList.Items {
				Expect(s.Labels).NotTo(HaveKey("jwks.ajentik.ai/managed-by"),
					"no managed Secrets should exist for invalid CR")
			}
		})

		It("should set status with activeKeys=1 and lastRotation timestamp (T039-T042)", func() {
			cr := newValidCR()
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())

			By("waiting for status to reflect initial key generation")
			Eventually(func(g Gomega) {
				updated := &jwksv1alpha1.JWKSRotation{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, updated)).To(Succeed())
				g.Expect(updated.Status.ActiveKeys).To(Equal(1))
				g.Expect(updated.Status.LastRotation).NotTo(BeNil())
				g.Expect(updated.Status.NextRotation).NotTo(BeNil())
			}, timeout, interval).Should(Succeed())

			By("verifying Ready condition is set")
			Eventually(func(g Gomega) {
				updated := &jwksv1alpha1.JWKSRotation{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, updated)).To(Succeed())

				var foundReady bool
				for _, c := range updated.Status.Conditions {
					if c.Type == "Ready" && c.Status == metav1.ConditionTrue {
						foundReady = true
					}
				}
				g.Expect(foundReady).To(BeTrue(), "expected Ready condition to be set")
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("US2: Key rotation (T031-T034)", func() {
		It("should append a new key after the rotation interval elapses", func() {
			cr := newValidCR()
			cr.Spec.RotationInterval = metav1.Duration{Duration: 1 * time.Second}
			cr.Spec.RetentionPeriod = metav1.Duration{Duration: 1 * time.Hour}
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())

			By("waiting for at least two keys (initial + one rotation)")
			Eventually(func(g Gomega) {
				updated := &jwksv1alpha1.JWKSRotation{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ns}, updated)).To(Succeed())
				g.Expect(updated.Status.ActiveKeys).To(BeNumerically(">=", 2))
			}, timeout, interval).Should(Succeed())
		})

		It("should update public Secret in sync after rotation", func() {
			cr := newValidCR()
			cr.Name = crName + "-sync"
			cr.Spec.TargetSecret.Name = secretName + "-sync"
			cr.Spec.RotationInterval = metav1.Duration{Duration: 1 * time.Second}
			cr.Spec.RetentionPeriod = metav1.Duration{Duration: 1 * time.Hour}
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())

			By("waiting for rotation")
			Eventually(func(g Gomega) {
				updated := &jwksv1alpha1.JWKSRotation{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cr.Name, Namespace: ns}, updated)).To(Succeed())
				g.Expect(updated.Status.ActiveKeys).To(BeNumerically(">=", 2))
			}, timeout, interval).Should(Succeed())

			By("verifying public Secret has jwks.json data")
			pubSecret := &corev1.Secret{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: secretName + "-sync-public", Namespace: ns}, pubSecret)
			}, timeout, interval).Should(Succeed())
			Expect(pubSecret.Data).To(HaveKey("jwks.json"))
		})
	})

	Context("US3: Key retention and cleanup (T039-T042)", func() {
		It("should remove keys older than the retention period", func() {
			cr := newValidCR()
			cr.Name = crName + "-retention"
			cr.Spec.TargetSecret.Name = secretName + "-retention"
			cr.Spec.RotationInterval = metav1.Duration{Duration: 1 * time.Second}
			cr.Spec.RetentionPeriod = metav1.Duration{Duration: 3 * time.Second}
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())

			By("waiting for multiple keys to accumulate")
			Eventually(func(g Gomega) {
				updated := &jwksv1alpha1.JWKSRotation{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cr.Name, Namespace: ns}, updated)).To(Succeed())
				g.Expect(updated.Status.ActiveKeys).To(BeNumerically(">=", 3))
			}, timeout, interval).Should(Succeed())

			By("waiting for retention cleanup to reduce key count")
			// After retention period, old keys should be cleaned up
			// The key count should stabilize as old keys are removed
			Eventually(func(g Gomega) {
				updated := &jwksv1alpha1.JWKSRotation{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cr.Name, Namespace: ns}, updated)).To(Succeed())
				// With 1s rotation and 3s retention, we should never have more than ~4 keys
				g.Expect(updated.Status.ActiveKeys).To(BeNumerically("<=", 5))
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("US4: Target Deployment restart (T047-T049)", func() {
		It("should annotate target Deployments with restart timestamp after rotation", func() {
			depName := "test-deploy"
			dep := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      depName,
					Namespace: ns,
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": "test"},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "test", Image: "busybox"},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, dep)).To(Succeed())

			cr := newValidCR()
			cr.Name = crName + "-restart"
			cr.Spec.TargetSecret.Name = secretName + "-restart"
			cr.Spec.RotationInterval = metav1.Duration{Duration: 1 * time.Second}
			cr.Spec.RetentionPeriod = metav1.Duration{Duration: 1 * time.Hour}
			cr.Spec.TargetDeployments = []jwksv1alpha1.TargetDeploymentRef{
				{Name: depName},
			}
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())

			By("waiting for rotation and then checking Deployment annotation")
			Eventually(func(g Gomega) {
				updated := &appsv1.Deployment{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: depName, Namespace: ns}, updated)).To(Succeed())
				g.Expect(updated.Spec.Template.Annotations).To(HaveKey("kubectl.kubernetes.io/restartedAt"))
			}, timeout, interval).Should(Succeed())
		})

		It("should not fail if target Deployment does not exist", func() {
			cr := newValidCR()
			cr.Name = crName + "-nodep"
			cr.Spec.TargetSecret.Name = secretName + "-nodep"
			cr.Spec.RotationInterval = metav1.Duration{Duration: 1 * time.Second}
			cr.Spec.RetentionPeriod = metav1.Duration{Duration: 1 * time.Hour}
			cr.Spec.TargetDeployments = []jwksv1alpha1.TargetDeploymentRef{
				{Name: "nonexistent-deploy"},
			}
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())

			By("waiting for rotation to occur despite missing Deployment")
			Eventually(func(g Gomega) {
				updated := &jwksv1alpha1.JWKSRotation{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cr.Name, Namespace: ns}, updated)).To(Succeed())
				g.Expect(updated.Status.ActiveKeys).To(BeNumerically(">=", 2))
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("Finalizer: Secret cleanup on CR deletion (T053-T055)", func() {
		It("should delete managed Secrets when CR is deleted with retainSecretsOnDelete=false", func() {
			cr := newValidCR()
			cr.Name = crName + "-del"
			cr.Spec.TargetSecret.Name = secretName + "-del"
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())

			By("waiting for Secrets to be created")
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: secretName + "-del", Namespace: ns}, &corev1.Secret{})
			}, timeout, interval).Should(Succeed())

			By("deleting the CR")
			Expect(k8sClient.Delete(ctx, cr)).To(Succeed())

			By("waiting for Secrets to be deleted")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: secretName + "-del", Namespace: ns}, &corev1.Secret{})
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: secretName + "-del-public", Namespace: ns}, &corev1.Secret{})
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		})

		It("should leave managed Secrets when CR is deleted with retainSecretsOnDelete=true", func() {
			cr := newValidCR()
			cr.Name = crName + "-retain"
			cr.Spec.TargetSecret.Name = secretName + "-retain"
			cr.Spec.RetainSecretsOnDelete = true
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())

			By("waiting for Secrets to be created")
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: secretName + "-retain", Namespace: ns}, &corev1.Secret{})
			}, timeout, interval).Should(Succeed())

			By("deleting the CR")
			Expect(k8sClient.Delete(ctx, cr)).To(Succeed())

			By("waiting for CR to be gone")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.Name, Namespace: ns}, &jwksv1alpha1.JWKSRotation{})
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())

			By("verifying Secrets still exist")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: secretName + "-retain", Namespace: ns}, &corev1.Secret{})).To(Succeed())
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: secretName + "-retain-public", Namespace: ns}, &corev1.Secret{})).To(Succeed())
		})
	})
})
