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
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"maps"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	"golang.org/x/crypto/bcrypt"

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

const (
	// prometheusComponent is the component name for Prometheus
	prometheusComponent = "prometheus"
	// prometheusContainerName is the container name for Prometheus
	prometheusContainerName = "prometheus"
	// prometheusImage is the default container image for Prometheus
	prometheusImage = "ghcr.io/observiq/bindplane-prometheus:1.96.3"
	// prometheusDataVolumeSuffix is the suffix for Prometheus data volume names
	prometheusDataVolumeSuffix = "prometheus-data"
	// prometheusHTTPPort is the HTTP port for Prometheus
	prometheusHTTPPort = 9090
	// prometheusHTTPPortName is the name of the HTTP port for Prometheus
	prometheusHTTPPortName = "http"

	// Prometheus basic auth (operator-generated secret)
	prometheusBasicAuthSecretSuffix  = "prometheus-basic-auth"
	prometheusBasicAuthUsername      = "prometheus"
	prometheusBasicAuthSecretKeyUser = "username"
	prometheusBasicAuthSecretKeyPass = "password"
	prometheusBasicAuthSecretKeyWeb  = "web-config"
	prometheusWebConfigVolumeName    = "prometheus-web-config"
	prometheusWebConfigMountPath     = "/etc/prometheus"
	prometheusWebConfigFileName      = "web.yml"
	prometheusProbeHTTPFileName      = "probe-http.yml"
	// Server cert mount when internal TLS is enabled for remote write
	prometheusTLSVolumeName            = "prometheus-tls"
	prometheusTLSMountPath             = "/etc/prometheus-tls"
	prometheusWebConfigConfigMapSuffix = "prometheus-web-config"
	// Cert-manager: separate server cert (web) and probe client cert (promtool) so EKU matches (ServerAuth vs ClientAuth).
	prometheusWebServerTLSVolumeName   = "prometheus-web-server-tls"
	prometheusWebServerTLSMountPath    = "/etc/prometheus-web-tls"
	prometheusProbeClientTLSVolumeName = "prometheus-probe-client-tls"
	prometheusProbeClientTLSMountPath  = "/etc/prometheus-probe-client"
	// Probe basic auth: existing basic-auth secret mounted as files for promtool username_file/password_file.
	prometheusProbeAuthVolumeName = "prometheus-probe-auth"
	prometheusProbeAuthMountPath  = "/etc/prometheus-probe-auth"

	// Prometheus exec probe timing (matches reference out.yaml: startup/readiness use /-/ready, liveness uses /-/healthy).
	prometheusProbeStartupFailureThreshold   int32 = 60
	prometheusProbeStartupPeriodSeconds      int32 = 15
	prometheusProbeStartupSuccessThreshold   int32 = 1
	prometheusProbeStartupTimeoutSeconds     int32 = 3
	prometheusProbeReadinessFailureThreshold int32 = 3
	prometheusProbeReadinessPeriodSeconds    int32 = 5
	prometheusProbeReadinessSuccessThreshold int32 = 1
	prometheusProbeReadinessTimeoutSeconds   int32 = 3
	prometheusProbeLivenessFailureThreshold  int32 = 6
	prometheusProbeLivenessPeriodSeconds     int32 = 5
	prometheusProbeLivenessSuccessThreshold  int32 = 1
	prometheusProbeLivenessTimeoutSeconds    int32 = 3
)

// prometheusWebConfig is the structure of Prometheus web.config.file (basic auth + TLS).
// See https://prometheus.io/docs/prometheus/latest/configuration/https/
// We use json tags because sigs.k8s.io/yaml uses JSON for marshaling; keys must be basic_auth_users, tls_server_config.
type prometheusWebConfig struct {
	BasicAuthUsers  map[string]string          `json:"basic_auth_users,omitempty"`
	TLSServerConfig *prometheusTLSServerConfig `json:"tls_server_config,omitempty"`
}

// prometheusTLSServerConfig is the tls_server_config section of Prometheus web config.
// ClientCAFile and ClientAuthType are only set for mTLS (client cert verification); omit for TLS-only.
type prometheusTLSServerConfig struct {
	CertFile       string `json:"cert_file"`
	KeyFile        string `json:"key_file"`
	ClientCAFile   string `json:"client_ca_file,omitempty"`
	ClientAuthType string `json:"client_auth_type,omitempty"`
}

