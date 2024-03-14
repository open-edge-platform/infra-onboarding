/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onbworkflowclient

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/user"
	"reflect"
	"testing"

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

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
)

const voucherEndPoint = "/api/v1/owner/vouchers/"

// MockHTTPServer creates a mock HTTP server and returns its URL.
func MockHTTPServer(handler http.Handler) (*httptest.Server, string) {
	server := httptest.NewServer(handler)
	return server, server.URL
}

func TestVoucherScript(t *testing.T) {
	type args struct {
		deviceInfo utils.DeviceInfo
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"test",
			args{},
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := VoucherScript(tt.args.deviceInfo)
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

func TestVoucherScript_Case(t *testing.T) {
	rvEnabled = flag.Bool("rvenable", true, "Set to true if you have enabled rv")
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Error generating private key: %v\n", err)
	}
	certTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
	}
	certData, err := x509.CreateCertificate(rand.Reader, &certTemplate, &certTemplate, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("Error creating certificate: %v\n", err)
	}
	usr, err := user.Current()
	if err != nil {
		t.Fatalf("Currrent user error : %v", err)
	}
	scriptDir := usr.HomeDir + "/.fdo-secrets/scripts/secrets/"
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		t.Fatalf("Error creating directory: %v\n", err)
	}
	fmt.Printf("Directory %s created successfully.\n", scriptDir)
	certOut, err := os.Create(scriptDir + "/ca-cert.pem")
	if err != nil {
		t.Fatalf("Failed to create certificate file: %v", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certData})
	certOut.Close()
	certKeyOut, err := os.Create(scriptDir + "/api-user.pem")
	if err != nil {
		t.Fatalf("Failed to create certificate key file: %v", err)
	}
	pem.Encode(certKeyOut, &pem.Block{Type: "CERTIFICATE", Bytes: certData})
	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Unable to marshal private key: %v", err)
	}
	pem.Encode(certKeyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	certKeyOut.Close()
	defer func() {
		if err := os.RemoveAll(scriptDir); err != nil {
			t.Fatalf("Error removing directory: %v", err)
		}
	}()
	listener, err := net.Listen("tcp", "localhost:58042")
	if err != nil {
		t.Fatalf("Error creating listener: %v", err)
	}
	defer listener.Close()
	listeners, lerr := net.Listen("tcp", "localhost:58039")
	if lerr != nil {
		t.Fatalf("Error creating listener: %v", err)
	}
	defer listeners.Close()
	os.Setenv("USER", "dummy")
	defer os.Unsetenv("USER")
	newPath := "/home"
	dir := newPath + "/dummy/.fdo-secrets/scripts/"
	if derr := os.MkdirAll(dir, 0755); derr != nil {
		t.Fatalf("Error creating text directory: %v\n", derr)
	}
	defer os.RemoveAll(newPath + "/dummy")
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/certificate" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock certificate response"))
		} else if r.URL.Path == "/api/v1/mfg/vouchers/123" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock voucher response"))
		} else if r.URL.Path == "/api/v1/certificate?alias=SECP256R1" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock voucher response"))
		} else if r.URL.Path == voucherEndPoint {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock owner voucher response"))
		} else if r.URL.Path == "/api/v1/to0/Mock owner voucher response" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock TO0 response"))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not found"))
		}
	}))
	server.Listener = listener
	server.Start()
	defer server.Close()
	server1 := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/certificate" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock certificate response"))
		} else if r.URL.Path == "/api/v1/mfg/vouchers/123" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock voucher response"))
		} else if r.URL.Path == "/api/v1/certificate?alias=SECP256R1" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock voucher response"))
		} else if r.URL.Path == voucherEndPoint {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock owner voucher response"))
		} else if r.URL.Path == "/api/v1/to0/" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock TO0 response"))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not found"))
		}
	}))
	server1.Listener = listeners
	server1.Start()
	defer server1.Close()
	// hostIP := ""
	// deviceSerial := "123"
	result, err := VoucherScript(utils.DeviceInfo{
		HwSerialID:  "123",
		FdoOwnerDNS: "localhost",
		FdoMfgDNS:   "localhost",
	})
	fmt.Println(result)
	assert.NoError(t, err)
	defer func() {
		rvEnabled = flag.Bool("rvenabl", false, "Set to true if you have enabled rv")
	}()
}

