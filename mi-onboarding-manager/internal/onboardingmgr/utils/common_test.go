/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/onboardingmgr"
)

func TestDeepCopyOnboardingRequest(t *testing.T) {
	type args struct {
		req *pb.OnboardingRequest
	}
	artifactData := pb.ArtifactData{}
	tests := []struct {
		name string
		args args
		want *pb.OnboardingRequest
	}{
		{
			name: "Test Case 1",
			args: args{},
			want: &pb.OnboardingRequest{},
		},
		{
			name: "Test Case 2",
			args: args{
				req: &pb.OnboardingRequest{ArtifactData: []*pb.ArtifactData{&artifactData}},
			},
			want: &pb.OnboardingRequest{ArtifactData: []*pb.ArtifactData{&artifactData}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DeepCopyOnboardingRequest(tt.args.req); reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeepCopyOnboardingRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChangeWorkingDirectory(t *testing.T) {
	type args struct {
		targetDir string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test Case 1",
			args: args{
				targetDir: "",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ChangeWorkingDirectory(tt.args.targetDir); (err != nil) != tt.wantErr {
				t.Errorf("ChangeWorkingDirectory() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMakeHTTPGETRequest(t *testing.T) {
	type args struct {
		hostIP     string
		guidValue  string
		caCertPath string
		certPath   string
	}
	// caCertPath := "/home/" + os.Getenv("USER") + "/.fdo-secrets/scripts/secrets/ca-cert.pem"
	// certPath := "/home/" + os.Getenv("USER") + "/.fdo-secrets/scripts/secrets/api-user.pem"
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "Test Case 1",
			args: args{
				hostIP:     "",
				guidValue:  "",
				caCertPath: "caCertPath",
				certPath:   "certPath",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MakeHTTPGETRequest(tt.args.hostIP, tt.args.guidValue, tt.args.caCertPath, tt.args.certPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("MakeHTTPGETRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MakeHTTPGETRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseAndUpdateUrl(t *testing.T) {
	type args struct {
		onboardingRequest *pb.OnboardingRequest
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test Case 1",
			args: args{
				onboardingRequest: &pb.OnboardingRequest{
					ArtifactData: []*pb.ArtifactData{
						{Category: pb.ArtifactData_OS, Name: "OS"},
					},
				},
			},
		},
		{
			name: "Test Case 2",
			args: args{
				onboardingRequest: &pb.OnboardingRequest{
					ArtifactData: []*pb.ArtifactData{
						{Category: pb.ArtifactData_PLATFORM, Name: "PLATFORM"},
					},
				},
			},
		},
		{
			name: "Test Case 3",
			args: args{
				onboardingRequest: &pb.OnboardingRequest{
					ArtifactData: []*pb.ArtifactData{
						{Category: pb.ArtifactData_BIOS},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ParseAndUpdateUrl(tt.args.onboardingRequest)
		})
	}
}

func TestClearFileAndWriteHeader(t *testing.T) {
	type args struct {
		filePath string
	}
	wd, _ := os.Getwd()
	fmt.Println(wd)
	tests := []struct {
		name    string
		args    args
		wantErr bool
		expectedText string
	}{
		{
			name: "Test Case 1",
			args: args{
				filePath: "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ClearFileAndWriteHeader(tt.args.filePath); (err != nil) != tt.wantErr {
				t.Errorf("ClearFileAndWriteHeader() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

