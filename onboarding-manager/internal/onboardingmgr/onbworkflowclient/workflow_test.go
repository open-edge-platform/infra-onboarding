/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onbworkflowclient

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/api/compute/v1"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/api/os/v1"
	statusv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/api/status/v1"
	inv_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/status"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/onboarding-manager/internal/env"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/onboarding-manager/internal/onboardingmgr/utils"
	om_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/onboarding-manager/internal/testing"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/onboarding-manager/internal/tinkerbell"
	"github.com/stretchr/testify/mock"
	tink "github.com/tinkerbell/tink/api/v1alpha1"
	"gotest.tools/assert"
	kubeErr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCheckStatusOrRunProdWorkflow(t *testing.T) {
	type args struct {
		ctx        context.Context
		deviceInfo utils.DeviceInfo
		instance   *computev1.InstanceResource
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Empty Context and Instance",
			args: args{
				ctx:      context.Background(),
				instance: &computev1.InstanceResource{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckStatusOrRunProdWorkflow(tt.args.ctx, tt.args.deviceInfo, tt.args.instance); (err != nil) != tt.wantErr {
				t.Errorf("CheckStatusOrRunProdWorkflow() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteTinkHardwareForHostIfExist(t *testing.T) {
	type args struct {
		ctx      context.Context
		hostUUID string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "DeleteTinkHardwareForHostIfExistsTest",
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DeleteTinkHardwareForHostIfExist(tt.args.ctx, tt.args.hostUUID); (err != nil) != tt.wantErr {
				t.Errorf("DeleteTinkHardwareForHostIfExist() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteProdWorkflowResourcesIfExist(t *testing.T) {
	type args struct {
		ctx      context.Context
		hostUUID string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "DeleteProdWorkflowResourcesIfExistsTest",
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DeleteProdWorkflowResourcesIfExist(tt.args.ctx, tt.args.hostUUID, "bkc"); (err != nil) != tt.wantErr {
				t.Errorf("DeleteProdWorkflowResourcesIfExist() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_handleWorkflowStatus(t *testing.T) {
	type args struct {
		instance                  *computev1.InstanceResource
		workflow                  *tink.Workflow
		onSuccessOnboardingStatus inv_status.ResourceStatus
		onFailureOnboardingStatus inv_status.ResourceStatus
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "HandleEmptyWorkflow",
			args: args{
				instance: &computev1.InstanceResource{
					Host: &computev1.HostResource{
						ResourceId: "host-084d9b08",
					},
				},
				workflow: &tink.Workflow{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := handleWorkflowStatus(tt.args.instance, tt.args.workflow, tt.args.onSuccessOnboardingStatus, tt.args.onFailureOnboardingStatus); (err != nil) != tt.wantErr {
				t.Errorf("handleWorkflowStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_handleWorkflowStatus_Case(t *testing.T) {
	type args struct {
		instance                  *computev1.InstanceResource
		workflow                  *tink.Workflow
		onSuccessOnboardingStatus inv_status.ResourceStatus
		onFailureOnboardingStatus inv_status.ResourceStatus
	}
	currK8sClientFactory := tinkerbell.K8sClientFactory
	defer func() {
		tinkerbell.K8sClientFactory = currK8sClientFactory
	}()
	tinkerbell.K8sClientFactory = om_testing.K8sCliMockFactory(false, true, false, false)
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "HandleSuccessfulWorkflowStatus",
			args: args{
				instance: &computev1.InstanceResource{
					Host: &computev1.HostResource{
						ResourceId: "host-084d9b08",
						Uuid:       uuid.NewString(),
					},
				},
				workflow: &tink.Workflow{
					Status: tink.WorkflowStatus{
						State: tink.WorkflowStateSuccess,
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := handleWorkflowStatus(tt.args.instance, tt.args.workflow, tt.args.onSuccessOnboardingStatus, tt.args.onFailureOnboardingStatus); (err != nil) != tt.wantErr {
				t.Errorf("handleWorkflowStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_handleWorkflowStatus_Case1(t *testing.T) {
	type args struct {
		instance                  *computev1.InstanceResource
		workflow                  *tink.Workflow
		onSuccessOnboardingStatus inv_status.ResourceStatus
		onFailureOnboardingStatus inv_status.ResourceStatus
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "HandleFailedWorkflowStatus",
			args: args{
				instance: &computev1.InstanceResource{
					Host: &computev1.HostResource{
						ResourceId: "host-084d9b08",
					},
				},
				workflow: &tink.Workflow{
					Status: tink.WorkflowStatus{
						State: tink.WorkflowStateFailed,
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := handleWorkflowStatus(tt.args.instance, tt.args.workflow, tt.args.onSuccessOnboardingStatus, tt.args.onFailureOnboardingStatus); (err != nil) != tt.wantErr {
				t.Errorf("handleWorkflowStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_handleWorkflowStatus_Case2(t *testing.T) {
	type args struct {
		instance                  *computev1.InstanceResource
		workflow                  *tink.Workflow
		onSuccessOnboardingStatus inv_status.ResourceStatus
		onFailureOnboardingStatus inv_status.ResourceStatus
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "HandleRunningWorkflowStatus",
			args: args{
				instance: &computev1.InstanceResource{
					Host: &computev1.HostResource{
						ResourceId: "host-084d9b08",
					},
				},
				workflow: &tink.Workflow{
					Status: tink.WorkflowStatus{
						State: tink.WorkflowStateRunning,
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := handleWorkflowStatus(tt.args.instance, tt.args.workflow, tt.args.onSuccessOnboardingStatus, tt.args.onFailureOnboardingStatus); (err != nil) != tt.wantErr {
				t.Errorf("handleWorkflowStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type MockClient struct {
	mock.Mock
}

// type MockCreateOption struct {
// 	ApplyFunc func(*client.CreateOptions)
// }

//	func (m *MockCreateOption) ApplyToCreate(opts *client.CreateOptions) {
//		if m.ApplyFunc != nil {
//			m.ApplyFunc(opts)
//		}
//	}
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

func Test_runProdWorkflow(t *testing.T) {
	os.Setenv("ONBOARDING_MANAGER_CLIENT_NAME", "env")
	os.Setenv("ONBOARDING_CREDENTIALS_SECRET_NAME", "env")

	type args struct {
		ctx        context.Context
		k8sCli     client.Client
		deviceInfo utils.DeviceInfo
	}
	mockClient := &MockClient{}
	mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockClient1 := &MockClient{}
	mockClient1.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("err"))
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := runProdWorkflow(tt.args.ctx, tt.args.k8sCli, tt.args.deviceInfo, &computev1.InstanceResource{
				Host:      &computev1.HostResource{},
				DesiredOs: &osv1.OperatingSystemResource{},
			}); (err != nil) != tt.wantErr {
				t.Errorf("runProdWorkflow() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	defer func() {
		os.Unsetenv("ONBOARDING_MANAGER_CLIENT_NAME")
		os.Unsetenv("ONBOARDING_CREDENTIALS_SECRET_NAME")
	}()
}

func Test_handleWorkflowStatus_Case3(t *testing.T) {
	type args struct {
		instance                  *computev1.InstanceResource
		workflow                  *tink.Workflow
		onSuccessOnboardingStatus inv_status.ResourceStatus
		onFailureOnboardingStatus inv_status.ResourceStatus
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				instance: &computev1.InstanceResource{
					Host: &computev1.HostResource{
						ResourceId: "host-084d9b08",
					},
				},
				workflow: &tink.Workflow{
					Status: tink.WorkflowStatus{
						State: "default",
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := handleWorkflowStatus(tt.args.instance, tt.args.workflow, tt.args.onSuccessOnboardingStatus, tt.args.onFailureOnboardingStatus); (err != nil) != tt.wantErr {
				t.Errorf("handleWorkflowStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_formatDuration(t *testing.T) {
	type args struct {
		d time.Duration
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Success",
			args: args{},
			want: "00",
		},
	}
	utils.Init("")
	utils.TimeStamp("")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatDuration(tt.args.d); got != tt.want {
				t.Errorf("formatDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getWorkflow(t *testing.T) {
	mockClient := MockClient{}
	mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockClient1 := MockClient{}
	mockClient1.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("err"))
	mockClient2 := MockClient{}
	mockClient2.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(kubeErr.NewNotFound(schema.GroupResource{Group: "example.com", Resource: "myresource"}, "resource-name"))
	os.Setenv("ENABLE_ACTION_TIMESTAMPS", "true")
	defer os.Unsetenv("ENABLE_ACTION_TIMESTAMPS")
	type args struct {
		ctx          context.Context
		k8sCli       client.Client
		workflowName string
	}
	tests := []struct {
		name    string
		args    args
		want    *tink.Workflow
		wantErr bool
	}{
		{
			name: "getWorkflow success",
			args: args{
				ctx:          context.Background(),
				workflowName: "name",
				k8sCli:       mockClient,
			},
			want:    &tink.Workflow{},
			wantErr: false,
		},
		{
			name: "getWorkflow failure",
			args: args{
				ctx:          context.Background(),
				workflowName: "name",
				k8sCli:       mockClient1,
			},
			want:    &tink.Workflow{},
			wantErr: true,
		},
		{
			name: "getWorkflow not found error",
			args: args{
				ctx:          context.Background(),
				workflowName: "name",
				k8sCli:       mockClient2,
			},
			want:    &tink.Workflow{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getWorkflow(tt.args.ctx, tt.args.k8sCli, tt.args.workflowName)
			if (err != nil) != tt.wantErr {
				t.Errorf("getWorkflow() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_handleWorkflowStatus_Case4(t *testing.T) {
	type args struct {
		instance                    *computev1.InstanceResource
		workflow                    *tink.Workflow
		onSuccessProvisioningStatus inv_status.ResourceStatus
		onFailureProvisioningStatus inv_status.ResourceStatus
	}
	currK8sClientFactory := tinkerbell.K8sClientFactory
	defer func() {
		tinkerbell.K8sClientFactory = currK8sClientFactory
	}()
	scheme := runtime.NewScheme()
	_ = tink.AddToScheme(scheme)

	tinkerbell.K8sClientFactory = func() (client.Client, error) {
		return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&tink.Hardware{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "machine-084d9b08",
					Namespace: env.K8sNamespace,
				},
				Spec: tink.HardwareSpec{
					Metadata: &tink.HardwareMetadata{
						Instance: &tink.MetadataInstance{
							OperatingSystem: &tink.MetadataInstanceOperatingSystem{
								OsSlug: "os-12345678",
							},
						},
					},
				},
			},
		).Build(), nil
	}

	tests := []struct {
		name                       string
		args                       args
		expectedProvisioningStatus string
		wantErr                    bool
	}{
		{
			name: "HandleSuccessfulWorkflowStatus",
			args: args{
				instance: &computev1.InstanceResource{
					Host: &computev1.HostResource{
						ResourceId: "host-084d9b08",
						Uuid:       "084d9b08",
					},
					ProvisioningStatus: "Provisioning In Progress",
				},
				workflow: &tink.Workflow{
					Status: tink.WorkflowStatus{
						State: tink.WorkflowStateSuccess,
					},
				},
				onSuccessProvisioningStatus: inv_status.New("Provisioned", statusv1.StatusIndication_STATUS_INDICATION_IDLE),
				onFailureProvisioningStatus: inv_status.New("Provisioning Failed", statusv1.StatusIndication_STATUS_INDICATION_ERROR),
			},
			expectedProvisioningStatus: "Provisioned",
			wantErr:                    false,
		},
		{
			name: "HandleFailedWorkflowStatus",
			args: args{
				instance: &computev1.InstanceResource{
					Host: &computev1.HostResource{
						ResourceId: "host-084d9b08",
						Uuid:       "084d9b08",
					},
					ProvisioningStatus: "Provisioning In Progress",
				},
				workflow: &tink.Workflow{
					Status: tink.WorkflowStatus{
						State: tink.WorkflowStateFailed,
					},
				},
				onSuccessProvisioningStatus: inv_status.New("Provisioned", statusv1.StatusIndication_STATUS_INDICATION_IDLE),
				onFailureProvisioningStatus: inv_status.New("Provisioning Failed", statusv1.StatusIndication_STATUS_INDICATION_ERROR),
			},
			expectedProvisioningStatus: "Provisioning Failed",
			wantErr:                    true,
		},
		{
			name: "HandleInProgressWorkflowStatus",
			args: args{
				instance: &computev1.InstanceResource{
					Host: &computev1.HostResource{
						ResourceId: "host-084d9b08",
						Uuid:       "084d9b08",
					},
					ProvisioningStatus: "Provisioning In Progress",
				},
				workflow: &tink.Workflow{
					Status: tink.WorkflowStatus{
						State: tink.WorkflowStateRunning,
					},
				},
				onSuccessProvisioningStatus: inv_status.New("Provisioned", statusv1.StatusIndication_STATUS_INDICATION_IDLE),
				onFailureProvisioningStatus: inv_status.New("Provisioning Failed", statusv1.StatusIndication_STATUS_INDICATION_ERROR),
			},
			expectedProvisioningStatus: "Provisioning In Progress",
			wantErr:                    true,
		},
	}
	for action, detail := range tinkerbell.WorkflowStepToStatusDetail {
		tests = append(tests, struct {
			name                       string
			args                       args
			expectedProvisioningStatus string
			wantErr                    bool
		}{
			name: fmt.Sprintf("SingleAction_%s_Success", action),
			args: args{
				instance: &computev1.InstanceResource{
					Host: &computev1.HostResource{
						ResourceId: "host-084d9b08",
						Uuid:       uuid.NewString(),
					},
					ProvisioningStatus: "Provisioning In Progress",
				},
				workflow: &tink.Workflow{
					Status: tink.WorkflowStatus{
						Tasks: []tink.Task{
							{
								Actions: []tink.Action{
									{
										Name:   action,
										Status: tink.WorkflowStateSuccess,
									},
								},
							},
						},
					},
				},
				onSuccessProvisioningStatus: inv_status.New("Provisioned", statusv1.StatusIndication_STATUS_INDICATION_IDLE),
				onFailureProvisioningStatus: inv_status.New("Provisioning Failed", statusv1.StatusIndication_STATUS_INDICATION_ERROR),
			},
			expectedProvisioningStatus: "Provisioning In Progress: 1/1: " + detail,
			wantErr:                    true,
		})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handleWorkflowStatus(tt.args.instance, tt.args.workflow, tt.args.onSuccessProvisioningStatus,
				tt.args.onFailureProvisioningStatus)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleWorkflowStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.expectedProvisioningStatus, tt.args.instance.ProvisioningStatus)
		})
	}
}