func TestVoucherExtension_Case(t *testing.T) {
	usr, err := user.Current()
	if err != nil {
		t.Fatalf("Currrent user error : %v", err)
	}
	scriptDir := usr.HomeDir + "/pri-fidoiot/component-samples/demo/scripts/"
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		return
	}
	fmt.Printf("Directory %s created successfully.\n", scriptDir)
	scriptContent := []byte("#!/bin/bash\n\n# Your script content here\n")
	shFilePath := scriptDir + "extend_upload.sh"
	if err := ioutil.WriteFile(shFilePath, scriptContent, 0755); err != nil {
		t.Fatalf("Error creating shell script file: %v", err)
	}

	fmt.Printf("Shell script file %s created successfully.\n", shFilePath)
	defer func() {
		if err := os.RemoveAll(scriptDir); err != nil {
			t.Fatalf("Error removing directory: %v", err)
		}
	}()
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

func TestVoucherScript_Case1(t *testing.T) {
	rvEnabled = flag.Bool("rvena", true, "Set to true if you have enabled rv")
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Error generating private key: %v\n", err)
	}
	certTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
	}
	certData, err := x509.CreateCertificate(rand.Reader, &certTemplate, &certTemplate, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("Error creating certificate: %v\n", err)
	}
	usr, err := user.Current()
	if err != nil {
		t.Fatalf("Currrent user error : %v", err)
	}
	scriptDir := usr.HomeDir + "/.fdo-secrets/scripts/secrets/"
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		t.Fatalf("Error creating directory: %v\n", err)
	}
	fmt.Printf("Directory %s created successfully.\n", scriptDir)
	certOut, err := os.Create(scriptDir + "/ca-cert.pem")
	if err != nil {
		t.Fatalf("Failed to create certificate file: %v", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certData})
	certOut.Close()
	certKeyOut, err := os.Create(scriptDir + "/api-user.pem")
	if err != nil {
		t.Fatalf("Failed to create certificate key file: %v", err)
	}
	pem.Encode(certKeyOut, &pem.Block{Type: "CERTIFICATE", Bytes: certData})
	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Unable to marshal private key: %v", err)
	}
	pem.Encode(certKeyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	certKeyOut.Close()
	defer func() {
		if err := os.RemoveAll(scriptDir); err != nil {
			t.Fatalf("Error removing directory: %v", err)
		}
	}()
	listener, err := net.Listen("tcp", "localhost:58042")
	if err != nil {
		t.Fatalf("Error creating listener: %v", err)
	}
	defer listener.Close()
	listeners, lerr := net.Listen("tcp", "localhost:58039")
	if lerr != nil {
		t.Fatalf("Error creating listener: %v", err)
	}
	defer listeners.Close()
	os.Setenv("USER", "dummy")
	defer os.Unsetenv("USER")
	newPath := "/home"
	dir := newPath + "/dummy/.fdo-secrets/scripts/"
	if derr := os.MkdirAll(dir, 0755); derr != nil {
		t.Fatalf("Error creating text directory: %v\n", derr)
	}
	defer os.RemoveAll(newPath + "/dummy")
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/certificate" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock certificate response"))
		} else if r.URL.Path == "/api/v1/mfg/vouchers/123" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock voucher response"))
		} else if r.URL.Path == "/api/v1/certificate?alias=SECP256R1" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock voucher response"))
		} else if r.URL.Path == voucherEndPoint {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock owner voucher response"))
		} else if r.URL.Path == "/api/v1/to0/Mock owner voucher response" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock TO0 response"))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not found"))
		}
	}))
	server.Listener = listener
	server.Start()
	defer server.Close()
	server1 := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/certificate" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock certificate response"))
		} else if r.URL.Path == "/api/v1/mfg/vouchers/123" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock voucher response"))
		} else if r.URL.Path == "/api/v1/certificate?alias=SECP256R1" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock voucher response"))
		} else if r.URL.Path == voucherEndPoint {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock owner voucher response"))
		} else if r.URL.Path == "/api/v1/to0/" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock TO0 response"))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not found"))
		}
	}))
	server1.Listener = listeners
	server1.Start()
	defer server1.Close()
	result, err := VoucherScript(utils.DeviceInfo{
		HwSerialID:  "123",
		FdoOwnerDNS: "localhost",
		FdoMfgDNS:   "localhost",
	})
	fmt.Println(result)
	assert.NoError(t, err)
	defer func() {
		rvEnabled = flag.Bool("rvabl", false, "Set to true if you have enabled rv")
	}()
}
func TestVoucherScript_Case2(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Error generating private key: %v\n", err)
	}
	certTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
	}
	certData, err := x509.CreateCertificate(rand.Reader, &certTemplate, &certTemplate, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("Error creating certificate: %v\n", err)
	}
	usr, err := user.Current()
	if err != nil {
		t.Fatalf("Currrent user error : %v", err)
	}
	scriptDir := usr.HomeDir + "/.fdo-secrets/scripts/secrets/"
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		t.Fatalf("Error creating directory: %v\n", err)
	}
	fmt.Printf("Directory %s created successfully.\n", scriptDir)
	certOut, err := os.Create(scriptDir + "/ca-cert.pem")
	if err != nil {
		t.Fatalf("Failed to create certificate file: %v", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certData})
	certOut.Close()
	certKeyOut, err := os.Create(scriptDir + "/api-user.pem")
	if err != nil {
		t.Fatalf("Failed to create certificate key file: %v", err)
	}
	pem.Encode(certKeyOut, &pem.Block{Type: "CERTIFICATE", Bytes: certData})
	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Unable to marshal private key: %v", err)
	}
	pem.Encode(certKeyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	certKeyOut.Close()
	defer func() {
		if err := os.RemoveAll(scriptDir); err != nil {
			t.Fatalf("Error removing directory: %v", err)
		}
	}()
	listener, err := net.Listen("tcp", "localhost:58042")
	if err != nil {
		t.Fatalf("Error creating listener: %v", err)
	}
	defer listener.Close()
	listeners, lerr := net.Listen("tcp", "localhost:58039")
	if lerr != nil {
		t.Fatalf("Error creating listener: %v", err)
	}
	defer listeners.Close()
	os.Setenv("USER", "dummy")
	defer os.Unsetenv("USER")
	newPath := "/home"
	dir := newPath + "/dummy/.fdo-secrets/scripts/"
	if derr := os.MkdirAll(dir, 0755); derr != nil {
		t.Fatalf("Error creating text directory: %v\n", derr)
	}
	defer os.RemoveAll(newPath + "/dummy")
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/certificate" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Mock certificate response"))
		} else if r.URL.Path == "/api/v1/mfg/vouchers/123" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock voucher response"))
		} else if r.URL.Path == "/api/v1/certificate?alias=SECP256R1" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Mock voucher response"))
		} else if r.URL.Path == voucherEndPoint {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock owner voucher response"))
		} else if r.URL.Path == "/api/v1/to0/Mock owner voucher response" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock TO0 response"))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not found"))
		}
	}))
	server.Listener = listener
	server.Start()
	defer server.Close()
	server1 := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/certificate" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Mock certificate response"))
		} else if r.URL.Path == "/api/v1/mfg/vouchers/123" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock voucher response"))
		} else if r.URL.Path == "/api/v1/certificate?alias=SECP256R1" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Mock voucher response"))
		} else if r.URL.Path == voucherEndPoint {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock owner voucher response"))
		} else if r.URL.Path == "/api/v1/to0/" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock TO0 response"))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not found"))
		}
	}))
	server1.Listener = listeners
	server1.Start()
	defer server1.Close()
	result, err := VoucherScript(utils.DeviceInfo{
		HwSerialID:  "123",
		FdoOwnerDNS: "localhost",
		FdoMfgDNS:   "localhost",
	})
	fmt.Println(result)
	assert.Error(t, err)
}

