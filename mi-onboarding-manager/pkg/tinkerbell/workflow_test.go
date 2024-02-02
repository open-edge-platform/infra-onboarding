// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package tinkerbell

import (
	"reflect"
	"testing"

	tink "github.com/tinkerbell/tink/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewWorkflow(t *testing.T) {
	type args struct {
		name string
		ns   string
		mac  string
	}
	tests := []struct {
		name string
		args args
		want *tink.Workflow
	}{
		{
			name: "Test Case 1",
			args: args{
				name: "workflow1",
				ns:   "namespace1",
				mac:  "00:11:22:33:44:55",
			},
			want: &tink.Workflow{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Workflow",
					APIVersion: "tinkerbell.org/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "workflow1",
					Namespace: "namespace1",
				},
				Spec: tink.WorkflowSpec{
					HardwareMap: map[string]string{
						"device_1": "00:11:22:33:44:55",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewWorkflow(tt.args.name, tt.args.ns, tt.args.mac); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewWorkflow() = %v, want %v", got, tt.want)
			}
		})
	}
}

