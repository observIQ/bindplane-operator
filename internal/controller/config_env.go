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
	if ev := secretOrValue(bindplaneSecretKeyEnvVar, auth.APIKey, auth.APIKeySecretRef); ev != nil {
		envVars = append(envVars, *ev)
	}
	envVars = append(envVars, getLDAPEnvVars(auth.LDAP)...)
	envVars = append(envVars, getOIDCEnvVars(auth.OIDC)...)
	envVars = append(envVars, getAuth0EnvVars(auth.Auth0)...)
	return envVars
}

// getAuth0EnvVars returns env vars for spec.config.auth.auth0.
// Returns nil when auth0 is nil.
func getAuth0EnvVars(a *bindplanev1alpha1.Auth0Config) []corev1.EnvVar {
	if a == nil {
		return nil
	}
	var envVars []corev1.EnvVar
	if a.ClientID != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAuth0ClientIDEnvVar, Value: a.ClientID})
	}
	if a.Domain != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAuth0DomainEnvVar, Value: a.Domain})
	}
	if a.Audience != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAuth0AudienceEnvVar, Value: a.Audience})
	}
	if a.ManagementDomain != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAuth0ManagementDomainEnvVar, Value: a.ManagementDomain})
	}
	if a.ManagementClientID != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAuth0ManagementClientIDEnvVar, Value: a.ManagementClientID})
	}
	if ev := secretOrValue(bindplaneAuth0ManagementClientSecretEnvVar, a.ManagementClientSecret, a.ManagementClientSecretSecretRef); ev != nil {
		envVars = append(envVars, *ev)
	}
	if sso := a.SSO; sso != nil {
		if sso.Enabled {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneAuth0SSOEnabledEnvVar, Value: "true"})
		}
		if sso.SelfServiceProfileID != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneAuth0SSOSelfServiceProfileIDEnvVar, Value: sso.SelfServiceProfileID})
		}
	}
	if wif := a.WIF; wif != nil {
		if wif.ClientID != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneAuth0WIFClientIDEnvVar, Value: wif.ClientID})
		}
		if ev := secretOrValue(bindplaneAuth0WIFClientSecretEnvVar, wif.ClientSecret, wif.ClientSecretSecretRef); ev != nil {
			envVars = append(envVars, *ev)
		}
		if wif.Audience != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneAuth0WIFAudienceEnvVar, Value: wif.Audience})
		}
	}
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
	if network != nil && network.RateLimits != nil {
		rl := network.RateLimits
		if rl.APIRate != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneNetworkRateLimitsAPIRateEnvVar, Value: rl.APIRate})
		}
		if rl.APIBurst > 0 {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneNetworkRateLimitsAPIBurstEnvVar, Value: strconv.Itoa(rl.APIBurst)})
		}
		if rl.GraphQLRate != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneNetworkRateLimitsGraphQLRateEnvVar, Value: rl.GraphQLRate})
		}
		if rl.GraphQLBurst > 0 {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneNetworkRateLimitsGraphQLBurstEnvVar, Value: strconv.Itoa(rl.GraphQLBurst)})
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

