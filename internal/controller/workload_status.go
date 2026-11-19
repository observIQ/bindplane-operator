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
	"fmt"
	"strconv"
	"strings"

	rolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

// conditionTypeReady is the condition that reports whether every workload has finished
// rolling out the spec at the Bindplane CR's current generation.
const conditionTypeReady = "Ready"

// Reasons used on the Ready condition.
const (
	reasonWorkloadsReady       = "WorkloadsReady"
	reasonWorkloadsProgressing = "WorkloadsProgressing"
	reasonWorkloadFailed       = "WorkloadFailed"
	reasonRolloutPaused        = "RolloutPaused"
	reasonMigrationInProgress  = "MigrationInProgress"
	reasonReconcileError       = "ReconcileError"
	reasonInvalid              = "Invalid"
	reasonMigrationFailed      = "MigrationFailed"
)

// Values written to status.phase.
const (
	phaseReady           = "Ready"
	phaseApplyingChanges = "ApplyingChanges"
	phaseDegraded        = "Degraded"
)

// jobsMigrateProgressComponent names the Jobs Migrate Job in Ready condition messages.
const jobsMigrateProgressComponent = "jobs-migrate"

// nodeRolloutPausedMessage is the progress message for a node Rollout awaiting promotion.
const nodeRolloutPausedMessage = "node rollout paused: promote with kubectl argo rollouts promote"

// workloadProgress is the rollout state of one workload. A workload that is neither done,
// failed nor paused is waiting, and message says why.
type workloadProgress struct {
	component string
	ready     int32
	done      bool
	failed    bool
	paused    bool
	message   string
}

// waiting returns a workloadProgress that is still rolling out.
func waiting(component string, ready int32, format string, args ...any) workloadProgress {
	return workloadProgress{component: component, ready: ready, message: fmt.Sprintf(format, args...)}
}

// replicasOrDefault returns *replicas, or 1 when replicas is nil (the Kubernetes default).
func replicasOrDefault(replicas *int32) int32 {
	if replicas == nil {
		return 1
	}
	return *replicas
}

// deploymentProgress reports a Deployment's rollout state using the same checks as
// `kubectl rollout status`. The desired count is the live spec.replicas, which an HPA
// may have written.
func deploymentProgress(component string, dep *appsv1.Deployment) workloadProgress {
	ready := dep.Status.ReadyReplicas
	if dep.Generation > dep.Status.ObservedGeneration {
		return waiting(component, ready, "waiting for deployment spec update to be observed")
	}
	for _, c := range dep.Status.Conditions {
		if c.Type == appsv1.DeploymentProgressing && c.Reason == "ProgressDeadlineExceeded" {
			return workloadProgress{component: component, ready: ready, failed: true,
				message: fmt.Sprintf("deployment %s exceeded its progress deadline", dep.Name)}
		}
	}
	desired := replicasOrDefault(dep.Spec.Replicas)
	updated := dep.Status.UpdatedReplicas
	switch {
	case updated < desired:
		return waiting(component, ready, "%d out of %d new replicas have been updated", updated, desired)
	case dep.Status.Replicas > updated:
		return waiting(component, ready, "%d old replicas are pending termination", dep.Status.Replicas-updated)
	case dep.Status.AvailableReplicas < updated:
		return waiting(component, ready, "%d of %d updated replicas are available", dep.Status.AvailableReplicas, updated)
	}
	return workloadProgress{component: component, ready: ready, done: true}
}

// statefulSetProgress reports a StatefulSet's rollout state: the generation is observed,
// every replica is ready and updated, and currentRevision equals updateRevision.
func statefulSetProgress(component string, sts *appsv1.StatefulSet) workloadProgress {
	ready := sts.Status.ReadyReplicas
	if sts.Status.ObservedGeneration == 0 || sts.Generation > sts.Status.ObservedGeneration {
		return waiting(component, ready, "waiting for statefulset spec update to be observed")
	}
	desired := replicasOrDefault(sts.Spec.Replicas)
	switch {
	case ready < desired:
		return waiting(component, ready, "%d of %d pods are ready", ready, desired)
	case sts.Status.UpdatedReplicas < desired:
		return waiting(component, ready, "%d out of %d pods have been updated", sts.Status.UpdatedReplicas, desired)
	case sts.Status.UpdateRevision != sts.Status.CurrentRevision:
		return waiting(component, ready, "waiting for rolling update to revision %s to complete", sts.Status.UpdateRevision)
	}
	return workloadProgress{component: component, ready: ready, done: true}
}

// rolloutProgress reports an Argo Rollout's state. It is waiting unless the string
// status.observedGeneration parses and equals metadata.generation, and while the controller
// is unpausing. Otherwise it maps status.phase Healthy to done, Degraded to failed and Paused
// to paused.
func rolloutProgress(component string, ro *rolloutsv1alpha1.Rollout) workloadProgress {
	ready := ro.Status.ReadyReplicas
	observed, err := strconv.ParseInt(ro.Status.ObservedGeneration, 10, 64)
	if err != nil || observed != ro.Generation {
		return waiting(component, ready, "waiting for rollout spec update to be observed")
	}
	if ro.Status.ControllerPause && len(ro.Status.PauseConditions) == 0 {
		return waiting(component, ready, "waiting for rollout to unpause")
	}
	switch ro.Status.Phase {
	case rolloutsv1alpha1.RolloutPhaseHealthy:
		return workloadProgress{component: component, ready: ready, done: true}
	case rolloutsv1alpha1.RolloutPhaseDegraded:
		return workloadProgress{component: component, ready: ready, failed: true, message: ro.Status.Message}
	case rolloutsv1alpha1.RolloutPhasePaused:
		return workloadProgress{component: component, ready: ready, paused: true, message: nodeRolloutPausedMessage}
	}
	if ro.Status.Message != "" {
		return waiting(component, ready, "%s", ro.Status.Message)
	}
	return waiting(component, ready, "rollout phase is %q", ro.Status.Phase)
}

