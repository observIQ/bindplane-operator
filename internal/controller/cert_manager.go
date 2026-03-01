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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

// Internal TLS certificate name suffixes (resource names and secret names).
// Pattern: one reconcile helper per interface; add suffixes for NATS, etc. later.
const (
	prometheusRemoteWriteServerCertSuffix = "prometheus-remote-write-server"
	prometheusRemoteWriteClientCertSuffix = "prometheus-remote-write-client"
	// prometheusProbeClientCertSuffix is the client cert for Prometheus pod's promtool (probe); ClientAuth EKU only.
	prometheusProbeClientCertSuffix = "prometheus-probe-client"
	natsTLSCertSuffix               = "nats-tls"
)

// reconcileInternalTLSCertificates reconciles cert-manager Certificate resources for
// internal mTLS (Prometheus remote write). Run before Prometheus and Node/Jobs so issued secrets exist.
// Server cert: when spec.prometheus.tls.certManager is set. Client cert: when spec.config.prometheus.tls.certManager is set.
func (r *BindplaneReconciler) reconcileInternalTLSCertificates(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
	// Server cert (Prometheus StatefulSet)
	if isPrometheusServerCertManagerTLSEnabled(bindplane) {
		if err := validatePrometheusComponentTLSConfig(bindplane); err != nil {
			return err
		}
		if err := r.reconcilePrometheusRemoteWriteServerCert(ctx, bindplane, log); err != nil {
			return err
		}
		if err := r.reconcilePrometheusProbeClientCert(ctx, bindplane, log); err != nil {
			return err
		}
	}
	// Client cert (Bindplane Node, Jobs, NATS)
	if isPrometheusClientCertManagerTLSEnabled(bindplane) {
		if err := validatePrometheusTLSConfig(bindplane); err != nil {
			return err
		}
		if err := r.reconcilePrometheusRemoteWriteClientCert(ctx, bindplane, log); err != nil {
			return err
		}
	}
	// NATS TLS (single cert for client, cluster, and HTTP)
	if isNatsCertManagerTLSEnabled(bindplane) {
		if err := validateNatsTLSConfig(bindplane); err != nil {
			return err
		}
		if err := r.reconcileNatsTLSCert(ctx, bindplane, log); err != nil {
			return err
		}
	}
	return nil
}

// isPrometheusClientCertManagerTLSEnabled returns true when cert-manager is used for the Bindplane→Prometheus client cert (spec.config.prometheus.tls.certManager).
func isPrometheusClientCertManagerTLSEnabled(bindplane *bindplanev1alpha1.Bindplane) bool {
	p := bindplane.Spec.Config.Prometheus
	return p != nil && p.TLS != nil && p.TLS.CertManager != nil && p.TLS.CertManager.Name != ""
}

// isPrometheusClientTLSEnabled returns true when the Bindplane client should use TLS for Prometheus remote write (config.prometheus.tls with certManager or secretName).
func isPrometheusClientTLSEnabled(bindplane *bindplanev1alpha1.Bindplane) bool {
	p := bindplane.Spec.Config.Prometheus
	if p == nil || p.TLS == nil {
		return false
	}
	return (p.TLS.CertManager != nil && p.TLS.CertManager.Name != "") || p.TLS.SecretName != ""
}

// isPrometheusServerCertManagerTLSEnabled returns true when cert-manager is used for the Prometheus server cert (spec.prometheus.tls.certManager).
func isPrometheusServerCertManagerTLSEnabled(bindplane *bindplanev1alpha1.Bindplane) bool {
	p := bindplane.Spec.Prometheus
	return p != nil && p.TLS != nil && p.TLS.CertManager != nil && p.TLS.CertManager.Name != ""
}

