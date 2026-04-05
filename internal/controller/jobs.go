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

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

const (
	// bindplaneJobsMigrateComponent is the component name for Bindplane Jobs Migrate
	bindplaneJobsMigrateComponent = "migrate"
	// bindplaneJobsComponent is the component name for Bindplane Jobs
	bindplaneJobsComponent = "jobs"
	// bindplaneJobsContainerName is the container name for Bindplane Jobs
	bindplaneJobsContainerName = "server"
	// bindplaneJobsHTTPPort is the HTTP port for Bindplane Jobs
	bindplaneJobsHTTPPort = int32(3001)
	// bindplaneJobsHTTPPortName is the name of the HTTP port for Bindplane Jobs
	bindplaneJobsHTTPPortName = "http"
	// bindplaneJobsMigrateModeValue is the value for BINDPLANE_MODE for migrate jobs
	bindplaneJobsMigrateModeValue = "migrate"
	// bindplaneJobsModeValue is the value for BINDPLANE_MODE for regular jobs
	bindplaneJobsModeValue = "all,-migrate"
	// forceMigrateAnnotation is the annotation key to force a migration run
	forceMigrateAnnotation = "k8s.bindplane.com/force-migrate"
	// migrateJobBackoffLimit is the backoff limit for the migrate Job
	migrateJobBackoffLimit = int32(3)
	// migrateJobTTLSeconds is the TTL for the migrate Job after completion (24 hours)
	migrateJobTTLSeconds = int32(86400)
)

// reconcileBindplaneJobsRegular reconciles the Bindplane Jobs deployment
func (r *BindplaneReconciler) reconcileBindplaneJobsRegular(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
	// Reconcile ServiceAccount
	sa := r.bindplaneJobsServiceAccount(bindplane)
	if err := r.reconcileServiceAccount(ctx, bindplane, sa, log); err != nil {
		return err
	}

	// Reconcile Deployment
	deployment := r.bindplaneJobsDeployment(bindplane)
	if err := r.reconcileDeployment(ctx, bindplane, deployment, log); err != nil {
		return err
	}

	return nil
}

func (r *BindplaneReconciler) bindplaneJobsMigrateServiceAccount(bindplane *bindplanev1alpha1.Bindplane) *corev1.ServiceAccount {
	return newServiceAccount(bindplane, bindplaneJobsMigrateComponent)
}

func (r *BindplaneReconciler) bindplaneJobsServiceAccount(bindplane *bindplanev1alpha1.Bindplane) *corev1.ServiceAccount {
	return newServiceAccount(bindplane, bindplaneJobsComponent)
}

