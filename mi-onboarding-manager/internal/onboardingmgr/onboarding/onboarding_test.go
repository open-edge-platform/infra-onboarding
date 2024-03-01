/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	dkam "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/api/grpc/dkammgr"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	onboarding "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/onboarding/onboardingmocks"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/api"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/os/v1"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

func TestInitOnboarding(t *testing.T) {
	type args struct {
		invClient *invclient.OnboardingInventoryClient
		dkamAddr  string
	}
	mockInvClient := &onboarding.MockInventoryClient{}
	inputargs := args{
		invClient: &invclient.OnboardingInventoryClient{
			Client: mockInvClient,
		},
	}
	inputargs1 := args{
		invClient: nil,
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "positive",
			args: inputargs,
		},
		{
			name: "negative",
			args: inputargs1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitOnboarding(tt.args.invClient, tt.args.dkamAddr)
		})
	}
}

func TestDeviceOnboardingManagerZt(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "sut_onboarding_list.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	type args struct {
		deviceInfo     utils.DeviceInfo
		kubeconfigPath string
		sutlabel       string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "negative",
			args: args{
				deviceInfo:     utils.DeviceInfo{},
				kubeconfigPath: "",
				sutlabel:       "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DeviceOnboardingManagerZt(tt.args.deviceInfo, tt.args.sutlabel); (err != nil) != tt.wantErr {
				t.Errorf("DeviceOnboardingManagerZt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConvertInstanceForOnboarding_Err(t *testing.T) {
	osInstance := &osv1.OperatingSystemResource{
		RepoUrl: "osUrl.raw.gz;overlayUrl",
	}
	host := &computev1.HostResource{
		HostNics: []*computev1.HostnicResource{
			{
				MacAddr:      "00:00:00:00:00:00",
				BmcInterface: true,
			},
		},
	}

	tests := []struct {
		name        string
		osResources []*osv1.OperatingSystemResource
		host        *computev1.HostResource
		instance    *computev1.InstanceResource
		want        []*pb.OnboardingRequest
		wantErr     bool
	}{
		{
			name:        "Test case 1",
			osResources: []*osv1.OperatingSystemResource{osInstance},
			host:        host,
			want: []*pb.OnboardingRequest{
				{
					ArtifactData: []*pb.ArtifactData{
						{
							Name:       "OS",
							PackageUrl: "osUrl.raw.gz",
							Category:   1,
						},
						{
							Name:       "PLATFORM",
							PackageUrl: "overlayUrl",
							Category:   1,
						},
					},
					Hwdata: []*pb.HwData{
						{
							MacId:         "00:00:00:00:00:00",
							DiskPartition: "123",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Test case - 2",
			osResources: []*osv1.OperatingSystemResource{
				{RepoUrl: "invalidUrl"},
			},
			host:    host,
			want:    nil,
			wantErr: true,
		},
		{
			name:        "Test case -3",
			osResources: []*osv1.OperatingSystemResource{osInstance},
			host:        &computev1.HostResource{},
			want:        nil,
			wantErr:     true,
		},
		{
			name:        "Test case - 4",
			osResources: []*osv1.OperatingSystemResource{osInstance},
			host: &computev1.HostResource{
				HostNics: []*computev1.HostnicResource{
					{
						MacAddr:      "00:00:00:00:00:00",
						BmcInterface: false, // BMC interface not enabled
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertInstanceForOnboarding(tt.osResources, tt.host, tt.instance)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertInstanceForOnboarding() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertInstanceForOnboarding() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetOSResourceFromDkamService(t *testing.T) {
	type args struct {
		ctx         context.Context
		profilename string
		platform    string
	}
	tests := []struct {
		name    string
		args    args
		want    *dkam.GetArtifactsResponse
		wantErr bool
	}{
		{
			name: "TestCase1",
			args: args{
				ctx: context.TODO(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "TestCase2",
			args: args{
				ctx: context.TODO(),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetOSResourceFromDkamService(tt.args.ctx, tt.args.profilename, tt.args.platform)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetOSResourceFromDkamService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetOSResourceFromDkamService() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOnboardingManager_StartOnboarding(t *testing.T) {
	type fields struct {
		OnBoardingEBServer pb.OnBoardingEBServer
	}
	type args struct {
		ctx context.Context
		req *pb.OnboardingRequest
	}
	t.Setenv("PD_IP", "000.000.0.000")
	defer os.Unsetenv("PD_IP")
	t.Setenv("IMAGE_TYPE", "prod_focal-ms")
	defer os.Unsetenv("IMAGE_TYPE")
	hwdata := &pb.HwData{Uuid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad", SutIp: "00.00.00.00"}
	dirPath, _ := os.Getwd()
	dirPath, _ = strings.CutSuffix(dirPath, "internal/onboardingmgr/onboarding")
	dirPaths := dirPath + "/cmd/onboardingmgr"
	err := os.Chdir(dirPaths)
	if err != nil {
		t.Fatalf("Failed to change working directory: %v", err)
	}
	hwdatas := []*pb.HwData{hwdata}
	mockClient := &onboarding.MockInventoryClient{}
	mockResources := &inv_v1.ListResourcesResponse{}
	mockClient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil)
	mockClient.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	_invClient = &invclient.OnboardingInventoryClient{
		Client: mockClient,
	}
	artifactDatasPlatform := []*pb.ArtifactData{
		{
			Category: pb.ArtifactData_PLATFORM,
			Name:     "PLATFORM",
		},
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.OnboardingResponse
		wantErr bool
	}{
		{
			name:   "Test Case",
			fields: fields{},
			args: args{
				ctx: context.TODO(),
				req: &pb.OnboardingRequest{
					Hwdata:       hwdatas,
					ArtifactData: artifactDatasPlatform,
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	originalDir, _ := os.Getwd()
	err = os.Chdir(originalDir)
	if err != nil {
		t.Fatalf("Failed to change working directory back to original: %v", err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &OnboardingManager{
				OnBoardingEBServer: tt.fields.OnBoardingEBServer,
			}
			got, err := s.StartOnboarding(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingManager.StartOnboarding() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OnboardingManager.StartOnboarding() = %v, want %v", got, tt.want)
			}
		})
	}
	os.Remove(dirPath + "/internal/onboardingmgr/azure_env/" + "azure-credentials.env_")
	os.Remove(dirPath + "/internal/onboardingmgr/azure_env/" + "azure-credentials.env_00:00:00:00:00:00")
}

func Test_parseNGetBkcURL(t *testing.T) {
	type args struct {
		onboardingRequest *pb.OnboardingRequest
	}
	tests := []struct {
		name string
		args args
		want utils.ArtifactData
	}{
		{
			name: "Test Case",
			args: args{
				&pb.OnboardingRequest{
					ArtifactData: []*pb.ArtifactData{
						{
							Name: "OS",
						},
					},
				},
			},
			want: utils.ArtifactData{},
		},
		{
			name: "Test Case 1",
			args: args{
				&pb.OnboardingRequest{
					ArtifactData: []*pb.ArtifactData{
						{
							Name: "OST",
						},
					},
				},
			},
			want: utils.ArtifactData{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseNGetBkcURL(tt.args.onboardingRequest); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseNGetBkcURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOnboardingManager_StartOnboarding_Case(t *testing.T) {
	type fields struct {
		OnBoardingEBServer pb.OnBoardingEBServer
	}
	type args struct {
		ctx context.Context
		req *pb.OnboardingRequest
	}
	t.Setenv("PD_IP", "000.000.0.000")
	defer os.Unsetenv("PD_IP")
	t.Setenv("IMAGE_TYPE", "prod_focal-ms")
	defer os.Unsetenv("IMAGE_TYPE")
	hwdata := &pb.HwData{Uuid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad", SutIp: "00.00.00.00"}
	dirPath, _ := os.Getwd()
	dirPath, _ = strings.CutSuffix(dirPath, "internal/onboardingmgr/onboarding")
	err := os.Chdir(dirPath)
	if err != nil {
		t.Fatalf("Failed to change working directory: %v", err)
	}
	hwdatas := []*pb.HwData{hwdata}
	mockClient := &onboarding.MockInventoryClient{}
	mockResources := &inv_v1.ListResourcesResponse{}
	mockClient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil)
	mockClient.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, errors.New("err"))
	_invClient = &invclient.OnboardingInventoryClient{
		Client: mockClient,
	}
	artifactDatasPlatform := []*pb.ArtifactData{
		{
			Category: pb.ArtifactData_PLATFORM,
			Name:     "PLATFORM",
		},
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.OnboardingResponse
		wantErr bool
	}{
		{
			name:   "Test Case",
			fields: fields{},
			args: args{
				ctx: context.TODO(),
				req: &pb.OnboardingRequest{
					Hwdata:       hwdatas,
					ArtifactData: artifactDatasPlatform,
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	originalDir, _ := os.Getwd()
	err = os.Chdir(originalDir)
	if err != nil {
		t.Fatalf("Failed to change working directory back to original: %v", err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &OnboardingManager{
				OnBoardingEBServer: tt.fields.OnBoardingEBServer,
			}
			got, err := s.StartOnboarding(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingManager.StartOnboarding() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OnboardingManager.StartOnboarding() = %v, want %v", got, tt.want)
			}
		})
	}
	os.Remove(dirPath + "/internal/onboardingmgr/azure_env/" + "azure-credentials.env_")
	os.Remove(dirPath + "/internal/onboardingmgr/azure_env/" + "azure-credentials.env_00:00:00:00:00:00")
}

func TestHandleSecureBootMismatch(t *testing.T) {
	type args struct {
		ctx context.Context
		req *pb.SecureBootResponse
	}
	mockClient := &onboarding.MockInventoryClient{}
	_invClient = &invclient.OnboardingInventoryClient{
		Client: mockClient,
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
				req: &pb.SecureBootResponse{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := IsSecureBootConfigAtEdgeNodeMismatch(tt.args.ctx, tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf("HandleSecureBootMismatch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandleSecureBootMismatch_Case(t *testing.T) {
	type args struct {
		ctx context.Context
		req *pb.SecureBootResponse
	}
	mockClient := &onboarding.MockInventoryClient{}
	mockHost := &computev1.HostResource{
		ResourceId: "host-084d9b08",
		Instance: &computev1.InstanceResource{
			ResourceId: "inst-084d9b08",
		},
	}
	mockResource := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: mockHost,
		},
	}
	mockResources := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource}},
	}
	mockClient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil)
	mockClient.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	_invClient = &invclient.OnboardingInventoryClient{
		Client: mockClient,
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
				req: &pb.SecureBootResponse{
					Guid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := IsSecureBootConfigAtEdgeNodeMismatch(tt.args.ctx, tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf("HandleSecureBootMismatch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandleSecureBootMismatch_Case1(t *testing.T) {
	type args struct {
		ctx context.Context
		req *pb.SecureBootResponse
	}
	mockClient := &onboarding.MockInventoryClient{}
	mockHost := &computev1.HostResource{
		ResourceId: "host-084d9b08",
		Instance: &computev1.InstanceResource{
			ResourceId: "inst-084d9b08",
		},
	}
	mockResource := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: mockHost,
		},
	}
	mockResources := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource}},
	}
	mockClient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil)
	mockClient.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, errors.New("err"))
	_invClient = &invclient.OnboardingInventoryClient{
		Client: mockClient,
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
				req: &pb.SecureBootResponse{
					Guid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := IsSecureBootConfigAtEdgeNodeMismatch(tt.args.ctx, tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf("HandleSecureBootMismatch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandleSecureBootMismatch_Case2(t *testing.T) {
	type args struct {
		ctx context.Context
		req *pb.SecureBootResponse
	}
	mockClient := &onboarding.MockInventoryClient{}
	mockHost := &computev1.HostResource{
		ResourceId: "host-084d9b08",
		Instance: &computev1.InstanceResource{
			ResourceId: "inst-084d9b08",
		},
	}
	mockResource := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: mockHost,
		},
	}
	mockResources := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource}},
	}
	mockClient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil)
	mockClient.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil).Once()
	mockClient.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, errors.New("err")).Once()
	_invClient = &invclient.OnboardingInventoryClient{
		Client: mockClient,
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
				req: &pb.SecureBootResponse{
					Guid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := IsSecureBootConfigAtEdgeNodeMismatch(tt.args.ctx, tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf("HandleSecureBootMismatch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOnboardingManager_SecureBootStatus(t *testing.T) {
	type fields struct {
		OnBoardingEBServer pb.OnBoardingEBServer
		OnBoardingSBServer pb.OnBoardingSBServer
	}
	type args struct {
		ctx context.Context
		req *pb.SecureBootStatRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.SecureBootResponse
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				ctx: context.Background(),
				req: &pb.SecureBootStatRequest{},
			},
			want:    &pb.SecureBootResponse{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &OnboardingManager{
				OnBoardingEBServer: tt.fields.OnBoardingEBServer,
				OnBoardingSBServer: tt.fields.OnBoardingSBServer,
			}
			got, err := s.SecureBootStatus(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingManager.SecureBootStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OnboardingManager.SecureBootStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeviceOnboardingManagerZt_Case(t *testing.T) {
	wd, _ := os.Getwd()
	fullPath := filepath.Join(wd, "sut_onboarding_list.txt")
	file, err := os.Create(fullPath)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()
	type args struct {
		deviceInfo     utils.DeviceInfo
		kubeconfigPath string
		sutlabel       string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "negative",
			args: args{
				deviceInfo:     utils.DeviceInfo{},
				kubeconfigPath: "",
				sutlabel:       "",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DeviceOnboardingManagerZt(tt.args.deviceInfo, tt.args.sutlabel); (err != nil) != tt.wantErr {
				t.Errorf("DeviceOnboardingManagerZt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	defer func() {
		if err := os.Remove(fullPath); err != nil {
			fmt.Println("Error removing file:", err)
		}
	}()
}

type MockClientConn struct {
	mock.Mock
}

// Invoke mocks the Invoke method of ClientConnInterface.
func (m *MockClientConn) Invoke(ctx context.Context, method string, args interface{}, reply interface{}, opts ...grpc.CallOption) error {
	argsMock := m.Called(ctx, method, args, reply, opts)
	return argsMock.Error(0)
}

// NewStream mocks the NewStream method of ClientConnInterface.
func (m *MockClientConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	argsMock := m.Called(ctx, desc, method, opts)
	return argsMock.Get(0).(grpc.ClientStream), argsMock.Error(1)
}

type MockDkamServiceClient struct {
	mock.Mock
}

// GetArtifacts mocks the GetArtifacts method of DkamServiceClient interface.
func (m *MockDkamServiceClient) GetArtifacts(ctx context.Context, in *dkam.GetArtifactsRequest, opts ...grpc.CallOption) (*GetArtifactsResponse, error) {
	args := m.Called(ctx, in, opts)
	return args.Get(0).(*GetArtifactsResponse), args.Error(1)
}

// func TestGetOSResourceFromDkamService_Case(t *testing.T) {
// 	os.Setenv("DKAMHOST", "localhost")
// 	os.Setenv("DKAMPORT", "7513")
// 	lis, err := net.Listen("tcp", "localhost:7513")
// 	if err != nil {
// 		t.Fatalf("Failed to listen: %v", err)
// 	}
// 	grpcServer := grpc.NewServer()
// 	go func() {
// 		defer lis.Close()
// 		if err := grpcServer.Serve(lis); err != nil {
// 			t.Fatalf("Failed to serve: %v", err)
// 		}
// 	}()
// 	mockClient := &MockClientConn{}
// 	mockClient.On("Invoke", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
// 	dkam.NewDkamServiceClient(mockClient)
// 	dkam.RegisterDkamServiceServer(grpcServer, dkam.DkamServiceServer{})
// 	conn, err := grpc.Dial("localhost:7513", grpc.WithInsecure())
// 	if err != nil {
// 		t.Fatalf("Failed to dial server: %v", err)
// 	}
// 	defer conn.Close()

// 	type args struct {
// 		ctx         context.Context
// 		profilename string
// 		platform    string
// 	}
// 	tests := []struct {
// 		name    string
// 		args    args
// 		want    *dkam.GetArtifactsResponse
// 		wantErr bool
// 	}{
// 		{
// 			name: "TestCase1",
// 			args: args{
// 				ctx: context.TODO(),
// 			},
// 			want:    nil,
// 			wantErr: true,
// 		},
// 		{
// 			name: "TestCase2",
// 			args: args{
// 				ctx: context.TODO(),
// 			},
// 			want:    nil,
// 			wantErr: true,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got, err := GetOSResourceFromDkamService(tt.args.ctx, tt.args.profilename, tt.args.platform)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("GetOSResourceFromDkamService() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("GetOSResourceFromDkamService() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }
