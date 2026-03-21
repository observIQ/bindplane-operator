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
	// tsdbComponent is the component name for Bindplane's TSDB (Prometheus by default)
	tsdbComponent = "tsdb"
	// tsdbContainerName is the container name for the TSDB StatefulSet
	tsdbContainerName = "tsdb"
	// tsdbDataVolumeSuffix is the suffix for TSDB data volume names
	tsdbDataVolumeSuffix = "tsdb-data"
	// tsdbHTTPPort is the HTTP port for Prometheus
	tsdbHTTPPort = 9090
	// tsdbHTTPPortName is the name of the HTTP port for Prometheus
	tsdbHTTPPortName = "http"

	// TSDB basic auth (operator-generated secret)
	tsdbBasicAuthSecretSuffix  = "tsdb-basic-auth" // #nosec G101 -- secret name suffix, not a credential
	tsdbBasicAuthUsername      = "tsdb"
	tsdbBasicAuthSecretKeyUser = "username"
	tsdbBasicAuthSecretKeyPass = "password"
	tsdbBasicAuthSecretKeyWeb  = "web-config"
	tsdbWebConfigVolumeName    = "tsdb-web-config"
	// Prometheus entrypoint reads --web.config.file from /etc/prometheus/web.yml.
	// Keep this path even after TSDB renaming so TLS/basic-auth settings are applied.
	tsdbWebConfigMountPath = "/etc/prometheus"
	tsdbWebConfigFileName  = "web.yml"
	tsdbProbeHTTPFileName  = "probe-http.yml"
	// Server cert mount when internal TLS is enabled for remote write
	tsdbTLSVolumeName            = "tsdb-tls"
	tsdbTLSMountPath             = "/etc/tsdb-tls"
	tsdbWebConfigConfigMapSuffix = "tsdb-web-config"
	// Cert-manager: separate server cert (web) and probe client cert (promtool) so EKU matches (ServerAuth vs ClientAuth).
	tsdbWebServerTLSVolumeName   = "tsdb-web-server-tls"
	tsdbWebServerTLSMountPath    = "/etc/tsdb-web-tls"
	tsdbProbeClientTLSVolumeName = "tsdb-probe-client-tls"
	tsdbProbeClientTLSMountPath  = "/etc/tsdb-probe-client"
	// Probe basic auth: existing basic-auth secret mounted as files for promtool username_file/password_file.
	tsdbProbeAuthVolumeName = "tsdb-probe-auth"
	tsdbProbeAuthMountPath  = "/etc/tsdb-probe-auth"

	// Prometheus exec probe timing (matches reference out.yaml: startup/readiness use /-/ready, liveness uses /-/healthy).
	tsdbProbeStartupFailureThreshold   int32 = 60
	tsdbProbeStartupPeriodSeconds      int32 = 15
	tsdbProbeStartupSuccessThreshold   int32 = 1
	tsdbProbeStartupTimeoutSeconds     int32 = 3
	tsdbProbeReadinessFailureThreshold int32 = 3
	tsdbProbeReadinessPeriodSeconds    int32 = 5
	tsdbProbeReadinessSuccessThreshold int32 = 1
	tsdbProbeReadinessTimeoutSeconds   int32 = 3
	tsdbProbeLivenessFailureThreshold  int32 = 6
	tsdbProbeLivenessPeriodSeconds     int32 = 5
	tsdbProbeLivenessSuccessThreshold  int32 = 1
	tsdbProbeLivenessTimeoutSeconds    int32 = 3
)

// tsdbWebConfig is the structure of Prometheus web.config.file (basic auth + TLS).
// See https://prometheus.io/docs/prometheus/latest/configuration/https/
// We use json tags because sigs.k8s.io/yaml uses JSON for marshaling; keys must be basic_auth_users, tls_server_config.
type tsdbWebConfig struct {
	BasicAuthUsers  map[string]string    `json:"basic_auth_users,omitempty"`
	TLSServerConfig *tsdbTLSServerConfig `json:"tls_server_config,omitempty"`
}