func (r *BindplaneReconciler) bindplaneJobsDeployment(bindplane *bindplanev1alpha1.Bindplane) *appsv1.Deployment {
	// maxSurge=1 allows a new pod to start before the old one is deleted,
	// so two pods may briefly run in parallel during a rollout.
	maxSurge := intstr.FromInt32(1)
	maxUnavailable := intstr.FromInt32(0)
	// Jobs resources: 1000m CPU, 1024Mi memory
	resources := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("1024Mi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1000m"),
			corev1.ResourceMemory: resource.MustParse("1024Mi"),
		},
	}
	if bindplane.Spec.BindplaneJobs != nil && bindplane.Spec.BindplaneJobs.Resources != nil {
		resources = *bindplane.Spec.BindplaneJobs.Resources
	}

	replicas := int32(1)
	labels := getLabels(bindplane, bindplaneJobsComponent)
	selectorLabels := getSelectorLabels(bindplane, bindplaneJobsComponent)
	configVols, configMounts := getConfigTLSVolumesAndMounts(bindplane)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getResourceName(bindplane, bindplaneJobsComponent),
			Namespace: bindplane.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge:       &maxSurge,
					MaxUnavailable: &maxUnavailable,
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Template: mergePodTemplateSpec(
				corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: selectorLabels,
					},
					Spec: corev1.PodSpec{
						Volumes:            configVols,
						ServiceAccountName: getResourceName(bindplane, bindplaneJobsComponent),
						SecurityContext:    newPodSecurityContext(),
						Affinity:           getBindplaneJobsAffinity(bindplane),
						Containers: []corev1.Container{
							{
								Name:         bindplaneJobsContainerName,
								Image:        getBindplaneEEImage(bindplane),
								VolumeMounts: configMounts,
								Ports: []corev1.ContainerPort{
									{
										Name:          bindplaneJobsHTTPPortName,
										ContainerPort: bindplaneJobsHTTPPort,
										Protocol:      corev1.ProtocolTCP,
									},
								},
								Env: combineEnvVars(
									getKubernetesEnvVars(bindplaneJobsContainerName),
									[]corev1.EnvVar{
										{
											Name:  bindplaneModeEnvVar,
											Value: bindplaneJobsModeValue,
										},
									},
									getBindplaneCommonEnvVars(bindplane, bindplaneJobsComponent),
									getNatsClientEnvVars(bindplane, true),
								),
								Resources: resources,
								StartupProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										TCPSocket: &corev1.TCPSocketAction{
											Port: intstr.FromString(bindplaneJobsHTTPPortName),
										},
									},
									InitialDelaySeconds: probeStartupInitialDelaySeconds,
									PeriodSeconds:       probeStartupPeriodSeconds,
									FailureThreshold:    probeStartupFailureThreshold,
									SuccessThreshold:    probeStartupSuccessThreshold,
									TimeoutSeconds:      probeStartupTimeoutSeconds,
								},
								ReadinessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										TCPSocket: &corev1.TCPSocketAction{
											Port: intstr.FromString(bindplaneJobsHTTPPortName),
										},
									},
									PeriodSeconds:    probePeriodSeconds,
									FailureThreshold: probeFailureThreshold,
									SuccessThreshold: probeSuccessThreshold,
									TimeoutSeconds:   probeTimeoutSeconds,
								},
								LivenessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										TCPSocket: &corev1.TCPSocketAction{
											Port: intstr.FromString(bindplaneJobsHTTPPortName),
										},
									},
									PeriodSeconds:    probePeriodSeconds,
									FailureThreshold: probeFailureThreshold,
									SuccessThreshold: probeSuccessThreshold,
									TimeoutSeconds:   probeTimeoutSeconds,
								},
								SecurityContext: newContainerSecurityContext(WithRunAsUser(defaultRunAsUser)),
								ImagePullPolicy: corev1.PullIfNotPresent,
								Lifecycle: &corev1.Lifecycle{
									PreStop: &corev1.LifecycleHandler{
										Exec: &corev1.ExecAction{
											Command: []string{preStopCommand, preStopArgs, preStopSleep},
										},
									},
								},
							},
						},
						TerminationGracePeriodSeconds: new(defaultTerminationGracePeriodSeconds),
					},
				},
				getBindplaneJobsPodTemplate(bindplane),
			),
		},
	}
}

// bindplaneJobsMigrateJob creates a batch/v1 Job for Bindplane database migrations.
func (r *BindplaneReconciler) bindplaneJobsMigrateJob(bindplane *bindplanev1alpha1.Bindplane) *batchv1.Job {
	labels := getLabels(bindplane, bindplaneJobsMigrateComponent)
	selectorLabels := getSelectorLabels(bindplane, bindplaneJobsMigrateComponent)
	configVols, configMounts := getConfigTLSVolumesAndMounts(bindplane)
	backoffLimit := migrateJobBackoffLimit
	ttl := migrateJobTTLSeconds

	migrateResources := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("2048Mi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("2048Mi"),
		},
	}
	if bindplane.Spec.BindplaneJobsMigrate != nil && bindplane.Spec.BindplaneJobsMigrate.Resources != nil {
		migrateResources = *bindplane.Spec.BindplaneJobsMigrate.Resources
	}

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getResourceName(bindplane, bindplaneJobsMigrateComponent),
			Namespace: bindplane.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttl,
			Template: mergePodTemplateSpec(
				corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: selectorLabels},
					Spec: corev1.PodSpec{
						RestartPolicy:      corev1.RestartPolicyOnFailure,
						ServiceAccountName: getResourceName(bindplane, bindplaneJobsMigrateComponent),
						Volumes:            configVols,
						SecurityContext:    newPodSecurityContext(),
						Affinity:           getBindplaneJobsMigrateAffinity(bindplane),
						Containers: []corev1.Container{{
							Name:         bindplaneJobsContainerName,
							Image:        getBindplaneEEImage(bindplane),
							Command:      []string{"/bindplane", "migrate", "-y"},
							VolumeMounts: configMounts,
							Env: combineEnvVars(
								getKubernetesEnvVars(bindplaneJobsContainerName),
								[]corev1.EnvVar{{Name: bindplaneModeEnvVar, Value: bindplaneJobsMigrateModeValue}},
								getBindplaneCommonEnvVars(bindplane, bindplaneJobsMigrateComponent),
							),
							Resources:       migrateResources,
							SecurityContext: newContainerSecurityContext(WithRunAsUser(defaultRunAsUser)),
							ImagePullPolicy: corev1.PullIfNotPresent,
						}},
					},
				},
				getBindplaneJobsMigratePodTemplate(bindplane),
			),
		},
	}
}

