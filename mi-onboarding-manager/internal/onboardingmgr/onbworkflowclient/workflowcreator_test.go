/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onbworkflowclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
)

func generatekubeconfigPath() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}

	kubeconfigPath := filepath.Join(currentUser.HomeDir, ".kube", "config")
	return kubeconfigPath, err
}

func TestCaSlculateRootF(t *testing.T) {
	// Test case 1: imageType is "bkc" and diskDev ends with a numeric digit
	partition := CalculateRootFS("bkc", "sda1")
	assert.Equal(t, "p1", partition, "Expected partition 'p1'")

	// Test case 2: imageType is "ms" and diskDev ends with a numeric digit
	partition = CalculateRootFS("ms", "nvme0n1p2")
	assert.Equal(t, "p1", partition, "Expected partition 'p1'")

	// Test case 3: imageType is "bkc" and diskDev does not end with a numeric digit
	partition = CalculateRootFS("bkc", "sdb")
	assert.Equal(t, "1", partition, "Expected partition '1'")

	// Test case 4: imageType is  "ms" and diskDev ends with a numeric digit
	partition = CalculateRootFS("other", "nvme0n1p3")
	assert.Equal(t, "p1", partition, "Expected partition 'p1'")
}

// MockHTTPServer creates a mock HTTP server and returns its URL.
func MockHTTPServer(handler http.Handler) (*httptest.Server, string) {
	server := httptest.NewServer(handler)
	return server, server.URL
}

