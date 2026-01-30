// // SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// // SPDX-License-Identifier: Apache-2.0
//
//nolint:testpackage // Keeping the test in the same package due to dependencies on unexported fields.
package grpcserver

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"
	"time"

	u_uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
	providerv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/provider/v1"
	statusv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/status/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/auth"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/flags"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/policy/rbac"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/providerconfiguration"
	inv_status "github.com/open-edge-platform/infra-core/inventory/v2/pkg/status"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/tenant"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/env"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/invclient"
	onboarding_types "github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/onboarding/types"
	om_testing "github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/testing"
	pb "github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/api/onboardingmgr/v1"
	om_status "github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/status"
)

const (
	tenant1   = "11111111-1111-1111-1111-111111111111"
	tenant2   = "22222222-2222-2222-2222-222222222222"
	rbacRules = "../../../../rego/authz.rego"
	sutIP     = "192.168.1.1"
	serialnum = "ABCDEHI"
)

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

type MockNonInteractiveOnboardingServiceOnboardNodeStreamServer struct {
	mock.Mock
}

func (m *MockNonInteractiveOnboardingServiceOnboardNodeStreamServer) Send(response *pb.OnboardNodeStreamResponse) error {
	args := m.Called(response)
	return args.Error(0)
}

func (m *MockNonInteractiveOnboardingServiceOnboardNodeStreamServer) Recv() (*pb.OnboardNodeStreamRequest, error) {
	args := m.Called()
	result, ok := args.Get(0).(*pb.OnboardNodeStreamRequest)
	if !ok {
		return nil, inv_errors.Errorf("unexpected type for *OnboardNodeStreamRequest: %T", args.Get(0))
	}
	return result, args.Error(1)
}

func (m *MockNonInteractiveOnboardingServiceOnboardNodeStreamServer) SetHeader(md metadata.MD) error {
	args := m.Called(md)
	return args.Error(0)
}

func (m *MockNonInteractiveOnboardingServiceOnboardNodeStreamServer) SendHeader(md metadata.MD) error {
	args := m.Called(md)
	return args.Error(0)
}

func (m *MockNonInteractiveOnboardingServiceOnboardNodeStreamServer) SetTrailer(md metadata.MD) {
	m.Called(md)
}

func (m *MockNonInteractiveOnboardingServiceOnboardNodeStreamServer) Context() context.Context {
	args := m.Called()
	result, ok := args.Get(0).(context.Context)
	if !ok {
		return nil
	}
	return result
}

func (m *MockNonInteractiveOnboardingServiceOnboardNodeStreamServer) SendMsg(msg interface{}) error {
	args := m.Called(msg)
	return args.Error(0)
}

func (m *MockNonInteractiveOnboardingServiceOnboardNodeStreamServer) RecvMsg(msg interface{}) error {
	args := m.Called(msg)
	return args.Error(0)
}

func getMD5Hash(text string) string {
	hasher := sha256.New()
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
	policyPath := projectRoot + "/out"
	migrationsDir := projectRoot + "/out"

	inv_testing.StartTestingEnvironment(policyPath, "", migrationsDir)
	run := m.Run() // run all tests
	inv_testing.StopTestingEnvironment()

	os.Exit(run)
}

func createIncomingContextWithENJWT(t *testing.T, tenantID string) context.Context {
	t.Helper()
	_, jwtToken, err := inv_testing.CreateENJWT(t, tenantID)
	require.NoError(t, err)
	return rbac.AddJWTToTheIncomingContext(context.Background(), jwtToken)
}

func TestNewInteractiveOnboardingService(t *testing.T) {
	type args struct {
		invClient  *invclient.OnboardingInventoryClient
		enableAuth bool
		rbac       string
	}
	tests := []struct {
		name    string
		args    args
		want    *InteractiveOnboardingService
		wantErr bool
	}{
		{
			name: "Valid Arguments with Authorization Enabled",
			args: args{
				invClient:  &invclient.OnboardingInventoryClient{},
				enableAuth: true,
				rbac:       "../../../../rego/authz.rego",
			},
			want:    &InteractiveOnboardingService{},
			wantErr: false,
		},
		{
			name: "Nil Inventory Client Error Handling",
			args: args{
				invClient: nil,
			},
			want:    &InteractiveOnboardingService{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewInteractiveOnboardingService(tt.args.invClient, "", false, tt.args.enableAuth, tt.args.rbac)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewInteractiveOnboardingService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewInteractiveOnboardingService() = %v, want %v", got, tt.want)
			}
		})
	}
}