// getStoreConfigEnvVars returns store-level (non-Postgres) env vars for spec.config.store.
// Returns nil when no store-level fields are set.
func getStoreConfigEnvVars(store *bindplanev1alpha1.StoreConfig) []corev1.EnvVar {
	if store == nil {
		return nil
	}
	var envVars []corev1.EnvVar
	if store.MaxEvents > 0 {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneStoreMaxEventsEnvVar, Value: strconv.Itoa(store.MaxEvents)})
	}
	if store.EventMergeWindow != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneStoreEventMergeWindowEnvVar, Value: store.EventMergeWindow})
	}
	if store.SummaryRollupRetentionDays != nil {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneStoreSummaryRollupRetentionDaysEnvVar, Value: strconv.Itoa(*store.SummaryRollupRetentionDays)})
	}
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
	if tracing.Type == "honeycomb" && tracing.Honeycomb != nil {
		if ev := secretOrValue(bindplaneTracingHoneycombAPIKeyEnvVar, tracing.Honeycomb.APIKey, tracing.Honeycomb.APIKeySecretRef); ev != nil {
			envVars = append(envVars, *ev)
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

// getMiscConfigEnvVars returns env vars for maxConcurrency (default defaultConcurrency),
// maxSimultaneousConnections (default defaultConcurrency), and auditTrail.retentionDays (default 365).
func getMiscConfigEnvVars(config *bindplanev1alpha1.BindplaneConfigSpec) []corev1.EnvVar {
	envVars := make([]corev1.EnvVar, 0, 3)
	maxConcurrency := config.MaxConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = defaultConcurrency
	}
	envVars = append(envVars, corev1.EnvVar{Name: bindplaneMaxConcurrencyEnvVar, Value: strconv.Itoa(maxConcurrency)})
	maxSimultaneousConnections := defaultConcurrency
	if config.Agents != nil && config.Agents.MaxSimultaneousConnections > 0 {
		maxSimultaneousConnections = config.Agents.MaxSimultaneousConnections
	}
	envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsMaxSimultaneousConnectionsEnvVar, Value: strconv.Itoa(maxSimultaneousConnections)})
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

	// Session secret: always injected. User-provided plain value or SecretRef takes precedence;
	// otherwise reference the operator-generated secret.
	var sessionSecretEV corev1.EnvVar
	if config.Auth != nil {
		if ev := secretOrValue(bindplaneSessionSecretEnvVar, config.Auth.SessionSecret, config.Auth.SessionSecretSecretRef); ev != nil {
			sessionSecretEV = *ev
		}
	}
	if sessionSecretEV.Name == "" {
		sessionSecretEV = corev1.EnvVar{
			Name: bindplaneSessionSecretEnvVar,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: getResourceName(bindplane, sessionSecretSuffix),
					},
					Key: sessionSecretKey,
				},
			},
		}
	}
	envVars = append(envVars, sessionSecretEV)

	envVars = append(envVars, getNetworkConfigEnvVars(config.Network, bindplane)...)
	envVars = append(envVars, corev1.EnvVar{Name: bindplaneStoreTypeEnvVar, Value: "postgres"})
	envVars = append(envVars, getPostgresConfigEnvVars(config.Store.Postgres)...)
	envVars = append(envVars, getStoreConfigEnvVars(&config.Store)...)
	envVars = append(envVars, getTracingConfigEnvVars(config.Tracing)...)
	envVars = append(envVars, getMetricsConfigEnvVars(config.Metrics)...)
	envVars = append(envVars, getMiscConfigEnvVars(config)...)
	envVars = append(envVars, getAgentsConfigEnvVars(config.Agents)...)
	envVars = append(envVars, getAgentVersionsConfigEnvVars(config.AgentVersions)...)
	envVars = append(envVars, getSaaSConfigEnvVars(config.SaaS)...)
	envVars = append(envVars, getEncryptionProviderEnvVars(config.EncryptionProvider)...)
	envVars = append(envVars, getFeaturesConfigEnvVars(config.Features)...)
	envVars = append(envVars, getErrorsConfigEnvVars(config.Errors)...)
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
			corev1.EnvVar{Name: bindplaneTSDBAuthTypeEnvVar, Value: "basic"},
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

// getTransformAgentTLSEnvVars returns env vars for Transform Agent TLS when cert-manager is enabled (spec.transformAgent.tls.certManager).
// Cert-manager secret contains tls.crt, tls.key, and ca.crt.
func getTransformAgentTLSEnvVars(bindplane *bindplanev1alpha1.Bindplane) []corev1.EnvVar {
	if !isTransformAgentCertManagerTLSEnabled(bindplane) {
		return nil
	}
	const certKey, keyKey, caKey = "tls.crt", "tls.key", "ca.crt"
	return []corev1.EnvVar{
		{Name: bindplaneTransformAgentTLSCertEnvVar, Value: internalTLSTransformAgentMountPath + "/" + certKey},
		{Name: bindplaneTransformAgentTLSKeyEnvVar, Value: internalTLSTransformAgentMountPath + "/" + keyKey},
		{Name: bindplaneTransformAgentTLSCAEnvVar, Value: internalTLSTransformAgentMountPath + "/" + caKey},
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
		return "bindplane-migrate"
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

	tlsVars := getNatsTLSEnvVars(bindplane)
	envVars := make([]corev1.EnvVar, 0, 4+len(tlsVars))
	envVars = append(envVars,
		corev1.EnvVar{Name: bindplaneEventBusTypeEnvVar, Value: natsEventBusType},
		corev1.EnvVar{
			Name: bindplaneNatsClientNameEnvVar,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: metadataNameFieldPath,
				},
			},
		},
		corev1.EnvVar{Name: bindplaneNatsClientEndpointEnvVar, Value: getNatsClientEndpoint(bindplane)},
		corev1.EnvVar{Name: bindplaneNatsClientSubjectEnvVar, Value: natsClientSubject},
	)
	envVars = append(envVars, tlsVars...)
	return envVars
}

