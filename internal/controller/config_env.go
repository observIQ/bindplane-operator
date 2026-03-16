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
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

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
	if ev := secretOrValue(bindplaneLDAPBindUserEnvVar, ldap.BindUser, ldap.BindUserSecretRef); ev != nil {
		envVars = append(envVars, *ev)
	}
	if ev := secretOrValue(bindplaneLDAPBindPasswordEnvVar, ldap.BindPassword, ldap.BindPasswordSecretRef); ev != nil {
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
	if ev := secretOrValue(bindplaneOIDCClientIDEnvVar, oidc.ClientID, oidc.ClientIDSecretRef); ev != nil {
		envVars = append(envVars, *ev)
	}
	if ev := secretOrValue(bindplaneOIDCClientSecretEnvVar, oidc.ClientSecret, oidc.ClientSecretSecretRef); ev != nil {
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

// getAuthConfigEnvVars returns env vars for spec.config.auth.
func getAuthConfigEnvVars(auth *bindplanev1alpha1.AuthConfig) []corev1.EnvVar {
	if auth == nil {
		return nil
	}
	var envVars []corev1.EnvVar
	if auth.Type != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAuthTypeEnvVar, Value: auth.Type})
	}
	if auth.SessionsStrictMode {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAuthSessionsStrictModeEnvVar, Value: "true"})
	}
	if ev := secretOrValue(bindplaneUsernameEnvVar, auth.Username, auth.UsernameSecretRef); ev != nil {
		envVars = append(envVars, *ev)
	}
	if ev := secretOrValue(bindplanePasswordEnvVar, auth.Password, auth.PasswordSecretRef); ev != nil {
		envVars = append(envVars, *ev)
	}
	envVars = append(envVars, getLDAPEnvVars(auth.LDAP)...)
	envVars = append(envVars, getOIDCEnvVars(auth.OIDC)...)
	return envVars
}

// getNetworkConfigEnvVars returns env vars for spec.config.network (host, port, remoteURL, tls).
func getNetworkConfigEnvVars(network *bindplanev1alpha1.NetworkConfig, bindplane *bindplanev1alpha1.Bindplane) []corev1.EnvVar {
	var envVars []corev1.EnvVar
	if network != nil {
		if network.Host != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneHostEnvVar, Value: network.Host})
		}
		if network.Port != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplanePortEnvVar, Value: network.Port})
		}
		if network.WebURL != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneWebURLEnvVar, Value: network.WebURL})
		}
		if network.CorsAllowedOrigins != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneCorsAllowedOriginsEnvVar, Value: network.CorsAllowedOrigins})
		}
		if network.TLS != nil {
			tls := network.TLS
			if tls.MinVersion != "" {
				envVars = append(envVars, corev1.EnvVar{Name: bindplaneTLSMinVersionEnvVar, Value: tls.MinVersion})
			}
			// Only set path env vars when the volume is created (secretName + certKey + keyKey)
			if tls.SecretName != "" && tls.CertKey != "" && tls.KeyKey != "" {
				envVars = append(envVars, corev1.EnvVar{Name: bindplaneTLSCertEnvVar, Value: networkTLSMountPath + "/" + tls.CertKey})
				envVars = append(envVars, corev1.EnvVar{Name: bindplaneTLSKeyEnvVar, Value: networkTLSMountPath + "/" + tls.KeyKey})
				if tls.CAKey != "" {
					envVars = append(envVars, corev1.EnvVar{Name: bindplaneTLSCAEnvVar, Value: networkTLSMountPath + "/" + tls.CAKey})
				}
			}
			if tls.SkipVerify {
				envVars = append(envVars, corev1.EnvVar{Name: bindplaneTLSSkipVerifyEnvVar, Value: "true"})
			}
		}
	}
	remoteURL := ""
	if network != nil {
		remoteURL = network.RemoteURL
	}
	if remoteURL == "" {
		remoteURL = fmt.Sprintf("http://%s-%s:%d", bindplane.Name, nodeComponent, nodeHTTPPort)
	}
	envVars = append(envVars, corev1.EnvVar{Name: bindplaneRemoteURLEnvVar, Value: remoteURL})
	return envVars
}

