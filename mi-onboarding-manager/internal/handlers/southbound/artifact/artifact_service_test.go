// // SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// // SPDX-License-Identifier: LicenseRef-Intel

package artifact

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/onboarding"
	repository "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/persistence"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/api"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewArtifactService(t *testing.T) {
	type args struct {
		invClient *invclient.OnboardingInventoryClient
	}
	mockInvClient := &onboarding.MockInventoryClient{}
	tests := []struct {
		name    string
		args    args
		want    *NodeArtifactService
		wantErr bool
	}{
		{
			name: "Test Case 1",
			args: args{
				invClient: &invclient.OnboardingInventoryClient{
					Client: mockInvClient,
				},
			},
			want:    &NodeArtifactService{},
			wantErr: false,
		},
		{
			name: "Test Case 2",
			args: args{
				invClient: nil,
			},
			want:    &NodeArtifactService{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewArtifactService(tt.args.invClient, "", false)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewArtifactService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewArtifactService() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopyNodeReqtoNodetData(t *testing.T) {
	type args struct {
		payload []*pb.NodeData
	}
	payload := []*pb.NodeData{}
	payloads := []*pb.NodeData{
		{
			Hwdata: []*pb.HwData{
				{
					MacId:          "mac1",
					SutIp:          "192.168.1.1",
					Uuid:           "uuid1",
					Serialnum:      "serial1",
					BmcIp:          "10.0.0.1",
					HostNicDevName: "eth0",
					BmcInterface:   true,
				},
			},
		},
	}
	tests := []struct {
		name    string
		args    args
		want    []*computev1.HostResource
		wantErr bool
	}{
		{
			name:    "Empty Payload",
			args:    args{payload: payload},
			want:    nil,
			wantErr: false,
		},
		{
			name:    "Non-empty Payload",
			args:    args{payload: payloads},
			want:    []*computev1.HostResource{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CopyNodeReqToNodeData(tt.args.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("CopyNodeReqToNodeData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopyNodeReqToNodeData() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopyNodeDatatoNodeResp(t *testing.T) {
	type args struct {
		payload []repository.NodeData
		result  string
	}
	tests := []struct {
		name    string
		args    args
		want    []*pb.NodeData
		wantErr bool
	}{
		{
			name: "Test with SUCCESS result",
			args: args{
				payload: []repository.NodeData{
					{
						ID: "1", HwID: "hw1", FwArtID: "fw1", OsArtID: "os1", AppArtID: "app1", PlatformArtID: "plat1",
						PlatformType: "type1", DeviceType: "device1", DeviceInfoAgent: "agent1", DeviceStatus: "status1",
					},
					{
						ID: "2", HwID: "hw2", FwArtID: "fw2", OsArtID: "os2", AppArtID: "app2", PlatformArtID: "plat2",
						PlatformType: "type2", DeviceType: "device2", DeviceInfoAgent: "agent2", DeviceStatus: "status2",
					},
				},
				result: "SUCCESS",
			},
			want: []*pb.NodeData{
				{
					NodeId: "1", HwId: "hw1", FwArtifactId: "fw1", OsArtifactId: "os1", AppArtifactId: "app1",
					PlatArtifactId: "plat1", PlatformType: "type1", DeviceType: "device1", DeviceInfoAgent: "agent1",
					DeviceStatus: "status1", Result: 0,
				},
				{
					NodeId: "2", HwId: "hw2", FwArtifactId: "fw2", OsArtifactId: "os2", AppArtifactId: "app2",
					PlatArtifactId: "plat2", PlatformType: "type2", DeviceType: "device2", DeviceInfoAgent: "agent2",
					DeviceStatus: "status2", Result: 0,
				},
			},
			wantErr: false,
		},
		{
			name: "Test with FAILURE result",
			args: args{
				payload: []repository.NodeData{
					{
						ID: "3", HwID: "hw3", FwArtID: "fw3", OsArtID: "os3", AppArtID: "app3", PlatformArtID: "plat3",
						PlatformType: "type3", DeviceType: "device3", DeviceInfoAgent: "agent3", DeviceStatus: "status3",
					},
					{
						ID: "4", HwID: "hw4", FwArtID: "fw4", OsArtID: "os4", AppArtID: "app4", PlatformArtID: "plat4",
						PlatformType: "type4", DeviceType: "device4", DeviceInfoAgent: "agent4", DeviceStatus: "status4",
					},
				},
				result: "FAILURE",
			},
			want: []*pb.NodeData{
				{
					NodeId: "3", HwId: "hw3", FwArtifactId: "fw3", OsArtifactId: "os3", AppArtifactId: "app3",
					PlatArtifactId: "plat3", PlatformType: "type3", DeviceType: "device3",
					DeviceInfoAgent: "agent3", DeviceStatus: "status3", Result: 1,
				},
				{
					NodeId: "4", HwId: "hw4", FwArtifactId: "fw4", OsArtifactId: "os4", AppArtifactId: "app4",
					PlatArtifactId: "plat4", PlatformType: "type4", DeviceType: "device4",
					DeviceInfoAgent: "agent4", DeviceStatus: "status4", Result: 1,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CopyNodeDatatoNodeResp(tt.args.payload, tt.args.result)
			if (err != nil) != tt.wantErr {
				t.Errorf("CopyNodeDatatoNodeResp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopyNodeDatatoNodeResp() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeArtifactService_CreateNodes_Case(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	hwdata := &pb.HwData{Uuid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad"}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	mockRequest := &pb.NodeRequest{
		Payload: payloads,
	}
	host := &computev1.HostResource{
		ResourceId: "host-084d9b08",
		Uuid:       "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
	}
	mockResource2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: host,
		},
	}
	mockInvClient1 := &onboarding.MockInventoryClient{}
	mockInvClient1.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource2,
	}, nil)
	mockResources := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource2}},
	}
	mockInvClient1.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil)
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name: "Negative1",
			fields: fields{invClient: &invclient.OnboardingInventoryClient{
				Client: mockInvClient1,
			}},
			args: args{
				ctx: context.TODO(),
				req: mockRequest,
			},
			want:    &pb.NodeResponse{Payload: payloads},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				invClient:                                tt.fields.invClient,
			}
			got, err := s.CreateNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.CreateNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeArtifactService.CreateNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeArtifactService_CreateNodes_Case1(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	mockInvClient1 := &onboarding.MockInventoryClient{}
	mockInvClient1.On("List", mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.ListResourcesResponse{}, errors.New("err"))
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name:   "Positive",
			fields: fields{invClient: &invclient.OnboardingInventoryClient{Client: mockInvClient1}},
			args: args{
				ctx: context.TODO(),
				req: &pb.NodeRequest{
					Payload: []*pb.NodeData{
						{
							Hwdata: []*pb.HwData{
								{
									BmcIp:          "123",
									Serialnum:      "123",
									Uuid:           "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
									MacId:          "00.00.00.00",
									HostNicDevName: "abc",
									BmcInterface:   true,
								},
							},
						},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				invClient:                                tt.fields.invClient,
			}
			got, err := s.CreateNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.CreateNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeArtifactService.CreateNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeArtifactService_CreateNodes_Case2(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	hwdata := &pb.HwData{Uuid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad"}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	mockRequest := &pb.NodeRequest{
		Payload: payloads,
	}
	host := &computev1.HostResource{
		ResourceId: "host-084d9b08",
		Uuid:       "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
	}
	mockResource2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: host,
		},
	}
	mockInvClient1 := &onboarding.MockInventoryClient{}
	mockInvClient1.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource2,
	}, errors.New("err"))
	mockInvClient1.On("List", mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.ListResourcesResponse{}, errors.New("err"))
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name:   "Negative1",
			fields: fields{invClient: &invclient.OnboardingInventoryClient{Client: mockInvClient1}},
			args: args{
				ctx: context.TODO(),
				req: mockRequest,
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				invClient:                                tt.fields.invClient,
			}
			got, err := s.CreateNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.CreateNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeArtifactService.CreateNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeArtifactService_CreateNodes_Case3(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	hwdata := &pb.HwData{Uuid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad"}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	mockRequest := &pb.NodeRequest{
		Payload: payloads,
	}
	mockResource2 := &inv_v1.Resource{}
	mockInvClient1 := &onboarding.MockInventoryClient{}
	mockInvClient1.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource2,
	}, status.Error(codes.NotFound, "Node not found"))
	mockInvClient1.On("Create", mock.Anything, mock.Anything).Return(&inv_v1.CreateResourceResponse{
		ResourceId: "host-b8be78c0",
	}, nil).Once()
	mockInvClient1.On("Create", mock.Anything, mock.Anything).Return(&inv_v1.CreateResourceResponse{
		ResourceId: "host-b8be78c0",
	}, nil).Once()
	mockResources := &inv_v1.ListResourcesResponse{
		// Resources: nil,
	}
	mockInvClient1.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources,
		status.Error(codes.NotFound, "Node not found"))
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name:   "Positive",
			fields: fields{invClient: &invclient.OnboardingInventoryClient{Client: mockInvClient1}},
			args: args{
				ctx: context.TODO(),
				req: mockRequest,
			},
			want:    &pb.NodeResponse{Payload: payloads},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				invClient:                                tt.fields.invClient,
			}
			got, err := s.CreateNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.CreateNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeArtifactService.CreateNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeArtifactService_CreateNodes_Case4(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	hwdata := &pb.HwData{Uuid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad"}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	mockRequest := &pb.NodeRequest{
		Payload: payloads,
	}
	mockResource2 := &inv_v1.Resource{}
	mockInvClient1 := &onboarding.MockInventoryClient{}
	mockInvClient1.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource2,
	}, status.Error(codes.NotFound, "Node not found"))
	mockInvClient1.On("Create", mock.Anything, mock.Anything).Return(&inv_v1.CreateResourceResponse{}, errors.New("err"))
	mockResources := &inv_v1.ListResourcesResponse{}
	mockInvClient1.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources,
		status.Error(codes.NotFound, "Node not found"))
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name:   "Negative",
			fields: fields{invClient: &invclient.OnboardingInventoryClient{Client: mockInvClient1}},
			args: args{
				ctx: context.TODO(),
				req: mockRequest,
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				invClient:                                tt.fields.invClient,
			}
			got, err := s.CreateNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.CreateNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeArtifactService.CreateNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeArtifactService_DeleteNodes_Case1(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	host := &computev1.HostResource{
		ResourceId: "host-084d9b08",
		Uuid:       "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
	}
	mockResource2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: host,
		},
	}
	mockclient := new(onboarding.MockInventoryClient)
	mockResources := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource2}},
	}
	mockclient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil)
	mockclient.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	mockclient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource2,
	}, nil)
	hwdata := &pb.HwData{Uuid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad"}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	hwdata1 := &pb.HwData{Uuid: ""}
	hwdatas1 := []*pb.HwData{hwdata1}
	payload1 := pb.NodeData{Hwdata: hwdatas1}
	payloads1 := []*pb.NodeData{&payload1}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name: "Positive",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{Client: mockclient},
			},
			args: args{
				ctx: context.TODO(),
				req: &pb.NodeRequest{Payload: payloads},
			},
			want:    &pb.NodeResponse{Payload: payloads},
			wantErr: false,
		},
		{
			name: "Negative",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{Client: mockclient},
			},
			args: args{
				ctx: context.TODO(),
				req: &pb.NodeRequest{Payload: payloads1},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				invClient:                                tt.fields.invClient,
			}
			got, err := s.DeleteNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.DeleteNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeArtifactService.DeleteNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeArtifactService_DeleteNodes_Case2(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	host := &computev1.HostResource{
		ResourceId: "host-084d9b08",
		Uuid:       "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
	}
	mockResource2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: host,
		},
	}
	mockResources := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource2}},
	}
	mockclient := new(onboarding.MockInventoryClient)
	mockclient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, errors.New("err"))
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				invClient:                                tt.fields.invClient,
			}
			got, err := s.DeleteNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.DeleteNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeArtifactService.DeleteNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeArtifactService_DeleteNodes_Case3(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	host := &computev1.HostResource{
		ResourceId: "host-084d9b08",
		Uuid:       "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
	}
	mockResource2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: host,
		},
	}
	mockResources := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource2}},
	}
	mockclient := new(onboarding.MockInventoryClient)
	mockclient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil)
	mockclient.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, errors.New("err"))
	mockclient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource2,
	}, nil)
	hwdata := &pb.HwData{Uuid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad"}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name: "Positive",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{Client: mockclient},
			},
			args: args{
				ctx: context.TODO(),
				req: &pb.NodeRequest{Payload: payloads},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				invClient:                                tt.fields.invClient,
			}
			got, err := s.DeleteNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.DeleteNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeArtifactService.DeleteNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeArtifactService_DeleteNodes_Case4(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	host := &computev1.HostResource{
		ResourceId: "host-084d9b08",
		Uuid:       "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
	}
	mockResource2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: host,
		},
	}
	mockResources := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource2}},
	}
	mockclient := new(onboarding.MockInventoryClient)
	mockclient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil)
	mockclient.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	mockclient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource2,
	}, errors.New("err"))
	hwdata := &pb.HwData{Uuid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad"}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name: "Positive",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{Client: mockclient},
			},
			args: args{
				ctx: context.TODO(),
				req: &pb.NodeRequest{Payload: payloads},
			},
			want:    &pb.NodeResponse{Payload: payloads},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				invClient:                                tt.fields.invClient,
			}
			got, err := s.DeleteNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.DeleteNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeArtifactService.DeleteNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeArtifactService_DeleteNodes_Case5(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	mockResource2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{},
	}
	mockResources := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource2}},
	}
	mockclient := new(onboarding.MockInventoryClient)
	mockclient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil)
	mockclient.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	mockclient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource2,
	}, errors.New("err"))
	hwdata := &pb.HwData{Uuid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad"}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name: "Negative",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{Client: mockclient},
			},
			args: args{
				ctx: context.TODO(),
				req: &pb.NodeRequest{Payload: payloads},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				invClient:                                tt.fields.invClient,
			}
			got, err := s.DeleteNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.DeleteNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeArtifactService.DeleteNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeArtifactService_DeleteNodes_Case6(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	mockResource2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{},
	}
	mockResources := &inv_v1.ListResourcesResponse{}
	mockclient := new(onboarding.MockInventoryClient)
	mockclient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil)
	mockclient.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	mockclient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource2,
	}, nil)
	hwdata := &pb.HwData{Uuid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad"}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name: "Negative",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{Client: mockclient},
			},
			args: args{
				ctx: context.TODO(),
				req: &pb.NodeRequest{Payload: payloads},
			},
			want: &pb.NodeResponse{
				Payload: payloads,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				invClient:                                tt.fields.invClient,
			}
			got, err := s.DeleteNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.DeleteNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeArtifactService.DeleteNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeArtifactService_DeleteNodes_Case7(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	host := &computev1.HostResource{
		ResourceId: "host-084d9b08",
		Uuid:       "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
	}
	mockResource2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: host,
		},
	}
	mockResources := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource2}, {Resource: mockResource2}},
	}
	mockclient := new(onboarding.MockInventoryClient)
	mockclient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil)
	mockclient.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	mockclient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource2,
	}, errors.New("err"))
	hwdata := &pb.HwData{Uuid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad"}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name: "Negative",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{Client: mockclient},
			},
			args: args{
				ctx: context.TODO(),
				req: &pb.NodeRequest{Payload: payloads},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				invClient:                                tt.fields.invClient,
			}
			got, err := s.DeleteNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.DeleteNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeArtifactService.DeleteNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeArtifactService_GetNodes_Case1(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	mockInvClient := &onboarding.MockInventoryClient{}
	host := &computev1.HostResource{
		ResourceId: "host-084d9b08",
		Uuid:       "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
	}
	mockResource2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: host,
		},
	}
	mockResources := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource2}},
	}
	mockInvClient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource2,
	}, nil)
	mockInvClient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil)

	hwdata := &pb.HwData{Uuid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad"}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	mockRequest := &pb.NodeRequest{
		Payload: payloads,
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name: "Positive",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{Client: mockInvClient},
			},
			args: args{
				ctx: context.Background(),
				req: mockRequest,
			},
			want: &pb.NodeResponse{
				Payload: payloads,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				invClient:                                tt.fields.invClient,
			}
			got, err := s.GetNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.GetNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeArtifactService.GetNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeArtifactService_GetNodes_Case2(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	mockInvClient := &onboarding.MockInventoryClient{}
	host := &computev1.HostResource{
		ResourceId: "host-084d9b08",
		Uuid:       "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
	}
	mockResource2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: host,
		},
	}
	mockResources := &inv_v1.ListResourcesResponse{}
	mockInvClient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource2,
	}, status.Error(codes.NotFound, "Node not found"))
	mockInvClient.On("List", mock.Anything, mock.Anything,
		mock.Anything).Return(mockResources, status.Error(codes.NotFound, "Node not found"))

	hwdata := &pb.HwData{Uuid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad"}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	mockRequest := &pb.NodeRequest{
		Payload: payloads,
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name: "Positive",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{Client: mockInvClient},
			},
			args: args{
				ctx: context.Background(),
				req: mockRequest,
			},
			want:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				invClient:                                tt.fields.invClient,
			}
			got, err := s.GetNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.GetNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeArtifactService.GetNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeArtifactService_GetNodes_Case3(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	mockInvClient := &onboarding.MockInventoryClient{}
	host := &computev1.HostResource{
		ResourceId: "host-084d9b08",
		Uuid:       "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
	}
	mockResource2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: host,
		},
	}
	mockResources := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource2}},
	}
	mockInvClient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource2,
	}, status.Error(codes.NotFound, "Node not found"))
	mockInvClient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, errors.New("err"))

	hwdata := &pb.HwData{Uuid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad"}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	mockRequest := &pb.NodeRequest{
		Payload: payloads,
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name: "Negative",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{Client: mockInvClient},
			},
			args: args{
				ctx: context.Background(),
				req: mockRequest,
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				invClient:                                tt.fields.invClient,
			}
			got, err := s.GetNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.GetNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeArtifactService.GetNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeArtifactService_UpdateNodes_Case1(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	hostNic := &computev1.HostnicResource{ResourceId: "hostnic-084d9b08"}
	hostNics := []*computev1.HostnicResource{hostNic}
	mockInvClient := &onboarding.MockInventoryClient{}
	host := &computev1.HostResource{
		ResourceId: "host-084d9b08",
		Uuid:       "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
		HostNics:   hostNics,
	}
	mockResource := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: host,
		},
	}
	mockResources := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource}},
	}
	mockInvClient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil)
	mockInvClient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource,
	}, nil)
	mockInvClient.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	hwdata := &pb.HwData{Uuid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad"}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	mockRequest := &pb.NodeRequest{
		Payload: payloads,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name: "Test case 1",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{Client: mockInvClient},
			},
			args: args{
				ctx: context.Background(),
				req: mockRequest,
			},
			want: &pb.NodeResponse{
				Payload: payloads,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				invClient:                                tt.fields.invClient,
			}
			got, err := s.UpdateNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.UpdateNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeArtifactService.UpdateNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeArtifactService_UpdateNodes_Case2(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	hostNic := &computev1.HostnicResource{ResourceId: "hostnic-084d9b08"}
	hostNics := []*computev1.HostnicResource{hostNic}
	mockInvClient := &onboarding.MockInventoryClient{}
	host := &computev1.HostResource{
		ResourceId: "host-084d9b08",
		Uuid:       "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
		HostNics:   hostNics,
	}
	mockResource := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: host,
		},
	}
	mockResources := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource}},
	}
	mockInvClient.On("List", mock.Anything, mock.Anything,
		mock.Anything).Return(mockResources, status.Error(codes.NotFound, "Node not found"))
	mockInvClient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource,
	}, status.Error(codes.NotFound, "Node not found"))
	mockInvClient.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	hwdata := &pb.HwData{Uuid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad"}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	mockRequest := &pb.NodeRequest{
		Payload: payloads,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name: "Test case 1",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{Client: mockInvClient},
			},
			args: args{
				ctx: context.Background(),
				req: mockRequest,
			},
			want: &pb.NodeResponse{
				Payload: payloads,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				invClient:                                tt.fields.invClient,
			}
			got, err := s.UpdateNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.UpdateNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeArtifactService.UpdateNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeArtifactService_UpdateNodes_Case3(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	hostNic := &computev1.HostnicResource{ResourceId: "hostnic-084d9b08"}
	hostNics := []*computev1.HostnicResource{hostNic}
	mockInvClient := &onboarding.MockInventoryClient{}
	host := &computev1.HostResource{
		ResourceId: "host-084d9b08",
		Uuid:       "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
		HostNics:   hostNics,
	}
	mockResource := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: host,
		},
	}
	mockResources := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource}},
	}
	mockInvClient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, errors.New("err"))
	mockInvClient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource,
	}, status.Error(codes.NotFound, "Node not found"))
	mockInvClient.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	hwdata := &pb.HwData{Uuid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad"}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	mockRequest := &pb.NodeRequest{
		Payload: payloads,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name: "Test case 1",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{Client: mockInvClient},
			},
			args: args{
				ctx: context.Background(),
				req: mockRequest,
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				invClient:                                tt.fields.invClient,
			}
			got, err := s.UpdateNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.UpdateNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeArtifactService.UpdateNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeArtifactService_UpdateNodes_Case4(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	hostNic := &computev1.HostnicResource{ResourceId: "hostnic-084d9b08"}
	hostNics := []*computev1.HostnicResource{hostNic}
	mockInvClient := &onboarding.MockInventoryClient{}
	host := &computev1.HostResource{
		ResourceId: "host-084d9b08",
		Uuid:       "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
		HostNics:   hostNics,
	}
	mockResource := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: host,
		},
	}
	mockResources := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource}},
	}
	mockInvClient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil)
	mockInvClient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource,
	}, nil)
	mockInvClient.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, errors.New("err"))
	hwdata := &pb.HwData{Uuid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad"}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	mockRequest := &pb.NodeRequest{
		Payload: payloads,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name: "Test case 1",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{Client: mockInvClient},
			},
			args: args{
				ctx: context.Background(),
				req: mockRequest,
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				invClient:                                tt.fields.invClient,
			}
			got, err := s.UpdateNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.UpdateNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeArtifactService.UpdateNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeArtifactService_UpdateNodes_Case5(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	hostNic := &computev1.HostnicResource{ResourceId: "hostnic-084d9b08"}
	hostNics := []*computev1.HostnicResource{hostNic}
	mockInvClient := &onboarding.MockInventoryClient{}
	host := &computev1.HostResource{
		ResourceId: "host-084d9b08",
		Uuid:       "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
		HostNics:   hostNics,
	}
	mockResource := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: host,
		},
	}
	mockInvClient.On("List", mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.ListResourcesResponse{}, status.Error(codes.NotFound, "Node not found"))
	mockInvClient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource,
	}, status.Error(codes.NotFound, "Node not found"))
	mockInvClient.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	hwdata := &pb.HwData{Uuid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad"}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	mockRequest := &pb.NodeRequest{
		Payload: payloads,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name: "Test case 1",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{Client: mockInvClient},
			},
			args: args{
				ctx: context.Background(),
				req: mockRequest,
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				invClient:                                tt.fields.invClient,
			}
			got, err := s.UpdateNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.UpdateNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeArtifactService.UpdateNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeArtifactService_UpdateNodes_Case6(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	hostNic := &computev1.HostnicResource{ResourceId: "hostnic-084d9b08", BmcInterface: true}
	hostNics := []*computev1.HostnicResource{hostNic}
	mockInvClient := &onboarding.MockInventoryClient{}
	host := &computev1.HostResource{
		ResourceId: "host-084d9b08",
		Uuid:       "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
		HostNics:   hostNics,
	}
	mockResource := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: host,
		},
	}
	mockResources := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource}},
	}
	mockInvClient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil)
	mockInvClient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource,
	}, nil)
	mockInvClient.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	hwdata := &pb.HwData{Uuid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad"}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	mockRequest := &pb.NodeRequest{
		Payload: payloads,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name: "Test case 1",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{Client: mockInvClient},
			},
			args: args{
				ctx: context.Background(),
				req: mockRequest,
			},
			want: &pb.NodeResponse{
				Payload: payloads,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				invClient:                                tt.fields.invClient,
			}
			got, err := s.UpdateNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.UpdateNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeArtifactService.UpdateNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeArtifactService_startZeroTouch(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
		invClientAPI                             *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx       context.Context
		hostResID string
	}
	mockHost := &computev1.HostResource{
		ResourceId:   "host-084d9b08",
		DesiredState: computev1.HostState_HOST_STATE_DELETED,
		Instance: &computev1.InstanceResource{
			ResourceId: "inst-084d9b08",
		},
	}
	mockResource := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: mockHost,
		},
	}
	mockInvClient := &onboarding.MockInventoryClient{}
	mockResources1 := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource}},
	}
	mockInvClient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{Resource: mockResource}, nil)
	mockInvClient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources1, nil)
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{
					Client: mockInvClient,
				},
				invClientAPI: &invclient.OnboardingInventoryClient{
					Client: &onboarding.MockInventoryClient{},
				},
			},
			args: args{
				ctx:       context.Background(),
				hostResID: "host-084d9b08",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				invClient:                                tt.fields.invClient,
				invClientAPI:                             tt.fields.invClientAPI,
			}
			if err := s.startZeroTouch(tt.args.ctx, tt.args.hostResID); (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.startZeroTouch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeArtifactService_startZeroTouch_Case(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
		invClientAPI                             *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx       context.Context
		hostResID string
	}
	mockHost := &computev1.HostResource{
		ResourceId: "host-084d9b08",
	}
	mockResource := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: mockHost,
		},
	}
	mockInvClient := &onboarding.MockInventoryClient{}
	mockResources1 := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource}},
	}
	mockInvClient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{Resource: mockResource}, nil)
	mockInvClient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources1, nil)
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{
					Client: mockInvClient,
				},
				invClientAPI: &invclient.OnboardingInventoryClient{
					Client: &onboarding.MockInventoryClient{},
				},
			},
			args: args{
				ctx:       context.Background(),
				hostResID: "host-084d9b08",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				invClient:                                tt.fields.invClient,
				invClientAPI:                             tt.fields.invClientAPI,
			}
			if err := s.startZeroTouch(tt.args.ctx, tt.args.hostResID); (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.startZeroTouch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