// getStatusEnvVars returns environment variables for the status check endpoint configuration.
func getStatusEnvVars(config *bindplanev1alpha1.BindplaneConfigSpec) []corev1.EnvVar {
	if config == nil || config.Status == nil {
		return nil
	}
	s := config.Status
	envVars := []corev1.EnvVar{
		{Name: bindplaneStatusEnabledEnvVar, Value: strconv.FormatBool(s.Enabled)},
	}
	if s.KeysSecretRef != nil {
		envVars = append(envVars, corev1.EnvVar{
			Name:      bindplaneStatusKeysEnvVar,
			ValueFrom: &corev1.EnvVarSource{SecretKeyRef: s.KeysSecretRef},
		})
	} else if len(s.Keys) > 0 {
		envVars = append(envVars, corev1.EnvVar{
			Name:  bindplaneStatusKeysEnvVar,
			Value: strings.Join(s.Keys, ","),
		})
	}
	return envVars
}

// getAnalyticsEnvVars returns env vars for spec.config.analytics.
// Returns nil when analytics is nil (analytics enabled, no custom key).
func getAnalyticsEnvVars(config *bindplanev1alpha1.BindplaneConfigSpec) []corev1.EnvVar {
	if config == nil || config.Analytics == nil {
		return nil
	}
	a := config.Analytics
	var envVars []corev1.EnvVar
	if a.Disabled {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAnalyticsDisabledEnvVar, Value: "true"})
	}
	if a.SegmentWriteKey != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAnalyticsSegmentWriteKeyEnvVar, Value: a.SegmentWriteKey})
	}
	return envVars
}

// defaultRequiredHosts calculates the default event bus health required hosts:
// floor(total / 2) + 1, where total = node + nats + jobs (1).
// Jobs Migrate is a batch/v1 Job (not a long-running pod) and is excluded from this total.
func defaultRequiredHosts(bindplane *bindplanev1alpha1.Bindplane) int32 {
	nodeReplicas := int32(3)
	if bindplane.Spec.Bindplane.Replicas != nil {
		nodeReplicas = *bindplane.Spec.Bindplane.Replicas
	}
	natsReplicas := *bindplane.Spec.Nats.Replicas
	total := nodeReplicas + natsReplicas + 1 // +1 jobs (migrate is now a batch Job)
	return total/2 + 1
}

// getEventBusHealthEnvVars returns env vars for the event bus health check.
// Returns nil when eventBus or eventBus.health is not configured.
func getEventBusHealthEnvVars(bindplane *bindplanev1alpha1.Bindplane) []corev1.EnvVar {
	config := &bindplane.Spec.Config
	if config.EventBus == nil || config.EventBus.Health == nil {
		return nil
	}
	h := config.EventBus.Health
	requiredHosts := defaultRequiredHosts(bindplane)
	if h.RequiredHosts != nil {
		requiredHosts = *h.RequiredHosts
	}
	envVars := []corev1.EnvVar{
		{Name: bindplaneEventBusHealthRequiredHostsEnvVar, Value: strconv.Itoa(int(requiredHosts))},
	}
	if h.Interval != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name:  bindplaneEventBusHealthIntervalEnvVar,
			Value: h.Interval,
		})
	}
	return envVars
}

// getLoggingConfigEnvVars returns env vars for spec.config.logging.
// Returns nil when logging is nil; Bindplane uses its own defaults in that case.
func getLoggingConfigEnvVars(config *bindplanev1alpha1.BindplaneConfigSpec) []corev1.EnvVar {
	if config == nil || config.Logging == nil {
		return nil
	}
	l := config.Logging
	level := l.Level
	if level == "" {
		level = "info"
	}
	loggingType := l.Type
	if loggingType == "" {
		loggingType = "stdout"
	}
	envVars := []corev1.EnvVar{
		{Name: bindplaneLoggingLevelEnvVar, Value: level},
		{Name: bindplaneLoggingTypeEnvVar, Value: loggingType},
	}
	if l.OTLP != nil {
		if l.OTLP.Endpoint != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneLoggingOTLPEndpointEnvVar, Value: l.OTLP.Endpoint})
		}
		if l.OTLP.Insecure {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneLoggingOTLPInsecureEnvVar, Value: "true"})
		}
		if l.OTLP.Interval != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneLoggingOTLPIntervalEnvVar, Value: l.OTLP.Interval})
		}
	}
	return envVars
}

