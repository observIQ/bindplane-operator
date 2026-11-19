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

	rolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

// progressState summarizes a workloadProgress for table assertions.
type progressState string

const (
	stateDone    progressState = "done"
	stateWaiting progressState = "waiting"
	stateFailed  progressState = "failed"
	statePaused  progressState = "paused"
)

func stateOf(p workloadProgress) progressState {
	switch {
	case p.failed:
		return stateFailed
	case p.paused:
		return statePaused
	case p.done:
		return stateDone
	default:
		return stateWaiting
	}
}

// rolledOutDeployment returns a Deployment whose status matches its spec at generation 1.
func rolledOutDeployment(name string, replicas int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespaceDefault, Generation: 1},
		Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
		Status: appsv1.DeploymentStatus{
			ObservedGeneration: 1,
			Replicas:           replicas,
			UpdatedReplicas:    replicas,
			ReadyReplicas:      replicas,
			AvailableReplicas:  replicas,
		},
	}
}

// rolledOutStatefulSet returns a StatefulSet whose status matches its spec at generation 1.
func rolledOutStatefulSet(name string, replicas int32) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespaceDefault, Generation: 1},
		Spec:       appsv1.StatefulSetSpec{Replicas: &replicas},
		Status: appsv1.StatefulSetStatus{
			ObservedGeneration: 1,
			Replicas:           replicas,
			ReadyReplicas:      replicas,
			UpdatedReplicas:    replicas,
			CurrentRevision:    name + "-1",
			UpdateRevision:     name + "-1",
		},
	}
}

// healthyNodeRollout returns the "bp-node" Rollout that the Argo Rollouts controller reports
// Healthy at generation 1.
func healthyNodeRollout(replicas int32) *rolloutsv1alpha1.Rollout {
	return &rolloutsv1alpha1.Rollout{
		ObjectMeta: metav1.ObjectMeta{Name: "bp-node", Namespace: testNamespaceDefault, Generation: 1},
		Spec:       rolloutsv1alpha1.RolloutSpec{Replicas: &replicas},
		Status: rolloutsv1alpha1.RolloutStatus{
			ObservedGeneration: "1",
			Phase:              rolloutsv1alpha1.RolloutPhaseHealthy,
			ReadyReplicas:      replicas,
		},
	}
}

var _ = Describe("deploymentProgress", func() {
	DescribeTable("reports the rollout state",
		func(mutate func(*appsv1.Deployment), want progressState, wantMessage string) {
			dep := rolledOutDeployment("bp-jobs", 3)
			mutate(dep)
			p := deploymentProgress("jobs", dep)
			Expect(stateOf(p)).To(Equal(want))
			Expect(p.component).To(Equal("jobs"))
			Expect(p.message).To(ContainSubstring(wantMessage))
		},
		Entry("rolled out", func(*appsv1.Deployment) {}, stateDone, ""),
		Entry("stale child: new generation not observed though fully rolled out",
			func(d *appsv1.Deployment) { d.Generation = 2 }, stateWaiting, "spec update to be observed"),
		Entry("HPA-written spec.replicas of 2 is done",
			func(d *appsv1.Deployment) {
				two := int32(2)
				d.Spec.Replicas = &two
				d.Status.Replicas, d.Status.UpdatedReplicas, d.Status.ReadyReplicas, d.Status.AvailableReplicas = 2, 2, 2, 2
			}, stateDone, ""),
		Entry("updated replicas below desired",
			func(d *appsv1.Deployment) { d.Status.UpdatedReplicas = 1 }, stateWaiting, "1 out of 3 new replicas have been updated"),
		Entry("old replicas pending termination",
			func(d *appsv1.Deployment) { d.Status.Replicas = 5 }, stateWaiting, "2 old replicas are pending termination"),
		Entry("updated replicas not yet available",
			func(d *appsv1.Deployment) { d.Status.AvailableReplicas = 2 }, stateWaiting, "2 of 3 updated replicas are available"),
		Entry("progress deadline exceeded",
			func(d *appsv1.Deployment) {
				d.Status.UpdatedReplicas = 1
				d.Status.Conditions = []appsv1.DeploymentCondition{{
					Type: appsv1.DeploymentProgressing, Status: corev1.ConditionFalse, Reason: "ProgressDeadlineExceeded",
				}}
			}, stateFailed, "exceeded its progress deadline"),
	)

	It("reports ready replicas", func() {
		dep := rolledOutDeployment("bp-jobs", 3)
		dep.Status.ReadyReplicas = 2
		Expect(deploymentProgress("jobs", dep).ready).To(Equal(int32(2)))
	})
})