// reconcileMigrateJob ensures the migration batch/v1 Job runs to completion before downstream
// workloads (NATS, Jobs, Node) are reconciled. Returns (migrationComplete, error).
func (r *BindplaneReconciler) reconcileMigrateJob(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) (bool, error) {
	jobName := getResourceName(bindplane, bindplaneJobsMigrateComponent)
	ns := bindplane.Namespace

	// Clean up legacy Deployment (operator upgrade path).
	oldDeployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: ns}, oldDeployment); err == nil {
		log.Info("deleting legacy Jobs Migrate Deployment", "name", jobName)
		if delErr := r.Delete(ctx, oldDeployment); delErr != nil && !errors.IsNotFound(delErr) {
			log.Error(delErr, "failed to delete legacy Jobs Migrate Deployment")
		}
	}

	// Reconcile ServiceAccount (always needed).
	sa := r.bindplaneJobsMigrateServiceAccount(bindplane)
	if err := r.reconcileServiceAccount(ctx, bindplane, sa, log); err != nil {
		return false, err
	}

	// Handle force annotation: clear it and reset MigratedImage so the normal
	// image-change flow triggers Job creation on the next reconcile.
	if bindplane.Annotations[forceMigrateAnnotation] == "true" {
		patch := client.MergeFrom(bindplane.DeepCopy())
		delete(bindplane.Annotations, forceMigrateAnnotation)
		if err := r.Patch(ctx, bindplane, patch); err != nil {
			return false, err
		}
		bindplane.Status.MigratedImage = ""
		if err := r.Status().Update(ctx, bindplane); err != nil {
			return false, err
		}
		return false, nil // requeue; next reconcile will detect MigratedImage mismatch
	}

	desiredImage := getBindplaneEEImage(bindplane)

	// Already migrated for this image — skip.
	if bindplane.Status.MigratedImage == desiredImage {
		return true, nil
	}

	// Look up existing Job.
	existingJob := &batchv1.Job{}
	err := r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: ns}, existingJob)
	if err != nil && !errors.IsNotFound(err) {
		return false, err
	}

	if errors.IsNotFound(err) {
		// No Job yet — create it.
		job := r.bindplaneJobsMigrateJob(bindplane)
		if err := controllerutil.SetControllerReference(bindplane, job, r.Scheme); err != nil {
			return false, err
		}
		log.Info("creating Jobs Migrate Job", "name", jobName, "image", desiredImage)
		if err := r.Create(ctx, job); err != nil {
			return false, err
		}
		return false, nil // requeue to check status
	}

	// Job exists — check image.
	if extractJobContainerImage(existingJob) != desiredImage {
		// Stale Job from a previous image version — delete and requeue.
		log.Info("deleting stale Jobs Migrate Job", "name", jobName)
		if err := r.deleteJobWithPods(ctx, existingJob, log); err != nil {
			return false, err
		}
		return false, nil
	}

	if isJobSucceeded(existingJob) {
		bindplane.Status.MigratedImage = desiredImage
		if err := r.Status().Update(ctx, bindplane); err != nil {
			return false, err
		}
		return true, nil
	}

	if isJobFailed(existingJob) {
		setMigrateFailureCondition(bindplane)
		if err := r.Status().Update(ctx, bindplane); err != nil {
			return false, err
		}
		return false, nil
	}

	// Job is still active (running).
	return false, nil
}

