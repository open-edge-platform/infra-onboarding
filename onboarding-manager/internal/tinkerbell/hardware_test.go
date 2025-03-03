// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
//
//nolint:testpackage // Keeping the test in the same package due to dependencies on unexported fields.
package tinkerbell

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/mock"
	tink "github.com/tinkerbell/tink/api/v1alpha1"
	error_k8 "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	onboarding_types "github.com/intel/infra-onboarding/onboarding-manager/internal/onboarding/types"
	om_testing "github.com/intel/infra-onboarding/onboarding-manager/internal/testing"
)

func TestNewHardware(t *testing.T) {
	type args struct {
		name         string
		ns           string
		id           string
		ip           string
		gateway      string
		osResourceID string
	}
	tests := []struct {
		name string
		args args
		want *tink.Hardware
	}{
		{
			name: "Create new hardware with default values",
			args: args{},
			want: &tink.Hardware{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewHardware(tt.args.name, tt.args.ns, tt.args.id,
				tt.args.ip, tt.args.gateway, tt.args.osResourceID); reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewHardware() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newK8SClient(t *testing.T) {
	currK8sClientFactory := K8sClientFactory
	defer func() {
		K8sClientFactory = currK8sClientFactory
	}()
	K8sClientFactory = om_testing.K8sCliMockFactory(false, false, false, false)
	tests := []struct {
		name    string
		want    client.Client
		wantErr bool
	}{
		{
			name:    "Valid Kubernetes client initialization",
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newK8SClient()
			if (err != nil) != tt.wantErr {
				t.Errorf("newK8SClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newK8SClient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeleteHardwareForHostIfExist(t *testing.T) {
	currK8sClientFactory := K8sClientFactory
	defer func() {
		K8sClientFactory = currK8sClientFactory
	}()
	K8sClientFactory = om_testing.K8sCliMockFactory(false, false, false, false)
	type args struct {
		ctx          context.Context
		k8sNamespace string
		hostUUID     string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "TestDeleteHardwareForHostIfExist_KubernetesEnvironment",
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DeleteHardwareForHostIfExist(tt.args.ctx, tt.args.k8sNamespace,
				tt.args.hostUUID); (err != nil) != tt.wantErr {
				t.Errorf("DeleteHardwareForHostIfExist() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateHardwareIfNotExists(t *testing.T) {
	type args struct {
		ctx          context.Context
		k8sCli       client.Client
		k8sNamespace string
		deviceInfo   onboarding_types.DeviceInfo
		osResourceID string
	}
	mockClient := om_testing.MockK8sClient{}
	mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockClient1 := om_testing.MockK8sClient{}
	mockClient1.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("err"))
	mockClient2 := om_testing.MockK8sClient{}
	mockClient2.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(error_k8.NewNotFound(schema.GroupResource{Group: "example.com", Resource: "myresource"}, "resource-name"))
	mockClient2.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockClient3 := om_testing.MockK8sClient{}
	mockClient3.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(error_k8.NewNotFound(schema.GroupResource{Group: "example.com", Resource: "myresource"}, "resource-name"))
	mockClient3.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("err"))
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "CreateHardwareSuccess",
			args: args{
				ctx:    context.Background(),
				k8sCli: &mockClient,
			},
		},
		{
			name: "GetHardwareError",
			args: args{
				ctx:    context.Background(),
				k8sCli: &mockClient1,
			},
			wantErr: true,
		},
		{
			name: "HardwareNotFoundCreate",
			args: args{
				ctx:    context.Background(),
				k8sCli: &mockClient2,
			},
			wantErr: false,
		},
		{
			name: "CreateHardwareError",
			args: args{
				ctx:    context.Background(),
				k8sCli: &mockClient3,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateHardwareIfNotExists(tt.args.ctx, tt.args.k8sCli, tt.args.k8sNamespace, tt.args.deviceInfo,
				tt.args.osResourceID); (err != nil) != tt.wantErr {
				t.Errorf("CreateHardwareIfNotExists() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetProdWorkflowName(t *testing.T) {
	type args struct {
		uuid string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "TestGetProdWorkflowNameUUID",
			want: "workflow--prod",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetProdWorkflowName(tt.args.uuid); got != tt.want {
				t.Errorf("GetProdWorkflowName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeleteHardwareForHostIfExist_ErrorScenario(t *testing.T) {
	type args struct {
		ctx          context.Context
		k8sNamespace string
		hostUUID     string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "Kubeclient error",
			args:    args{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DeleteHardwareForHostIfExist(tt.args.ctx, tt.args.k8sNamespace,
				tt.args.hostUUID); (err != nil) != tt.wantErr {
				t.Errorf("DeleteHardwareForHostIfExist() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