// getPostgresConfigEnvVars returns env vars for spec.config.store.postgres.
func getPostgresConfigEnvVars(pg *bindplanev1alpha1.PostgresConfig) []corev1.EnvVar {
	if pg == nil {
		return nil
	}
	var envVars []corev1.EnvVar
	if pg.Host != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplanePostgresHostEnvVar, Value: pg.Host})
	}
	if pg.Port != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplanePostgresPortEnvVar, Value: pg.Port})
	}
	if pg.ConnectTimeout != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplanePostgresConnectTimeoutEnvVar, Value: pg.ConnectTimeout})
	}
	if pg.StatementTimeout != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplanePostgresStatementTimeoutEnvVar, Value: pg.StatementTimeout})
	}
	if pg.Database != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplanePostgresDatabaseEnvVar, Value: pg.Database})
	}
	sslMode := pg.SSLMode
	if sslMode == "" {
		sslMode = postgresSSLModeDisable
	}
	envVars = append(envVars, corev1.EnvVar{Name: bindplanePostgresSSLModeEnvVar, Value: sslMode})
	if pg.TLS != nil && pg.TLS.SecretName != "" && pg.TLS.CAKey != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplanePostgresSSLRootCertEnvVar, Value: postgresTLSMountPath + "/" + pg.TLS.CAKey})
		if pg.TLS.CertKey != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplanePostgresSSLCertEnvVar, Value: postgresTLSMountPath + "/" + pg.TLS.CertKey})
		}
		if pg.TLS.KeyKey != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplanePostgresSSLKeyEnvVar, Value: postgresTLSMountPath + "/" + pg.TLS.KeyKey})
		}
	}
	if ev := secretOrValue(bindplanePostgresUsernameEnvVar, pg.Username, pg.UsernameSecretRef); ev != nil {
		envVars = append(envVars, *ev)
	}
	if ev := secretOrValue(bindplanePostgresPasswordEnvVar, pg.Password, pg.PasswordSecretRef); ev != nil {
		envVars = append(envVars, *ev)
	}
	if pg.MaxConnections > 0 {
		envVars = append(envVars, corev1.EnvVar{Name: bindplanePostgresMaxConnectionsEnvVar, Value: strconv.Itoa(pg.MaxConnections)})
	}
	if pg.MaxIdleConnections != nil {
		envVars = append(envVars, corev1.EnvVar{Name: bindplanePostgresMaxIdleConnectionsEnvVar, Value: strconv.Itoa(*pg.MaxIdleConnections)})
	}
	if pg.MaxLifetime != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplanePostgresMaxLifetimeEnvVar, Value: pg.MaxLifetime})
	}
	if pg.MaxIdleTime != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplanePostgresMaxIdleTimeEnvVar, Value: pg.MaxIdleTime})
	}
	if pg.Schema != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplanePostgresSchemaEnvVar, Value: pg.Schema})
	}
	return envVars
}

// getTracingConfigEnvVars returns env vars for spec.config.tracing. Returns nil when tracing is disabled.
func getTracingConfigEnvVars(tracing *bindplanev1alpha1.TracingConfig) []corev1.EnvVar {
	if tracing == nil || tracing.Type == "" {
		return nil
	}
	envVars := []corev1.EnvVar{
		{Name: bindplaneTracingTypeEnvVar, Value: tracing.Type},
	}
	if tracing.Type == "otlp" && tracing.OTLP != nil {
		if tracing.OTLP.Endpoint != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneTracingOTLPEndpointEnvVar, Value: tracing.OTLP.Endpoint})
		}
		if tracing.OTLP.Insecure {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneTracingOTLPInsecureEnvVar, Value: "true"})
		}
	}
	if tracing.SamplingRate != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneTracingSamplingRateEnvVar, Value: tracing.SamplingRate})
	}
	return envVars
}

