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

var _ = Describe("isTSDBClientCertManagerTLSEnabled", func() {
	It("returns false when config.TSDB is nil", func() {
		bindplane := &bindplanev1alpha1.Bindplane{}
		Expect(isTSDBClientCertManagerTLSEnabled(bindplane)).To(BeFalse())
	})
	It("returns false when TLS or CertManager is nil", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					TSDB: &bindplanev1alpha1.TSDBConfig{},
				},
			},
		}
		Expect(isTSDBClientCertManagerTLSEnabled(bindplane)).To(BeFalse())
	})
	It("returns false when CertManager has empty name", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					TSDB: &bindplanev1alpha1.TSDBConfig{
						TLS: &bindplanev1alpha1.TSDBTLSConfig{
							CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: ""},
						},
					},
				},
			},
		}
		Expect(isTSDBClientCertManagerTLSEnabled(bindplane)).To(BeFalse())
	})
	It("returns true when TLS.CertManager is set with name", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					TSDB: &bindplanev1alpha1.TSDBConfig{
						TLS: &bindplanev1alpha1.TSDBTLSConfig{
							CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "ca-issuer"},
						},
					},
				},
			},
		}
		Expect(isTSDBClientCertManagerTLSEnabled(bindplane)).To(BeTrue())
	})
})

var _ = Describe("isTSDBServerCertManagerTLSEnabled", func() {
	It("returns false when spec.TSDB is nil", func() {
		bindplane := &bindplanev1alpha1.Bindplane{}
		Expect(isTSDBServerCertManagerTLSEnabled(bindplane)).To(BeFalse())
	})
	It("returns true when spec.tsdb.tls.certManager is set with name", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				TSDB: &bindplanev1alpha1.TSDBComponentSpec{
					TLS: &bindplanev1alpha1.TSDBTLSConfig{
						CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "my-issuer"},
					},
				},
			},
		}
		Expect(isTSDBServerCertManagerTLSEnabled(bindplane)).To(BeTrue())
	})
})

var _ = Describe("validateTSDBComponentTLSConfig", func() {
	It("returns error when both secretName and certManager are set", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				TSDB: &bindplanev1alpha1.TSDBComponentSpec{
					TLS: &bindplanev1alpha1.TSDBTLSConfig{
						SecretName:  "x",
						CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "issuer"},
					},
				},
			},
		}
		Expect(validateTSDBComponentTLSConfig(bindplane)).NotTo(Succeed())
		Expect(validateTSDBComponentTLSConfig(bindplane).Error()).To(ContainSubstring("spec.tsdb.tls"))
	})
})

var _ = Describe("validateTSDBTLSConfig", func() {
	It("returns nil when config.TSDB is nil", func() {
		bindplane := &bindplanev1alpha1.Bindplane{}
		Expect(validateTSDBTLSConfig(bindplane)).To(Succeed())
	})
	It("returns error when both secretName and certManager are set", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					TSDB: &bindplanev1alpha1.TSDBConfig{
						TLS: &bindplanev1alpha1.TSDBTLSConfig{
							SecretName:  "my-secret",
							CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "ca"},
						},
					},
				},
			},
		}
		Expect(validateTSDBTLSConfig(bindplane)).NotTo(Succeed())
		Expect(validateTSDBTLSConfig(bindplane).Error()).To(ContainSubstring("mutually exclusive"))
	})
	It("returns nil when only CertManager is set with valid name", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					TSDB: &bindplanev1alpha1.TSDBConfig{
						TLS: &bindplanev1alpha1.TSDBTLSConfig{
							CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "my-issuer", Kind: "Issuer"},
						},
					},
				},
			},
		}
		Expect(validateTSDBTLSConfig(bindplane)).To(Succeed())
	})
})

