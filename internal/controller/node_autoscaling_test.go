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

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

var _ = Describe("nodeHPAScaleTargetRef", func() {
	var bindplane *bindplanev1alpha1.Bindplane

	BeforeEach(func() {
		bindplane = newTestBindplane("my-bp", "default")
		bindplane.Spec.Bindplane.Autoscaling = &bindplanev1alpha1.NodeAutoscalingSpec{Enabled: true}
	})

	It("targets a Deployment when ArgoRollout is nil", func() {
		ref := nodeHPAScaleTargetRef(bindplane)
		Expect(ref.Kind).To(Equal("Deployment"))
		Expect(ref.APIVersion).To(Equal("apps/v1"))
		Expect(ref.Name).To(Equal("my-bp-node"))
	})

	It("targets a Deployment when ArgoRollout.Enabled is false", func() {
		bindplane.Spec.Bindplane.ArgoRollout = &bindplanev1alpha1.ArgoRolloutSpec{Enabled: false}
		ref := nodeHPAScaleTargetRef(bindplane)
		Expect(ref.Kind).To(Equal("Deployment"))
		Expect(ref.APIVersion).To(Equal("apps/v1"))
	})

	It("targets a Rollout when ArgoRollout.Enabled is true", func() {
		bindplane.Spec.Bindplane.ArgoRollout = &bindplanev1alpha1.ArgoRolloutSpec{Enabled: true}
		ref := nodeHPAScaleTargetRef(bindplane)
		Expect(ref.Kind).To(Equal("Rollout"))
		Expect(ref.APIVersion).To(Equal("argoproj.io/v1alpha1"))
		Expect(ref.Name).To(Equal("my-bp-node"))
	})
})

var _ = Describe("nodeHPA with ArgoRollout", func() {
	var (
		r         *BindplaneReconciler
		bindplane *bindplanev1alpha1.Bindplane
	)

	BeforeEach(func() {
		r = newReconciler()
		bindplane = newTestBindplane("my-bp", "default")
		bindplane.Spec.Bindplane.Autoscaling = &bindplanev1alpha1.NodeAutoscalingSpec{Enabled: true}
	})

	It("sets ScaleTargetRef to Deployment when ArgoRollout is nil", func() {
		hpa := r.nodeHPA(bindplane)
		Expect(hpa.Spec.ScaleTargetRef.Kind).To(Equal("Deployment"))
		Expect(hpa.Spec.ScaleTargetRef.APIVersion).To(Equal("apps/v1"))
	})

	It("sets ScaleTargetRef to Rollout when ArgoRollout is enabled", func() {
		bindplane.Spec.Bindplane.ArgoRollout = &bindplanev1alpha1.ArgoRolloutSpec{Enabled: true}
		hpa := r.nodeHPA(bindplane)
		Expect(hpa.Spec.ScaleTargetRef.Kind).To(Equal("Rollout"))
		Expect(hpa.Spec.ScaleTargetRef.APIVersion).To(Equal("argoproj.io/v1alpha1"))
	})
})