// promtoolProbeHTTPConfig is the config file for promtool check (HTTP client TLS). Used by exec probes.
// See https://prometheus.io/docs/prometheus/latest/configuration/configuration/#http_config
type promtoolProbeHTTPConfig struct {
	BasicAuth *promtoolProbeBasicAuthConfig `json:"basic_auth,omitempty"`
	TLSConfig *promtoolProbeTLSConfig       `json:"tls_config,omitempty"`
}

// promtoolProbeBasicAuthConfig is the basic_auth section; use username_file and password_file (plaintext files from mounted secret).
type promtoolProbeBasicAuthConfig struct {
	UsernameFile string `json:"username_file"`
	PasswordFile string `json:"password_file"`
}

type promtoolProbeTLSConfig struct {
	CAFile             string `json:"ca_file,omitempty"`
	CertFile           string `json:"cert_file,omitempty"`
	KeyFile            string `json:"key_file,omitempty"`
	ServerName         string `json:"server_name,omitempty"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify,omitempty"`
}

// isPrometheusServerTLSEnabled returns true when the Prometheus server (StatefulSet) should serve TLS (spec.prometheus.tls with certManager or secretName).
func isPrometheusServerTLSEnabled(bindplane *bindplanev1alpha1.Bindplane) bool {
	p := bindplane.Spec.Prometheus
	if p == nil || p.TLS == nil {
		return false
	}
	return (p.TLS.CertManager != nil && p.TLS.CertManager.Name != "") || p.TLS.SecretName != ""
}

// isPrometheusServerMTLSEnabled returns true when the Prometheus server should require and verify client certs (mTLS).
// True when: (cert-manager server and cert-manager client both enabled) or (user secret with CAKey set).
func isPrometheusServerMTLSEnabled(bindplane *bindplanev1alpha1.Bindplane) bool {
	p := bindplane.Spec.Prometheus
	if p == nil || p.TLS == nil {
		return false
	}
	tls := p.TLS
	if tls.CertManager != nil && tls.CertManager.Name != "" {
		return isPrometheusClientCertManagerTLSEnabled(bindplane)
	}
	// User-defined secret: mTLS only when user provides a CA for client verification
	return tls.CAKey != ""
}

// getPrometheusProbeServerName returns the server_name for promtool probe (must match a SAN/CN of the Prometheus server cert).
// Prefer the service FQDN so verification works when connecting to 127.0.0.1; fallback to localhost.
func getPrometheusProbeServerName(bindplane *bindplanev1alpha1.Bindplane) string {
	names := getPrometheusServerCertDNSNames(bindplane)
	if len(names) > 0 {
		return names[0] // e.g. my-bp-prometheus.default.svc.cluster.local
	}
	return "localhost"
}

// isPrometheusServerProbeTLSWithCA returns true when the server TLS secret has a CA file (for probe to verify server cert).
func isPrometheusServerProbeTLSWithCA(bindplane *bindplanev1alpha1.Bindplane) bool {
	p := bindplane.Spec.Prometheus
	if p == nil || p.TLS == nil {
		return false
	}
	if p.TLS.CertManager != nil && p.TLS.CertManager.Name != "" {
		return true // cert-manager secrets include ca.crt
	}
	return p.TLS.CAKey != "" // user secret with CA
}