var _ = Describe("statefulSetProgress", func() {
	DescribeTable("reports the rollout state",
		func(mutate func(*appsv1.StatefulSet), want progressState, wantMessage string) {
			sts := rolledOutStatefulSet("bp-nats", 3)
			mutate(sts)
			p := statefulSetProgress("nats", sts)
			Expect(stateOf(p)).To(Equal(want))
			Expect(p.message).To(ContainSubstring(wantMessage))
		},
		Entry("rolled out", func(*appsv1.StatefulSet) {}, stateDone, ""),
		Entry("never observed", func(s *appsv1.StatefulSet) { s.Status.ObservedGeneration = 0 }, stateWaiting, "spec update to be observed"),
		Entry("stale child: new generation not observed though fully rolled out",
			func(s *appsv1.StatefulSet) { s.Generation = 2 }, stateWaiting, "spec update to be observed"),
		Entry("pods not ready", func(s *appsv1.StatefulSet) { s.Status.ReadyReplicas = 2 }, stateWaiting, "2 of 3 pods are ready"),
		Entry("pods not updated", func(s *appsv1.StatefulSet) { s.Status.UpdatedReplicas = 1 }, stateWaiting, "1 out of 3 pods have been updated"),
		Entry("revision rollout incomplete", func(s *appsv1.StatefulSet) { s.Status.UpdateRevision = "bp-nats-2" }, stateWaiting, "bp-nats-2"),
	)
})

var _ = Describe("rolloutProgress", func() {
	DescribeTable("reports the rollout state",
		func(mutate func(*rolloutsv1alpha1.Rollout), want progressState, wantMessage string) {
			ro := healthyNodeRollout(3)
			mutate(ro)
			p := rolloutProgress("node", ro)
			Expect(stateOf(p)).To(Equal(want))
			Expect(p.message).To(ContainSubstring(wantMessage))
		},
		Entry("healthy", func(*rolloutsv1alpha1.Rollout) {}, stateDone, ""),
		Entry("stale child: healthy at an old generation",
			func(r *rolloutsv1alpha1.Rollout) { r.Generation = 2 }, stateWaiting, "spec update to be observed"),
		Entry("observedGeneration not numeric",
			func(r *rolloutsv1alpha1.Rollout) { r.Status.ObservedGeneration = "5d8f7c9b" }, stateWaiting, "spec update to be observed"),
		Entry("observedGeneration empty",
			func(r *rolloutsv1alpha1.Rollout) { r.Status.ObservedGeneration = "" }, stateWaiting, "spec update to be observed"),
		Entry("unpausing", func(r *rolloutsv1alpha1.Rollout) { r.Status.ControllerPause = true }, stateWaiting, "waiting for rollout to unpause"),
		Entry("progressing",
			func(r *rolloutsv1alpha1.Rollout) {
				r.Status.Phase = rolloutsv1alpha1.RolloutPhaseProgressing
				r.Status.Message = "active service cutover pending"
			}, stateWaiting, "active service cutover pending"),
		Entry("degraded",
			func(r *rolloutsv1alpha1.Rollout) {
				r.Status.Phase = rolloutsv1alpha1.RolloutPhaseDegraded
				r.Status.Message = "ProgressDeadlineExceeded: ReplicaSet has timed out progressing."
			}, stateFailed, "ProgressDeadlineExceeded"),
		Entry("paused awaiting promotion",
			func(r *rolloutsv1alpha1.Rollout) {
				r.Status.Phase = rolloutsv1alpha1.RolloutPhasePaused
				r.Status.ControllerPause = true
				r.Status.PauseConditions = []rolloutsv1alpha1.PauseCondition{{Reason: rolloutsv1alpha1.PauseReasonBlueGreenPause}}
			}, statePaused, nodeRolloutPausedMessage),
	)
})

