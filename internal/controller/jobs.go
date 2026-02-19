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

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	bindplanev1alpha1 "github.com/bindplane-operator/bindplane-operator/api/v1alpha1"
)

const (
	// bindplaneJobsMigrateComponent is the component name for Bindplane Jobs Migrate
	bindplaneJobsMigrateComponent = "jobs-migrate"
	// bindplaneJobsComponent is the component name for Bindplane Jobs
	bindplaneJobsComponent = "jobs"
	// bindplaneJobsContainerName is the container name for Bindplane Jobs
	bindplaneJobsContainerName = "server"
	// bindplaneJobsImage is the default container image for Bindplane Jobs
	bindplaneJobsImage = "ghcr.io/observiq/bindplane-ee:1.96.3"
	// bindplaneJobsHTTPPort is the HTTP port for Bindplane Jobs
	bindplaneJobsHTTPPort = int32(3001)
	// bindplaneJobsHTTPPortName is the name of the HTTP port for Bindplane Jobs
	bindplaneJobsHTTPPortName = "http"
	// bindplaneJobsMigrateModeValue is the value for BINDPLANE_MODE for migrate jobs
	bindplaneJobsMigrateModeValue = "migrate"
	// bindplaneJobsModeValue is the value for BINDPLANE_MODE for regular jobs
	bindplaneJobsModeValue = "all,-migrate"
)

// reconcileBindplaneJobs reconciles all Bindplane Jobs resources
// Note: These deployments do NOT create Services, as traffic should not be routed to them
func (r *BindplaneReconciler) reconcileBindplaneJobs(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
	// Reconcile Jobs Migrate
	if err := r.reconcileBindplaneJobsMigrate(ctx, bindplane, log); err != nil {
		return err
	}

	// Reconcile Jobs
	if err := r.reconcileBindplaneJobsRegular(ctx, bindplane, log); err != nil {
		return err
	}

	return nil
}

// reconcileBindplaneJobsMigrate reconciles the Bindplane Jobs Migrate deployment
func (r *BindplaneReconciler) reconcileBindplaneJobsMigrate(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
	// Reconcile ServiceAccount
	sa := r.bindplaneJobsMigrateServiceAccount(bindplane)
	if err := r.reconcileServiceAccount(ctx, bindplane, sa, log); err != nil {
		return err
	}

	// Reconcile Deployment
	deployment := r.bindplaneJobsMigrateDeployment(bindplane)
	if err := r.reconcileDeployment(ctx, bindplane, deployment, log); err != nil {
		return err
	}

	return nil
}

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

func (r *BindplaneReconciler) bindplaneJobsMigrateDeployment(bindplane *bindplanev1alpha1.Bindplane) *appsv1.Deployment {
	// Jobs Migrate resources: 100m CPU, 2048Mi memory
	resources := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("2048Mi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("2048Mi"),
		},
	}
	// maxSurge=0 ensures the old pod is deleted before the new pod is created,
	// so only one migrate pod runs at a time.
	maxSurge := intstr.FromInt32(0)
	maxUnavailable := intstr.FromInt32(1)
	return r.bindplaneJobsDeploymentCommon(bindplane, bindplaneJobsMigrateComponent, bindplaneJobsMigrateModeValue, appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxSurge:       &maxSurge,
			MaxUnavailable: &maxUnavailable,
		},
	}, false, resources) // false = don't include NATS client config
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
	return r.bindplaneJobsDeploymentCommon(bindplane, bindplaneJobsComponent, bindplaneJobsModeValue, appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxSurge:       &maxSurge,
			MaxUnavailable: &maxUnavailable,
		},
	}, true, resources) // true = include NATS client config
}

