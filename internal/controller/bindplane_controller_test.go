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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	bindplanev1alpha1 "github.com/bindplane-operator/bindplane-operator/api/v1alpha1"
)

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
					// TODO(user): Specify other spec details if needed.
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
				TerminationGracePeriodSeconds: int64Ptr(60),
				SecurityContext: &corev1.PodSecurityContext{
					FSGroup:    int64Ptr(65534),
					RunAsGroup: int64Ptr(65534),
					RunAsUser:  int64Ptr(65534),
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
			userFSGroup := int64Ptr(1000)
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						SecurityContext: &corev1.PodSecurityContext{
							FSGroup:   userFSGroup,
							RunAsUser: int64Ptr(1000),
						},
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.SecurityContext).ToNot(BeNil())
			Expect(result.Spec.SecurityContext.FSGroup).To(Equal(userFSGroup))
			Expect(result.Spec.SecurityContext.RunAsUser).To(Equal(int64Ptr(1000)))
			// Operator-managed fields should be preserved if not overridden
			Expect(result.Spec.SecurityContext.RunAsGroup).To(Equal(int64Ptr(65534)))
		})

		It("should handle nil securityContext in operator-managed template", func() {
			operatorManaged.Spec.SecurityContext = nil

			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						SecurityContext: &corev1.PodSecurityContext{
							FSGroup: int64Ptr(1000),
						},
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.SecurityContext).ToNot(BeNil())
			Expect(result.Spec.SecurityContext.FSGroup).To(Equal(int64Ptr(1000)))
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
						TerminationGracePeriodSeconds: int64Ptr(30),
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.TerminationGracePeriodSeconds).To(Equal(int64Ptr(60)))
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
			Expect(result.Spec.TerminationGracePeriodSeconds).To(Equal(int64Ptr(60)))
		})
	})
})