// getMetricsConfigEnvVars returns env vars for spec.config.metrics. When metrics is nil, returns default prometheus env vars.
func getMetricsConfigEnvVars(metrics *bindplanev1alpha1.MetricsConfig) []corev1.EnvVar {
	if metrics == nil {
		return []corev1.EnvVar{
			{Name: bindplaneMetricsTypeEnvVar, Value: "prometheus"},
			{Name: bindplaneMetricsIntervalEnvVar, Value: "60s"},
			{Name: bindplaneMetricsPrometheusEndpointEnvVar, Value: "/metrics"},
		}
	}
	metricsType := metrics.Type
	if metricsType == "" {
		metricsType = "prometheus"
	}
	interval := metrics.Interval
	if interval == "" {
		interval = "60s"
	}
	envVars := []corev1.EnvVar{
		{Name: bindplaneMetricsTypeEnvVar, Value: metricsType},
		{Name: bindplaneMetricsIntervalEnvVar, Value: interval},
	}
	if metricsType == "prometheus" {
		endpoint := "/metrics"
		if metrics.Prometheus != nil && metrics.Prometheus.Endpoint != "" {
			endpoint = metrics.Prometheus.Endpoint
		}
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneMetricsPrometheusEndpointEnvVar, Value: endpoint})
		if metrics.Prometheus != nil {
			if metrics.Prometheus.Username != "" {
				envVars = append(envVars, corev1.EnvVar{Name: bindplaneMetricsPrometheusUsernameEnvVar, Value: metrics.Prometheus.Username})
			}
			if ev := secretOrValue(bindplaneMetricsPrometheusPasswordEnvVar, metrics.Prometheus.Password, metrics.Prometheus.PasswordSecretRef); ev != nil {
				envVars = append(envVars, *ev)
			}
		}
	}
	if metricsType == "otlp" && metrics.OTLP != nil {
		if metrics.OTLP.Endpoint != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneMetricsOTLPEndpointEnvVar, Value: metrics.OTLP.Endpoint})
		}
		if metrics.OTLP.Insecure {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneMetricsOTLPInsecureEnvVar, Value: "true"})
		}
	}
	return envVars
}

// getMiscConfigEnvVars returns env vars for maxConcurrency (default 10) and auditTrail.retentionDays (default 365).
func getMiscConfigEnvVars(config *bindplanev1alpha1.BindplaneConfigSpec) []corev1.EnvVar {
	var envVars []corev1.EnvVar
	maxConcurrency := config.MaxConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = 10
	}
	envVars = append(envVars, corev1.EnvVar{Name: bindplaneMaxConcurrencyEnvVar, Value: strconv.Itoa(maxConcurrency)})
	retentionDays := 365
	if config.AuditTrail != nil && config.AuditTrail.RetentionDays > 0 {
		retentionDays = config.AuditTrail.RetentionDays
	}
	envVars = append(envVars, corev1.EnvVar{Name: bindplaneAuditTrailRetentionDaysEnvVar, Value: strconv.Itoa(retentionDays)})
	return envVars
}

// getBindplaneConfigEnvVars converts BindplaneConfigSpec to environment variables
// following the naming convention from override_test.go (BINDPLANE_*)
func getBindplaneConfigEnvVars(bindplane *bindplanev1alpha1.Bindplane) []corev1.EnvVar {
	config := &bindplane.Spec.Config

	var envVars []corev1.EnvVar
	if ev := secretOrValue(bindplaneLicenseEnvVar, config.License, config.LicenseSecretRef); ev != nil {
		envVars = append(envVars, *ev)
	}
	envVars = append(envVars, getAuthConfigEnvVars(config.Auth)...)
	envVars = append(envVars, getNetworkConfigEnvVars(config.Network, bindplane)...)
	envVars = append(envVars, corev1.EnvVar{Name: bindplaneStoreTypeEnvVar, Value: "postgres"})
	envVars = append(envVars, getPostgresConfigEnvVars(config.Store.Postgres)...)
	envVars = append(envVars, getTracingConfigEnvVars(config.Tracing)...)
	envVars = append(envVars, getMetricsConfigEnvVars(config.Metrics)...)
	envVars = append(envVars, getMiscConfigEnvVars(config)...)
	return envVars
}

