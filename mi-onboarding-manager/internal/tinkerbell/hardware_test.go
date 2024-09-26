// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package tinkerbell

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/common"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
	om_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/testing"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/flags"
	"github.com/stretchr/testify/mock"
	tink "github.com/tinkerbell/tink/api/v1alpha1"
	error_k8 "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestNewHardware(t *testing.T) {
	type args struct {
		name         string
		ns           string
		id           string
		device       string
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
			if got := NewHardware(tt.args.name, tt.args.ns, tt.args.id, tt.args.device,
				tt.args.ip, tt.args.gateway, tt.args.osResourceID); reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewHardware() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newK8SClient(t *testing.T) {
	currK8sClientFactory := K8sClientFactory
	currFlagEnableDeviceInitialization := *flags.FlagDisableCredentialsManagement
	defer func() {
		K8sClientFactory = currK8sClientFactory
		*common.FlagEnableDeviceInitialization = currFlagEnableDeviceInitialization
	}()
	*common.FlagEnableDeviceInitialization = true
	K8sClientFactory = om_testing.K8sCliMockFactory(false, false, false)
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
	currFlagEnableDeviceInitialization := *flags.FlagDisableCredentialsManagement
	defer func() {
		K8sClientFactory = currK8sClientFactory
		*common.FlagEnableDeviceInitialization = currFlagEnableDeviceInitialization
	}()
	*common.FlagEnableDeviceInitialization = true
	K8sClientFactory = om_testing.K8sCliMockFactory(false, false, false)
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
			if err := DeleteHardwareForHostIfExist(tt.args.ctx, tt.args.k8sNamespace, tt.args.hostUUID); (err != nil) != tt.wantErr {
				t.Errorf("DeleteHardwareForHostIfExist() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type MockClient struct {
	mock.Mock
}

func (m MockClient) Scheme() *runtime.Scheme {
	args := m.Called()
	return args.Get(0).(*runtime.Scheme)
}

func (m MockClient) RESTMapper() meta.RESTMapper {
	args := m.Called()
	return args.Get(0).(meta.RESTMapper)
}

func (m MockClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	args := m.Called(obj)
	return args.Get(0).(schema.GroupVersionKind), args.Error(1)
}

func (m MockClient) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	args := m.Called(obj)
	return args.Bool(0), args.Error(1)
}

func (m MockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	args := m.Called(ctx, key, obj, opts)
	return args.Error(0)
}

func (m MockClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	args := m.Called(ctx, list, opts)
	return args.Error(0)
}

func (m MockClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m MockClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m MockClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m MockClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	args := m.Called(ctx, obj, patch, opts)
	return args.Error(0)
}

func (m MockClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m MockClient) Status() client.SubResourceWriter {
	args := m.Called()
	return args.Get(0).(client.SubResourceWriter)
}

func (m MockClient) SubResource(subResource string) client.SubResourceClient {
	args := m.Called(subResource)
	return args.Get(0).(client.SubResourceClient)
}

func TestCreateHardwareIfNotExists(t *testing.T) {
	type args struct {
		ctx          context.Context
		k8sCli       client.Client
		k8sNamespace string
		deviceInfo   utils.DeviceInfo
		osResourceID string
	}
	mockClient := MockClient{}
	mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockClient1 := MockClient{}
	mockClient1.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("err"))
	mockClient2 := MockClient{}
	mockClient2.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(error_k8.NewNotFound(schema.GroupResource{Group: "example.com", Resource: "myresource"}, "resource-name"))
	mockClient2.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockClient3 := MockClient{}
	mockClient3.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(error_k8.NewNotFound(schema.GroupResource{Group: "example.com", Resource: "myresource"}, "resource-name"))
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
				k8sCli: mockClient,
			},
		},
		{
			name: "GetHardwareError",
			args: args{
				ctx:    context.Background(),
				k8sCli: mockClient1,
			},
			wantErr: true,
		},
		{
			name: "HardwareNotFoundCreate",
			args: args{
				ctx:    context.Background(),
				k8sCli: mockClient2,
			},
			wantErr: false,
		},
		{
			name: "CreateHardwareError",
			args: args{
				ctx:    context.Background(),
				k8sCli: mockClient3,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateHardwareIfNotExists(tt.args.ctx, tt.args.k8sCli, tt.args.k8sNamespace, tt.args.deviceInfo, tt.args.osResourceID); (err != nil) != tt.wantErr {
				t.Errorf("CreateHardwareIfNotExists() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetDIWorkflowName(t *testing.T) {
	type args struct {
		uuid string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "TestGetDIWorkflowNameUUID",
			want: "di-workflow-",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetDIWorkflowName(tt.args.uuid); got != tt.want {
				t.Errorf("GetDIWorkflowName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRebootWorkflowName(t *testing.T) {
	type args struct {
		uuid string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "TestGetRebootWorkflowNameUUID",
			want: "reboot-workflow-",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetRebootWorkflowName(tt.args.uuid); got != tt.want {
				t.Errorf("GetRebootWorkflowName() = %v, want %v", got, tt.want)
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
			if err := DeleteHardwareForHostIfExist(tt.args.ctx, tt.args.k8sNamespace, tt.args.hostUUID); (err != nil) != tt.wantErr {
				t.Errorf("DeleteHardwareForHostIfExist() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
