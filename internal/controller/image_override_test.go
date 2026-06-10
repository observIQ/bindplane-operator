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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

const testVersion = "1.99.1"
const customImage = "myregistry.example.com/bindplane-ee:custom-tag"

func newImageTestBindplane() *bindplanev1alpha1.Bindplane {
	return &bindplanev1alpha1.Bindplane{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: bindplanev1alpha1.BindplaneSpec{
			Version: testVersion,
		},
	}
}

func TestGetNodeImage_DefaultsToVersion(t *testing.T) {
	bp := newImageTestBindplane()
	want := "ghcr.io/observiq/bindplane-ee:" + testVersion
	if got := getNodeImage(bp); got != want {
		t.Errorf("getNodeImage() = %q, want %q", got, want)
	}
}

func TestGetNodeImage_UsesOverride(t *testing.T) {
	bp := newImageTestBindplane()
	bp.Spec.Bindplane.Image = customImage
	if got := getNodeImage(bp); got != customImage {
		t.Errorf("getNodeImage() = %q, want %q", got, customImage)
	}
}

func TestGetOpAMPImage_NilSpec_DefaultsToVersion(t *testing.T) {
	bp := newImageTestBindplane()
	want := "ghcr.io/observiq/bindplane-ee:" + testVersion
	if got := getOpAMPImage(bp); got != want {
		t.Errorf("getOpAMPImage() with nil OpAMP = %q, want %q", got, want)
	}
}

func TestGetOpAMPImage_UsesOverride(t *testing.T) {
	bp := newImageTestBindplane()
	bp.Spec.OpAMP = &bindplanev1alpha1.OpAMPComponentSpec{Image: customImage}
	if got := getOpAMPImage(bp); got != customImage {
		t.Errorf("getOpAMPImage() = %q, want %q", got, customImage)
	}
}

func TestGetNatsImage_NilSpec_DefaultsToVersion(t *testing.T) {
	bp := newImageTestBindplane()
	want := "ghcr.io/observiq/bindplane-ee:" + testVersion
	if got := getNatsImage(bp); got != want {
		t.Errorf("getNatsImage() with nil Nats = %q, want %q", got, want)
	}
}

func TestGetNatsImage_UsesOverride(t *testing.T) {
	bp := newImageTestBindplane()
	bp.Spec.Nats = &bindplanev1alpha1.NatsComponentSpec{Image: customImage}
	if got := getNatsImage(bp); got != customImage {
		t.Errorf("getNatsImage() = %q, want %q", got, customImage)
	}
}

func TestGetBindplaneJobsImage_NilSpec_DefaultsToVersion(t *testing.T) {
	bp := newImageTestBindplane()
	want := "ghcr.io/observiq/bindplane-ee:" + testVersion
	if got := getBindplaneJobsImage(bp); got != want {
		t.Errorf("getBindplaneJobsImage() with nil BindplaneJobs = %q, want %q", got, want)
	}
}

func TestGetBindplaneJobsImage_UsesOverride(t *testing.T) {
	bp := newImageTestBindplane()
	bp.Spec.BindplaneJobs = &bindplanev1alpha1.BindplaneJobsComponentSpec{Image: customImage}
	if got := getBindplaneJobsImage(bp); got != customImage {
		t.Errorf("getBindplaneJobsImage() = %q, want %q", got, customImage)
	}
}

func TestGetBindplaneJobsMigrateImage_NilSpec_DefaultsToVersion(t *testing.T) {
	bp := newImageTestBindplane()
	want := "ghcr.io/observiq/bindplane-ee:" + testVersion
	if got := getBindplaneJobsMigrateImage(bp); got != want {
		t.Errorf("getBindplaneJobsMigrateImage() with nil BindplaneJobsMigrate = %q, want %q", got, want)
	}
}

func TestGetBindplaneJobsMigrateImage_UsesOverride(t *testing.T) {
	bp := newImageTestBindplane()
	bp.Spec.BindplaneJobsMigrate = &bindplanev1alpha1.BindplaneJobsMigrateComponentSpec{Image: customImage}
	if got := getBindplaneJobsMigrateImage(bp); got != customImage {
		t.Errorf("getBindplaneJobsMigrateImage() = %q, want %q", got, customImage)
	}
}

func TestGetTransformAgentImage_NilSpec_DefaultsToVersion(t *testing.T) {
	bp := newImageTestBindplane()
	want := "ghcr.io/observiq/bindplane-transform-agent:" + testVersion + "-bindplane"
	if got := getTransformAgentImage(bp); got != want {
		t.Errorf("getTransformAgentImage() with nil TransformAgent = %q, want %q", got, want)
	}
}

func TestGetTransformAgentImage_UsesOverride(t *testing.T) {
	bp := newImageTestBindplane()
	customTA := "myregistry.example.com/bindplane-transform-agent:custom-tag"
	bp.Spec.TransformAgent = &bindplanev1alpha1.TransformAgentComponentSpec{Image: customTA}
	if got := getTransformAgentImage(bp); got != customTA {
		t.Errorf("getTransformAgentImage() = %q, want %q", got, customTA)
	}
}