var _ = Describe("migrateProgress", func() {
	bp := newTestBindplane("bp", testNamespaceDefault)

	It("reports nothing when migration is complete", func() {
		_, ok := migrateProgress(bp, migrateComplete)
		Expect(ok).To(BeFalse())
	})

	It("reports a running Job as waiting", func() {
		p, ok := migrateProgress(bp, migrateInProgress)
		Expect(ok).To(BeTrue())
		Expect(stateOf(p)).To(Equal(stateWaiting))
		Expect(p.component).To(Equal(jobsMigrateProgressComponent))
		Expect(p.message).To(ContainSubstring("bp-migrate"))
	})

	It("reports a failed Job as failed", func() {
		p, ok := migrateProgress(bp, migrateFailed)
		Expect(ok).To(BeTrue())
		Expect(stateOf(p)).To(Equal(stateFailed))
	})
})

var _ = Describe("setPhaseFromProgress", func() {
	done := func(c string) workloadProgress { return workloadProgress{component: c, done: true} }
	pending := func(c, msg string) workloadProgress { return workloadProgress{component: c, message: msg} }
	failed := func(c, msg string) workloadProgress {
		return workloadProgress{component: c, failed: true, message: msg}
	}
	paused := func(c string) workloadProgress {
		return workloadProgress{component: c, paused: true, message: nodeRolloutPausedMessage}
	}

	DescribeTable("sets the phase and Ready condition",
		func(progress []workloadProgress, wantPhase string, wantStatus metav1.ConditionStatus, wantReason, wantMessage string) {
			bp := newTestBindplane("bp", testNamespaceDefault)
			bp.Generation = 7
			setPhaseFromProgress(bp, progress)
			Expect(bp.Status.Phase).To(Equal(wantPhase))
			cond := meta.FindStatusCondition(bp.Status.Conditions, conditionTypeReady)
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(wantStatus))
			Expect(cond.Reason).To(Equal(wantReason))
			Expect(cond.ObservedGeneration).To(Equal(int64(7)))
			Expect(cond.Message).To(ContainSubstring(wantMessage))
		},
		Entry("all done", []workloadProgress{done("node"), done("nats")},
			phaseReady, metav1.ConditionTrue, reasonWorkloadsReady, ""),
		Entry("one waiting", []workloadProgress{done("node"), pending("opamp", "1 of 2 updated replicas are available")},
			phaseApplyingChanges, metav1.ConditionFalse, reasonWorkloadsProgressing, "opamp: 1 of 2 updated replicas are available"),
		Entry("paused Rollout gives RolloutPaused", []workloadProgress{paused("node"), pending("nats", "x")},
			phaseApplyingChanges, metav1.ConditionFalse, reasonRolloutPaused, "node: "+nodeRolloutPausedMessage),
		Entry("failed wins over paused and waiting", []workloadProgress{paused("node"), pending("nats", "x"), failed("jobs", "boom")},
			phaseDegraded, metav1.ConditionFalse, reasonWorkloadFailed, "jobs: boom"),
	)

	It("lists every waiting component", func() {
		bp := newTestBindplane("bp", testNamespaceDefault)
		setPhaseFromProgress(bp, []workloadProgress{pending("node", "a"), pending("opamp", "b")})
		cond := meta.FindStatusCondition(bp.Status.Conditions, conditionTypeReady)
		Expect(cond.Message).To(Equal("node: a; opamp: b"))
	})
})

