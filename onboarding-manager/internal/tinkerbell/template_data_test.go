// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tinkerbell_test

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v2"

	osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
	dkam_testing "github.com/open-edge-platform/infra-onboarding/dkam/testing"
	onboarding_types "github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/onboarding/types"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/tinkerbell"
)

func TestNewTemplateDataUbuntu(t *testing.T) {
	dkam_testing.PrepareTestCaCertificateFile(t)
	dkam_testing.PrepareTestInfraConfig(t)

	type args struct {
		name       string
		deviceInfo onboarding_types.DeviceInfo
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
			name: "Success",
			args: args{
				name: "TestWorkflow",
				deviceInfo: onboarding_types.DeviceInfo{
					OsType:   osv1.OsType_OS_TYPE_MUTABLE,
					TenantID: "test-tenantid",
					Hostname: "test-hostname",
				},
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