func TestDiWorkflowCreation(t *testing.T) {
	k, _ := generatekubeconfigPath()
	type args struct {
		deviceInfo     utils.DeviceInfo
		kubeconfigpath string
	}
	inputargs := args{
		deviceInfo:     utils.DeviceInfo{},
		kubeconfigpath: k,
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"test",
			inputargs,
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DiWorkflowCreation(tt.args.deviceInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("DiWorkflowCreation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DiWorkflowCreation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVoucherScript(t *testing.T) {
	type args struct {
		hostIP       string
		deviceSerial string
	}
	inputargs := args{
		hostIP:       "",
		deviceSerial: "123",
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"test",
			inputargs,
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := VoucherScript(tt.args.hostIP, tt.args.deviceSerial)
			if (err != nil) != tt.wantErr {
				t.Errorf("VoucherScript() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("VoucherScript() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProdWorkflowCreation(t *testing.T) {
	type args struct {
		deviceInfo   utils.DeviceInfo
		imgtype      string
		artifactinfo utils.ArtifactData
	}
	inputargs := args{
		deviceInfo:   utils.DeviceInfo{},
		imgtype:      "",
		artifactinfo: utils.ArtifactData{},
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"test",
			inputargs,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ProdWorkflowCreation(tt.args.deviceInfo, tt.args.imgtype,
				tt.args.artifactinfo); (err != nil) != tt.wantErr {
				t.Errorf("ProdWorkflowCreation() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

var runtimeScheme = runtime.NewScheme()

func Initialize() {
	_ = v1.AddToScheme(runtimeScheme)
}

type MockInterface struct {
	mock.Mock
}

// Resource is a mock implementation for the Resource method.
func (m *MockInterface) Resource(resource schema.GroupVersionResource) dynamic.NamespaceableResourceInterface {
	args := m.Called(resource)
	return args.Get(0).(dynamic.NamespaceableResourceInterface)
}

type MockNamespaceableResource struct {
	mock.Mock
}
type ResourceInterfaceMock struct {
	mock.Mock
}

// Namespace mocks the Namespace method of NamespaceableResourceInterface.
func (m *ResourceInterfaceMock) Namespace(ns string) dynamic.ResourceInterface {
	args := m.Called(ns)
	return args.Get(0).(dynamic.ResourceInterface)
}

func (m *ResourceInterfaceMock) Create(ctx context.Context, obj *unstructured.Unstructured,
	options metav1.CreateOptions, subresources ...string,
) (*unstructured.Unstructured, error) {
	args := m.Called(ctx, obj, options, subresources)
	return args.Get(0).(*unstructured.Unstructured), args.Error(1)
}

func (m *ResourceInterfaceMock) Update(ctx context.Context, obj *unstructured.Unstructured,
	options metav1.UpdateOptions, subresources ...string,
) (*unstructured.Unstructured, error) {
	args := m.Called(ctx, obj, options, subresources)
	return args.Get(0).(*unstructured.Unstructured), args.Error(1)
}

func (m *ResourceInterfaceMock) UpdateStatus(ctx context.Context, obj *unstructured.Unstructured,
	options metav1.UpdateOptions,
) (*unstructured.Unstructured, error) {
	args := m.Called(ctx, obj, options)
	return args.Get(0).(*unstructured.Unstructured), args.Error(1)
}

func (m *ResourceInterfaceMock) Delete(ctx context.Context, name string,
	options metav1.DeleteOptions, subresources ...string,
) error {
	args := m.Called(ctx, name, options, subresources)
	return args.Error(0)
}

func (m *ResourceInterfaceMock) DeleteCollection(ctx context.Context, options metav1.DeleteOptions,
	listOptions metav1.ListOptions,
) error {
	args := m.Called(ctx, options, listOptions)
	return args.Error(0)
}

func (m *ResourceInterfaceMock) Get(ctx context.Context, name string, options metav1.GetOptions,
	subresources ...string,
) (*unstructured.Unstructured, error) {
	args := m.Called(ctx, name, options, subresources)
	return args.Get(0).(*unstructured.Unstructured), args.Error(1)
}

func (m *ResourceInterfaceMock) List(ctx context.Context, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(*unstructured.UnstructuredList), args.Error(1)
}

func (m *ResourceInterfaceMock) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(watch.Interface), args.Error(1)
}

func (m *ResourceInterfaceMock) Patch(ctx context.Context, name string, pt types.PatchType,
	data []byte, options metav1.PatchOptions, subresources ...string,
) (*unstructured.Unstructured, error) {
	args := m.Called(ctx, name, pt, data, options, subresources)
	return args.Get(0).(*unstructured.Unstructured), args.Error(1)
}

func (m *ResourceInterfaceMock) Apply(ctx context.Context, name string, obj *unstructured.Unstructured,
	options metav1.ApplyOptions, subresources ...string,
) (*unstructured.Unstructured, error) {
	args := m.Called(ctx, name, obj, options, subresources)
	return args.Get(0).(*unstructured.Unstructured), args.Error(1)
}

func (m *ResourceInterfaceMock) ApplyStatus(ctx context.Context, name string, obj *unstructured.Unstructured,
	options metav1.ApplyOptions,
) (*unstructured.Unstructured, error) {
	args := m.Called(ctx, name, obj, options)
	return args.Get(0).(*unstructured.Unstructured), args.Error(1)
}

// Namespace mocks the Namespace method of NamespaceableResourceInterface.
func (m *MockNamespaceableResource) Namespace(ns string) dynamic.ResourceInterface {
	args := m.Called(ns)
	return args.Get(0).(dynamic.ResourceInterface)
}

func (m *MockNamespaceableResource) Create(ctx context.Context, obj *unstructured.Unstructured,
	options metav1.CreateOptions, subresources ...string,
) (*unstructured.Unstructured, error) {
	args := m.Called(ctx, obj, options, subresources)
	return args.Get(0).(*unstructured.Unstructured), args.Error(1)
}

func (m *MockNamespaceableResource) Update(ctx context.Context, obj *unstructured.Unstructured,
	options metav1.UpdateOptions, subresources ...string,
) (*unstructured.Unstructured, error) {
	args := m.Called(ctx, obj, options, subresources)
	return args.Get(0).(*unstructured.Unstructured), args.Error(1)
}

func (m *MockNamespaceableResource) UpdateStatus(ctx context.Context, obj *unstructured.Unstructured,
	options metav1.UpdateOptions,
) (*unstructured.Unstructured, error) {
	args := m.Called(ctx, obj, options)
	return args.Get(0).(*unstructured.Unstructured), args.Error(1)
}

func (m *MockNamespaceableResource) Delete(ctx context.Context, name string, options metav1.DeleteOptions,
	subresources ...string,
) error {
	args := m.Called(ctx, name, options, subresources)
	return args.Error(0)
}

func (m *MockNamespaceableResource) DeleteCollection(ctx context.Context, options metav1.DeleteOptions,
	listOptions metav1.ListOptions,
) error {
	args := m.Called(ctx, options, listOptions)
	return args.Error(0)
}

func (m *MockNamespaceableResource) Get(ctx context.Context, name string, options metav1.GetOptions,
	subresources ...string,
) (*unstructured.Unstructured, error) {
	args := m.Called(ctx, name, options, subresources)
	return args.Get(0).(*unstructured.Unstructured), args.Error(1)
}

func (m *MockNamespaceableResource) List(ctx context.Context, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(*unstructured.UnstructuredList), args.Error(1)
}

func (m *MockNamespaceableResource) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(watch.Interface), args.Error(1)
}

func (m *MockNamespaceableResource) Patch(ctx context.Context, name string, pt types.PatchType,
	data []byte, options metav1.PatchOptions, subresources ...string,
) (*unstructured.Unstructured, error) {
	args := m.Called(ctx, name, pt, data, options, subresources)
	return args.Get(0).(*unstructured.Unstructured), args.Error(1)
}

func (m *MockNamespaceableResource) Apply(ctx context.Context, name string, obj *unstructured.Unstructured,
	options metav1.ApplyOptions, subresources ...string,
) (*unstructured.Unstructured, error) {
	args := m.Called(ctx, name, obj, options, subresources)
	return args.Get(0).(*unstructured.Unstructured), args.Error(1)
}

func (m *MockNamespaceableResource) ApplyStatus(ctx context.Context, name string,
	obj *unstructured.Unstructured, options metav1.ApplyOptions,
) (*unstructured.Unstructured, error) {
	args := m.Called(ctx, name, obj, options)
	return args.Get(0).(*unstructured.Unstructured), args.Error(1)
}

func Test_unsetEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Test Case",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unsetEnvironmentVariables()
		})
	}
}

func Test_readUIDFromFile(t *testing.T) {
	type args struct {
		filePath string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "Test Case 1",
			args:    args{},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readUIDFromFile(tt.args.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("readUIDFromFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("readUIDFromFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVoucherExtension(t *testing.T) {
	type args struct {
		hostIP       string
		deviceSerial string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "Test Case 1",
			args:    args{},
			want:    "",
			wantErr: true,
		},
		{
			name: "Test Case 2",
			args: args{
				deviceSerial: "123",
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := VoucherExtension(tt.args.hostIP, tt.args.deviceSerial)
			if (err != nil) != tt.wantErr {
				t.Errorf("VoucherExtension() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("VoucherExtension() = %v, want %v", got, tt.want)
			}
		})
	}
}