func TestVoucherScript_Case3(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Error generating private key: %v\n", err)
	}
	certTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
	}
	certData, err := x509.CreateCertificate(rand.Reader, &certTemplate, &certTemplate, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("Error creating certificate: %v\n", err)
	}
	usr, err := user.Current()
	if err != nil {
		t.Fatalf("Currrent user error : %v", err)
	}
	scriptDir := usr.HomeDir + "/.fdo-secrets/scripts/secrets/"
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		t.Fatalf("Error creating directory: %v\n", err)
	}
	fmt.Printf("Directory %s created successfully.\n", scriptDir)
	certOut, err := os.Create(scriptDir + "/ca-cert.pem")
	if err != nil {
		t.Fatalf("Failed to create certificate file: %v", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certData})
	certOut.Close()
	certKeyOut, err := os.Create(scriptDir + "/api-user.pem")
	if err != nil {
		t.Fatalf("Failed to create certificate key file: %v", err)
	}
	pem.Encode(certKeyOut, &pem.Block{Type: "CERTIFICATE", Bytes: certData})
	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Unable to marshal private key: %v", err)
	}
	pem.Encode(certKeyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	certKeyOut.Close()
	defer func() {
		if err := os.RemoveAll(scriptDir); err != nil {
			t.Fatalf("Error removing directory: %v", err)
		}
	}()
	listener, err := net.Listen("tcp", "localhost:58042")
	if err != nil {
		t.Fatalf("Error creating listener: %v", err)
	}
	defer listener.Close()
	listeners, lerr := net.Listen("tcp", "localhost:58039")
	if lerr != nil {
		t.Fatalf("Error creating listener: %v", err)
	}
	defer listeners.Close()
	os.Setenv("USER", "dummy")
	defer os.Unsetenv("USER")
	newPath := "/home"
	dir := newPath + "/dummy/.fdo-secrets/scripts/"
	if derr := os.MkdirAll(dir, 0755); derr != nil {
		t.Fatalf("Error creating text directory: %v\n", derr)
	}
	defer os.RemoveAll(newPath + "/dummy")
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/certificate" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock certificate response"))
		} else if r.URL.Path == "/api/v1/mfg/vouchers/123" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Mock voucher response"))
		} else if r.URL.Path == "/api/v1/certificate?alias=SECP256R1" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock voucher response"))
		} else if r.URL.Path == voucherEndPoint {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock owner voucher response"))
		} else if r.URL.Path == "/api/v1/to0/Mock owner voucher response" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock TO0 response"))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not found"))
		}
	}))
	server.Listener = listener
	server.Start()
	defer server.Close()
	server1 := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/certificate" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock certificate response"))
		} else if r.URL.Path == "/api/v1/mfg/vouchers/123" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Mock voucher response"))
		} else if r.URL.Path == "/api/v1/certificate?alias=SECP256R1" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock voucher response"))
		} else if r.URL.Path == voucherEndPoint {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock owner voucher response"))
		} else if r.URL.Path == "/api/v1/to0/" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock TO0 response"))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not found"))
		}
	}))
	server1.Listener = listeners
	server1.Start()
	defer server1.Close()
	result, err := VoucherScript(utils.DeviceInfo{
		HwSerialID:  "123",
		FdoOwnerDNS: "localhost",
		FdoMfgDNS:   "localhost",
	})
	fmt.Println(result)
	assert.Error(t, err)
}