// tsdbTLSServerConfig is the tls_server_config section of Prometheus web config.
// ClientCAFile and ClientAuthType are only set for mTLS (client cert verification); omit for TLS-only.
type tsdbTLSServerConfig struct {
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

// isTSDBRemoteEnabled returns true when Bindplane should use a user-managed remote Prometheus backend.
func isTSDBRemoteEnabled(bindplane *bindplanev1alpha1.Bindplane) bool {
	p := bindplane.Spec.Config.TSDB
	return p != nil && p.Remote != nil && p.Remote.Enable
}

// isTSDBServerTLSEnabled returns true when the Prometheus server (StatefulSet) should serve TLS (spec.prometheus.tls with certManager or secretName).
func isTSDBServerTLSEnabled(bindplane *bindplanev1alpha1.Bindplane) bool {
	p := bindplane.Spec.TSDB
	if p == nil || p.TLS == nil {
		return false
	}
	return (p.TLS.CertManager != nil && p.TLS.CertManager.Name != "") || p.TLS.SecretName != ""
}

// isTSDBServerMTLSEnabled returns true when the Prometheus server should require and verify client certs (mTLS).
// True when: (cert-manager server and cert-manager client both enabled) or (user secret with CAKey set).
func isTSDBServerMTLSEnabled(bindplane *bindplanev1alpha1.Bindplane) bool {
	p := bindplane.Spec.TSDB
	if p == nil || p.TLS == nil {
		return false
	}
	tls := p.TLS
	if tls.CertManager != nil && tls.CertManager.Name != "" {
		return isTSDBClientCertManagerTLSEnabled(bindplane)
	}
	// User-defined secret: mTLS only when user provides a CA for client verification
	return tls.CAKey != ""
}

// getTSDBProbeServerName returns the server_name for promtool probe (must match a SAN/CN of the Prometheus server cert).
// Prefer the service FQDN so verification works when connecting to 127.0.0.1; fallback to localhost.
func getTSDBProbeServerName(bindplane *bindplanev1alpha1.Bindplane) string {
	names := getTSDBServerCertDNSNames(bindplane)
	if len(names) > 0 {
		return names[0] // e.g. my-bp-prometheus.default.svc.cluster.local
	}
	return "localhost"
}

// isTSDBServerProbeTLSWithCA returns true when the server TLS secret has a CA file (for probe to verify server cert).
func isTSDBServerProbeTLSWithCA(bindplane *bindplanev1alpha1.Bindplane) bool {
	p := bindplane.Spec.TSDB
	if p == nil || p.TLS == nil {
		return false
	}
	if p.TLS.CertManager != nil && p.TLS.CertManager.Name != "" {
		return true // cert-manager secrets include ca.crt
	}
	return p.TLS.CAKey != "" // user secret with CA
}

// buildTSDBProbeHTTPConfigYAML returns the probe-http.yml content for promtool exec probes when server TLS is enabled.
// With cert-manager: use probe CLIENT cert (ClientAuth) and CA from server secret; server_name must match server cert SAN.
// With user secret: use same paths as web (single secret); may use insecure_skip_verify if no CA.
// basic_auth uses username_file and password_file (existing basic-auth secret mounted at tsdbProbeAuthMountPath).
func buildTSDBProbeHTTPConfigYAML(bindplane *bindplanev1alpha1.Bindplane) ([]byte, error) {
	cfg := promtoolProbeHTTPConfig{
		BasicAuth: &promtoolProbeBasicAuthConfig{
			UsernameFile: tsdbProbeAuthMountPath + "/username",
			PasswordFile: tsdbProbeAuthMountPath + "/password",
		},
		TLSConfig: &promtoolProbeTLSConfig{
			ServerName: getTSDBProbeServerName(bindplane),
		},
	}
	tlsCfg := cfg.TLSConfig
	if isTSDBServerCertManagerTLSEnabled(bindplane) {
		tlsCfg.CAFile = tsdbWebServerTLSMountPath + "/ca.crt"
		tlsCfg.CertFile = tsdbProbeClientTLSMountPath + "/tls.crt"
		tlsCfg.KeyFile = tsdbProbeClientTLSMountPath + "/tls.key"
	} else if isTSDBServerProbeTLSWithCA(bindplane) {
		tlsCfg.CAFile = tsdbTLSMountPath + "/ca.crt"
		tlsCfg.CertFile = tsdbTLSMountPath + "/tls.crt"
		tlsCfg.KeyFile = tsdbTLSMountPath + "/tls.key"
	} else {
		tlsCfg.InsecureSkipVerify = true
	}
	return yaml.Marshal(&cfg)
}

// reconcileTSDB reconciles all Prometheus resources
func (r *BindplaneReconciler) reconcileTSDB(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
	if isTSDBRemoteEnabled(bindplane) {
		log.Info("Skipping operator-managed TSDB resources because spec.config.tsdb.remote.enable is true")
		return nil
	}
	// Reconcile Prometheus basic auth Secret first (create-only) so it exists for StatefulSet and Bindplane pods
	if err := r.reconcileTSDBBasicAuthSecret(ctx, bindplane, log); err != nil {
		return err
	}
	// Reconcile web-config ConfigMap (basic auth + optional tls_server_config) — always needed for probe credentials.
	if err := r.reconcileTSDBWebConfigConfigMap(ctx, bindplane, log); err != nil {
		return err
	}

	// Reconcile ServiceAccount
	sa := r.tsdbServiceAccount(bindplane)
	if err := r.reconcileServiceAccount(ctx, bindplane, sa, log); err != nil {
		return err
	}

	// Reconcile StatefulSet
	statefulSet := r.tsdbStatefulSet(bindplane)
	if err := r.reconcileStatefulSet(ctx, bindplane, statefulSet, log); err != nil {
		return err
	}

	// Reconcile Service
	service := r.tsdbService(bindplane)
	if err := r.reconcileService(ctx, bindplane, service, log); err != nil {
		return err
	}

	return nil
}

// generateTSDBBasicAuthSecretData returns secret data (username, password, web-config YAML).
// Password is 24 random bytes base64-encoded (32 chars); web-config is Prometheus basic_auth_users YAML with bcrypt hash.
func generateTSDBBasicAuthSecretData() (map[string][]byte, error) {
	passwordBytes := make([]byte, 24)
	if _, err := rand.Read(passwordBytes); err != nil {
		return nil, fmt.Errorf("generate password: %w", err)
	}
	password := base64.URLEncoding.EncodeToString(passwordBytes) // 32 chars

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("bcrypt password: %w", err)
	}
	cfg := tsdbWebConfig{
		BasicAuthUsers: map[string]string{tsdbBasicAuthUsername: string(hash)},
	}
	webConfigBytes, err := yaml.Marshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal web config: %w", err)
	}

	return map[string][]byte{
		tsdbBasicAuthSecretKeyUser: []byte(tsdbBasicAuthUsername),
		tsdbBasicAuthSecretKeyPass: []byte(password),
		tsdbBasicAuthSecretKeyWeb:  webConfigBytes,
	}, nil
}

