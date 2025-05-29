// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tinkerbell_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/mock"
	tink "github.com/tinkerbell/tink/api/v1alpha1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	om_testing "github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/testing"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/tinkerbell"
)

func TestNewWorkflow(t *testing.T) {
	type args struct {
		name        string
		ns          string
		hardwareRef string
		templateRef string
		hardwareMap map[string]string
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
				hardwareMap: map[string]string{
					"device_1": "00:11:22:33:44:55",
				},
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
			if got := tinkerbell.NewWorkflow(tt.args.name, tt.args.ns, tt.args.hardwareRef,
				tt.args.templateRef, tt.args.hardwareMap); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewWorkflow() = %v, want %v", got, tt.want)
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
		Return(k8sErrors.NewNotFound(schema.GroupResource{Group: "example.com", Resource: "myresource"}, "resource-name"))
	mockClient2.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockClient3 := om_testing.MockK8sClient{}
	mockClient3.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(k8sErrors.NewNotFound(schema.GroupResource{Group: "example.com", Resource: "myresource"}, "resource-name"))
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
			if err := tinkerbell.CreateWorkflowIfNotExists(tt.args.ctx, tt.args.k8sCli,
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
			if err := tinkerbell.DeleteWorkflowIfExists(tt.args.ctx, tt.args.k8sNamespace,
				tt.args.hostUUID); (err != nil) != tt.wantErr {
				t.Errorf("DeleteWorkflowIfExists() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteProdWorkflowResourcesIfExist_Case(t *testing.T) {
	currK8sClientFactory := tinkerbell.K8sClientFactory
	defer func() {
		tinkerbell.K8sClientFactory = currK8sClientFactory
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