// validatePrometheusComponentTLSConfig returns an error when spec.prometheus.tls has both secretName and certManager set.
func validatePrometheusComponentTLSConfig(bindplane *bindplanev1alpha1.Bindplane) error {
	p := bindplane.Spec.Prometheus
	if p == nil || p.TLS == nil {
		return nil
	}
	tls := p.TLS
	hasSecret := tls.SecretName != ""
	hasCertManager := tls.CertManager != nil && tls.CertManager.Name != ""
	if hasSecret && hasCertManager {
		return fmt.Errorf("spec.prometheus.tls: secretName and certManager are mutually exclusive")
	}
	return nil
}

// validatePrometheusTLSConfig returns an error when tls is set but both or neither of secretName and certManager are set,
// or when certManager is set with an empty name.
func validatePrometheusTLSConfig(bindplane *bindplanev1alpha1.Bindplane) error {
	cfg := bindplane.Spec.Config.Prometheus
	if cfg == nil || cfg.TLS == nil {
		return nil
	}
	tls := cfg.TLS
	hasSecret := tls.SecretName != ""
	hasCertManager := tls.CertManager != nil && tls.CertManager.Name != ""
	if hasSecret && hasCertManager {
		return fmt.Errorf("spec.config.prometheus.tls: secretName and certManager are mutually exclusive")
	}
	if !hasSecret && !hasCertManager {
		return nil // tls block present but neither set is valid (no-op)
	}
	if hasCertManager {
		// already checked Name != "" above
		return nil
	}
	return nil
}

// reconcilePrometheusRemoteWriteServerCert creates or updates the server Certificate for the Prometheus StatefulSet.
// Server cert: ServerAuth EKU only; DNS SANs for service name; optional localhost and 127.0.0.1 for probes.
func (r *BindplaneReconciler) reconcilePrometheusRemoteWriteServerCert(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
	issuerRef := issuerRefToCM(*bindplane.Spec.Prometheus.TLS.CertManager)
	serverDNSNames := getPrometheusServerCertDNSNames(bindplane)
	serverCert := buildCertificate(
		bindplane,
		getResourceName(bindplane, prometheusRemoteWriteServerCertSuffix),
		getResourceName(bindplane, prometheusRemoteWriteServerCertSuffix),
		issuerRef,
		serverDNSNames,
		[]string{"127.0.0.1"},
		nil,
	)
	if err := controllerutil.SetControllerReference(bindplane, serverCert, r.Scheme); err != nil {
		return err
	}
	if err := r.reconcileCertificate(ctx, serverCert, log); err != nil {
		return fmt.Errorf("reconcile Prometheus server certificate: %w", err)
	}
	return nil
}

// reconcilePrometheusProbeClientCert creates or updates the client Certificate for the Prometheus pod's promtool (probe).
// Client cert: ClientAuth EKU only; used by probe-http.yml so promtool can authenticate to the Prometheus web server (mTLS).
func (r *BindplaneReconciler) reconcilePrometheusProbeClientCert(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
	issuerRef := issuerRefToCM(*bindplane.Spec.Prometheus.TLS.CertManager)
	clientCert := buildCertificate(
		bindplane,
		getResourceName(bindplane, prometheusProbeClientCertSuffix),
		getResourceName(bindplane, prometheusProbeClientCertSuffix),
		issuerRef,
		nil,
		nil,
		stringPtr("prometheus-probe"),
	)
	if err := controllerutil.SetControllerReference(bindplane, clientCert, r.Scheme); err != nil {
		return err
	}
	if err := r.reconcileCertificate(ctx, clientCert, log); err != nil {
		return fmt.Errorf("reconcile Prometheus probe client certificate: %w", err)
	}
	return nil
}

// reconcilePrometheusRemoteWriteClientCert creates or updates the client Certificate for Bindplane pods (Node, Jobs, NATS).
func (r *BindplaneReconciler) reconcilePrometheusRemoteWriteClientCert(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
	issuerRef := issuerRefToCM(*bindplane.Spec.Config.Prometheus.TLS.CertManager)
	clientCert := buildCertificate(
		bindplane,
		getResourceName(bindplane, prometheusRemoteWriteClientCertSuffix),
		getResourceName(bindplane, prometheusRemoteWriteClientCertSuffix),
		issuerRef,
		nil,
		nil,
		stringPtr("bindplane-prometheus-remote-write-client"),
	)
	if err := controllerutil.SetControllerReference(bindplane, clientCert, r.Scheme); err != nil {
		return err
	}
	if err := r.reconcileCertificate(ctx, clientCert, log); err != nil {
		return fmt.Errorf("reconcile Prometheus client certificate: %w", err)
	}
	return nil
}

