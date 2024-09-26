/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onbworkflowclient

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/common"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/env"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
	om_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/testing"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/tinkerbell"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/compute/v1"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/os/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/flags"
	inv_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/status"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	tink "github.com/tinkerbell/tink/api/v1alpha1"
	kubeErr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Test_checkTO2StatusCompleted(t *testing.T) {
	type args struct {
		in0        context.Context
		deviceInfo utils.DeviceInfo
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "CheckTO2StatusCompleted_WhenContextIsBackground",
			args: args{
				in0: context.Background(),
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkTO2StatusCompleted(tt.args.in0, tt.args.deviceInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkTO2StatusCompleted() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkTO2StatusCompleted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_checkTO2StatusCompleted_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/owner/state/id" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"to2CompletedOn": "completed",
				"to0Expiry": ""
				}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not found"))
		}
	}))
	defer server.Close()

	type args struct {
		in0        context.Context
		deviceInfo utils.DeviceInfo
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "Success",
			args: args{
				in0: context.Background(),
				deviceInfo: utils.DeviceInfo{
					FdoGUID: "id",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Failed",
			args: args{
				in0:        context.Background(),
				deviceInfo: utils.DeviceInfo{
					// empty fdoGUID to return error
				},
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env.FdoOwnerDNS = "127.0.0.1"
			env.FdoOwnerPort = strings.Split(server.URL, ":")[2]
			got, err := checkTO2StatusCompleted(tt.args.in0, tt.args.deviceInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkTO2StatusCompleted() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkTO2StatusCompleted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_checkTO2StatusCompleted_Case1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/owner/state/id" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"to2CompletedOn": "",
				"to0Expiry": ""
				}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not found"))
		}
	}))
	defer server.Close()

	type args struct {
		in0        context.Context
		deviceInfo utils.DeviceInfo
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "TO2StatusCompleted_WhenValidResponseReceived",
			args: args{
				in0: context.Background(),
				deviceInfo: utils.DeviceInfo{
					FdoGUID: "id",
				},
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env.FdoOwnerDNS = "127.0.0.1"
			env.FdoOwnerPort = strings.Split(server.URL, ":")[2]
			got, err := checkTO2StatusCompleted(tt.args.in0, tt.args.deviceInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkTO2StatusCompleted() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkTO2StatusCompleted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_checkTO2StatusCompleted_Case2(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:58042")
	if err != nil {
		t.Fatalf("Error creating listener: %v", err)
	}
	defer listener.Close()
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/owner/state/id" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(""))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not found"))
		}
	}))
	server.Listener = listener
	server.Start()
	defer server.Close()
	type args struct {
		in0        context.Context
		deviceInfo utils.DeviceInfo
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "TO2StatusCompleted_WhenLocalServerResponds",
			args: args{
				in0: context.Background(),
				deviceInfo: utils.DeviceInfo{
					FdoGUID: "id",
				},
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env.FdoOwnerDNS = "localhost"
			env.FdoOwnerPort = "58042"
			got, err := checkTO2StatusCompleted(tt.args.in0, tt.args.deviceInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkTO2StatusCompleted() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkTO2StatusCompleted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_checkTO2StatusCompleted_Case3(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:58042")
	if err != nil {
		t.Fatalf("Error creating listener: %v", err)
	}
	defer listener.Close()
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/owner/state/id" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"to2CompletedOn": "abc",
				"to0Expiry": ""
				}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not found"))
		}
	}))
	server.Listener = listener
	server.Start()
	defer server.Close()
	type args struct {
		in0        context.Context
		deviceInfo utils.DeviceInfo
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "TO2StatusCompleted_WhenCompletedOnValuePresent",
			args: args{
				in0: context.Background(),
				deviceInfo: utils.DeviceInfo{
					FdoGUID: "id",
				},
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env.FdoOwnerDNS = "localhost"
			env.FdoOwnerPort = "58042"
			got, err := checkTO2StatusCompleted(tt.args.in0, tt.args.deviceInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkTO2StatusCompleted() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkTO2StatusCompleted() = %v, want %v", got, tt.want)
			}
		})
	}
}

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

