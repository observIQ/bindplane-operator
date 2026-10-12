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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

func TestSupportsHealthProbes(t *testing.T) {
	tests := []struct {
		image string
		want  bool
	}{
		{image: "ghcr.io/observiq/bindplane-ee:1.104.0", want: false},
		{image: "ghcr.io/observiq/bindplane-ee:1.104.1", want: false},
		{image: "ghcr.io/observiq/bindplane-ee:1.105.0-beta.0", want: true},
		{image: "ghcr.io/observiq/bindplane-ee:1.105.0", want: true},
		{image: "ghcr.io/observiq/bindplane-ee:v1.106.2", want: true},
		{image: "ghcr.io/observiq/bindplane-ee:2.0.0", want: true},
		{image: "ghcr.io/observiq/bindplane-ee:latest", want: false},
		{image: "ghcr.io/observiq/bindplane-ee@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", want: false},
		{image: "ghcr.io/observiq/bindplane-ee:1.105.0@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", want: false},
		{image: "reg:5000/x:1.105.0", want: true},
		{image: "reg:5000/x", want: false},
		{image: "bindplane-ee", want: false},
		{image: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.image, func(t *testing.T) {
			if got := supportsHealthProbes(tt.image); got != tt.want {
				t.Errorf("supportsHealthProbes(%q) = %v, want %v", tt.image, got, tt.want)
			}
		})
	}
}

func tcpProbeHandler(port string) corev1.ProbeHandler {
	return corev1.ProbeHandler{TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromString(port)}}
}

func httpProbeHandler(path string) corev1.ProbeHandler {
	return corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{
		Path:   path,
		Port:   intstr.FromString(bindplaneHealthPortName),
		Scheme: corev1.URISchemeHTTP,
	}}
}

func TestMergeSingleContainer_Probes(t *testing.T) {
	tcpStartup := &corev1.Probe{ProbeHandler: tcpProbeHandler("http"), PeriodSeconds: 5}
	httpStartup := &corev1.Probe{ProbeHandler: httpProbeHandler(bindplaneStartupPath), PeriodSeconds: 7}
	tcpReadiness := &corev1.Probe{ProbeHandler: tcpProbeHandler("http"), PeriodSeconds: 10}
	httpReadiness := &corev1.Probe{ProbeHandler: httpProbeHandler(bindplaneReadinessPath), FailureThreshold: 9}
	tcpLiveness := &corev1.Probe{ProbeHandler: tcpProbeHandler("http"), TimeoutSeconds: 5}
	httpLiveness := &corev1.Probe{ProbeHandler: httpProbeHandler(bindplaneLivenessPath), TimeoutSeconds: 2}

	t.Run("user tcpSocket replaces operator httpGet", func(t *testing.T) {
		g := NewWithT(t)
		operator := corev1.Container{Name: "c", StartupProbe: httpStartup, ReadinessProbe: httpReadiness, LivenessProbe: httpLiveness}
		user := corev1.Container{Name: "c", StartupProbe: tcpStartup, ReadinessProbe: tcpReadiness, LivenessProbe: tcpLiveness}
		got := mergeSingleContainer(operator, user)
		g.Expect(got.StartupProbe).To(Equal(tcpStartup))
		g.Expect(got.ReadinessProbe).To(Equal(tcpReadiness))
		g.Expect(got.LivenessProbe).To(Equal(tcpLiveness))
	})

	t.Run("user httpGet replaces operator tcpSocket", func(t *testing.T) {
		g := NewWithT(t)
		operator := corev1.Container{Name: "c", StartupProbe: tcpStartup, ReadinessProbe: tcpReadiness, LivenessProbe: tcpLiveness}
		user := corev1.Container{Name: "c", StartupProbe: httpStartup, ReadinessProbe: httpReadiness, LivenessProbe: httpLiveness}
		got := mergeSingleContainer(operator, user)
		g.Expect(got.StartupProbe).To(Equal(httpStartup))
		g.Expect(got.ReadinessProbe).To(Equal(httpReadiness))
		g.Expect(got.LivenessProbe).To(Equal(httpLiveness))
	})

	t.Run("unset user probes keep the operator probes", func(t *testing.T) {
		g := NewWithT(t)
		operator := corev1.Container{Name: "c", StartupProbe: httpStartup, ReadinessProbe: httpReadiness, LivenessProbe: httpLiveness}
		user := corev1.Container{Name: "c", ReadinessProbe: tcpReadiness}
		got := mergeSingleContainer(operator, user)
		g.Expect(got.StartupProbe).To(Equal(httpStartup))
		g.Expect(got.ReadinessProbe).To(Equal(tcpReadiness))
		g.Expect(got.LivenessProbe).To(Equal(httpLiveness))
	})

	t.Run("user timing only probes are merged into the operator probes", func(t *testing.T) {
		g := NewWithT(t)
		operator := corev1.Container{Name: "c", ReadinessProbe: httpReadiness, LivenessProbe: tcpLiveness}
		user := corev1.Container{
			Name:           "c",
			ReadinessProbe: &corev1.Probe{PeriodSeconds: 30},
			LivenessProbe:  &corev1.Probe{FailureThreshold: 6},
		}
		got := mergeSingleContainer(operator, user)

		expectedReadiness := httpReadiness.DeepCopy()
		expectedReadiness.PeriodSeconds = 30
		g.Expect(got.ReadinessProbe).To(Equal(expectedReadiness))

		expectedLiveness := tcpLiveness.DeepCopy()
		expectedLiveness.FailureThreshold = 6
		g.Expect(got.LivenessProbe).To(Equal(expectedLiveness))
	})
}

