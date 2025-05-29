// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
//
//nolint:testpackage // Keeping the test in the same package due to dependencies on unexported fields.
package tinkerbell

import (
	"context"
	"errors"
	onboarding_types "github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/onboarding/types"
	om_testing "github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/testing"
	"github.com/stretchr/testify/mock"
	tink "github.com/tinkerbell/tink/api/v1alpha1"
	error_k8 "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

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
		name         string
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
			if err := DeleteHardware(tt.args.k8sNamespace,
				tt.args.name); (err != nil) != tt.wantErr {
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
			if err := CreateHardwareIfNotExists(tt.args.k8sNamespace, "test"); (err != nil) != tt.wantErr {
				t.Errorf("CreateHardwareIfNotExists() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteHardwareForHostIfExist_ErrorScenario(t *testing.T) {
	type args struct {
		ctx          context.Context
		k8sNamespace string
		name         string
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
			if err := DeleteHardware(tt.args.k8sNamespace, tt.args.name); (err != nil) != tt.wantErr {
				t.Errorf("DeleteHardwareForHostIfExist() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateWorkflowIfNotExists(t *testing.T) {
	type args struct {
		ctx      context.Context
		k8sCli   client.Client
		workflow *tink.Workflow
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
			name: "CreateWorkflow_Success",
			args: args{
				ctx:      context.Background(),
				k8sCli:   &mockClient,
				workflow: &tink.Workflow{},
			},
		},
		{
			name: "CreateWorkflow_ClientError",
			args: args{
				ctx:      context.Background(),
				k8sCli:   &mockClient1,
				workflow: &tink.Workflow{},
			},
			wantErr: true,
		},
		{
			name: "CreateWorkflow_WorkflowNotFound",
			args: args{
				ctx:      context.Background(),
				k8sCli:   &mockClient2,
				workflow: &tink.Workflow{},
			},
			wantErr: false,
		},
		{
			name: "CreateWorkflow_CreateError",
			args: args{
				ctx:      context.Background(),
				k8sCli:   &mockClient3,
				workflow: &tink.Workflow{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateWorkflowIfNotExists(tt.args.ctx, tt.args.k8sCli,
				tt.args.workflow); (err != nil) != tt.wantErr {
				t.Errorf("CreateWorkflowIfNotExists() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteProdWorkflowResourcesIfExist(t *testing.T) {
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
			name: "TestDeleteProdWorkflowResourcesIfExist_WithExistingResources",
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DeleteWorkflowIfExists(tt.args.ctx, tt.args.k8sNamespace,
				tt.args.hostUUID); (err != nil) != tt.wantErr {
				t.Errorf("DeleteWorkflowIfExists() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteProdWorkflowResourcesIfExist_Case(t *testing.T) {
	currK8sClientFactory := K8sClientFactory
	defer func() {
		K8sClientFactory = currK8sClientFactory
	}()
	tinkerbell.K8sClientFactory = om_testing.K8sCliMockFactory(false, false, false, false)
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
			name: "TestDeleteProdWorkflowResourcesIfExist_WithExistingResources",
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tinkerbell.DeleteWorkflowIfExists(tt.args.ctx, tt.args.k8sNamespace,
				tt.args.hostUUID); (err != nil) != tt.wantErr {
				t.Errorf("DeleteWorkflowIfExists() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