var _ = Describe("isNatsCertManagerTLSEnabled", func() {
	It("returns false when config.Nats is nil", func() {
		bindplane := &bindplanev1alpha1.Bindplane{}
		Expect(isNatsCertManagerTLSEnabled(bindplane)).To(BeFalse())
	})
	It("returns false when TLS or CertManager is nil", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Nats: &bindplanev1alpha1.NatsConfig{},
				},
			},
		}
		Expect(isNatsCertManagerTLSEnabled(bindplane)).To(BeFalse())
	})
	It("returns false when CertManager has empty name", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Nats: &bindplanev1alpha1.NatsConfig{
						TLS: &bindplanev1alpha1.NatsTLSConfig{
							CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: ""},
						},
					},
				},
			},
		}
		Expect(isNatsCertManagerTLSEnabled(bindplane)).To(BeFalse())
	})
	It("returns true when TLS.CertManager is set with name", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Nats: &bindplanev1alpha1.NatsConfig{
						TLS: &bindplanev1alpha1.NatsTLSConfig{
							CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "nats-issuer"},
						},
					},
				},
			},
		}
		Expect(isNatsCertManagerTLSEnabled(bindplane)).To(BeTrue())
	})
})

var _ = Describe("validateNatsTLSConfig", func() {
	It("returns nil when config.Nats is nil", func() {
		bindplane := &bindplanev1alpha1.Bindplane{}
		Expect(validateNatsTLSConfig(bindplane)).To(Succeed())
	})
	It("returns nil when TLS is nil", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Nats: &bindplanev1alpha1.NatsConfig{},
				},
			},
		}
		Expect(validateNatsTLSConfig(bindplane)).To(Succeed())
	})
	It("returns error when TLS is set but certManager.name is empty", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Nats: &bindplanev1alpha1.NatsConfig{
						TLS: &bindplanev1alpha1.NatsTLSConfig{
							CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: ""},
						},
					},
				},
			},
		}
		Expect(validateNatsTLSConfig(bindplane)).NotTo(Succeed())
		Expect(validateNatsTLSConfig(bindplane).Error()).To(ContainSubstring("spec.config.nats.tls"))
		Expect(validateNatsTLSConfig(bindplane).Error()).To(ContainSubstring("certManager.name"))
	})
	It("returns nil when certManager is set with non-empty name", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Nats: &bindplanev1alpha1.NatsConfig{
						TLS: &bindplanev1alpha1.NatsTLSConfig{
							CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "nats-issuer", Kind: "Issuer"},
						},
					},
				},
			},
		}
		Expect(validateNatsTLSConfig(bindplane)).To(Succeed())
	})
})

var _ = Describe("isTransformAgentCertManagerTLSEnabled", func() {
	It("returns false when spec.transformAgent is nil", func() {
		bindplane := &bindplanev1alpha1.Bindplane{}
		Expect(isTransformAgentCertManagerTLSEnabled(bindplane)).To(BeFalse())
	})

	It("returns false when TLS or CertManager is nil", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				TransformAgent: &bindplanev1alpha1.TransformAgentComponentSpec{},
			},
		}
		Expect(isTransformAgentCertManagerTLSEnabled(bindplane)).To(BeFalse())
	})

	It("returns true when TLS.CertManager is set with name", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				TransformAgent: &bindplanev1alpha1.TransformAgentComponentSpec{
					TLS: &bindplanev1alpha1.TransformAgentTLSConfig{
						CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "ta-issuer"},
					},
				},
			},
		}
		Expect(isTransformAgentCertManagerTLSEnabled(bindplane)).To(BeTrue())
	})
})

var _ = Describe("validateTransformAgentTLSConfig", func() {
	It("returns nil when spec.transformAgent is nil", func() {
		bindplane := &bindplanev1alpha1.Bindplane{}
		Expect(validateTransformAgentTLSConfig(bindplane)).To(Succeed())
	})

	It("returns error when TLS is set but certManager.name is empty", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				TransformAgent: &bindplanev1alpha1.TransformAgentComponentSpec{
					TLS: &bindplanev1alpha1.TransformAgentTLSConfig{
						CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{},
					},
				},
			},
		}
		Expect(validateTransformAgentTLSConfig(bindplane)).NotTo(Succeed())
		Expect(validateTransformAgentTLSConfig(bindplane).Error()).To(ContainSubstring("spec.transformAgent.tls"))
	})

	It("returns nil when certManager is set with non-empty name", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				TransformAgent: &bindplanev1alpha1.TransformAgentComponentSpec{
					TLS: &bindplanev1alpha1.TransformAgentTLSConfig{
						CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "ta-issuer"},
					},
				},
			},
		}
		Expect(validateTransformAgentTLSConfig(bindplane)).To(Succeed())
	})
})