// buildPrometheusProbeHTTPConfigYAML returns the probe-http.yml content for promtool exec probes when server TLS is enabled.
// With cert-manager: use probe CLIENT cert (ClientAuth) and CA from server secret; server_name must match server cert SAN.
// With user secret: use same paths as web (single secret); may use insecure_skip_verify if no CA.
// basic_auth uses username_file and password_file (existing basic-auth secret mounted at prometheusProbeAuthMountPath).
func buildPrometheusProbeHTTPConfigYAML(bindplane *bindplanev1alpha1.Bindplane) ([]byte, error) {
	cfg := promtoolProbeHTTPConfig{
		BasicAuth: &promtoolProbeBasicAuthConfig{
			UsernameFile: prometheusProbeAuthMountPath + "/username",
			PasswordFile: prometheusProbeAuthMountPath + "/password",
		},
		TLSConfig: &promtoolProbeTLSConfig{
			ServerName: getPrometheusProbeServerName(bindplane),
		},
	}
	tlsCfg := cfg.TLSConfig
	if isPrometheusServerCertManagerTLSEnabled(bindplane) {
		tlsCfg.CAFile = prometheusWebServerTLSMountPath + "/ca.crt"
		tlsCfg.CertFile = prometheusProbeClientTLSMountPath + "/tls.crt"
		tlsCfg.KeyFile = prometheusProbeClientTLSMountPath + "/tls.key"
	} else if isPrometheusServerProbeTLSWithCA(bindplane) {
		tlsCfg.CAFile = prometheusTLSMountPath + "/ca.crt"
		tlsCfg.CertFile = prometheusTLSMountPath + "/tls.crt"
		tlsCfg.KeyFile = prometheusTLSMountPath + "/tls.key"
	} else {
		tlsCfg.InsecureSkipVerify = true
	}
	return yaml.Marshal(&cfg)
}

// reconcilePrometheus reconciles all Prometheus resources
func (r *BindplaneReconciler) reconcilePrometheus(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
	// Reconcile Prometheus basic auth Secret first (create-only) so it exists for StatefulSet and Bindplane pods
	if err := r.reconcilePrometheusBasicAuthSecret(ctx, bindplane, log); err != nil {
		return err
	}
	// When Prometheus server TLS is enabled (spec.prometheus.tls), reconcile web-config ConfigMap (basic auth + tls_server_config)
	if isPrometheusServerTLSEnabled(bindplane) {
		if err := r.reconcilePrometheusWebConfigConfigMap(ctx, bindplane, log); err != nil {
			return err
		}
	}

	// Reconcile ServiceAccount
	sa := r.prometheusServiceAccount(bindplane)
	if err := r.reconcileServiceAccount(ctx, bindplane, sa, log); err != nil {
		return err
	}

	// Reconcile StatefulSet
	statefulSet := r.prometheusStatefulSet(bindplane)
	if err := r.reconcileStatefulSet(ctx, bindplane, statefulSet, log); err != nil {
		return err
	}

	// Reconcile Service
	service := r.prometheusService(bindplane)
	if err := r.reconcileService(ctx, bindplane, service, log); err != nil {
		return err
	}

	return nil
}

// generatePrometheusBasicAuthSecretData returns secret data (username, password, web-config YAML).
// Password is 24 random bytes base64-encoded (32 chars); web-config is Prometheus basic_auth_users YAML with bcrypt hash.
func generatePrometheusBasicAuthSecretData() (map[string][]byte, error) {
	passwordBytes := make([]byte, 24)
	if _, err := rand.Read(passwordBytes); err != nil {
		return nil, fmt.Errorf("generate password: %w", err)
	}
	password := base64.URLEncoding.EncodeToString(passwordBytes) // 32 chars

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("bcrypt password: %w", err)
	}
	cfg := prometheusWebConfig{
		BasicAuthUsers: map[string]string{prometheusBasicAuthUsername: string(hash)},
	}
	webConfigBytes, err := yaml.Marshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal web config: %w", err)
	}

	return map[string][]byte{
		prometheusBasicAuthSecretKeyUser: []byte(prometheusBasicAuthUsername),
		prometheusBasicAuthSecretKeyPass: []byte(password),
		prometheusBasicAuthSecretKeyWeb:  webConfigBytes,
	}, nil
}

// reconcilePrometheusBasicAuthSecret creates the Prometheus basic auth Secret if it does not exist.
// Existing Secret data is never updated to avoid rotating credentials.
func (r *BindplaneReconciler) reconcilePrometheusBasicAuthSecret(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
	secretName := getResourceName(bindplane, prometheusBasicAuthSecretSuffix)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: bindplane.Namespace,
			Labels:    getLabels(bindplane, prometheusBasicAuthSecretSuffix),
		},
	}

	if err := controllerutil.SetControllerReference(bindplane, secret, r.Scheme); err != nil {
		return err
	}

	existing := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: bindplane.Namespace}, existing)
	if err == nil {
		// Secret exists; do not overwrite data
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	data, err := generatePrometheusBasicAuthSecretData()
	if err != nil {
		return err
	}
	secret.Data = data

	log.Info("Creating Prometheus basic auth Secret", "name", secretName, "namespace", bindplane.Namespace)
	return r.Create(ctx, secret)
}