func TestVoucherScript_Case4(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Error generating private key: %v\n", err)
	}
	certTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
	}
	certData, err := x509.CreateCertificate(rand.Reader, &certTemplate, &certTemplate, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("Error creating certificate: %v\n", err)
	}
	usr, err := user.Current()
	if err != nil {
		t.Fatalf("Currrent user error : %v", err)
	}
	scriptDir := usr.HomeDir + "/.fdo-secrets/scripts/secrets/"
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		t.Fatalf("Error creating directory: %v\n", err)
	}
	fmt.Printf("Directory %s created successfully.\n", scriptDir)
	certOut, err := os.Create(scriptDir + "/ca-cert.pem")
	if err != nil {
		t.Fatalf("Failed to create certificate file: %v", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certData})
	certOut.Close()
	certKeyOut, err := os.Create(scriptDir + "/api-user.pem")
	if err != nil {
		t.Fatalf("Failed to create certificate key file: %v", err)
	}
	pem.Encode(certKeyOut, &pem.Block{Type: "CERTIFICATE", Bytes: certData})
	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Unable to marshal private key: %v", err)
	}
	pem.Encode(certKeyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	certKeyOut.Close()
	defer func() {
		if err := os.RemoveAll(scriptDir); err != nil {
			t.Fatalf("Error removing directory: %v", err)
		}
	}()
	listener, err := net.Listen("tcp", "localhost:58042")
	if err != nil {
		t.Fatalf("Error creating listener: %v", err)
	}
	defer listener.Close()
	listeners, lerr := net.Listen("tcp", "localhost:58039")
	if lerr != nil {
		t.Fatalf("Error creating listener: %v", err)
	}
	defer listeners.Close()
	os.Setenv("USER", "dummy")
	defer os.Unsetenv("USER")
	newPath := "/home"
	dir := newPath + "/dummy/.fdo-secrets/scripts/"
	if derr := os.MkdirAll(dir, 0755); derr != nil {
		t.Fatalf("Error creating text directory: %v\n", derr)
	}
	defer os.RemoveAll(newPath + "/dummy")
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/certificate" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock certificate response"))
		} else if r.URL.Path == "/api/v1/mfg/vouchers/123" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock voucher response"))
		} else if r.URL.Path == "/api/v1/certificate?alias=SECP256R1" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock voucher response"))
		} else if r.URL.Path == voucherEndPoint {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Mock owner voucher response"))
		} else if r.URL.Path == "/api/v1/to0/Mock owner voucher response" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock TO0 response"))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not found"))
		}
	}))
	server.Listener = listener
	server.Start()
	defer server.Close()
	server1 := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/certificate" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock certificate response"))
		} else if r.URL.Path == "/api/v1/mfg/vouchers/123" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock voucher response"))
		} else if r.URL.Path == "/api/v1/certificate?alias=SECP256R1" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock voucher response"))
		} else if r.URL.Path == voucherEndPoint {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Mock owner voucher response"))
		} else if r.URL.Path == "/api/v1/to0/" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock TO0 response"))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not found"))
		}
	}))
	server1.Listener = listeners
	server1.Start()
	defer server1.Close()
	result, err := VoucherScript(utils.DeviceInfo{
		HwSerialID:  "123",
		FdoOwnerDNS: "localhost",
		FdoMfgDNS:   "localhost",
	})
	fmt.Println(result)
	assert.Error(t, err)
}

