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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

var _ = Describe("isPrometheusClientCertManagerTLSEnabled", func() {
	It("returns false when config.Prometheus is nil", func() {
		bindplane := &bindplanev1alpha1.Bindplane{}
		Expect(isPrometheusClientCertManagerTLSEnabled(bindplane)).To(BeFalse())
	})
	It("returns false when TLS or CertManager is nil", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Prometheus: &bindplanev1alpha1.Prometheus{},
				},
			},
		}
		Expect(isPrometheusClientCertManagerTLSEnabled(bindplane)).To(BeFalse())
	})
	It("returns false when CertManager has empty name", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Prometheus: &bindplanev1alpha1.Prometheus{
						TLS: &bindplanev1alpha1.PrometheusTLSConfig{
							CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: ""},
						},
					},
				},
			},
		}
		Expect(isPrometheusClientCertManagerTLSEnabled(bindplane)).To(BeFalse())
	})
	It("returns true when TLS.CertManager is set with name", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Prometheus: &bindplanev1alpha1.Prometheus{
						TLS: &bindplanev1alpha1.PrometheusTLSConfig{
							CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "ca-issuer"},
						},
					},
				},
			},
		}
		Expect(isPrometheusClientCertManagerTLSEnabled(bindplane)).To(BeTrue())
	})
})

var _ = Describe("isPrometheusServerCertManagerTLSEnabled", func() {
	It("returns false when spec.Prometheus is nil", func() {
		bindplane := &bindplanev1alpha1.Bindplane{}
		Expect(isPrometheusServerCertManagerTLSEnabled(bindplane)).To(BeFalse())
	})
	It("returns true when spec.prometheus.tls.certManager is set with name", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Prometheus: &bindplanev1alpha1.PrometheusComponentSpec{
					TLS: &bindplanev1alpha1.PrometheusTLSConfig{
						CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "my-issuer"},
					},
				},
			},
		}
		Expect(isPrometheusServerCertManagerTLSEnabled(bindplane)).To(BeTrue())
	})
})

var _ = Describe("validatePrometheusComponentTLSConfig", func() {
	It("returns error when both secretName and certManager are set", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Prometheus: &bindplanev1alpha1.PrometheusComponentSpec{
					TLS: &bindplanev1alpha1.PrometheusTLSConfig{
						SecretName:  "x",
						CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "issuer"},
					},
				},
			},
		}
		Expect(validatePrometheusComponentTLSConfig(bindplane)).NotTo(Succeed())
		Expect(validatePrometheusComponentTLSConfig(bindplane).Error()).To(ContainSubstring("spec.prometheus.tls"))
	})
})

var _ = Describe("validatePrometheusTLSConfig", func() {
	It("returns nil when config.Prometheus is nil", func() {
		bindplane := &bindplanev1alpha1.Bindplane{}
		Expect(validatePrometheusTLSConfig(bindplane)).To(Succeed())
	})
	It("returns error when both secretName and certManager are set", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Prometheus: &bindplanev1alpha1.Prometheus{
						TLS: &bindplanev1alpha1.PrometheusTLSConfig{
							SecretName:  "my-secret",
							CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "ca"},
						},
					},
				},
			},
		}
		Expect(validatePrometheusTLSConfig(bindplane)).NotTo(Succeed())
		Expect(validatePrometheusTLSConfig(bindplane).Error()).To(ContainSubstring("mutually exclusive"))
	})
	It("returns nil when only CertManager is set with valid name", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Prometheus: &bindplanev1alpha1.Prometheus{
						TLS: &bindplanev1alpha1.PrometheusTLSConfig{
							CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "my-issuer", Kind: "Issuer"},
						},
					},
				},
			},
		}
		Expect(validatePrometheusTLSConfig(bindplane)).To(Succeed())
	})
})

var _ = Describe("getPrometheusServerCertDNSNames", func() {
	It("returns service and pod DNS names for the bindplane prometheus component", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "my-bp", Namespace: "default"},
		}
		names := getPrometheusServerCertDNSNames(bindplane)
		Expect(names).To(ContainElement("my-bp-prometheus.default.svc.cluster.local"))
		Expect(names).To(ContainElement("my-bp-prometheus.default.svc"))
		Expect(names).To(ContainElement("my-bp-prometheus-0.my-bp-prometheus.default.svc.cluster.local"))
		Expect(names).To(ContainElement("localhost"))
		Expect(names).To(HaveLen(4))
	})
})

var _ = Describe("issuerRefToCM", func() {
	It("converts name, kind, and group", func() {
		ref := bindplanev1alpha1.CertManagerTLSIssuerRef{
			Name:  "my-cluster-issuer",
			Kind:  "ClusterIssuer",
			Group: "cert-manager.io",
		}
		out := issuerRefToCM(ref)
		Expect(out.Name).To(Equal("my-cluster-issuer"))
		Expect(out.Kind).To(Equal("ClusterIssuer"))
		Expect(out.Group).To(Equal("cert-manager.io"))
	})
	It("defaults kind to Issuer and group to cert-manager.io when empty", func() {
		ref := bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "ca-issuer"}
		out := issuerRefToCM(ref)
		Expect(out.Name).To(Equal("ca-issuer"))
		Expect(out.Kind).To(Equal("Issuer"))
		Expect(out.Group).To(Equal("cert-manager.io"))
	})
})

var _ = Describe("buildCertificate", func() {
	var bindplane *bindplanev1alpha1.Bindplane

	BeforeEach(func() {
		bindplane = &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "bp", Namespace: "ns"},
		}
	})

	It("builds server certificate with DNS names and server auth usage", func() {
		issuerRef := cmmeta.IssuerReference{Name: "ca", Kind: "Issuer", Group: "cert-manager.io"}
		dnsNames := []string{"prom.ns.svc.cluster.local"}
		cert := buildCertificate(bindplane, "bp-prom-server", "bp-prom-server", issuerRef, dnsNames, nil, nil)
		Expect(cert.Name).To(Equal("bp-prom-server"))
		Expect(cert.Namespace).To(Equal("ns"))
		Expect(cert.Spec.SecretName).To(Equal("bp-prom-server"))
		Expect(cert.Spec.IssuerRef).To(Equal(issuerRef))
		Expect(cert.Spec.DNSNames).To(Equal(dnsNames))
		Expect(cert.Spec.CommonName).To(BeEmpty())
		Expect(cert.Spec.Usages).To(ContainElement(cmapi.UsageServerAuth))
		Expect(cert.Spec.PrivateKey).ToNot(BeNil())
		Expect(cert.Spec.PrivateKey.Algorithm).To(Equal(cmapi.RSAKeyAlgorithm))
	})

	It("builds client certificate with commonName and client auth usage", func() {
		issuerRef := cmmeta.IssuerReference{Name: "ca", Kind: "ClusterIssuer", Group: "cert-manager.io"}
		cn := "bindplane-prometheus-remote-write-client"
		cert := buildCertificate(bindplane, "bp-prom-client", "bp-prom-client", issuerRef, nil, nil, &cn)
		Expect(cert.Spec.CommonName).To(Equal(cn))
		Expect(cert.Spec.DNSNames).To(BeNil())
		Expect(cert.Spec.Usages).To(ContainElement(cmapi.UsageClientAuth))
	})
})
