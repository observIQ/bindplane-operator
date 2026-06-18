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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

const testOpVolName = "op-vol"

// --- appendExtraVolumes ---

func TestAppendExtraVolumes_NilExtra(t *testing.T) {
	op := []corev1.Volume{{Name: testOpVolName}}
	result := appendExtraVolumes(op, nil)
	if len(result) != 1 || result[0].Name != testOpVolName {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestAppendExtraVolumes_EmptyExtra(t *testing.T) {
	op := []corev1.Volume{{Name: testOpVolName}}
	result := appendExtraVolumes(op, []corev1.Volume{})
	if len(result) != 1 || result[0].Name != testOpVolName {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestAppendExtraVolumes_OperatorFirst(t *testing.T) {
	op := []corev1.Volume{{Name: testOpVolName}}
	extra := []corev1.Volume{{Name: "user-vol"}}
	result := appendExtraVolumes(op, extra)
	if len(result) != 2 {
		t.Fatalf("expected 2 volumes, got %d", len(result))
	}
	if result[0].Name != testOpVolName {
		t.Errorf("expected %s first, got %s", testOpVolName, result[0].Name)
	}
	if result[1].Name != "user-vol" {
		t.Errorf("expected user-vol second, got %s", result[1].Name)
	}
}

func TestAppendExtraVolumes_NilOperator(t *testing.T) {
	extra := []corev1.Volume{{Name: "user-vol"}}
	result := appendExtraVolumes(nil, extra)
	if len(result) != 1 || result[0].Name != "user-vol" {
		t.Errorf("unexpected result: %v", result)
	}
}

// --- appendExtraVolumeMounts ---

func TestAppendExtraVolumeMounts_NilExtra(t *testing.T) {
	op := []corev1.VolumeMount{{Name: "op-vol", MountPath: "/op"}}
	result := appendExtraVolumeMounts(op, nil)
	if len(result) != 1 || result[0].Name != "op-vol" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestAppendExtraVolumeMounts_EmptyExtra(t *testing.T) {
	op := []corev1.VolumeMount{{Name: "op-vol", MountPath: "/op"}}
	result := appendExtraVolumeMounts(op, []corev1.VolumeMount{})
	if len(result) != 1 || result[0].Name != "op-vol" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestAppendExtraVolumeMounts_OperatorFirst(t *testing.T) {
	op := []corev1.VolumeMount{{Name: "op-vol", MountPath: "/op"}}
	extra := []corev1.VolumeMount{{Name: "user-vol", MountPath: "/user"}}
	result := appendExtraVolumeMounts(op, extra)
	if len(result) != 2 {
		t.Fatalf("expected 2 mounts, got %d", len(result))
	}
	if result[0].MountPath != "/op" {
		t.Errorf("expected /op first, got %s", result[0].MountPath)
	}
	if result[1].MountPath != "/user" {
		t.Errorf("expected /user second, got %s", result[1].MountPath)
	}
}

// --- accessor nil safety ---

func newMinimalBindplane() *bindplanev1alpha1.Bindplane {
	replicas := int32(1)
	natsReplicas := int32(2)
	taReplicas := int32(2)
	return &bindplanev1alpha1.Bindplane{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: bindplanev1alpha1.BindplaneSpec{
			Version: "1.99.0",
			Config: bindplanev1alpha1.BindplaneConfigSpec{
				Store: bindplanev1alpha1.StoreConfig{
					Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"},
				},
			},
			Bindplane: bindplanev1alpha1.BindplaneComponentSpec{
				Replicas: &replicas,
			},
			Nats: &bindplanev1alpha1.NatsComponentSpec{
				Replicas: &natsReplicas,
			},
			TransformAgent: &bindplanev1alpha1.TransformAgentComponentSpec{
				Replicas: &taReplicas,
			},
		},
	}
}

func TestGetOpAMPExtraVolumes_NilComponent(t *testing.T) {
	bp := newMinimalBindplane()
	if vols := getOpAMPExtraVolumes(bp); vols != nil {
		t.Errorf("expected nil, got %v", vols)
	}
}

func TestGetOpAMPExtraVolumeMounts_NilComponent(t *testing.T) {
	bp := newMinimalBindplane()
	if mounts := getOpAMPExtraVolumeMounts(bp); mounts != nil {
		t.Errorf("expected nil, got %v", mounts)
	}
}

func TestGetBindplaneJobsExtraVolumes_NilComponent(t *testing.T) {
	bp := newMinimalBindplane()
	if vols := getBindplaneJobsExtraVolumes(bp); vols != nil {
		t.Errorf("expected nil, got %v", vols)
	}
}

func TestGetBindplaneJobsMigrateExtraVolumes_NilComponent(t *testing.T) {
	bp := newMinimalBindplane()
	if vols := getBindplaneJobsMigrateExtraVolumes(bp); vols != nil {
		t.Errorf("expected nil, got %v", vols)
	}
}

func TestGetNatsExtraVolumes_NilComponent(t *testing.T) {
	bp := newMinimalBindplane()
	if vols := getNatsExtraVolumes(bp); vols != nil {
		t.Errorf("expected nil, got %v", vols)
	}
}

func TestGetTransformAgentExtraVolumes_NilComponent(t *testing.T) {
	bp := newMinimalBindplane()
	if vols := getTransformAgentExtraVolumes(bp); vols != nil {
		t.Errorf("expected nil, got %v", vols)
	}
}

func TestGetTSDBExtraVolumes_NilComponent(t *testing.T) {
	bp := newMinimalBindplane()
	if vols := getTSDBExtraVolumes(bp); vols != nil {
		t.Errorf("expected nil, got %v", vols)
	}
}

func TestGetTSDBExtraVolumeMounts_NilComponent(t *testing.T) {
	bp := newMinimalBindplane()
	if mounts := getTSDBExtraVolumeMounts(bp); mounts != nil {
		t.Errorf("expected nil, got %v", mounts)
	}
}

// --- integration: volume injection reaches the pod spec ---

func TestNodeDeployment_ExtraVolumesInjected(t *testing.T) {
	bp := newMinimalBindplane()
	bp.Spec.Bindplane.ExtraVolumes = []corev1.Volume{
		{
			Name:         "redis-ca",
			VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "redis-ca-secret"}},
		},
	}
	bp.Spec.Bindplane.ExtraVolumeMounts = []corev1.VolumeMount{
		{Name: "redis-ca", MountPath: "/etc/redis-ca"},
	}

	r := &BindplaneReconciler{}
	dep := r.nodeDeployment(bp)

	assertVolumePresent(t, dep.Spec.Template.Spec.Volumes, "redis-ca")
	assertMountPresent(t, dep.Spec.Template.Spec.Containers[0].VolumeMounts, "/etc/redis-ca")
}

func TestJobsDeployment_ExtraVolumesInjected(t *testing.T) {
	bp := newMinimalBindplane()
	bp.Spec.BindplaneJobs = &bindplanev1alpha1.BindplaneJobsComponentSpec{
		ExtraVolumes: []corev1.Volume{
			{Name: "redis-ca", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "redis-ca-secret"}}},
		},
		ExtraVolumeMounts: []corev1.VolumeMount{
			{Name: "redis-ca", MountPath: "/etc/redis-ca"},
		},
	}

	r := &BindplaneReconciler{}
	dep := r.bindplaneJobsDeployment(bp)

	assertVolumePresent(t, dep.Spec.Template.Spec.Volumes, "redis-ca")
	assertMountPresent(t, dep.Spec.Template.Spec.Containers[0].VolumeMounts, "/etc/redis-ca")
}

func TestTSDBStatefulSet_ExtraVolumesInjected(t *testing.T) {
	bp := newMinimalBindplane()
	bp.Spec.TSDB = &bindplanev1alpha1.TSDBComponentSpec{
		ExtraVolumes: []corev1.Volume{
			{Name: "prom-rules", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "my-rules"}}}},
		},
		ExtraVolumeMounts: []corev1.VolumeMount{
			{Name: "prom-rules", MountPath: "/etc/prometheus/rules.d"},
		},
	}

	r := &BindplaneReconciler{}
	ss := r.tsdbStatefulSet(bp)

	assertVolumePresent(t, ss.Spec.Template.Spec.Volumes, "prom-rules")
	assertMountPresent(t, ss.Spec.Template.Spec.Containers[0].VolumeMounts, "/etc/prometheus/rules.d")
}

func assertVolumePresent(t *testing.T, vols []corev1.Volume, name string) {
	t.Helper()
	for _, v := range vols {
		if v.Name == name {
			return
		}
	}
	t.Errorf("volume %q not found in %v", name, vols)
}

func assertMountPresent(t *testing.T, mounts []corev1.VolumeMount, mountPath string) {
	t.Helper()
	for _, m := range mounts {
		if m.MountPath == mountPath {
			return
		}
	}
	t.Errorf("mount %q not found in %v", mountPath, mounts)
}