func getPrometheusServerCertDNSNames(bindplane *bindplanev1alpha1.Bindplane) []string {
	name := getResourceName(bindplane, prometheusComponent)
	ns := bindplane.Namespace
	return []string{
		fmt.Sprintf("%s.%s.svc.cluster.local", name, ns),
		fmt.Sprintf("%s.%s.svc", name, ns),
		fmt.Sprintf("%s-0.%s.%s.svc.cluster.local", name, name, ns),
		"localhost",
	}
}

// isNatsCertManagerTLSEnabled returns true when NATS TLS is enabled via cert-manager (spec.config.nats.tls.certManager).
func isNatsCertManagerTLSEnabled(bindplane *bindplanev1alpha1.Bindplane) bool {
	n := bindplane.Spec.Config.Nats
	return n != nil && n.TLS != nil && n.TLS.CertManager != nil && n.TLS.CertManager.Name != ""
}

// validateNatsTLSConfig returns an error when spec.config.nats.tls is set but certManager.name is empty.
func validateNatsTLSConfig(bindplane *bindplanev1alpha1.Bindplane) error {
	n := bindplane.Spec.Config.Nats
	if n == nil || n.TLS == nil {
		return nil
	}
	if n.TLS.CertManager == nil || n.TLS.CertManager.Name == "" {
		return fmt.Errorf("spec.config.nats.tls: certManager.name is required when TLS is enabled")
	}
	return nil
}

// getNatsServerCertDNSNames returns DNS names for the NATS certificate (client, cluster, and HTTP use the same cert).
// Includes short form <service>.<namespace> so clients connecting to e.g. bindplane-sample-nats-client.default can verify the cert.
func getNatsServerCertDNSNames(bindplane *bindplanev1alpha1.Bindplane) []string {
	name := getResourceName(bindplane, natsComponent)
	clientSvcName := getNatsClientServiceName(bindplane)
	headlessName := getNatsClusterServiceName(bindplane)
	ns := bindplane.Namespace
	replicas := int32(1)
	if bindplane.Spec.Nats != nil && bindplane.Spec.Nats.Replicas != nil {
		replicas = *bindplane.Spec.Nats.Replicas
	}
	names := []string{
		fmt.Sprintf("%s.%s", clientSvcName, ns),
		fmt.Sprintf("%s.%s.svc.cluster.local", clientSvcName, ns),
		fmt.Sprintf("%s.%s.svc", clientSvcName, ns),
		fmt.Sprintf("%s.%s", headlessName, ns),
		fmt.Sprintf("%s.%s.svc.cluster.local", headlessName, ns),
		fmt.Sprintf("%s.%s.svc", headlessName, ns),
		"localhost",
	}
	for i := int32(0); i < replicas; i++ {
		names = append(names, fmt.Sprintf("%s-%d.%s.%s.svc.cluster.local", name, i, headlessName, ns))
	}
	return names
}

// reconcileNatsTLSCert creates or updates the NATS TLS certificate (used for client, cluster, and HTTP ports).
func (r *BindplaneReconciler) reconcileNatsTLSCert(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
	issuerRef := issuerRefToCM(*bindplane.Spec.Config.Nats.TLS.CertManager)
	dnsNames := getNatsServerCertDNSNames(bindplane)
	cert := buildNatsCertificate(
		bindplane,
		getResourceName(bindplane, natsTLSCertSuffix),
		getResourceName(bindplane, natsTLSCertSuffix),
		issuerRef,
		dnsNames,
	)
	if err := controllerutil.SetControllerReference(bindplane, cert, r.Scheme); err != nil {
		return err
	}
	if err := r.reconcileCertificate(ctx, cert, log); err != nil {
		return fmt.Errorf("reconcile NATS TLS certificate: %w", err)
	}
	return nil
}