var _ = Describe("updateReadyReplicaStatus with an Argo Rollout node", func() {
	newFakeReconciler := func(objs ...client.Object) *BindplaneReconciler {
		c := newFakeClient(objs...)
		return &BindplaneReconciler{Client: c, Scheme: c.Scheme()}
	}

	newRolloutBindplane := func() *bindplanev1alpha1.Bindplane {
		bp := newTestBindplane("bp", testNamespaceDefault)
		bp.Generation = 1
		bp.Spec.Bindplane.ArgoRollout = &bindplanev1alpha1.ArgoRolloutSpec{Enabled: true}
		return bp
	}

	// children returns the rolled out non-node workloads for the "bp" CR.
	children := func() []client.Object {
		return []client.Object{
			rolledOutStatefulSet("bp-nats", 3),
			rolledOutStatefulSet("bp-tsdb", 1),
			rolledOutDeployment("bp-transform-agent", 1),
			rolledOutDeployment("bp-jobs", 1),
		}
	}

	It("reports ApplyingChanges while the node Rollout is Progressing", func() {
		ro := healthyNodeRollout(2)
		ro.Status.Phase = rolloutsv1alpha1.RolloutPhaseProgressing
		ro.Status.Message = "active service cutover pending"
		r := newFakeReconciler(append(children(), ro)...)

		bp := newRolloutBindplane()
		poll := r.updateReadyReplicaStatus(context.Background(), bp, migrateComplete)

		Expect(poll).To(BeTrue())
		Expect(bp.Status.Phase).To(Equal(phaseApplyingChanges))
		cond := meta.FindStatusCondition(bp.Status.Conditions, conditionTypeReady)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Reason).To(Equal(reasonWorkloadsProgressing))
		Expect(cond.Message).To(Equal("node: active service cutover pending"))
	})

	It("reports RolloutPaused while a blue-green node Rollout awaits promotion", func() {
		ro := healthyNodeRollout(2)
		ro.Status.Phase = rolloutsv1alpha1.RolloutPhasePaused
		ro.Status.ControllerPause = true
		ro.Status.PauseConditions = []rolloutsv1alpha1.PauseCondition{{
			Reason:    rolloutsv1alpha1.PauseReasonBlueGreenPause,
			StartTime: metav1.Now(),
		}}
		r := newFakeReconciler(append(children(), ro)...)

		bp := newRolloutBindplane()
		poll := r.updateReadyReplicaStatus(context.Background(), bp, migrateComplete)

		Expect(poll).To(BeTrue())
		Expect(bp.Status.Phase).To(Equal(phaseApplyingChanges))
		cond := meta.FindStatusCondition(bp.Status.Conditions, conditionTypeReady)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Status).To(Equal(metav1.ConditionFalse))
		Expect(cond.Reason).To(Equal(reasonRolloutPaused))
		Expect(cond.Message).To(HavePrefix("node: node rollout paused"))
	})

	It("reports Degraded and polls while the node Rollout is Degraded", func() {
		ro := healthyNodeRollout(2)
		ro.Status.Phase = rolloutsv1alpha1.RolloutPhaseDegraded
		ro.Status.Message = "ProgressDeadlineExceeded: rollout exceeded its progress deadline"
		r := newFakeReconciler(append(children(), ro)...)

		bp := newRolloutBindplane()
		poll := r.updateReadyReplicaStatus(context.Background(), bp, migrateComplete)

		Expect(poll).To(BeTrue())
		Expect(bp.Status.Phase).To(Equal(phaseDegraded))
		cond := meta.FindStatusCondition(bp.Status.Conditions, conditionTypeReady)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Reason).To(Equal(reasonWorkloadFailed))
		Expect(cond.Message).To(Equal("node: " + ro.Status.Message))
	})

	It("reports Ready when the node Rollout is Healthy", func() {
		r := newFakeReconciler(append(children(), healthyNodeRollout(2))...)

		bp := newRolloutBindplane()
		poll := r.updateReadyReplicaStatus(context.Background(), bp, migrateComplete)

		Expect(poll).To(BeFalse())
		Expect(bp.Status.Phase).To(Equal(phaseReady))
		Expect(bp.Status.Components.Bindplane.ReadyReplicas).To(Equal(int32(2)))
		Expect(bp.Status.Components.Nats.ReadyReplicas).To(Equal(int32(3)))
		cond := meta.FindStatusCondition(bp.Status.Conditions, conditionTypeReady)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Status).To(Equal(metav1.ConditionTrue))
	})

	It("does not poll a Rollout that is watched", func() {
		for _, phase := range []rolloutsv1alpha1.RolloutPhase{
			rolloutsv1alpha1.RolloutPhaseProgressing,
			rolloutsv1alpha1.RolloutPhasePaused,
			rolloutsv1alpha1.RolloutPhaseDegraded,
		} {
			ro := healthyNodeRollout(2)
			ro.Status.Phase = phase
			r := newFakeReconciler(append(children(), ro)...)
			r.rolloutWatched = true

			bp := newRolloutBindplane()
			poll := r.updateReadyReplicaStatus(context.Background(), bp, migrateComplete)

			Expect(poll).To(BeFalse(), "phase %s", phase)
			Expect(bp.Status.Phase).NotTo(Equal(phaseReady), "phase %s", phase)
		}
	})

	It("does not read a node Deployment when the Rollout is enabled", func() {
		objs := append(children(), rolledOutDeployment("bp-node", 3))
		r := newFakeReconciler(objs...)

		bp := newRolloutBindplane()
		r.updateReadyReplicaStatus(context.Background(), bp, migrateComplete)

		Expect(bp.Status.Phase).To(Equal(phaseApplyingChanges))
		cond := meta.FindStatusCondition(bp.Status.Conditions, conditionTypeReady)
		Expect(cond.Message).To(HavePrefix("node: "))
		Expect(cond.Message).To(ContainSubstring("not found"))
	})
})