// reconcilePrometheusWebConfigConfigMap creates or updates a ConfigMap with web config (basic auth + tls_server_config)
// when internal TLS is enabled for Prometheus remote write. Reads the basic auth Secret to get basic_auth_users.
func (r *BindplaneReconciler) reconcilePrometheusWebConfigConfigMap(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
	secretName := getResourceName(bindplane, prometheusBasicAuthSecretSuffix)
	existingSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: bindplane.Namespace}, existingSecret); err != nil {
		return fmt.Errorf("get basic auth secret for web config: %w", err)
	}
	webConfigBytes, ok := existingSecret.Data[prometheusBasicAuthSecretKeyWeb]
	if !ok {
		return fmt.Errorf("basic auth secret missing key %q", prometheusBasicAuthSecretKeyWeb)
	}
	var cfg prometheusWebConfig
	if err := yaml.Unmarshal(webConfigBytes, &cfg); err != nil {
		return fmt.Errorf("unmarshal web config from secret: %w", err)
	}
	// TLS server config: cert and key always; client CA and client_auth_type only for mTLS.
	// With cert-manager: server secret at /etc/prometheus-web-tls (server cert + ca.crt for client verification).
	// With user secret: single secret at /etc/prometheus-tls.
	tlsCfg := &prometheusTLSServerConfig{}
	if isPrometheusServerCertManagerTLSEnabled(bindplane) {
		tlsCfg.CertFile = prometheusWebServerTLSMountPath + "/tls.crt"
		tlsCfg.KeyFile = prometheusWebServerTLSMountPath + "/tls.key"
		if isPrometheusServerMTLSEnabled(bindplane) {
			tlsCfg.ClientCAFile = prometheusWebServerTLSMountPath + "/ca.crt"
			tlsCfg.ClientAuthType = "RequireAndVerifyClientCert"
		}
	} else {
		tlsCfg.CertFile = prometheusTLSMountPath + "/tls.crt"
		tlsCfg.KeyFile = prometheusTLSMountPath + "/tls.key"
		if isPrometheusServerMTLSEnabled(bindplane) {
			tlsCfg.ClientCAFile = prometheusTLSMountPath + "/ca.crt"
			tlsCfg.ClientAuthType = "RequireAndVerifyClientCert"
		}
	}
	cfg.TLSServerConfig = tlsCfg
	mergedWebConfig, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshal web config: %w", err)
	}
	probeHTTPYAML, err := buildPrometheusProbeHTTPConfigYAML(bindplane)
	if err != nil {
		return fmt.Errorf("build probe-http config: %w", err)
	}

	cmName := getResourceName(bindplane, prometheusWebConfigConfigMapSuffix)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: bindplane.Namespace,
			Labels:    getLabels(bindplane, prometheusWebConfigConfigMapSuffix),
		},
		Data: map[string]string{
			prometheusWebConfigFileName: string(mergedWebConfig),
			prometheusProbeHTTPFileName: string(probeHTTPYAML),
		},
	}
	if err := controllerutil.SetControllerReference(bindplane, cm, r.Scheme); err != nil {
		return err
	}
	existingCM := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: cmName, Namespace: bindplane.Namespace}, existingCM)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if errors.IsNotFound(err) {
		log.Info("Creating Prometheus web-config ConfigMap", "name", cmName, "namespace", bindplane.Namespace)
		return r.Create(ctx, cm)
	}
	existingCM.Data = cm.Data
	existingCM.Labels = cm.Labels
	return r.Update(ctx, existingCM)
}

func (r *BindplaneReconciler) prometheusServiceAccount(bindplane *bindplanev1alpha1.Bindplane) *corev1.ServiceAccount {
	return newServiceAccount(bindplane, prometheusComponent)
}