// getAgentsConfigEnvVars returns env vars for spec.config.agents.
// Returns nil when agents is nil (Bindplane uses its own defaults).
func getAgentsConfigEnvVars(agents *bindplanev1alpha1.AgentsConfig) []corev1.EnvVar {
	if agents == nil {
		return nil
	}
	var envVars []corev1.EnvVar

	// Auth
	if agents.Auth != nil {
		auth := agents.Auth
		if auth.Type != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsAuthTypeEnvVar, Value: auth.Type})
		}
		if auth.SecretKey != nil && len(auth.SecretKey.Headers) > 0 {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsAuthSecretKeyHeadersEnvVar, Value: strings.Join(auth.SecretKey.Headers, ",")})
		}
		if auth.OAuth != nil {
			oauth := auth.OAuth
			if oauth.Issuer != "" {
				envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsAuthOAuthIssuerEnvVar, Value: oauth.Issuer})
			}
			if len(oauth.Audiences) > 0 {
				envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsAuthOAuthAudiencesEnvVar, Value: strings.Join(oauth.Audiences, ",")})
			}
			if len(oauth.RequiredClaims) > 0 {
				envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsAuthOAuthRequiredClaimsEnvVar, Value: strings.Join(oauth.RequiredClaims, ",")})
			}
			if len(oauth.RequiredScopes) > 0 {
				envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsAuthOAuthRequiredScopesEnvVar, Value: strings.Join(oauth.RequiredScopes, ",")})
			}
			if oauth.CacheTTL != "" {
				envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsAuthOAuthCacheTTLEnvVar, Value: oauth.CacheTTL})
			}
		}
	}

	// Heartbeat
	if agents.HeartbeatInterval != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsHeartbeatIntervalEnvVar, Value: agents.HeartbeatInterval})
	}
	if agents.HeartbeatTTL != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsHeartbeatTTLEnvVar, Value: agents.HeartbeatTTL})
	}
	if agents.HeartbeatExpiryInterval != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsHeartbeatExpiryIntervalEnvVar, Value: agents.HeartbeatExpiryInterval})
	}

	// Rebalance
	if agents.RebalanceInterval != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsRebalanceIntervalEnvVar, Value: agents.RebalanceInterval})
	}
	if agents.RebalancePercentage != nil {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsRebalancePercentageEnvVar, Value: strconv.Itoa(*agents.RebalancePercentage)})
	}
	if agents.RebalanceJitter != nil {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsRebalanceJitterEnvVar, Value: strconv.Itoa(*agents.RebalanceJitter)})
	}

	// Connection registry middleware
	envVars = append(envVars, getAgentsConnectionRegistryEnvVars(agents)...)

	// Duplication prevention
	envVars = append(envVars, getAgentsDuplicationPreventionEnvVars(agents.DuplicationPrevention)...)

	return envVars
}

// getAgentsConnectionRegistryEnvVars returns connection registry env vars for spec.config.agents.
func getAgentsConnectionRegistryEnvVars(agents *bindplanev1alpha1.AgentsConfig) []corev1.EnvVar {
	var envVars []corev1.EnvVar
	if agents.EnableConnectionRegistryMiddleware {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsEnableConnectionRegistryMiddlewareEnvVar, Value: "true"})
	}
	if agents.ConnectionRegistryHeartbeatInterval != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsConnectionRegistryHeartbeatIntervalEnvVar, Value: agents.ConnectionRegistryHeartbeatInterval})
	}
	if agents.ConnectionRegistryStaleDuration != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsConnectionRegistryStaleDurationEnvVar, Value: agents.ConnectionRegistryStaleDuration})
	}
	if agents.ConnectionRegistryLockTimeout != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsConnectionRegistryLockTimeoutEnvVar, Value: agents.ConnectionRegistryLockTimeout})
	}
	if agents.ConnectionClaimTimeout != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsConnectionClaimTimeoutEnvVar, Value: agents.ConnectionClaimTimeout})
	}
	return envVars
}

