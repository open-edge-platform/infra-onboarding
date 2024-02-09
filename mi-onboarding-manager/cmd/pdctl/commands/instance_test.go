/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: LicenseRef-Intel
 */
package commands

import (
	"bytes"
	"context"
	"net"
	"os"
	"reflect"
	"strings"
	"testing"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/onboardingmgr"
	pbinv "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/onboardingmgr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

func TestInstanceResourceCmd_Get(t *testing.T) {

	actual := new(bytes.Buffer)
	RootCmd := InstanceResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"get", "--addr=localhost:51051", "--insecure"})

	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestInstanceResourceCmd_Create(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := InstanceResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"create", "--addr=localhost:52051", "--insecure"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestInstanceResourceCmd_Update(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := InstanceResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"update", "--addr=localhost:53051", "--insecure", "--artifact_id=123"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestInstanceResourceCmd_Delete(t *testing.T) {

	actual := new(bytes.Buffer)
	RootCmd := InstanceResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"delete", "--addr=localhost:54051", "--insecure"})

	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestInstanceResourceCmd_Create_Case(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := InstanceResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"create", "--addr=localhost:55051", "--insecure", "--input_file=artifact_sample.yaml"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestInstanceResourceCmd_Create_Case1(t *testing.T) {
	wd, _ := os.Getwd()
	wd, _ = strings.CutSuffix(wd, "/commands")
	wds := wd + "/yaml/artifact_sample.yaml"
	actual := new(bytes.Buffer)
	RootCmd := InstanceResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"create", "--addr=localhost:56051", "--insecure", "--input_file=" + wds})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestInstanceResourceCmd_Get_Case(t *testing.T) {

	actual := new(bytes.Buffer)
	RootCmd := InstanceResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"get", "--addr=localhost:57051", "--insecure", "--category=6"})

	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestInstanceResourceCmd_Create_Case2(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := InstanceResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"create", "--addr=localhost:58051", "--insecure", "--category=6"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestInstanceResourceCmd_Update_Case(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := InstanceResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"update", "--addr=localhost:59051", "--insecure", "--artifact_id=123", "--category=6"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestInstanceResourceCmd_Update_Case1(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := InstanceResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"update", "--addr=localhost:60051", "--insecure", "--artifact_id="})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestInstanceResourceCmd_Delete_Case(t *testing.T) {

	actual := new(bytes.Buffer)
	RootCmd := InstanceResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"delete", "--addr=localhost:70051", "--insecure", "--category=6"})

	err := RootCmd.Execute()
	assert.Error(t, err)
}

