// Copyright (c) 2026 Volodymyr Ajentik
// SPDX-License-Identifier: MIT

package controller

import (
	"fmt"
	"maps"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	jwksv1alpha1 "github.com/ajentik/jwks-rotater/api/v1alpha1"
)

var _ = Describe("JWKSRotationPolicy Controller", func() {
	const (
		timeout  = 30 * time.Second
		interval = 250 * time.Millisecond
	)

	var (
		ns       string
		nsPCount int
	)

	BeforeEach(func() {
		nsPCount++
		ns = fmt.Sprintf("test-policy-ns-%d-%d", GinkgoParallelProcess(), nsPCount)

		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		}
		Expect(k8sClient.Create(ctx, namespace)).To(Succeed())
	})

	createDeployment := func(name, namespace string, extraLabels map[string]string) *appsv1.Deployment {
		allLabels := map[string]string{"app": name}
		maps.Copy(allLabels, extraLabels)
		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels:    allLabels,
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": name},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": name},
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
		return dep
	}

	Context("US5: Auto-discovery via label selector (T058-T060)", func() {
		It("should create a JWKS Secret for a Deployment matching the label selector", func() {
			createDeployment("my-svc", ns, map[string]string{
				"jwks.ajentik.ai/rotate": "true",
			})

			policy := &jwksv1alpha1.JWKSRotationPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("auto-discover-%s", ns),
				},
				Spec: jwksv1alpha1.JWKSRotationPolicySpec{
					Selector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"jwks.ajentik.ai/rotate": "true",
						},
					},
					KeyType:          jwksv1alpha1.RSA,
					KeySize:          2048,
					RotationInterval: metav1.Duration{Duration: 24 * time.Hour},
					RetentionPeriod:  metav1.Duration{Duration: 72 * time.Hour},
				},
			}
			Expect(k8sClient.Create(ctx, policy)).To(Succeed())

			By("waiting for JWKS Secret to be created")
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: "my-svc-jwks", Namespace: ns}, &corev1.Secret{})
			}, timeout, interval).Should(Succeed())

			By("verifying public Secret also created")
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: "my-svc-jwks-public", Namespace: ns}, &corev1.Secret{})
			}, timeout, interval).Should(Succeed())

			By("verifying policy status")
			Eventually(func(g Gomega) {
				updated := &jwksv1alpha1.JWKSRotationPolicy{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: policy.Name}, updated)).To(Succeed())
				g.Expect(updated.Status.MatchedDeployments).To(BeNumerically(">=", 1))
				g.Expect(updated.Status.ManagedSecrets).To(BeNumerically(">=", 1))
			}, timeout, interval).Should(Succeed())
		})

		It("should skip a Deployment that has an explicit JWKSRotation CR", func() {
			// Use a unique label to avoid matching deployments from other tests
			labelValue := fmt.Sprintf("skip-%s", ns)
			createDeployment("explicit-svc", ns, map[string]string{
				"jwks.ajentik.ai/rotate": labelValue,
			})

			cr := &jwksv1alpha1.JWKSRotation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "explicit-rotation",
					Namespace: ns,
				},
				Spec: jwksv1alpha1.JWKSRotationSpec{
					KeyType:          jwksv1alpha1.RSA,
					KeySize:          2048,
					RotationInterval: metav1.Duration{Duration: 24 * time.Hour},
					RetentionPeriod:  metav1.Duration{Duration: 72 * time.Hour},
					TargetSecret:     jwksv1alpha1.TargetSecretRef{Name: "explicit-svc-jwks"},
				},
			}
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())

			By("waiting for the explicit CR to create its Secret")
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: "explicit-svc-jwks", Namespace: ns}, &corev1.Secret{})
			}, timeout, interval).Should(Succeed())

			policy := &jwksv1alpha1.JWKSRotationPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("skip-explicit-%s", ns),
				},
				Spec: jwksv1alpha1.JWKSRotationPolicySpec{
					Selector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"jwks.ajentik.ai/rotate": labelValue,
						},
					},
					KeyType:          jwksv1alpha1.RSA,
					KeySize:          2048,
					RotationInterval: metav1.Duration{Duration: 24 * time.Hour},
					RetentionPeriod:  metav1.Duration{Duration: 72 * time.Hour},
				},
			}
			Expect(k8sClient.Create(ctx, policy)).To(Succeed())

			By("verifying policy detects matched deployments but skips managed count")
			Eventually(func(g Gomega) {
				updated := &jwksv1alpha1.JWKSRotationPolicy{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: policy.Name}, updated)).To(Succeed())
				g.Expect(updated.Status.MatchedDeployments).To(Equal(1))
				g.Expect(updated.Status.ManagedSecrets).To(Equal(0))
			}, timeout, interval).Should(Succeed())
		})
	})
})
