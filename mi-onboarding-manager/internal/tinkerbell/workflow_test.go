// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package tinkerbell

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/common"
	om_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/testing"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/flags"
	"github.com/stretchr/testify/mock"
	tink "github.com/tinkerbell/tink/api/v1alpha1"
	error "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestNewWorkflow(t *testing.T) {
	type args struct {
		name        string
		ns          string
		mac         string
		hardwareRef string
		templateRef string
	}
	tests := []struct {
		name string
		args args
		want *tink.Workflow
	}{
		{
			name: "TestNewWorkflow_Creation_Success",
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
			if got := NewWorkflow(tt.args.name, tt.args.ns, tt.args.mac, tt.args.hardwareRef, tt.args.templateRef); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewWorkflow() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeleteDIWorkflowResourcesIfExist(t *testing.T) {
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
			name:    "Test delete DI workflow resources IfExists -NoError",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DeleteDIWorkflowResourcesIfExist(tt.args.ctx, tt.args.k8sNamespace, tt.args.hostUUID); (err != nil) != tt.wantErr {
				t.Errorf("DeleteDIWorkflowResourcesIfExist() error = %v, wantErr %v", err, tt.wantErr)
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
	mockClient := MockClient{}
	mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockClient1 := MockClient{}
	mockClient1.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("err"))
	mockClient2 := MockClient{}
	mockClient2.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(error.NewNotFound(schema.GroupResource{Group: "example.com", Resource: "myresource"}, "resource-name"))
	mockClient2.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockClient3 := MockClient{}
	mockClient3.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(error.NewNotFound(schema.GroupResource{Group: "example.com", Resource: "myresource"}, "resource-name"))
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
				k8sCli:   mockClient,
				workflow: &tink.Workflow{},
			},
		},
		{
			name: "CreateWorkflow_ClientError",
			args: args{
				ctx:      context.Background(),
				k8sCli:   mockClient1,
				workflow: &tink.Workflow{},
			},
			wantErr: true,
		},
		{
			name: "CreateWorkflow_WorkflowNotFound",
			args: args{
				ctx:      context.Background(),
				k8sCli:   mockClient2,
				workflow: &tink.Workflow{},
			},
			wantErr: false,
		},
		{
			name: "CreateWorkflow_CreateError",
			args: args{
				ctx:      context.Background(),
				k8sCli:   mockClient3,
				workflow: &tink.Workflow{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateWorkflowIfNotExists(tt.args.ctx, tt.args.k8sCli, tt.args.workflow); (err != nil) != tt.wantErr {
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
		imgType      string
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
			if err := DeleteProdWorkflowResourcesIfExist(tt.args.ctx, tt.args.k8sNamespace, tt.args.hostUUID, tt.args.imgType); (err != nil) != tt.wantErr {
				t.Errorf("DeleteProdWorkflowResourcesIfExist() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteProdWorkflowResourcesIfExist_Case(t *testing.T) {
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
		imgType      string
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
			if err := DeleteProdWorkflowResourcesIfExist(tt.args.ctx, tt.args.k8sNamespace, tt.args.hostUUID, tt.args.imgType); (err != nil) != tt.wantErr {
				t.Errorf("DeleteProdWorkflowResourcesIfExist() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteRebootWorkflowResourcesIfExist(t *testing.T) {
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
			name: "TestDeleteRebootWorkflowResourcesIfExist_WithExistingResources",
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DeleteRebootWorkflowResourcesIfExist(tt.args.ctx, tt.args.k8sNamespace, tt.args.hostUUID); (err != nil) != tt.wantErr {
				t.Errorf("DeleteRebootWorkflowResourcesIfExist() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteRebootWorkflowResourcesIfExist_Case(t *testing.T) {
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
			name: "TestDeleteRebootWorkflowResourcesIfExist_WithExistingResources",
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DeleteRebootWorkflowResourcesIfExist(tt.args.ctx, tt.args.k8sNamespace, tt.args.hostUUID); (err != nil) != tt.wantErr {
				t.Errorf("DeleteRebootWorkflowResourcesIfExist() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
