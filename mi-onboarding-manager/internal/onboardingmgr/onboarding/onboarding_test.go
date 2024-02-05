/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	"context"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"

	dkam "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/api/grpc/dkammgr"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/os/v1"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/onboardingmgr"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/invclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/utils"
	"github.com/stretchr/testify/mock"
)

func TestInitOnboarding(t *testing.T) {
	type args struct {
		invClient *invclient.OnboardingInventoryClient
		dkamAddr  string
	}
	mockInvClient := &MockInventoryClient{}
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
	tmpFile, err := ioutil.TempFile("", "sut_onboarding_list.txt")
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
			if err := DeviceOnboardingManagerZt(tt.args.deviceInfo, tt.args.kubeconfigPath, tt.args.sutlabel); (err != nil) != tt.wantErr {
				t.Errorf("DeviceOnboardingManagerZt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// func TestDeviceOnboardingManagerNzt(t *testing.T) {
// 	mockOnboardingClient := new(MockOnboardingClient)
// 	mockOnboardingClient.On("ImageDownload", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
// 	mockOnboardingClient.On("DiWorkflowCreation", mock.Anything).Return("mocked-guid", nil)
// 	mockOnboardingClient.On("ProdWorkflowCreation", mock.Anything, mock.Anything).Return(nil)

// 	type args struct {
// 		deviceDetails    utils.DeviceInfo
// 		artifactDetails  utils.ArtifactData
// 		nodeExistCount   *int
// 		totalNodes       int
// 		ErrCh            chan error
// 		ImgDownldLock    *sync.Mutex
// 		focalImgDdLock   *sync.Mutex
// 		jammyImgDdLock   *sync.Mutex
// 		focalMsImgDdLock *sync.Mutex
// 	}
// 	// inputargs := args{
// 	// 	deviceDetails:    utils.DeviceInfo{},
// 	// 	artifactDetails:  utils.ArtifactData{},
// 	// 	nodeExistCount:   new(int),
// 	// 	totalNodes:       1,
// 	// 	ErrCh:            make(chan error),
// 	// 	ImgDownldLock:    new(sync.Mutex),
// 	// 	focalImgDdLock:   new(sync.Mutex),
// 	// 	jammyImgDdLock:   new(sync.Mutex),
// 	// 	focalMsImgDdLock: new(sync.Mutex),
// 	// }
// 	inputargs1 := args{
// 		deviceDetails: utils.DeviceInfo{
// 			ImType: "prod_focal-ms",
// 			// HwIP: "000.00.00.00",
// 		},
// 		artifactDetails:  utils.ArtifactData{},
// 		nodeExistCount:   new(int),
// 		totalNodes:       1,
// 		ErrCh:            make(chan error),
// 		ImgDownldLock:    new(sync.Mutex),
// 		focalImgDdLock:   new(sync.Mutex),
// 		jammyImgDdLock:   new(sync.Mutex),
// 		focalMsImgDdLock: new(sync.Mutex),
// 	}
// 	// enable := true
// 	// enableImgDownload = &enable
// 	// enableDI = &enable
// 	tests := []struct {
// 		name string
// 		args args
// 	}{
// 		// {
// 		// 	name: "Test Case 1",
// 		// 	args: inputargs,
// 		// },
// 		{
// 			name: "Test Case 2",
// 			args: inputargs1,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			DeviceOnboardingManagerNzt(tt.args.deviceDetails, tt.args.artifactDetails, tt.args.nodeExistCount, tt.args.totalNodes, tt.args.ErrCh, tt.args.ImgDownldLock, tt.args.focalImgDdLock, tt.args.jammyImgDdLock, tt.args.focalMsImgDdLock)
// 		})
// 	}
// }

// func TestConvertInstanceForOnboarding(t *testing.T) {
// 	type args struct {
// 		instances   []*computev1.InstanceResource
// 		osinstances []*osv1.OperatingSystemResource
// 		host        *computev1.HostResource
// 	}
// 	os.Setenv("DISABLE_FEATUREX", "true")
// 	instance := &computev1.InstanceResource{}
// 	osInstance := &osv1.OperatingSystemResource{
// 		RepoUrl: "osurl: https://af01p-png.devtools.intel.com/artifactory/hspe-edge-png-local/ubuntu-base/20230911-1844/default/ubuntu-22.04-desktop-amd64+intel-iot-37-custom.img.bz2\noverlayscripturl: https://ubit-artifactory-sh.intel.com/artifactory/sed-dgn-local/yocto/dev-test-image/DKAM/IAAS/ADL/installer23WW44.4_2148.sh\n",
// 	}
// 	host := &computev1.HostResource{}
// 	artifiact := pb.ArtifactData{
// 		Name:     "OS",
// 		Category: pb.ArtifactData_BIOS,
// 	}
// 	artifiact1 := pb.ArtifactData{
// 		Name:     "PLATFORM",
// 		Category: pb.ArtifactData_BIOS,
// 	}
// 	artifactData := []*pb.ArtifactData{&artifiact, &artifiact1}
// 	hw := pb.HwData{
// 		DiskPartition: "123",
// 	}
// 	hwdata := []*pb.HwData{&hw}
// 	req := pb.OnboardingRequest{
// 		ArtifactData: artifactData,
// 		Hwdata:       hwdata,
// 	}
// 	want := []*pb.OnboardingRequest{&req}

// 	tests := []struct {
// 		name    string
// 		args    args
// 		want    []*pb.OnboardingRequest
// 		wantErr bool
// 	}{
// 		{
// 			name: "Test Case 1",
// 			args: args{
// 				instances:   []*computev1.InstanceResource{instance},
// 				osinstances: []*osv1.OperatingSystemResource{osInstance},
// 				host:        host,
// 			},
// 			want:    want,
// 			wantErr: false,
// 		},
// 	}
// 	defer func() {
// 		os.Unsetenv("DISABLE_FEATUREX")
// 	}()
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got, err := ConvertInstanceForOnboarding(tt.args.instances, tt.args.osinstances, tt.args.host)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("ConvertInstanceForOnboarding() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("ConvertInstanceForOnboarding() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

func TestConvertInstanceForOnboarding_Err(t *testing.T) {
	type args struct {
		instances   []*computev1.InstanceResource
		osinstances []*osv1.OperatingSystemResource
		host        *computev1.HostResource
	}
	instance := &computev1.InstanceResource{}
	osInstance := &osv1.OperatingSystemResource{
		RepoUrl: "osUrl;overlayUrl",
	}
	host := &computev1.HostResource{
		HostNics: []*computev1.HostnicResource{
			{
				MacAddr: "00:00:00:00:00:00",
			},
		},
	}
	tests := []struct {
		name    string
		args    args
		want    []*pb.OnboardingRequest
		wantErr bool
	}{
		{
			name: "Test Case 1",
			args: args{
				instances:   []*computev1.InstanceResource{instance},
				osinstances: []*osv1.OperatingSystemResource{osInstance},
				host:        host,
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertInstanceForOnboarding(tt.args.instances, tt.args.osinstances, tt.args.host)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertInstanceForOnboarding() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) {
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
	os.Setenv("PD_IP", "000.000.0.000")
	defer os.Unsetenv("PD_IP")
	os.Setenv("IMAGE_TYPE", "prod_focal-ms")
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
	mockClient := &MockInventoryClient{}
	mockResources := &inv_v1.ListResourcesResponse{}
	mockClient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil)
	_invClient = &invclient.OnboardingInventoryClient{
		Client: mockClient,
	}
	// artifactDatas := []*pb.ArtifactData{
	// 	{
	// 		Category: pb.ArtifactData_OS,
	// 		Name:     "OS",
	// 	},
	// }
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
		// {
		// 	name:   "Test Case 1",
		// 	fields: fields{},
		// 	args: args{
		// 		ctx: context.TODO(),
		// 		req: &pb.OnboardingRequest{
		// 			Hwdata:       hwdatas,
		// 			ArtifactData: artifactDatas,
		// 		},
		// 	},
		// 	want: &pb.OnboardingResponse{
		// 		Status: "Success",
		// 	},
		// 	wantErr: false,
		// },
		{
			name:   "Test Case 2",
			fields: fields{},
			args: args{
				ctx: context.TODO(),
				req: &pb.OnboardingRequest{
					Hwdata:       hwdatas,
					ArtifactData: artifactDatasPlatform,
				},
			},
			want: &pb.OnboardingResponse{
				Status: "Success",
			},
			wantErr: false,
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
}

