// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tinkerbell_test

import (
	"reflect"
	"testing"

	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/tinkerbell"
	tink "github.com/tinkerbell/tink/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewTemplate(t *testing.T) {
	type args struct {
		tpData string
		name   string
		ns     string
	}
	tests := []struct {
		name string
		args args
		want *tink.Template
	}{
		{
			name: "Test case 1",
			args: args{
				tpData: "templateData",
				name:   "exampleTemplate",
				ns:     "exampleNamespace",
			},
			want: &tink.Template{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Template",
					APIVersion: "tinkerbell.org/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "exampleTemplate",
					Namespace: "exampleNamespace",
				},
				Spec: tink.TemplateSpec{
					Data: func() *string {
						s := "templateData"
						return &s
					}(),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tinkerbell.NewTemplate(tt.args.tpData, tt.args.name, tt.args.ns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}