// bindplaneJobsDeploymentCommon creates a deployment for Bindplane Jobs with configurable component, mode, and strategy
func (r *BindplaneReconciler) bindplaneJobsDeploymentCommon(bindplane *bindplanev1alpha1.Bindplane, component string, modeValue string, strategy appsv1.DeploymentStrategy, includeNatsClient bool, resources corev1.ResourceRequirements) *appsv1.Deployment {
	replicas := int32(1)
	labels := getLabels(bindplane, component)
	selectorLabels := getSelectorLabels(bindplane, component)
	ldapVols, ldapMounts := getLDAPTLSVolumeAndMount(bindplane)

	// Get the appropriate PodTemplate and Affinity based on component
	var podTemplate *bindplanev1alpha1.PodTemplateSpec
	var affinity *corev1.Affinity
	if component == bindplaneJobsMigrateComponent {
		podTemplate = getBindplaneJobsMigratePodTemplate(bindplane)
		affinity = getBindplaneJobsMigrateAffinity(bindplane)
	} else {
		podTemplate = getBindplaneJobsPodTemplate(bindplane)
		affinity = getBindplaneJobsAffinity(bindplane)
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getResourceName(bindplane, component),
			Namespace: bindplane.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Strategy: strategy,
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Template: mergePodTemplateSpec(
				corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: selectorLabels,
					},
					Spec: corev1.PodSpec{
						Volumes:            ldapVols,
						ServiceAccountName: getResourceName(bindplane, component),
						SecurityContext: &corev1.PodSecurityContext{
							FSGroup:    new(defaultRunAsGroup),
							RunAsGroup: new(defaultRunAsGroup),
							RunAsUser:  new(defaultRunAsUser),
						},
						Affinity: affinity,
						Containers: []corev1.Container{
							{
								Name:         bindplaneJobsContainerName,
								Image:        bindplaneJobsImage,
								VolumeMounts: ldapMounts,
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
											Value: modeValue,
										},
									},
									getBindplaneConfigEnvVars(bindplane),
									getPrometheusEnvVars(bindplane),
									getTransformAgentEnvVars(bindplane),
									getNatsClientEnvVars(bindplane, includeNatsClient),
								),
								Resources: resources,
								StartupProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Path: healthCheckPath,
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
										HTTPGet: &corev1.HTTPGetAction{
											Path: healthCheckPath,
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
										HTTPGet: &corev1.HTTPGetAction{
											Path: healthCheckPath,
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
				podTemplate,
			),
		},
	}
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

// secretOrValue returns an EnvVar sourced from a Secret when ref is set, or from
// a plain value when value is non-empty. Returns nil when neither is provided.
// Secret ref takes precedence when both are set.
func secretOrValue(name, value string, ref *corev1.SecretKeySelector) *corev1.EnvVar {
	if ref != nil {
		return &corev1.EnvVar{
			Name:      name,
			ValueFrom: &corev1.EnvVarSource{SecretKeyRef: ref},
		}
	}
	if value != "" {
		return &corev1.EnvVar{Name: name, Value: value}
	}
	return nil
}

// getLDAPEnvVars returns LDAP / Active Directory environment variables.
// Returns nil when ldap is nil.
func getLDAPEnvVars(ldap *bindplanev1alpha1.LDAPConfig) []corev1.EnvVar {
	if ldap == nil {
		return nil
	}
	var envVars []corev1.EnvVar
	if ldap.Protocol != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneLDAPProtocolEnvVar, Value: ldap.Protocol})
	}
	if ldap.Server != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneLDAPServerEnvVar, Value: ldap.Server})
	}
	if ldap.Port != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneLDAPPortEnvVar, Value: ldap.Port})
	}
	if ldap.BaseDN != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneLDAPBaseDNEnvVar, Value: ldap.BaseDN})
	}
	if ldap.BindUser != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneLDAPBindUserEnvVar, Value: ldap.BindUser})
	}
	if ev := secretOrValue(bindplaneLDAPBindPasswordEnvVar, ldap.BindPassword, ldap.BindPasswordSecret); ev != nil {
		envVars = append(envVars, *ev)
	}
	if ldap.SearchFilter != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneLDAPSearchFilterEnvVar, Value: ldap.SearchFilter})
	}
	if ldap.TLS != nil {
		if ldap.TLS.CertKey != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneLDAPTLSCertEnvVar, Value: ldapTLSMountPath + "/" + ldap.TLS.CertKey})
		}
		if ldap.TLS.KeyKey != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneLDAPTLSKeyEnvVar, Value: ldapTLSMountPath + "/" + ldap.TLS.KeyKey})
		}
		if ldap.TLS.CAKey != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneLDAPTLSCAEnvVar, Value: ldapTLSMountPath + "/" + ldap.TLS.CAKey})
		}
	}
	if ldap.TLSSkipVerify {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneLDAPTLSSkipVerifyEnvVar, Value: "true"})
	}
	return envVars
}