func Test_getArtifacts(t *testing.T) {
	mockClient := &mockNodeArtifactServiceNBServer{}
	mockClient.On("GetArtifacts", mock.Anything, mock.Anything).Return(&pbinv.ArtifactResponse{}, nil)
	lis, err := net.Listen("tcp", "localhost:13051")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pbinv.RegisterNodeArtifactServiceNBServer(grpcServer, mockClient)
	go func() {
		defer lis.Close()
		if err := grpcServer.Serve(lis); err != nil {
			t.Fatalf("Failed to serve: %v", err)
		}
	}()
	conn, err := grpc.Dial("localhost:13051", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial server: %v", err)
	}
	defer conn.Close()
	type args struct {
		ctx      context.Context
		cc       *grpc.ClientConn
		artifact *pb.ArtifactData
	}
	tests := []struct {
		name    string
		args    args
		want    *artifactData
		wantErr bool
	}{
		{
			name: "test case",
			args: args{
				ctx:      context.Background(),
				cc:       conn,
				artifact: &pbinv.ArtifactData{},
			},
			want:    &artifactData{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getArtifacts(tt.args.ctx, tt.args.cc, tt.args.artifact)
			if (err != nil) != tt.wantErr {
				t.Errorf("getArtifacts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getArtifacts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_createArtifacts(t *testing.T) {
	mockClient := &mockNodeArtifactServiceNBServer{}
	mockClient.On("CreateArtifacts", mock.Anything, mock.Anything).Return(&pbinv.ArtifactResponse{}, nil)
	lis, err := net.Listen("tcp", "localhost:14051")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pbinv.RegisterNodeArtifactServiceNBServer(grpcServer, mockClient)
	go func() {
		defer lis.Close()
		if err := grpcServer.Serve(lis); err != nil {
			t.Fatalf("Failed to serve: %v", err)
		}
	}()
	conn, err := grpc.Dial("localhost:14051", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial server: %v", err)
	}
	defer conn.Close()
	type args struct {
		ctx      context.Context
		cc       *grpc.ClientConn
		artifact *pb.ArtifactData
	}
	tests := []struct {
		name    string
		args    args
		want    *artifactData
		wantErr bool
	}{
		{
			name: "test case",
			args: args{
				ctx:      context.Background(),
				cc:       conn,
				artifact: &pbinv.ArtifactData{},
			},
			want:    &artifactData{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createArtifacts(tt.args.ctx, tt.args.cc, tt.args.artifact)
			if (err != nil) != tt.wantErr {
				t.Errorf("createArtifacts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createArtifacts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_updateArtifactsById(t *testing.T) {
	mockClient := &mockNodeArtifactServiceNBServer{}
	mockClient.On("UpdateArtifactsById", mock.Anything, mock.Anything).Return(&pbinv.ArtifactResponse{}, nil)
	lis, err := net.Listen("tcp", "localhost:15051")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pbinv.RegisterNodeArtifactServiceNBServer(grpcServer, mockClient)
	go func() {
		defer lis.Close()
		if err := grpcServer.Serve(lis); err != nil {
			t.Fatalf("Failed to serve: %v", err)
		}
	}()
	conn, err := grpc.Dial("localhost:15051", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial server: %v", err)
	}
	defer conn.Close()
	type args struct {
		ctx        context.Context
		cc         *grpc.ClientConn
		artifactID string
		artifact   *pb.ArtifactData
	}
	tests := []struct {
		name    string
		args    args
		want    *artifactData
		wantErr bool
	}{
		{
			name: "test case",
			args: args{
				ctx:      context.Background(),
				cc:       conn,
				artifact: &pbinv.ArtifactData{},
			},
			want:    &artifactData{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := updateArtifactsByID(tt.args.ctx, tt.args.cc, tt.args.artifactID, tt.args.artifact)
			if (err != nil) != tt.wantErr {
				t.Errorf("updateArtifactsById() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("updateArtifactsById() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_deleteArtifacts(t *testing.T) {
	mockClient := &mockNodeArtifactServiceNBServer{}
	mockClient.On("DeleteArtifacts", mock.Anything, mock.Anything).Return(&pbinv.ArtifactResponse{}, nil)
	lis, err := net.Listen("tcp", "localhost:16051")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pbinv.RegisterNodeArtifactServiceNBServer(grpcServer, mockClient)
	go func() {
		defer lis.Close()
		if err := grpcServer.Serve(lis); err != nil {
			t.Fatalf("Failed to serve: %v", err)
		}
	}()
	conn, err := grpc.Dial("localhost:16051", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial server: %v", err)
	}
	defer conn.Close()
	type args struct {
		ctx        context.Context
		cc         *grpc.ClientConn
		artifact   *pb.ArtifactData
	}
	tests := []struct {
		name    string
		args    args
		want    *artifactData
		wantErr bool
	}{
		{
			name: "test case",
			args: args{
				ctx:      context.Background(),
				cc:       conn,
				artifact: &pbinv.ArtifactData{},
			},
			want:    &artifactData{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := deleteArtifacts(tt.args.ctx, tt.args.cc, tt.args.artifact)
			if (err != nil) != tt.wantErr {
				t.Errorf("deleteArtifacts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("deleteArtifacts() = %v, want %v", got, tt.want)
			}
		})
	}
}
