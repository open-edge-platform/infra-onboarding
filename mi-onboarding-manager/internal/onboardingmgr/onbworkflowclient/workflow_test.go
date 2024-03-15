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
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
	
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/status"
	"github.com/stretchr/testify/mock"
	tink "github.com/tinkerbell/tink/api/v1alpha1"
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
			name: "Test Case",
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
	listener, err := net.Listen("tcp", "localhost:58042")
	if err != nil {
		t.Fatalf("Error creating listener: %v", err)
	}
	defer listener.Close()
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/owner/state/" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock owner voucher response"))
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
			name: "Test Case1",
			args: args{
				in0: context.Background(),
				deviceInfo: utils.DeviceInfo{
					FdoOwnerDNS:  "localhost",
					FdoOwnerPort: "58042",
					FdoGUID:      "id",
				},
			},
			want:    false,
			wantErr: false,
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

func Test_checkTO2StatusCompleted_Case1(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:58042")
	if err != nil {
		t.Fatalf("Error creating listener: %v", err)
	}
	defer listener.Close()
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			name: "Test Case1",
			args: args{
				in0: context.Background(),
				deviceInfo: utils.DeviceInfo{
					FdoOwnerDNS:  "localhost",
					FdoOwnerPort: "58042",
					FdoGUID:      "id",
				},
			},
			want:    false,
			wantErr: false,
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
			name: "Test Case1",
			args: args{
				in0: context.Background(),
				deviceInfo: utils.DeviceInfo{
					FdoOwnerDNS:  "localhost",
					FdoOwnerPort: "58042",
					FdoGUID:      "id",
				},
			},
			want:    false,
			wantErr: false,
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
			name: "Test Case1",
			args: args{
				in0: context.Background(),
				deviceInfo: utils.DeviceInfo{
					FdoOwnerDNS:  "localhost",
					FdoOwnerPort: "58042",
					FdoGUID:      "id",
				},
			},
			want:    false,
			wantErr: false,
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
			name: "Test Case",
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

func TestCheckStatusOrRunProdWorkflow_Case1(t *testing.T) {
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
	cerr := os.MkdirAll(path+dummypath, 0755)
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
			name: "Test Case",
			args: args{
				ctx: context.Background(),
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
			if err := CheckStatusOrRunProdWorkflow(tt.args.ctx, tt.args.deviceInfo, tt.args.instance); (err != nil) != tt.wantErr {
				t.Errorf("CheckStatusOrRunProdWorkflow() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckTO2StatusOrRunFDOActions(t *testing.T) {
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
		} else if r.URL.Path == "/api/v1/owner/state/123" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"to2CompletedOn": "",
				"to0Expiry": ""
				}`))
		} else if r.URL.Path == "/api/v1/to0/Mock owner voucher response" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock TO0 response"))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"to2CompletedOn": "",
				"to0Expiry": ""
				}`))
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
			name: "Test Case",
			args: args{
				ctx: context.Background(),
				deviceInfo: utils.DeviceInfo{
					HwSerialID:   "123",
					FdoOwnerDNS:  "localhost",
					FdoMfgDNS:    "localhost",
					FdoOwnerPort: "58042",
				},
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
			if err := CheckTO2StatusOrRunFDOActions(tt.args.ctx, tt.args.deviceInfo, tt.args.instance); (err != nil) != tt.wantErr {
				t.Errorf("CheckTO2StatusOrRunFDOActions() error = %v, wantErr %v", err, tt.wantErr)
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
			name: "Test Case",
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
	cerr := os.MkdirAll(path+dummypath, 0755)
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
			name: "Test Case",
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
			name: "Test Case",
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
			name: "Test Case",
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
			name: "Test Case",
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
		onSuccessStatus           computev1.HostStatus
		onFailureStatus           computev1.HostStatus
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
				workflow: &tink.Workflow{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := handleWorkflowStatus(tt.args.instance, tt.args.workflow, tt.args.onSuccessStatus, tt.args.onFailureStatus, tt.args.onSuccessOnboardingStatus, tt.args.onFailureOnboardingStatus); (err != nil) != tt.wantErr {
				t.Errorf("handleWorkflowStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_handleWorkflowStatus_Case(t *testing.T) {
	type args struct {
		instance                  *computev1.InstanceResource
		workflow                  *tink.Workflow
		onSuccessStatus           computev1.HostStatus
		onFailureStatus           computev1.HostStatus
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
						State: tink.WorkflowStateSuccess,
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := handleWorkflowStatus(tt.args.instance, tt.args.workflow, tt.args.onSuccessStatus, tt.args.onFailureStatus, tt.args.onSuccessOnboardingStatus, tt.args.onFailureOnboardingStatus); (err != nil) != tt.wantErr {
				t.Errorf("handleWorkflowStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_handleWorkflowStatus_Case1(t *testing.T) {
	type args struct {
		instance                  *computev1.InstanceResource
		workflow                  *tink.Workflow
		onSuccessStatus           computev1.HostStatus
		onFailureStatus           computev1.HostStatus
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
						State: tink.WorkflowStateFailed,
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := handleWorkflowStatus(tt.args.instance, tt.args.workflow, tt.args.onSuccessStatus, tt.args.onFailureStatus, tt.args.onSuccessOnboardingStatus, tt.args.onFailureOnboardingStatus); (err != nil) != tt.wantErr {
				t.Errorf("handleWorkflowStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_handleWorkflowStatus_Case2(t *testing.T) {
	type args struct {
		instance                  *computev1.InstanceResource
		workflow                  *tink.Workflow
		onSuccessStatus           computev1.HostStatus
		onFailureStatus           computev1.HostStatus
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
						State: tink.WorkflowStateRunning,
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := handleWorkflowStatus(tt.args.instance, tt.args.workflow, tt.args.onSuccessStatus, tt.args.onFailureStatus, tt.args.onSuccessOnboardingStatus, tt.args.onFailureOnboardingStatus); (err != nil) != tt.wantErr {
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
	type args struct {
		ctx        context.Context
		k8sCli     client.Client
		deviceInfo utils.DeviceInfo
	}
	mockClient := MockClient{}
	mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockClient1 := MockClient{}
	mockClient1.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("err"))
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				k8sCli: mockClient,
			},
		},
		{
			name: "Test Case1",
			args: args{
				k8sCli: mockClient1,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := runProdWorkflow(tt.args.ctx, tt.args.k8sCli, tt.args.deviceInfo); (err != nil) != tt.wantErr {
				t.Errorf("runProdWorkflow() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_runDIWorkflow(t *testing.T) {
	type args struct {
		ctx        context.Context
		k8sCli     client.Client
		deviceInfo utils.DeviceInfo
	}
	mockClient := MockClient{}
	mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockClient1 := MockClient{}
	mockClient1.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("err"))
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				ctx: context.Background(),
				k8sCli: mockClient,
			},
		},
		{
			name: "Test Case1",
			args: args{
				ctx: context.Background(),
				k8sCli: mockClient1,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := runDIWorkflow(tt.args.ctx, tt.args.k8sCli, tt.args.deviceInfo); (err != nil) != tt.wantErr {
				t.Errorf("runDIWorkflow() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
