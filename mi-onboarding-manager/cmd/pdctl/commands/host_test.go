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
	"sync"
	"testing"

	inv_client "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/client"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/onboarding"
	pbinv "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

func TestHostResourceCmd(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := HostResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"add", "--addr=localhost:50151", "--insecure"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestHostResourceCmd_Get(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := HostResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"get", "--addr=localhost:50251", "--insecure"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestHostResourceCmd_Update(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := HostResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"update", "--addr=localhost:50351", "--insecure", "--hw-id=123"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestHostResourceCmd_Delete(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := HostResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"delete", "--addr=localhost:50451", "--insecure"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestHostResourceCmd_Add(t *testing.T) {
	caCertPath := "/home/" + os.Getenv("USER") + "/.fdo-secrets/scripts/secrets/ca-cert.pem"
	certPath := "/home/" + os.Getenv("USER") + "/.fdo-secrets/scripts/secrets/api-user.pem"
	actual := new(bytes.Buffer)
	RootCmd := HostResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	cert := "--cert=" + certPath
	cacert := "--cacert=" + caCertPath
	RootCmd.SetArgs([]string{"add", "--addr=localhost:50551", cert, "--key=123", cacert})
	err := RootCmd.Execute()
	assert.Error(t, err)
}

func TestHostResourceCmd_Case(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := HostResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"add", "--addr=localhost:50651", "--insecure", "--input_file=artifact_sample.yaml"})
	err := RootCmd.Execute()
	assert.Error(t, err)
}
func TestHostResourceCmd_Case1(t *testing.T) {
	wd, _ := os.Getwd()
	wd, _ = strings.CutSuffix(wd, "/commands")
	wds := wd + "/yaml/artifact_sample.yaml"
	actual := new(bytes.Buffer)
	RootCmd := HostResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"add", "--addr=localhost:50751", "--insecure", "--input_file=" + wds})
	err := RootCmd.Execute()
	assert.Error(t, err)
}
func TestHostResourceCmd_Update_Case1(t *testing.T) {
	actual := new(bytes.Buffer)
	RootCmd := HostResourceCmd()
	RootCmd.SetOut(actual)
	RootCmd.SetErr(actual)
	RootCmd.SetArgs([]string{"update", "--addr=localhost:50351", "--insecure", "--hw-id="})
	err := RootCmd.Execute()
	assert.Error(t, err)
}
func TestNewInventoryClient(t *testing.T) {
	type args struct {
		ctx  context.Context
		wg   *sync.WaitGroup
		addr string
	}
	mockClient := &onboarding.MockInventoryClient{}
	tests := []struct {
		name    string
		args    args
		want    inv_client.InventoryClient
		want1   chan *inv_client.WatchEvents
		wantErr bool
	}{
		{
			name: "Test Case 1",
			args: args{
				ctx: context.Background(),
				wg:  &sync.WaitGroup{},
			},
			want:    mockClient,
			want1:   make(chan *inv_client.WatchEvents),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := NewInventoryClient(tt.args.ctx, tt.args.wg, tt.args.addr)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewInventoryClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewInventoryClient() got = %v, want %v", got, tt.want)
			}
			if reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("NewInventoryClient() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

type mockNodeArtifactServiceNBServer struct {
	pbinv.NodeArtifactServiceNBServer
	mock.Mock
}

func (m *mockNodeArtifactServiceNBServer) CreateArtifacts(ctx context.Context, req *pbinv.ArtifactRequest) (*pbinv.ArtifactResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*pbinv.ArtifactResponse), args.Error(1)
}

func (m *mockNodeArtifactServiceNBServer) GetArtifacts(ctx context.Context, req *pbinv.ArtifactRequest) (*pbinv.ArtifactResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*pbinv.ArtifactResponse), args.Error(1)
}

func (m *mockNodeArtifactServiceNBServer) UpdateArtifactsById(ctx context.Context, req *pbinv.ArtifactRequest) (*pbinv.ArtifactResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*pbinv.ArtifactResponse), args.Error(1)
}

func (m *mockNodeArtifactServiceNBServer) DeleteArtifacts(ctx context.Context, req *pbinv.ArtifactRequest) (*pbinv.ArtifactResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*pbinv.ArtifactResponse), args.Error(1)
}

func (m *mockNodeArtifactServiceNBServer) CreateNodes(ctx context.Context, req *pbinv.NodeRequest) (*pbinv.NodeResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*pbinv.NodeResponse), args.Error(1)
}

