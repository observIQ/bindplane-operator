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

func newTestBindplaneWithArgoRollout(name, namespace string) *bindplanev1alpha1.Bindplane {
	bp := newTestBindplane(name, namespace)
	bp.Spec.Bindplane.ArgoRollout = &bindplanev1alpha1.ArgoRolloutSpec{
		Enabled: true,
	}
	return bp
}

var _ = Describe("nodeRollout", func() {
	var (
		r         *BindplaneReconciler
		bindplane *bindplanev1alpha1.Bindplane
	)

	BeforeEach(func() {
		r = newReconciler()
		bindplane = newTestBindplaneWithArgoRollout("my-bp", "default")
	})

	It("uses node component name and labels", func() {
		rollout := r.nodeRollout(bindplane)
		Expect(rollout.Name).To(Equal("my-bp-node"))
		Expect(rollout.Namespace).To(Equal("default"))
		Expect(rollout.Labels[labelKeyComponent]).To(Equal(nodeComponent))
	})

	It("sets BlueGreen strategy with activeService pointing to node service", func() {
		rollout := r.nodeRollout(bindplane)
		Expect(rollout.Spec.Strategy.BlueGreen).NotTo(BeNil())
		Expect(rollout.Spec.Strategy.BlueGreen.ActiveService).To(Equal("my-bp-node"))
	})

	It("enables autoPromotion by default", func() {
		rollout := r.nodeRollout(bindplane)
		Expect(rollout.Spec.Strategy.BlueGreen.AutoPromotionEnabled).NotTo(BeNil())
		Expect(*rollout.Spec.Strategy.BlueGreen.AutoPromotionEnabled).To(BeTrue())
	})

	It("respects a user-provided AutoPromotionEnabled=false", func() {
		f := false
		bindplane.Spec.Bindplane.ArgoRollout.AutoPromotionEnabled = &f
		rollout := r.nodeRollout(bindplane)
		Expect(rollout.Spec.Strategy.BlueGreen.AutoPromotionEnabled).NotTo(BeNil())
		Expect(*rollout.Spec.Strategy.BlueGreen.AutoPromotionEnabled).To(BeFalse())
	})

	It("does not set ScaleDownDelaySeconds when not provided", func() {
		rollout := r.nodeRollout(bindplane)
		Expect(rollout.Spec.Strategy.BlueGreen.ScaleDownDelaySeconds).To(BeNil())
	})

	It("sets ScaleDownDelaySeconds when provided", func() {
		delay := int32(120)
		bindplane.Spec.Bindplane.ArgoRollout.ScaleDownDelaySeconds = &delay
		rollout := r.nodeRollout(bindplane)
		Expect(rollout.Spec.Strategy.BlueGreen.ScaleDownDelaySeconds).NotTo(BeNil())
		Expect(*rollout.Spec.Strategy.BlueGreen.ScaleDownDelaySeconds).To(Equal(int32(120)))
	})

	It("uses the user-specified replicas", func() {
		replicas := int32(5)
		bindplane.Spec.Bindplane.Replicas = &replicas
		rollout := r.nodeRollout(bindplane)
		Expect(rollout.Spec.Replicas).NotTo(BeNil())
		Expect(*rollout.Spec.Replicas).To(Equal(int32(5)))
	})

	It("sets Replicas to nil when HPA is enabled", func() {
		bindplane.Spec.Bindplane.Autoscaling = &bindplanev1alpha1.NodeAutoscalingSpec{Enabled: true}
		rollout := r.nodeRollout(bindplane)
		Expect(rollout.Spec.Replicas).To(BeNil())
	})

	It("uses the correct image", func() {
		rollout := r.nodeRollout(bindplane)
		Expect(rollout.Spec.Template.Spec.Containers).NotTo(BeEmpty())
		Expect(rollout.Spec.Template.Spec.Containers[0].Image).To(Equal(getNodeImage(bindplane)))
	})

	It("uses operator-managed selector labels", func() {
		rollout := r.nodeRollout(bindplane)
		Expect(rollout.Spec.Selector).NotTo(BeNil())
		Expect(rollout.Spec.Selector.MatchLabels[labelKeyComponent]).To(Equal(nodeComponent))
		Expect(rollout.Spec.Selector.MatchLabels[labelKeyInstance]).To(Equal("my-bp"))
	})

	It("applies default minReadySeconds from termination grace period", func() {
		rollout := r.nodeRollout(bindplane)
		Expect(rollout.Spec.MinReadySeconds).To(Equal(int32(nodeTerminationGracePeriodSeconds(bindplane))))
	})

	It("respects user-provided MinReadySeconds", func() {
		mrs := int32(42)
		bindplane.Spec.Bindplane.MinReadySeconds = &mrs
		rollout := r.nodeRollout(bindplane)
		Expect(rollout.Spec.MinReadySeconds).To(Equal(int32(42)))
	})

	It("sets BINDPLANE_MODE=node", func() {
		rollout := r.nodeRollout(bindplane)
		containers := rollout.Spec.Template.Spec.Containers
		Expect(containers).NotTo(BeEmpty())
		modeVar := findEnvVar(containers[0].Env, bindplaneModeEnvVar)
		Expect(modeVar).NotTo(BeNil())
		Expect(modeVar.Value).To(Equal(nodeModeValue))
	})
})