// isJobSucceeded returns true when the Job's Complete condition is True.
func isJobSucceeded(job *batchv1.Job) bool {
	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobComplete && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// isJobFailed returns true when the Job's Failed condition is True.
func isJobFailed(job *batchv1.Job) bool {
	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// extractJobContainerImage returns the image of the first container in the Job's pod template, or "".
func extractJobContainerImage(job *batchv1.Job) string {
	if len(job.Spec.Template.Spec.Containers) == 0 {
		return ""
	}
	return job.Spec.Template.Spec.Containers[0].Image
}

// deleteJobWithPods deletes a Job with foreground propagation so pods are removed too.
func (r *BindplaneReconciler) deleteJobWithPods(ctx context.Context, job *batchv1.Job, log logr.Logger) error {
	propagation := metav1.DeletePropagationForeground
	if err := r.Delete(ctx, job, &client.DeleteOptions{PropagationPolicy: &propagation}); err != nil && !errors.IsNotFound(err) {
		log.Error(err, "failed to delete Jobs Migrate Job", "name", job.Name)
		return err
	}
	return nil
}

// setMigrateFailureCondition records a MigrationFailed condition on the Bindplane CR.
func setMigrateFailureCondition(bindplane *bindplanev1alpha1.Bindplane) {
	meta.SetStatusCondition(&bindplane.Status.Conditions, metav1.Condition{
		Type:               "Reconciled",
		Status:             metav1.ConditionFalse,
		Reason:             "MigrationFailed",
		Message:            "Jobs Migrate Job failed; downstream workloads are blocked until migration succeeds",
		ObservedGeneration: bindplane.Generation,
		LastTransitionTime: metav1.Now(),
	})
	bindplane.Status.Phase = "Degraded"
	bindplane.Status.ObservedGeneration = bindplane.Generation
}

// getBindplaneJobsAffinity returns the affinity configuration for Bindplane Jobs pods
// This is a fallback for when user doesn't provide podTemplate - will be overridden by mergePodTemplateSpec
func getBindplaneJobsAffinity(bindplane *bindplanev1alpha1.Bindplane) *corev1.Affinity {
	if bindplane.Spec.BindplaneJobs != nil && bindplane.Spec.BindplaneJobs.PodTemplate != nil {
		return bindplane.Spec.BindplaneJobs.PodTemplate.Spec.Affinity
	}
	return nil
}

// getBindplaneJobsPodTemplate returns the user-provided pod template spec for Bindplane Jobs
func getBindplaneJobsPodTemplate(bindplane *bindplanev1alpha1.Bindplane) *bindplanev1alpha1.PodTemplateSpec {
	if bindplane.Spec.BindplaneJobs != nil {
		return bindplane.Spec.BindplaneJobs.PodTemplate
	}
	return nil
}

// getBindplaneJobsMigrateAffinity returns the affinity configuration for Bindplane Jobs Migrate pods
// This is a fallback for when user doesn't provide podTemplate - will be overridden by mergePodTemplateSpec
func getBindplaneJobsMigrateAffinity(bindplane *bindplanev1alpha1.Bindplane) *corev1.Affinity {
	if bindplane.Spec.BindplaneJobsMigrate != nil && bindplane.Spec.BindplaneJobsMigrate.PodTemplate != nil {
		return bindplane.Spec.BindplaneJobsMigrate.PodTemplate.Spec.Affinity
	}
	return nil
}

// getBindplaneJobsMigratePodTemplate returns the user-provided pod template spec for Bindplane Jobs Migrate
func getBindplaneJobsMigratePodTemplate(bindplane *bindplanev1alpha1.Bindplane) *bindplanev1alpha1.PodTemplateSpec {
	if bindplane.Spec.BindplaneJobsMigrate != nil {
		return bindplane.Spec.BindplaneJobsMigrate.PodTemplate
	}
	return nil
}