// newFakeClient returns a fake client holding objs, with the core, Argo Rollouts and
// Bindplane types registered.
func newFakeClient(objs ...client.Object) client.Client {
	s := runtime.NewScheme()
	Expect(clientgoscheme.AddToScheme(s)).To(Succeed())
	Expect(rolloutsv1alpha1.AddToScheme(s)).To(Succeed())
	Expect(bindplanev1alpha1.AddToScheme(s)).To(Succeed())
	return fake.NewClientBuilder().WithScheme(s).WithObjects(objs...).Build()
}

var _ = Describe("updateReadyReplicaStatus with a Deployment node", func() {
	newBindplane := func() *bindplanev1alpha1.Bindplane {
		bp := newTestBindplane("bp", testNamespaceDefault)
		bp.Generation = 1
		return bp
	}

	// children returns the rolled out non-node workloads for the "bp" CR, without TSDB.
	children := func() []client.Object {
		return []client.Object{
			rolledOutStatefulSet("bp-nats", 3),
			rolledOutDeployment("bp-transform-agent", 1),
			rolledOutDeployment("bp-jobs", 1),
		}
	}

	It("reads workload status through APIReader rather than Client", func() {
		// Client holds the node Deployment as it was before the spec update; APIReader holds
		// it at the new generation, not yet observed by the Deployment controller.
		stale := rolledOutDeployment("bp-node", 3)
		fresh := rolledOutDeployment("bp-node", 3)
		fresh.Generation = 2
		cached := newFakeClient(append(children(), rolledOutStatefulSet("bp-tsdb", 1), stale)...)
		live := newFakeClient(append(children(), rolledOutStatefulSet("bp-tsdb", 1), fresh)...)
		r := &BindplaneReconciler{Client: cached, Scheme: cached.Scheme(), APIReader: live}
		Expect(r.statusReader()).To(BeIdenticalTo(live))

		bp := newBindplane()
		poll := r.updateReadyReplicaStatus(context.Background(), bp, migrateComplete)

		Expect(poll).To(BeFalse(), "a node Deployment is watched, so it is not polled")
		Expect(bp.Status.Phase).To(Equal(phaseApplyingChanges))
		cond := meta.FindStatusCondition(bp.Status.Conditions, conditionTypeReady)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Reason).To(Equal(reasonWorkloadsProgressing))
		Expect(cond.Message).To(Equal("node: waiting for deployment spec update to be observed"))
	})

	It("does not read the TSDB StatefulSet when remote TSDB is enabled", func() {
		c := newFakeClient(append(children(), rolledOutDeployment("bp-node", 3))...)
		r := &BindplaneReconciler{Client: c, Scheme: c.Scheme(), APIReader: c}

		bp := newBindplane()
		bp.Spec.Config.TSDB = &bindplanev1alpha1.TSDBConfig{
			Remote: &bindplanev1alpha1.TSDBRemoteConfig{Enable: true},
		}
		poll := r.updateReadyReplicaStatus(context.Background(), bp, migrateComplete)

		Expect(poll).To(BeFalse())
		Expect(bp.Status.Phase).To(Equal(phaseReady))
		Expect(bp.Status.Components.TSDB.ReadyReplicas).To(BeZero())
		cond := meta.FindStatusCondition(bp.Status.Conditions, conditionTypeReady)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		Expect(cond.Message).NotTo(ContainSubstring("tsdb:"))
	})
})