func (r *BindplaneReconciler) prometheusStatefulSet(bindplane *bindplanev1alpha1.Bindplane) *appsv1.StatefulSet {
	replicas := int32(1)
	labels := getLabels(bindplane, prometheusComponent)
	selectorLabels := getSelectorLabels(bindplane, prometheusComponent)
	serviceName := getResourceName(bindplane, prometheusComponent)
	volumes := getPrometheusVolumes(bindplane)
	volumeMounts := getPrometheusVolumeMounts(bindplane)

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: bindplane.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:            &replicas,
			ServiceName:         serviceName,
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Template: mergePodTemplateSpec(
				corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: selectorLabels,
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: serviceName,
						Volumes:            volumes,
						SecurityContext: &corev1.PodSecurityContext{
							FSGroup:    new(defaultRunAsGroup),
							RunAsGroup: new(defaultRunAsGroup),
							RunAsUser:  new(defaultRunAsUser),
						},
						Affinity: getPrometheusAffinity(bindplane),
						Containers: []corev1.Container{
							{
								Name:  prometheusContainerName,
								Image: prometheusImage,
								Args:  []string{"--web.config.file=" + prometheusWebConfigMountPath + "/" + prometheusWebConfigFileName},
								Ports: []corev1.ContainerPort{
									{
										Name:          prometheusHTTPPortName,
										ContainerPort: prometheusHTTPPort,
										Protocol:      corev1.ProtocolTCP,
									},
								},
								Env: getKubernetesEnvVars(prometheusContainerName),
								Resources: corev1.ResourceRequirements{
									Limits: corev1.ResourceList{
										corev1.ResourceMemory: resource.MustParse("500Mi"),
									},
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("250m"),
										corev1.ResourceMemory: resource.MustParse("500Mi"),
									},
								},
								VolumeMounts: volumeMounts,
								StartupProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										Exec: &corev1.ExecAction{
											Command: getPrometheusProbeCommand(bindplane, "ready"),
										},
									},
									PeriodSeconds:    prometheusProbeStartupPeriodSeconds,
									FailureThreshold: prometheusProbeStartupFailureThreshold,
									SuccessThreshold: prometheusProbeStartupSuccessThreshold,
									TimeoutSeconds:   prometheusProbeStartupTimeoutSeconds,
								},
								LivenessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										Exec: &corev1.ExecAction{
											Command: getPrometheusProbeCommand(bindplane, "healthy"),
										},
									},
									PeriodSeconds:    prometheusProbeLivenessPeriodSeconds,
									FailureThreshold: prometheusProbeLivenessFailureThreshold,
									SuccessThreshold: prometheusProbeLivenessSuccessThreshold,
									TimeoutSeconds:   prometheusProbeLivenessTimeoutSeconds,
								},
								ReadinessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										Exec: &corev1.ExecAction{
											Command: getPrometheusProbeCommand(bindplane, "ready"),
										},
									},
									PeriodSeconds:    prometheusProbeReadinessPeriodSeconds,
									FailureThreshold: prometheusProbeReadinessFailureThreshold,
									SuccessThreshold: prometheusProbeReadinessSuccessThreshold,
									TimeoutSeconds:   prometheusProbeReadinessTimeoutSeconds,
								},
								SecurityContext: newContainerSecurityContext(WithRunAsUser(defaultRunAsUser)),
								ImagePullPolicy: corev1.PullIfNotPresent,
							},
						},
						TerminationGracePeriodSeconds: new(defaultTerminationGracePeriodSeconds),
					},
				},
				getPrometheusPodTemplateSpec(bindplane),
			),
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				getPrometheusVolumeClaimTemplate(bindplane, labels),
			},
			PersistentVolumeClaimRetentionPolicy: &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
				WhenDeleted: appsv1.RetainPersistentVolumeClaimRetentionPolicyType,
				WhenScaled:  appsv1.RetainPersistentVolumeClaimRetentionPolicyType,
			},
		},
	}
}

func (r *BindplaneReconciler) prometheusService(bindplane *bindplanev1alpha1.Bindplane) *corev1.Service {
	return newService(bindplane, prometheusComponent, WithPort(prometheusHTTPPortName, prometheusHTTPPort))
}

