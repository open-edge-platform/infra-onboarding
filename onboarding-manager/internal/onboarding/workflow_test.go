/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

//nolint:testpackage // Keeping the test in the same package due to dependencies on unexported fields.
package onboarding

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	tink "github.com/tinkerbell/tink/api/v1alpha1"
	"gotest.tools/assert"
	kubeErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	computev1 "github.com/intel/infra-core/inventory/v2/pkg/api/compute/v1"
	osv1 "github.com/intel/infra-core/inventory/v2/pkg/api/os/v1"
	statusv1 "github.com/intel/infra-core/inventory/v2/pkg/api/status/v1"
	inv_status "github.com/intel/infra-core/inventory/v2/pkg/status"
	"github.com/intel/infra-onboarding/onboarding-manager/internal/env"
	onboarding_types "github.com/intel/infra-onboarding/onboarding-manager/internal/onboarding/types"
	om_testing "github.com/intel/infra-onboarding/onboarding-manager/internal/testing"
	"github.com/intel/infra-onboarding/onboarding-manager/internal/tinkerbell"
)

func TestCheckStatusOrRunProdWorkflow(t *testing.T) {
	currK8sClientFactory := tinkerbell.K8sClientFactory
	defer func() {
		tinkerbell.K8sClientFactory = currK8sClientFactory
	}()
	tinkerbell.K8sClientFactory = om_testing.K8sCliMockFactory(false, true, false, false)

	type args struct {
		ctx        context.Context
		deviceInfo onboarding_types.DeviceInfo
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
		{
			name: "CheckStatusOrRunWorkflow",
			args: args{
				ctx: context.Background(),
				instance: &computev1.InstanceResource{
					Host: &computev1.HostResource{
						ResourceId: "host-084d9b08",
					},
					DesiredOs: &osv1.OperatingSystemResource{},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckStatusOrRunProdWorkflow(tt.args.ctx, tt.args.deviceInfo,
				tt.args.instance); (err != nil) != tt.wantErr {
				t.Errorf("CheckStatusOrRunProdWorkflow() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

//nolint:dupl // These tests cover different scenarios.
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

//nolint:dupl // These tests cover different scenarios.
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
			if err := DeleteProdWorkflowResourcesIfExist(tt.args.ctx, tt.args.hostUUID); (err != nil) != tt.wantErr {
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
			if err := handleWorkflowStatus(tt.args.instance, tt.args.workflow, tt.args.onSuccessOnboardingStatus,
				tt.args.onFailureOnboardingStatus); (err != nil) != tt.wantErr {
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
			if err := handleWorkflowStatus(tt.args.instance, tt.args.workflow, tt.args.onSuccessOnboardingStatus,
				tt.args.onFailureOnboardingStatus); (err != nil) != tt.wantErr {
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
			if err := handleWorkflowStatus(tt.args.instance, tt.args.workflow, tt.args.onSuccessOnboardingStatus,
				tt.args.onFailureOnboardingStatus); (err != nil) != tt.wantErr {
				t.Errorf("handleWorkflowStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_runProdWorkflow(t *testing.T) {
	t.Setenv("ONBOARDING_MANAGER_CLIENT_NAME", "env")
	t.Setenv("ONBOARDING_CREDENTIALS_SECRET_NAME", "env")

	type args struct {
		ctx        context.Context
		k8sCli     client.Client
		deviceInfo onboarding_types.DeviceInfo
	}
	mockClient := &om_testing.MockK8sClient{}
	mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockClient1 := &om_testing.MockK8sClient{}
	mockClient1.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("err"))
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{}
	defer func() {
		os.Unsetenv("ONBOARDING_MANAGER_CLIENT_NAME")
		os.Unsetenv("ONBOARDING_CREDENTIALS_SECRET_NAME")
	}()
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
			if err := handleWorkflowStatus(tt.args.instance, tt.args.workflow, tt.args.onSuccessOnboardingStatus,
				tt.args.onFailureOnboardingStatus); (err != nil) != tt.wantErr {
				t.Errorf("handleWorkflowStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_getWorkflow(t *testing.T) {
	mockClient := om_testing.MockK8sClient{}
	mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockClient1 := om_testing.MockK8sClient{}
	mockClient1.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("err"))
	mockClient2 := om_testing.MockK8sClient{}
	mockClient2.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(kubeErr.NewNotFound(schema.GroupResource{Group: "example.com", Resource: "myresource"}, "resource-name"))
	t.Setenv("ENABLE_ACTION_TIMESTAMPS", "true")
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
				k8sCli:       &mockClient,
			},
			want:    &tink.Workflow{},
			wantErr: false,
		},
		{
			name: "getWorkflow failure",
			args: args{
				ctx:          context.Background(),
				workflowName: "name",
				k8sCli:       &mockClient1,
			},
			want:    &tink.Workflow{},
			wantErr: true,
		},
		{
			name: "getWorkflow not found error",
			args: args{
				ctx:          context.Background(),
				workflowName: "name",
				k8sCli:       &mockClient2,
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
	currK8sClientFactory := tinkerbell.K8sClientFactory
	defer func() {
		tinkerbell.K8sClientFactory = currK8sClientFactory
	}()
	scheme := runtime.NewScheme()
	err := tink.AddToScheme(scheme)
	if err != nil {
		t.Log(err)
	}
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

	tests := []handleWorkflowTestCase{
		createTestCase("HandleSuccessfulWorkflowStatus", tink.WorkflowStateSuccess, "Provisioned", false),
		createTestCase("HandleFailedWorkflowStatus", tink.WorkflowStateFailed, "Provisioning Failed", true),
		createTestCase("HandleInProgressWorkflowStatus", tink.WorkflowStateRunning, "Provisioning In Progress", true),
	}
	for action, detail := range tinkerbell.WorkflowStepToStatusDetail {
		tests = append(tests, handleWorkflowTestCase{
			name: fmt.Sprintf("SingleAction_%s_Success", action),
			args: handleWorkflowArgs{
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
				onFailureProvisioningStatus: inv_status.New("Provisioning Failed",
					statusv1.StatusIndication_STATUS_INDICATION_ERROR),
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

type handleWorkflowArgs struct {
	instance                    *computev1.InstanceResource
	workflow                    *tink.Workflow
	onSuccessProvisioningStatus inv_status.ResourceStatus
	onFailureProvisioningStatus inv_status.ResourceStatus
}
type handleWorkflowTestCase struct {
	name                       string
	args                       handleWorkflowArgs
	expectedProvisioningStatus string
	wantErr                    bool
}

func createTestCase(name string, workflowState tink.WorkflowState, expectedStatus string, wantErr bool) handleWorkflowTestCase {
	return handleWorkflowTestCase{
		name: name,
		args: handleWorkflowArgs{
			instance: &computev1.InstanceResource{
				Host: &computev1.HostResource{
					ResourceId: "host-084d9b08",
					Uuid:       "084d9b08",
				},
				ProvisioningStatus: "Provisioning In Progress",
			},
			workflow: &tink.Workflow{
				Status: tink.WorkflowStatus{
					State: workflowState,
				},
			},
			onSuccessProvisioningStatus: inv_status.New("Provisioned", statusv1.StatusIndication_STATUS_INDICATION_IDLE),
			onFailureProvisioningStatus: inv_status.New("Provisioning Failed",
				statusv1.StatusIndication_STATUS_INDICATION_ERROR),
		},
		expectedProvisioningStatus: expectedStatus,
		wantErr:                    wantErr,
	}
}