// reconcileTSDBBasicAuthSecret creates the Prometheus basic auth Secret if it does not exist.
// Existing Secret data is never updated to avoid rotating credentials.
func (r *BindplaneReconciler) reconcileTSDBBasicAuthSecret(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
	secretName := getResourceName(bindplane, tsdbBasicAuthSecretSuffix)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: bindplane.Namespace,
			Labels:    getLabels(bindplane, tsdbBasicAuthSecretSuffix),
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

	data, err := generateTSDBBasicAuthSecretData()
	if err != nil {
		return err
	}
	secret.Data = data

	log.Info("Creating Prometheus basic auth Secret", "name", secretName, "namespace", bindplane.Namespace)
	return r.Create(ctx, secret)
}

// reconcileTSDBWebConfigConfigMap creates or updates a ConfigMap with web config (basic auth + tls_server_config)
// when internal TLS is enabled for Prometheus remote write. Reads the basic auth Secret to get basic_auth_users.
func (r *BindplaneReconciler) reconcileTSDBWebConfigConfigMap(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
	secretName := getResourceName(bindplane, tsdbBasicAuthSecretSuffix)
	existingSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: bindplane.Namespace}, existingSecret); err != nil {
		return fmt.Errorf("get basic auth secret for web config: %w", err)
	}
	webConfigBytes, ok := existingSecret.Data[tsdbBasicAuthSecretKeyWeb]
	if !ok {
		return fmt.Errorf("basic auth secret missing key %q", tsdbBasicAuthSecretKeyWeb)
	}
	var cfg tsdbWebConfig
	if err := yaml.Unmarshal(webConfigBytes, &cfg); err != nil {
		return fmt.Errorf("unmarshal web config from secret: %w", err)
	}
	// TLS server config: only set when TLS is enabled. Without TLS, web.yml contains basic auth only.
	// With cert-manager: server secret at /etc/prometheus-web-tls (server cert + ca.crt for client verification).
	// With user secret: single secret at /etc/prometheus-tls.
	if isTSDBServerTLSEnabled(bindplane) {
		tlsCfg := &tsdbTLSServerConfig{}
		if isTSDBServerCertManagerTLSEnabled(bindplane) {
			tlsCfg.CertFile = tsdbWebServerTLSMountPath + "/tls.crt"
			tlsCfg.KeyFile = tsdbWebServerTLSMountPath + "/tls.key"
			if isTSDBServerMTLSEnabled(bindplane) {
				tlsCfg.ClientCAFile = tsdbWebServerTLSMountPath + "/ca.crt"
				tlsCfg.ClientAuthType = "RequireAndVerifyClientCert"
			}
		} else {
			tlsCfg.CertFile = tsdbTLSMountPath + "/tls.crt"
			tlsCfg.KeyFile = tsdbTLSMountPath + "/tls.key"
			if isTSDBServerMTLSEnabled(bindplane) {
				tlsCfg.ClientCAFile = tsdbTLSMountPath + "/ca.crt"
				tlsCfg.ClientAuthType = "RequireAndVerifyClientCert"
			}
		}
		cfg.TLSServerConfig = tlsCfg
	}
	mergedWebConfig, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshal web config: %w", err)
	}
	probeHTTPYAML, err := buildTSDBProbeHTTPConfigYAML(bindplane)
	if err != nil {
		return fmt.Errorf("build probe-http config: %w", err)
	}

	cmName := getResourceName(bindplane, tsdbWebConfigConfigMapSuffix)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: bindplane.Namespace,
			Labels:    getLabels(bindplane, tsdbWebConfigConfigMapSuffix),
		},
		Data: map[string]string{
			tsdbWebConfigFileName: string(mergedWebConfig),
			tsdbProbeHTTPFileName: string(probeHTTPYAML),
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

func (r *BindplaneReconciler) tsdbServiceAccount(bindplane *bindplanev1alpha1.Bindplane) *corev1.ServiceAccount {
	return newServiceAccount(bindplane, tsdbComponent)
}

func (r *BindplaneReconciler) tsdbStatefulSet(bindplane *bindplanev1alpha1.Bindplane) *appsv1.StatefulSet {
	replicas := int32(1)
	labels := getLabels(bindplane, tsdbComponent)
	selectorLabels := getSelectorLabels(bindplane, tsdbComponent)
	serviceName := getResourceName(bindplane, tsdbComponent)
	volumes := getTSDBVolumes(bindplane)
	volumeMounts := getTSDBVolumeMounts(bindplane)

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
						Affinity: getTSDBAffinity(bindplane),
						Containers: []corev1.Container{
							{
								Name:  tsdbContainerName,
								Image: getTSDBImage(bindplane),
								Args:  []string{"--web.config.file=" + tsdbWebConfigMountPath + "/" + tsdbWebConfigFileName},
								Ports: []corev1.ContainerPort{
									{
										Name:          tsdbHTTPPortName,
										ContainerPort: tsdbHTTPPort,
										Protocol:      corev1.ProtocolTCP,
									},
								},
								Env: getKubernetesEnvVars(tsdbContainerName),
								Resources: corev1.ResourceRequirements{
									Limits: corev1.ResourceList{
										corev1.ResourceMemory: resource.MustParse("2048Mi"),
									},
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("1000m"),
										corev1.ResourceMemory: resource.MustParse("2048Mi"),
									},
								},
								VolumeMounts: volumeMounts,
								StartupProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										Exec: &corev1.ExecAction{
											Command: getTSDBProbeCommand(bindplane, "ready"),
										},
									},
									PeriodSeconds:    tsdbProbeStartupPeriodSeconds,
									FailureThreshold: tsdbProbeStartupFailureThreshold,
									SuccessThreshold: tsdbProbeStartupSuccessThreshold,
									TimeoutSeconds:   tsdbProbeStartupTimeoutSeconds,
								},
								LivenessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										Exec: &corev1.ExecAction{
											Command: getTSDBProbeCommand(bindplane, "healthy"),
										},
									},
									PeriodSeconds:    tsdbProbeLivenessPeriodSeconds,
									FailureThreshold: tsdbProbeLivenessFailureThreshold,
									SuccessThreshold: tsdbProbeLivenessSuccessThreshold,
									TimeoutSeconds:   tsdbProbeLivenessTimeoutSeconds,
								},
								ReadinessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										Exec: &corev1.ExecAction{
											Command: getTSDBProbeCommand(bindplane, "ready"),
										},
									},
									PeriodSeconds:    tsdbProbeReadinessPeriodSeconds,
									FailureThreshold: tsdbProbeReadinessFailureThreshold,
									SuccessThreshold: tsdbProbeReadinessSuccessThreshold,
									TimeoutSeconds:   tsdbProbeReadinessTimeoutSeconds,
								},
								SecurityContext: newContainerSecurityContext(WithRunAsUser(defaultRunAsUser)),
								ImagePullPolicy: corev1.PullIfNotPresent,
							},
						},
						TerminationGracePeriodSeconds: new(defaultTerminationGracePeriodSeconds),
					},
				},
				getTSDBPodTemplateSpec(bindplane),
			),
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				getTSDBVolumeClaimTemplate(bindplane, labels),
			},
			PersistentVolumeClaimRetentionPolicy: &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
				WhenDeleted: appsv1.RetainPersistentVolumeClaimRetentionPolicyType,
				WhenScaled:  appsv1.RetainPersistentVolumeClaimRetentionPolicyType,
			},
		},
	}
}