// getAgentsDuplicationPreventionEnvVars returns env vars for spec.config.agents.duplicationPrevention.
func getAgentsDuplicationPreventionEnvVars(dp *bindplanev1alpha1.AgentDuplicationPreventionConfig) []corev1.EnvVar {
	if dp == nil {
		return nil
	}
	var envVars []corev1.EnvVar
	if dp.EnableMiddleware {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsDupPrevEnableMiddlewareEnvVar, Value: "true"})
	}
	if dp.ReassignID {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsDupPrevReassignIDEnvVar, Value: "true"})
	}
	if dp.DetectionStrategy != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsDupPrevDetectionStrategyEnvVar, Value: dp.DetectionStrategy})
	}
	if dp.DetectionGracePeriod != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsDupPrevDetectionGracePeriodEnvVar, Value: dp.DetectionGracePeriod})
	}
	if dp.MinGracePeriodFailures > 0 {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsDupPrevMinGracePeriodFailuresEnvVar, Value: strconv.Itoa(dp.MinGracePeriodFailures)})
	}
	if dp.RetryAfter != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsDupPrevRetryAfterEnvVar, Value: dp.RetryAfter})
	}
	if dp.MaxReassignmentAttempts > 0 {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsDupPrevMaxReassignmentAttemptsEnvVar, Value: strconv.Itoa(dp.MaxReassignmentAttempts)})
	}
	if dp.ReassignmentCacheTTL != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsDupPrevReassignmentCacheTTLEnvVar, Value: dp.ReassignmentCacheTTL})
	}
	if dp.ReassignmentRetryAfter != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsDupPrevReassignmentRetryAfterEnvVar, Value: dp.ReassignmentRetryAfter})
	}
	if dp.EnableDuplicateNotifications {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsDupPrevEnableDuplicateNotificationsEnvVar, Value: "true"})
	}
	if dp.EnablePerOrgEnforcement {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentsDupPrevEnablePerOrgEnforcementEnvVar, Value: "true"})
	}
	return envVars
}

// getAgentVersionsConfigEnvVars returns env vars for spec.config.agentVersions.
// Returns nil when agentVersions is nil (Bindplane uses its own defaults).
func getAgentVersionsConfigEnvVars(av *bindplanev1alpha1.AgentVersionsConfig) []corev1.EnvVar {
	if av == nil {
		return nil
	}
	var envVars []corev1.EnvVar
	if av.SyncInterval != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentVersionsSyncIntervalEnvVar, Value: av.SyncInterval})
	}
	if len(av.Clients) > 0 {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAgentVersionsClientsEnvVar, Value: strings.Join(av.Clients, ",")})
	}
	return envVars
}

// getSaaSConfigEnvVars returns env vars for spec.config.saas.
// Returns nil when saas is nil.
func getSaaSConfigEnvVars(s *bindplanev1alpha1.SaaSConfig) []corev1.EnvVar {
	if s == nil {
		return nil
	}
	var envVars []corev1.EnvVar
	if s.Enabled {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneSaaSEnabledEnvVar, Value: "true"})
	}
	if s.LicenseServerAddress != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneSaaSLicenseServerAddressEnvVar, Value: s.LicenseServerAddress})
	}
	if ev := secretOrValue(bindplaneSaaSLicenseServerAPIKeyEnvVar, s.LicenseServerAPIKey, s.LicenseServerAPIKeySecretRef); ev != nil {
		envVars = append(envVars, *ev)
	}
	if s.JanitorOrganization != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneSaaSJanitorOrganizationEnvVar, Value: s.JanitorOrganization})
	}
	if s.UseStagePublicRSAKey {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneSaaSUseStagePublicRSAKeyEnvVar, Value: "true"})
	}
	envVars = append(envVars, getSaaSStripeEnvVars(s.Stripe)...)
	return envVars
}