// getOIDCEnvVars returns OIDC environment variables.
// Returns nil when oidc is nil.
func getOIDCEnvVars(oidc *bindplanev1alpha1.OIDCConfig) []corev1.EnvVar {
	if oidc == nil {
		return nil
	}
	var envVars []corev1.EnvVar
	if ev := secretOrValue(bindplaneOIDCClientIDEnvVar, oidc.ClientID, oidc.ClientIDSecret); ev != nil {
		envVars = append(envVars, *ev)
	}
	if ev := secretOrValue(bindplaneOIDCClientSecretEnvVar, oidc.ClientSecret, oidc.ClientSecretSecret); ev != nil {
		envVars = append(envVars, *ev)
	}
	if oidc.Issuer != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneOIDCIssuerEnvVar, Value: oidc.Issuer})
	}
	if len(oidc.Scopes) > 0 {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneOIDCScopesEnvVar, Value: strings.Join(oidc.Scopes, ",")})
	}
	return envVars
}

// getBindplaneConfigEnvVars converts BindplaneConfigSpec to environment variables
// following the naming convention from override_test.go (BINDPLANE_*)
func getBindplaneConfigEnvVars(bindplane *bindplanev1alpha1.Bindplane) []corev1.EnvVar {
	var envVars []corev1.EnvVar
	config := &bindplane.Spec.Config

	// License
	if ev := secretOrValue(bindplaneLicenseEnvVar, config.License, config.LicenseSecret); ev != nil {
		envVars = append(envVars, *ev)
	}

	// Auth configuration
	if config.Auth != nil {
		if config.Auth.Type != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  bindplaneAuthTypeEnvVar,
				Value: config.Auth.Type,
			})
		}
		if config.Auth.SessionsStrictMode {
			envVars = append(envVars, corev1.EnvVar{
				Name:  bindplaneAuthSessionsStrictModeEnvVar,
				Value: "true",
			})
		}
		if ev := secretOrValue(bindplaneUsernameEnvVar, config.Auth.Username, config.Auth.UsernameSecret); ev != nil {
			envVars = append(envVars, *ev)
		}
		if ev := secretOrValue(bindplanePasswordEnvVar, config.Auth.Password, config.Auth.PasswordSecret); ev != nil {
			envVars = append(envVars, *ev)
		}

		// LDAP / Active Directory
		envVars = append(envVars, getLDAPEnvVars(config.Auth.LDAP)...)

		// OIDC
		envVars = append(envVars, getOIDCEnvVars(config.Auth.OIDC)...)
	}

	// Network configuration
	if config.Network != nil {
		if config.Network.Host != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  bindplaneHostEnvVar,
				Value: config.Network.Host,
			})
		}
		if config.Network.Port != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  bindplanePortEnvVar,
				Value: config.Network.Port,
			})
		}
	}

	// Remote URL is always set; user-provided value takes precedence over the default.
	remoteURL := ""
	if config.Network != nil {
		remoteURL = config.Network.RemoteURL
	}
	if remoteURL == "" {
		remoteURL = fmt.Sprintf("http://%s-%s:%d", bindplane.Name, nodeComponent, nodeHTTPPort)
	}
	envVars = append(envVars, corev1.EnvVar{
		Name:  bindplaneRemoteURLEnvVar,
		Value: remoteURL,
	})

	// Store type is always postgres
	envVars = append(envVars, corev1.EnvVar{
		Name:  bindplaneStoreTypeEnvVar,
		Value: "postgres",
	})

	// Postgres configuration
	if config.Store.Postgres != nil {
		if config.Store.Postgres.Host != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  bindplanePostgresHostEnvVar,
				Value: config.Store.Postgres.Host,
			})
		}
		if config.Store.Postgres.Port != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  bindplanePostgresPortEnvVar,
				Value: config.Store.Postgres.Port,
			})
		}
		if config.Store.Postgres.ConnectTimeout != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  bindplanePostgresConnectTimeoutEnvVar,
				Value: config.Store.Postgres.ConnectTimeout,
			})
		}
		if config.Store.Postgres.StatementTimeout != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  bindplanePostgresStatementTimeoutEnvVar,
				Value: config.Store.Postgres.StatementTimeout,
			})
		}
		if config.Store.Postgres.Database != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  bindplanePostgresDatabaseEnvVar,
				Value: config.Store.Postgres.Database,
			})
		}
		if config.Store.Postgres.SSLMode != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  bindplanePostgresSSLModeEnvVar,
				Value: config.Store.Postgres.SSLMode,
			})
		}
		if ev := secretOrValue(bindplanePostgresUsernameEnvVar, config.Store.Postgres.Username, config.Store.Postgres.UsernameSecret); ev != nil {
			envVars = append(envVars, *ev)
		}
		if ev := secretOrValue(bindplanePostgresPasswordEnvVar, config.Store.Postgres.Password, config.Store.Postgres.PasswordSecret); ev != nil {
			envVars = append(envVars, *ev)
		}
		if config.Store.Postgres.MaxConnections > 0 {
			envVars = append(envVars, corev1.EnvVar{
				Name:  bindplanePostgresMaxConnectionsEnvVar,
				Value: strconv.Itoa(config.Store.Postgres.MaxConnections),
			})
		}
		if config.Store.Postgres.MaxLifetime != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  bindplanePostgresMaxLifetimeEnvVar,
				Value: config.Store.Postgres.MaxLifetime,
			})
		}
		if config.Store.Postgres.Schema != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  bindplanePostgresSchemaEnvVar,
				Value: config.Store.Postgres.Schema,
			})
		}
	}

	return envVars
}

