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

package v1alpha1

import (
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BindplaneSpec defines the desired state of Bindplane.
type BindplaneSpec struct {
	// Version specifies the default Bindplane release version used to derive container images
	// for all components. Individual components can override their image via their own image
	// field (e.g. spec.bindplane.image); those overrides take precedence over this value.
	// Changing this value triggers a rolling update of every component that does not have an
	// explicit image override, plus a new database migration Job before downstream workloads
	// are updated.
	// +optional
	// +kubebuilder:default="1.99.1"
	Version string `json:"version,omitempty"`

	// Config contains Bindplane's configuration (license, auth, network, store, eventBus)
	// This config is shared by Node, Jobs, and Jobs Migrate
	Config BindplaneConfigSpec `json:"config"`

	// Bindplane configuration and pod specification
	// +optional
	// +kubebuilder:default={}
	Bindplane BindplaneComponentSpec `json:"bindplane,omitempty"`

	// Bindplane Jobs pod specification
	// +optional
	BindplaneJobs *BindplaneJobsComponentSpec `json:"bindplaneJobs,omitempty"`

	// Bindplane Jobs Migrate pod specification
	// +optional
	BindplaneJobsMigrate *BindplaneJobsMigrateComponentSpec `json:"bindplaneJobsMigrate,omitempty"`

	// Transform Agent pod specification
	// +optional
	// +kubebuilder:default={}
	TransformAgent *TransformAgentComponentSpec `json:"transformAgent,omitempty"`

	// TSDB pod specification
	// +optional
	TSDB *TSDBComponentSpec `json:"tsdb,omitempty"`

	// NATS pod specification
	// +optional
	// +kubebuilder:default={}
	Nats *NatsComponentSpec `json:"nats,omitempty"`

	// OpAMP, when enabled, runs a dedicated Deployment for OpAMP/agent traffic
	// alongside the primary Node deployment. When nil or disabled (the default),
	// the primary Node deployment serves both frontend and OpAMP traffic.
	// +optional
	OpAMP *OpAMPComponentSpec `json:"opamp,omitempty"`
}

// NodeAutoscalingSpec configures horizontal pod autoscaling for Bindplane Node.
// When enabled, the operator creates a HorizontalPodAutoscaler and the
// spec.bindplane.replicas field is ignored — the HPA controls replica count.
//
// All fields are optional. Omitted fields use defaults tuned for Bindplane Node's
// stateful WebSocket (OpAMP) workload.
type NodeAutoscalingSpec struct {
	// Enabled enables the HorizontalPodAutoscaler for Bindplane Node.
	// When false (the default), static replica counts from spec.bindplane.replicas
	// are used and no HPA is created.
	// +optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// MinReplicas is the lower replica bound for the autoscaler. Default: 2.
	// +optional
	// +kubebuilder:default=2
	// +kubebuilder:validation:Minimum=1
	MinReplicas *int32 `json:"minReplicas,omitempty"`

	// MaxReplicas is the upper replica bound for the autoscaler. Default: 10.
	// +optional
	// +kubebuilder:default=10
	// +kubebuilder:validation:Minimum=1
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`

	// Metrics contains the specifications for which metrics to use when calculating
	// the desired replica count. When omitted, defaults to CPU at 50% target utilization.
	// +optional
	Metrics []autoscalingv2.MetricSpec `json:"metrics,omitempty"`

	// Behavior configures the scaling behavior in both Up and Down directions.
	// When omitted, the default scaleDown policy enforces slow scale-down
	// (1 pod per 5 minutes) to prevent agent reconnection storms.
	// +optional
	Behavior *autoscalingv2.HorizontalPodAutoscalerBehavior `json:"behavior,omitempty"`
}

// ServiceAccountSpec defines configuration for an operator-managed ServiceAccount.
type ServiceAccountSpec struct {
	// Annotations are added to the ServiceAccount's metadata.annotations.
	// Use this to attach cloud workload identity annotations, e.g. AWS IRSA
	// (eks.amazonaws.com/role-arn) or GKE Workload Identity (iam.gke.io/gcp-service-account).
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// BindplaneComponentSpec defines the Bindplane component pod specification
type BindplaneComponentSpec struct {
	// Replicas specifies the number of replicas for Bindplane Node deployment
	// +optional
	// +kubebuilder:default=3
	Replicas *int32 `json:"replicas,omitempty"`

	// Resources defines compute resource requests and limits for the Bindplane Node primary container.
	// If podTemplate.spec.containers[server].resources is also set, the podTemplate value takes
	// precedence because it is more specific.
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// PodTemplate defines pod template specification for Bindplane Node
	// +optional
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	PodTemplate *PodTemplateSpec `json:"podTemplate,omitempty"`

	// DisablePodDisruptionBudget disables the operator-managed PodDisruptionBudget for this component.
	// When false (default), the operator creates a PDB with minAvailable: 1.
	// +optional
	// +kubebuilder:default=false
	DisablePodDisruptionBudget bool `json:"disablePodDisruptionBudget,omitempty"`

	// MinReadySeconds is the minimum number of seconds a newly created Node pod must be
	// ready (passing its readiness probe) before it is considered available. During a
	// rolling update the next pod is not replaced until this window elapses. When omitted,
	// the operator defaults this to the pod's termination grace period, giving agents
	// that were connected to the outgoing pod enough time to reconnect to healthy nodes
	// (including the new pod) before another pod is taken out of service.
	// +optional
	// +kubebuilder:validation:Minimum=0
	MinReadySeconds *int32 `json:"minReadySeconds,omitempty"`

	// Strategy defines the rollout strategy for the Bindplane Node Deployment.
	// When omitted, defaults to RollingUpdate with maxSurge=1 and maxUnavailable=0,
	// ensuring a replacement pod is running before the old pod is removed.
	// Mutually exclusive with ArgoRollout.Enabled.
	// +optional
	Strategy *appsv1.DeploymentStrategy `json:"strategy,omitempty"`

	// ArgoRollout, when set with Enabled=true, manages the primary Bindplane component
	// as an Argo Rollouts Rollout (BlueGreen strategy) instead of a standard Deployment.
	// The argoproj.io/v1alpha1 Rollout CRD and the Argo Rollouts controller must be
	// installed in the cluster.
	//
	// When enabled, BindplaneComponentSpec.Strategy is rejected by validation
	// (mutually exclusive — Rollout strategy is BlueGreen-only here).
	//
	// RECOMMENDED: also set spec.opamp.enabled=true. BlueGreen promotions cut over
	// active traffic atomically; routing OpAMP/agent traffic to a dedicated Deployment
	// prevents agent reconnect storms during promotion.
	// +optional
	ArgoRollout *ArgoRolloutSpec `json:"argoRollout,omitempty"`

	// Autoscaling configures optional horizontal pod autoscaling for Bindplane Node.
	// When autoscaling is enabled, spec.bindplane.replicas is ignored and the
	// HorizontalPodAutoscaler controls the replica count.
	// +optional
	Autoscaling *NodeAutoscalingSpec `json:"autoscaling,omitempty"`

	// ExtraEnv is a list of additional environment variables to inject into the
	// primary container of this component. These are prepended BEFORE the
	// operator-managed environment variables, so a duplicate Name set here will
	// be ignored — Kubernetes uses the LAST entry for a given Name and the
	// operator will not let user entries override its own values.
	//
	// This is the supported way to add custom environment variables. Setting
	// env on podTemplate.spec.containers[<name>] is intentionally ignored.
	//
	// Environment variable names starting with BINDPLANE_ are rejected by the
	// validating webhook unless the operator is started with --allow-bindplane-extra-env=true.
	//
	// +optional
	// +listType=map
	// +listMapKey=name
	ExtraEnv []corev1.EnvVar `json:"extraEnv,omitempty"`

	// ExtraVolumes is a list of additional volumes to add to this component's pod.
	// Volume names must be unique and must not collide with operator-managed volume names.
	// Allowed sources: secret, configMap, projected, csi, emptyDir, downwardAPI.
	// hostPath and other sources are rejected by the validating webhook.
	// +optional
	// +listType=map
	// +listMapKey=name
	ExtraVolumes []corev1.Volume `json:"extraVolumes,omitempty"`

	// ExtraVolumeMounts is a list of additional volume mounts for this component's primary container.
	// Each mount's name must reference a volume defined in extraVolumes on the same component.
	// mountPath must be absolute and must not collide with an operator-managed mount path.
	// +optional
	// +listType=map
	// +listMapKey=mountPath
	ExtraVolumeMounts []corev1.VolumeMount `json:"extraVolumeMounts,omitempty"`

	// Image overrides the container image for Bindplane Node. When set, the value is used
	// verbatim as a full OCI reference (e.g. "myregistry.example.com/bindplane-ee:1.99.1" or
	// "ghcr.io/observiq/bindplane-ee@sha256:..."). When empty, the image is derived from spec.version.
	// +optional
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image,omitempty"`

	// ServiceAccount configures the operator-managed ServiceAccount for this component.
	// +optional
	ServiceAccount *ServiceAccountSpec `json:"serviceAccount,omitempty"`
}

// ArgoRolloutSpec configures BlueGreen Argo Rollouts management for the primary
// Bindplane component. Only BlueGreen is supported in this release.
type ArgoRolloutSpec struct {
	// Enabled toggles Argo Rollout management for the primary Bindplane component.
	// +optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// AutoPromotionEnabled controls whether the new ReplicaSet is automatically
	// promoted to active once it becomes available. Defaults to true.
	// +optional
	AutoPromotionEnabled *bool `json:"autoPromotionEnabled,omitempty"`

	// ScaleDownDelaySeconds is how long the previous ReplicaSet remains running
	// after promotion. When omitted, Argo Rollouts applies its own default (30s).
	// +optional
	// +kubebuilder:validation:Minimum=0
	ScaleDownDelaySeconds *int32 `json:"scaleDownDelaySeconds,omitempty"`
}

// OpAMPComponentSpec defines an optional dedicated Bindplane Deployment that
// serves OpAMP/agent traffic. When enabled, the operator provisions a second
// Deployment running BINDPLANE_MODE=node alongside the primary Node deployment.
// Both Deployments share the same Bindplane configuration (license, store, auth,
// event bus). They differ in resources, replicas, autoscaling, PDB, and
// OpAMP-specific tuning environment variables.
//
// Use this when you want to scale agent-handling capacity independently from
// the frontend (UI/API), for example when you have a large fleet of agents but
// modest UI traffic.
type OpAMPComponentSpec struct {
	// Enabled enables the dedicated OpAMP deployment. When false (the default),
	// the primary Node deployment serves both frontend and OpAMP traffic.
	// +optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// Replicas specifies the number of replicas for the OpAMP deployment.
	// Ignored when Autoscaling.Enabled is true.
	// +optional
	// +kubebuilder:default=3
	Replicas *int32 `json:"replicas,omitempty"`

	// Resources defines compute resource requests and limits for the OpAMP
	// primary container. If podTemplate.spec.containers[server].resources is
	// also set, the podTemplate value takes precedence because it is more specific.
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// PodTemplate defines pod template specification for the OpAMP deployment.
	// Merged on top of operator-managed defaults using the same merge rules as
	// other component podTemplates.
	// +optional
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	PodTemplate *PodTemplateSpec `json:"podTemplate,omitempty"`

	// DisablePodDisruptionBudget disables the operator-managed PodDisruptionBudget
	// for the OpAMP deployment. When false (the default), the operator creates
	// a PDB with minAvailable: 1.
	// +optional
	// +kubebuilder:default=false
	DisablePodDisruptionBudget bool `json:"disablePodDisruptionBudget,omitempty"`

	// MinReadySeconds is the minimum number of seconds a newly created OpAMP pod
	// must be ready before it is considered available. When omitted, the operator
	// defaults this to the pod's termination grace period.
	// +optional
	// +kubebuilder:validation:Minimum=0
	MinReadySeconds *int32 `json:"minReadySeconds,omitempty"`

	// Strategy defines the rollout strategy for the OpAMP Deployment. When
	// omitted, defaults to RollingUpdate with maxSurge=1 and maxUnavailable=0.
	// +optional
	Strategy *appsv1.DeploymentStrategy `json:"strategy,omitempty"`

	// Autoscaling configures optional horizontal pod autoscaling for OpAMP.
	// When enabled, spec.bindplane.opamp.replicas is ignored.
	// +optional
	Autoscaling *NodeAutoscalingSpec `json:"autoscaling,omitempty"`

	// MaxSimultaneousConnections sets BINDPLANE_AGENTS_MAX_SIMULTANEOUS_CONNECTIONS
	// for the OpAMP deployment only. When unset, falls back to
	// spec.config.agents.maxSimultaneousConnections which is shared
	// across all node-mode Deployments. Useful when you want OpAMP pods to handle
	// a higher concurrency than the frontend pods.
	// +optional
	// +kubebuilder:validation:Minimum=1
	MaxSimultaneousConnections *int64 `json:"maxSimultaneousConnections,omitempty"`

	// ShutdownGracePeriodTarget sets BINDPLANE_ADVANCED_SERVER_OPAMP_SHUTDOWN_GRACE_PERIOD_TARGET
	// for the OpAMP deployment. This is a 0-1 fraction (e.g. "0.6") of the OpAMP
	// shutdown grace period after which the server stops accepting new OpAMP
	// connections. Only applied when set.
	// +optional
	ShutdownGracePeriodTarget string `json:"shutdownGracePeriodTarget,omitempty"`

	// ExtraEnv is a list of additional environment variables to inject into the
	// primary container of this component. These are prepended BEFORE the
	// operator-managed environment variables, so a duplicate Name set here will
	// be ignored — Kubernetes uses the LAST entry for a given Name and the
	// operator will not let user entries override its own values.
	//
	// This is the supported way to add custom environment variables. Setting
	// env on podTemplate.spec.containers[<name>] is intentionally ignored.
	//
	// Environment variable names starting with BINDPLANE_ are rejected by the
	// validating webhook unless the operator is started with --allow-bindplane-extra-env=true.
	//
	// +optional
	// +listType=map
	// +listMapKey=name
	ExtraEnv []corev1.EnvVar `json:"extraEnv,omitempty"`

	// ExtraVolumes is a list of additional volumes to add to this component's pod.
	// Volume names must be unique and must not collide with operator-managed volume names.
	// Allowed sources: secret, configMap, projected, csi, emptyDir, downwardAPI.
	// hostPath and other sources are rejected by the validating webhook.
	// +optional
	// +listType=map
	// +listMapKey=name
	ExtraVolumes []corev1.Volume `json:"extraVolumes,omitempty"`

	// ExtraVolumeMounts is a list of additional volume mounts for this component's primary container.
	// Each mount's name must reference a volume defined in extraVolumes on the same component.
	// mountPath must be absolute and must not collide with an operator-managed mount path.
	// +optional
	// +listType=map
	// +listMapKey=mountPath
	ExtraVolumeMounts []corev1.VolumeMount `json:"extraVolumeMounts,omitempty"`

	// Image overrides the container image for the OpAMP deployment. When set, the value is used
	// verbatim as a full OCI reference. When empty, the image is derived from spec.version.
	// +optional
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image,omitempty"`

	// ServiceAccount configures the operator-managed ServiceAccount for this component.
	// +optional
	ServiceAccount *ServiceAccountSpec `json:"serviceAccount,omitempty"`
}

// BindplaneJobsComponentSpec defines the Bindplane Jobs component pod specification
type BindplaneJobsComponentSpec struct {
	// Resources defines compute resource requests and limits for the Bindplane Jobs primary container.
	// If podTemplate.spec.containers[server].resources is also set, the podTemplate value takes
	// precedence because it is more specific.
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// PodTemplate defines pod template specification for Bindplane Jobs
	// Note: Jobs are restricted to 1 replica and cannot be scaled
	// +optional
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	PodTemplate *PodTemplateSpec `json:"podTemplate,omitempty"`

	// ExtraEnv is a list of additional environment variables to inject into the
	// primary container of this component. These are prepended BEFORE the
	// operator-managed environment variables, so a duplicate Name set here will
	// be ignored — Kubernetes uses the LAST entry for a given Name and the
	// operator will not let user entries override its own values.
	//
	// This is the supported way to add custom environment variables. Setting
	// env on podTemplate.spec.containers[<name>] is intentionally ignored.
	//
	// Environment variable names starting with BINDPLANE_ are rejected by the
	// validating webhook unless the operator is started with --allow-bindplane-extra-env=true.
	//
	// +optional
	// +listType=map
	// +listMapKey=name
	ExtraEnv []corev1.EnvVar `json:"extraEnv,omitempty"`

	// ExtraVolumes is a list of additional volumes to add to this component's pod.
	// Volume names must be unique and must not collide with operator-managed volume names.
	// Allowed sources: secret, configMap, projected, csi, emptyDir, downwardAPI.
	// hostPath and other sources are rejected by the validating webhook.
	// +optional
	// +listType=map
	// +listMapKey=name
	ExtraVolumes []corev1.Volume `json:"extraVolumes,omitempty"`

	// ExtraVolumeMounts is a list of additional volume mounts for this component's primary container.
	// Each mount's name must reference a volume defined in extraVolumes on the same component.
	// mountPath must be absolute and must not collide with an operator-managed mount path.
	// +optional
	// +listType=map
	// +listMapKey=mountPath
	ExtraVolumeMounts []corev1.VolumeMount `json:"extraVolumeMounts,omitempty"`

	// Image overrides the container image for Bindplane Jobs. When set, the value is used
	// verbatim as a full OCI reference. When empty, the image is derived from spec.version.
	// +optional
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image,omitempty"`

	// ServiceAccount configures the operator-managed ServiceAccount for this component.
	// +optional
	ServiceAccount *ServiceAccountSpec `json:"serviceAccount,omitempty"`
}

// BindplaneJobsMigrateComponentSpec defines the Bindplane Jobs Migrate component pod specification.
// Jobs Migrate runs as a Kubernetes batch/v1 Job that performs database migrations at install time
// and whenever the Bindplane image version changes.
type BindplaneJobsMigrateComponentSpec struct {
	// Resources defines compute resource requests and limits for the Bindplane Jobs Migrate primary container.
	// If podTemplate.spec.containers[server].resources is also set, the podTemplate value takes
	// precedence because it is more specific.
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// PodTemplate defines pod template specification for the Bindplane Jobs Migrate batch/v1 Job
	// +optional
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	PodTemplate *PodTemplateSpec `json:"podTemplate,omitempty"`

	// ExtraEnv is a list of additional environment variables to inject into the
	// primary container of this component. These are prepended BEFORE the
	// operator-managed environment variables, so a duplicate Name set here will
	// be ignored — Kubernetes uses the LAST entry for a given Name and the
	// operator will not let user entries override its own values.
	//
	// This is the supported way to add custom environment variables. Setting
	// env on podTemplate.spec.containers[<name>] is intentionally ignored.
	//
	// Environment variable names starting with BINDPLANE_ are rejected by the
	// validating webhook unless the operator is started with --allow-bindplane-extra-env=true.
	//
	// +optional
	// +listType=map
	// +listMapKey=name
	ExtraEnv []corev1.EnvVar `json:"extraEnv,omitempty"`

	// ExtraVolumes is a list of additional volumes to add to this component's pod.
	// Volume names must be unique and must not collide with operator-managed volume names.
	// Allowed sources: secret, configMap, projected, csi, emptyDir, downwardAPI.
	// hostPath and other sources are rejected by the validating webhook.
	// +optional
	// +listType=map
	// +listMapKey=name
	ExtraVolumes []corev1.Volume `json:"extraVolumes,omitempty"`

	// ExtraVolumeMounts is a list of additional volume mounts for this component's primary container.
	// Each mount's name must reference a volume defined in extraVolumes on the same component.
	// mountPath must be absolute and must not collide with an operator-managed mount path.
	// +optional
	// +listType=map
	// +listMapKey=mountPath
	ExtraVolumeMounts []corev1.VolumeMount `json:"extraVolumeMounts,omitempty"`

	// Image overrides the container image for Bindplane Jobs Migrate. When set, the value is used
	// verbatim as a full OCI reference. When empty, the image is derived from spec.version.
	// Setting this decouples jobs-migrate from spec.version — ensure the image is compatible with
	// the bindplane-ee image used by node, jobs, and nats; the operator does not enforce this.
	// +optional
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image,omitempty"`

	// ServiceAccount configures the operator-managed ServiceAccount for this component.
	// +optional
	ServiceAccount *ServiceAccountSpec `json:"serviceAccount,omitempty"`
}

// BindplaneConfigSpec defines Bindplane's configuration
// +kubebuilder:validation:XValidation:rule="has(self.license) != has(self.licenseSecretRef)",message="exactly one of license or licenseSecretRef must be set"
type BindplaneConfigSpec struct {
	// License is the Bindplane license key
	// +optional
	// +kubebuilder:validation:MinLength=1
	License string `json:"license,omitempty"`

	// LicenseSecretRef references a Kubernetes Secret containing the Bindplane license key.
	// Takes precedence over License if both are set.
	// +optional
	LicenseSecretRef *corev1.SecretKeySelector `json:"licenseSecretRef,omitempty"`

	// Auth configuration for Bindplane
	// +optional
	Auth *AuthConfig `json:"auth,omitempty"`

	// Network configuration for Bindplane
	// +optional
	Network *NetworkConfig `json:"network,omitempty"`

	// Store configuration for Bindplane
	Store StoreConfig `json:"store"`

	// Tracing configuration for Bindplane. When omitted or type empty, tracing is disabled.
	// +optional
	Tracing *TracingConfig `json:"tracing,omitempty"`

	// Metrics configuration for Bindplane. When omitted, defaults to prometheus type with interval 60s and endpoint /metrics.
	// +optional
	Metrics *MetricsConfig `json:"metrics,omitempty"`

	// MaxConcurrency is the maximum number of concurrent OpAMP operations.
	// Generally set to the same value as spec.config.agents.maxSimultaneousConnections.
	// Do not modify unless directed by Bindplane support.
	// +optional
	MaxConcurrency int `json:"maxConcurrency,omitempty"`

	// AuditTrail configures audit trail retention. When omitted, retentionDays defaults to 365.
	// +optional
	AuditTrail *AuditTrailConfig `json:"auditTrail,omitempty"`

	// TSDB configures TLS and remote settings for Bindplane's TSDB integration.
	// +optional
	TSDB *TSDBConfig `json:"tsdb,omitempty"`

	// Nats configures TLS for the NATS event bus (client and server). Cert-manager only.
	// +optional
	Nats *NatsConfig `json:"nats,omitempty"`

	// EventBus configures the event bus (NATS) integration, including health checks.
	// +optional
	EventBus *EventBusConfig `json:"eventBus,omitempty"`

	// Status configures the Bindplane status check endpoints.
	// +optional
	Status *StatusConfig `json:"status,omitempty"`

	// Logging configures Bindplane log behavior.
	// +optional
	Logging *LoggingConfig `json:"logging,omitempty"`

	// Agents configures Bindplane agent connection, heartbeat, rebalance, and authentication options.
	// +optional
	Agents *AgentsConfig `json:"agents,omitempty"`

	// AgentVersions configures agent version sync behavior.
	// +optional
	AgentVersions *AgentVersionsConfig `json:"agentVersions,omitempty"`
}

// StatusConfig configures the Bindplane status check endpoints.
type StatusConfig struct {
	// Enabled controls whether the status check endpoints are enabled.
	// Defaults to true.
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// Keys are UUIDs used to authenticate requests to the status check endpoints.
	// Supports multiple keys to allow rotation. Optional: when status is enabled and no
	// keys (or keysSecretRef) are supplied, the operator generates and manages a key
	// automatically in a Kubernetes Secret.
	// +optional
	Keys []string `json:"keys,omitempty"`

	// KeysSecretRef references a Kubernetes Secret containing status check keys.
	// The secret value should be comma-delimited UUIDs to support rotation.
	// Takes precedence over Keys if both are set.
	// +optional
	KeysSecretRef *corev1.SecretKeySelector `json:"keysSecretRef,omitempty"`
}

// AgentsConfig configures how Bindplane communicates with agents.
type AgentsConfig struct {
	// Auth configures authentication for agent connections.
	// +optional
	Auth *AgentsAuthConfig `json:"auth,omitempty"`

	// HeartbeatInterval is the interval on which to perform a heartbeat over agent connections (e.g. "30s").
	// Defaults to 30s.
	// +optional
	// +kubebuilder:default="30s"
	HeartbeatInterval string `json:"heartbeatInterval,omitempty"`

	// HeartbeatTTL is the amount of time between agent-initiated heartbeat messages before an agent
	// connection expires (e.g. "1m"). Must be greater than HeartbeatInterval. Defaults to 1m.
	// +optional
	// +kubebuilder:default="1m"
	HeartbeatTTL string `json:"heartbeatTTL,omitempty"`

	// HeartbeatExpiryInterval is the interval between reaping expired agents (e.g. "30s").
	// Defaults to 30s.
	// +optional
	// +kubebuilder:default="30s"
	HeartbeatExpiryInterval string `json:"heartbeatExpiryInterval,omitempty"`

	// RebalanceInterval is the interval between rebalancing agents (e.g. "1h").
	// Defaults to 1h.
	// +optional
	// +kubebuilder:default="1h"
	RebalanceInterval string `json:"rebalanceInterval,omitempty"`

	// RebalancePercentage is the percentage of agents to rebalance (0–100).
	// 0 disables percentage-based rebalancing. Defaults to 0 (disabled).
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	RebalancePercentage *int `json:"rebalancePercentage,omitempty"`

	// RebalanceJitter is the maximum percentage jitter to add to the rebalance interval (0–100).
	// Defaults to 0 (no jitter).
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	RebalanceJitter *int `json:"rebalanceJitter,omitempty"`

	// MaxSimultaneousConnections is the maximum number of goroutines that will service
	// OpAMP connections concurrently. Generally set to the same value as
	// spec.config.maxConcurrency. Do not modify unless directed by Bindplane support.
	// +optional
	// +kubebuilder:default=10
	MaxSimultaneousConnections int `json:"maxSimultaneousConnections,omitempty"`
}

// AgentVersionsConfig configures how Bindplane syncs agent versions.
type AgentVersionsConfig struct {
	// SyncInterval is the interval at which to sync agent versions (e.g. "2h").
	// Must be at least 1h. Defaults to 1h.
	// +optional
	// +kubebuilder:default="1h"
	SyncInterval string `json:"syncInterval,omitempty"`
}

// AgentsAuthConfig configures authentication for agent connections.
type AgentsAuthConfig struct {
	// Type specifies the authentication method(s) for agent connections.
	// Can be a single method or a comma-separated list (e.g. "oauth,secretKey").
	// Valid values: secretKey, oauth. Defaults to secretKey.
	// +optional
	// +kubebuilder:default=secretKey
	Type string `json:"type,omitempty"`

	// SecretKey configures the secret key authentication method.
	// +optional
	SecretKey *AgentsAuthSecretKeyConfig `json:"secretKey,omitempty"`
}

// AgentsAuthSecretKeyConfig configures secret key authentication for agent connections.
type AgentsAuthSecretKeyConfig struct {
	// Headers is the list of HTTP headers to read the secret key from.
	// Defaults to ["X-Bindplane-Authorization", "Authorization"].
	// +optional
	Headers []string `json:"headers,omitempty"`
}

// AuditTrailConfig defines audit trail configuration
type AuditTrailConfig struct {
	// RetentionDays is the number of days to retain audit trail events.
	// +optional
	// +kubebuilder:default=365
	RetentionDays int `json:"retentionDays,omitempty"`
}

// EventBusHealthConfig configures the Bindplane event bus health check.
// The health check sends an event over NATS and waits for responses from other pods.
// Health check failures affect only the status page in the Bindplane web interface;
// they do not cause pod shutdown or failure.
type EventBusHealthConfig struct {
	// RequiredHosts is the minimum number of pods that must respond to the health check
	// event for the event bus to be considered healthy. When omitted, defaults to
	// floor(total / 2) + 1, where total is the sum of node, NATS, and jobs replicas.
	// Jobs Migrate is a batch/v1 Job (not a long-running pod) and is excluded from this total.
	// +optional
	// +kubebuilder:validation:Minimum=1
	RequiredHosts *int32 `json:"requiredHosts,omitempty"`

	// Interval is how often the event bus health check is performed (e.g. 15s, 1m).
	// When omitted, the Bindplane server default is used.
	// +optional
	Interval string `json:"interval,omitempty"`
}

// EventBusConfig configures the Bindplane event bus (NATS) integration.
type EventBusConfig struct {
	// Health configures the event bus health check endpoints.
	// +optional
	Health *EventBusHealthConfig `json:"health,omitempty"`
}

// TSDBConfig configures Bindplane's TSDB component (default implementation: Prometheus).
type TSDBConfig struct {
	// Remote configures Bindplane to use an externally managed TSDB-compatible backend
	// (for example, Prometheus, Mimir, or VictoriaMetrics) instead of the operator-managed TSDB StatefulSet.
	// +optional
	Remote *TSDBRemoteConfig `json:"remote,omitempty"`

	// TLS configures TLS for TSDB remote write.
	// +optional
	TLS *TSDBTLSConfig `json:"tls,omitempty"`
}

// TSDBRemoteConfig defines how Bindplane connects to an externally managed TSDB-compatible backend.
// +kubebuilder:validation:XValidation:rule="self.enable || (!has(self.host) && !has(self.queryPathPrefix) && !has(self.remoteWrite) && !has(self.port))",message="host, port, queryPathPrefix, and remoteWrite must be unset when enable is false"
// +kubebuilder:validation:XValidation:rule="!self.enable || has(self.host)",message="host is required when enable is true"
// +kubebuilder:validation:XValidation:rule="!self.enable || has(self.port)",message="port is required when enable is true"
type TSDBRemoteConfig struct {
	// Enable controls whether Bindplane should connect to an external TSDB-compatible backend.
	// When false, all other fields in this object must be omitted.
	// +optional
	Enable bool `json:"enable,omitempty"`

	// Host is the hostname or IP of the external TSDB-compatible backend.
	// Required when enable is true.
	// +optional
	Host string `json:"host,omitempty"`

	// Port is the TCP port of the external TSDB-compatible backend.
	// Required when enable is true.
	// +optional
	// +kubebuilder:default=9090
	Port int32 `json:"port,omitempty"`

	// QueryPathPrefix is an optional prefix path for PromQL APIs (for example, /prometheus).
	// +optional
	QueryPathPrefix string `json:"queryPathPrefix,omitempty"`

	// RemoteWrite optionally overrides where Bindplane sends TSDB remote write traffic.
	// +optional
	RemoteWrite *TSDBRemoteWriteConfig `json:"remoteWrite,omitempty"`
}

// TSDBRemoteWriteConfig defines optional remote write endpoint overrides.
// +kubebuilder:validation:XValidation:rule="(has(self.host) && has(self.port)) || (!has(self.host) && !has(self.port))",message="host and port must be set together"
type TSDBRemoteWriteConfig struct {
	// Host is the remote write hostname or IP. Must be set together with port.
	// +optional
	Host string `json:"host,omitempty"`

	// Port is the remote write TCP port. Must be set together with host.
	// +optional
	Port int32 `json:"port,omitempty"`

	// Endpoint is the remote write HTTP path.
	// +optional
	// +kubebuilder:default="/api/v1/write"
	Endpoint string `json:"endpoint,omitempty"`
}

// TSDBTLSConfig defines TLS for TSDB remote write.
// Exactly one of secretName (user-defined Secret) or certManager (cert-manager Issuer/ClusterIssuer) should be set.
type TSDBTLSConfig struct {
	// SecretName is the name of the Secret containing the TLS certificate, key, and optionally CA (user-defined TLS).
	// Omit when using certManager.
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// CertKey is the key in the Secret for the TLS certificate.
	// +optional
	CertKey string `json:"certKey,omitempty"`

	// KeyKey is the key in the Secret for the TLS private key.
	// +optional
	KeyKey string `json:"keyKey,omitempty"`

	// CAKey is the key in the Secret for the CA certificate.
	// +optional
	CAKey string `json:"caKey,omitempty"`

	// CertManager references a cert-manager Issuer or ClusterIssuer to issue server and client certs (mTLS).
	// Mutually exclusive with secretName.
	// +optional
	CertManager *CertManagerTLSIssuerRef `json:"certManager,omitempty"`

	// SkipVerify disables TLS certificate verification for the TSDB remote write client. Only set for testing.
	// +optional
	SkipVerify bool `json:"skipVerify,omitempty"`
}

// NatsConfig configures the NATS event bus (client and server use the same TLS config).
type NatsConfig struct {
	// TLS configures mutual TLS for NATS via cert-manager. When set, a single certificate is used for client, cluster, and HTTP ports.
	// +optional
	TLS *NatsTLSConfig `json:"tls,omitempty"`
}

// NatsTLSConfig defines TLS for NATS. Only cert-manager is supported; no secretName.
type NatsTLSConfig struct {
	// CertManager references a cert-manager Issuer or ClusterIssuer to issue the NATS certificate (used for client, cluster, and HTTP).
	// +optional
	CertManager *CertManagerTLSIssuerRef `json:"certManager,omitempty"`

	// SkipVerify disables TLS certificate verification for NATS connections.
	// Not recommended for production use.
	// +optional
	SkipVerify bool `json:"skipVerify,omitempty"`
}

// TransformAgentTLSConfig defines TLS for the Transform Agent. Only cert-manager is supported; no secretName.
type TransformAgentTLSConfig struct {
	// CertManager references a cert-manager Issuer or ClusterIssuer to issue the Transform Agent certificate
	// used by both the Transform Agent server and Bindplane clients.
	// +optional
	CertManager *CertManagerTLSIssuerRef `json:"certManager,omitempty"`
}

// CertManagerTLSIssuerRef references a cert-manager Issuer or ClusterIssuer.
// See https://cert-manager.io/docs/concepts/issuer/
type CertManagerTLSIssuerRef struct {
	// Name is the name of the Issuer or ClusterIssuer resource.
	Name string `json:"name"`

	// Kind is the type of issuer. Either "Issuer" (namespaced) or "ClusterIssuer" (cluster-scoped).
	// +optional
	// +kubebuilder:validation:Enum=Issuer;ClusterIssuer
	// +kubebuilder:default=Issuer
	Kind string `json:"kind,omitempty"`

	// Group is the API group of the issuer. Defaults to cert-manager.io.
	// +optional
	// +kubebuilder:default=cert-manager.io
	Group string `json:"group,omitempty"`
}

// TracingConfig defines tracing configuration
type TracingConfig struct {
	// Type specifies the tracing type. One of: otlp, google. When empty, tracing is disabled.
	// +optional
	// +kubebuilder:validation:Enum=otlp;google
	Type string `json:"type,omitempty"`

	// OTLP configures OTLP tracing when Type is otlp.
	// +optional
	OTLP *TracingOTLPConfig `json:"otlp,omitempty"`

	// SamplingRate is the ratio between 0 and 1 of traces to keep. Omit or 0 to disable sampling.
	// +optional
	SamplingRate string `json:"samplingRate,omitempty"`
}

// TracingOTLPConfig defines OTLP tracing configuration
type TracingOTLPConfig struct {
	// Endpoint is the OTLP endpoint to send traces to (e.g. http://localhost:4317).
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// Insecure disables TLS verification for the OTLP connection.
	// +optional
	Insecure bool `json:"insecure,omitempty"`
}

// MetricsConfig defines metrics configuration
type MetricsConfig struct {
	// Type specifies the metrics type. One of: otlp, prometheus.
	// +optional
	// +kubebuilder:validation:Enum=otlp;prometheus
	// +kubebuilder:default=prometheus
	Type string `json:"type,omitempty"`

	// Interval is the interval at which to export metrics (e.g. 60s). Used when Type is otlp.
	// +optional
	// +kubebuilder:default="60s"
	Interval string `json:"interval,omitempty"`

	// Prometheus configures Prometheus metrics when Type is prometheus.
	// +optional
	Prometheus *MetricsPrometheusConfig `json:"prometheus,omitempty"`

	// OTLP configures OTLP metrics when Type is otlp.
	// +optional
	OTLP *MetricsOTLPConfig `json:"otlp,omitempty"`
}

// MetricsPrometheusConfig defines Prometheus metrics configuration
type MetricsPrometheusConfig struct {
	// Endpoint is the HTTP path to serve metrics on (e.g. /metrics).
	// +optional
	// +kubebuilder:default="/metrics"
	Endpoint string `json:"endpoint,omitempty"`

	// Username is the basic auth username for the metrics endpoint, if any.
	// +optional
	Username string `json:"username,omitempty"`

	// Password is the basic auth password for the metrics endpoint.
	// +optional
	Password string `json:"password,omitempty"`

	// PasswordSecretRef references a Kubernetes Secret containing the metrics endpoint password.
	// Takes precedence over Password if both are set.
	// +optional
	PasswordSecretRef *corev1.SecretKeySelector `json:"passwordSecretRef,omitempty"`
}

// LoggingConfig defines user-configurable logging options.
// The logging output destination is always stdout and is not user-configurable.
type LoggingConfig struct {
	// Level specifies the log level. One of: debug, info, warn, error.
	// +optional
	// +kubebuilder:validation:Enum=debug;info;warn;error
	// +kubebuilder:default=info
	Level string `json:"level,omitempty"`
}

// MetricsOTLPConfig defines OTLP metrics configuration
type MetricsOTLPConfig struct {
	// Endpoint is the gRPC endpoint to send metrics to (e.g. localhost:4317).
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// Insecure disables TLS verification for the OTLP connection.
	// +optional
	Insecure bool `json:"insecure,omitempty"`
}

// TransformAgentComponentSpec defines the Transform Agent component pod specification
type TransformAgentComponentSpec struct {
	// Replicas specifies the number of replicas for Transform Agent deployment
	// +optional
	// +kubebuilder:default=2
	Replicas *int32 `json:"replicas,omitempty"`

	// Resources defines compute resource requests and limits for the Transform Agent primary container.
	// If podTemplate.spec.containers[transform-agent].resources is also set, the podTemplate value takes
	// precedence because it is more specific.
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// PodTemplate defines pod template specification for Transform Agent
	// +optional
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	PodTemplate *PodTemplateSpec `json:"podTemplate,omitempty"`

	// TLS configures mutual TLS for the Transform Agent via cert-manager. When set, a single certificate
	// is used for the Transform Agent server and Bindplane clients.
	// +optional
	TLS *TransformAgentTLSConfig `json:"tls,omitempty"`

	// DisablePodDisruptionBudget disables the operator-managed PodDisruptionBudget for this component.
	// When false (default), the operator creates a PDB with minAvailable: 1.
	// +optional
	DisablePodDisruptionBudget bool `json:"disablePodDisruptionBudget,omitempty"`

	// ExtraEnv is a list of additional environment variables to inject into the
	// primary container of this component. These are prepended BEFORE the
	// operator-managed environment variables, so a duplicate Name set here will
	// be ignored — Kubernetes uses the LAST entry for a given Name and the
	// operator will not let user entries override its own values.
	//
	// This is the supported way to add custom environment variables. Setting
	// env on podTemplate.spec.containers[<name>] is intentionally ignored.
	//
	// Environment variable names starting with BINDPLANE_ are rejected by the
	// validating webhook unless the operator is started with --allow-bindplane-extra-env=true.
	//
	// +optional
	// +listType=map
	// +listMapKey=name
	ExtraEnv []corev1.EnvVar `json:"extraEnv,omitempty"`

	// ExtraVolumes is a list of additional volumes to add to this component's pod.
	// Volume names must be unique and must not collide with operator-managed volume names.
	// Allowed sources: secret, configMap, projected, csi, emptyDir, downwardAPI.
	// hostPath and other sources are rejected by the validating webhook.
	// +optional
	// +listType=map
	// +listMapKey=name
	ExtraVolumes []corev1.Volume `json:"extraVolumes,omitempty"`

	// ExtraVolumeMounts is a list of additional volume mounts for this component's primary container.
	// Each mount's name must reference a volume defined in extraVolumes on the same component.
	// mountPath must be absolute and must not collide with an operator-managed mount path.
	// +optional
	// +listType=map
	// +listMapKey=mountPath
	ExtraVolumeMounts []corev1.VolumeMount `json:"extraVolumeMounts,omitempty"`

	// Image overrides the container image for Transform Agent. When set, the value is used
	// verbatim as a full OCI reference. When empty, the image is derived from spec.version
	// using the ghcr.io/observiq/bindplane-transform-agent registry.
	// +optional
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image,omitempty"`

	// ServiceAccount configures the operator-managed ServiceAccount for this component.
	// +optional
	ServiceAccount *ServiceAccountSpec `json:"serviceAccount,omitempty"`
}

// TSDBComponentSpec defines the TSDB component pod specification.
// By default, this deploys a Prometheus StatefulSet managed by the operator.
type TSDBComponentSpec struct {
	// Resources defines compute resource requests and limits for the TSDB primary container.
	// If podTemplate.spec.containers[tsdb].resources is also set, the podTemplate value takes
	// precedence because it is more specific.
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// PodTemplate defines pod template specification for the TSDB component
	// +optional
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	PodTemplate *PodTemplateSpec `json:"podTemplate,omitempty"`

	// Storage defines the persistent storage configuration for the TSDB component
	// +optional
	Storage *StorageSpec `json:"storage,omitempty"`

	// TLS configures TLS for the TSDB server (StatefulSet). Use either secretName (user-defined Secret)
	// or certManager (cert-manager Issuer/ClusterIssuer), not both. When set, the TSDB serves remote write over TLS.
	// +optional
	TLS *TSDBTLSConfig `json:"tls,omitempty"`

	// ExtraEnv is a list of additional environment variables to inject into the
	// primary container of this component. These are prepended BEFORE the
	// operator-managed environment variables, so a duplicate Name set here will
	// be ignored — Kubernetes uses the LAST entry for a given Name and the
	// operator will not let user entries override its own values.
	//
	// This is the supported way to add custom environment variables. Setting
	// env on podTemplate.spec.containers[<name>] is intentionally ignored.
	//
	// Environment variable names starting with BINDPLANE_ are rejected by the
	// validating webhook unless the operator is started with --allow-bindplane-extra-env=true.
	//
	// +optional
	// +listType=map
	// +listMapKey=name
	ExtraEnv []corev1.EnvVar `json:"extraEnv,omitempty"`

	// ExtraVolumes is a list of additional volumes to add to this component's pod.
	// Volume names must be unique and must not collide with operator-managed volume names.
	// Allowed sources: secret, configMap, projected, csi, emptyDir, downwardAPI.
	// hostPath and other sources are rejected by the validating webhook.
	// +optional
	// +listType=map
	// +listMapKey=name
	ExtraVolumes []corev1.Volume `json:"extraVolumes,omitempty"`

	// ExtraVolumeMounts is a list of additional volume mounts for this component's primary container.
	// Each mount's name must reference a volume defined in extraVolumes on the same component.
	// mountPath must be absolute and must not collide with an operator-managed mount path.
	// +optional
	// +listType=map
	// +listMapKey=mountPath
	ExtraVolumeMounts []corev1.VolumeMount `json:"extraVolumeMounts,omitempty"`

	// Image overrides the container image for TSDB (Prometheus). When set, the value is used
	// verbatim as a full OCI reference. When empty, the image is derived from spec.version
	// using the ghcr.io/observiq/bindplane-prometheus registry.
	// +optional
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image,omitempty"`

	// ServiceAccount configures the operator-managed ServiceAccount for this component.
	// +optional
	ServiceAccount *ServiceAccountSpec `json:"serviceAccount,omitempty"`
}

// NatsComponentSpec defines the NATS component pod specification
type NatsComponentSpec struct {
	// Replicas specifies the number of replicas for NATS StatefulSet
	// +optional
	// +kubebuilder:default=2
	Replicas *int32 `json:"replicas,omitempty"`

	// Resources defines compute resource requests and limits for the NATS primary container.
	// If podTemplate.spec.containers[server].resources is also set, the podTemplate value takes
	// precedence because it is more specific.
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// PodTemplate defines pod template specification for NATS
	// +optional
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	PodTemplate *PodTemplateSpec `json:"podTemplate,omitempty"`

	// DisablePodDisruptionBudget disables the operator-managed PodDisruptionBudget for this component.
	// When false (default), the operator creates a PDB with minAvailable: 1.
	// +optional
	DisablePodDisruptionBudget bool `json:"disablePodDisruptionBudget,omitempty"`

	// ExtraEnv is a list of additional environment variables to inject into the
	// primary container of this component. These are prepended BEFORE the
	// operator-managed environment variables, so a duplicate Name set here will
	// be ignored — Kubernetes uses the LAST entry for a given Name and the
	// operator will not let user entries override its own values.
	//
	// This is the supported way to add custom environment variables. Setting
	// env on podTemplate.spec.containers[<name>] is intentionally ignored.
	//
	// Environment variable names starting with BINDPLANE_ are rejected by the
	// validating webhook unless the operator is started with --allow-bindplane-extra-env=true.
	//
	// +optional
	// +listType=map
	// +listMapKey=name
	ExtraEnv []corev1.EnvVar `json:"extraEnv,omitempty"`

	// ExtraVolumes is a list of additional volumes to add to this component's pod.
	// Volume names must be unique and must not collide with operator-managed volume names.
	// Allowed sources: secret, configMap, projected, csi, emptyDir, downwardAPI.
	// hostPath and other sources are rejected by the validating webhook.
	// +optional
	// +listType=map
	// +listMapKey=name
	ExtraVolumes []corev1.Volume `json:"extraVolumes,omitempty"`

	// ExtraVolumeMounts is a list of additional volume mounts for this component's primary container.
	// Each mount's name must reference a volume defined in extraVolumes on the same component.
	// mountPath must be absolute and must not collide with an operator-managed mount path.
	// +optional
	// +listType=map
	// +listMapKey=mountPath
	ExtraVolumeMounts []corev1.VolumeMount `json:"extraVolumeMounts,omitempty"`

	// Image overrides the container image for NATS. When set, the value is used
	// verbatim as a full OCI reference. When empty, the image is derived from spec.version.
	// +optional
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image,omitempty"`

	// ServiceAccount configures the operator-managed ServiceAccount for this component.
	// +optional
	ServiceAccount *ServiceAccountSpec `json:"serviceAccount,omitempty"`
}

// StorageSpec defines persistent storage configuration
type StorageSpec struct {
	// VolumeClaimTemplate defines the template for creating PersistentVolumeClaims
	// This follows the same structure as StatefulSet volumeClaimTemplates
	VolumeClaimTemplate *VolumeClaimTemplate `json:"volumeClaimTemplate,omitempty"`
}

// VolumeClaimTemplate defines a template for creating PersistentVolumeClaims
type VolumeClaimTemplate struct {
	// Metadata for the PersistentVolumeClaim
	// +optional
	Metadata *metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the PersistentVolumeClaim specification
	Spec corev1.PersistentVolumeClaimSpec `json:"spec"`
}

// PodTemplateSpec defines pod template specification.
// This embeds corev1.PodTemplateSpec to allow arbitrary pod spec fields.
// Note: The operator will merge this with operator-managed fields, ensuring
// operator-managed fields (like ServiceAccountName, containers, etc.) take precedence.
// +kubebuilder:pruning:PreserveUnknownFields
type PodTemplateSpec struct {
	// Embedded PodTemplateSpec allows users to specify arbitrary pod spec fields
	// such as affinity, tolerations, nodeSelector, securityContext, etc.
	// Operator-managed fields (ServiceAccountName, containers, etc.) will be preserved.
	corev1.PodTemplateSpec `json:",inline"`
}

// AuthConfig defines authentication configuration
type AuthConfig struct {
	// Type specifies the authentication type.
	// +optional
	// +kubebuilder:validation:Enum=system;ldap;active-directory;oidc
	Type string `json:"type,omitempty"`

	// Username for authentication
	// +optional
	Username string `json:"username,omitempty"`

	// UsernameSecretRef references a Kubernetes Secret containing the auth username.
	// Takes precedence over Username if both are set.
	// +optional
	UsernameSecretRef *corev1.SecretKeySelector `json:"usernameSecretRef,omitempty"`

	// Password for authentication
	// +optional
	Password string `json:"password,omitempty"`

	// PasswordSecretRef references a Kubernetes Secret containing the auth password.
	// Takes precedence over Password if both are set.
	// +optional
	PasswordSecretRef *corev1.SecretKeySelector `json:"passwordSecretRef,omitempty"`

	// SessionSecret is a plain-text secret used to sign session cookies.
	// +optional
	SessionSecret string `json:"sessionSecret,omitempty"`

	// SessionSecretSecretRef references a Secret containing the session secret.
	// Takes precedence over SessionSecret when both are set.
	// +optional
	SessionSecretSecretRef *corev1.SecretKeySelector `json:"sessionSecretSecretRef,omitempty"`

	// APIKey is a plain-text API key for programmatic access.
	// +optional
	APIKey string `json:"apiKey,omitempty"`

	// APIKeySecretRef references a Secret containing the API key.
	// Takes precedence over APIKey when both are set.
	// +optional
	APIKeySecretRef *corev1.SecretKeySelector `json:"apiKeySecretRef,omitempty"`

	// SessionsStrictMode enables strict mode for session cookies.
	// +optional
	SessionsStrictMode bool `json:"sessionsStrictMode,omitempty"`

	// LDAP is the configuration for ldap or active-directory auth types.
	// +optional
	LDAP *LDAPConfig `json:"ldap,omitempty"`

	// OIDC is the configuration for the oidc auth type.
	// +optional
	OIDC *OIDCConfig `json:"oidc,omitempty"`
}

// LDAPConfig defines LDAP and Active Directory authentication configuration
type LDAPConfig struct {
	// Protocol to use when connecting to the LDAP server. One of: ldap|ldaps
	// +optional
	Protocol string `json:"protocol,omitempty"`

	// Server is the LDAP server hostname
	// +optional
	Server string `json:"server,omitempty"`

	// Port is the LDAP server port
	// +optional
	Port string `json:"port,omitempty"`

	// BaseDN is the base distinguished name for user searches
	// +optional
	BaseDN string `json:"baseDN,omitempty"`

	// BindUser is the username used to bind to the LDAP server
	// +optional
	BindUser string `json:"bindUser,omitempty"`

	// BindUserSecretRef references a Kubernetes Secret containing the LDAP bind username.
	// Takes precedence over BindUser if both are set.
	// +optional
	BindUserSecretRef *corev1.SecretKeySelector `json:"bindUserSecretRef,omitempty"`

	// BindPassword is the password used to bind to the LDAP server
	// +optional
	BindPassword string `json:"bindPassword,omitempty"`

	// BindPasswordSecretRef references a Kubernetes Secret containing the LDAP bind password.
	// Takes precedence over BindPassword if both are set.
	// +optional
	BindPasswordSecretRef *corev1.SecretKeySelector `json:"bindPasswordSecretRef,omitempty"`

	// SearchFilter is the LDAP search filter used to locate users
	// +optional
	SearchFilter string `json:"searchFilter,omitempty"`

	// TLS configures TLS for LDAP using a Secret. The operator mounts the Secret and sets
	// BINDPLANE_LDAP_TLS_CERT, BINDPLANE_LDAP_TLS_KEY, and BINDPLANE_LDAP_TLS_CA to the
	// mounted file paths. Omit TLS to disable mutual TLS / custom CA.
	// +optional
	TLS *LDAPTLSConfig `json:"tls,omitempty"`

	// TLSSkipVerify disables TLS certificate verification
	// +optional
	TLSSkipVerify bool `json:"tlsSkipVerify,omitempty"`
}

// LDAPTLSConfig defines TLS for LDAP by referencing a Secret. The Secret is mounted
// at a fixed path; the operator sets the TLS env vars to the mounted file paths.
// Users specify only the secret name and key names, not mount paths.
type LDAPTLSConfig struct {
	// SecretName is the name of the Secret containing the TLS certificate, key, and optionally CA.
	SecretName string `json:"secretName"`

	// CertKey is the key in the Secret for the TLS certificate (for mutual TLS).
	// +optional
	CertKey string `json:"certKey,omitempty"`

	// KeyKey is the key in the Secret for the TLS private key (for mutual TLS).
	// +optional
	KeyKey string `json:"keyKey,omitempty"`

	// CAKey is the key in the Secret for the CA certificate. Omit to use system CAs.
	// +optional
	CAKey string `json:"caKey,omitempty"`
}

// NetworkTLSConfig defines TLS for the Bindplane server by referencing a Secret. The Secret is mounted
// at a fixed path; the operator sets the TLS env vars to the mounted file paths.
// Users specify only the secret name and key names, not mount paths.
// Server-side TLS: set secretName, certKey, and keyKey. Mutual TLS: also set caKey.
type NetworkTLSConfig struct {
	// MinVersion is the minimum TLS version. One of: 1.2, 1.3. Omit to use server default.
	// +optional
	// +kubebuilder:validation:Enum=1.2;1.3
	MinVersion string `json:"minVersion,omitempty"`

	// SecretName is the name of the Secret containing the TLS certificate, key, and optionally CA.
	SecretName string `json:"secretName"`

	// CertKey is the key in the Secret for the TLS certificate (server or mutual TLS).
	// +optional
	CertKey string `json:"certKey,omitempty"`

	// KeyKey is the key in the Secret for the TLS private key (server or mutual TLS).
	// +optional
	KeyKey string `json:"keyKey,omitempty"`

	// CAKey is the key in the Secret for the CA certificate. Set for mutual TLS (client cert verification); generally not used.
	// +optional
	CAKey string `json:"caKey,omitempty"`

	// SkipVerify disables TLS certificate verification. Only set for testing.
	// +optional
	SkipVerify bool `json:"skipVerify,omitempty"`
}

// PostgresTLSConfig defines TLS for PostgreSQL by referencing a Secret. The Secret is mounted
// at a fixed path; the operator sets the TLS env vars (sslRootCert, sslCert, sslKey) to the mounted file paths.
// Users specify only the secret name and key names, not mount paths.
// Server-side TLS: set secretName and caKey. Mutual TLS: set secretName, caKey, certKey, and keyKey.
type PostgresTLSConfig struct {
	// SecretName is the name of the Secret containing the CA and optionally client cert and key.
	SecretName string `json:"secretName"`

	// CAKey is the key in the Secret for the root CA (maps to sslRootCert). Required for TLS; enables server-side TLS.
	// +optional
	CAKey string `json:"caKey,omitempty"`

	// CertKey is the key in the Secret for the client certificate (maps to sslCert). Set with KeyKey for mutual TLS.
	// +optional
	CertKey string `json:"certKey,omitempty"`

	// KeyKey is the key in the Secret for the client private key (maps to sslKey). Set with CertKey for mutual TLS.
	// +optional
	KeyKey string `json:"keyKey,omitempty"`
}

// OIDCConfig defines OpenID Connect authentication configuration
type OIDCConfig struct {
	// ClientID is the OIDC OAuth2 client ID
	// +optional
	ClientID string `json:"clientID,omitempty"`

	// ClientIDSecretRef references a Kubernetes Secret containing the OIDC client ID.
	// Takes precedence over ClientID if both are set.
	// +optional
	ClientIDSecretRef *corev1.SecretKeySelector `json:"clientIDSecretRef,omitempty"`

	// ClientSecret is the OIDC OAuth2 client secret
	// +optional
	ClientSecret string `json:"clientSecret,omitempty"`

	// ClientSecretSecretRef references a Kubernetes Secret containing the OIDC client secret.
	// Takes precedence over ClientSecret if both are set.
	// +optional
	ClientSecretSecretRef *corev1.SecretKeySelector `json:"clientSecretSecretRef,omitempty"`

	// Issuer is the URL of the OIDC provider
	// +optional
	Issuer string `json:"issuer,omitempty"`

	// Scopes is the list of OAuth2 scopes to request
	// +optional
	Scopes []string `json:"scopes,omitempty"`

	// DisableInvitations disables the invitation flow for OIDC-authenticated users.
	// When true, users cannot be invited via email and must log in via OIDC directly.
	// +optional
	DisableInvitations bool `json:"disableInvitations,omitempty"`
}

// NetworkConfig defines network configuration
type NetworkConfig struct {
	// Host specifies the bind address
	// +optional
	Host string `json:"host,omitempty"`

	// Port specifies the port to listen on
	// +optional
	Port string `json:"port,omitempty"`

	// RemoteURL specifies the remote URL for Bindplane.
	// Defaults to http://<bindplane-name>-node:3001 (the internal node service URL).
	// Override this when using ingress, e.g. https://bindplane.my-corp.net
	// +optional
	RemoteURL string `json:"remoteURL,omitempty"`

	// WebURL is the URL used by the client for the web interface. Defaults to RemoteURL when not set. Only set when explicitly configuring.
	// +optional
	WebURL string `json:"webURL,omitempty"`

	// CorsAllowedOrigins is the allowed origin for CORS requests. Only set when explicitly configuring.
	// +optional
	CorsAllowedOrigins string `json:"corsAllowedOrigins,omitempty"`

	// TLS configures TLS for the Bindplane server using a Secret. The operator mounts the Secret and sets
	// BINDPLANE_TLS_CERT, BINDPLANE_TLS_KEY, and optionally BINDPLANE_TLS_CA to the mounted file paths.
	// Omit or omit secretName/certKey/keyKey to disable server TLS (e.g. when using Ingress to terminate TLS).
	// +optional
	TLS *NetworkTLSConfig `json:"tls,omitempty"`
}

// StoreConfig defines store configuration
type StoreConfig struct {
	// Postgres configuration
	Postgres *PostgresConfig `json:"postgres"`

	// MaxEvents is the maximum number of events to merge into a single event. Defaults to 100.
	// +optional
	// +kubebuilder:default=100
	MaxEvents int `json:"maxEvents,omitempty"`

	// EventMergeWindow is the window during which events are merged (e.g. "100ms"). Defaults to 100ms.
	// +optional
	// +kubebuilder:default="100ms"
	EventMergeWindow string `json:"eventMergeWindow,omitempty"`

	// SummaryRollupRetentionDays is the number of days to retain daily rollup data.
	// 0 means indefinite retention (rollups are never deleted). Defaults to 365.
	// +optional
	// +kubebuilder:default=365
	SummaryRollupRetentionDays *int `json:"summaryRollupRetentionDays,omitempty"`
}

// PostgresConfig defines PostgreSQL store configuration
type PostgresConfig struct {
	// Host specifies the PostgreSQL host
	Host string `json:"host"`

	// Port specifies the PostgreSQL port
	// +optional
	Port string `json:"port,omitempty"`

	// ConnectTimeout specifies the connection timeout
	// +optional
	ConnectTimeout string `json:"connectTimeout,omitempty"`

	// StatementTimeout specifies the statement timeout
	// +optional
	StatementTimeout string `json:"statementTimeout,omitempty"`

	// Database specifies the database name
	// +optional
	Database string `json:"database,omitempty"`

	// SSLMode specifies the PostgreSQL SSL mode. One of: disable, require, verify-ca, verify-full.
	// +optional
	// +kubebuilder:default=disable
	// +kubebuilder:validation:Enum=disable;require;verify-ca;verify-full
	SSLMode string `json:"sslmode,omitempty"`

	// TLS configures TLS for PostgreSQL using a Secret. The operator mounts the Secret and sets
	// BINDPLANE_POSTGRES_SSL_ROOT_CERT, BINDPLANE_POSTGRES_SSL_CERT, and BINDPLANE_POSTGRES_SSL_KEY to the
	// mounted file paths. Server-side TLS: set secretName and caKey. Mutual TLS: also set certKey and keyKey.
	// +optional
	TLS *PostgresTLSConfig `json:"tls,omitempty"`

	// Username specifies the PostgreSQL username
	// +optional
	Username string `json:"username,omitempty"`

	// UsernameSecretRef references a Kubernetes Secret containing the PostgreSQL username.
	// Takes precedence over Username if both are set.
	// +optional
	UsernameSecretRef *corev1.SecretKeySelector `json:"usernameSecretRef,omitempty"`

	// Password specifies the PostgreSQL password
	// +optional
	Password string `json:"password,omitempty"`

	// PasswordSecretRef references a Kubernetes Secret containing the PostgreSQL password.
	// Takes precedence over Password if both are set.
	// +optional
	PasswordSecretRef *corev1.SecretKeySelector `json:"passwordSecretRef,omitempty"`

	// MaxConnections specifies the maximum number of connections
	// +optional
	MaxConnections int `json:"maxConnections,omitempty"`

	// MaxIdleConnections specifies the maximum number of idle connections. Optional; no default.
	// +optional
	MaxIdleConnections *int `json:"maxIdleConnections,omitempty"`

	// MaxLifetime specifies the maximum connection lifetime
	// +optional
	MaxLifetime string `json:"maxLifetime,omitempty"`

	// MaxIdleTime specifies the maximum time a connection may remain idle (e.g. 20s, 1m). Optional; no default.
	// +optional
	MaxIdleTime string `json:"maxIdleTime,omitempty"`

	// Schema specifies the database schema
	// +optional
	Schema string `json:"schema,omitempty"`
}

// ComponentStatus reports the resolved image and runtime health of a single Bindplane component.
type ComponentStatus struct {
	// Image is the fully-resolved container image in use by this component.
	// +optional
	Image string `json:"image,omitempty"`

	// ReadyReplicas is the number of pods currently ready for this component.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`
}

// BindplaneComponents groups the per-component status for all Bindplane components.
type BindplaneComponents struct {
	// Bindplane reports the image and ready replica count for the Bindplane Node deployment.
	// +optional
	Bindplane ComponentStatus `json:"bindplane,omitempty"`

	// OpAMP reports the image and ready replica count for the OpAMP deployment.
	// Empty when OpAMP is not enabled.
	// +optional
	OpAMP ComponentStatus `json:"opamp,omitempty"`

	// Jobs reports the image and ready replica count for the Bindplane Jobs deployment.
	// +optional
	Jobs ComponentStatus `json:"jobs,omitempty"`

	// JobsMigrate reports the image for which a successful database migration has completed.
	// The controller uses this to determine whether migration must run before applying
	// an image change to NATS, Jobs, and Node workloads. ReadyReplicas is not set because
	// the migration Job is transient.
	// +optional
	JobsMigrate ComponentStatus `json:"jobsMigrate,omitempty"`

	// Nats reports the image and ready replica count for the NATS StatefulSet.
	// +optional
	Nats ComponentStatus `json:"nats,omitempty"`

	// TransformAgent reports the image and ready replica count for the Transform Agent deployment.
	// +optional
	TransformAgent ComponentStatus `json:"transformAgent,omitempty"`

	// TSDB reports the image and ready replica count for the TSDB (Prometheus) StatefulSet.
	// Empty when remote TSDB is enabled.
	// +optional
	TSDB ComponentStatus `json:"tsdb,omitempty"`
}

// BindplaneStatus defines the observed state of Bindplane.
type BindplaneStatus struct {
	// Conditions represent the latest available observations of the Bindplane's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Phase summarizes the overall deployment state.
	// +optional
	// +kubebuilder:validation:Enum=Pending;ApplyingChanges;Ready;Degraded;Paused;Deleting
	Phase string `json:"phase,omitempty"`

	// ObservedGeneration is the most recent generation observed by the controller.
	// It corresponds to the Bindplane's generation, which is updated on mutation
	// by the API Server. This field is used by GitOps tools (Argo CD, Flux) and
	// kubectl wait to determine whether the controller has processed the latest spec.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Components holds the per-component image and readiness status for each deployed
	// Bindplane component.
	// +optional
	Components BindplaneComponents `json:"components,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=bindplanes,singular=bindplane,scope=Namespaced
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Overall deployment phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:webhook:path=/validate-k8s-bindplane-com-v1alpha1-bindplane,mutating=false,failurePolicy=fail,sideEffects=None,groups=k8s.bindplane.com,resources=bindplanes,verbs=create;update,versions=v1alpha1,name=vbindplane.kb.io,admissionReviewVersions=v1

// Bindplane is the Schema for the bindplanes API.
type Bindplane struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BindplaneSpec   `json:"spec,omitempty"`
	Status BindplaneStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BindplaneList contains a list of Bindplane.
type BindplaneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Bindplane `json:"items"`
}
