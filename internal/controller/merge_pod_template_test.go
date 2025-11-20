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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	bindplanev1alpha1 "github.com/bindplane-operator/bindplane-operator/api/v1alpha1"
)

var _ = Describe("mergePodTemplateSpec", func() {
	var operatorManaged corev1.PodTemplateSpec

	BeforeEach(func() {
		operatorManaged = corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app.kubernetes.io/name":      "bindplane",
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
							"user-label":                  "user-value",
							"app.kubernetes.io/component": "user-override",
						},
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.ObjectMeta.Labels).To(HaveKeyWithValue("app.kubernetes.io/name", "bindplane"))
			Expect(result.ObjectMeta.Labels).To(HaveKeyWithValue("user-label", "user-value"))
			Expect(result.ObjectMeta.Labels).To(HaveKeyWithValue("app.kubernetes.io/component", "user-override"))
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
			operatorManaged.ObjectMeta.Labels = nil
			operatorManaged.ObjectMeta.Annotations = nil

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