//nolint:funlen // reason: function is long due to necessary test cases.
func TestInteractiveOnboardingService_CreateNodes_Case(t *testing.T) {
	type fields struct {
		UnimplementedInteractiveOnboardingServiceServer pb.UnimplementedInteractiveOnboardingServiceServer
		invClient                                       *invclient.OnboardingInventoryClient
		enableAuth                                      bool
		rbac                                            *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
	type args struct {
		ctx context.Context
		req *pb.CreateNodesRequest
	}

	macID := generateValidMacID()
	serialnum := serialnum

	hwdata := &pb.HwData{
		Uuid:      "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
		MacId:     macID,
		SutIp:     sutIP,
		Serialnum: serialnum,
	}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	mockRequest := &pb.CreateNodesRequest{
		Payload: payloads,
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	ctx := inv_testing.CreateIncomingContextWithENJWT(t, context.Background(), tenant1)
	ctx = tenant.AddTenantIDToContext(ctx, tenant1)
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.CreateNodesResponse
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
			want:    &pb.CreateNodesResponse{Payload: payloads},
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
		{
			name: "Test case for invalid serial number",
			fields: fields{
				invClient:  om_testing.InvClient,
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: &pb.CreateNodesRequest{
					Payload: []*pb.NodeData{
						{
							Hwdata: []*pb.HwData{
								{
									Serialnum: "123",
									Uuid:      "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
									MacId:     "00.00.00.00",
								},
							},
						},
					},
				},
			},
			want: &pb.CreateNodesResponse{
				Payload: []*pb.NodeData{
					{
						Hwdata: []*pb.HwData{
							{
								Uuid:      "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
								MacId:     "00.00.00.00",
								Serialnum: "N/A",
							},
						},
					},
				},
				ProjectId: tenant1,
			},
			wantErr: false,
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
				req: &pb.CreateNodesRequest{
					Payload: []*pb.NodeData{
						{
							Hwdata: []*pb.HwData{
								{
									Serialnum: "123",
									Uuid:      "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
									MacId:     "00.00.00.00",
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
	//nolint:dupl // These tests cover different scenarios.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InteractiveOnboardingService{
				UnimplementedInteractiveOnboardingServiceServer: tt.fields.UnimplementedInteractiveOnboardingServiceServer,
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
			}
			got, err := s.CreateNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("InteractiveOnboardingService.CreateNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InteractiveOnboardingService.CreateNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInteractiveOnboardingService_CreateNodes_Case_Success(t *testing.T) {
	type fields struct {
		UnimplementedInteractiveOnboardingServiceServer pb.UnimplementedInteractiveOnboardingServiceServer
		invClient                                       *invclient.OnboardingInventoryClient
		enableAuth                                      bool
		rbac                                            *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
	type args struct {
		ctx context.Context
		req *pb.CreateNodesRequest
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	ctx := inv_testing.CreateIncomingContextWithENJWT(t, context.Background(), tenant1)
	ctx = tenant.AddTenantIDToContext(ctx, tenant1)
	dao := inv_testing.NewInvResourceDAOOrFail(t)
	host := dao.CreateHost(t, tenant1)
	hwdata1 := &pb.HwData{Uuid: host.GetUuid(), Serialnum: "ABCDEFG", MacId: generateValidMacID(), SutIp: sutIP}
	hwdatas1 := []*pb.HwData{hwdata1}
	payload1 := pb.NodeData{Hwdata: hwdatas1}
	payloads1 := []*pb.NodeData{&payload1}
	mockRequest1 := &pb.CreateNodesRequest{
		Payload: payloads1,
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.CreateNodesResponse
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
			want:    &pb.CreateNodesResponse{Payload: payloads1, ProjectId: tenant1},
			wantErr: false,
		},
	}
	//nolint:dupl // These tests cover different scenarios.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InteractiveOnboardingService{
				UnimplementedInteractiveOnboardingServiceServer: tt.fields.UnimplementedInteractiveOnboardingServiceServer,
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
			}
			got, err := s.CreateNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("InteractiveOnboardingService.CreateNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InteractiveOnboardingService.CreateNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInteractiveOnboardingService_startZeroTouch(t *testing.T) {
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	host := inv_testing.CreateHost(t, nil, nil)
	osRes := inv_testing.CreateOs(t)
	dao := inv_testing.NewInvResourceDAOOrFail(t)

	s := &InteractiveOnboardingService{
		InventoryClientService: InventoryClientService{
			invClient:    om_testing.InvClient,
			invClientAPI: om_testing.InvClient,
		},
	}

	t.Run("ZeroTouch without Provider", func(t *testing.T) {
		if err := s.startZeroTouch(context.Background(), host.TenantId, host.ResourceId); err != nil {
			t.Errorf("InteractiveOnboardingService.startZeroTouch() error = %v", err)
		}
	})

	t.Run("ZeroTouch with provider and AutoProvision false", func(t *testing.T) {
		providerConfig := fmt.Sprintf(`{"defaultOs":%q,"autoProvision":false}`, osRes.GetResourceId())
		dao.CreateProvider(t, host.GetTenantId(), onboarding_types.DefaultProviderName,
			inv_testing.ProviderConfig(providerConfig),
			inv_testing.ProviderKind(providerv1.ProviderKind_PROVIDER_KIND_BAREMETAL),
		)
		if err := s.startZeroTouch(context.Background(), host.TenantId, host.ResourceId); err != nil {
			t.Errorf("InteractiveOnboardingService.startZeroTouch() error = %v", err)
		}
	})

	t.Run("ZeroTouch with pre-provisioned instance", func(t *testing.T) {
		instance := inv_testing.CreateInstance(t, host, osRes)
		// Ensure internal channel is empty before staring the actual test.
		cleanupInternalWatcher()

		assertInternalEvent(t, &client.ResourceTenantIDCarrier{
			TenantId:   instance.GetTenantId(),
			ResourceId: instance.GetResourceId(),
		})
		if err := s.startZeroTouch(context.Background(), host.TenantId, host.ResourceId); err != nil {
			t.Errorf("InteractiveOnboardingService.startZeroTouch() error = %v", err)
		}
	})
}

func TestInteractiveOnboardingService_startZeroTouch_AutoProvision(t *testing.T) {
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	host := inv_testing.CreateHost(t, nil, nil)
	osRes := inv_testing.CreateOs(t)
	dao := inv_testing.NewInvResourceDAOOrFail(t)

	s := &InteractiveOnboardingService{
		InventoryClientService: InventoryClientService{
			invClient:    om_testing.InvClient,
			invClientAPI: om_testing.InvClient,
		},
	}
	localAccount := inv_testing.CreateLocalAccount(t, "user", "ssh-ed25519 AAAAC3NzaC1lZDI1")
	providerConfig := fmt.Sprintf(
		`{
			"defaultOs": %q,
			"autoProvision": true,
			"OSSecurityFeatureEnable": false,
			"defaultLocalAccount": %q
		}`,
		osRes.GetResourceId(),
		localAccount.GetResourceId(),
	)
	dao.CreateProvider(t, host.GetTenantId(), onboarding_types.DefaultProviderName,
		inv_testing.ProviderConfig(providerConfig),
		inv_testing.ProviderKind(providerv1.ProviderKind_PROVIDER_KIND_BAREMETAL),
	)

	if err := s.startZeroTouch(context.Background(), host.GetTenantId(), host.GetResourceId()); err != nil {
		t.Errorf("InteractiveOnboardingService.startZeroTouch() error = %v", err)
	}
	instances, err := om_testing.InvClient.GetInstanceResources(context.Background())
	require.NoError(t, err)
	require.Len(t, instances, 1, "Wrong number of expected instances for autoProvision")
	autoProvInst := instances[0]
	assert.Equal(t, host.GetResourceId(), autoProvInst.GetHost().GetResourceId(), "Instance host resource id mismatch")

	dao.HardDeleteInstance(t, autoProvInst.GetTenantId(), autoProvInst.GetResourceId())
}

func TestInteractiveOnboardingService_startZeroTouch_WrongProvider(t *testing.T) {
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	host := inv_testing.CreateHost(t, nil, nil)
	inv_testing.CreateProvider(t, onboarding_types.DefaultProviderName)

	s := &InteractiveOnboardingService{
		InventoryClientService: InventoryClientService{
			invClient:    om_testing.InvClient,
			invClientAPI: om_testing.InvClient,
		},
	}
	err := s.startZeroTouch(context.Background(), host.GetTenantId(), host.GetResourceId())
	require.NoError(t, err)
}

func TestInteractiveOnboardingService_startZeroTouch_Case(t *testing.T) {
	type fields struct {
		UnimplementedInteractiveOnboardingServiceServer pb.UnimplementedInteractiveOnboardingServiceServer
		invClient                                       *invclient.OnboardingInventoryClient
		invClientAPI                                    *invclient.OnboardingInventoryClient
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
			s := &InteractiveOnboardingService{
				UnimplementedInteractiveOnboardingServiceServer: tt.fields.UnimplementedInteractiveOnboardingServiceServer,
				InventoryClientService: InventoryClientService{
					invClient:    tt.fields.invClient,
					invClientAPI: tt.fields.invClientAPI,
				},
			}
			if err := s.startZeroTouch(tt.args.ctx, tt.args.hostTenantID, tt.args.hostResID); (err != nil) != tt.wantErr {
				t.Errorf("InteractiveOnboardingService.startZeroTouch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewInteractiveOnboardingService_Case(t *testing.T) {
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
		want    *InteractiveOnboardingService
		wantErr bool
	}{
		{
			name: "NewInteractiveOnboardingService_WithInvalidRBACFile",
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
			got, err := NewInteractiveOnboardingService(tt.args.invClient, tt.args.inventoryAdr,
				tt.args.enableTracing, tt.args.enableAuth, tt.args.rbac)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewInteractiveOnboardingService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewInteractiveOnboardingService() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInteractiveOnboardingService_CreateNodes_Case5(t *testing.T) {
	type fields struct {
		UnimplementedInteractiveOnboardingServiceServer pb.UnimplementedInteractiveOnboardingServiceServer
		invClient                                       *invclient.OnboardingInventoryClient
		enableAuth                                      bool
		rbac                                            *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
	type args struct {
		ctx context.Context
		req *pb.CreateNodesRequest
	}
	macID := generateValidMacID()
	serialnum := serialnum

	hwdata := &pb.HwData{
		Uuid:      "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
		MacId:     macID,
		SutIp:     sutIP,
		Serialnum: serialnum,
	}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	mockRequest := &pb.CreateNodesRequest{
		Payload: payloads,
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	ctx := inv_testing.CreateIncomingContextWithENJWT(t, context.Background(), tenant1)
	ctx = tenant.AddTenantIDToContext(ctx, tenant1)
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.CreateNodesResponse
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
	//nolint:dupl // These tests cover different scenarios.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InteractiveOnboardingService{
				UnimplementedInteractiveOnboardingServiceServer: tt.fields.UnimplementedInteractiveOnboardingServiceServer,
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
			}
			got, err := s.CreateNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("InteractiveOnboardingService.CreateNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) && !tt.wantErr {
				t.Errorf("InteractiveOnboardingService.CreateNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInteractiveOnboardingService_startZeroTouch_Case2(t *testing.T) {
	type fields struct {
		UnimplementedInteractiveOnboardingServiceServer pb.UnimplementedInteractiveOnboardingServiceServer
		invClient                                       *invclient.OnboardingInventoryClient
		invClientAPI                                    *invclient.OnboardingInventoryClient
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
			s := &InteractiveOnboardingService{
				UnimplementedInteractiveOnboardingServiceServer: tt.fields.UnimplementedInteractiveOnboardingServiceServer,
				InventoryClientService: InventoryClientService{
					invClient:    tt.fields.invClient,
					invClientAPI: tt.fields.invClientAPI,
				},
			}
			if err := s.startZeroTouch(tt.args.ctx, tt.args.hostTenantID, tt.args.hostResID); (err != nil) != tt.wantErr {
				t.Errorf("InteractiveOnboardingService.startZeroTouch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInteractiveOnboardingService_startZeroTouch_MultiTenant(t *testing.T) {
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})

	dao := inv_testing.NewInvResourceDAOOrFail(t)

	// Create two providers with the same name but with different tenants
	dao.CreateProvider(t, tenant1, onboarding_types.DefaultProviderName,
		inv_testing.ProviderKind(providerv1.ProviderKind_PROVIDER_KIND_BAREMETAL))
	dao.CreateProvider(t, tenant2, onboarding_types.DefaultProviderName,
		inv_testing.ProviderKind(providerv1.ProviderKind_PROVIDER_KIND_BAREMETAL))

	host := dao.CreateHost(t, tenant1)

	s := &InteractiveOnboardingService{
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

func TestInteractiveOnboardingService_CreateNodes_Case6(t *testing.T) {
	type fields struct {
		UnimplementedInteractiveOnboardingServiceServer pb.UnimplementedInteractiveOnboardingServiceServer
		invClient                                       *invclient.OnboardingInventoryClient
		enableAuth                                      bool
		rbac                                            *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
	type args struct {
		ctx context.Context
		req *pb.CreateNodesRequest
	}
	hwdata := &pb.HwData{Uuid: "9fa8a788-f9f8-434a-8620-bbed2a12b0a"}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	mockRequest := &pb.CreateNodesRequest{
		Payload: payloads,
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	tenantID := u_uuid.NewString()
	ctx := createIncomingContextWithENJWT(t, tenantID)
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.CreateNodesResponse
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
			want:    &pb.CreateNodesResponse{Payload: payloads},
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
			s := &InteractiveOnboardingService{
				UnimplementedInteractiveOnboardingServiceServer: tt.fields.UnimplementedInteractiveOnboardingServiceServer,
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
			}
			_, err := s.CreateNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("InteractiveOnboardingService.CreateNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestInteractiveOnboardingService_CreateNodes_Case7(t *testing.T) {
	type fields struct {
		UnimplementedInteractiveOnboardingServiceServer pb.UnimplementedInteractiveOnboardingServiceServer
		invClient                                       *invclient.OnboardingInventoryClient
		enableAuth                                      bool
		rbac                                            *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
	type args struct {
		ctx context.Context
		req *pb.CreateNodesRequest
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	macID := generateValidMacID()
	serialnum := serialnum

	hwdata := &pb.HwData{
		Uuid:      "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
		MacId:     macID,
		SutIp:     sutIP,
		Serialnum: serialnum,
	}
	hwdatas := []*pb.HwData{hwdata}
	payload := pb.NodeData{Hwdata: hwdatas}
	payloads := []*pb.NodeData{&payload}
	mockRequest := &pb.CreateNodesRequest{
		Payload: payloads,
	}

	ctx := inv_testing.CreateIncomingContextWithENJWT(t, context.Background(), tenant1)
	ctx = tenant.AddTenantIDToContext(ctx, tenant1)
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.CreateNodesResponse
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
	//nolint:dupl // These tests cover different scenarios.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InteractiveOnboardingService{
				UnimplementedInteractiveOnboardingServiceServer: tt.fields.UnimplementedInteractiveOnboardingServiceServer,
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
			}
			got, err := s.CreateNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("InteractiveOnboardingService.CreateNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) && !tt.wantErr {
				t.Errorf("InteractiveOnboardingService.CreateNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInteractiveOnboardingService_checkNCreateInstance(t *testing.T) {
	type fields struct {
		UnimplementedInteractiveOnboardingServiceServer pb.UnimplementedInteractiveOnboardingServiceServer
		invClient                                       *invclient.OnboardingInventoryClient
		invClientAPI                                    *invclient.OnboardingInventoryClient
		rbac                                            *rbac.Policy
		authEnabled                                     bool
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	type args struct {
		ctx      context.Context
		pconf    providerconfiguration.ProviderConfig
		tenentID string
		host     *computev1.HostResource
	}
	tenantID := u_uuid.NewString()
	ctx := createIncomingContextWithENJWT(t, tenantID)
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
				pconf: providerconfiguration.ProviderConfig{
					AutoProvision: true,
				},
				host:     &computev1.HostResource{},
				tenentID: "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InteractiveOnboardingService{
				UnimplementedInteractiveOnboardingServiceServer: tt.fields.UnimplementedInteractiveOnboardingServiceServer,
				InventoryClientService: InventoryClientService{
					invClient:    tt.fields.invClient,
					invClientAPI: tt.fields.invClientAPI,
				},
				rbac:        tt.fields.rbac,
				authEnabled: tt.fields.authEnabled,
			}
			if err := s.checkNCreateInstance(tt.args.ctx, tt.args.tenentID, tt.args.pconf,
				tt.args.host); (err != nil) != tt.wantErr {
				t.Errorf("InteractiveOnboardingService.checkNCreateInstance() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInteractiveOnboardingService_handleRegisteredState(t *testing.T) {
	type fields struct {
		UnimplementedNonInteractiveOnboardingServiceServer pb.UnimplementedNonInteractiveOnboardingServiceServer
		invClient                                          *invclient.OnboardingInventoryClient
		invClientAPI                                       *invclient.OnboardingInventoryClient
		rbac                                               *rbac.Policy
		authEnabled                                        bool
	}
	var art MockNonInteractiveOnboardingServiceOnboardNodeStreamServer
	art.On("Send", mock.Anything).Return(errors.New("err"))
	var art1 MockNonInteractiveOnboardingServiceOnboardNodeStreamServer
	art1.On("Send", mock.Anything).Return(nil)
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	host := inv_testing.CreateHost(t, nil, nil)
	type args struct {
		stream  pb.NonInteractiveOnboardingService_OnboardNodeStreamServer
		hostInv *computev1.HostResource
		req     *pb.OnboardNodeStreamRequest
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
			name: "positive",
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
				req: &pb.OnboardNodeStreamRequest{
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
				t.Errorf("InteractiveOnboardingService.handleRegisteredState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInteractiveOnboardingService_handleOnboardedState(t *testing.T) {
	var nioMock MockNonInteractiveOnboardingServiceOnboardNodeStreamServer
	nioMock.On("Send", mock.Anything).Return(nil)
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})

	s := &NonInteractiveOnboardingService{
		InventoryClientService: InventoryClientService{
			invClient:    om_testing.InvClient,
			invClientAPI: om_testing.InvClient,
		},
	}

	dao := inv_testing.NewInvResourceDAOOrFail(t)
	host := dao.CreateHost(t, tenant1)
	fakeHost := &computev1.HostResource{
		TenantId:   tenant1,
		Uuid:       u_uuid.NewString(),
		ResourceId: "host-12345678",
	}

	// Ensure to restore AuthServiceFactory after the test.
	currAuthServiceFactory := auth.AuthServiceFactory
	t.Cleanup(func() {
		auth.AuthServiceFactory = currAuthServiceFactory
	})

	t.Run("Failing Create Credentials", func(t *testing.T) {
		auth.AuthServiceFactory = om_testing.AuthServiceMockFactory(true, true, false)
		err := s.handleOnboardedState(&nioMock, fakeHost, &pb.OnboardNodeStreamRequest{Uuid: fakeHost.GetUuid()})
		require.Error(t, err)
		assert.ErrorContains(t, err, "Failed to check if EN credentials for host exist.")
	})

	auth.AuthServiceFactory = om_testing.AuthServiceMockFactory(false, false, false)

	t.Run("Non existing host", func(t *testing.T) {
		err := s.handleOnboardedState(&nioMock, fakeHost, &pb.OnboardNodeStreamRequest{Uuid: fakeHost.GetUuid()})
		require.Error(t, err)
		assert.True(t, inv_errors.IsNotFound(err))
	})

	t.Run("Positive", func(t *testing.T) {
		err := s.handleOnboardedState(&nioMock, host, &pb.OnboardNodeStreamRequest{Uuid: host.GetUuid()})
		assert.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
		om_testing.AssertHost(t, host.GetTenantId(), host.GetResourceId(),
			computev1.HostState_HOST_STATE_ONBOARDED,
			computev1.HostState_HOST_STATE_ONBOARDED,
			inv_status.New(inv_status.DefaultHostStatus, statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED),
		)
	})
}

func TestInteractiveOnboardingService_handleDefaultState(t *testing.T) {
	var art MockNonInteractiveOnboardingServiceOnboardNodeStreamServer
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
				t.Errorf("InteractiveOnboardingService.handleDefaultState() error = %v, wantErr %v", err, tt.wantErr)
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

//nolint:funlen // reason: function is long due to necessary test cases.
func TestInteractiveOnboardingServiceOnboardNodeStream(t *testing.T) {
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

	t.Setenv("ONBOARDING_MANAGER_CLIENT_NAME", "env")
	t.Setenv("ONBOARDING_CREDENTIALS_SECRET_NAME", "env")
	art, art1, art2, art3 := setupMockOnboardNodeStreamServers(host)
	currAuthServiceFactory := auth.AuthServiceFactory
	currFlagDisableCredentialsManagement := *flags.FlagDisableCredentialsManagement
	t.Cleanup(func() {
		auth.AuthServiceFactory = currAuthServiceFactory
		*flags.FlagDisableCredentialsManagement = currFlagDisableCredentialsManagement
	})
	*flags.FlagDisableCredentialsManagement = false
	auth.AuthServiceFactory = om_testing.AuthServiceMockFactory(false, false, true)
	hostWithNASerial := inv_testing.CreateHostWithArgs(t, "host-1", "44414747-3031-3052-b030-453347474122",
		serialNumNotAvailable, "", nil, nil, true)
	art4 := new(MockNonInteractiveOnboardingServiceOnboardNodeStreamServer)
	art4.On("Send", mock.Anything).Return(nil)
	art4.On("Recv").Return(&pb.OnboardNodeStreamRequest{
		Serialnum: hostWithNASerial.SerialNumber,
		Uuid:      hostWithNASerial.Uuid,
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
				stream: art,
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
				stream: art1,
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
				stream: art2,
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
				stream: art3,
			},
			wantErr: false,
		},
		{
			name: "OnboardNodeStream with incorrect Serial Number",
			fields: fields{
				UnimplementedNonInteractiveOnboardingServiceServer: pb.UnimplementedNonInteractiveOnboardingServiceServer{},
				invClient:    om_testing.InvClient,
				invClientAPI: &invclient.OnboardingInventoryClient{},
			},
			args: args{
				stream: art4,
			},
			wantErr: false,
		},
	}
	defer func() {
		if err := os.Unsetenv("ONBOARDING_MANAGER_CLIENT_NAME"); err != nil {
			t.Logf("Failed to unset env: %v", err)
		}
		if err := os.Unsetenv("ONBOARDING_CREDENTIALS_SECRET_NAME"); err != nil {
			t.Logf("Failed to unset env: %v", err)
		}
	}()
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
				t.Errorf("InteractiveOnboardingService.OnboardNodeStream() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInteractiveOnboardingServiceOnboardNodeStream_WithInstance(t *testing.T) {
	currAuthServiceFactory := auth.AuthServiceFactory
	t.Cleanup(func() {
		auth.AuthServiceFactory = currAuthServiceFactory
	})
	auth.AuthServiceFactory = om_testing.AuthServiceMockFactory(false, false, false)
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})

	dao := inv_testing.NewInvResourceDAOOrFail(t)
	host := dao.CreateHost(t, tenant1)
	osRes := dao.CreateOs(t, tenant1)
	instance := dao.CreateInstance(t, tenant1, host, osRes)

	var nioStreamMock MockNonInteractiveOnboardingServiceOnboardNodeStreamServer
	nioStreamMock.On("Send", mock.Anything).Return(nil)
	nioStreamMock.On("Recv").Return(&pb.OnboardNodeStreamRequest{
		Uuid:      host.Uuid,
		Serialnum: host.SerialNumber,
		MacId:     host.PxeMac,
		HostIp:    host.BmcIp,
	}, nil)

	s := &NonInteractiveOnboardingService{
		InventoryClientService: InventoryClientService{
			invClient:    om_testing.InvClient,
			invClientAPI: om_testing.InvClient,
		},
	}
	cleanupInternalWatcher()
	assertInternalEvent(t, &client.ResourceTenantIDCarrier{
		TenantId:   instance.GetTenantId(),
		ResourceId: instance.GetResourceId(),
	})

	err := s.OnboardNodeStream(&nioStreamMock)
	require.NoError(t, err)
}

func setupMockOnboardNodeStreamServers(host *computev1.HostResource) (
	streamServer *MockNonInteractiveOnboardingServiceOnboardNodeStreamServer,
	streamServer1 *MockNonInteractiveOnboardingServiceOnboardNodeStreamServer,
	streamServer2 *MockNonInteractiveOnboardingServiceOnboardNodeStreamServer,
	streamServer3 *MockNonInteractiveOnboardingServiceOnboardNodeStreamServer,
) {
	art := new(MockNonInteractiveOnboardingServiceOnboardNodeStreamServer)
	art1 := new(MockNonInteractiveOnboardingServiceOnboardNodeStreamServer)
	art2 := new(MockNonInteractiveOnboardingServiceOnboardNodeStreamServer)
	art3 := new(MockNonInteractiveOnboardingServiceOnboardNodeStreamServer)
	// Mock the first stream with an error
	art.On("Recv").Return(&pb.OnboardNodeStreamRequest{}, errors.New("err"))

	// Mock the second stream
	art1.On("Send", mock.Anything).Return(nil)
	art1.On("Recv").Return(&pb.OnboardNodeStreamRequest{
		Uuid:      host.Uuid,
		Serialnum: host.SerialNumber,
		MacId:     host.PxeMac,
		HostIp:    host.BmcIp,
	}, nil)
	// Mock the third stream
	art2.On("Send", mock.Anything).Return(nil)
	art2.On("Recv").Return(&pb.OnboardNodeStreamRequest{
		Uuid:      host.Uuid,
		Serialnum: host.SerialNumber,
		MacId:     host.PxeMac,
		HostIp:    host.BmcIp,
	}, nil)

	// Mock the fourth stream
	art3.On("Send", mock.Anything).Return(nil)
	art3.On("Recv").Return(&pb.OnboardNodeStreamRequest{
		Serialnum: host.SerialNumber,
		Uuid:      host.Uuid,
	}, nil)

	return art, art1, art2, art3
}

func TestInteractiveOnboardingService_getHostResource(t *testing.T) {
	type fields struct {
		UnimplementedNonInteractiveOnboardingServiceServer pb.UnimplementedNonInteractiveOnboardingServiceServer
		invClient                                          *invclient.OnboardingInventoryClient
		invClientAPI                                       *invclient.OnboardingInventoryClient
	}
	type args struct {
		req *pb.OnboardNodeStreamRequest
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
				req: &pb.OnboardNodeStreamRequest{
					Uuid: "f9f8-434a-8620-bbed2a12b0ad",
				},
			},
			want:    nil,
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
				req: &pb.OnboardNodeStreamRequest{
					Serialnum: "12345",
				},
			},
			want:    nil,
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
				req: &pb.OnboardNodeStreamRequest{
					Uuid:      host.Uuid,
					Serialnum: "12345",
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "getHostResource test case with empty request",
			fields: fields{
				UnimplementedNonInteractiveOnboardingServiceServer: pb.UnimplementedNonInteractiveOnboardingServiceServer{},
				invClient:    om_testing.InvClient,
				invClientAPI: &invclient.OnboardingInventoryClient{},
			},
			args: args{
				req: &pb.OnboardNodeStreamRequest{},
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			s := &NonInteractiveOnboardingService{
				UnimplementedNonInteractiveOnboardingServiceServer: tt.fields.UnimplementedNonInteractiveOnboardingServiceServer,
				InventoryClientService: InventoryClientService{
					invClient:    tt.fields.invClient,
					invClientAPI: tt.fields.invClientAPI,
				},
			}
			_, err := s.getHostResource(tt.args.req)
			if (err != nil) && !errors.Is(err, err) != tt.wantErr {
				t.Errorf("getHostResource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

//nolint:funlen // reason: function is long due to necessary test cases.
func TestInteractiveOnboardingService_getHostResourcetest(t *testing.T) {
	type fields struct {
		UnimplementedNonInteractiveOnboardingServiceServer pb.UnimplementedNonInteractiveOnboardingServiceServer
		invClient                                          *invclient.OnboardingInventoryClient
		invClientAPI                                       *invclient.OnboardingInventoryClient
	}
	type args struct {
		req *pb.OnboardNodeStreamRequest
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	// Create a host for testing
	host1 := inv_testing.CreateHostWithArgs(t, "host-1", "44414747-3031-3052-b030-453347474122", "", "", nil, nil, true)
	host2 := inv_testing.CreateHostWithArgs(t, "host-2", "", "ABCDEFG", "", nil, nil, true)
	host3 := inv_testing.CreateHostWithArgs(t, "host-3", "44414747-3031-3052-b030-453347474166", serialnum, "", nil, nil, true)
	host4 := inv_testing.CreateHostWithArgs(t, "host-3", "44414747-3031-3052-b030-453347474168", "", "", nil, nil, true)
	host5 := inv_testing.CreateHostWithArgs(t, "host-3", "44414747-3031-3052-b030-453347474169", "", "", nil, nil, true)

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
				req: &pb.OnboardNodeStreamRequest{
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
				req: &pb.OnboardNodeStreamRequest{
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
				req: &pb.OnboardNodeStreamRequest{
					Uuid:      "44414747-3031-3052-b030-453347474166",
					Serialnum: serialnum,
				},
			},
			want:    host3,
			wantErr: true,
		},
		{
			name: "getHostResource test case with serial number empty and correct host uuid",
			fields: fields{
				UnimplementedNonInteractiveOnboardingServiceServer: pb.UnimplementedNonInteractiveOnboardingServiceServer{},
				invClient:    om_testing.InvClient,
				invClientAPI: &invclient.OnboardingInventoryClient{},
			},
			args: args{
				req: &pb.OnboardNodeStreamRequest{
					Uuid:      "44414747-3031-3052-b030-453347474168",
					Serialnum: "",
				},
			},
			want:    host4,
			wantErr: false,
		},
		{
			name: "getHostResource test case with incorrect serial number and correct host uuid",
			fields: fields{
				UnimplementedNonInteractiveOnboardingServiceServer: pb.UnimplementedNonInteractiveOnboardingServiceServer{},
				invClient:    om_testing.InvClient,
				invClientAPI: &invclient.OnboardingInventoryClient{},
			},
			args: args{
				req: &pb.OnboardNodeStreamRequest{
					Uuid:      "44414747-3031-3052-b030-453347474169",
					Serialnum: "To be filled by O.E.M",
				},
			},
			want:    host5,
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
				t.Errorf("InteractiveOnboardingService.getHostResource() error = %v, wantErr %v", err, tt.wantErr)
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
	t.Setenv("DEFAULT_K8S_NAMESPACE", "test")
	t.Setenv("TINKER_VERSION", "value")
	t.Setenv("TINKER_ARTIFACT_NAME", "test")

	tests := []struct {
		name string
	}{
		{
			name: "Positive Test Case",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			env.MustEnsureRequired()
		})
	}
}

// FUZZ test cases.
func FuzzCreateNodes(f *testing.F) {
	f.Add("node1", "platform1", "9fa0a788-f9f8-434a-8620-bbed2a12b0ad", "00:1A:2B:3C:4D:5E", "ABCDEFGH", "192.168.1.1")
	f.Fuzz(func(t *testing.T, hwId string, platformType string, uuid string, macId string, serialNum string, sutIp string) {
		om_testing.CreateInventoryOnboardingClientForTesting()
		t.Cleanup(func() {
			om_testing.DeleteInventoryOnboardingClientForTesting()
		})
		if hwId == "" || platformType == "" || uuid == "" || macId == "" || serialNum == "" || sutIp == "" {
			t.Skip("Skipping test because one of the required fields is empty")
			return
		}

		macId = validateOrGenerate(macId, isValidMacID, generateValidMacID)
		serialNum = validateOrGenerate(serialNum, isValidserialNum, generateValidSerialNum)
		sutIp = validateOrGenerate(sutIp, isValidsutIP, generateValidSutIP)
		ctx := inv_testing.CreateIncomingContextWithENJWT(t, context.Background(), tenant1)
		ctx = tenant.AddTenantIDToContext(ctx, tenant1)
		rbacServer, err := rbac.New(rbacRules)
		if err != nil {
			t.Errorf("Error at the RBAC rules %v", err)
		}
		s := &InteractiveOnboardingService{
			UnimplementedInteractiveOnboardingServiceServer: pb.UnimplementedInteractiveOnboardingServiceServer{},
			InventoryClientService: InventoryClientService{
				invClient:    om_testing.InvClient,
				invClientAPI: om_testing.InvClient,
			},
			rbac:        rbacServer,
			authEnabled: true,
		}
		hwdata := &pb.HwData{
			MacId:     macId,
			Uuid:      u_uuid.NewString(),
			Serialnum: serialNum,
			SutIp:     sutIp,
		}
		hwdatas := []*pb.HwData{hwdata}
		payload1 := pb.NodeData{Hwdata: hwdatas}
		payloads := []*pb.NodeData{&payload1}
		mockRequest := &pb.CreateNodesRequest{
			Payload: payloads,
		}
		_, err = s.CreateNodes(ctx, mockRequest)
		if err != nil {
			t.Errorf("CreateNodes returned an error: %v", err)
		}
	})
}

func validateOrGenerate(value string, isValidFunc func(string) bool, generateFunc func() string) string {
	if !isValidFunc(value) {
		return generateFunc()
	}
	return value
}

// isValidMacID checks if the given string is a valid MAC ID.
func isValidMacID(mac string) bool {
	re := regexp.MustCompile(`^([0-9a-fA-F]{2}([-:])){5}[0-9a-fA-F]{2}$`)
	return re.MatchString(mac)
}

func isValidserialNum(serialNum string) bool {
	re := regexp.MustCompile(`^[A-Za-z0-9]{5,20}$`)
	return re.MatchString(serialNum)
}

func isValidsutIP(sutIP string) bool {
	re := regexp.MustCompile(`^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`)
	return re.MatchString(sutIP)
}

// generateValidMacID generates a valid MAC ID.
func generateValidMacID() string {
	mac := make([]byte, 6)
	_, err := rand.Read(mac)
	if err != nil {
		fmt.Println(err)
	}
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", mac[0], mac[1], mac[2], mac[3], mac[4], mac[5])
}

func generateValidSerialNum() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	const length = 8

	serialNum := make([]byte, length)
	for i := range serialNum {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			fmt.Println(err)
		}
		serialNum[i] = charset[n.Int64()]
	}
	return string(serialNum)
}

func generateValidSutIP() string {
	generateOctet := func() int {
		n, err := rand.Int(rand.Reader, big.NewInt(256))
		if err != nil {
			fmt.Println(err)
		}
		return int(n.Int64())
	}

	// Generate 4 octets and format them as a valid IP address
	return fmt.Sprintf("%d.%d.%d.%d",
		generateOctet(),
		generateOctet(),
		generateOctet(),
		generateOctet(),
	)
}

func FuzzOnboardNodeStream(f *testing.F) {
	f.Add("hostip")
	f.Fuzz(func(t *testing.T, ip string) {
		resp := &pb.OnboardNodeStreamRequest{
			Uuid:      u_uuid.New().String(),
			Serialnum: getFirstNChars(getMD5Hash(ip), 8),
			MacId:     "",
			HostIp:    getFirstNChars(getMD5Hash(ip), 6),
		}

		var art MockNonInteractiveOnboardingServiceOnboardNodeStreamServer
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

func cleanupInternalWatcher() {
	go func() {
		select {
		case <-om_testing.InvClient.InternalWatcher:
		default:
		}
	}()
}

func assertInternalEvent(t *testing.T, expectedEvent *client.ResourceTenantIDCarrier) {
	t.Helper()
	go func() {
		select {
		case ev, ok := <-om_testing.InvClient.InternalWatcher:
			require.True(t, ok)
			t.Logf("Got expected internal event")

			if eq, diff := inv_testing.ProtoEqualOrDiff(ev, expectedEvent); !eq {
				t.Errorf("Unexpected internal event content: %v", diff)
			}
		case <-time.After(1 * time.Second):
			t.Errorf("Expected internal event not delivered!")
		}
	}()
}

func TestInteractiveOnboardingService_startZeroTouch_OSSecurityFeatureDisable(t *testing.T) {
	// Create inventory onboarding client for testing
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})

	// Create test data Host and OS resources
	host := inv_testing.CreateHost(t, nil, nil)
	osRes := inv_testing.CreateOs(t)
	dao := inv_testing.NewInvResourceDAOOrFail(t)

	// Initialize the service with the test client
	s := &InteractiveOnboardingService{
		InventoryClientService: InventoryClientService{
			invClient:    om_testing.InvClient,
			invClientAPI: om_testing.InvClient,
		},
	}

	// Prepare provider configuration with os security feature disable
	localAccount := inv_testing.CreateLocalAccount(t, "user", "ssh-ed25519 AAAAC3NzaC1lZDI1")
	providerConfig := fmt.Sprintf(
		`{"defaultOs":%q, "autoProvision":true, "OSSecurityFeatureEnable":false, 
		"defaultLocalAccount":%q}`,
		osRes.GetResourceId(),
		localAccount.GetResourceId(),
	)
	dao.CreateProvider(t, host.GetTenantId(), onboarding_types.DefaultProviderName,
		inv_testing.ProviderConfig(providerConfig),
		inv_testing.ProviderKind(providerv1.ProviderKind_PROVIDER_KIND_BAREMETAL),
	)

	err := s.startZeroTouch(context.Background(), host.GetTenantId(), host.GetResourceId())
	require.NoError(t, err, "Expected no error during zero touch provisioning")

	// Verify that an instance was created
	instances, err := om_testing.InvClient.GetInstanceResources(context.Background())
	require.NoError(t, err, "Failed to get instance resources")
	require.Len(t, instances, 1, "Wrong number of expected instances for autoProvision")

	// Verify the instance details
	autoProvInst := instances[0]
	assert.Equal(t, osv1.SecurityFeature_SECURITY_FEATURE_NONE, autoProvInst.GetSecurityFeature(), "OS security feature match")
	// Delete the created instance
	dao.HardDeleteInstance(t, autoProvInst.GetTenantId(), autoProvInst.GetResourceId())
}

func TestInteractiveOnboardingService_startZeroTouch_OSSecurityFeatureEnable(t *testing.T) {
	// Create inventory onboarding client for testing
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})

	// Create Host and OS resources
	host := inv_testing.CreateHost(t, nil, nil)
	osRes := inv_testing.CreateOsWithArgs(t, "", "", "profile:profile",
		osv1.SecurityFeature_SECURITY_FEATURE_SECURE_BOOT_AND_FULL_DISK_ENCRYPTION, osv1.OsType_OS_TYPE_MUTABLE)
	dao := inv_testing.NewInvResourceDAOOrFail(t)

	// Initialize the service with the test client
	s := &InteractiveOnboardingService{
		InventoryClientService: InventoryClientService{
			invClient:    om_testing.InvClient,
			invClientAPI: om_testing.InvClient,
		},
	}

	// Prepare provider configuration with security feature enabled
	localAccount := inv_testing.CreateLocalAccount(t, "user", "ssh-ed25519 AAAAC3NzaC1lZDI1")
	providerConfig := fmt.Sprintf(
		`{"defaultOs":%q, "autoProvision":true, "OSSecurityFeatureEnable":true, 
		"defaultLocalAccount":%q}`,
		osRes.GetResourceId(),
		localAccount.GetResourceId(),
	)
	dao.CreateProvider(t, host.GetTenantId(), onboarding_types.DefaultProviderName,
		inv_testing.ProviderConfig(providerConfig),
		inv_testing.ProviderKind(providerv1.ProviderKind_PROVIDER_KIND_BAREMETAL),
	)

	// Execute the function under test
	err := s.startZeroTouch(context.Background(), host.GetTenantId(), host.GetResourceId())
	require.NoError(t, err, "Expected no error during zero touch provisioning")

	// Verify that an instance was created
	instances, err := om_testing.InvClient.GetInstanceResources(context.Background())
	require.NoError(t, err, "Failed to get instance resources")
	require.Len(t, instances, 1, "Wrong number of expected instances for autoProvision")

	// Verify the instance details
	autoProvInst := instances[0]
	assert.Equal(t, osv1.SecurityFeature_SECURITY_FEATURE_SECURE_BOOT_AND_FULL_DISK_ENCRYPTION, autoProvInst.GetSecurityFeature(),
		"OS security feature match")

	// Clean up: Delete the created instance
	dao.HardDeleteInstance(t, autoProvInst.GetTenantId(), autoProvInst.GetResourceId())
}

//nolint:funlen // reason: function is long due to necessary test cases.
func TestInteractiveOnboardingService_CreateNodes_CaseUpdatedSerialNumberPattern(t *testing.T) {
	type fields struct {
		UnimplementedInteractiveOnboardingServiceServer pb.UnimplementedInteractiveOnboardingServiceServer
		invClient                                       *invclient.OnboardingInventoryClient
		enableAuth                                      bool
		rbac                                            *rbac.Policy
	}
	rbacServer, err := rbac.New(rbacRules)
	require.NoError(t, err)
	type args struct {
		ctx context.Context
		req *pb.CreateNodesRequest
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	ctx := inv_testing.CreateIncomingContextWithENJWT(t, context.Background(), tenant1)
	ctx = tenant.AddTenantIDToContext(ctx, tenant1)
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.CreateNodesResponse
		wantErr bool
	}{
		{
			name: "Positive test case for creating node",
			fields: fields{
				invClient:  om_testing.InvClient,
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: &pb.CreateNodesRequest{
					Payload: []*pb.NodeData{
						{
							Hwdata: []*pb.HwData{
								{
									Serialnum: "To be filled",
									Uuid:      "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
									MacId:     "00.00.00.00",
								},
							},
						},
					},
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "Negative test case - missing tenant in context",
			fields: fields{
				invClient:  &invclient.OnboardingInventoryClient{},
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: context.TODO(),
				req: &pb.CreateNodesRequest{
					Payload: []*pb.NodeData{
						{
							Hwdata: []*pb.HwData{
								{
									Serialnum: "To be filled",
									Uuid:      "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
									MacId:     "00.00.00.00",
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
			name: "Negative test case - invalid serial number",
			fields: fields{
				invClient:  om_testing.InvClient,
				enableAuth: true,
				rbac:       rbacServer,
			},
			args: args{
				ctx: ctx,
				req: &pb.CreateNodesRequest{
					Payload: []*pb.NodeData{
						{
							Hwdata: []*pb.HwData{
								{
									Serialnum: "12",
									Uuid:      "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
									MacId:     "00.00.00.00",
								},
							},
						},
					},
				},
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InteractiveOnboardingService{
				UnimplementedInteractiveOnboardingServiceServer: tt.fields.UnimplementedInteractiveOnboardingServiceServer,
				InventoryClientService: InventoryClientService{
					invClient: tt.fields.invClient,
				},
				authEnabled: tt.fields.enableAuth,
				rbac:        tt.fields.rbac,
			}
			_, err := s.CreateNodes(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("InteractiveOnboardingService.CreateNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