func TestVoucherScript_Case5(t *testing.T) {
	rvEnabled = flag.Bool("rven", true, "Set to true if you have enabled rv")
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Error generating private key: %v\n", err)
	}
	certTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
	}
	certData, err := x509.CreateCertificate(rand.Reader, &certTemplate, &certTemplate, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("Error creating certificate: %v\n", err)
	}
	usr, err := user.Current()
	if err != nil {
		t.Fatalf("Currrent user error : %v", err)
	}
	scriptDir := usr.HomeDir + "/.fdo-secrets/scripts/secrets/"
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		t.Fatalf("Error creating directory: %v\n", err)
	}
	fmt.Printf("Directory %s created successfully.\n", scriptDir)
	certOut, err := os.Create(scriptDir + "/ca-cert.pem")
	if err != nil {
		t.Fatalf("Failed to create certificate file: %v", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certData})
	certOut.Close()
	certKeyOut, err := os.Create(scriptDir + "/api-user.pem")
	if err != nil {
		t.Fatalf("Failed to create certificate key file: %v", err)
	}
	pem.Encode(certKeyOut, &pem.Block{Type: "CERTIFICATE", Bytes: certData})
	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Unable to marshal private key: %v", err)
	}
	pem.Encode(certKeyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	certKeyOut.Close()
	defer func() {
		if err := os.RemoveAll(scriptDir); err != nil {
			t.Fatalf("Error removing directory: %v", err)
		}
	}()
	listener, err := net.Listen("tcp", "localhost:58042")
	if err != nil {
		t.Fatalf("Error creating listener: %v", err)
	}
	defer listener.Close()
	listeners, lerr := net.Listen("tcp", "localhost:58039")
	if lerr != nil {
		t.Fatalf("Error creating listener: %v", err)
	}
	defer listeners.Close()
	os.Setenv("USER", "dummy")
	defer os.Unsetenv("USER")
	newPath := "/home"
	dir := newPath + "/dummy/.fdo-secrets/scripts/"
	if derr := os.MkdirAll(dir, 0755); derr != nil {
		t.Fatalf("Error creating text directory: %v\n", derr)
	}
	defer os.RemoveAll(newPath + "/dummy")
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/certificate" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock certificate response"))
		} else if r.URL.Path == "/api/v1/mfg/vouchers/123" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock voucher response"))
		} else if r.URL.Path == "/api/v1/certificate?alias=SECP256R1" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock voucher response"))
		} else if r.URL.Path == voucherEndPoint {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock owner voucher response"))
		} else if r.URL.Path == "/api/v1/to0/Mock owner voucher response" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Mock TO0 response"))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not found"))
		}
	}))
	server.Listener = listener
	server.Start()
	defer server.Close()
	server1 := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/certificate" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock certificate response"))
		} else if r.URL.Path == "/api/v1/mfg/vouchers/123" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock voucher response"))
		} else if r.URL.Path == "/api/v1/certificate?alias=SECP256R1" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock voucher response"))
		} else if r.URL.Path == voucherEndPoint {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock owner voucher response"))
		} else if r.URL.Path == "/api/v1/to0/" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock TO0 response"))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not found"))
		}
	}))
	server1.Listener = listeners
	server1.Start()
	defer server1.Close()
	result, err := VoucherScript(utils.DeviceInfo{
		HwSerialID:  "123",
		FdoOwnerDNS: "localhost",
		FdoMfgDNS:   "localhost",
	})
	fmt.Println(result)
	assert.Error(t, err)
	defer func() {
		rvEnabled = flag.Bool("ren", true, "Set to true if you have enabled rv")
	}()
}

func Test_apiCalls(t *testing.T) {
	type args struct {
		httpMethod   string
		url          string
		authType     string
		apiUser      string
		onrApiPasswd string
		bodyData     []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *http.Response
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				authType: "mtls",
			},
			wantErr: true,
		},
		{
			name: "Test Case",
			args: args{
				authType: "abc",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := apiCalls(tt.args.httpMethod, tt.args.url, tt.args.apiUser, tt.args.onrApiPasswd, tt.args.bodyData)
			if (err != nil) != tt.wantErr {
				t.Errorf("apiCalls() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("apiCalls() = %v, want %v", got, tt.want)
			}
		})
	}
}
