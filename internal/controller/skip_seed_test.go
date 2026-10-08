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

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

// Only the Bindplane Jobs pod seeds resources on startup. Every other
// `bindplane serve` workload passes --skip-seed.
var _ = Describe("skip seed", func() {
	var r *BindplaneReconciler

	newBindplane := func() *bindplanev1alpha1.Bindplane {
		bp := newTestBindplane("my-bp", "default")
		bp.Spec.Bindplane.Replicas = new(int32(2))
		bp.Spec.Nats = &bindplanev1alpha1.NatsComponentSpec{Replicas: new(int32(3))}
		return bp
	}

	BeforeEach(func() {
		r = newReconciler()
	})

	It("node deployment passes --skip-seed", func() {
		bp := newBindplane()
		container := r.nodeDeployment(bp).Spec.Template.Spec.Containers[0]
		Expect(container.Args).To(Equal([]string{skipSeedArg}))
	})

	It("node rollout passes --skip-seed", func() {
		bp := newTestBindplaneWithArgoRollout("my-bp", "default")
		container := r.nodeRollout(bp).Spec.Template.Spec.Containers[0]
		Expect(container.Args).To(Equal([]string{skipSeedArg}))
	})

	It("opamp deployment passes --skip-seed", func() {
		bp := newTestBindplaneWithOpAMP("my-bp", "default")
		container := r.opampDeployment(bp).Spec.Template.Spec.Containers[0]
		Expect(container.Args).To(Equal([]string{skipSeedArg}))
	})

	It("nats statefulset passes --skip-seed", func() {
		bp := newBindplane()
		container := r.natsStatefulSet(bp).Spec.Template.Spec.Containers[0]
		Expect(container.Args).To(Equal([]string{skipSeedArg}))
	})

	It("jobs deployment does not pass --skip-seed so it seeds on startup", func() {
		bp := newTestBindplane("my-bp", "default")
		container := r.bindplaneJobsDeployment(bp).Spec.Template.Spec.Containers[0]
		Expect(container.Args).NotTo(ContainElement(skipSeedArg))
		Expect(container.Command).NotTo(ContainElement(skipSeedArg))
	})

	It("jobs migrate job does not pass --skip-seed", func() {
		bp := newTestBindplane("my-bp", "default")
		container := r.bindplaneJobsMigrateJob(bp).Spec.Template.Spec.Containers[0]
		Expect(container.Command).NotTo(ContainElement(skipSeedArg))
		Expect(container.Args).NotTo(ContainElement(skipSeedArg))
	})

	It("user podTemplate cannot remove --skip-seed from the node container", func() {
		bp := newBindplane()
		bp.Spec.Bindplane.PodTemplate = &bindplanev1alpha1.PodTemplateSpec{
			PodTemplateSpec: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: nodeContainerName,
							Args: []string{"--something-else"},
						},
					},
				},
			},
		}
		container := r.nodeDeployment(bp).Spec.Template.Spec.Containers[0]
		Expect(container.Args).To(Equal([]string{skipSeedArg}))
	})
})