var _ = Describe("ignoreStatusOnlyUpdates", func() {
	base := func() *bindplanev1alpha1.Bindplane {
		bp := newTestBindplane("bp", testNamespaceDefault)
		bp.Generation = 1
		bp.ResourceVersion = "1"
		bp.Annotations = map[string]string{"a": "b"}
		bp.Finalizers = []string{bindplaneFinalizer}
		return bp
	}
	update := func(mutate func(*bindplanev1alpha1.Bindplane)) bool {
		oldObj := base()
		newObj := base()
		newObj.ResourceVersion = "2"
		mutate(newObj)
		return ignoreStatusOnlyUpdates().Update(event.UpdateEvent{ObjectOld: oldObj, ObjectNew: newObj})
	}

	It("ignores a status-only update", func() {
		Expect(update(func(bp *bindplanev1alpha1.Bindplane) {
			bp.Status.Phase = phaseApplyingChanges
			bp.Status.Conditions = []metav1.Condition{{Type: conditionTypeReady, Message: "changed"}}
		})).To(BeFalse())
	})

	It("passes a spec change", func() {
		Expect(update(func(bp *bindplanev1alpha1.Bindplane) { bp.Generation = 2 })).To(BeTrue())
	})

	It("passes an annotation change", func() {
		Expect(update(func(bp *bindplanev1alpha1.Bindplane) {
			bp.Annotations[annotationPauseReconciliation] = "true"
		})).To(BeTrue())
	})

	It("passes a label change", func() {
		Expect(update(func(bp *bindplanev1alpha1.Bindplane) {
			bp.Labels = map[string]string{"x": "y"}
		})).To(BeTrue())
	})

	It("passes a finalizer change", func() {
		Expect(update(func(bp *bindplanev1alpha1.Bindplane) { bp.Finalizers = nil })).To(BeTrue())
	})

	It("passes a deletion", func() {
		Expect(update(func(bp *bindplanev1alpha1.Bindplane) {
			now := metav1.Now()
			bp.DeletionTimestamp = &now
		})).To(BeTrue())
	})

	It("passes create and delete events", func() {
		p := ignoreStatusOnlyUpdates()
		Expect(p.Create(event.CreateEvent{Object: base()})).To(BeTrue())
		Expect(p.Delete(event.DeleteEvent{Object: base()})).To(BeTrue())
	})
})