// getPrometheusVolumes returns volumes for the Prometheus container. When server TLS is enabled (spec.prometheus.tls),
// uses ConfigMap for web config and adds server cert Secret volume (operator-created or user-defined).
func getPrometheusVolumes(bindplane *bindplanev1alpha1.Bindplane) []corev1.Volume {
	if !isPrometheusServerTLSEnabled(bindplane) {
		return []corev1.Volume{
			{
				Name: prometheusWebConfigVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: getResourceName(bindplane, prometheusBasicAuthSecretSuffix),
						Items: []corev1.KeyToPath{
							{Key: prometheusBasicAuthSecretKeyWeb, Path: prometheusWebConfigFileName},
						},
					},
				},
			},
		}
	}
	tls := bindplane.Spec.Prometheus.TLS
	// Server cert volume: cert-manager (server + probe client) or user secret
	var secretVols []corev1.Volume
	if tls.CertManager != nil && tls.CertManager.Name != "" {
		secretVols = []corev1.Volume{
			{
				Name: prometheusWebServerTLSVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: getResourceName(bindplane, prometheusRemoteWriteServerCertSuffix),
					},
				},
			},
			{
				Name: prometheusProbeClientTLSVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: getResourceName(bindplane, prometheusProbeClientCertSuffix),
					},
				},
			},
			{
				Name: prometheusProbeAuthVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: getResourceName(bindplane, prometheusBasicAuthSecretSuffix),
						Items: []corev1.KeyToPath{
							{Key: prometheusBasicAuthSecretKeyUser, Path: "username"},
							{Key: prometheusBasicAuthSecretKeyPass, Path: "password"},
						},
					},
				},
			},
		}
	} else {
		// User-defined secret; map keys to tls.crt, tls.key; optionally ca.crt when CAKey set (mTLS).
		certKey, keyKey := tls.CertKey, tls.KeyKey
		if certKey == "" {
			certKey = "tls.crt"
		}
		if keyKey == "" {
			keyKey = "tls.key"
		}
		items := []corev1.KeyToPath{
			{Key: certKey, Path: "tls.crt"},
			{Key: keyKey, Path: "tls.key"},
		}
		if tls.CAKey != "" {
			items = append(items, corev1.KeyToPath{Key: tls.CAKey, Path: "ca.crt"})
		}
		secretVols = []corev1.Volume{
			{
				Name: prometheusTLSVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: tls.SecretName,
						Items:      items,
					},
				},
			},
			{
				Name: prometheusProbeAuthVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: getResourceName(bindplane, prometheusBasicAuthSecretSuffix),
						Items: []corev1.KeyToPath{
							{Key: prometheusBasicAuthSecretKeyUser, Path: "username"},
							{Key: prometheusBasicAuthSecretKeyPass, Path: "password"},
						},
					},
				},
			},
		}
	}
	return append([]corev1.Volume{
		{
			Name: prometheusWebConfigVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: getResourceName(bindplane, prometheusWebConfigConfigMapSuffix),
					},
				},
			},
		},
	}, secretVols...)
}

// getPrometheusVolumeMounts returns volume mounts for the Prometheus container.
// When server TLS is enabled, the web ConfigMap is mounted at /etc/prometheus so both web.yml and probe-http.yml are available.
func getPrometheusVolumeMounts(bindplane *bindplanev1alpha1.Bindplane) []corev1.VolumeMount {
	mounts := []corev1.VolumeMount{
		{Name: getResourceName(bindplane, prometheusDataVolumeSuffix), MountPath: "/prometheus"},
	}
	if isPrometheusServerTLSEnabled(bindplane) {
		// ConfigMap keys mounted with subPath so only web.yml and probe-http.yml are present at /etc/prometheus.
		mounts = append(mounts, corev1.VolumeMount{
			Name:      prometheusWebConfigVolumeName,
			MountPath: prometheusWebConfigMountPath + "/" + prometheusWebConfigFileName,
			SubPath:   prometheusWebConfigFileName,
			ReadOnly:  true,
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      prometheusWebConfigVolumeName,
			MountPath: prometheusWebConfigMountPath + "/" + prometheusProbeHTTPFileName,
			SubPath:   prometheusProbeHTTPFileName,
			ReadOnly:  true,
		})
		if isPrometheusServerCertManagerTLSEnabled(bindplane) {
			mounts = append(mounts, corev1.VolumeMount{Name: prometheusWebServerTLSVolumeName, MountPath: prometheusWebServerTLSMountPath, ReadOnly: true})
			mounts = append(mounts, corev1.VolumeMount{Name: prometheusProbeClientTLSVolumeName, MountPath: prometheusProbeClientTLSMountPath, ReadOnly: true})
		} else {
			mounts = append(mounts, corev1.VolumeMount{Name: prometheusTLSVolumeName, MountPath: prometheusTLSMountPath, ReadOnly: true})
		}
		mounts = append(mounts, corev1.VolumeMount{Name: prometheusProbeAuthVolumeName, MountPath: prometheusProbeAuthMountPath, ReadOnly: true})
	} else {
		// Secret with web.yml only (subPath so only that key is mounted)
		mounts = append(mounts, corev1.VolumeMount{
			Name:      prometheusWebConfigVolumeName,
			MountPath: prometheusWebConfigMountPath + "/" + prometheusWebConfigFileName,
			SubPath:   prometheusWebConfigFileName,
			ReadOnly:  true,
		})
	}
	return mounts
}