func (r *BindplaneReconciler) tsdbService(bindplane *bindplanev1alpha1.Bindplane) *corev1.Service {
	return newService(bindplane, tsdbComponent, WithPort(tsdbHTTPPortName, tsdbHTTPPort))
}

// getTSDBVolumes returns volumes for the Prometheus container. When server TLS is enabled (spec.prometheus.tls),
// uses ConfigMap for web config and adds server cert Secret volume (operator-created or user-defined).
func getTSDBVolumes(bindplane *bindplanev1alpha1.Bindplane) []corev1.Volume {
	if !isTSDBServerTLSEnabled(bindplane) {
		return []corev1.Volume{
			{
				Name: tsdbWebConfigVolumeName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: getResourceName(bindplane, tsdbWebConfigConfigMapSuffix),
						},
					},
				},
			},
			{
				Name: tsdbProbeAuthVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: getResourceName(bindplane, tsdbBasicAuthSecretSuffix),
						Items: []corev1.KeyToPath{
							{Key: tsdbBasicAuthSecretKeyUser, Path: "username"},
							{Key: tsdbBasicAuthSecretKeyPass, Path: "password"},
						},
					},
				},
			},
		}
	}
	tls := bindplane.Spec.TSDB.TLS
	// Server cert volume: cert-manager (server + probe client) or user secret
	var secretVols []corev1.Volume
	if tls.CertManager != nil && tls.CertManager.Name != "" {
		secretVols = []corev1.Volume{
			{
				Name: tsdbWebServerTLSVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: getResourceName(bindplane, tsdbRemoteWriteServerCertSuffix),
					},
				},
			},
			{
				Name: tsdbProbeClientTLSVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: getResourceName(bindplane, tsdbProbeClientCertSuffix),
					},
				},
			},
			{
				Name: tsdbProbeAuthVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: getResourceName(bindplane, tsdbBasicAuthSecretSuffix),
						Items: []corev1.KeyToPath{
							{Key: tsdbBasicAuthSecretKeyUser, Path: "username"},
							{Key: tsdbBasicAuthSecretKeyPass, Path: "password"},
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
				Name: tsdbTLSVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: tls.SecretName,
						Items:      items,
					},
				},
			},
			{
				Name: tsdbProbeAuthVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: getResourceName(bindplane, tsdbBasicAuthSecretSuffix),
						Items: []corev1.KeyToPath{
							{Key: tsdbBasicAuthSecretKeyUser, Path: "username"},
							{Key: tsdbBasicAuthSecretKeyPass, Path: "password"},
						},
					},
				},
			},
		}
	}
	return append([]corev1.Volume{
		{
			Name: tsdbWebConfigVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: getResourceName(bindplane, tsdbWebConfigConfigMapSuffix),
					},
				},
			},
		},
	}, secretVols...)
}

