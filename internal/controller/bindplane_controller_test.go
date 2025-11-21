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