func TestCheckStatusOrRunDIWorkflow(t *testing.T) {
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
			name: "Valid Instance",
			args: args{
				ctx:        context.Background(),
				deviceInfo: utils.DeviceInfo{},
				instance: &computev1.InstanceResource{
					Host: &computev1.HostResource{
						ResourceId: "host-084d9b08",
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckStatusOrRunDIWorkflow(tt.args.ctx, tt.args.deviceInfo, tt.args.instance); (err != nil) != tt.wantErr {
				t.Errorf("CheckStatusOrRunDIWorkflow() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckStatusOrRunDIWorkflow_Case1(t *testing.T) {
	currK8sClientFactory := tinkerbell.K8sClientFactory
	currFlagEnableDeviceInitialization := *flags.FlagDisableCredentialsManagement
	defer func() {
		tinkerbell.K8sClientFactory = currK8sClientFactory
		*common.FlagEnableDeviceInitialization = currFlagEnableDeviceInitialization
	}()
	*common.FlagEnableDeviceInitialization = true
	tinkerbell.K8sClientFactory = om_testing.K8sCliMockFactory(false, false, false)
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
			name: "Valid K8sClientFactory",
			args: args{
				ctx:        context.Background(),
				deviceInfo: utils.DeviceInfo{},
				instance: &computev1.InstanceResource{
					Host: &computev1.HostResource{
						ResourceId: "host-084d9b08",
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckStatusOrRunDIWorkflow(tt.args.ctx, tt.args.deviceInfo, tt.args.instance); (err != nil) != tt.wantErr {
				t.Errorf("CheckStatusOrRunDIWorkflow() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckStatusOrRunDIWorkflow_Case(t *testing.T) {
	*common.FlagEnableDeviceInitialization = false
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
			name: "Valid Instance",
			args: args{
				ctx:        context.Background(),
				deviceInfo: utils.DeviceInfo{},
				instance: &computev1.InstanceResource{
					Host: &computev1.HostResource{
						ResourceId: "host-084d9b08",
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckStatusOrRunDIWorkflow(tt.args.ctx, tt.args.deviceInfo, tt.args.instance); (err != nil) != tt.wantErr {
				t.Errorf("CheckStatusOrRunDIWorkflow() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	defer func() {
		*common.FlagEnableDeviceInitialization = true
	}()
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

func TestDeleteDIWorkflowResourcesIfExist(t *testing.T) {
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
			name: "DeleteDIWorkflowResourcesIfExistsTest",
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DeleteDIWorkflowResourcesIfExist(tt.args.ctx, tt.args.hostUUID); (err != nil) != tt.wantErr {
				t.Errorf("DeleteDIWorkflowResourcesIfExist() error = %v, wantErr %v", err, tt.wantErr)
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
	currFlagEnableDeviceInitialization := *flags.FlagDisableCredentialsManagement
	defer func() {
		tinkerbell.K8sClientFactory = currK8sClientFactory
		*common.FlagEnableDeviceInitialization = currFlagEnableDeviceInitialization
	}()
	*common.FlagEnableDeviceInitialization = false
	tinkerbell.K8sClientFactory = om_testing.K8sCliMockFactory(false, true, false)
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
	defer func() {
		*common.FlagEnableDeviceInitialization = true
	}()
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
	resp := &ResponseData{
		To2CompletedOn: "completed",
		To0Expiry:      "",
	}
	jsonData, err := json.Marshal(resp)
	require.NoError(t, err)
	os.Setenv("ONBOARDING_MANAGER_CLIENT_NAME", "env")
	os.Setenv("ONBOARDING_CREDENTIALS_SECRET_NAME", "env")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Write(jsonData)
	}))
	defer srv.Close()

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
	}{
		// {
		// 	name: "TestRunProdWorkflow",
		// 	args: args{
		// 		ctx:    context.Background(),
		// 		k8sCli: mockClient,
		// 		deviceInfo: utils.DeviceInfo{
		// 			GUID:   "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
		// 			FdoGUID: uuid.NewString(),
		// 		},
		// 	},
		// 	wantErr: false,
		// },
		// {
		// 	name: "TestRunProdWorkflow_ClientGetError",
		// 	args: args{
		// 		k8sCli: mockClient1,
		// 	},
		// 	wantErr: true,
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env.FdoOwnerDNS = "127.0.0.1"
			env.FdoOwnerPort = strings.Split(srv.URL, ":")[2]
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

func Test_runDIWorkflow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	type args struct {
		ctx        context.Context
		k8sCli     client.Client
		deviceInfo utils.DeviceInfo
		instance   *computev1.InstanceResource
	}
	mockClient := &MockClient{}
	mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockClient1 := &MockClient{}
	mockClient1.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("err"))
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "TestRunDIWorkflow_SuccessfulRequest",
			args: args{
				ctx:        context.Background(),
				k8sCli:     mockClient,
				deviceInfo: utils.DeviceInfo{},
				instance: &computev1.InstanceResource{
					DesiredOs: &osv1.OperatingSystemResource{},
				},
			},
		},
		{
			name: "TestRunDIWorkflow_ClientGetError",
			args: args{
				ctx:    context.Background(),
				k8sCli: mockClient1,
				instance: &computev1.InstanceResource{
					DesiredOs: &osv1.OperatingSystemResource{},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env.FdoOwnerDNS = srv.URL
			if err := runDIWorkflow(tt.args.ctx, tt.args.k8sCli, tt.args.deviceInfo, tt.args.instance); (err != nil) != tt.wantErr {
				t.Errorf("runDIWorkflow() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunFDOActions(t *testing.T) {
	*common.FlagEnableDeviceInitialization = true
	type args struct {
		ctx        context.Context
		deviceInfo *utils.DeviceInfo
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "TestRunFDOActions_DeviceInfoEmpty",
			args: args{
				ctx:        context.Background(),
				deviceInfo: &utils.DeviceInfo{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env.FdoOwnerDNS = "localhost"
			env.FdoOwnerPort = "58042"
			if err := RunFDOActions(tt.args.ctx, tt.args.deviceInfo); (err != nil) != tt.wantErr {
				t.Errorf("RunFDOActions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	defer func() {
		*common.FlagEnableDeviceInitialization = true
	}()

}

func TestCheckStatusOrRunRebootWorkflow(t *testing.T) {
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
			name: "TestCheckStatusOrRunRebootWorkflow_HostNotReady",
			args: args{
				ctx:      context.Background(),
				instance: &computev1.InstanceResource{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckStatusOrRunRebootWorkflow(tt.args.ctx, tt.args.deviceInfo, tt.args.instance); (err != nil) != tt.wantErr {
				t.Errorf("CheckStatusOrRunRebootWorkflow() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_runRebootWorkflow(t *testing.T) {
	mockClient := MockClient{}
	mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockClient1 := MockClient{}
	mockClient1.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("err"))
	type args struct {
		ctx        context.Context
		k8sCli     client.Client
		deviceInfo utils.DeviceInfo
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "SuccessfulRebootWorkflow",
			args: args{
				ctx:    context.Background(),
				k8sCli: mockClient,
			},
		},
		{
			name: "FailedRebootWorkflow",
			args: args{
				ctx:    context.Background(),
				k8sCli: mockClient1,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := runRebootWorkflow(tt.args.ctx, tt.args.k8sCli, tt.args.deviceInfo); (err != nil) != tt.wantErr {
				t.Errorf("runRebootWorkflow() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteRebootWorkflowResourcesIfExist(t *testing.T) {
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
			name: "DeleteRebootWorkflowResources",
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DeleteRebootWorkflowResourcesIfExist(tt.args.ctx, tt.args.hostUUID); (err != nil) != tt.wantErr {
				t.Errorf("DeleteRebootWorkflowResourcesIfExist() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_checkTO2StatusCompleted_Case4(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/owner/state/id" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"to2CompletedOn": "completed",
				"to0Expiry": 123
				}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not found"))
		}
	}))
	defer server.Close()

	type args struct {
		in0        context.Context
		deviceInfo utils.DeviceInfo
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "Success",
			args: args{
				in0: context.Background(),
				deviceInfo: utils.DeviceInfo{
					FdoGUID: "id",
				},
			},
			want:    true,
			wantErr: true,
		},
		{
			name: "Failed",
			args: args{
				in0:        context.Background(),
				deviceInfo: utils.DeviceInfo{
					// empty fdoGUID to return error
				},
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env.FdoOwnerDNS = "127.0.0.1"
			env.FdoOwnerPort = strings.Split(server.URL, ":")[2]
			_, err := checkTO2StatusCompleted(tt.args.in0, tt.args.deviceInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkTO2StatusCompleted() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_runProdWorkflow_Case(t *testing.T) {
	common.FlagEnableDeviceInitialization = flag.Bool("enable", false,
		"Enables ")
	resp := &ResponseData{
		To2CompletedOn: "completed",
		To0Expiry:      "",
	}
	jsonData, err := json.Marshal(resp)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Write(jsonData)
	}))
	defer srv.Close()

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
	}{
		{
			name: "Test Case",
			args: args{
				ctx:    context.Background(),
				k8sCli: mockClient,
				deviceInfo: utils.DeviceInfo{
					GUID:    uuid.NewString(),
					FdoGUID: uuid.NewString(),
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env.FdoOwnerDNS = "127.0.0.1"
			env.FdoOwnerPort = strings.Split(srv.URL, ":")[2]
			if err := runProdWorkflow(tt.args.ctx, tt.args.k8sCli, tt.args.deviceInfo, &computev1.InstanceResource{
				Host: &computev1.HostResource{},
			}); (err != nil) != tt.wantErr {
				t.Errorf("runProdWorkflow() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	defer func() {
		common.FlagEnableDeviceInitialization = flag.Bool("enab", true,
			"Enables")
	}()
}

func TestRunFDOActions_Case1(t *testing.T) {
	common.FlagEnableDeviceInitialization = flag.Bool("enle", false,
		"Enabl")
	type args struct {
		ctx        context.Context
		deviceInfo *utils.DeviceInfo
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				ctx:        context.Background(),
				deviceInfo: &utils.DeviceInfo{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env.FdoOwnerDNS = "localhost"
			env.FdoOwnerPort = "58042"
			if err := RunFDOActions(tt.args.ctx, tt.args.deviceInfo); (err != nil) != tt.wantErr {
				t.Errorf("RunFDOActions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	defer func() {
		common.FlagEnableDeviceInitialization = flag.Bool("en", true,
			"Ena")
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
			want: "00:00:00",
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