// getTSDBEnvVars returns the Prometheus environment variables
// Used by Node, Jobs, Jobs Migrate, and NATS deployments.
// Username and password (for remote write basic auth) are read from the operator-generated Prometheus basic auth Secret.
// When internal TLS is enabled for Prometheus remote write, also adds BINDPLANE_PROMETHEUS_ENABLE_TLS and cert paths.
func getTSDBEnvVars(bindplane *bindplanev1alpha1.Bindplane) []corev1.EnvVar {
	remoteEnabled := isTSDBRemoteEnabled(bindplane)
	tsdbHost := ""
	tsdbPort := int32(tsdbHTTPPort)
	if remoteEnabled {
		remote := bindplane.Spec.Config.TSDB.Remote
		tsdbHost = remote.Host
		if remote.Port > 0 {
			tsdbPort = remote.Port
		}
	} else {
		tsdbServiceName := getResourceName(bindplane, tsdbComponent)
		tsdbHost = strings.Join([]string{tsdbServiceName, bindplane.Namespace, "svc"}, ".")
	}
	envVars := []corev1.EnvVar{
		{Name: bindplaneTSDBEnableRemoteEnvVar, Value: enableRemoteValue},
		{Name: bindplaneTSDBHostEnvVar, Value: tsdbHost},
		{Name: bindplaneTSDBPortEnvVar, Value: strconv.Itoa(int(tsdbPort))},
	}
	if remoteEnabled {
		remote := bindplane.Spec.Config.TSDB.Remote
		if remote.QueryPathPrefix != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneTSDBQueryPathPrefixEnvVar, Value: remote.QueryPathPrefix})
		}
		if remote.RemoteWrite != nil {
			remoteWrite := remote.RemoteWrite
			envVars = append(envVars, corev1.EnvVar{
				Name:  bindplaneTSDBRemoteWriteHostEnvVar,
				Value: remoteWrite.Host,
			})
			envVars = append(envVars, corev1.EnvVar{
				Name:  bindplaneTSDBRemoteWritePortEnvVar,
				Value: strconv.Itoa(int(remoteWrite.Port)),
			})
			remoteWriteEndpoint := remoteWrite.Endpoint
			if remoteWriteEndpoint == "" {
				remoteWriteEndpoint = "/api/v1/write"
			}
			envVars = append(envVars, corev1.EnvVar{
				Name:  bindplaneTSDBRemoteWriteEndpointEnvVar,
				Value: remoteWriteEndpoint,
			})
		}
	} else {
		secretName := getResourceName(bindplane, tsdbBasicAuthSecretSuffix)
		envVars = append(envVars,
			corev1.EnvVar{
				Name: bindplaneTSDBAuthUsernameEnvVar,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
						Key:                  tsdbBasicAuthSecretKeyUser,
					},
				},
			},
			corev1.EnvVar{
				Name: bindplaneTSDBAuthPasswordEnvVar,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
						Key:                  tsdbBasicAuthSecretKeyPass,
					},
				},
			},
		)
	}
	envVars = append(envVars, getTSDBRemoteWriteTLSEnvVars(bindplane)...)
	return envVars
}

// getTSDBRemoteWriteTLSEnvVars returns env vars for Prometheus remote write TLS when client TLS is enabled (config.prometheus.tls).
func getTSDBRemoteWriteTLSEnvVars(bindplane *bindplanev1alpha1.Bindplane) []corev1.EnvVar {
	if !isTSDBClientTLSEnabled(bindplane) {
		return nil
	}
	// Cert-manager uses tls.crt, tls.key, ca.crt; user secret is mounted with Items to same names
	const certKey, keyKey, caKey = "tls.crt", "tls.key", "ca.crt"
	envVars := []corev1.EnvVar{
		{Name: bindplaneTSDBEnableTLSEnvVar, Value: "true"},
		{Name: bindplaneTSDBTLSCertEnvVar, Value: internalTLSTSDBClientMountPath + "/" + certKey},
		{Name: bindplaneTSDBTLSKeyEnvVar, Value: internalTLSTSDBClientMountPath + "/" + keyKey},
		{Name: bindplaneTSDBTLSCAEnvVar, Value: internalTLSTSDBClientMountPath + "/" + caKey},
	}
	if bindplane.Spec.Config.TSDB.TLS.SkipVerify {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneTSDBTLSSkipVerifyEnvVar, Value: "true"})
	}
	return envVars
}