// getSaaSStripeEnvVars returns env vars for spec.config.saas.stripe.
func getSaaSStripeEnvVars(stripe *bindplanev1alpha1.SaaSStripeConfig) []corev1.EnvVar {
	if stripe == nil {
		return nil
	}
	var envVars []corev1.EnvVar
	if stripe.Enabled {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneSaaSStripeEnabledEnvVar, Value: "true"})
	}
	if ev := secretOrValue(bindplaneSaaSStripeSecretKeyEnvVar, stripe.SecretKey, stripe.SecretKeySecretRef); ev != nil {
		envVars = append(envVars, *ev)
	}
	if ev := secretOrValue(bindplaneSaaSStripePublishableKeyEnvVar, stripe.PublishableKey, stripe.PublishableKeySecretRef); ev != nil {
		envVars = append(envVars, *ev)
	}
	if ev := secretOrValue(bindplaneSaaSStripeWebhookSecretEnvVar, stripe.WebhookSecret, stripe.WebhookSecretSecretRef); ev != nil {
		envVars = append(envVars, *ev)
	}
	if ids := stripe.GrowthPlanIDs; ids != nil {
		if ids.BaseRate != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneSaaSStripeGrowthPlanIDsBaseRateEnvVar, Value: ids.BaseRate})
		}
		if ids.UsageRates != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneSaaSStripeGrowthPlanIDsUsageRatesEnvVar, Value: ids.UsageRates})
		}
	}
	if mn := stripe.GrowthPlanMeterNames; mn != nil {
		if mn.Logs != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneSaaSStripeGrowthMeterNamesLogsEnvVar, Value: mn.Logs})
		}
		if mn.Metrics != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneSaaSStripeGrowthMeterNamesMetricsEnvVar, Value: mn.Metrics})
		}
		if mn.Traces != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneSaaSStripeGrowthMeterNamesTracesEnvVar, Value: mn.Traces})
		}
		if mn.Collectors != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneSaaSStripeGrowthMeterNamesCollectorsEnvVar, Value: mn.Collectors})
		}
	}
	return envVars
}

// getFeaturesConfigEnvVars returns env vars for spec.config.features.
// Returns nil when features is nil.
func getFeaturesConfigEnvVars(f *bindplanev1alpha1.FeaturesConfig) []corev1.EnvVar {
	if f == nil {
		return nil
	}
	var envVars []corev1.EnvVar
	if f.Type != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneFeaturesTypeEnvVar, Value: f.Type})
	}
	if f.PostHog != nil {
		envVars = append(envVars, getPostHogEnvVars(f.PostHog)...)
	}
	envVars = append(envVars, getFeatureOverridesEnvVars(f.Overrides)...)
	return envVars
}

// getPostHogEnvVars returns env vars for the PostHog feature flag config.
func getPostHogEnvVars(ph *bindplanev1alpha1.PostHogConfig) []corev1.EnvVar {
	var envVars []corev1.EnvVar
	if ev := secretOrValue(bindplaneFeaturesPostHogProjectAPIKeyEnvVar, ph.ProjectAPIKey, ph.ProjectAPIKeySecretRef); ev != nil {
		envVars = append(envVars, *ev)
	}
	if ev := secretOrValue(bindplaneFeaturesPostHogPersonalAPIKeyEnvVar, ph.PersonalAPIKey, ph.PersonalAPIKeySecretRef); ev != nil {
		envVars = append(envVars, *ev)
	}
	if ph.Endpoint != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneFeaturesPostHogEndpointEnvVar, Value: ph.Endpoint})
	}
	if ph.DefaultFeatureFlagsPollingInterval != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneFeaturesPostHogPollingIntervalEnvVar, Value: ph.DefaultFeatureFlagsPollingInterval})
	}
	return envVars
}

// getAdvancedStoreStatsEnvVars returns env vars for spec.config.advanced.store.stats.
func getAdvancedStoreStatsEnvVars(s *bindplanev1alpha1.AdvancedStoreStatsConfig) []corev1.EnvVar {
	if s == nil {
		return nil
	}
	var envVars []corev1.EnvVar
	if s.BatchFlushInterval != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAdvancedStoreStatsBatchFlushIntervalEnvVar, Value: s.BatchFlushInterval})
	}
	if s.WorkerCount > 0 {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAdvancedStoreStatsWorkerCountEnvVar, Value: strconv.Itoa(s.WorkerCount)})
	}
	if s.EnableSorting {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAdvancedStoreStatsEnableSortingEnvVar, Value: "true"})
	}
	if s.MetricChannelSize > 0 {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAdvancedStoreStatsMetricChannelSizeEnvVar, Value: strconv.Itoa(s.MetricChannelSize)})
	}
	if s.BatchChannelSize > 0 {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAdvancedStoreStatsBatchChannelSizeEnvVar, Value: strconv.Itoa(s.BatchChannelSize)})
	}
	return envVars
}