func (m *mockNodeArtifactServiceNBServer) GetNodes(ctx context.Context, req *pbinv.NodeRequest) (*pbinv.NodeResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*pbinv.NodeResponse), args.Error(1)
}

func (m *mockNodeArtifactServiceNBServer) UpdateNodes(ctx context.Context, req *pbinv.NodeRequest) (*pbinv.NodeResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*pbinv.NodeResponse), args.Error(1)
}

func (m *mockNodeArtifactServiceNBServer) DeleteNodes(ctx context.Context, req *pbinv.NodeRequest) (*pbinv.NodeResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*pbinv.NodeResponse), args.Error(1)
}
func (m *mockNodeArtifactServiceNBServer) mustEmbedUnimplementedNodeArtifactServiceNBServer() {}

func Test_getNodes(t *testing.T) {
	mockClient := &mockNodeArtifactServiceNBServer{}
	mockClient.On("GetNodes", mock.Anything, mock.Anything).Return(&pbinv.NodeResponse{}, nil)
	lis, err := net.Listen("tcp", "localhost:30051")
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
	conn, err := grpc.Dial("localhost:30051", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial server: %v", err)
	}
	defer conn.Close()

	type args struct {
		ctx  context.Context
		cc   *grpc.ClientConn
		node *pbinv.NodeData
	}
	tests := []struct {
		name    string
		args    args
		want    *nodeData
		wantErr bool
	}{
		{
			name: "test case",
			args: args{
				ctx:  context.Background(),
				cc:   conn,
				node: &pbinv.NodeData{},
			},
			want:    &nodeData{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getNodes(tt.args.ctx, tt.args.cc, tt.args.node)
			if (err != nil) != tt.wantErr {
				t.Errorf("getNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_updateNodes(t *testing.T) {
	mockClient := &mockNodeArtifactServiceNBServer{}
	mockClient.On("UpdateNodes", mock.Anything, mock.Anything).Return(&pbinv.NodeResponse{}, nil)
	lis, err := net.Listen("tcp", "localhost:20051")
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
	conn, err := grpc.Dial("localhost:20051", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial server: %v", err)
	}
	defer conn.Close()
	type args struct {
		ctx  context.Context
		cc   *grpc.ClientConn
		node *pbinv.NodeData
	}
	tests := []struct {
		name    string
		args    args
		want    *nodeData
		wantErr bool
	}{
		{
			name: "test case",
			args: args{
				ctx:  context.Background(),
				cc:   conn,
				node: &pbinv.NodeData{},
			},
			want:    &nodeData{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := updateNodes(tt.args.ctx, tt.args.cc, tt.args.node)
			if (err != nil) != tt.wantErr {
				t.Errorf("updateNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("updateNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_deleteNodes(t *testing.T) {
	mockClient := &mockNodeArtifactServiceNBServer{}
	mockClient.On("DeleteNodes", mock.Anything, mock.Anything).Return(&pbinv.NodeResponse{}, nil)
	lis, err := net.Listen("tcp", "localhost:10051")
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
	conn, err := grpc.Dial("localhost:10051", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial server: %v", err)
	}
	defer conn.Close()
	type args struct {
		ctx  context.Context
		cc   *grpc.ClientConn
		node *pbinv.NodeData
	}
	tests := []struct {
		name    string
		args    args
		want    *nodeData
		wantErr bool
	}{
		{
			name: "test case",
			args: args{
				ctx:  context.Background(),
				cc:   conn,
				node: &pbinv.NodeData{},
			},
			want:    &nodeData{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := deleteNodes(tt.args.ctx, tt.args.cc, tt.args.node)
			if (err != nil) != tt.wantErr {
				t.Errorf("deleteNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("deleteNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_addNodes(t *testing.T) {
	mockClient := &mockNodeArtifactServiceNBServer{}
	mockClient.On("CreateNodes", mock.Anything, mock.Anything).Return(&pbinv.NodeResponse{}, nil)
	lis, err := net.Listen("tcp", "localhost:12051")
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
	conn, err := grpc.Dial("localhost:12051", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial server: %v", err)
	}
	defer conn.Close()
	type args struct {
		ctx  context.Context
		cc   *grpc.ClientConn
		node *pbinv.NodeData
	}
	tests := []struct {
		name    string
		args    args
		want    *nodeData
		wantErr bool
	}{
		{
			name: "test case",
			args: args{
				ctx:  context.Background(),
				cc:   conn,
				node: &pbinv.NodeData{},
			},
			want:    &nodeData{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := addNodes(tt.args.ctx, tt.args.cc, tt.args.node)
			if (err != nil) != tt.wantErr {
				t.Errorf("addNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("addNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}
 
 