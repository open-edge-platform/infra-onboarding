// // SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// // SPDX-License-Identifier: LicenseRef-Intel

package artifact

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	onboarding_mocks "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/onboarding/onboardingmocks"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/api"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	providerv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/provider/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/policy/rbac"
	inv_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/testing"
)

const rbacRules = "../../../../rego/authz.rego"

func createIncomingContextWithENJWT(t *testing.T) context.Context {
	t.Helper()
	_, jwtToken, err := inv_testing.CreateENJWT(t)
	require.NoError(t, err)
	return rbac.AddJWTToTheIncomingContext(context.Background(), jwtToken)
}

func TestNewArtifactService(t *testing.T) {
	type args struct {
		invClient  *invclient.OnboardingInventoryClient
		enableAuth bool
		rbac       string
	}
	mockInvClient := &onboarding_mocks.MockInventoryClient{}
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
				enableAuth: true,
				rbac:       "../../../../rego/authz.rego",
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
			got, err := NewArtifactService(tt.args.invClient, tt.args.enableAuth, tt.args.rbac)
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

func TestNodeArtifactService_CreateNodes_Case(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
		enableAuth                               bool
		rbac                                     *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
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
	mockInvClient1 := &onboarding_mocks.MockInventoryClient{}
	mockInvClient1.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource2,
	}, nil)
	mockResources := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource2}},
	}
	mockInvClient1.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil)
	ctx := createIncomingContextWithENJWT(t)
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
			},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: mockRequest,
			},
			want:    &pb.NodeResponse{Payload: payloads},
			wantErr: false,
		},
		{
			name: "NoJWT",
			fields: fields{invClient: &invclient.OnboardingInventoryClient{
				Client: mockInvClient1,
			},
				enableAuth: true,
				rbac:       rbacServer,
			},
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
				authEnabled:                              tt.fields.enableAuth,
				rbac:                                     tt.fields.rbac,
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
		enableAuth                               bool
		rbac                                     *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	mockInvClient1 := &onboarding_mocks.MockInventoryClient{}
	mockInvClient1.On("List", mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.ListResourcesResponse{}, errors.New("err"))
	ctx, cancel := inv_testing.CreateContextWithJWT(t)
	defer cancel()
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name: "Positive",
			fields: fields{invClient: &invclient.OnboardingInventoryClient{Client: mockInvClient1},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
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
		{
			name: "Negative",
			fields: fields{invClient: &invclient.OnboardingInventoryClient{Client: mockInvClient1},
				enableAuth: true,
				rbac:       rbacServer,
			},
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
				authEnabled:                              tt.fields.enableAuth,
				rbac:                                     tt.fields.rbac,
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
		enableAuth                               bool
		rbac                                     *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
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
	mockInvClient1 := &onboarding_mocks.MockInventoryClient{}
	mockInvClient1.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource2,
	}, errors.New("err"))
	mockInvClient1.On("List", mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.ListResourcesResponse{}, errors.New("err"))
	ctx, cancel := inv_testing.CreateContextWithJWT(t)
	defer cancel()
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name: "Negative1",
			fields: fields{invClient: &invclient.OnboardingInventoryClient{Client: mockInvClient1},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: mockRequest,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "NoJWT",
			fields: fields{invClient: &invclient.OnboardingInventoryClient{Client: mockInvClient1},
				enableAuth: true,
				rbac:       rbacServer,
			},
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
				authEnabled:                              tt.fields.enableAuth,
				rbac:                                     tt.fields.rbac,
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
		enableAuth                               bool
		rbac                                     *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
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
	mockInvClient1 := &onboarding_mocks.MockInventoryClient{}
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
	ctx := createIncomingContextWithENJWT(t)
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name: "Positive",
			fields: fields{invClient: &invclient.OnboardingInventoryClient{Client: mockInvClient1},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: mockRequest,
			},
			want:    &pb.NodeResponse{Payload: payloads},
			wantErr: false,
		},
		{
			name: "NoJWT",
			fields: fields{invClient: &invclient.OnboardingInventoryClient{Client: mockInvClient1},
				enableAuth: true,
				rbac:       rbacServer,
			},
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
				authEnabled:                              tt.fields.enableAuth,
				rbac:                                     tt.fields.rbac,
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
		enableAuth                               bool
		rbac                                     *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
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
	mockInvClient1 := &onboarding_mocks.MockInventoryClient{}
	mockInvClient1.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource2,
	}, status.Error(codes.NotFound, "Node not found"))
	mockInvClient1.On("Create", mock.Anything, mock.Anything).Return(&inv_v1.CreateResourceResponse{}, errors.New("err"))
	mockResources := &inv_v1.ListResourcesResponse{}
	mockInvClient1.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources,
		status.Error(codes.NotFound, "Node not found"))
	ctx, cancel := inv_testing.CreateContextWithJWT(t)
	defer cancel()
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name: "Negative",
			fields: fields{invClient: &invclient.OnboardingInventoryClient{Client: mockInvClient1},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: mockRequest,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "NoJWT",
			fields: fields{invClient: &invclient.OnboardingInventoryClient{Client: mockInvClient1},
				enableAuth: true,
				rbac:       rbacServer,
			},
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
				authEnabled:                              tt.fields.enableAuth,
				rbac:                                     tt.fields.rbac,
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
		enableAuth                               bool
		rbac                                     *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
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
	mockclient := &onboarding_mocks.MockInventoryClient{}
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
	ctx := createIncomingContextWithENJWT(t)
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
				invClient:  &invclient.OnboardingInventoryClient{Client: mockclient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: &pb.NodeRequest{Payload: payloads},
			},
			want:    &pb.NodeResponse{Payload: payloads},
			wantErr: false,
		},
		{
			name: "Negative",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{Client: mockclient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: &pb.NodeRequest{Payload: payloads1},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "NoJWT",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{Client: mockclient},
				enableAuth: true,
				rbac:       rbacServer,
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
				authEnabled:                              tt.fields.enableAuth,
				rbac:                                     tt.fields.rbac,
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
		enableAuth                               bool
		rbac                                     *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
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
	mockclient := &onboarding_mocks.MockInventoryClient{}
	mockclient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, errors.New("err"))
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name: "NoJWT",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{Client: mockclient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: context.TODO(),
				req: nil,
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
				authEnabled:                              tt.fields.enableAuth,
				rbac:                                     tt.fields.rbac,
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
		enableAuth                               bool
		rbac                                     *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
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
	mockclient := &onboarding_mocks.MockInventoryClient{}
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
	ctx, cancel := inv_testing.CreateContextWithJWT(t)
	defer cancel()
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
				invClient:  &invclient.OnboardingInventoryClient{Client: mockclient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: &pb.NodeRequest{Payload: payloads},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "NoJWT",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{Client: mockclient},
				enableAuth: true,
				rbac:       rbacServer,
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
				authEnabled:                              tt.fields.enableAuth,
				rbac:                                     tt.fields.rbac,
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
		enableAuth                               bool
		rbac                                     *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
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
	mockclient := &onboarding_mocks.MockInventoryClient{}
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
	ctx := createIncomingContextWithENJWT(t)
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
				invClient:  &invclient.OnboardingInventoryClient{Client: mockclient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: &pb.NodeRequest{Payload: payloads},
			},
			want:    &pb.NodeResponse{Payload: payloads},
			wantErr: false,
		},
		{
			name: "NoJWT",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{Client: mockclient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: context.TODO(),
				req: nil,
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
				authEnabled:                              tt.fields.enableAuth,
				rbac:                                     tt.fields.rbac,
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
		enableAuth                               bool
		rbac                                     *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
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
	mockclient := &onboarding_mocks.MockInventoryClient{}
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
	ctx, cancel := inv_testing.CreateContextWithJWT(t)
	defer cancel()
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
				invClient:  &invclient.OnboardingInventoryClient{Client: mockclient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: &pb.NodeRequest{Payload: payloads},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "NoJWT",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{Client: mockclient},
				enableAuth: true,
				rbac:       rbacServer,
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
				authEnabled:                              tt.fields.enableAuth,
				rbac:                                     tt.fields.rbac,
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
		enableAuth                               bool
		rbac                                     *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	mockResource2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{},
	}
	mockResources := &inv_v1.ListResourcesResponse{}
	mockclient := &onboarding_mocks.MockInventoryClient{}
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
	ctx := createIncomingContextWithENJWT(t)
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
				invClient:  &invclient.OnboardingInventoryClient{Client: mockclient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: &pb.NodeRequest{Payload: payloads},
			},
			want: &pb.NodeResponse{
				Payload: payloads,
			},
			wantErr: false,
		},
		{
			name: "NoJWT",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{Client: mockclient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: context.TODO(),
				req: nil,
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
				authEnabled:                              tt.fields.enableAuth,
				rbac:                                     tt.fields.rbac,
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
		enableAuth                               bool
		rbac                                     *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
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
	mockclient := &onboarding_mocks.MockInventoryClient{}
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
	ctx, cancel := inv_testing.CreateContextWithJWT(t)
	defer cancel()
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
				invClient:  &invclient.OnboardingInventoryClient{Client: mockclient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: &pb.NodeRequest{Payload: payloads},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "NoJWT",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{Client: mockclient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: context.TODO(),
				req: nil,
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
				authEnabled:                              tt.fields.enableAuth,
				rbac:                                     tt.fields.rbac,
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
		enableAuth                               bool
		rbac                                     *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	mockInvClient := &onboarding_mocks.MockInventoryClient{}
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
	ctx := createIncomingContextWithENJWT(t)
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
				invClient:  &invclient.OnboardingInventoryClient{Client: mockInvClient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: mockRequest,
			},
			want: &pb.NodeResponse{
				Payload: payloads,
			},
			wantErr: false,
		},
		{
			name: "NoJWT",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{Client: mockInvClient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: context.TODO(),
				req: nil,
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
				authEnabled:                              tt.fields.enableAuth,
				rbac:                                     tt.fields.rbac,
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
		enableAuth                               bool
		rbac                                     *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	mockInvClient := &onboarding_mocks.MockInventoryClient{}
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
	ctx := createIncomingContextWithENJWT(t)
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
				invClient:  &invclient.OnboardingInventoryClient{Client: mockInvClient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: mockRequest,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "NoJWT",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{Client: mockInvClient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: context.TODO(),
				req: nil,
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
				authEnabled:                              tt.fields.enableAuth,
				rbac:                                     tt.fields.rbac,
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
		enableAuth                               bool
		rbac                                     *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	mockInvClient := &onboarding_mocks.MockInventoryClient{}
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
	ctx, cancel := inv_testing.CreateContextWithJWT(t)
	defer cancel()
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
				invClient:  &invclient.OnboardingInventoryClient{Client: mockInvClient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: mockRequest,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "NoJWT",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{Client: mockInvClient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: context.TODO(),
				req: nil,
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
				authEnabled:                              tt.fields.enableAuth,
				rbac:                                     tt.fields.rbac,
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
		enableAuth                               bool
		rbac                                     *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	hostNic := &computev1.HostnicResource{ResourceId: "hostnic-084d9b08"}
	hostNics := []*computev1.HostnicResource{hostNic}
	mockInvClient := &onboarding_mocks.MockInventoryClient{}
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
	ctx := createIncomingContextWithENJWT(t)
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
				invClient:  &invclient.OnboardingInventoryClient{Client: mockInvClient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: mockRequest,
			},
			want: &pb.NodeResponse{
				Payload: payloads,
			},
			wantErr: false,
		},
		{
			name: "NoJWT",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{Client: mockInvClient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: context.TODO(),
				req: nil,
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
				authEnabled:                              tt.fields.enableAuth,
				rbac:                                     tt.fields.rbac,
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
		enableAuth                               bool
		rbac                                     *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	hostNic := &computev1.HostnicResource{ResourceId: "hostnic-084d9b08"}
	hostNics := []*computev1.HostnicResource{hostNic}
	mockInvClient := &onboarding_mocks.MockInventoryClient{}
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
	ctx := createIncomingContextWithENJWT(t)
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
				invClient:  &invclient.OnboardingInventoryClient{Client: mockInvClient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: mockRequest,
			},
			want: &pb.NodeResponse{
				Payload: payloads,
			},
			wantErr: false,
		},
		{
			name: "NoJWT",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{Client: mockInvClient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: context.TODO(),
				req: nil,
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
				authEnabled:                              tt.fields.enableAuth,
				rbac:                                     tt.fields.rbac,
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
		enableAuth                               bool
		rbac                                     *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	hostNic := &computev1.HostnicResource{ResourceId: "hostnic-084d9b08"}
	hostNics := []*computev1.HostnicResource{hostNic}
	mockInvClient := &onboarding_mocks.MockInventoryClient{}
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
	ctx, cancel := inv_testing.CreateContextWithJWT(t)
	defer cancel()
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
				invClient:  &invclient.OnboardingInventoryClient{Client: mockInvClient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: mockRequest,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "NoJWT",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{Client: mockInvClient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: context.TODO(),
				req: nil,
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
				authEnabled:                              tt.fields.enableAuth,
				rbac:                                     tt.fields.rbac,
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
		enableAuth                               bool
		rbac                                     *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	hostNic := &computev1.HostnicResource{ResourceId: "hostnic-084d9b08"}
	hostNics := []*computev1.HostnicResource{hostNic}
	mockInvClient := &onboarding_mocks.MockInventoryClient{}
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
	ctx, cancel := inv_testing.CreateContextWithJWT(t)
	defer cancel()
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
				invClient:  &invclient.OnboardingInventoryClient{Client: mockInvClient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: mockRequest,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "NoJWT",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{Client: mockInvClient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: context.TODO(),
				req: nil,
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
				authEnabled:                              tt.fields.enableAuth,
				rbac:                                     tt.fields.rbac,
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
		enableAuth                               bool
		rbac                                     *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	hostNic := &computev1.HostnicResource{ResourceId: "hostnic-084d9b08"}
	hostNics := []*computev1.HostnicResource{hostNic}
	mockInvClient := &onboarding_mocks.MockInventoryClient{}
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
	ctx, cancel := inv_testing.CreateContextWithJWT(t)
	defer cancel()
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
				invClient:  &invclient.OnboardingInventoryClient{Client: mockInvClient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: mockRequest,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "NoJWT",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{Client: mockInvClient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: context.TODO(),
				req: nil,
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
				authEnabled:                              tt.fields.enableAuth,
				rbac:                                     tt.fields.rbac,
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
		enableAuth                               bool
		rbac                                     *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	hostNic := &computev1.HostnicResource{ResourceId: "hostnic-084d9b08", BmcInterface: true}
	hostNics := []*computev1.HostnicResource{hostNic}
	mockInvClient := &onboarding_mocks.MockInventoryClient{}
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
	ctx := createIncomingContextWithENJWT(t)
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
				invClient:  &invclient.OnboardingInventoryClient{Client: mockInvClient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: mockRequest,
			},
			want: &pb.NodeResponse{
				Payload: payloads,
			},
			wantErr: false,
		},
		{
			name: "NoJWT",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{Client: mockInvClient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: context.TODO(),
				req: nil,
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
				authEnabled:                              tt.fields.enableAuth,
				rbac:                                     tt.fields.rbac,
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
	mockInvClient := &onboarding_mocks.MockInventoryClient{}
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
	mockInvClient := &onboarding_mocks.MockInventoryClient{}
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
			}
			if err := s.startZeroTouch(tt.args.ctx, tt.args.hostResID); (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.startZeroTouch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewArtifactService_Case(t *testing.T) {
	type args struct {
		invClient  *invclient.OnboardingInventoryClient
		enableAuth bool
		rbac       string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				invClient: &invclient.OnboardingInventoryClient{
					Client: &onboarding_mocks.MockInventoryClient{},
				},
				enableAuth: true,
				rbac:       "../../../../rego/authz.rego",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewArtifactService(tt.args.invClient, tt.args.enableAuth, tt.args.rbac)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewArtifactService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.NotNil(t, got.rbac)
			assert.NotNil(t, got.invClient)
			assert.True(t, got.authEnabled)
		})
	}
}

func TestNodeArtifactService_CreateNodes_Case5(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
		enableAuth                               bool
		rbac                                     *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
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
	mockInvClient1 := &onboarding_mocks.MockInventoryClient{}
	mockInvClient1.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource2,
	}, status.Error(codes.NotFound, "Node not found"))
	mockInvClient1.On("Create", mock.Anything, mock.Anything).Return(&inv_v1.CreateResourceResponse{
		ResourceId: "host-b8be78c0",
	}, nil).Once()
	mockInvClient1.On("Create", mock.Anything, mock.Anything).Return(&inv_v1.CreateResourceResponse{
		ResourceId: "host-b8be78c0",
	}, errors.New("err")).Once()
	mockResources := &inv_v1.ListResourcesResponse{
		// Resources: nil,
	}
	mockInvClient1.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources,
		status.Error(codes.NotFound, "Node not found"))
	ctx := createIncomingContextWithENJWT(t)
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name: "Positive",
			fields: fields{invClient: &invclient.OnboardingInventoryClient{Client: mockInvClient1},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: mockRequest,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "NoJWT",
			fields: fields{invClient: &invclient.OnboardingInventoryClient{Client: mockInvClient1},
				enableAuth: true,
				rbac:       rbacServer,
			},
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
				authEnabled:                              tt.fields.enableAuth,
				rbac:                                     tt.fields.rbac,
			}
			got, err := s.CreateNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.CreateNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) && !tt.wantErr {
				t.Errorf("NodeArtifactService.CreateNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeArtifactService_UpdateNodes_Case7(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
		enableAuth                               bool
		rbac                                     *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
	type args struct {
		ctx context.Context
		req *pb.NodeRequest
	}
	hostNic := &computev1.HostnicResource{ResourceId: "hostnic-084d9b08", BmcInterface: true}
	hostNics := []*computev1.HostnicResource{hostNic}
	mockInvClient := &onboarding_mocks.MockInventoryClient{}
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
	mockInvClient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil).Once()
	mockInvClient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource,
	}, nil).Once()
	mockInvClient.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil).Once()
	mockInvClient.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, errors.New("err")).Once()
	hwdata := &pb.HwData{Uuid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad"}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	mockRequest := &pb.NodeRequest{
		Payload: payloads,
	}
	ctx := createIncomingContextWithENJWT(t)
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
				invClient:  &invclient.OnboardingInventoryClient{Client: mockInvClient},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: mockRequest,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "NoJWT",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{Client: mockInvClient},
				enableAuth: true,
				rbac:       rbacServer,
			},
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
				authEnabled:                              tt.fields.enableAuth,
				rbac:                                     tt.fields.rbac,
			}
			got, err := s.UpdateNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.UpdateNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) && !tt.wantErr {
				t.Errorf("NodeArtifactService.UpdateNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeArtifactService_startZeroTouch_Case1(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
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
	mockResource1 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Provider{
			Provider: &providerv1.ProviderResource{
				Name: DefaultProviderName,
			},
		},
	}
	mockInvClient := &onboarding_mocks.MockInventoryClient{}
	mockResources1 := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource1}},
	}
	mockInvClient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{Resource: mockResource}, nil).Once()
	mockInvClient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources1, nil).Once()
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
			}
			if err := s.startZeroTouch(tt.args.ctx, tt.args.hostResID); (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.startZeroTouch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeArtifactService_startZeroTouch_Case2(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
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
	mockResource1 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Provider{
			Provider: &providerv1.ProviderResource{
				Config: "config",
				Name:   DefaultProviderName,
			},
		},
	}
	mockInvClient := &onboarding_mocks.MockInventoryClient{}
	mockResources1 := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource1}},
	}
	mockInvClient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{Resource: mockResource}, nil).Once()
	mockInvClient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources1, nil).Once()
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
			},
			args: args{
				ctx:       context.Background(),
				hostResID: "host-084d9b08",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				invClient:                                tt.fields.invClient,
			}
			if err := s.startZeroTouch(tt.args.ctx, tt.args.hostResID); (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.startZeroTouch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeArtifactService_startZeroTouch_Case3(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
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
	mockResource1 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Provider{
			Provider: &providerv1.ProviderResource{
				Config: "{\"defaultOs\":\"linux\",\"autoProvision\":true}",
				Name:   DefaultProviderName,
			},
		},
	}
	mockInvClient := &onboarding_mocks.MockInventoryClient{}
	mockResources1 := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource1}},
	}
	mockInvClient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{Resource: mockResource}, nil).Once()
	mockInvClient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources1, nil).Once()
	mockInvClient.On("Create", mock.Anything, mock.Anything).Return(&inv_v1.CreateResourceResponse{}, nil).Once()
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
			}
			if err := s.startZeroTouch(tt.args.ctx, tt.args.hostResID); (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.startZeroTouch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeArtifactService_startZeroTouch_Case4(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
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
	mockResource1 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Provider{
			Provider: &providerv1.ProviderResource{
				Config: "{\"defaultOs\":\"linux\",\"autoProvision\":true}",
				Name:   DefaultProviderName,
			},
		},
	}
	mockInvClient := &onboarding_mocks.MockInventoryClient{}
	mockResources1 := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource1}},
	}
	mockInvClient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{Resource: mockResource}, nil).Once()
	mockInvClient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources1, nil).Once()
	mockInvClient.On("Create", mock.Anything, mock.Anything).Return(&inv_v1.CreateResourceResponse{}, errors.New("err")).Once()
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
			},
			args: args{
				ctx:       context.Background(),
				hostResID: "host-084d9b08",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				invClient:                                tt.fields.invClient,
			}
			if err := s.startZeroTouch(tt.args.ctx, tt.args.hostResID); (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.startZeroTouch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
