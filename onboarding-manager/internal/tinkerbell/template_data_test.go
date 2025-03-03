// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tinkerbell_test

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v2"

	onboarding_types "github.com/intel/infra-onboarding/onboarding-manager/internal/onboarding/types"
	"github.com/intel/infra-onboarding/onboarding-manager/internal/tinkerbell"
)

func TestNewTemplateDataProdBKC(t *testing.T) {
	type args struct {
		name       string
		deviceInfo onboarding_types.DeviceInfo
		enableDI   bool
	}
	wf := tinkerbell.Workflow{}
	want, marshalWorkflowError := marshalWorkflow(&wf)
	if marshalWorkflowError != nil {
		t.Errorf("marshalWorkflowError%v", marshalWorkflowError)
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "Success - DI disabled",
			args: args{
				name:       "TestWorkflow",
				deviceInfo: onboarding_types.DeviceInfo{},
				enableDI:   false,
			},
			want:    want,
			wantErr: false,
		},
		{
			name: "Success - DI enabled",
			args: args{
				name:       "TestWorkflow",
				deviceInfo: onboarding_types.DeviceInfo{},
			},
			want:    want,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tinkerbell.NewTemplateDataUbuntu(tt.args.name, tt.args.deviceInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTemplateDataUbuntu() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTemplateDataUbuntu() = %v, want %v", got, tt.want)
			}
		})
	}
}

func marshalWorkflow(wf *tinkerbell.Workflow) ([]byte, error) {
	return yaml.Marshal(wf)
}
