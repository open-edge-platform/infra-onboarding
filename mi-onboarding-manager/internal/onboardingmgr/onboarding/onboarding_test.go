/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	"context"
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
	mockClient := &MockInventoryClient{}
	mockResources := &inv_v1.ListResourcesResponse{}
	mockClient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil)
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