// getTSDBVolumeMounts returns volume mounts for the Prometheus container.
// Use subPath for /etc/prometheus files so we don't shadow the bundled prometheus.yml file.
func getTSDBVolumeMounts(bindplane *bindplanev1alpha1.Bindplane) []corev1.VolumeMount {
	mounts := []corev1.VolumeMount{
		{Name: getResourceName(bindplane, tsdbDataVolumeSuffix), MountPath: "/prometheus"},
	}
	if isTSDBServerTLSEnabled(bindplane) {
		// ConfigMap keys mounted with subPath so only web.yml and probe-http.yml are present at /etc/prometheus.
		mounts = append(mounts, corev1.VolumeMount{
			Name:      tsdbWebConfigVolumeName,
			MountPath: tsdbWebConfigMountPath + "/" + tsdbWebConfigFileName,
			SubPath:   tsdbWebConfigFileName,
			ReadOnly:  true,
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      tsdbWebConfigVolumeName,
			MountPath: tsdbWebConfigMountPath + "/" + tsdbProbeHTTPFileName,
			SubPath:   tsdbProbeHTTPFileName,
			ReadOnly:  true,
		})
		if isTSDBServerCertManagerTLSEnabled(bindplane) {
			mounts = append(mounts, corev1.VolumeMount{Name: tsdbWebServerTLSVolumeName, MountPath: tsdbWebServerTLSMountPath, ReadOnly: true})
			mounts = append(mounts, corev1.VolumeMount{Name: tsdbProbeClientTLSVolumeName, MountPath: tsdbProbeClientTLSMountPath, ReadOnly: true})
		} else {
			mounts = append(mounts, corev1.VolumeMount{Name: tsdbTLSVolumeName, MountPath: tsdbTLSMountPath, ReadOnly: true})
		}
		mounts = append(mounts, corev1.VolumeMount{Name: tsdbProbeAuthVolumeName, MountPath: tsdbProbeAuthMountPath, ReadOnly: true})
	} else {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      tsdbWebConfigVolumeName,
			MountPath: tsdbWebConfigMountPath + "/" + tsdbWebConfigFileName,
			SubPath:   tsdbWebConfigFileName,
			ReadOnly:  true,
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      tsdbWebConfigVolumeName,
			MountPath: tsdbWebConfigMountPath + "/" + tsdbProbeHTTPFileName,
			SubPath:   tsdbProbeHTTPFileName,
			ReadOnly:  true,
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      tsdbProbeAuthVolumeName,
			MountPath: tsdbProbeAuthMountPath,
			ReadOnly:  true,
		})
	}
	return mounts
}

