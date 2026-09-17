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

func TestAnnotationEnabled(t *testing.T) {
	const key = "k8s.bindplane.com/example"

	cases := []struct {
		name        string
		annotations map[string]string
		want        bool
	}{
		{name: "nil map", annotations: nil, want: false},
		{name: "absent", annotations: map[string]string{"other": "true"}, want: false},
		{name: "empty value", annotations: map[string]string{key: ""}, want: false},
		{name: "lowercase true", annotations: map[string]string{key: "true"}, want: true},
		{name: "titlecase True", annotations: map[string]string{key: "True"}, want: true},
		{name: "uppercase TRUE", annotations: map[string]string{key: "TRUE"}, want: true},
		{name: "shorthand t", annotations: map[string]string{key: "t"}, want: true},
		{name: "numeric 1", annotations: map[string]string{key: "1"}, want: true},
		{name: "lowercase false", annotations: map[string]string{key: "false"}, want: false},
		{name: "uppercase FALSE", annotations: map[string]string{key: "FALSE"}, want: false},
		{name: "numeric 0", annotations: map[string]string{key: "0"}, want: false},
		{name: "unparseable yes", annotations: map[string]string{key: "yes"}, want: false},
		{name: "unparseable typo", annotations: map[string]string{key: "ture"}, want: false},
		{name: "padded value", annotations: map[string]string{key: " true "}, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := annotationEnabled(tc.annotations, key); got != tc.want {
				t.Errorf("annotationEnabled(%q) = %v, want %v", tc.annotations[key], got, tc.want)
			}
		})
	}
}

func TestSkipMigrateCheck(t *testing.T) {
	cases := []struct {
		name  string
		value string
		set   bool
		want  bool
	}{
		{name: "annotation absent", set: false, want: false},
		{name: "true", set: true, value: "true", want: true},
		{name: "True", set: true, value: "True", want: true},
		{name: "1", set: true, value: "1", want: true},
		{name: "false", set: true, value: "false", want: false},
		{name: "garbage", set: true, value: "maybe", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bp := &bindplanev1alpha1.Bindplane{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			}
			if tc.set {
				bp.Annotations = map[string]string{annotationSkipMigrateCheck: tc.value}
			}
			if got := skipMigrateCheck(bp); got != tc.want {
				t.Errorf("skipMigrateCheck() = %v, want %v", got, tc.want)
			}
		})
	}
}