func TestGetTSDBImage_NilSpec_DefaultsToVersion(t *testing.T) {
	bp := newImageTestBindplane()
	want := "ghcr.io/observiq/bindplane-prometheus:" + testVersion
	if got := getTSDBImage(bp); got != want {
		t.Errorf("getTSDBImage() with nil TSDB = %q, want %q", got, want)
	}
}

func TestGetTSDBImage_UsesOverride(t *testing.T) {
	bp := newImageTestBindplane()
	customTSDB := "myregistry.example.com/bindplane-prometheus:custom-tag"
	bp.Spec.TSDB = &bindplanev1alpha1.TSDBComponentSpec{Image: customTSDB}
	if got := getTSDBImage(bp); got != customTSDB {
		t.Errorf("getTSDBImage() = %q, want %q", got, customTSDB)
	}
}

func TestResolveImage_EmptyOverrideFallsBack(t *testing.T) {
	if got := resolveImage("", "default"); got != "default" {
		t.Errorf("resolveImage(\"\", \"default\") = %q, want %q", got, "default")
	}
}

func TestResolveImage_NonEmptyOverrideWins(t *testing.T) {
	if got := resolveImage("override", "default"); got != "override" {
		t.Errorf("resolveImage(\"override\", \"default\") = %q, want %q", got, "override")
	}
}

func TestUpdateImageStatus_DefaultImages(t *testing.T) {
	bp := newImageTestBindplane()
	updateImageStatus(bp)

	wantEE := "ghcr.io/observiq/bindplane-ee:" + testVersion
	wantTA := "ghcr.io/observiq/bindplane-transform-agent:" + testVersion + "-bindplane"
	wantTSDB := "ghcr.io/observiq/bindplane-prometheus:" + testVersion

	if bp.Status.Components.Bindplane.Image != wantEE {
		t.Errorf("Bindplane.Image = %q, want %q", bp.Status.Components.Bindplane.Image, wantEE)
	}
	if bp.Status.Components.Jobs.Image != wantEE {
		t.Errorf("Jobs.Image = %q, want %q", bp.Status.Components.Jobs.Image, wantEE)
	}
	if bp.Status.Components.Nats.Image != wantEE {
		t.Errorf("Nats.Image = %q, want %q", bp.Status.Components.Nats.Image, wantEE)
	}
	if bp.Status.Components.TransformAgent.Image != wantTA {
		t.Errorf("TransformAgent.Image = %q, want %q", bp.Status.Components.TransformAgent.Image, wantTA)
	}
	if bp.Status.Components.TSDB.Image != wantTSDB {
		t.Errorf("TSDB.Image = %q, want %q", bp.Status.Components.TSDB.Image, wantTSDB)
	}
	// OpAMP disabled by default — field must be empty.
	if bp.Status.Components.OpAMP.Image != "" {
		t.Errorf("OpAMP.Image = %q, want empty (OpAMP not enabled)", bp.Status.Components.OpAMP.Image)
	}
	// JobsMigrate is managed by migration gate, not updateImageStatus.
	if bp.Status.Components.JobsMigrate.Image != "" {
		t.Errorf("JobsMigrate.Image = %q, want empty (set by migration gate only)", bp.Status.Components.JobsMigrate.Image)
	}
}

func TestUpdateImageStatus_OpAMPEnabled(t *testing.T) {
	bp := newImageTestBindplane()
	bp.Spec.OpAMP = &bindplanev1alpha1.OpAMPComponentSpec{Enabled: true}
	updateImageStatus(bp)

	want := "ghcr.io/observiq/bindplane-ee:" + testVersion
	if bp.Status.Components.OpAMP.Image != want {
		t.Errorf("OpAMP.Image = %q, want %q", bp.Status.Components.OpAMP.Image, want)
	}
}

func TestUpdateImageStatus_OpAMPDisabled_ClearsField(t *testing.T) {
	bp := newImageTestBindplane()
	bp.Status.Components.OpAMP.Image = "stale-image"
	bp.Spec.OpAMP = &bindplanev1alpha1.OpAMPComponentSpec{Enabled: false}
	updateImageStatus(bp)

	if bp.Status.Components.OpAMP.Image != "" {
		t.Errorf("OpAMP.Image = %q, want empty when OpAMP disabled", bp.Status.Components.OpAMP.Image)
	}
}

func TestUpdateImageStatus_TSDBRemote_ClearsField(t *testing.T) {
	bp := newImageTestBindplane()
	bp.Status.Components.TSDB.Image = "stale-image"
	bp.Spec.Config.TSDB = &bindplanev1alpha1.TSDBConfig{
		Remote: &bindplanev1alpha1.TSDBRemoteConfig{Enable: true},
	}
	updateImageStatus(bp)

	if bp.Status.Components.TSDB.Image != "" {
		t.Errorf("TSDB.Image = %q, want empty when remote TSDB enabled", bp.Status.Components.TSDB.Image)
	}
}

func TestUpdateImageStatus_ImageOverride(t *testing.T) {
	bp := newImageTestBindplane()
	bp.Spec.Bindplane.Image = customImage
	updateImageStatus(bp)

	if bp.Status.Components.Bindplane.Image != customImage {
		t.Errorf("Bindplane.Image = %q, want %q (override)", bp.Status.Components.Bindplane.Image, customImage)
	}
}