// getPrometheusProbeCommand returns the exec command for startup/liveness/readiness probes.
// check is "ready" (for /-/ready) or "healthy" (for /-/healthy). Uses HTTP when server TLS is disabled,
// HTTPS with probe-http.yml when TLS or mTLS is enabled.
func getPrometheusProbeCommand(bindplane *bindplanev1alpha1.Bindplane, check string) []string {
	url := "http://127.0.0.1:" + fmt.Sprintf("%d", prometheusHTTPPort)
	configFile := ""
	if isPrometheusServerTLSEnabled(bindplane) {
		url = "https://127.0.0.1:" + fmt.Sprintf("%d", prometheusHTTPPort)
		configFile = " --http.config.file=" + prometheusWebConfigMountPath + "/" + prometheusProbeHTTPFileName
	}
	script := "promtool check " + check + " --url=" + url + configFile
	return []string{"/bin/sh", "-ec", script}
}

// getPrometheusAffinity returns the affinity configuration for Prometheus pods
// This is a fallback for when user doesn't provide podTemplate - will be overridden by mergePodTemplateSpec
func getPrometheusAffinity(bindplane *bindplanev1alpha1.Bindplane) *corev1.Affinity {
	if bindplane.Spec.Prometheus != nil && bindplane.Spec.Prometheus.PodTemplate != nil {
		return bindplane.Spec.Prometheus.PodTemplate.Spec.Affinity
	}
	return nil
}

// getPrometheusPodTemplateSpec returns the user-provided pod template spec for Prometheus
func getPrometheusPodTemplateSpec(bindplane *bindplanev1alpha1.Bindplane) *bindplanev1alpha1.PodTemplateSpec {
	if bindplane.Spec.Prometheus != nil {
		return bindplane.Spec.Prometheus.PodTemplate
	}
	return nil
}

// getPrometheusVolumeClaimTemplate returns the PersistentVolumeClaim template for Prometheus
func getPrometheusVolumeClaimTemplate(bindplane *bindplanev1alpha1.Bindplane, labels map[string]string) corev1.PersistentVolumeClaim {
	volumeName := getResourceName(bindplane, prometheusDataVolumeSuffix)

	// Default PVC spec
	defaultSpec := corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		Resources: corev1.VolumeResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("60Gi"),
			},
		},
	}

	// Use user-provided storage configuration if available
	if bindplane.Spec.Prometheus != nil && bindplane.Spec.Prometheus.Storage != nil && bindplane.Spec.Prometheus.Storage.VolumeClaimTemplate != nil {
		userTemplate := bindplane.Spec.Prometheus.Storage.VolumeClaimTemplate

		// Start with user-provided spec
		pvcSpec := userTemplate.Spec.DeepCopy()

		// Build metadata
		pvcMeta := metav1.ObjectMeta{
			Name:   volumeName,
			Labels: labels,
		}

		// Merge user-provided metadata if present
		if userTemplate.Metadata != nil {
			if userTemplate.Metadata.Labels != nil {
				if pvcMeta.Labels == nil {
					pvcMeta.Labels = make(map[string]string)
				}
				maps.Copy(pvcMeta.Labels, userTemplate.Metadata.Labels)
			}
			if userTemplate.Metadata.Annotations != nil {
				pvcMeta.Annotations = make(map[string]string)
				maps.Copy(pvcMeta.Annotations, userTemplate.Metadata.Annotations)
			}
		}

		return corev1.PersistentVolumeClaim{
			ObjectMeta: pvcMeta,
			Spec:       *pvcSpec,
		}
	}

	// Return default PVC
	return corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:   volumeName,
			Labels: labels,
		},
		Spec: defaultSpec,
	}
}
