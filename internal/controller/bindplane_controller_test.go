/*
Copyright 2025.

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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

var _ = Describe("validateBindplaneName", func() {
	It("rejects empty name", func() {
		Expect(validateBindplaneName("")).NotTo(Succeed())
	})
	It("rejects name starting with a number (DNS-1035)", func() {
		Expect(validateBindplaneName("7539")).NotTo(Succeed())
		Expect(validateBindplaneName("123-abc")).NotTo(Succeed())
	})
	It("rejects name starting with uppercase", func() {
		Expect(validateBindplaneName("MyBindplane")).NotTo(Succeed())
	})
	It("rejects name ending with hyphen", func() {
		Expect(validateBindplaneName("my-name-")).NotTo(Succeed())
	})
	It("rejects name with invalid characters", func() {
		Expect(validateBindplaneName("my_name")).NotTo(Succeed())
		Expect(validateBindplaneName("my.name")).NotTo(Succeed())
	})
	It("rejects name that would exceed DNS label length", func() {
		long := "a" + strings.Repeat("x", maxResourceNamePrefixLen)
		Expect(long).To(HaveLen(maxResourceNamePrefixLen + 1))
		Expect(validateBindplaneName(long)).NotTo(Succeed())
	})
	It("accepts valid DNS-1035 names", func() {
		Expect(validateBindplaneName("a")).To(Succeed())
		Expect(validateBindplaneName("my-name")).To(Succeed())
		Expect(validateBindplaneName("abc-123")).To(Succeed())
		Expect(validateBindplaneName("bindplane")).To(Succeed())
	})
})

var _ = Describe("validateLicenseConfig", func() {
	It("rejects when neither license nor licenseSecretRef is set", func() {
		cfg := &bindplanev1alpha1.BindplaneConfigSpec{}
		Expect(validateLicenseConfig(cfg)).NotTo(Succeed())
	})

	It("rejects when both license and licenseSecretRef are set", func() {
		cfg := &bindplanev1alpha1.BindplaneConfigSpec{
			License: "test-license",
			LicenseSecretRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "bindplane-license"},
				Key:                  "license",
			},
		}
		Expect(validateLicenseConfig(cfg)).NotTo(Succeed())
	})

	It("accepts a direct license only", func() {
		cfg := &bindplanev1alpha1.BindplaneConfigSpec{
			License: "test-license",
		}
		Expect(validateLicenseConfig(cfg)).To(Succeed())
	})

	It("accepts a license secret ref only", func() {
		cfg := &bindplanev1alpha1.BindplaneConfigSpec{
			LicenseSecretRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "bindplane-license"},
				Key:                  "license",
			},
		}
		Expect(validateLicenseConfig(cfg)).To(Succeed())
	})
})

var _ = Describe("validateProfilingConfig", func() {
	It("accepts when profiling is nil or disabled", func() {
		Expect(validateProfilingConfig(nil)).To(Succeed())
		Expect(validateProfilingConfig(&bindplanev1alpha1.BindplaneConfigSpec{})).To(Succeed())
		Expect(validateProfilingConfig(&bindplanev1alpha1.BindplaneConfigSpec{
			Profiling: &bindplanev1alpha1.ProfilingConfig{Enabled: false},
		})).To(Succeed())
	})

	It("rejects when profiling is enabled but projectID is empty", func() {
		cfg := &bindplanev1alpha1.BindplaneConfigSpec{
			Profiling: &bindplanev1alpha1.ProfilingConfig{Enabled: true},
		}
		Expect(validateProfilingConfig(cfg)).NotTo(Succeed())
	})

	It("accepts when profiling is enabled and projectID is set", func() {
		cfg := &bindplanev1alpha1.BindplaneConfigSpec{
			Profiling: &bindplanev1alpha1.ProfilingConfig{
				Enabled:   true,
				ProjectID: "my-project",
			},
		}
		Expect(validateProfilingConfig(cfg)).To(Succeed())
	})
})

var _ = Describe("validatePprofConfig", func() {
	It("accepts when pprof is nil, disabled, or endpoint unset", func() {
		Expect(validatePprofConfig(nil)).To(Succeed())
		Expect(validatePprofConfig(&bindplanev1alpha1.BindplaneConfigSpec{})).To(Succeed())
		Expect(validatePprofConfig(&bindplanev1alpha1.BindplaneConfigSpec{
			Pprof: &bindplanev1alpha1.PprofConfig{Enabled: true},
		})).To(Succeed())
	})

	It("rejects when pprof is enabled and endpoint is invalid", func() {
		cfg := &bindplanev1alpha1.BindplaneConfigSpec{
			Pprof: &bindplanev1alpha1.PprofConfig{
				Enabled:  true,
				Endpoint: "not-host-port",
			},
		}
		Expect(validatePprofConfig(cfg)).NotTo(Succeed())
	})

	It("accepts when pprof is enabled and endpoint is valid host:port", func() {
		cfg := &bindplanev1alpha1.BindplaneConfigSpec{
			Pprof: &bindplanev1alpha1.PprofConfig{
				Enabled:  true,
				Endpoint: "127.0.0.1:6060",
			},
		}
		Expect(validatePprofConfig(cfg)).To(Succeed())
	})
})

var _ = Describe("Bindplane Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		bindplane := &bindplanev1alpha1.Bindplane{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Bindplane")
			err := k8sClient.Get(ctx, typeNamespacedName, bindplane)
			if err != nil && errors.IsNotFound(err) {
				resource := &bindplanev1alpha1.Bindplane{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: bindplanev1alpha1.BindplaneSpec{
						Config: bindplanev1alpha1.BindplaneConfigSpec{
							License: "test-license",
							Store: bindplanev1alpha1.StoreConfig{
								Postgres: &bindplanev1alpha1.PostgresConfig{
									Host: "postgres-host",
								},
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &bindplanev1alpha1.Bindplane{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Bindplane")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &BindplaneReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})

var _ = Describe("newService", func() {
	var bindplane *bindplanev1alpha1.Bindplane

	BeforeEach(func() {
		bindplane = &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-bindplane",
				Namespace: "default",
			},
		}
	})

	Context("when creating a service with a single port", func() {
		It("should create a ClusterIP service with the correct port configuration", func() {
			service := newService(bindplane, "test-component", WithPort("http", 8080))

			Expect(service).NotTo(BeNil())
			Expect(service.Name).To(Equal("test-bindplane-test-component"))
			Expect(service.Namespace).To(Equal("default"))
			Expect(service.Spec.Type).To(Equal(corev1.ServiceTypeClusterIP))
			Expect(service.Spec.Ports).To(HaveLen(1))
			Expect(service.Spec.Ports[0].Name).To(Equal("http"))
			Expect(service.Spec.Ports[0].Port).To(Equal(int32(8080)))
			Expect(service.Spec.Ports[0].TargetPort).To(Equal(intstr.FromInt(8080)))
			Expect(service.Spec.Ports[0].Protocol).To(Equal(corev1.ProtocolTCP))
		})

		It("should have correct labels", func() {
			service := newService(bindplane, "test-component", WithPort("http", 8080))

			Expect(service.Labels).To(HaveKeyWithValue(labelKeyName, labelValueName))
			Expect(service.Labels).To(HaveKeyWithValue(labelKeyInstance, "test-bindplane"))
			Expect(service.Labels).To(HaveKeyWithValue(labelKeyComponent, "test-component"))
			Expect(service.Labels).To(HaveKeyWithValue(labelKeyManagedBy, labelValueManagedBy))
			Expect(service.Labels).To(HaveKeyWithValue(labelKeyPartOf, labelValuePartOf))
		})

		It("should have correct selector labels", func() {
			service := newService(bindplane, "test-component", WithPort("http", 8080))

			Expect(service.Spec.Selector).To(HaveKeyWithValue(labelKeyName, labelValueName))
			Expect(service.Spec.Selector).To(HaveKeyWithValue(labelKeyInstance, "test-bindplane"))
			Expect(service.Spec.Selector).To(HaveKeyWithValue(labelKeyComponent, "test-component"))
			Expect(service.Spec.Selector).NotTo(HaveKey(labelKeyManagedBy))
			Expect(service.Spec.Selector).NotTo(HaveKey(labelKeyPartOf))
		})
	})

	Context("when creating a service with multiple ports", func() {
		It("should create a service with all specified ports", func() {
			service := newService(bindplane, "test-component",
				WithPort("http", 8080),
				WithPort("metrics", 9090),
				WithPort("grpc", 50051),
			)

			Expect(service).NotTo(BeNil())
			Expect(service.Spec.Ports).To(HaveLen(3))

			// Verify first port
			Expect(service.Spec.Ports[0].Name).To(Equal("http"))
			Expect(service.Spec.Ports[0].Port).To(Equal(int32(8080)))
			Expect(service.Spec.Ports[0].TargetPort).To(Equal(intstr.FromInt(8080)))

			// Verify second port
			Expect(service.Spec.Ports[1].Name).To(Equal("metrics"))
			Expect(service.Spec.Ports[1].Port).To(Equal(int32(9090)))
			Expect(service.Spec.Ports[1].TargetPort).To(Equal(intstr.FromInt(9090)))

			// Verify third port
			Expect(service.Spec.Ports[2].Name).To(Equal("grpc"))
			Expect(service.Spec.Ports[2].Port).To(Equal(int32(50051)))
			Expect(service.Spec.Ports[2].TargetPort).To(Equal(intstr.FromInt(50051)))

			// All ports should use TCP protocol
			for _, port := range service.Spec.Ports {
				Expect(port.Protocol).To(Equal(corev1.ProtocolTCP))
			}
		})

		It("should maintain port order", func() {
			service := newService(bindplane, "test-component",
				WithPort("first", 1000),
				WithPort("second", 2000),
				WithPort("third", 3000),
			)

			Expect(service.Spec.Ports).To(HaveLen(3))
			Expect(service.Spec.Ports[0].Name).To(Equal("first"))
			Expect(service.Spec.Ports[1].Name).To(Equal("second"))
			Expect(service.Spec.Ports[2].Name).To(Equal("third"))
		})
	})

	Context("when creating a service with no ports", func() {
		It("should create a service with empty ports slice", func() {
			service := newService(bindplane, "test-component")

			Expect(service).NotTo(BeNil())
			Expect(service.Spec.Ports).To(BeEmpty())
			Expect(service.Spec.Type).To(Equal(corev1.ServiceTypeClusterIP))
		})

		It("should still have correct labels and selectors", func() {
			service := newService(bindplane, "test-component")

			Expect(service.Labels).To(HaveKeyWithValue(labelKeyComponent, "test-component"))
			Expect(service.Spec.Selector).To(HaveKeyWithValue(labelKeyComponent, "test-component"))
		})
	})

	Context("when using different component names", func() {
		It("should generate correct service names", func() {
			service1 := newService(bindplane, "component-a", WithPort("http", 8080))
			service2 := newService(bindplane, "component-b", WithPort("http", 8080))

			Expect(service1.Name).To(Equal("test-bindplane-component-a"))
			Expect(service2.Name).To(Equal("test-bindplane-component-b"))
		})

		It("should use component name in labels and selectors", func() {
			service := newService(bindplane, "my-component", WithPort("http", 8080))

			Expect(service.Labels[labelKeyComponent]).To(Equal("my-component"))
			Expect(service.Spec.Selector[labelKeyComponent]).To(Equal("my-component"))
		})
	})

	Context("when using WithPort option", func() {
		It("should set both Port and TargetPort to the same value", func() {
			service := newService(bindplane, "test-component", WithPort("http", 8080))

			port := service.Spec.Ports[0]
			Expect(port.Port).To(Equal(int32(8080)))
			Expect(port.TargetPort).To(Equal(intstr.FromInt(8080)))
		})

		It("should handle different port values", func() {
			service := newService(bindplane, "test-component",
				WithPort("http", 80),
				WithPort("https", 443),
				WithPort("custom", 12345),
			)

			Expect(service.Spec.Ports[0].Port).To(Equal(int32(80)))
			Expect(service.Spec.Ports[1].Port).To(Equal(int32(443)))
			Expect(service.Spec.Ports[2].Port).To(Equal(int32(12345)))
		})
	})
})

var _ = Describe("newContainerSecurityContext", func() {
	Context("when creating a security context with default options", func() {
		It("should create a security context with all default security settings", func() {
			sc := newContainerSecurityContext()

			Expect(sc).NotTo(BeNil())
			Expect(sc.AllowPrivilegeEscalation).NotTo(BeNil())
			Expect(*sc.AllowPrivilegeEscalation).To(BeFalse())
			Expect(sc.Capabilities).NotTo(BeNil())
			Expect(sc.Capabilities.Drop).To(ContainElement(corev1.Capability("ALL")))
			Expect(sc.ReadOnlyRootFilesystem).NotTo(BeNil())
			Expect(*sc.ReadOnlyRootFilesystem).To(BeTrue())
			Expect(sc.RunAsNonRoot).NotTo(BeNil())
			Expect(*sc.RunAsNonRoot).To(BeTrue())
			Expect(sc.RunAsUser).NotTo(BeNil())
			Expect(*sc.RunAsUser).To(Equal(int64(65534))) // Default nobody user
		})

		It("should have correct default RunAsUser", func() {
			sc := newContainerSecurityContext()

			Expect(sc.RunAsUser).NotTo(BeNil())
			Expect(*sc.RunAsUser).To(Equal(int64(65534)))
		})
	})

	Context("when using WithRunAsUser option", func() {
		It("should override the default RunAsUser", func() {
			sc := newContainerSecurityContext(WithRunAsUser(1000))

			Expect(sc.RunAsUser).NotTo(BeNil())
			Expect(*sc.RunAsUser).To(Equal(int64(1000)))
		})

		It("should maintain all other security settings when using WithRunAsUser", func() {
			sc := newContainerSecurityContext(WithRunAsUser(2000))

			// Verify RunAsUser is overridden
			Expect(*sc.RunAsUser).To(Equal(int64(2000)))

			// Verify all other settings remain the same
			Expect(*sc.AllowPrivilegeEscalation).To(BeFalse())
			Expect(sc.Capabilities.Drop).To(ContainElement(corev1.Capability("ALL")))
			Expect(*sc.ReadOnlyRootFilesystem).To(BeTrue())
			Expect(*sc.RunAsNonRoot).To(BeTrue())
		})

		It("should handle different RunAsUser values", func() {
			testCases := []struct {
				userID int64
			}{
				{0},     // root
				{1000},  // regular user
				{65534}, // nobody (default)
				{9999},  // custom user
			}

			for _, tc := range testCases {
				sc := newContainerSecurityContext(WithRunAsUser(tc.userID))
				Expect(sc.RunAsUser).NotTo(BeNil())
				Expect(*sc.RunAsUser).To(Equal(tc.userID))
			}
		})
	})

	Context("when verifying security best practices", func() {
		It("should always disable privilege escalation", func() {
			sc := newContainerSecurityContext()
			Expect(*sc.AllowPrivilegeEscalation).To(BeFalse())

			sc2 := newContainerSecurityContext(WithRunAsUser(1000))
			Expect(*sc2.AllowPrivilegeEscalation).To(BeFalse())
		})

		It("should always drop all capabilities", func() {
			sc := newContainerSecurityContext()
			Expect(sc.Capabilities).NotTo(BeNil())
			Expect(sc.Capabilities.Drop).To(ContainElement(corev1.Capability("ALL")))
			Expect(sc.Capabilities.Add).To(BeEmpty())

			sc2 := newContainerSecurityContext(WithRunAsUser(1000))
			Expect(sc2.Capabilities).NotTo(BeNil())
			Expect(sc2.Capabilities.Drop).To(ContainElement(corev1.Capability("ALL")))
		})

		It("should always use read-only root filesystem", func() {
			sc := newContainerSecurityContext()
			Expect(*sc.ReadOnlyRootFilesystem).To(BeTrue())

			sc2 := newContainerSecurityContext(WithRunAsUser(1000))
			Expect(*sc2.ReadOnlyRootFilesystem).To(BeTrue())
		})

		It("should always run as non-root", func() {
			sc := newContainerSecurityContext()
			Expect(*sc.RunAsNonRoot).To(BeTrue())

			sc2 := newContainerSecurityContext(WithRunAsUser(1000))
			Expect(*sc2.RunAsNonRoot).To(BeTrue())
		})
	})

	Context("when comparing default vs custom RunAsUser", func() {
		It("should produce different RunAsUser values", func() {
			defaultSC := newContainerSecurityContext()
			customSC := newContainerSecurityContext(WithRunAsUser(1000))

			Expect(*defaultSC.RunAsUser).To(Equal(int64(65534)))
			Expect(*customSC.RunAsUser).To(Equal(int64(1000)))
			Expect(*defaultSC.RunAsUser).NotTo(Equal(*customSC.RunAsUser))
		})

		It("should produce identical security settings except RunAsUser", func() {
			defaultSC := newContainerSecurityContext()
			customSC := newContainerSecurityContext(WithRunAsUser(1000))

			// RunAsUser should be different
			Expect(*defaultSC.RunAsUser).NotTo(Equal(*customSC.RunAsUser))

			// All other settings should be identical
			Expect(*defaultSC.AllowPrivilegeEscalation).To(Equal(*customSC.AllowPrivilegeEscalation))
			Expect(defaultSC.Capabilities.Drop).To(Equal(customSC.Capabilities.Drop))
			Expect(*defaultSC.ReadOnlyRootFilesystem).To(Equal(*customSC.ReadOnlyRootFilesystem))
			Expect(*defaultSC.RunAsNonRoot).To(Equal(*customSC.RunAsNonRoot))
		})
	})

	Context("when using WithRunAsUser with the default value", func() {
		It("should work correctly when explicitly setting the default value", func() {
			sc := newContainerSecurityContext(WithRunAsUser(65534))

			Expect(sc.RunAsUser).NotTo(BeNil())
			Expect(*sc.RunAsUser).To(Equal(int64(65534)))
		})
	})
})

var _ = Describe("mergePodTemplateSpec", func() {
	var operatorManaged corev1.PodTemplateSpec

	BeforeEach(func() {
		operatorManaged = corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app.kubernetes.io/name":      "bindplane",
					"app.kubernetes.io/instance":  "test-instance",
					"app.kubernetes.io/component": "test",
				},
				Annotations: map[string]string{
					"operator-managed": "true",
				},
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: "operator-managed-sa",
				Containers: []corev1.Container{
					{
						Name:  "operator-container",
						Image: "operator-image:latest",
					},
				},
				TerminationGracePeriodSeconds: new(int64(60)),
				SecurityContext: &corev1.PodSecurityContext{
					FSGroup:    new(int64(65534)),
					RunAsGroup: new(int64(65534)),
					RunAsUser:  new(int64(65534)),
				},
				Volumes: []corev1.Volume{
					{
						Name: "operator-volume",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
			},
		}
	})

	Context("when user-provided template is nil", func() {
		It("should return operator-managed template unchanged", func() {
			result := mergePodTemplateSpec(operatorManaged, nil)
			Expect(result).To(Equal(operatorManaged))
		})
	})

	Context("when merging metadata", func() {
		It("should merge labels from user-provided template", func() {
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"user-label": "user-value",
						},
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.ObjectMeta.Labels).To(HaveKeyWithValue("app.kubernetes.io/name", "bindplane"))
			Expect(result.ObjectMeta.Labels).To(HaveKeyWithValue("app.kubernetes.io/instance", "test-instance"))
			Expect(result.ObjectMeta.Labels).To(HaveKeyWithValue("app.kubernetes.io/component", "test"))
			Expect(result.ObjectMeta.Labels).To(HaveKeyWithValue("user-label", "user-value"))
		})

		It("should protect operator-managed selector labels from user overrides", func() {
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app.kubernetes.io/name":      "user-override-name",
							"app.kubernetes.io/instance":  "user-override-instance",
							"app.kubernetes.io/component": "user-override-component",
							"user-label":                  "user-value",
						},
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			// Protected labels should retain operator-managed values
			Expect(result.ObjectMeta.Labels).To(HaveKeyWithValue("app.kubernetes.io/name", "bindplane"))
			Expect(result.ObjectMeta.Labels).To(HaveKeyWithValue("app.kubernetes.io/instance", "test-instance"))
			Expect(result.ObjectMeta.Labels).To(HaveKeyWithValue("app.kubernetes.io/component", "test"))
			// User labels should still be merged
			Expect(result.ObjectMeta.Labels).To(HaveKeyWithValue("user-label", "user-value"))
		})

		It("should merge annotations from user-provided template", func() {
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"user-annotation": "user-value",
						},
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.ObjectMeta.Annotations).To(HaveKeyWithValue("operator-managed", "true"))
			Expect(result.ObjectMeta.Annotations).To(HaveKeyWithValue("user-annotation", "user-value"))
		})

		It("should handle nil labels and annotations in operator-managed template", func() {
			operatorManaged.Labels = nil
			operatorManaged.Annotations = nil

			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"user-label": "user-value",
						},
						Annotations: map[string]string{
							"user-annotation": "user-value",
						},
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.ObjectMeta.Labels).To(HaveKeyWithValue("user-label", "user-value"))
			Expect(result.ObjectMeta.Annotations).To(HaveKeyWithValue("user-annotation", "user-value"))
		})
	})

	Context("when merging affinity", func() {
		It("should use user-provided affinity", func() {
			userAffinity := &corev1.Affinity{
				PodAntiAffinity: &corev1.PodAntiAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
						{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"app": "test",
								},
							},
							TopologyKey: "kubernetes.io/hostname",
						},
					},
				},
			}

			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Affinity: userAffinity,
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.Affinity).To(Equal(userAffinity))
		})

		It("should preserve nil affinity when user doesn't provide it", func() {
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.Affinity).To(BeNil())
		})
	})

	Context("when merging tolerations", func() {
		It("should use user-provided tolerations", func() {
			userTolerations := []corev1.Toleration{
				{
					Key:      "key1",
					Operator: corev1.TolerationOpEqual,
					Value:    "value1",
					Effect:   corev1.TaintEffectNoSchedule,
				},
			}

			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Tolerations: userTolerations,
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.Tolerations).To(Equal(userTolerations))
		})
	})

	Context("when merging nodeSelector", func() {
		It("should use user-provided nodeSelector", func() {
			userNodeSelector := map[string]string{
				"disktype": "ssd",
				"zone":     "us-west-1",
			}

			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						NodeSelector: userNodeSelector,
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.NodeSelector).To(Equal(userNodeSelector))
		})
	})

	Context("when merging priorityClassName", func() {
		It("should use user-provided priorityClassName", func() {
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						PriorityClassName: "high-priority",
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.PriorityClassName).To(Equal("high-priority"))
		})

		It("should not override when priorityClassName is empty", func() {
			operatorManaged.Spec.PriorityClassName = "operator-priority"

			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						PriorityClassName: "",
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.PriorityClassName).To(Equal("operator-priority"))
		})
	})

	Context("when merging runtimeClassName", func() {
		It("should use user-provided runtimeClassName", func() {
			runtimeClass := "gvisor"
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						RuntimeClassName: &runtimeClass,
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.RuntimeClassName).To(Equal(&runtimeClass))
		})
	})

	Context("when merging hostNetwork, hostPID, hostIPC", func() {
		It("should use user-provided hostNetwork", func() {
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						HostNetwork: true,
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.HostNetwork).To(BeTrue())
		})

		It("should not override when hostNetwork is false", func() {
			operatorManaged.Spec.HostNetwork = true

			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						HostNetwork: false,
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.HostNetwork).To(BeTrue())
		})
	})

	Context("when merging volumes", func() {
		It("should merge user volumes with operator volumes", func() {
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: "user-volume",
								VolumeSource: corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "user-config",
										},
									},
								},
							},
						},
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.Volumes).To(HaveLen(2))
			volumeNames := make(map[string]bool)
			for _, vol := range result.Spec.Volumes {
				volumeNames[vol.Name] = true
			}
			Expect(volumeNames).To(HaveKey("operator-volume"))
			Expect(volumeNames).To(HaveKey("user-volume"))
		})

		It("should allow user to override operator volume with same name", func() {
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: "operator-volume",
								VolumeSource: corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "user-override",
										},
									},
								},
							},
						},
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.Volumes).To(HaveLen(1))
			Expect(result.Spec.Volumes[0].Name).To(Equal("operator-volume"))
			Expect(result.Spec.Volumes[0].ConfigMap).ToNot(BeNil())
			Expect(result.Spec.Volumes[0].ConfigMap.Name).To(Equal("user-override"))
		})
	})

	Context("when merging initContainers", func() {
		It("should use user-provided initContainers", func() {
			userInitContainers := []corev1.Container{
				{
					Name:    "user-init",
					Image:   "busybox:latest",
					Command: []string{"sh", "-c", "echo init"},
				},
			}

			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						InitContainers: userInitContainers,
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.InitContainers).To(Equal(userInitContainers))
		})
	})

	Context("when merging securityContext", func() {
		It("should merge user securityContext fields with operator securityContext", func() {
			userFSGroup := new(int64(1000))
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						SecurityContext: &corev1.PodSecurityContext{
							FSGroup:   userFSGroup,
							RunAsUser: new(int64(1000)),
						},
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.SecurityContext).ToNot(BeNil())
			Expect(result.Spec.SecurityContext.FSGroup).To(Equal(userFSGroup))
			Expect(result.Spec.SecurityContext.RunAsUser).To(Equal(new(int64(1000))))
			// Operator-managed fields should be preserved if not overridden
			Expect(result.Spec.SecurityContext.RunAsGroup).To(Equal(new(int64(65534))))
		})

		It("should handle nil securityContext in operator-managed template", func() {
			operatorManaged.Spec.SecurityContext = nil

			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						SecurityContext: &corev1.PodSecurityContext{
							FSGroup: new(int64(1000)),
						},
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.SecurityContext).ToNot(BeNil())
			Expect(result.Spec.SecurityContext.FSGroup).To(Equal(new(int64(1000))))
		})
	})

	Context("when preserving operator-managed fields", func() {
		It("should preserve ServiceAccountName", func() {
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						ServiceAccountName: "user-sa",
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.ServiceAccountName).To(Equal("operator-managed-sa"))
		})

		It("should preserve Containers", func() {
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "user-container",
								Image: "user-image:latest",
							},
						},
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.Containers).To(HaveLen(1))
			Expect(result.Spec.Containers[0].Name).To(Equal("operator-container"))
			Expect(result.Spec.Containers[0].Image).To(Equal("operator-image:latest"))
		})

		It("should preserve TerminationGracePeriodSeconds", func() {
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						TerminationGracePeriodSeconds: new(int64(30)),
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.TerminationGracePeriodSeconds).To(Equal(new(int64(60))))
		})
	})

	Context("when merging DNS settings", func() {
		It("should use user-provided DNSPolicy", func() {
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						DNSPolicy: corev1.DNSClusterFirst,
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.DNSPolicy).To(Equal(corev1.DNSClusterFirst))
		})

		It("should use user-provided DNSConfig", func() {
			userDNSConfig := &corev1.PodDNSConfig{
				Nameservers: []string{"8.8.8.8"},
				Searches:    []string{"example.com"},
			}

			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						DNSConfig: userDNSConfig,
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.DNSConfig).To(Equal(userDNSConfig))
		})
	})

	Context("when merging imagePullSecrets", func() {
		It("should use user-provided imagePullSecrets", func() {
			userImagePullSecrets := []corev1.LocalObjectReference{
				{Name: "user-secret"},
			}

			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						ImagePullSecrets: userImagePullSecrets,
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.ImagePullSecrets).To(Equal(userImagePullSecrets))
		})
	})

	Context("complex merge scenarios", func() {
		It("should correctly merge multiple fields at once", func() {
			userAffinity := &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      "disktype",
										Operator: corev1.NodeSelectorOpIn,
										Values:   []string{"ssd"},
									},
								},
							},
						},
					},
				},
			}

			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"user-label": "value",
						},
					},
					Spec: corev1.PodSpec{
						Affinity: userAffinity,
						Tolerations: []corev1.Toleration{
							{
								Key:      "key1",
								Operator: corev1.TolerationOpEqual,
								Value:    "value1",
								Effect:   corev1.TaintEffectNoSchedule,
							},
						},
						NodeSelector: map[string]string{
							"zone": "us-west-1",
						},
						PriorityClassName: "high-priority",
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			// Verify all user fields are merged
			Expect(result.ObjectMeta.Labels).To(HaveKeyWithValue("user-label", "value"))
			Expect(result.Spec.Affinity).To(Equal(userAffinity))
			Expect(result.Spec.Tolerations).To(HaveLen(1))
			Expect(result.Spec.NodeSelector).To(HaveKeyWithValue("zone", "us-west-1"))
			Expect(result.Spec.PriorityClassName).To(Equal("high-priority"))

			// Verify operator-managed fields are preserved
			Expect(result.Spec.ServiceAccountName).To(Equal("operator-managed-sa"))
			Expect(result.Spec.Containers).To(HaveLen(1))
			Expect(result.Spec.Containers[0].Name).To(Equal("operator-container"))
			Expect(result.Spec.TerminationGracePeriodSeconds).To(Equal(new(int64(60))))
		})
	})
})

// envVarByName returns the value of the env var with the given name from the slice, or empty string if not found.
func envVarByName(envVars []corev1.EnvVar, name string) string {
	for _, ev := range envVars {
		if ev.Name == name {
			if ev.ValueFrom != nil && ev.ValueFrom.SecretKeyRef != nil {
				return "(secret)"
			}
			return ev.Value
		}
	}
	return ""
}

var _ = Describe("getBindplaneConfigEnvVars", func() {
	baseBindplane := func() *bindplanev1alpha1.Bindplane {
		return &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-bp",
				Namespace: "default",
			},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "license",
					Store: bindplanev1alpha1.StoreConfig{
						Postgres: &bindplanev1alpha1.PostgresConfig{
							Host: "pg",
						},
					},
				},
			},
		}
	}

	It("sets default metrics env vars when Metrics config is omitted", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneConfigEnvVars(bindplane)

		Expect(envVarByName(envVars, "BINDPLANE_METRICS_TYPE")).To(Equal("prometheus"))
		Expect(envVarByName(envVars, "BINDPLANE_METRICS_INTERVAL")).To(Equal("60s"))
		Expect(envVarByName(envVars, "BINDPLANE_METRICS_PROMETHEUS_ENDPOINT")).To(Equal("/metrics"))
		Expect(envVarByName(envVars, "BINDPLANE_TRACING_TYPE")).To(BeEmpty())
	})

	It("sets explicit metrics type prometheus, interval 60s, endpoint /metrics", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Metrics = &bindplanev1alpha1.MetricsConfig{
			Type:     "prometheus",
			Interval: "60s",
			Prometheus: &bindplanev1alpha1.MetricsPrometheusConfig{
				Endpoint: "/metrics",
			},
		}
		envVars := getBindplaneConfigEnvVars(bindplane)

		Expect(envVarByName(envVars, "BINDPLANE_METRICS_TYPE")).To(Equal("prometheus"))
		Expect(envVarByName(envVars, "BINDPLANE_METRICS_INTERVAL")).To(Equal("60s"))
		Expect(envVarByName(envVars, "BINDPLANE_METRICS_PROMETHEUS_ENDPOINT")).To(Equal("/metrics"))
	})

	It("sets tracing type otlp with endpoint, insecure, and sampling rate", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Tracing = &bindplanev1alpha1.TracingConfig{
			Type: "otlp",
			OTLP: &bindplanev1alpha1.TracingOTLPConfig{
				Endpoint: "http://otel:4317",
				Insecure: true,
			},
			SamplingRate: "0.5",
		}
		envVars := getBindplaneConfigEnvVars(bindplane)

		Expect(envVarByName(envVars, "BINDPLANE_TRACING_TYPE")).To(Equal("otlp"))
		Expect(envVarByName(envVars, "BINDPLANE_TRACING_OTLP_ENDPOINT")).To(Equal("http://otel:4317"))
		Expect(envVarByName(envVars, "BINDPLANE_TRACING_OTLP_INSECURE")).To(Equal("true"))
		Expect(envVarByName(envVars, "BINDPLANE_TRACING_SAMPLING_RATE")).To(Equal("0.5"))
	})

	It("sets metrics Prometheus auth username and password secret ref", func() {
		bindplane := baseBindplane()
		secretName := "metrics-auth"
		bindplane.Spec.Config.Metrics = &bindplanev1alpha1.MetricsConfig{
			Type:     "prometheus",
			Interval: "60s",
			Prometheus: &bindplanev1alpha1.MetricsPrometheusConfig{
				Endpoint: "/metrics",
				Username: "metrics-user",
				PasswordSecretRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
					Key:                  "password",
				},
			},
		}
		envVars := getBindplaneConfigEnvVars(bindplane)

		Expect(envVarByName(envVars, "BINDPLANE_METRICS_PROMETHEUS_USERNAME")).To(Equal("metrics-user"))
		Expect(envVarByName(envVars, "BINDPLANE_METRICS_PROMETHEUS_PASSWORD")).To(Equal("(secret)"))
	})

	It("does not set tracing env vars when Tracing is nil", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_TRACING_TYPE")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_TRACING_OTLP_ENDPOINT")).To(BeEmpty())
	})

	It("does not set tracing env vars when Tracing Type is empty", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Tracing = &bindplanev1alpha1.TracingConfig{}
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_TRACING_TYPE")).To(BeEmpty())
	})

	It("sets maxConcurrency and auditTrail retention with defaults when omitted", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_MAX_CONCURRENCY")).To(Equal("10"))
		Expect(envVarByName(envVars, "BINDPLANE_AUDIT_TRAIL_RETENTION_DAYS")).To(Equal("365"))
	})

	It("sets explicit maxConcurrency and auditTrail.retentionDays", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.MaxConcurrency = 20
		bindplane.Spec.Config.AuditTrail = &bindplanev1alpha1.AuditTrailConfig{RetentionDays: 180}
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_MAX_CONCURRENCY")).To(Equal("20"))
		Expect(envVarByName(envVars, "BINDPLANE_AUDIT_TRAIL_RETENTION_DAYS")).To(Equal("180"))
	})

	It("sets network webURL and corsAllowedOrigins only when configured", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_WEB_URL")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_CORS_ALLOWED_ORIGINS")).To(BeEmpty())

		bindplane.Spec.Config.Network = &bindplanev1alpha1.NetworkConfig{
			WebURL:             "https://bindplane.example.com",
			CorsAllowedOrigins: "https://app.example.com",
		}
		envVars = getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_WEB_URL")).To(Equal("https://bindplane.example.com"))
		Expect(envVarByName(envVars, "BINDPLANE_CORS_ALLOWED_ORIGINS")).To(Equal("https://app.example.com"))
	})

	It("does not set network TLS env vars when Network or TLS is nil", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_TLS_MIN_VERSION")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_TLS_CERT")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_TLS_KEY")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_TLS_CA")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_TLS_SKIP_VERIFY")).To(BeEmpty())

		bindplane.Spec.Config.Network = &bindplanev1alpha1.NetworkConfig{}
		envVars = getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_TLS_CERT")).To(BeEmpty())
	})

	It("sets network TLS minVersion and skipVerify only when no secret (no path env vars)", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Network = &bindplanev1alpha1.NetworkConfig{
			TLS: &bindplanev1alpha1.NetworkTLSConfig{
				MinVersion: "1.2",
				SkipVerify: true,
			},
		}
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_TLS_MIN_VERSION")).To(Equal("1.2"))
		Expect(envVarByName(envVars, "BINDPLANE_TLS_SKIP_VERIFY")).To(Equal("true"))
		Expect(envVarByName(envVars, "BINDPLANE_TLS_CERT")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_TLS_KEY")).To(BeEmpty())
	})

	It("sets network TLS cert and key paths when secretName, certKey, keyKey are set", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Network = &bindplanev1alpha1.NetworkConfig{
			TLS: &bindplanev1alpha1.NetworkTLSConfig{
				SecretName: "tls-secret",
				CertKey:    "tls.crt",
				KeyKey:     "tls.key",
			},
		}
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_TLS_CERT")).To(Equal("/etc/bindplane/network-tls/tls.crt"))
		Expect(envVarByName(envVars, "BINDPLANE_TLS_KEY")).To(Equal("/etc/bindplane/network-tls/tls.key"))
		Expect(envVarByName(envVars, "BINDPLANE_TLS_CA")).To(BeEmpty())
	})

	It("sets network TLS cert, key, and ca paths when caKey is set (mutual TLS)", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Network = &bindplanev1alpha1.NetworkConfig{
			TLS: &bindplanev1alpha1.NetworkTLSConfig{
				SecretName: "tls-secret",
				CertKey:    "tls.crt",
				KeyKey:     "tls.key",
				CAKey:      "ca.crt",
			},
		}
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_TLS_CERT")).To(Equal("/etc/bindplane/network-tls/tls.crt"))
		Expect(envVarByName(envVars, "BINDPLANE_TLS_KEY")).To(Equal("/etc/bindplane/network-tls/tls.key"))
		Expect(envVarByName(envVars, "BINDPLANE_TLS_CA")).To(Equal("/etc/bindplane/network-tls/ca.crt"))
	})
})

var _ = Describe("getNetworkTLSVolumeAndMount", func() {
	It("returns nil when Network or TLS is nil", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Store: bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
				},
			},
		}
		vols, mounts := getNetworkTLSVolumeAndMount(bindplane)
		Expect(vols).To(BeNil())
		Expect(mounts).To(BeNil())

		bindplane.Spec.Config.Network = &bindplanev1alpha1.NetworkConfig{}
		vols, mounts = getNetworkTLSVolumeAndMount(bindplane)
		Expect(vols).To(BeNil())
		Expect(mounts).To(BeNil())
	})

	It("returns nil when secretName or certKey or keyKey is missing", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Store: bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
					Network: &bindplanev1alpha1.NetworkConfig{
						TLS: &bindplanev1alpha1.NetworkTLSConfig{SecretName: "tls-secret"},
					},
				},
			},
		}
		vols, mounts := getNetworkTLSVolumeAndMount(bindplane)
		Expect(vols).To(BeNil())
		Expect(mounts).To(BeNil())
	})

	It("returns one volume and one mount when server TLS is configured", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Store: bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
					Network: &bindplanev1alpha1.NetworkConfig{
						TLS: &bindplanev1alpha1.NetworkTLSConfig{
							SecretName: "tls-secret",
							CertKey:    "tls.crt",
							KeyKey:     "tls.key",
						},
					},
				},
			},
		}
		vols, mounts := getNetworkTLSVolumeAndMount(bindplane)
		Expect(vols).To(HaveLen(1))
		Expect(vols[0].Name).To(Equal("network-tls"))
		Expect(vols[0].Secret).ToNot(BeNil())
		Expect(vols[0].Secret.SecretName).To(Equal("tls-secret"))
		Expect(mounts).To(HaveLen(1))
		Expect(mounts[0].Name).To(Equal("network-tls"))
		Expect(mounts[0].MountPath).To(Equal("/etc/bindplane/network-tls"))
	})
})

var _ = Describe("getBindplaneConfigEnvVars Postgres TLS", func() {
	baseBindplane := func() *bindplanev1alpha1.Bindplane {
		return &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "test-bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "license",
					Store: bindplanev1alpha1.StoreConfig{
						Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"},
					},
				},
			},
		}
	}

	It("defaults sslMode to disable when omitted", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_SSL_MODE")).To(Equal(postgresSSLModeDisable))
	})

	It("does not set postgres TLS path env vars when TLS is nil", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_SSL_ROOT_CERT")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_SSL_CERT")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_SSL_KEY")).To(BeEmpty())
	})

	It("sets postgres TLS root cert only when TLS has secretName and caKey (server-side TLS)", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Store.Postgres.TLS = &bindplanev1alpha1.PostgresTLSConfig{
			SecretName: "pg-tls",
			CAKey:      "ca.crt",
		}
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_SSL_ROOT_CERT")).To(Equal("/etc/bindplane/postgres-tls/ca.crt"))
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_SSL_CERT")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_SSL_KEY")).To(BeEmpty())
	})

	It("sets postgres TLS root cert, cert, and key when mutual TLS (caKey, certKey, keyKey)", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Store.Postgres.TLS = &bindplanev1alpha1.PostgresTLSConfig{
			SecretName: "pg-tls",
			CAKey:      "ca.crt",
			CertKey:    "tls.crt",
			KeyKey:     "tls.key",
		}
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_SSL_ROOT_CERT")).To(Equal("/etc/bindplane/postgres-tls/ca.crt"))
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_SSL_CERT")).To(Equal("/etc/bindplane/postgres-tls/tls.crt"))
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_SSL_KEY")).To(Equal("/etc/bindplane/postgres-tls/tls.key"))
	})

	It("does not set maxIdleConnections or maxIdleTime when omitted", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_MAX_IDLE_CONNECTIONS")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_MAX_IDLE_TIME")).To(BeEmpty())
	})

	It("sets maxIdleConnections and maxIdleTime when provided", func() {
		bindplane := baseBindplane()
		maxIdle := 5
		bindplane.Spec.Config.Store.Postgres.MaxIdleConnections = &maxIdle
		bindplane.Spec.Config.Store.Postgres.MaxIdleTime = "20s"
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_MAX_IDLE_CONNECTIONS")).To(Equal("5"))
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_MAX_IDLE_TIME")).To(Equal("20s"))
	})
})

var _ = Describe("getBindplaneCommonEnvVars profiling and pprof", func() {
	baseBindplane := func() *bindplanev1alpha1.Bindplane {
		return &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-bp",
				Namespace: "default",
			},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "license",
					Store: bindplanev1alpha1.StoreConfig{
						Postgres: &bindplanev1alpha1.PostgresConfig{
							Host: "pg",
						},
					},
				},
			},
		}
	}

	It("does not set profiling or pprof env vars when disabled or omitted", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneProfilingEnabledEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplanePprofEnabledEnvVar)).To(BeEmpty())
	})

	It("sets profiling env vars with default serviceName per component", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Profiling = &bindplanev1alpha1.ProfilingConfig{
			Enabled:   true,
			ProjectID: "my-gcp-project",
		}
		for _, tc := range []struct {
			component string
			wantName  string
		}{
			{nodeComponent, "bindplane-node"},
			{bindplaneJobsComponent, "bindplane-jobs"},
			{bindplaneJobsMigrateComponent, "bindplane-jobs-migrate"},
			{natsComponent, "bindplane-nats"},
		} {
			envVars := getBindplaneCommonEnvVars(bindplane, tc.component)
			Expect(envVarByName(envVars, bindplaneProfilingEnabledEnvVar)).To(Equal("true"))
			Expect(envVarByName(envVars, bindplaneProfilingProjectIDEnvVar)).To(Equal("my-gcp-project"))
			Expect(envVarByName(envVars, bindplaneProfilingServiceNameEnvVar)).To(Equal(tc.wantName))
		}
	})

	It("sets profiling env vars with explicit serviceName when set", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Profiling = &bindplanev1alpha1.ProfilingConfig{
			Enabled:     true,
			ProjectID:   "my-gcp-project",
			ServiceName: "custom-service",
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneProfilingServiceNameEnvVar)).To(Equal("custom-service"))
	})

	It("sets profiling noCPU, noAlloc, noHeap, noGoroutine, mutex when enabled", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Profiling = &bindplanev1alpha1.ProfilingConfig{
			Enabled:     true,
			ProjectID:   "proj",
			NoCPU:       true,
			NoAlloc:     true,
			NoHeap:      true,
			NoGoroutine: true,
			Mutex:       true,
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneProfilingNoCPUEnvVar)).To(Equal("true"))
		Expect(envVarByName(envVars, bindplaneProfilingNoAllocEnvVar)).To(Equal("true"))
		Expect(envVarByName(envVars, bindplaneProfilingNoHeapEnvVar)).To(Equal("true"))
		Expect(envVarByName(envVars, bindplaneProfilingNoGoroutineEnvVar)).To(Equal("true"))
		Expect(envVarByName(envVars, bindplaneProfilingMutexEnvVar)).To(Equal("true"))
	})

	It("sets pprof env vars with default endpoint when enabled and endpoint unset", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Pprof = &bindplanev1alpha1.PprofConfig{Enabled: true}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplanePprofEnabledEnvVar)).To(Equal("true"))
		Expect(envVarByName(envVars, bindplanePprofEndpointEnvVar)).To(Equal(defaultPprofEndpoint))
	})

	It("sets pprof env vars with explicit endpoint when set", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Pprof = &bindplanev1alpha1.PprofConfig{
			Enabled:  true,
			Endpoint: "0.0.0.0:6061",
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplanePprofEndpointEnvVar)).To(Equal("0.0.0.0:6061"))
	})
})

var _ = Describe("getPostgresTLSVolumeAndMount", func() {
	It("returns nil when Postgres or TLS is nil", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Store: bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
				},
			},
		}
		vols, mounts := getPostgresTLSVolumeAndMount(bindplane)
		Expect(vols).To(BeNil())
		Expect(mounts).To(BeNil())
	})

	It("returns nil when secretName or caKey is missing", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Store: bindplanev1alpha1.StoreConfig{
						Postgres: &bindplanev1alpha1.PostgresConfig{
							Host: "pg",
							TLS:  &bindplanev1alpha1.PostgresTLSConfig{SecretName: "pg-tls"},
						},
					},
				},
			},
		}
		vols, mounts := getPostgresTLSVolumeAndMount(bindplane)
		Expect(vols).To(BeNil())
		Expect(mounts).To(BeNil())
	})

	It("returns one volume and one mount when server-side TLS (caKey) is configured", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Store: bindplanev1alpha1.StoreConfig{
						Postgres: &bindplanev1alpha1.PostgresConfig{
							Host: "pg",
							TLS: &bindplanev1alpha1.PostgresTLSConfig{
								SecretName: "pg-tls",
								CAKey:      "ca.crt",
							},
						},
					},
				},
			},
		}
		vols, mounts := getPostgresTLSVolumeAndMount(bindplane)
		Expect(vols).To(HaveLen(1))
		Expect(vols[0].Name).To(Equal("postgres-tls"))
		Expect(vols[0].Secret).ToNot(BeNil())
		Expect(vols[0].Secret.SecretName).To(Equal("pg-tls"))
		Expect(mounts).To(HaveLen(1))
		Expect(mounts[0].MountPath).To(Equal("/etc/bindplane/postgres-tls"))
	})
})

// envVarSecretKeyRef returns the SecretKeySelector for the env var with the given name, or nil.
func envVarSecretKeyRef(envVars []corev1.EnvVar, name string) *corev1.SecretKeySelector {
	for _, ev := range envVars {
		if ev.Name == name && ev.ValueFrom != nil && ev.ValueFrom.SecretKeyRef != nil {
			return ev.ValueFrom.SecretKeyRef
		}
	}
	return nil
}

var _ = Describe("getTSDBEnvVars", func() {
	It("returns enable_remote, host, port, and username/password from generated secret when internal TLS is disabled", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "my-bp", Namespace: "default"},
		}
		envVars := getTSDBEnvVars(bindplane)
		Expect(envVars).To(HaveLen(5))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_ENABLE_REMOTE")).To(Equal("true"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_HOST")).To(Equal("my-bp-tsdb.default.svc"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_PORT")).To(Equal("9090"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_AUTH_USERNAME")).To(Equal("(secret)"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_AUTH_PASSWORD")).To(Equal("(secret)"))
		refUser := envVarSecretKeyRef(envVars, "BINDPLANE_PROMETHEUS_AUTH_USERNAME")
		refPass := envVarSecretKeyRef(envVars, "BINDPLANE_PROMETHEUS_AUTH_PASSWORD")
		Expect(refUser).ToNot(BeNil())
		Expect(refPass).ToNot(BeNil())
		Expect(refUser.Name).To(Equal("my-bp-tsdb-basic-auth"))
		Expect(refUser.Key).To(Equal(tsdbBasicAuthSecretKeyUser))
		Expect(refPass.Name).To(Equal("my-bp-tsdb-basic-auth"))
		Expect(refPass.Key).To(Equal(tsdbBasicAuthSecretKeyPass))
	})
	It("uses remote Prometheus env vars and skips operator-generated basic auth when remote.enable is true", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "my-bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "x",
					Store:   bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
					TSDB: &bindplanev1alpha1.TSDBConfig{
						Remote: &bindplanev1alpha1.TSDBRemoteConfig{
							Enable:          true,
							Host:            "vm.example.internal",
							QueryPathPrefix: "/select/0/prometheus",
							RemoteWrite: &bindplanev1alpha1.TSDBRemoteWriteConfig{
								Host: "vm-write.example.internal",
								Port: 8480,
							},
						},
					},
				},
			},
		}
		envVars := getTSDBEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_ENABLE_REMOTE")).To(Equal("true"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_HOST")).To(Equal("vm.example.internal"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_PORT")).To(Equal("9090"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_QUERY_PATH_PREFIX")).To(Equal("/select/0/prometheus"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_REMOTE_WRITE_HOST")).To(Equal("vm-write.example.internal"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_REMOTE_WRITE_PORT")).To(Equal("8480"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_REMOTE_WRITE_ENDPOINT")).To(Equal("/api/v1/write"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_AUTH_USERNAME")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_AUTH_PASSWORD")).To(BeEmpty())
	})
	It("uses remote write endpoint override when provided", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "my-bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "x",
					Store:   bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
					TSDB: &bindplanev1alpha1.TSDBConfig{
						Remote: &bindplanev1alpha1.TSDBRemoteConfig{
							Enable: true,
							Host:   "vm.example.internal",
							Port:   8080,
							RemoteWrite: &bindplanev1alpha1.TSDBRemoteWriteConfig{
								Host:     "vm-write.example.internal",
								Port:     18480,
								Endpoint: "/api/v1/push",
							},
						},
					},
				},
			},
		}
		envVars := getTSDBEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_PORT")).To(Equal("8080"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_REMOTE_WRITE_ENDPOINT")).To(Equal("/api/v1/push"))
	})
	It("adds Prometheus TLS env vars when cert-manager TLS is enabled", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "my-bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "x",
					Store:   bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
					TSDB: &bindplanev1alpha1.TSDBConfig{
						TLS: &bindplanev1alpha1.TSDBTLSConfig{
							CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "ca-issuer", Kind: "ClusterIssuer"},
						},
					},
				},
			},
		}
		envVars := getTSDBEnvVars(bindplane)
		Expect(envVars).To(HaveLen(9))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_ENABLE_TLS")).To(Equal("true"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_TLS_CERT")).To(Equal(internalTLSTSDBClientMountPath + "/tls.crt"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_TLS_KEY")).To(Equal(internalTLSTSDBClientMountPath + "/tls.key"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_TLS_CA")).To(Equal(internalTLSTSDBClientMountPath + "/ca.crt"))
	})
	It("adds BINDPLANE_PROMETHEUS_TLS_SKIP_VERIFY when prometheus TLS skipVerify is true", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "my-bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "x",
					Store:   bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
					TSDB: &bindplanev1alpha1.TSDBConfig{
						TLS: &bindplanev1alpha1.TSDBTLSConfig{
							CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "ca-issuer"},
							SkipVerify:  true,
						},
					},
				},
			},
		}
		envVars := getTSDBEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_TLS_SKIP_VERIFY")).To(Equal("true"))
		Expect(envVars).To(HaveLen(10))
	})
})

var _ = Describe("reconcileTSDB", func() {
	It("short-circuits when remote Prometheus mode is enabled", func() {
		reconciler := &BindplaneReconciler{}
		bindplane := &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "my-bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "x",
					Store:   bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
					TSDB: &bindplanev1alpha1.TSDBConfig{
						Remote: &bindplanev1alpha1.TSDBRemoteConfig{
							Enable: true,
							Host:   "vm.example.internal",
						},
					},
				},
			},
		}
		Expect(reconciler.reconcileTSDB(context.Background(), bindplane, logf.Log.WithName("test"))).To(Succeed())
	})
})

var _ = Describe("validateTSDBTLSConfig (controller_test)", func() {
	It("returns nil when config.Prometheus is nil", func() {
		bindplane := &bindplanev1alpha1.Bindplane{Spec: bindplanev1alpha1.BindplaneSpec{Config: bindplanev1alpha1.BindplaneConfigSpec{License: "x", Store: bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}}}}}
		Expect(validateTSDBTLSConfig(bindplane)).To(Succeed())
	})
	It("returns error when both secretName and certManager are set", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "x",
					Store:   bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
					TSDB: &bindplanev1alpha1.TSDBConfig{
						TLS: &bindplanev1alpha1.TSDBTLSConfig{
							SecretName:  "x",
							CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "issuer"},
						},
					},
				},
			},
		}
		Expect(validateTSDBTLSConfig(bindplane)).NotTo(Succeed())
		Expect(validateTSDBTLSConfig(bindplane).Error()).To(ContainSubstring("mutually exclusive"))
	})
	It("returns nil when CertManager has a valid name", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "x",
					Store:   bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
					TSDB: &bindplanev1alpha1.TSDBConfig{
						TLS: &bindplanev1alpha1.TSDBTLSConfig{
							CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "my-issuer", Kind: "Issuer"},
						},
					},
				},
			},
		}
		Expect(validateTSDBTLSConfig(bindplane)).To(Succeed())
	})
})

var _ = Describe("getInternalTLSVolumesAndMounts", func() {
	It("returns nil when cert-manager TLS is not configured", func() {
		bindplane := &bindplanev1alpha1.Bindplane{ObjectMeta: metav1.ObjectMeta{Name: "bp", Namespace: "default"}}
		vols, mounts := getInternalTLSVolumesAndMounts(bindplane)
		Expect(vols).To(BeNil())
		Expect(mounts).To(BeNil())
	})
	It("returns one volume and one mount when cert-manager TLS is enabled", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "x",
					Store:   bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
					TSDB: &bindplanev1alpha1.TSDBConfig{
						TLS: &bindplanev1alpha1.TSDBTLSConfig{
							CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "ca"},
						},
					},
				},
			},
		}
		vols, mounts := getInternalTLSVolumesAndMounts(bindplane)
		Expect(vols).To(HaveLen(1))
		Expect(mounts).To(HaveLen(1))
		Expect(vols[0].Name).To(Equal(internalTLSTSDBClientVolumeName))
		Expect(vols[0].Secret.SecretName).To(Equal("bp-tsdb-remote-write-client"))
		Expect(mounts[0].Name).To(Equal(internalTLSTSDBClientVolumeName))
		Expect(mounts[0].MountPath).To(Equal(internalTLSTSDBClientMountPath))
	})
})

var _ = Describe("getNatsTLSEnvVars", func() {
	It("returns nil when NATS TLS cert-manager is not configured", func() {
		bindplane := &bindplanev1alpha1.Bindplane{ObjectMeta: metav1.ObjectMeta{Name: "bp", Namespace: "default"}}
		envVars := getNatsTLSEnvVars(bindplane)
		Expect(envVars).To(BeNil())
	})
	It("returns NATS TLS env vars when spec.config.nats.tls.certManager is set", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Nats: &bindplanev1alpha1.NatsConfig{
						TLS: &bindplanev1alpha1.NatsTLSConfig{
							CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "nats-issuer"},
						},
					},
				},
			},
		}
		envVars := getNatsTLSEnvVars(bindplane)
		Expect(envVars).NotTo(BeNil())
		Expect(envVarByName(envVars, bindplaneNatsEnableTLSEnvVar)).To(Equal("true"))
		Expect(envVarByName(envVars, bindplaneNatsTLSCertEnvVar)).To(Equal(internalTLSNatsMountPath + "/tls.crt"))
		Expect(envVarByName(envVars, bindplaneNatsTLSKeyEnvVar)).To(Equal(internalTLSNatsMountPath + "/tls.key"))
		Expect(envVarByName(envVars, bindplaneNatsTLSCAEnvVar)).To(Equal(internalTLSNatsMountPath + "/ca.crt"))
		Expect(envVarByName(envVars, "BINDPLANE_NATS_TLS_SKIP_VERIFY")).To(BeEmpty())
	})
})

var _ = Describe("getNatsTLSVolumesAndMounts", func() {
	It("returns nil when NATS TLS cert-manager is not configured", func() {
		bindplane := &bindplanev1alpha1.Bindplane{ObjectMeta: metav1.ObjectMeta{Name: "bp", Namespace: "default"}}
		vols, mounts := getNatsTLSVolumesAndMounts(bindplane)
		Expect(vols).To(BeNil())
		Expect(mounts).To(BeNil())
	})
	It("returns one volume and one mount when spec.config.nats.tls.certManager is set", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Nats: &bindplanev1alpha1.NatsConfig{
						TLS: &bindplanev1alpha1.NatsTLSConfig{
							CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "nats-issuer"},
						},
					},
				},
			},
		}
		vols, mounts := getNatsTLSVolumesAndMounts(bindplane)
		Expect(vols).To(HaveLen(1))
		Expect(mounts).To(HaveLen(1))
		Expect(vols[0].Name).To(Equal(internalTLSNatsVolumeName))
		Expect(vols[0].Secret.SecretName).To(Equal("bp-nats-tls"))
		Expect(mounts[0].Name).To(Equal(internalTLSNatsVolumeName))
		Expect(mounts[0].MountPath).To(Equal(internalTLSNatsMountPath))
	})
})

var _ = Describe("generateTSDBBasicAuthSecretData", func() {
	It("returns username, password, and web-config with bcrypt hash", func() {
		data, err := generateTSDBBasicAuthSecretData()
		Expect(err).NotTo(HaveOccurred())
		Expect(data).To(HaveKey(tsdbBasicAuthSecretKeyUser))
		Expect(data).To(HaveKey(tsdbBasicAuthSecretKeyPass))
		Expect(data).To(HaveKey(tsdbBasicAuthSecretKeyWeb))
		Expect(string(data[tsdbBasicAuthSecretKeyUser])).To(Equal(tsdbBasicAuthUsername))
		Expect(data[tsdbBasicAuthSecretKeyPass]).To(HaveLen(32))
		webConfig := string(data[tsdbBasicAuthSecretKeyWeb])
		Expect(webConfig).To(ContainSubstring("basic_auth_users:"))
		Expect(webConfig).To(ContainSubstring(tsdbBasicAuthUsername + ":"))
		Expect(webConfig).To(MatchRegexp(`\$2[aby]\$\d{2}\$`)) // bcrypt hash prefix
	})
})