// migrateProgress returns a progress entry for the Jobs Migrate Job when the migration gate
// was bypassed and the Job is not complete. ok is false when there is nothing to report.
func migrateProgress(bindplane *bindplanev1alpha1.Bindplane, state migrateState) (progress workloadProgress, ok bool) {
	jobName := getResourceName(bindplane, bindplaneJobsMigrateComponent)
	switch state {
	case migrateFailed:
		return workloadProgress{component: jobsMigrateProgressComponent, failed: true,
			message: fmt.Sprintf("job %s failed", jobName)}, true
	case migrateInProgress:
		return waiting(jobsMigrateProgressComponent, 0, "job %s is not complete", jobName), true
	default:
		return workloadProgress{}, false
	}
}

// setPhaseFromProgress sets status.phase and the Ready condition from the progress of
// every workload. A failed workload wins over a paused one, and a paused one wins over
// one that is still rolling out.
func setPhaseFromProgress(bindplane *bindplanev1alpha1.Bindplane, progress []workloadProgress) {
	var failed, paused, pending []string
	for _, p := range progress {
		entry := fmt.Sprintf("%s: %s", p.component, p.message)
		switch {
		case p.failed:
			failed = append(failed, entry)
		case p.paused:
			paused = append(paused, entry)
		case !p.done:
			pending = append(pending, entry)
		}
	}

	switch {
	case len(failed) > 0:
		bindplane.Status.Phase = phaseDegraded
		setReadyCondition(bindplane, metav1.ConditionFalse, reasonWorkloadFailed, strings.Join(failed, "; "))
	case len(paused) > 0:
		bindplane.Status.Phase = phaseApplyingChanges
		setReadyCondition(bindplane, metav1.ConditionFalse, reasonRolloutPaused, strings.Join(paused, "; "))
	case len(pending) > 0:
		bindplane.Status.Phase = phaseApplyingChanges
		setReadyCondition(bindplane, metav1.ConditionFalse, reasonWorkloadsProgressing, strings.Join(pending, "; "))
	default:
		bindplane.Status.Phase = phaseReady
		setReadyCondition(bindplane, metav1.ConditionTrue, reasonWorkloadsReady, "All workloads are rolled out and ready")
	}
}

// setReadyCondition sets the Ready condition at the CR's current generation and reports
// whether the condition changed.
func setReadyCondition(bindplane *bindplanev1alpha1.Bindplane, status metav1.ConditionStatus, reason, message string) bool {
	return meta.SetStatusCondition(&bindplane.Status.Conditions, metav1.Condition{
		Type:               conditionTypeReady,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: bindplane.Generation,
	})
}

// statusReader returns the reader used for child workload status. APIReader bypasses the
// informer cache, so a child updated earlier in the same reconcile is read at its new
// generation rather than as the cached copy from before the update.
func (r *BindplaneReconciler) statusReader() client.Reader {
	if r.APIReader != nil {
		return r.APIReader
	}
	return r.Client
}

// getDeploymentProgress reads a Deployment and reports its rollout state. A read error,
// including NotFound, is reported as waiting with the error as the message.
func (r *BindplaneReconciler) getDeploymentProgress(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, component string) workloadProgress {
	dep := &appsv1.Deployment{}
	key := types.NamespacedName{Name: getResourceName(bindplane, component), Namespace: bindplane.Namespace}
	if err := r.statusReader().Get(ctx, key, dep); err != nil {
		return waiting(component, 0, "%s", err.Error())
	}
	return deploymentProgress(component, dep)
}

// getStatefulSetProgress reads a StatefulSet and reports its rollout state. A read error,
// including NotFound, is reported as waiting with the error as the message.
func (r *BindplaneReconciler) getStatefulSetProgress(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, component string) workloadProgress {
	sts := &appsv1.StatefulSet{}
	key := types.NamespacedName{Name: getResourceName(bindplane, component), Namespace: bindplane.Namespace}
	if err := r.statusReader().Get(ctx, key, sts); err != nil {
		return waiting(component, 0, "%s", err.Error())
	}
	return statefulSetProgress(component, sts)
}

// getRolloutProgress reads an Argo Rollout and reports its rollout state. A read error,
// including NotFound, is reported as waiting with the error as the message.
func (r *BindplaneReconciler) getRolloutProgress(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, component string) workloadProgress {
	ro := &rolloutsv1alpha1.Rollout{}
	key := types.NamespacedName{Name: getResourceName(bindplane, component), Namespace: bindplane.Namespace}
	if err := r.statusReader().Get(ctx, key, ro); err != nil {
		return waiting(component, 0, "%s", err.Error())
	}
	return rolloutProgress(component, ro)
}