// getAdvancedServerEnvVars returns env vars for spec.config.advanced.server.
func getAdvancedServerEnvVars(srv *bindplanev1alpha1.AdvancedServerConfig) []corev1.EnvVar {
	if srv == nil {
		return nil
	}
	var envVars []corev1.EnvVar
	if srv.MaxRequestBytes > 0 {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAdvancedServerMaxRequestBytesEnvVar, Value: strconv.FormatInt(srv.MaxRequestBytes, 10)})
	}
	if srv.ShutdownGracePeriod != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAdvancedServerShutdownGracePeriodEnvVar, Value: srv.ShutdownGracePeriod})
	}
	if srv.OpAMPShutdownGracePeriodTarget != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAdvancedServerOpAMPShutdownGracePeriodTargetEnvVar, Value: srv.OpAMPShutdownGracePeriodTarget})
	}
	return envVars
}

// getAdvancedCacheRedisEnvVars returns env vars for spec.config.advanced.cache.redis.
func getAdvancedCacheRedisEnvVars(r *bindplanev1alpha1.AdvancedCacheRedisConfig) []corev1.EnvVar {
	if r == nil {
		return nil
	}
	var envVars []corev1.EnvVar
	if r.Address != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAdvancedCacheRedisAddressEnvVar, Value: r.Address})
	}
	if ev := secretOrValue(bindplaneAdvancedCacheRedisPasswordEnvVar, r.Password, r.PasswordSecretRef); ev != nil {
		envVars = append(envVars, *ev)
	}
	if r.DB > 0 {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAdvancedCacheRedisDBEnvVar, Value: strconv.Itoa(r.DB)})
	}
	if r.ReadTimeout != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAdvancedCacheRedisReadTimeoutEnvVar, Value: r.ReadTimeout})
	}
	if r.WriteTimeout != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAdvancedCacheRedisWriteTimeoutEnvVar, Value: r.WriteTimeout})
	}
	if r.EnableTLS {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAdvancedCacheRedisEnableTLSEnvVar, Value: "true"})
	}
	if r.TLS != nil && r.TLS.SecretName != "" {
		tls := r.TLS
		if tls.CertKey != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneAdvancedCacheRedisTLSCertEnvVar, Value: advancedCacheRedisTLSMountPath + "/" + tls.CertKey})
		}
		if tls.KeyKey != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneAdvancedCacheRedisTLSKeyEnvVar, Value: advancedCacheRedisTLSMountPath + "/" + tls.KeyKey})
		}
		if tls.CAKey != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneAdvancedCacheRedisTLSCAEnvVar, Value: advancedCacheRedisTLSMountPath + "/" + tls.CAKey})
		}
		if tls.SkipVerify {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneAdvancedCacheRedisTLSSkipVerifyEnvVar, Value: "true"})
		}
		if tls.MinTLSVersion != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneAdvancedCacheRedisTLSMinVersionEnvVar, Value: tls.MinTLSVersion})
		}
	}
	return envVars
}

// getEncryptionProviderEnvVars returns env vars for spec.config.encryptionProvider.
// Returns nil when encryptionProvider is nil (Bindplane uses its built-in encryption).
// Note: BINDPLANE_ENCRYPTIONPROVIDER_GOOGLEKMS_KEY_DELETION_JOB is NOT included here;
// it is injected directly into the Jobs Migrate workload only.
func getEncryptionProviderEnvVars(ep *bindplanev1alpha1.EncryptionProviderConfig) []corev1.EnvVar {
	if ep == nil {
		return nil
	}
	var envVars []corev1.EnvVar
	if ep.Type != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneEncryptionProviderTypeEnvVar, Value: ep.Type})
	}
	if ep.GoogleKMS != nil {
		kms := ep.GoogleKMS
		if kms.ProjectID != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneEncryptionProviderGoogleKMSProjectIDEnvVar, Value: kms.ProjectID})
		}
		if kms.Location != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneEncryptionProviderGoogleKMSLocationEnvVar, Value: kms.Location})
		}
		if kms.KeyRotationPeriod != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneEncryptionProviderGoogleKMSKeyRotationPeriodEnvVar, Value: kms.KeyRotationPeriod})
		}
	}
	return envVars
}