var _ = Describe("health probes", func() {
	const (
		legacyVersion    = "1.104.0"
		supportedVersion = "1.105.0"
		digestImage      = "ghcr.io/observiq/bindplane-ee@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	)

	type workload struct {
		newBindplane func() *bindplanev1alpha1.Bindplane
		container    func(r *BindplaneReconciler, bp *bindplanev1alpha1.Bindplane) corev1.Container
		setImage     func(bp *bindplanev1alpha1.Bindplane, image string)
		tcpPortName  string
	}

	newBindplane := func() *bindplanev1alpha1.Bindplane {
		bp := newTestBindplane("my-bp", "default")
		bp.Spec.Bindplane.Replicas = new(int32(2))
		bp.Spec.Nats = &bindplanev1alpha1.NatsComponentSpec{Replicas: new(int32(3))}
		return bp
	}

	nodeWorkload := workload{
		newBindplane: newBindplane,
		container: func(r *BindplaneReconciler, bp *bindplanev1alpha1.Bindplane) corev1.Container {
			return r.nodeDeployment(bp).Spec.Template.Spec.Containers[0]
		},
		setImage:    func(bp *bindplanev1alpha1.Bindplane, image string) { bp.Spec.Bindplane.Image = image },
		tcpPortName: nodeHTTPPortName,
	}
	nodeRolloutWorkload := workload{
		newBindplane: func() *bindplanev1alpha1.Bindplane {
			bp := newBindplane()
			bp.Spec.Bindplane.ArgoRollout = &bindplanev1alpha1.ArgoRolloutSpec{Enabled: true}
			return bp
		},
		container: func(r *BindplaneReconciler, bp *bindplanev1alpha1.Bindplane) corev1.Container {
			return r.nodeRollout(bp).Spec.Template.Spec.Containers[0]
		},
		setImage:    func(bp *bindplanev1alpha1.Bindplane, image string) { bp.Spec.Bindplane.Image = image },
		tcpPortName: nodeHTTPPortName,
	}
	opampWorkload := workload{
		newBindplane: func() *bindplanev1alpha1.Bindplane {
			bp := newBindplane()
			bp.Spec.OpAMP = &bindplanev1alpha1.OpAMPComponentSpec{Enabled: true}
			return bp
		},
		container: func(r *BindplaneReconciler, bp *bindplanev1alpha1.Bindplane) corev1.Container {
			return r.opampDeployment(bp).Spec.Template.Spec.Containers[0]
		},
		setImage:    func(bp *bindplanev1alpha1.Bindplane, image string) { bp.Spec.OpAMP.Image = image },
		tcpPortName: opampHTTPPortName,
	}
	jobsWorkload := workload{
		newBindplane: newBindplane,
		container: func(r *BindplaneReconciler, bp *bindplanev1alpha1.Bindplane) corev1.Container {
			return r.bindplaneJobsDeployment(bp).Spec.Template.Spec.Containers[0]
		},
		setImage: func(bp *bindplanev1alpha1.Bindplane, image string) {
			bp.Spec.BindplaneJobs = &bindplanev1alpha1.BindplaneJobsComponentSpec{Image: image}
		},
		tcpPortName: bindplaneJobsHTTPPortName,
	}
	natsWorkload := workload{
		newBindplane: newBindplane,
		container: func(r *BindplaneReconciler, bp *bindplanev1alpha1.Bindplane) corev1.Container {
			return r.natsStatefulSet(bp).Spec.Template.Spec.Containers[0]
		},
		setImage:    func(bp *bindplanev1alpha1.Bindplane, image string) { bp.Spec.Nats.Image = image },
		tcpPortName: natsHTTPPortName,
	}

	expectLegacyHealth := func(c corev1.Container, tcpPortName string) {
		Expect(c.StartupProbe).To(Equal(&corev1.Probe{
			ProbeHandler:        tcpProbeHandler(tcpPortName),
			InitialDelaySeconds: probeStartupInitialDelaySeconds,
			PeriodSeconds:       probeStartupPeriodSeconds,
			FailureThreshold:    probeStartupFailureThreshold,
			SuccessThreshold:    probeStartupSuccessThreshold,
			TimeoutSeconds:      probeStartupTimeoutSeconds,
		}))
		legacy := &corev1.Probe{
			ProbeHandler:     tcpProbeHandler(tcpPortName),
			PeriodSeconds:    probePeriodSeconds,
			FailureThreshold: probeFailureThreshold,
			SuccessThreshold: probeSuccessThreshold,
			TimeoutSeconds:   probeTimeoutSeconds,
		}
		Expect(c.ReadinessProbe).To(Equal(legacy))
		Expect(c.LivenessProbe).To(Equal(legacy))
		Expect(c.Lifecycle).To(Equal(&corev1.Lifecycle{
			PreStop: &corev1.LifecycleHandler{
				Exec: &corev1.ExecAction{Command: []string{"sh", "-c", "sleep 5"}},
			},
		}))
		for _, p := range c.Ports {
			Expect(p.Name).NotTo(Equal(bindplaneHealthPortName))
			Expect(p.ContainerPort).NotTo(Equal(bindplaneHealthPort))
		}
		for _, e := range c.Env {
			Expect(e.Name).NotTo(Equal(bindplaneProbePortEnvVar))
		}
	}

	expectHealthProbes := func(c corev1.Container) {
		Expect(c.Ports).To(ContainElement(corev1.ContainerPort{
			Name:          bindplaneHealthPortName,
			ContainerPort: 3002,
			Protocol:      corev1.ProtocolTCP,
		}))
		Expect(c.Env).To(ContainElement(corev1.EnvVar{Name: "BINDPLANE_PROBE_PORT", Value: "3002"}))
		Expect(c.StartupProbe).To(Equal(&corev1.Probe{
			ProbeHandler:        httpProbeHandler("/startupz"),
			InitialDelaySeconds: probeStartupInitialDelaySeconds,
			PeriodSeconds:       probeStartupPeriodSeconds,
			FailureThreshold:    probeStartupFailureThreshold,
			SuccessThreshold:    probeStartupSuccessThreshold,
			TimeoutSeconds:      probeStartupTimeoutSeconds,
		}))
		Expect(c.ReadinessProbe).To(Equal(&corev1.Probe{
			ProbeHandler:     httpProbeHandler("/readyz"),
			PeriodSeconds:    probePeriodSeconds,
			FailureThreshold: probeFailureThreshold,
			SuccessThreshold: probeSuccessThreshold,
			TimeoutSeconds:   probeTimeoutSeconds,
		}))
		Expect(c.LivenessProbe).To(Equal(&corev1.Probe{
			ProbeHandler:     httpProbeHandler("/livez"),
			PeriodSeconds:    probePeriodSeconds,
			FailureThreshold: probeFailureThreshold,
			SuccessThreshold: probeSuccessThreshold,
			TimeoutSeconds:   probeTimeoutSeconds,
		}))
		Expect(c.Lifecycle).To(BeNil())
	}

	var r *BindplaneReconciler

	BeforeEach(func() {
		r = newReconciler()
	})

	workloadEntries := []TableEntry{
		Entry("node deployment", nodeWorkload),
		Entry("node rollout", nodeRolloutWorkload),
		Entry("opamp deployment", opampWorkload),
		Entry("jobs deployment", jobsWorkload),
		Entry("nats statefulset", natsWorkload),
	}

	DescribeTable("uses TCP probes and preStop before the minimum version",
		func(w workload) {
			bp := w.newBindplane()
			bp.Spec.Version = legacyVersion
			expectLegacyHealth(w.container(r, bp), w.tcpPortName)
		},
		workloadEntries,
	)

	DescribeTable("uses HTTP probes on the health port from the minimum version",
		func(w workload) {
			bp := w.newBindplane()
			bp.Spec.Version = supportedVersion
			expectHealthProbes(w.container(r, bp))
		},
		workloadEntries,
	)

	It("places BINDPLANE_PROBE_PORT after extraEnv so the operator value wins", func() {
		bp := newBindplane()
		bp.Spec.Version = supportedVersion
		bp.Spec.Bindplane.ExtraEnv = []corev1.EnvVar{{Name: bindplaneProbePortEnvVar, Value: "9999"}}
		var last corev1.EnvVar
		for _, e := range r.nodeDeployment(bp).Spec.Template.Spec.Containers[0].Env {
			if e.Name == bindplaneProbePortEnvVar {
				last = e
			}
		}
		Expect(last.Value).To(Equal("3002"))
	})

	DescribeTable("gates on the resolved component image, not spec.version",
		func(w workload, image string) {
			bp := w.newBindplane()
			bp.Spec.Version = supportedVersion
			w.setImage(bp, image)
			expectLegacyHealth(w.container(r, bp), w.tcpPortName)
		},
		Entry("node deployment with 1.104.0 override", nodeWorkload, "ghcr.io/observiq/bindplane-ee:"+legacyVersion),
		Entry("node deployment with digest override", nodeWorkload, digestImage),
		Entry("node rollout with 1.104.0 override", nodeRolloutWorkload, "ghcr.io/observiq/bindplane-ee:"+legacyVersion),
		Entry("node rollout with digest override", nodeRolloutWorkload, digestImage),
		Entry("opamp deployment with 1.104.0 override", opampWorkload, "ghcr.io/observiq/bindplane-ee:"+legacyVersion),
		Entry("opamp deployment with digest override", opampWorkload, digestImage),
		Entry("jobs deployment with 1.104.0 override", jobsWorkload, "ghcr.io/observiq/bindplane-ee:"+legacyVersion),
		Entry("jobs deployment with digest override", jobsWorkload, digestImage),
		Entry("nats statefulset with 1.104.0 override", natsWorkload, "ghcr.io/observiq/bindplane-ee:"+legacyVersion),
		Entry("nats statefulset with digest override", natsWorkload, digestImage),
	)

	It("keeps the termination grace period unchanged", func() {
		bp := newBindplane()
		bp.Spec.Version = supportedVersion
		Expect(*r.nodeDeployment(bp).Spec.Template.Spec.TerminationGracePeriodSeconds).To(Equal(defaultTerminationGracePeriodSeconds))
		Expect(*r.natsStatefulSet(bp).Spec.Template.Spec.TerminationGracePeriodSeconds).To(Equal(defaultTerminationGracePeriodSeconds))
		Expect(*r.bindplaneJobsDeployment(bp).Spec.Template.Spec.TerminationGracePeriodSeconds).To(Equal(defaultTerminationGracePeriodSeconds))
	})

	It("lets a user podTemplate probe replace the operator HTTP probe", func() {
		bp := newBindplane()
		bp.Spec.Version = supportedVersion
		userProbe := &corev1.Probe{ProbeHandler: tcpProbeHandler(nodeHTTPPortName), PeriodSeconds: 30}
		bp.Spec.Bindplane.PodTemplate = &bindplanev1alpha1.PodTemplateSpec{
			PodTemplateSpec: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: nodeContainerName, ReadinessProbe: userProbe}},
				},
			},
		}
		c := r.nodeDeployment(bp).Spec.Template.Spec.Containers[0]
		Expect(c.ReadinessProbe).To(Equal(userProbe))
		Expect(c.StartupProbe.HTTPGet).NotTo(BeNil())
		Expect(c.LivenessProbe.HTTPGet).NotTo(BeNil())
	})

	It("leaves the jobs migrate Job without ports, probes, or the probe env", func() {
		bp := newBindplane()
		bp.Spec.Version = supportedVersion
		c := r.bindplaneJobsMigrateJob(bp).Spec.Template.Spec.Containers[0]
		Expect(c.Ports).To(BeEmpty())
		Expect(c.StartupProbe).To(BeNil())
		Expect(c.ReadinessProbe).To(BeNil())
		Expect(c.LivenessProbe).To(BeNil())
		Expect(c.Lifecycle).To(BeNil())
		for _, e := range c.Env {
			Expect(e.Name).NotTo(Equal(bindplaneProbePortEnvVar))
		}
	})

	It("does not expose the health port on any Service", func() {
		bp := newBindplane()
		bp.Spec.Version = supportedVersion
		bp.Spec.OpAMP = &bindplanev1alpha1.OpAMPComponentSpec{Enabled: true}
		services := []*corev1.Service{
			r.nodeService(bp, false),
			r.nodeService(bp, true),
			r.opampService(bp),
			r.natsService(bp),
			r.natsHeadlessService(bp),
		}
		for _, svc := range services {
			for _, p := range svc.Spec.Ports {
				Expect(p.Port).NotTo(Equal(bindplaneHealthPort), "service %s", svc.Name)
				Expect(p.TargetPort.IntVal).NotTo(Equal(bindplaneHealthPort), "service %s", svc.Name)
				Expect(p.TargetPort.StrVal).NotTo(Equal(bindplaneHealthPortName), "service %s", svc.Name)
				Expect(p.Name).NotTo(Equal(bindplaneHealthPortName), "service %s", svc.Name)
			}
		}
	})
})