// getTSDBProbeCommand returns the exec command for startup/liveness/readiness probes.
// check is "ready" (for /-/ready) or "healthy" (for /-/healthy). Uses HTTP when server TLS is disabled,
// HTTPS with probe-http.yml when TLS or mTLS is enabled.
func getTSDBProbeCommand(bindplane *bindplanev1alpha1.Bindplane, check string) []string {
	url := "http://127.0.0.1:" + fmt.Sprintf("%d", tsdbHTTPPort)
	if isTSDBServerTLSEnabled(bindplane) {
		url = "https://127.0.0.1:" + fmt.Sprintf("%d", tsdbHTTPPort)
	}
	configFile := " --http.config.file=" + tsdbWebConfigMountPath + "/" + tsdbProbeHTTPFileName
	script := "promtool check " + check + " --url=" + url + configFile
	return []string{"/bin/sh", "-ec", script}
}

// getTSDBAffinity returns the affinity configuration for Prometheus pods
// This is a fallback for when user doesn't provide podTemplate - will be overridden by mergePodTemplateSpec
func getTSDBAffinity(bindplane *bindplanev1alpha1.Bindplane) *corev1.Affinity {
	if bindplane.Spec.TSDB != nil && bindplane.Spec.TSDB.PodTemplate != nil {
		return bindplane.Spec.TSDB.PodTemplate.Spec.Affinity
	}
	return nil
}

// getTSDBPodTemplateSpec returns the user-provided pod template spec for Prometheus
func getTSDBPodTemplateSpec(bindplane *bindplanev1alpha1.Bindplane) *bindplanev1alpha1.PodTemplateSpec {
	if bindplane.Spec.TSDB != nil {
		return bindplane.Spec.TSDB.PodTemplate
	}
	return nil
}

// getTSDBVolumeClaimTemplate returns the PersistentVolumeClaim template for Prometheus
func getTSDBVolumeClaimTemplate(bindplane *bindplanev1alpha1.Bindplane, labels map[string]string) corev1.PersistentVolumeClaim {
	volumeName := getResourceName(bindplane, tsdbDataVolumeSuffix)

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
	if bindplane.Spec.TSDB != nil && bindplane.Spec.TSDB.Storage != nil && bindplane.Spec.TSDB.Storage.VolumeClaimTemplate != nil {
		userTemplate := bindplane.Spec.TSDB.Storage.VolumeClaimTemplate

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
