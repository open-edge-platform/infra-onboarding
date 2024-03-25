// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package tinkerbell

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	tink "github.com/tinkerbell/tink/api/v1alpha1"
	error_k8 "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
)

func TestNewHardware(t *testing.T) {
	type args struct {
		name    string
		ns      string
		id      string
		device  string
		ip      string
		gateway string
	}
	tests := []struct {
		name string
		args args
		want *tink.Hardware
	}{
		{
			name: "Test Case 1",
			args: args{},
			want: &tink.Hardware{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewHardware(tt.args.name, tt.args.ns, tt.args.id, tt.args.device,
				tt.args.ip, tt.args.gateway); reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewHardware() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newK8SClient(t *testing.T) {
	os.Setenv("KUBERNETES_SERVICE_HOST", "localhost")
	os.Setenv("KUBERNETES_SERVICE_PORT", "2521")
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Println("Failed to generate private key:", err)
		return
	}
	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{"Dummy Org"}},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caCertBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		fmt.Println("Failed to create CA certificate:", err)
		return
	}
	path := "/var"
	dummypath := "/run/secrets/kubernetes.io/serviceaccount/"
	cerr := os.MkdirAll(path+dummypath, 0o755)
	if cerr != nil {
		t.Fatalf("Error creating directory: %v", cerr)
	}
	file, crErr := os.Create(path + dummypath + "token")
	if crErr != nil {
		t.Fatalf("Error creating file: %v", crErr)
	}
	fmt.Println("token File :", file.Name())
	defer func() {
		remErr := os.RemoveAll("/run/secrets/kubernetes.io/serviceaccount/token")
		if remErr != nil {
			t.Fatalf("Error while removing file: %v", remErr)
		}
	}()
	dummyData := "Thisissomedummydata"
	_, err = file.WriteString(dummyData)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}
	certOut, cerrErr := os.Create(path + dummypath + "ca.crt")
	if cerrErr != nil {
		t.Fatalf("Error creating cert file: %v", cerrErr)
	}
	fmt.Println("certOut File :", certOut.Name())
	fmt.Println("CA certificate created successfully as ca.crt")
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: caCertBytes})
	defer func() {
		remErr := os.RemoveAll("/run/secrets/kubernetes.io/serviceaccount/ca.crt")
		if remErr != nil {
			t.Fatalf("Error while removing file: %v", remErr)
		}
	}()
	file.Close()
	certOut.Close()
	tests := []struct {
		name    string
		want    client.Client
		wantErr bool
	}{
		{
			name:    "Test Case",
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newK8SClient()
			if (err != nil) != tt.wantErr {
				t.Errorf("newK8SClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) {
				t.Errorf("newK8SClient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeleteHardwareForHostIfExist(t *testing.T) {
	os.Setenv("KUBERNETES_SERVICE_HOST", "localhost")
	os.Setenv("KUBERNETES_SERVICE_PORT", "2521")
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Println("Failed to generate private key:", err)
		return
	}
	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{"Dummy Org"}},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caCertBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		fmt.Println("Failed to create CA certificate:", err)
		return
	}
	path := "/var"
	dummypath := "/run/secrets/kubernetes.io/serviceaccount/"
	cerr := os.MkdirAll(path+dummypath, 0o755)
	if cerr != nil {
		t.Fatalf("Error creating directory: %v", cerr)
	}
	file, crErr := os.Create(path + dummypath + "token")
	if crErr != nil {
		t.Fatalf("Error creating file: %v", crErr)
	}
	fmt.Println("token File :", file.Name())
	defer func() {
		remErr := os.RemoveAll("/run/secrets/kubernetes.io/serviceaccount/token")
		if remErr != nil {
			t.Fatalf("Error while removing file: %v", remErr)
		}
	}()
	dummyData := "Thisissomedummydata"
	_, err = file.WriteString(dummyData)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}
	certOut, cerrErr := os.Create(path + dummypath + "ca.crt")
	if cerrErr != nil {
		t.Fatalf("Error creating cert file: %v", cerrErr)
	}
	fmt.Println("certOut File :", certOut.Name())
	fmt.Println("CA certificate created successfully as ca.crt")
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: caCertBytes})
	defer func() {
		remErr := os.RemoveAll("/run/secrets/kubernetes.io/serviceaccount/ca.crt")
		if remErr != nil {
			t.Fatalf("Error while removing file: %v", remErr)
		}
	}()
	file.Close()
	certOut.Close()
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
			name: "Test Case",
			args: args{
				ctx: context.Background(),
			},
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
			name: "Test Case",
			args: args{
				ctx:    context.Background(),
				k8sCli: mockClient,
			},
		},
		{
			name: "Test Case1",
			args: args{
				ctx:    context.Background(),
				k8sCli: mockClient1,
			},
			wantErr: true,
		},
		{
			name: "Test Case2",
			args: args{
				ctx:    context.Background(),
				k8sCli: mockClient2,
			},
			wantErr: false,
		},
		{
			name: "Test Case3",
			args: args{
				ctx:    context.Background(),
				k8sCli: mockClient3,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateHardwareIfNotExists(tt.args.ctx, tt.args.k8sCli, tt.args.k8sNamespace, tt.args.deviceInfo); (err != nil) != tt.wantErr {
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
			name: "Test Case",
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
			name: "Test Case",
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
			name: "Test Case",
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
