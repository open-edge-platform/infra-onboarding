// // SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// // SPDX-License-Identifier: LicenseRef-Intel

package artifact

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	u_uuid "github.com/google/uuid"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/env"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
	om_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/testing"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/api"
	om_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/status"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/compute/v1"
	providerv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/provider/v1"
	statusv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/status/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/auth"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/flags"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/policy/rbac"
	inv_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/status"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/tenant"
	inv_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/testing"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

const (
	tenant1   = "11111111-1111-1111-1111-111111111111"
	tenant2   = "22222222-2222-2222-2222-222222222222"
	rbacRules = "../../../../rego/authz.rego"
)

var mutex sync.Mutex

/*const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateRandomString(length int) string {
	seed := rand.NewSource(time.Now().UnixNano())
	r := rand.New(seed)
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}*/

func getMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func getFirstNChars(hash string, n int) string {
	if len(hash) < n {
		return hash
	}
	return hash[:n]
}

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
			got, err := CopyNodeReqToNodeData(tt.args.payload, tenant1)
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
	ctx := inv_testing.CreateIncomingContextWithENJWT(t, context.Background())
	ctx = tenant.AddTenantIDToContext(ctx, tenant1)
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
	ctx := inv_testing.CreateIncomingContextWithENJWT(t, context.Background())
	ctx = tenant.AddTenantIDToContext(ctx, tenant1)
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
func TestNodeArtifactService_CreateNodes_Case_Success(t *testing.T) {
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
	ctx := inv_testing.CreateIncomingContextWithENJWT(t, context.Background())
	ctx = tenant.AddTenantIDToContext(ctx, tenant1)
	dao := inv_testing.NewInvResourceDAOOrFail(t)
	host := dao.CreateHost(t, tenant1)
	hwdata1 := &pb.HwData{Uuid: host.GetUuid(), Serialnum: "ABCDEFG"}
	hwdatas1 := []*pb.HwData{hwdata1}
	payload1 := pb.NodeData{Hwdata: hwdatas1}
	payloads1 := []*pb.NodeData{&payload1}
	mockRequest1 := &pb.NodeRequest{
		Payload: payloads1,
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.NodeResponse
		wantErr bool
	}{
		{
			name: "Success Test case for CreateNodes serial number miss match",
			fields: fields{
				invClient:  om_testing.InvClient,
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: mockRequest1,
			},
			want:    &pb.NodeResponse{Payload: payloads1},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
	ctx := inv_testing.CreateIncomingContextWithENJWT(t, context.Background())
	ctx = tenant.AddTenantIDToContext(ctx, tenant1)
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
		{
			name: "Invalid Ctx ",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{},
				enableAuth: false,
				rbac:       rbacServer,
			},
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
				req: &pb.NodeRequest{},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
	ctx := inv_testing.CreateIncomingContextWithENJWT(t, context.Background())
	ctx = tenant.AddTenantIDToContext(ctx, tenant1)
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
	ctx := inv_testing.CreateIncomingContextWithENJWT(t, context.Background())
	ctx = tenant.AddTenantIDToContext(ctx, tenant1)
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
	ctx = tenant.AddTenantIDToContext(ctx, tenant1)

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
		{
			name: "Invalid Ctx",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{},
				enableAuth: false,
				rbac:       rbacServer,
			},
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
	ctx = tenant.AddTenantIDToContext(ctx, tenant1)
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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

func TestNodeArtifactService_GetNodes_MultiTenant(t *testing.T) {

	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)

	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})

	dao := inv_testing.NewInvResourceDAOOrFail(t)

	host1 := dao.CreateHost(t, tenant1)
	hwdata1 := &pb.HwData{
		Uuid: host1.Uuid,
	}
	hwdatas1 := []*pb.HwData{hwdata1}
	payload1 := pb.NodeData{Hwdata: hwdatas1}
	payloads1 := []*pb.NodeData{&payload1}
	mockRequest1 := &pb.NodeRequest{
		Payload: payloads1,
	}

	host2 := dao.CreateHost(t, tenant2)
	hwdata2 := &pb.HwData{
		Uuid: host2.Uuid,
	}
	hwdatas2 := []*pb.HwData{hwdata2}
	payload2 := pb.NodeData{Hwdata: hwdatas2}
	payloads2 := []*pb.NodeData{&payload2}
	mockRequest2 := &pb.NodeRequest{
		Payload: payloads2,
	}

	ctx1 := inv_testing.CreateIncomingContextWithENJWT(t, context.Background())
	ctx1 = tenant.AddTenantIDToContext(ctx1, tenant1)

	ctx2 := inv_testing.CreateIncomingContextWithENJWT(t, context.Background())
	ctx2 = tenant.AddTenantIDToContext(ctx2, tenant2)

	s := &NodeArtifactService{
		InventoryClientService: InventoryClientService{
			invClient: om_testing.InvClient,
		},
		authEnabled: true,
		rbac:        rbacServer,
	}

	t.Run("GetNode_Valid_Tenant1", func(t *testing.T) {
		nodes, err := s.GetNodes(ctx1, mockRequest1)
		require.NoError(t, err)
		require.NotNil(t, nodes)
	})

	t.Run("GetNode_Invalid_Tenant1", func(t *testing.T) {
		nodes, _ := s.GetNodes(ctx2, mockRequest1)
		require.Nil(t, nodes)
	})

	t.Run("GetNode_Valid_Tenant2", func(t *testing.T) {
		nodes, err := s.GetNodes(ctx2, mockRequest2)
		require.NoError(t, err)
		require.NotNil(t, nodes)
	})

	t.Run("GetNode_Invalid_Tenant2", func(t *testing.T) {
		nodes, _ := s.GetNodes(ctx1, mockRequest2)
		require.Nil(t, nodes)
	})
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
	dao := inv_testing.NewInvResourceDAOOrFail(t)
	host := dao.CreateHost(t, tenant1)
	hwdata := &pb.HwData{
		Uuid: host.Uuid,
	}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	mockRequest := &pb.NodeRequest{
		Payload: payloads,
	}
	ctx := inv_testing.CreateIncomingContextWithENJWT(t, context.Background())
	ctx = tenant.AddTenantIDToContext(ctx, tenant1)
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
	dao := inv_testing.NewInvResourceDAOOrFail(t)
	host := dao.CreateHost(t, tenant1)
	hwdata := &pb.HwData{
		Uuid: host.Uuid,
	}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	mockRequest := &pb.NodeRequest{
		Payload: payloads,
	}

	ctx := inv_testing.CreateIncomingContextWithENJWT(t, context.Background())
	ctx = tenant.AddTenantIDToContext(ctx, tenant1)
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
	dao := inv_testing.NewInvResourceDAOOrFail(t)
	host := dao.CreateHost(t, tenant1)
	hwdata := &pb.HwData{
		Uuid: host.Uuid,
	}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	mockRequest := &pb.NodeRequest{
		Payload: payloads,
	}
	ctx := inv_testing.CreateIncomingContextWithENJWT(t, context.Background())
	ctx = tenant.AddTenantIDToContext(ctx, tenant1)
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
		ctx          context.Context
		hostTenantID string
		hostResID    string
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	host := inv_testing.CreateHost(t, nil, nil)
	os := inv_testing.CreateOs(t)
	inv_testing.CreateInstance(t, host, os)
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
				ctx:          context.Background(),
				hostTenantID: host.TenantId,
				hostResID:    host.ResourceId,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				InventoryClientService: InventoryClientService{
					invClient:    tt.fields.invClient,
					invClientAPI: tt.fields.invClientAPI,
				},
			}
			if err := s.startZeroTouch(tt.args.ctx, tt.args.hostTenantID, tt.args.hostResID); (err != nil) != tt.wantErr {
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
		ctx          context.Context
		hostTenantID string
		hostResID    string
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
				ctx:          context.Background(),
				hostTenantID: "123",
				hostResID:    "host-084d9b08",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				InventoryClientService: InventoryClientService{
					invClient:    tt.fields.invClient,
					invClientAPI: tt.fields.invClientAPI,
				},
			}
			if err := s.startZeroTouch(tt.args.ctx, tt.args.hostTenantID, tt.args.hostResID); (err != nil) != tt.wantErr {
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
	ctx := inv_testing.CreateIncomingContextWithENJWT(t, context.Background())
	ctx = tenant.AddTenantIDToContext(ctx, tenant1)
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
	dao := inv_testing.NewInvResourceDAOOrFail(t)
	host := dao.CreateHost(t, tenant1)
	hwdata := &pb.HwData{
		Uuid: host.Uuid,
	}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	mockRequest := &pb.NodeRequest{
		Payload: payloads,
	}
	ctx := inv_testing.CreateIncomingContextWithENJWT(t, context.Background())
	ctx = tenant.AddTenantIDToContext(ctx, tenant1)
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
		ctx          context.Context
		hostTenantID string
		hostResID    string
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	host := inv_testing.CreateHost(t, nil, nil)
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
				ctx:          context.Background(),
				hostTenantID: host.TenantId,
				hostResID:    host.ResourceId,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
			}
			if err := s.startZeroTouch(tt.args.ctx, tt.args.hostTenantID, tt.args.hostResID); (err != nil) != tt.wantErr {
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
		ctx          context.Context
		hostTenantID string
		hostResID    string
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	host := inv_testing.CreateHost(t, nil, nil)
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
				ctx:          context.Background(),
				hostTenantID: host.TenantId,
				hostResID:    host.ResourceId,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				InventoryClientService: InventoryClientService{
					invClient:    tt.fields.invClient,
					invClientAPI: tt.fields.invClientAPI,
				},
			}
			if err := s.startZeroTouch(tt.args.ctx, tt.args.hostTenantID, tt.args.hostResID); (err != nil) != tt.wantErr {
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
		ctx          context.Context
		hostTenantID string
		hostResID    string
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	host := inv_testing.CreateHost(t, nil, nil)
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
				ctx:          context.Background(),
				hostTenantID: host.TenantId,
				hostResID:    host.ResourceId,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				InventoryClientService: InventoryClientService{
					invClient:    tt.fields.invClient,
					invClientAPI: tt.fields.invClientAPI,
				},
			}
			if err := s.startZeroTouch(tt.args.ctx, tt.args.hostTenantID, tt.args.hostResID); (err != nil) != tt.wantErr {
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
		ctx          context.Context
		hostTenantID string
		hostResID    string
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	host := inv_testing.CreateHost(t, nil, nil)
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
				ctx:          context.Background(),
				hostTenantID: host.TenantId,
				hostResID:    host.ResourceId,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NodeArtifactService{
				UnimplementedNodeArtifactServiceNBServer: tt.fields.UnimplementedNodeArtifactServiceNBServer,
				InventoryClientService: InventoryClientService{
					invClient:    tt.fields.invClient,
					invClientAPI: tt.fields.invClientAPI,
				},
			}
			if err := s.startZeroTouch(tt.args.ctx, tt.args.hostTenantID, tt.args.hostResID); (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.startZeroTouch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeArtifactService_startZeroTouch_MultiTenant(t *testing.T) {

	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})

	dao := inv_testing.NewInvResourceDAOOrFail(t)

	// Create two providers with the same name but with different tenants
	dao.CreateProvider(t, tenant1, utils.DefaultProviderName,
		inv_testing.ProviderKind(providerv1.ProviderKind_PROVIDER_KIND_BAREMETAL))
	dao.CreateProvider(t, tenant2, utils.DefaultProviderName,
		inv_testing.ProviderKind(providerv1.ProviderKind_PROVIDER_KIND_BAREMETAL))

	host := dao.CreateHost(t, tenant1)

	s := &NodeArtifactService{
		InventoryClientService: InventoryClientService{
			invClient:    om_testing.InvClient,
			invClientAPI: om_testing.InvClient,
		},
	}

	t.Run("Start ZeroTouch with multi-tenant provider creation", func(t *testing.T) {
		err := s.startZeroTouch(context.Background(), host.TenantId, host.ResourceId)
		require.NoError(t, err)
	})
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
	ctx = tenant.AddTenantIDToContext(ctx, tenant1)
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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

	ctx := inv_testing.CreateIncomingContextWithENJWT(t, context.Background())
	ctx = tenant.AddTenantIDToContext(ctx, tenant1)
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
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
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
		ctx      context.Context
		pconf    invclient.ProviderConfig
		tenentID string
		host     *computev1.HostResource
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
				host:     &computev1.HostResource{},
				tenentID: "",
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
				InventoryClientService: InventoryClientService{
					invClient:    tt.fields.invClient,
					invClientAPI: tt.fields.invClientAPI,
				},
				rbac:        tt.fields.rbac,
				authEnabled: tt.fields.authEnabled,
			}
			if err := s.checkNCreateInstance(tt.args.ctx, tt.args.tenentID, tt.args.pconf, tt.args.host); (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.checkNCreateInstance() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type MockNonInteractiveOnboardingService_OnboardNodeStreamServer struct {
	mock.Mock
}

func (m *MockNonInteractiveOnboardingService_OnboardNodeStreamServer) Send(response *pb.OnboardStreamResponse) error {
	args := m.Called(response)
	return args.Error(0)
}

func (m *MockNonInteractiveOnboardingService_OnboardNodeStreamServer) Recv() (*pb.OnboardStreamRequest, error) {
	args := m.Called()
	return args.Get(0).(*pb.OnboardStreamRequest), args.Error(1)
}

func (m *MockNonInteractiveOnboardingService_OnboardNodeStreamServer) SetHeader(md metadata.MD) error {
	args := m.Called(md)
	return args.Error(0)
}

func (m *MockNonInteractiveOnboardingService_OnboardNodeStreamServer) SendHeader(md metadata.MD) error {
	args := m.Called(md)
	return args.Error(0)
}

func (m *MockNonInteractiveOnboardingService_OnboardNodeStreamServer) SetTrailer(md metadata.MD) {
	m.Called(md)
}

func (m *MockNonInteractiveOnboardingService_OnboardNodeStreamServer) Context() context.Context {
	args := m.Called()
	return args.Get(0).(context.Context)
}

func (m *MockNonInteractiveOnboardingService_OnboardNodeStreamServer) SendMsg(msg interface{}) error {
	args := m.Called(msg)
	return args.Error(0)
}

func (m *MockNonInteractiveOnboardingService_OnboardNodeStreamServer) RecvMsg(msg interface{}) error {
	args := m.Called(msg)
	return args.Error(0)
}

func Test_sendStreamErrorResponse(t *testing.T) {
	type args struct {
		stream  pb.NonInteractiveOnboardingService_OnboardNodeStreamServer
		code    codes.Code
		message string
	}
	var art MockNonInteractiveOnboardingService_OnboardNodeStreamServer
	art.On("Send", mock.Anything).Return(errors.New("err"))
	var art1 MockNonInteractiveOnboardingService_OnboardNodeStreamServer
	art1.On("Send", mock.Anything).Return(nil)
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "negative",
			args: args{
				stream:  &art,
				code:    codes.InvalidArgument,
				message: "error",
			},
			wantErr: true,
		},
		{
			name: "positive",
			args: args{
				stream: &art1,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := sendStreamErrorResponse(tt.args.stream, tt.args.code, tt.args.message); (err != nil) != tt.wantErr {
				t.Errorf("sendStreamErrorResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeArtifactService_handleRegisteredState(t *testing.T) {
	type fields struct {
		UnimplementedNonInteractiveOnboardingServiceServer pb.UnimplementedNonInteractiveOnboardingServiceServer
		invClient                                          *invclient.OnboardingInventoryClient
		invClientAPI                                       *invclient.OnboardingInventoryClient
		rbac                                               *rbac.Policy
		authEnabled                                        bool
	}
	var art MockNonInteractiveOnboardingService_OnboardNodeStreamServer
	art.On("Send", mock.Anything).Return(errors.New("err"))
	var art1 MockNonInteractiveOnboardingService_OnboardNodeStreamServer
	art1.On("Send", mock.Anything).Return(nil)
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	host := inv_testing.CreateHost(t, nil, nil)
	type args struct {
		stream  pb.NonInteractiveOnboardingService_OnboardNodeStreamServer
		hostInv *computev1.HostResource
		req     *pb.OnboardStreamRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "negative",
			fields: fields{
				UnimplementedNonInteractiveOnboardingServiceServer: pb.UnimplementedNonInteractiveOnboardingServiceServer{},
				invClient:    &invclient.OnboardingInventoryClient{},
				invClientAPI: &invclient.OnboardingInventoryClient{},
				rbac:         &rbac.Policy{},
				authEnabled:  false,
			},
			args: args{
				stream:  &art,
				hostInv: &computev1.HostResource{},
			},
			wantErr: true,
		},
		{
			name: "postive",
			fields: fields{
				UnimplementedNonInteractiveOnboardingServiceServer: pb.UnimplementedNonInteractiveOnboardingServiceServer{},
				invClient:    om_testing.InvClient,
				invClientAPI: om_testing.InvClient,
				rbac:         &rbac.Policy{},
				authEnabled:  false,
			},
			args: args{
				stream:  &art1,
				hostInv: host,
				req: &pb.OnboardStreamRequest{
					Uuid: "f9f8-434a-8620-bbed2a12b0ad",
				},
			},
			wantErr: false,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NonInteractiveOnboardingService{
				UnimplementedNonInteractiveOnboardingServiceServer: tt.fields.UnimplementedNonInteractiveOnboardingServiceServer,
				InventoryClientService: InventoryClientService{
					invClient:    tt.fields.invClient,
					invClientAPI: tt.fields.invClientAPI,
				},
			}
			if err := s.handleRegisteredState(tt.args.stream, tt.args.hostInv, tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.handleRegisteredState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeArtifactService_handleOnboardedState(t *testing.T) {
	type fields struct {
		UnimplementedNonInteractiveOnboardingServiceServer pb.UnimplementedNonInteractiveOnboardingServiceServer
		invClient                                          *invclient.OnboardingInventoryClient
		invClientAPI                                       *invclient.OnboardingInventoryClient
		rbac                                               *rbac.Policy
		authEnabled                                        bool
	}
	var art MockNonInteractiveOnboardingService_OnboardNodeStreamServer
	art.On("Send", mock.Anything).Return(errors.New("err"))
	var art1 MockNonInteractiveOnboardingService_OnboardNodeStreamServer
	art1.On("Send", mock.Anything).Return(nil)
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	type args struct {
		stream  pb.NonInteractiveOnboardingService_OnboardNodeStreamServer
		hostInv *computev1.HostResource
		req     *pb.OnboardStreamRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "negative",
			fields: fields{
				UnimplementedNonInteractiveOnboardingServiceServer: pb.UnimplementedNonInteractiveOnboardingServiceServer{},
				invClient:    &invclient.OnboardingInventoryClient{},
				invClientAPI: &invclient.OnboardingInventoryClient{},
				rbac:         &rbac.Policy{},
				authEnabled:  false,
			},
			args: args{
				stream:  &art,
				hostInv: &computev1.HostResource{},
				req: &pb.OnboardStreamRequest{
					Uuid: "f9f8-434a-8620-bbed2a12b0ad",
				},
			},
			wantErr: true,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NonInteractiveOnboardingService{
				UnimplementedNonInteractiveOnboardingServiceServer: tt.fields.UnimplementedNonInteractiveOnboardingServiceServer,
			}
			if err := s.handleOnboardedState(tt.args.stream, tt.args.hostInv, tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.handleOnboardedState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeArtifactService_handleDefaultState(t *testing.T) {
	var art MockNonInteractiveOnboardingService_OnboardNodeStreamServer
	art.On("Send", mock.Anything).Return(errors.New("err"))
	type fields struct {
		UnimplementedNonInteractiveOnboardingServiceServer pb.UnimplementedNonInteractiveOnboardingServiceServer
		invClient                                          *invclient.OnboardingInventoryClient
		invClientAPI                                       *invclient.OnboardingInventoryClient
	}
	type args struct {
		stream pb.NonInteractiveOnboardingService_OnboardNodeStreamServer
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "HandleDefaultState_WithSendError",
			fields: fields{
				UnimplementedNonInteractiveOnboardingServiceServer: pb.UnimplementedNonInteractiveOnboardingServiceServer{},
				invClient:    &invclient.OnboardingInventoryClient{},
				invClientAPI: &invclient.OnboardingInventoryClient{},
			},
			args: args{
				stream: &art,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NonInteractiveOnboardingService{
				UnimplementedNonInteractiveOnboardingServiceServer: tt.fields.UnimplementedNonInteractiveOnboardingServiceServer,
				InventoryClientService: InventoryClientService{
					invClient:    tt.fields.invClient,
					invClientAPI: tt.fields.invClientAPI,
				},
			}
			if err := s.handleDefaultState(tt.args.stream); (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.handleDefaultState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
func TestNewNonInteractiveOnboardingService(t *testing.T) {
	type args struct {
		invClient     *invclient.OnboardingInventoryClient
		inventoryAdr  string
		enableTracing bool
	}
	tests := []struct {
		name    string
		args    args
		want    *NonInteractiveOnboardingService
		wantErr bool
	}{
		{
			name: "NewNonInteractiveOnboardingService Test Case",
			args: args{
				invClient:     &invclient.OnboardingInventoryClient{},
				inventoryAdr:  "",
				enableTracing: false,
			},
			want:    &NonInteractiveOnboardingService{},
			wantErr: false,
		},
		{
			name: "Nil inventory client",
			args: args{
				invClient:     nil,
				inventoryAdr:  "",
				enableTracing: false,
			},
			want:    &NonInteractiveOnboardingService{},
			wantErr: true,
		},
		{
			name: "NewNonInteractiveOnboardingService Test Case with dummy inventory address",
			args: args{
				invClient:     &invclient.OnboardingInventoryClient{},
				inventoryAdr:  "123",
				enableTracing: false,
			},
			want:    &NonInteractiveOnboardingService{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewNonInteractiveOnboardingService(tt.args.invClient, tt.args.inventoryAdr, tt.args.enableTracing)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewNonInteractiveOnboardingService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewNonInteractiveOnboardingService() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestNodeArtifactServiceOnboardNodeStream(t *testing.T) {
	type fields struct {
		UnimplementedNonInteractiveOnboardingServiceServer pb.UnimplementedNonInteractiveOnboardingServiceServer
		invClient                                          *invclient.OnboardingInventoryClient
		invClientAPI                                       *invclient.OnboardingInventoryClient
	}
	type args struct {
		stream pb.NonInteractiveOnboardingService_OnboardNodeStreamServer
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	host := inv_testing.CreateHost(t, nil, nil)
	os.Setenv("ONBOARDING_MANAGER_CLIENT_NAME", "env")
	os.Setenv("ONBOARDING_CREDENTIALS_SECRET_NAME", "env")
	var art MockNonInteractiveOnboardingService_OnboardNodeStreamServer
	art.On("Recv").Return(&pb.OnboardStreamRequest{}, errors.New("err"))
	var art1 MockNonInteractiveOnboardingService_OnboardNodeStreamServer
	currAuthServiceFactory := auth.AuthServiceFactory
	currFlagDisableCredentialsManagement := *flags.FlagDisableCredentialsManagement
	defer func() {
		auth.AuthServiceFactory = currAuthServiceFactory
		*flags.FlagDisableCredentialsManagement = currFlagDisableCredentialsManagement
	}()
	*flags.FlagDisableCredentialsManagement = false
	auth.AuthServiceFactory = om_testing.AuthServiceMockFactory(false, false, true)
	art1.On("Send", mock.Anything).Return(nil)
	art1.On("Recv").Return(&pb.OnboardStreamRequest{
		Uuid:      host.Uuid,
		Serialnum: host.SerialNumber,
		MacId:     host.PxeMac,
		HostIp:    host.BmcIp,
	}, nil)
	var art2 MockNonInteractiveOnboardingService_OnboardNodeStreamServer
	art2.On("Send", mock.Anything).Return(nil)
	art2.On("Recv").Return(&pb.OnboardStreamRequest{
		Uuid:      host.Uuid,
		Serialnum: host.SerialNumber,
		MacId:     host.PxeMac,
		HostIp:    host.BmcIp,
	}, nil)
	var art3 MockNonInteractiveOnboardingService_OnboardNodeStreamServer
	art3.On("Send", mock.Anything).Return(nil)
	art3.On("Recv").Return(&pb.OnboardStreamRequest{
		Serialnum: host.SerialNumber,
		Uuid:      host.Uuid,
	}, nil)
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "OnboardNodeStream Negative Test Case",
			fields: fields{
				UnimplementedNonInteractiveOnboardingServiceServer: pb.UnimplementedNonInteractiveOnboardingServiceServer{},
				invClient:    &invclient.OnboardingInventoryClient{},
				invClientAPI: &invclient.OnboardingInventoryClient{},
			},
			args: args{
				stream: &art,
			},
			wantErr: true,
		},
		{
			name: "OnboardNodeStream Positive Test Case",
			fields: fields{
				UnimplementedNonInteractiveOnboardingServiceServer: pb.UnimplementedNonInteractiveOnboardingServiceServer{},
				invClient:    om_testing.InvClient,
				invClientAPI: &invclient.OnboardingInventoryClient{},
			},
			args: args{
				stream: &art1,
			},
			wantErr: false,
		},
		{
			name: "OnboardNodeStream current state as onboarded",
			fields: fields{
				UnimplementedNonInteractiveOnboardingServiceServer: pb.UnimplementedNonInteractiveOnboardingServiceServer{},
				invClient:    om_testing.InvClient,
				invClientAPI: &invclient.OnboardingInventoryClient{},
			},
			args: args{
				stream: &art2,
			},
			wantErr: false,
		},
		{
			name: "OnboardNodeStream with uuid and serial number",
			fields: fields{
				UnimplementedNonInteractiveOnboardingServiceServer: pb.UnimplementedNonInteractiveOnboardingServiceServer{},
				invClient:    om_testing.InvClient,
				invClientAPI: &invclient.OnboardingInventoryClient{},
			},
			args: args{
				stream: &art3,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NonInteractiveOnboardingService{
				UnimplementedNonInteractiveOnboardingServiceServer: tt.fields.UnimplementedNonInteractiveOnboardingServiceServer,
				InventoryClientService: InventoryClientService{
					invClient:    tt.fields.invClient,
					invClientAPI: tt.fields.invClientAPI,
				},
			}
			if err := s.OnboardNodeStream(tt.args.stream); (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.OnboardNodeStream() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	defer func() {
		os.Unsetenv("ONBOARDING_MANAGER_CLIENT_NAME")
		os.Unsetenv("ONBOARDING_CREDENTIALS_SECRET_NAME")
	}()
}

func TestNodeArtifactService_getHostResource(t *testing.T) {
	type fields struct {
		UnimplementedNonInteractiveOnboardingServiceServer pb.UnimplementedNonInteractiveOnboardingServiceServer
		invClient                                          *invclient.OnboardingInventoryClient
		invClientAPI                                       *invclient.OnboardingInventoryClient
	}
	type args struct {
		req *pb.OnboardStreamRequest
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	host := inv_testing.CreateHost(t, nil, nil)
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *computev1.HostResource
		wantErr bool
	}{
		{
			name: "getHostResource test case with uuid",
			fields: fields{
				UnimplementedNonInteractiveOnboardingServiceServer: pb.UnimplementedNonInteractiveOnboardingServiceServer{},
				invClient:    om_testing.InvClient,
				invClientAPI: &invclient.OnboardingInventoryClient{},
			},
			args: args{
				req: &pb.OnboardStreamRequest{
					Uuid: "f9f8-434a-8620-bbed2a12b0ad",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "getHostResource test case with serial number",
			fields: fields{
				UnimplementedNonInteractiveOnboardingServiceServer: pb.UnimplementedNonInteractiveOnboardingServiceServer{},
				invClient:    om_testing.InvClient,
				invClientAPI: &invclient.OnboardingInventoryClient{},
			},
			args: args{
				req: &pb.OnboardStreamRequest{
					Serialnum: "12345",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "getHostResource test case with serial number and host uuid",
			fields: fields{
				UnimplementedNonInteractiveOnboardingServiceServer: pb.UnimplementedNonInteractiveOnboardingServiceServer{},
				invClient:    om_testing.InvClient,
				invClientAPI: &invclient.OnboardingInventoryClient{},
			},
			args: args{
				req: &pb.OnboardStreamRequest{
					Uuid:      host.Uuid,
					Serialnum: "12345",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "getHostResource test case with empty request",
			fields: fields{
				UnimplementedNonInteractiveOnboardingServiceServer: pb.UnimplementedNonInteractiveOnboardingServiceServer{},
				invClient:    om_testing.InvClient,
				invClientAPI: &invclient.OnboardingInventoryClient{},
			},
			args: args{
				req: &pb.OnboardStreamRequest{},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NonInteractiveOnboardingService{
				UnimplementedNonInteractiveOnboardingServiceServer: tt.fields.UnimplementedNonInteractiveOnboardingServiceServer,
				InventoryClientService: InventoryClientService{
					invClient:    tt.fields.invClient,
					invClientAPI: tt.fields.invClientAPI,
				},
			}
			s.getHostResource(tt.args.req)
		})
	}
}

func TestNodeArtifactService_getHostResourcetest(t *testing.T) {
	type fields struct {
		UnimplementedNonInteractiveOnboardingServiceServer pb.UnimplementedNonInteractiveOnboardingServiceServer
		invClient                                          *invclient.OnboardingInventoryClient
		invClientAPI                                       *invclient.OnboardingInventoryClient
	}
	type args struct {
		req *pb.OnboardStreamRequest
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	// Create a host for testing
	host1 := inv_testing.CreateHostWithArgs(t, "host-1", "44414747-3031-3052-b030-453347474122", "", "", nil, nil, true)
	host2 := inv_testing.CreateHostWithArgs(t, "host-2", "", "ABCDEFG", "", nil, nil, true)
	host3 := inv_testing.CreateHostWithArgs(t, "host-3", "44414747-3031-3052-b030-453347474166", "ABCDEHI", "", nil, nil, true)
	//host4 := inv_testing.CreateHostWithArgs(t, "host-4", "", "", "", nil, nil, true)

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *computev1.HostResource
		wantErr bool
	}{
		{
			name: "getHostResource test case with uuid",
			fields: fields{
				UnimplementedNonInteractiveOnboardingServiceServer: pb.UnimplementedNonInteractiveOnboardingServiceServer{},
				invClient:    om_testing.InvClient,
				invClientAPI: &invclient.OnboardingInventoryClient{},
			},
			args: args{
				req: &pb.OnboardStreamRequest{
					Uuid: "44414747-3031-3052-b030-453347474122",
				},
			},
			want:    host1,
			wantErr: false,
		},
		{
			name: "getHostResource test case with serial number",
			fields: fields{
				UnimplementedNonInteractiveOnboardingServiceServer: pb.UnimplementedNonInteractiveOnboardingServiceServer{},
				invClient:    om_testing.InvClient,
				invClientAPI: &invclient.OnboardingInventoryClient{},
			},
			args: args{
				req: &pb.OnboardStreamRequest{
					Serialnum: "ABCDEFG",
				},
			},
			want:    host2,
			wantErr: false,
		},
		{
			name: "getHostResource test case with serial number and host uuid",
			fields: fields{
				UnimplementedNonInteractiveOnboardingServiceServer: pb.UnimplementedNonInteractiveOnboardingServiceServer{},
				invClient:    om_testing.InvClient,
				invClientAPI: &invclient.OnboardingInventoryClient{},
			},
			args: args{
				req: &pb.OnboardStreamRequest{
					Uuid:      "44414747-3031-3052-b030-453347474166",
					Serialnum: "ABCDEHI",
				},
			},
			want:    host3,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &NonInteractiveOnboardingService{
				UnimplementedNonInteractiveOnboardingServiceServer: tt.fields.UnimplementedNonInteractiveOnboardingServiceServer,
				InventoryClientService: InventoryClientService{
					invClient:    tt.fields.invClient,
					invClientAPI: tt.fields.invClientAPI,
				},
			}
			got, err := s.getHostResource(tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeArtifactService.getHostResource() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if got != tt.want {
					t.Logf("Expected: %+v\n", tt.want)
					t.Logf("Got: %+v\n", got)
				}
			} else {
				if got != nil {
					t.Logf("Expected result to be nil when there is an error, but got: %+v\n", got)
				}
			}
		})
	}
}

func TestHostRegistrationSerialNumFailedWithDetails(t *testing.T) {
	type args struct {
		detail string
	}
	tests := []struct {
		name string
		args args
		want inv_status.ResourceStatus
	}{
		{
			name: "Test Case",
			args: args{},
			want: inv_status.ResourceStatus{
				Status:          "Host Registration Failed due to mismatch of Serial Number, Reported Serial Number is: ",
				StatusIndicator: statusv1.StatusIndication_STATUS_INDICATION_ERROR,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := om_status.HostRegistrationSerialNumFailedWithDetails(tt.args.detail); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HostRegistrationSerialNumFailedWithDetails() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMustEnsureRequired(t *testing.T) {
	os.Setenv("TINKER_VERSION", "value")
	defer os.Unsetenv("TINKER_VERSION")
	tests := []struct {
		name string
	}{
		{
			name: "Positive Test Case",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env.MustEnsureRequired()
		})
	}
}

// FUZZ test cases
func FuzzCreateNodes(f *testing.F) {
	om_testing.CreateInventoryOnboardingClientForTesting()
	f.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	mutex.Lock()
	dao := inv_testing.NewInvResourceDAOOrFail(f)
	host := dao.CreateHostNoCleanup(f, tenant1)

	f.Add("node1", "platform1", "9fa0a788-f9f8-434a-8620-bbed2a12b0ad")
	mutex.Unlock()
	f.Fuzz(func(t *testing.T, hwId string, platformType string, uuid string) {
		mutex.Lock()
		defer mutex.Unlock()
		if hwId == "" || platformType == "" || uuid == "" {
			t.Skip("Skipping test because resourceID is empty")
			return
		}

		ctx := inv_testing.CreateIncomingContextWithENJWT(t, context.Background())
		ctx = tenant.AddTenantIDToContext(ctx, tenant1)

		rbacServer, err := rbac.New(rbacRules)
		if err != nil {
			t.Errorf("Error at the RBAC rules %v", err)
		}

		s := &NodeArtifactService{
			UnimplementedNodeArtifactServiceNBServer: pb.UnimplementedNodeArtifactServiceNBServer{},
			InventoryClientService: InventoryClientService{
				invClient:    om_testing.InvClient,
				invClientAPI: om_testing.InvClient,
			},
			rbac:        rbacServer,
			authEnabled: true,
		}

		hwdata := &pb.HwData{
			HwId:         getFirstNChars(getMD5Hash(hwId), 6),
			PlatformType: getFirstNChars(getMD5Hash(platformType), 10),
			Uuid:         host.GetUuid(),
		}
		hwdatas := []*pb.HwData{hwdata}
		payload1 := pb.NodeData{Hwdata: hwdatas}
		payloads := []*pb.NodeData{&payload1}
		mockRequest := &pb.NodeRequest{
			Payload: payloads,
		}

		_, err = s.CreateNodes(ctx, mockRequest)
		if err != nil {
			t.Errorf("CreateNodes returned an error: %v", err)
		}

	})

}

func FuzzOnboardNodeStream(f *testing.F) {

	f.Add("hostip")
	f.Fuzz(func(t *testing.T, ip string) {
		resp := &pb.OnboardStreamRequest{
			Uuid:      u_uuid.New().String(),
			Serialnum: getFirstNChars(getMD5Hash(ip), 8),
			MacId:     "",
			HostIp:    getFirstNChars(getMD5Hash(ip), 6),
		}

		var art MockNonInteractiveOnboardingService_OnboardNodeStreamServer
		art.On("Send", mock.Anything).Return(nil)
		art.On("Recv").Return(resp, nil)

		service := &NonInteractiveOnboardingService{
			UnimplementedNonInteractiveOnboardingServiceServer: pb.UnimplementedNonInteractiveOnboardingServiceServer{},
			InventoryClientService: InventoryClientService{
				invClient:    &invclient.OnboardingInventoryClient{},
				invClientAPI: &invclient.OnboardingInventoryClient{},
			},
		}

		err := service.OnboardNodeStream(&art)
		if err != nil {
			t.Errorf("Error at OnboardNodeStram %v", err)
		}
	})
}
