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

var _ = Describe("Bindplane webhook", Ordered, func() {
	AfterEach(func() {
		cleanupBindplane(webhookBindplaneName, bindplaneNamespace, 30*time.Second)
	})

	DescribeTable("rejects invalid manifests and still accepts the minimal valid manifest",
		func(fixtureName, expectedMessage string) {
			expectFixtureApplyFailure(fixtureName, bindplaneNamespace, expectedMessage)

			By("applying the minimal paused Bindplane manifest")
			_, err := applyFixture("bindplane-webhook-valid.yaml", bindplaneNamespace)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for the Bindplane custom resource to be admitted")
			waitForBindplaneFinalizer(webhookBindplaneName, bindplaneNamespace, defaultEventuallyShortTimeout)
			waitForBindplaneCondition(
				webhookBindplaneName,
				bindplaneNamespace,
				"Reconciled",
				metav1.ConditionFalse,
				"Paused",
				defaultEventuallyShortTimeout,
			)
		},
		Entry("when license configuration is missing",
			"bindplane-invalid-missing-license.yaml",
			"exactly one of license or licenseSecretRef must be set",
		),
		Entry("when postgres host is empty",
			"bindplane-invalid-postgres-host.yaml",
			"spec.config.store.postgres.host must not be empty",
		),
		Entry("when bindplane replicas is zero",
			"bindplane-invalid-zero-replicas.yaml",
			"spec.bindplane.replicas must be >= 1",
		),
		Entry("when the Bindplane name is too long",
			"bindplane-invalid-name-too-long.yaml",
			"is too long: must be at most",
		),
	)

	It("rejects invalid updates on an existing Bindplane custom resource", func() {
		By("creating a valid paused Bindplane custom resource")
		_, err := applyFixture("bindplane-webhook-valid.yaml", bindplaneNamespace)
		Expect(err).NotTo(HaveOccurred())
		waitForBindplaneFinalizer(webhookBindplaneName, bindplaneNamespace, defaultEventuallyShortTimeout)

		By("verifying the webhook rejects an invalid update")
		_, err = runCmd(kubectl(
			bindplaneNamespace,
			"patch",
			"bindplane",
			webhookBindplaneName,
			"--type=merge",
			"-p",
			`{"spec":{"bindplane":{"replicas":0}}}`,
		))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("spec.bindplane.replicas must be >= 1"))

		By("confirming the original resource still exists in the paused state")
		waitForBindplaneCondition(
			webhookBindplaneName,
			bindplaneNamespace,
			"Reconciled",
			metav1.ConditionFalse,
			"Paused",
			defaultEventuallyShortTimeout,
		)
	})
})
