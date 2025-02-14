// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tinkerbell

import (
	"reflect"
	"testing"

	"github.com/intel/infra-onboarding/onboarding-manager/internal/onboardingmgr/utils"
)

func TestNewTemplateDataProdBKC(t *testing.T) {
	type args struct {
		name       string
		deviceInfo utils.DeviceInfo
		enableDI   bool
	}
	wf := Workflow{}
	want, _ := marshalWorkflow(&wf)
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
				deviceInfo: utils.DeviceInfo{},
				enableDI:   false,
			},
			want:    want,
			wantErr: false,
		},
		{
			name: "Success - DI enabled",
			args: args{
				name:       "TestWorkflow",
				deviceInfo: utils.DeviceInfo{},
			},
			want:    want,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTemplateDataUbuntu(tt.args.name, tt.args.deviceInfo)
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