// getNatsTLSEnvVars returns env vars for NATS TLS when cert-manager is enabled (spec.config.nats.tls.certManager).
// Cert-manager secret contains tls.crt, tls.key, ca.crt.
func getNatsTLSEnvVars(bindplane *bindplanev1alpha1.Bindplane) []corev1.EnvVar {
	if !isNatsCertManagerTLSEnabled(bindplane) {
		return nil
	}
	const certKey, keyKey, caKey = "tls.crt", "tls.key", "ca.crt"
	return []corev1.EnvVar{
		{Name: bindplaneNatsEnableTLSEnvVar, Value: "true"},
		{Name: bindplaneNatsTLSCertEnvVar, Value: internalTLSNatsMountPath + "/" + certKey},
		{Name: bindplaneNatsTLSKeyEnvVar, Value: internalTLSNatsMountPath + "/" + keyKey},
		{Name: bindplaneNatsTLSCAEnvVar, Value: internalTLSNatsMountPath + "/" + caKey},
	}
}

// getTransformAgentEnvVars returns the Transform Agent environment variables
// Used by Node, Jobs, Jobs Migrate, and NATS deployments
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

// getProfilingServiceNameDefault returns the default profiling service name for a component when spec does not set it.
func getProfilingServiceNameDefault(component string) string {
	switch component {
	case nodeComponent:
		return "bindplane-node"
	case bindplaneJobsComponent:
		return "bindplane-jobs"
	case bindplaneJobsMigrateComponent:
		return "bindplane-jobs-migrate"
	case natsComponent:
		return "bindplane-nats"
	default:
		return "bindplane-" + component
	}
}

// getProfilingEnvVars returns env vars for spec.config.profiling (Google Cloud Profiler). Only adds vars when profiling is enabled.
func getProfilingEnvVars(config *bindplanev1alpha1.BindplaneConfigSpec, component string) []corev1.EnvVar {
	if config == nil || config.Profiling == nil || !config.Profiling.Enabled {
		return nil
	}
	p := config.Profiling
	serviceName := getProfilingServiceNameDefault(component)
	envVars := []corev1.EnvVar{
		{Name: bindplaneProfilingEnabledEnvVar, Value: "true"},
		{Name: bindplaneProfilingProjectIDEnvVar, Value: p.ProjectID},
		{Name: bindplaneProfilingServiceNameEnvVar, Value: serviceName},
	}
	if p.NoCPU {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneProfilingNoCPUEnvVar, Value: "true"})
	}
	if p.NoAlloc {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneProfilingNoAllocEnvVar, Value: "true"})
	}
	if p.NoHeap {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneProfilingNoHeapEnvVar, Value: "true"})
	}
	if p.NoGoroutine {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneProfilingNoGoroutineEnvVar, Value: "true"})
	}
	if p.Mutex {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneProfilingMutexEnvVar, Value: "true"})
	}
	return envVars
}

// getPprofEnvVars returns env vars for spec.config.pprof. Only adds vars when pprof is enabled.
func getPprofEnvVars(config *bindplanev1alpha1.BindplaneConfigSpec) []corev1.EnvVar {
	if config == nil || config.Pprof == nil || !config.Pprof.Enabled {
		return nil
	}
	endpoint := config.Pprof.Endpoint
	if endpoint == "" {
		endpoint = defaultPprofEndpoint
	}
	return []corev1.EnvVar{
		{Name: bindplanePprofEnabledEnvVar, Value: "true"},
		{Name: bindplanePprofEndpointEnvVar, Value: endpoint},
	}
}

// getNatsClientEnvVars returns the NATS client environment variables for Node and Jobs deployments
func getNatsClientEnvVars(bindplane *bindplanev1alpha1.Bindplane, includeNatsClient bool) []corev1.EnvVar {
	if !includeNatsClient {
		return nil
	}

	envVars := []corev1.EnvVar{
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
	envVars = append(envVars, getNatsTLSEnvVars(bindplane)...)
	return envVars
}

// getBindplaneCommonEnvVars returns env vars shared by Node, Jobs, Jobs Migrate, and NATS.
// component is used to set the default profiling service name (e.g. bindplane-node, bindplane-jobs).
func getBindplaneCommonEnvVars(bindplane *bindplanev1alpha1.Bindplane, component string) []corev1.EnvVar {
	config := &bindplane.Spec.Config
	return combineEnvVars(
		getBindplaneConfigEnvVars(bindplane),
		getTSDBEnvVars(bindplane),
		getTransformAgentEnvVars(bindplane),
		getProfilingEnvVars(config, component),
		getPprofEnvVars(config),
	)
}