// buildNatsCertificate builds a Certificate with both ServerAuth and ClientAuth for NATS (client, cluster, HTTP).
// Includes 127.0.0.1 and ::1 in IP SANs so the embedded NATS client (connecting over localhost) can verify the server.
func buildNatsCertificate(
	bindplane *bindplanev1alpha1.Bindplane,
	name, secretName string,
	issuerRef cmmeta.IssuerReference,
	dnsNames []string,
) *cmapi.Certificate {
	labels := getLabels(bindplane, name)
	return &cmapi.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: bindplane.Namespace,
			Labels:    labels,
		},
		Spec: cmapi.CertificateSpec{
			SecretName:  secretName,
			IssuerRef:   issuerRef,
			DNSNames:    dnsNames,
			IPAddresses: []string{"127.0.0.1", "::1"},
			PrivateKey: &cmapi.CertificatePrivateKey{
				Algorithm: cmapi.RSAKeyAlgorithm,
				Size:      2048,
			},
			Usages: []cmapi.KeyUsage{
				cmapi.UsageDigitalSignature,
				cmapi.UsageKeyEncipherment,
				cmapi.UsageServerAuth,
				cmapi.UsageClientAuth,
			},
		},
	}
}

// issuerRefToCM converts a CertManagerTLSIssuerRef to a cert-manager IssuerReference,
// defaulting Kind to "Issuer" and Group to "cert-manager.io" when empty.
func issuerRefToCM(ref bindplanev1alpha1.CertManagerTLSIssuerRef) cmmeta.IssuerReference {
	kind := ref.Kind
	if kind == "" {
		kind = "Issuer"
	}
	group := ref.Group
	if group == "" {
		group = "cert-manager.io"
	}
	return cmmeta.IssuerReference{Name: ref.Name, Kind: kind, Group: group}
}

func buildCertificate(
	bindplane *bindplanev1alpha1.Bindplane,
	name, secretName string,
	issuerRef cmmeta.IssuerReference,
	dnsNames []string,
	ipAddresses []string,
	commonName *string,
) *cmapi.Certificate {
	labels := getLabels(bindplane, name)
	spec := cmapi.CertificateSpec{
		SecretName: secretName,
		IssuerRef:  issuerRef,
		DNSNames:   dnsNames,
		PrivateKey: &cmapi.CertificatePrivateKey{
			Algorithm: cmapi.RSAKeyAlgorithm,
			Size:      2048,
		},
		Usages: []cmapi.KeyUsage{cmapi.UsageDigitalSignature, cmapi.UsageKeyEncipherment, cmapi.UsageServerAuth},
	}
	if len(ipAddresses) > 0 {
		spec.IPAddresses = ipAddresses
	}
	if commonName != nil {
		spec.CommonName = *commonName
		// Client cert: ClientAuth usage only (no ServerAuth)
		spec.Usages = []cmapi.KeyUsage{cmapi.UsageDigitalSignature, cmapi.UsageKeyEncipherment, cmapi.UsageClientAuth}
	}
	return &cmapi.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: bindplane.Namespace,
			Labels:    labels,
		},
		Spec: spec,
	}
}

func stringPtr(s string) *string { return &s }

func (r *BindplaneReconciler) reconcileCertificate(ctx context.Context, desired *cmapi.Certificate, log logr.Logger) error {
	existing := &cmapi.Certificate{}
	err := r.Get(ctx, client.ObjectKeyFromObject(desired), existing)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if errors.IsNotFound(err) {
		log.Info("Creating Certificate", "name", desired.Name, "namespace", desired.Namespace)
		return r.Create(ctx, desired)
	}
	// Update spec and labels; leave status and other metadata to cert-manager.
	existing.Spec = desired.Spec
	existing.Labels = desired.Labels
	return r.Update(ctx, existing)
}
