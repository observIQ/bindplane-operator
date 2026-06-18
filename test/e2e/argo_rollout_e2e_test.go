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

package e2e

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Bindplane with Argo Rollouts", Ordered, Label(ginkgoLabelRequiresArgoRollouts), func() {
	AfterEach(func() {
		cleanupBindplane(argoRolloutsBindplaneName, bindplaneNamespace, 60*time.Second)
	})

	It("accepts a paused Bindplane with argoRollout enabled and adds the finalizer", func() {
		By("applying the paused Bindplane fixture with argoRollout enabled")
		_, err := applyFixture("bindplane-argo-rollout-paused.yaml", bindplaneNamespace)
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the finalizer to be set (proves operator started with Argo Rollouts CRD watch)")
		waitForBindplaneFinalizer(argoRolloutsBindplaneName, bindplaneNamespace, defaultEventuallyShortTimeout)

		By("waiting for the Reconciled=False/Paused condition")
		waitForBindplaneCondition(
			argoRolloutsBindplaneName,
			bindplaneNamespace,
			"Reconciled",
			metav1.ConditionFalse,
			"Paused",
			defaultEventuallyShortTimeout,
		)
	})

	It("creates a Rollout (not a Deployment) for the node component", func() {
		By("applying the Bindplane fixture with argoRollout enabled")
		_, err := applyFixture("bindplane-argo-rollout.yaml", bindplaneNamespace)
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the finalizer to be set")
		waitForBindplaneFinalizer(argoRolloutsBindplaneName, bindplaneNamespace, defaultEventuallyShortTimeout)

		By("bypassing the migration gate (no postgres in this test)")
		skipMigrateJob(argoRolloutsBindplaneName, bindplaneNamespace, defaultEventuallyShortTimeout)

		By("waiting for the node Rollout to be created")
		waitForRolloutExists(argoRolloutsBindplaneName+"-node", bindplaneNamespace, defaultEventuallyShortTimeout)

		By("verifying no Deployment exists for the node component")
		_, err = getDeployment(argoRolloutsBindplaneName+"-node", bindplaneNamespace)
		Expect(err).To(HaveOccurred(), "expected no Deployment for the node component in Argo Rollout mode")
	})
})
