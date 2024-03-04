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

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/api"
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
			ParseAndUpdateURL(tt.args.onboardingRequest)
		})
	}
}

func TestClearFileAndWriteHeader(t *testing.T) {
	type args struct {
		filePath string
	}
	tests := []struct {
		name         string
		args         args
		wantErr      bool
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

func TestClearFileAndWriteHeader_case(t *testing.T) {
	type args struct {
		filePath string
	}
	wd, _ := os.Getwd()
	tempFile, err := os.CreateTemp(wd, "common_file_*.go")
	if err != nil {
		t.Errorf("Error creating temporary file: %v", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	fmt.Println(wd)
	tests := []struct {
		name         string
		args         args
		wantErr      bool
		expectedText string
	}{
		{
			name: "Test Case 1",
			args: args{
				filePath: tempFile.Name(),
			},
			wantErr: false,
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