var _ = Describe("getNatsServerCertDNSNames", func() {
	It("returns client service, headless, pod DNS names and localhost", func() {
		replicas := int32(2)
		bindplane := &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "my-bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Nats: &bindplanev1alpha1.NatsComponentSpec{
					Replicas: &replicas,
				},
			},
		}
		names := getNatsServerCertDNSNames(bindplane)
		Expect(names).To(ContainElement("my-bp-nats-client.default"))
		Expect(names).To(ContainElement("my-bp-nats-client.default.svc.cluster.local"))
		Expect(names).To(ContainElement("my-bp-nats-client.default.svc"))
		Expect(names).To(ContainElement("my-bp-nats-cluster.default"))
		Expect(names).To(ContainElement("my-bp-nats-cluster.default.svc.cluster.local"))
		Expect(names).To(ContainElement("localhost"))
		Expect(names).To(ContainElement("my-bp-nats-0.my-bp-nats-cluster.default"))
		Expect(names).To(ContainElement("my-bp-nats-0.my-bp-nats-cluster.default.svc.cluster.local"))
		Expect(names).To(ContainElement("my-bp-nats-1.my-bp-nats-cluster.default"))
		Expect(names).To(ContainElement("my-bp-nats-1.my-bp-nats-cluster.default.svc.cluster.local"))
	})
})

var _ = Describe("getTransformAgentServerCertDNSNames", func() {
	It("returns service DNS names used by Bindplane clients", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "my-bp", Namespace: "default"},
		}

		names := getTransformAgentServerCertDNSNames(bindplane)
		Expect(names).To(ContainElement("my-bp-transform-agent"))
		Expect(names).To(ContainElement("my-bp-transform-agent.default"))
		Expect(names).To(ContainElement("my-bp-transform-agent.default.svc"))
		Expect(names).To(ContainElement("my-bp-transform-agent.default.svc.cluster.local"))
		Expect(names).To(ContainElement("localhost"))
		Expect(names).To(HaveLen(5))
	})
})

var _ = Describe("getTSDBServerCertDNSNames", func() {
	It("returns service and pod DNS names for the bindplane tsdb component", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "my-bp", Namespace: "default"},
		}
		names := getTSDBServerCertDNSNames(bindplane)
		Expect(names).To(ContainElement("my-bp-tsdb.default.svc.cluster.local"))
		Expect(names).To(ContainElement("my-bp-tsdb.default.svc"))
		Expect(names).To(ContainElement("my-bp-tsdb-0.my-bp-tsdb.default.svc.cluster.local"))
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
		cn := "bindplane-tsdb-remote-write-client"
		cert := buildCertificate(bindplane, "bp-prom-client", "bp-prom-client", issuerRef, nil, nil, &cn)
		Expect(cert.Spec.CommonName).To(Equal(cn))
		Expect(cert.Spec.DNSNames).To(BeNil())
		Expect(cert.Spec.Usages).To(ContainElement(cmapi.UsageClientAuth))
	})
})

var _ = Describe("buildTransformAgentCertificate", func() {
	It("builds a dual-use certificate for the Transform Agent", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "bp", Namespace: "ns"},
		}
		issuerRef := cmmeta.IssuerReference{Name: "ca", Kind: "Issuer", Group: "cert-manager.io"}
		dnsNames := []string{"bp-transform-agent", "bp-transform-agent.ns.svc.cluster.local"}

		cert := buildTransformAgentCertificate(bindplane, "bp-transform-agent-tls", "bp-transform-agent-tls", issuerRef, dnsNames)

		Expect(cert.Name).To(Equal("bp-transform-agent-tls"))
		Expect(cert.Spec.SecretName).To(Equal("bp-transform-agent-tls"))
		Expect(cert.Spec.DNSNames).To(Equal(dnsNames))
		Expect(cert.Spec.Usages).To(ContainElement(cmapi.UsageServerAuth))
		Expect(cert.Spec.Usages).To(ContainElement(cmapi.UsageClientAuth))
		Expect(cert.Spec.PrivateKey).ToNot(BeNil())
		Expect(cert.Spec.PrivateKey.Algorithm).To(Equal(cmapi.RSAKeyAlgorithm))
	})
})