// getFeatureOverridesEnvVars returns env vars for feature flag overrides.
// Returns nil when overrides is nil.
func getFeatureOverridesEnvVars(o *bindplanev1alpha1.FeatureOverridesConfig) []corev1.EnvVar {
	if o == nil {
		return nil
	}
	var envVars []corev1.EnvVar
	if o.GrowthLicense {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneFeaturesOverridesGrowthLicenseEnvVar, Value: "true"})
	}
	if o.SecopsTheme {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneFeaturesOverridesSecopsThemeEnvVar, Value: "true"})
	}
	if o.SecopsIntegration {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneFeaturesOverridesSecopsIntegrationEnvVar, Value: "true"})
	}
	if o.LLMFeatures {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneFeaturesOverridesLLMFeaturesEnvVar, Value: "true"})
	}
	if o.PipelineIntelligence {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneFeaturesOverridesPipelineIntelligenceEnvVar, Value: "true"})
	}
	if o.PipelineIntelligenceSnapshotLogTypes {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneFeaturesOverridesPipelineIntelligenceSnapshotLogTypesEnvVar, Value: "true"})
	}
	if o.PipelineIntelligenceOtelConfigImport {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneFeaturesOverridesPipelineIntelligenceOtelConfigImportEnvVar, Value: "true"})
	}
	if o.PipelineIntelligenceChronicleForwarderConfigImport {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneFeaturesOverridesPipelineIntelligenceChronicleForwarderConfigImportEnvVar, Value: "true"})
	}
	if o.PipelineIntelligenceParseField {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneFeaturesOverridesPipelineIntelligenceParseFieldEnvVar, Value: "true"})
	}
	if o.PipelineIntelligenceGenerateProcessors {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneFeaturesOverridesPipelineIntelligenceGenerateProcessorsEnvVar, Value: "true"})
	}
	if o.RawConfigLegacy {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneFeaturesOverridesRawConfigLegacyEnvVar, Value: "true"})
	}
	if o.Notifications {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneFeaturesOverridesNotificationsEnvVar, Value: "true"})
	}
	return envVars
}

// getErrorsConfigEnvVars returns env vars for spec.config.errors.
// Returns nil when errors is nil (error tracking is disabled).
func getErrorsConfigEnvVars(e *bindplanev1alpha1.ErrorsConfig) []corev1.EnvVar {
	if e == nil {
		return nil
	}
	var envVars []corev1.EnvVar
	if e.Enabled {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneErrorsEnabledEnvVar, Value: "true"})
	}
	if e.BackendDSN != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneErrorsBackendDSNEnvVar, Value: e.BackendDSN})
	}
	if e.FrontendDSN != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneErrorsFrontendDSNEnvVar, Value: e.FrontendDSN})
	}
	if e.Environment != "" {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneErrorsEnvironmentEnvVar, Value: e.Environment})
	}
	return envVars
}

// getAdvancedConfigEnvVars returns env vars for spec.config.advanced.
// Returns nil when advanced is nil.
func getAdvancedConfigEnvVars(config *bindplanev1alpha1.BindplaneConfigSpec) []corev1.EnvVar {
	if config == nil || config.Advanced == nil {
		return nil
	}
	adv := config.Advanced
	var envVars []corev1.EnvVar

	if adv.Store != nil {
		envVars = append(envVars, getAdvancedStoreStatsEnvVars(adv.Store.Stats)...)
	}
	if adv.Rollout != nil && adv.Rollout.DisableUpdater {
		envVars = append(envVars, corev1.EnvVar{Name: bindplaneAdvancedRolloutDisableUpdaterEnvVar, Value: "true"})
	}
	envVars = append(envVars, getAdvancedServerEnvVars(adv.Server)...)
	if adv.Cache != nil {
		c := adv.Cache
		if c.Type != "" {
			envVars = append(envVars, corev1.EnvVar{Name: bindplaneAdvancedCacheTypeEnvVar, Value: c.Type})
		}
		envVars = append(envVars, getAdvancedCacheRedisEnvVars(c.Redis)...)
	}

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
		getTransformAgentTLSEnvVars(bindplane),
		getProfilingEnvVars(config, component),
		getPprofEnvVars(config),
		getStatusEnvVars(config),
		getAnalyticsEnvVars(config),
		getLoggingConfigEnvVars(config),
		getEventBusHealthEnvVars(bindplane),
		getAdvancedConfigEnvVars(config),
	)
}
