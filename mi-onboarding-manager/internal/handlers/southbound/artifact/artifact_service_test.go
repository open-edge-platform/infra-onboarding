// // SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// // SPDX-License-Identifier: LicenseRef-Intel

package artifact

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
	om_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/testing"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/api"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/policy/rbac"
	inv_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/testing"
	"github.com/stretchr/testify/require"
)

const rbacRules = "../../../../rego/authz.rego"

func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(wd))))
	policyPath := projectRoot + "/build"
	migrationsDir := projectRoot + "/build"

	inv_testing.StartTestingEnvironment(policyPath, "", migrationsDir)
	run := m.Run() // run all tests
	inv_testing.StopTestingEnvironment()

	os.Exit(run)
}

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
	tests := []struct {
		name    string
		args    args
		want    *NodeArtifactService
		wantErr bool
	}{
		{
			name: "Valid Arguments with Authorization Enabled",
			args: args{
				invClient:  &invclient.OnboardingInventoryClient{},
				enableAuth: true,
				rbac:       "../../../../rego/authz.rego",
			},
			want:    &NodeArtifactService{},
			wantErr: false,
		},
		{
			name: "Nil Inventory Client Error Handling",
			args: args{
				invClient: nil,
			},
			want:    &NodeArtifactService{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewArtifactService(tt.args.invClient, "", false, tt.args.enableAuth, tt.args.rbac)
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
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	ctx := createIncomingContextWithENJWT(t)
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name: "Success Test case for CreateNodes",
			fields: fields{
				invClient:  om_testing.InvClient,
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
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{},
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
			name: "Positive test case for creating node",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{},
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
			name: "Negative test case for creating node",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{},
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
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{},
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
				invClient:  &invclient.OnboardingInventoryClient{},
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
	ctx := createIncomingContextWithENJWT(t)
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
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
				invClient:  om_testing.InvClient,
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
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{},
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
				invClient:  &invclient.OnboardingInventoryClient{},
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
				invClient:  &invclient.OnboardingInventoryClient{},
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
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
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
				invClient:  om_testing.InvClient,
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
				invClient:  &invclient.OnboardingInventoryClient{},
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
				invClient:  &invclient.OnboardingInventoryClient{},
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
				invClient:  &invclient.OnboardingInventoryClient{},
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
				invClient:  &invclient.OnboardingInventoryClient{},
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
				invClient:  &invclient.OnboardingInventoryClient{},
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
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
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
				invClient:  om_testing.InvClient,
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
				invClient:  &invclient.OnboardingInventoryClient{},
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
				invClient:  &invclient.OnboardingInventoryClient{},
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
				invClient:  &invclient.OnboardingInventoryClient{},
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
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	hwdata := &pb.HwData{Uuid: "1f3cb271-9847-43e1-95b4-adb17570eb7a"}
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
				invClient:  om_testing.InvClient,
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
				invClient:  &invclient.OnboardingInventoryClient{},
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
				invClient:  &invclient.OnboardingInventoryClient{},
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
				invClient:  &invclient.OnboardingInventoryClient{},
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
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})

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
				invClient:  om_testing.InvClient,
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
				invClient:  &invclient.OnboardingInventoryClient{},
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
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})

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
				invClient:  om_testing.InvClient,
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
				invClient:  &invclient.OnboardingInventoryClient{},
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
	hwdata := &pb.HwData{Uuid: "1f3cb271-9847-43e1-95b4-adb17570eb7a"}
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
				invClient:  &invclient.OnboardingInventoryClient{},
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
				invClient:  &invclient.OnboardingInventoryClient{},
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
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	hwdata := &pb.HwData{
		Uuid: host.Uuid,
	}
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
			name: "UpdateNodes with ValidPayload -Success",
			fields: fields{
				invClient:  om_testing.InvClient,
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
			name: "Update Nodes with Missing JWT - Error",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{},
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
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	hwdata := &pb.HwData{
		Uuid: host.Uuid,
	}
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
			name: "UpdateNodes_Success_ValidData",
			fields: fields{
				invClient:  om_testing.InvClient,
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
			name: "UpdateNodes_Error_NoJWT",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{},
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
	hwdata := &pb.HwData{
		Uuid:           "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
		HostNicDevName: "ethx",
	}
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
			name: "UpdateNodes with list resources - Failure ",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{},
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
			name: "UpdateNodes with no JWT provided -Error ",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{},
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
	hwdata := &pb.HwData{
		Uuid:           "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
		HostNicDevName: "ethx",
	}
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
			name: "UpdateNodes_Error_UpdateResourceFailure",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{},
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
			name: "UpdateNodes with no JWT provided -Error",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{},
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
	hwdata := &pb.HwData{
		Uuid:           "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
		HostNicDevName: "ethx",
	}
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
			name: "Update nodes with list resources not found -Error",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{},
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
			name: "UpdateNodes_Error_NoJWTProvided",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{},
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
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	hwdata := &pb.HwData{
		Uuid: host.Uuid,
	}
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
			name: "UpdateNodes with valid payload and BMC interface -Success",
			fields: fields{
				invClient:  om_testing.InvClient,
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
			name: "UpdateNodes with no JWT provided -Error",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{},
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
		invClientAPI                             *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx       context.Context
		hostResID string
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Start zeroTouch host deleted -Success",
			fields: fields{
				invClient:    om_testing.InvClient,
				invClientAPI: om_testing.InvClient,
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
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			fields: fields{
				invClient:    om_testing.InvClient,
				invClientAPI: om_testing.InvClient,
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

func TestNewArtifactService_Case(t *testing.T) {
	type args struct {
		invClient     *invclient.OnboardingInventoryClient
		inventoryAdr  string
		enableTracing bool
		enableAuth    bool
		rbac          string
	}
	tests := []struct {
		name    string
		args    args
		want    *NodeArtifactService
		wantErr bool
	}{
		{
			name: "NewArtifactService_WithInvalidRBACFile",
			args: args{
				invClient:     &invclient.OnboardingInventoryClient{},
				inventoryAdr:  "addr",
				enableTracing: false,
				enableAuth:    true,
				rbac:          "../../../../rego/authz.rego",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewArtifactService(tt.args.invClient, tt.args.inventoryAdr, tt.args.enableTracing, tt.args.enableAuth, tt.args.rbac)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewArtifactService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewArtifactService() = %v, want %v", got, tt.want)
			}
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
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
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
				invClient:  om_testing.InvClient,
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
				invClient:  &invclient.OnboardingInventoryClient{},
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
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	hwdata := &pb.HwData{
		Uuid: host.Uuid,
	}
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
			name: "Update nodes withcJWT -Successful",
			fields: fields{
				invClient:  om_testing.InvClient,
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
				invClient:  &invclient.OnboardingInventoryClient{},
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
		invClientAPI                             *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx       context.Context
		hostResID string
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Start ZeroTouch with provider -Successful",
			fields: fields{
				invClient:    om_testing.InvClient,
				invClientAPI: om_testing.InvClient,
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

func TestNodeArtifactService_startZeroTouch_Case2(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
		invClientAPI                             *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx       context.Context
		hostResID string
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Start ZeroTouch with no provider creation -Successful",
			fields: fields{
				invClient:    om_testing.InvClient,
				invClientAPI: om_testing.InvClient,
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

func TestNodeArtifactService_startZeroTouch_Case3(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
		invClientAPI                             *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx       context.Context
		hostResID string
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Start ZeroTouch with provider creation -Success",
			fields: fields{
				invClient:    om_testing.InvClient,
				invClientAPI: om_testing.InvClient,
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

func TestNodeArtifactService_startZeroTouch_Case4(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
		invClientAPI                             *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx       context.Context
		hostResID string
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	osRes := inv_testing.CreateOs(t)
	inv_testing.CreateInstance(t, host, osRes)
	inv_testing.CreateProvider(t, utils.DefaultProviderName)
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Start ZeroTouch with provider creation -Error",
			fields: fields{
				invClient:    om_testing.InvClient,
				invClientAPI: om_testing.InvClient,
			},
			args: args{
				ctx:       context.Background(),
				hostResID: host.ResourceId,
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

func TestNodeArtifactService_GetNodes_Case4(t *testing.T) {
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
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	hwdata := &pb.HwData{Uuid: "1f3cb271-9847-43e1-95b4-adb17570eb7a"}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	mockRequest := &pb.NodeRequest{
		Payload: payloads,
	}

	hwdata1 := &pb.HwData{Uuid: "9da8a789-f9f8-434a-8720-bbead"}
	hwdatas1 := []*pb.HwData{hwdata1}
	payload1 := pb.NodeData{Hwdata: hwdatas1}
	payloads1 := []*pb.NodeData{&payload1}
	mockRequest1 := &pb.NodeRequest{
		Payload: payloads1,
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
			name: "Host Not found error",
			fields: fields{
				invClient:  om_testing.InvClient,
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
				invClient:  &invclient.OnboardingInventoryClient{},
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
		{
			name: "Failed to create node",
			fields: fields{
				invClient:  om_testing.InvClient,
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: mockRequest1,
			},
			want: &pb.NodeResponse{
				Payload: payloads,
			},
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
			_, err := s.GetNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.GetNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestNodeArtifactService_UpdateNodes_Case8(t *testing.T) {
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
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	hwdata := &pb.HwData{}
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
				invClient:  om_testing.InvClient,
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
			wantErr: true,
		},
		{
			name: "NoJWT",
			fields: fields{
				invClient:  om_testing.InvClient,
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
			_, err := s.UpdateNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.UpdateNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestNodeArtifactService_UpdateNodes_Case9(t *testing.T) {
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
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	hwdata := &pb.HwData{Uuid: "1f3cb271-9847-43e1-95b4-adb17570eb7a"}
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
				invClient:  om_testing.InvClient,
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
			wantErr: true,
		},
		{
			name: "NoJWT",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{},
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
			_, err := s.UpdateNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.UpdateNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestNodeArtifactService_CreateNodes_Case6(t *testing.T) {
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
	hwdata := &pb.HwData{Uuid: "9fa8a788-f9f8-434a-8620-bbed2a12b0a"}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	mockRequest := &pb.NodeRequest{
		Payload: payloads,
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
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
			fields: fields{
				invClient:  om_testing.InvClient,
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: mockRequest,
			},
			want:    &pb.NodeResponse{Payload: payloads},
			wantErr: true,
		},
		{
			name: "NoJWT",
			fields: fields{
				invClient:  om_testing.InvClient,
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
			_, err := s.CreateNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.CreateNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestNodeArtifactService_CreateNodes_Case7(t *testing.T) {
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
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
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
				invClient:  om_testing.InvClient,
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
				invClient:  &invclient.OnboardingInventoryClient{},
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

func TestNodeArtifactService_checkNCreateInstance(t *testing.T) {
	type fields struct {
		UnimplementedNodeArtifactServiceNBServer pb.UnimplementedNodeArtifactServiceNBServer
		invClient                                *invclient.OnboardingInventoryClient
		invClientAPI                             *invclient.OnboardingInventoryClient
		rbac                                     *rbac.Policy
		authEnabled                              bool
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	type args struct {
		ctx   context.Context
		pconf invclient.ProviderConfig
		host  *computev1.HostResource
	}
	ctx := createIncomingContextWithENJWT(t)
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Create Instance Failure",
			fields: fields{
				invClient:    om_testing.InvClient,
				invClientAPI: om_testing.InvClient,
				rbac:         rbacServer,
				authEnabled:  true,
			},
			args: args{
				ctx: ctx,
				pconf: invclient.ProviderConfig{
					AutoProvision: true,
				},
				host: &computev1.HostResource{},
			},
			wantErr: true,
		},
		{
			name: "Create Instance Failure",
			fields: fields{
				invClient:    om_testing.InvClient,
				invClientAPI: om_testing.InvClient,
				rbac:         rbacServer,
				authEnabled:  true,
			},
			args: args{
				ctx: ctx,
				pconf: invclient.ProviderConfig{
					AutoProvision: false,
				},
				host: &computev1.HostResource{},
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
				rbac:                                     tt.fields.rbac,
				authEnabled:                              tt.fields.authEnabled,
			}
			if err := s.checkNCreateInstance(tt.args.ctx, tt.args.pconf, tt.args.host); (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.checkNCreateInstance() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