// getPrometheusEnvVars returns the Prometheus environment variables
// Used by jobs, jobs-migrate, and node deployments
func getPrometheusEnvVars(bindplane *bindplanev1alpha1.Bindplane) []corev1.EnvVar {
	prometheusServiceName := getResourceName(bindplane, prometheusComponent)
	prometheusPort := strconv.Itoa(int(prometheusHTTPPort))

	return []corev1.EnvVar{
		{
			Name:  bindplanePrometheusEnableRemoteEnvVar,
			Value: enableRemoteValue,
		},
		{
			Name:  bindplanePrometheusHostEnvVar,
			Value: prometheusServiceName,
		},
		{
			Name:  bindplanePrometheusPortEnvVar,
			Value: prometheusPort,
		},
	}
}

// getTransformAgentEnvVars returns the Transform Agent environment variables
// Used by jobs, jobs-migrate, and node deployments
func getTransformAgentEnvVars(bindplane *bindplanev1alpha1.Bindplane) []corev1.EnvVar {
	transformAgentServiceName := getResourceName(bindplane, transformAgentComponent)
	transformAgentPort := strconv.Itoa(int(transformAgentHTTPPort))
	transformAgentRemoteAgents := transformAgentServiceName + ":" + transformAgentPort

	return []corev1.EnvVar{
		{
			Name:  bindplaneTransformAgentEnableRemoteEnvVar,
			Value: enableRemoteValue,
		},
		{
			Name:  bindplaneTransformAgentRemoteAgentsEnvVar,
			Value: transformAgentRemoteAgents,
		},
	}
}

// getNatsClientEnvVars returns the NATS client environment variables for jobs deployment
func getNatsClientEnvVars(bindplane *bindplanev1alpha1.Bindplane, includeNatsClient bool) []corev1.EnvVar {
	if !includeNatsClient {
		return nil
	}

	return []corev1.EnvVar{
		{
			Name:  bindplaneEventBusTypeEnvVar,
			Value: natsEventBusType,
		},
		{
			Name: bindplaneNatsClientNameEnvVar,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: metadataNameFieldPath,
				},
			},
		},
		{
			Name:  bindplaneNatsClientEndpointEnvVar,
			Value: getNatsClientEndpoint(bindplane),
		},
		{
			Name:  bindplaneNatsClientSubjectEnvVar,
			Value: natsClientSubject,
		},
	}
}
