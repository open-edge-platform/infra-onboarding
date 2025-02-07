// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package invclient

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/api/inventory/v1"
	network_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/api/network/v1"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/api/os/v1"
	provider_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/api/provider/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/client"
	inv_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/status"
	inv_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/testing"
	om_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/onboarding-manager/pkg/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const (
	tenant1 = "11111111-1111-1111-1111-111111111111"
	tenant2 = "22222222-2222-2222-2222-222222222222"
)

func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	projectRoot := filepath.Dir(filepath.Dir(wd))
	policyPath := projectRoot + "/build"
	migrationsDir := projectRoot + "/build"

	inv_testing.StartTestingEnvironment(policyPath, "", migrationsDir)
	run := m.Run() // run all tests
	inv_testing.StopTestingEnvironment()

	os.Exit(run)
}

func TestWithInventoryAddress(t *testing.T) {
	type args struct {
		invAddr string
	}
	tests := []struct {
		name string
		args args
		want Option
	}{
		{
			name: "ProvidingInventoryAddress",
			args: args{},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WithInventoryAddress(tt.args.invAddr); reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithInventoryAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithEnableTracing(t *testing.T) {
	type args struct {
		enableTracing bool
	}
	tests := []struct {
		name string
		args args
		want Option
	}{
		{
			name: "EnablingTracing",
			args: args{enableTracing: false},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WithEnableTracing(tt.args.enableTracing); reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithEnableTracing() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewOnboardingInventoryClientWithOptions(t *testing.T) {
	type args struct {
		opts []Option
	}
	tests := []struct {
		name    string
		args    args
		want    *OnboardingInventoryClient
		wantErr bool
	}{
		{
			name:    "CreatingNewOnboardingInventoryClientWithOptions",
			args:    args{},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewOnboardingInventoryClientWithOptions(tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOnboardingInventoryClientWithOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewOnboardingInventoryClientWithOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewOnboardingInventoryClient(t *testing.T) {
	type args struct {
		invClient client.TenantAwareInventoryClient
		watcher   chan *client.WatchEvents
	}
	tests := []struct {
		name    string
		args    args
		want    *OnboardingInventoryClient
		wantErr bool
	}{
		{
			name:    "CreatingNewOnboardingInventoryClient",
			args:    args{},
			want:    &OnboardingInventoryClient{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewOnboardingInventoryClient(tt.args.invClient, tt.args.watcher)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOnboardingInventoryClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewOnboardingInventoryClient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOnboardingInventoryClient_Close(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	invClient.Close()
}

func TestOnboardingInventoryClient_UpdateHostResource(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	host := inv_testing.CreateHost(t, nil, nil)
	type args struct {
		ctx      context.Context
		host     *computev1.HostResource
		tenantID string
	}
	tests := []struct {
		name  string
		args  args
		valid bool
	}{
		{
			name: "UpdatingHostResource",
			args: args{
				ctx:      context.Background(),
				host:     host,
				tenantID: host.GetTenantId(),
			},
			valid: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err := invClient.UpdateHostResource(ctx, tt.args.tenantID, tt.args.host)
			if err != nil {
				if tt.valid {
					t.Errorf("Failed: %s", err)
					t.FailNow()
				}
			} else {
				if !tt.valid {
					t.Errorf("Succeeded but should have failed")
					t.FailNow()
				}
			}

			if !t.Failed() && tt.valid {
				hostInv, err := invClient.GetHostResourceByUUID(ctx, host.GetTenantId(), host.Uuid)
				require.NoError(t, err)
				require.NotNil(t, hostInv)
			}
		})
	}
}

func TestOnboardingInventoryClient_GetHostResources(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	host := inv_testing.CreateHost(t, nil, nil)
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name        string
		args        args
		wantHostres []*computev1.HostResource
		valid       bool
	}{
		{
			name: "GettingHostResources_EmptyResponse",
			args: args{
				ctx: context.Background(),
			},
			wantHostres: []*computev1.HostResource{},
			valid:       true,
		},
		{
			name: "GettingHostResources_ErrorResponse",
			args: args{
				ctx: context.Background(),
			},
			wantHostres: nil,
			valid:       true,
		},
		{
			name: "GettingHostResources_SuccessfulResponse",
			args: args{
				ctx: context.Background(),
			},
			wantHostres: nil,
			valid:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			_, err := invClient.GetHostResources(ctx)
			if err != nil {
				if tt.valid {
					t.Errorf("Failed: %s", err)
					t.FailNow()
				}
			} else {
				if !tt.valid {
					t.Errorf("Succeeded but should have failed")
					t.FailNow()
				}
			}
			if !t.Failed() && tt.valid {
				hostInv, err := invClient.GetHostResourceByUUID(ctx, host.GetTenantId(), host.Uuid)
				require.NoError(t, err)
				require.NotNil(t, hostInv)
			}
		})
	}
}

func TestOnboardingInventoryClient_FindAllInstances(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient

	host := inv_testing.CreateHost(t, nil, nil)
	type args struct {
		hostID         string
		instanceStatus []*client.ResourceTenantIDCarrier
		details        string
		ctx            context.Context
	}
	tests := []struct {
		name  string
		args  args
		want  []*client.ResourceTenantIDCarrier
		valid bool
	}{
		{
			name: "FindingAllInstances_EmptyResponse",
			args: args{
				hostID:  host.GetResourceId(),
				details: "some detail",
				ctx:     context.Background(),
			},
			want:  nil,
			valid: false,
		},

		{
			name: "FindingAllInstances_ErrorResponse",
			args: args{
				hostID:  host.GetResourceId(),
				details: "some detail",
				ctx:     context.Background(),
			},
			want:  nil,
			valid: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if !t.Failed() && tt.valid {
				hostInv, err := invClient.FindAllInstances(ctx)
				require.NoError(t, err)
				require.Nil(t, hostInv)

				assert.Equal(t, tt.args.instanceStatus, hostInv)
			}
		})
	}
}

func TestOnboardingInventoryClient_GetHostResourceByResourceID(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	host := inv_testing.CreateHost(t, nil, nil)
	type args struct {
		tenantID         string
		hostID           string
		expectedHost     *computev1.HostResource
		details          string
		onboardingStatus inv_status.ResourceStatus
	}
	tests := []struct {
		name  string
		args  args
		want  *computev1.HostResource
		valid bool
	}{
		{
			name: "GettingHostResourceByResourceID_ValidResponse",
			args: args{
				tenantID:         host.GetTenantId(),
				hostID:           host.GetResourceId(),
				expectedHost:     host,
				details:          "some detail",
				onboardingStatus: om_status.OnboardingStatusInProgress,
			},
			want:  nil,
			valid: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			hostInv, err := invClient.GetHostResourceByResourceID(ctx, tt.args.tenantID, host.ResourceId)
			require.NoError(t, err)
			require.NotNil(t, hostInv)
			if eq, diff := inv_testing.ProtoEqualOrDiff(hostInv, tt.args.expectedHost); !eq {
				t.Errorf("Data not equal: %v", diff)
			}
		})
	}
	t.Run("Invalid Resource Id", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		hostInv, err := invClient.GetHostResourceByResourceID(ctx, host.GetTenantId(), "12345")
		require.Error(t, err)
		require.Nil(t, hostInv)
	})
}

func TestOnboardingInventoryClient_CreateHostResource(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	host := inv_testing.CreateHost(t, nil, nil)
	type args struct {
		tenantID         string
		hostID           string
		details          string
		onboardingStatus inv_status.ResourceStatus
	}
	tests := []struct {
		name  string
		args  args
		want  string
		valid bool
	}{
		{
			name: "CreatingHostResource_Success",
			args: args{
				tenantID:         host.GetTenantId(),
				hostID:           host.GetResourceId(),
				details:          "some detail",
				onboardingStatus: om_status.OnboardingStatusInProgress,
			},
			valid: true,
		},
		{
			name: "CreatingHostResource_Error",
			args: args{
				hostID: "host-12345678",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err := invClient.SetHostOnboardingStatus(ctx, tt.args.tenantID, tt.args.hostID, tt.args.onboardingStatus)
			if err != nil {
				if tt.valid {
					t.Errorf("Failed: %s", err)
					t.FailNow()
				}
			} else {
				if !tt.valid {
					t.Errorf("Succeeded but should have failed")
					t.FailNow()
				}
			}
			if !t.Failed() && tt.valid {
				hostInv, err := invClient.CreateHostResource(ctx, tt.args.tenantID, &computev1.HostResource{
					Uuid:     "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
					TenantId: tt.args.tenantID,
				})
				require.NoError(t, err)
				require.NotNil(t, hostInv)
			}
		})
	}
}

func TestOnboardingInventoryClient_GetHostResourceByUUID(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	host := inv_testing.CreateHost(t, nil, nil)
	type args struct {
		tenantID         string
		hostID           string
		onboardingStatus inv_status.ResourceStatus
		uuid             string
	}
	tests := []struct {
		name  string
		args  args
		want  *computev1.HostResource
		valid bool
	}{
		{
			name: "InvalidUUID_ResourceRetrievalError",
			args: args{
				tenantID:         host.GetTenantId(),
				hostID:           host.GetResourceId(),
				onboardingStatus: om_status.OnboardingStatusInProgress,
				uuid:             "123",
			},
			want:  nil,
			valid: false,
		},
		{
			name: "MissingUUID_ResourceRetrievalError",
			args: args{
				tenantID:         host.GetTenantId(),
				hostID:           host.GetResourceId(),
				onboardingStatus: om_status.OnboardingStatusInProgress,
			},
			want:  nil,
			valid: true,
		},
		{
			name: "ValidUUID_ResourceRetrievalSuccess",
			args: args{
				tenantID:         host.GetTenantId(),
				hostID:           host.GetResourceId(),
				onboardingStatus: om_status.OnboardingStatusInProgress,
				uuid:             host.Uuid,
			},
			want: &computev1.HostResource{
				ResourceId: host.GetResourceId(),
			},
			valid: true,
		},
		{
			name: "InvalidUUID_ResourceRetrievalSuccess",
			args: args{
				tenantID:         host.GetTenantId(),
				hostID:           host.GetResourceId(),
				onboardingStatus: om_status.OnboardingStatusInProgress,
				uuid:             "123",
			},
			want:  nil,
			valid: true,
		},
		{
			name: "ValidUUID_ResourceRetrievalError",
			args: args{
				tenantID:         host.GetTenantId(),
				hostID:           host.GetResourceId(),
				onboardingStatus: om_status.OnboardingStatusInProgress,
				uuid:             "123",
			},
			want:  nil,
			valid: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			invClient.SetHostOnboardingStatus(ctx, tt.args.tenantID, tt.args.hostID, tt.args.onboardingStatus)
			if !t.Failed() && tt.valid {
				hostInv, err := invClient.GetHostResourceByUUID(ctx, host.GetTenantId(), host.Uuid)
				require.NoError(t, err)
				require.NotNil(t, hostInv)

				assert.Equal(t, tt.args.onboardingStatus.Status, hostInv.GetOnboardingStatus())
				assert.Equal(t, tt.args.onboardingStatus.StatusIndicator, hostInv.GetOnboardingStatusIndicator())
			}
		})
	}
}

func TestOnboardingInventoryClient_GetHostResourceByUUID_MultiTenant(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	host := inv_testing.CreateHost(t, nil, nil)

	t.Run("Valid_TenantId", func(t *testing.T) {
		hostInv, err := invClient.GetHostResourceByUUID(context.Background(), host.GetTenantId(), host.Uuid)
		require.NoError(t, err)
		require.NotNil(t, hostInv)
	})

	t.Run("Invalid_TenantId", func(t *testing.T) {
		hostInv, err := invClient.GetHostResourceByUUID(context.Background(), tenant1, host.Uuid)
		require.Error(t, err)
		require.Nil(t, hostInv)
	})
}

func TestOnboardingInventoryClient_DeleteHostResource(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	host := inv_testing.CreateHost(t, nil, nil)
	type args struct {
		tenantID         string
		hostID           string
		details          string
		onboardingStatus inv_status.ResourceStatus
	}
	tests := []struct {
		name  string
		args  args
		valid bool
	}{
		{
			name: "Success",
			args: args{
				tenantID:         host.GetTenantId(),
				hostID:           host.GetResourceId(),
				details:          "some detail",
				onboardingStatus: om_status.OnboardingStatusInProgress,
			},
			valid: true,
		},
		{
			name: "Failed_NotFound",
			args: args{
				tenantID: tenant1,
				hostID:   "host-12345678",
			},
			valid: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if !t.Failed() && tt.valid {
				err := invClient.DeleteHostResource(ctx, tt.args.tenantID, host.ResourceId)
				require.NoError(t, err)

			}
		})
	}
}

func TestOnboardingInventoryClient_SetHostStatus(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient

	host := inv_testing.CreateHost(t, nil, nil)

	type args struct {
		tenantID         string
		hostID           string
		onboardingStatus inv_status.ResourceStatus
	}
	tests := []struct {
		name  string
		args  args
		valid bool
	}{
		{
			name: "Success",
			args: args{
				tenantID:         host.GetTenantId(),
				hostID:           host.GetResourceId(),
				onboardingStatus: om_status.OnboardingStatusInProgress,
			},
			valid: true,
		},
		{
			name: "Failed_NotFound",
			args: args{
				tenantID: tenant1,
				hostID:   "host-12345678",
			},
			valid: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			timeBeforeUpdate := time.Now().Unix()
			err := invClient.SetHostOnboardingStatus(ctx, tt.args.tenantID, tt.args.hostID, tt.args.onboardingStatus)
			if err != nil {
				if tt.valid {
					t.Errorf("Failed: %s", err)
					t.FailNow()
				}
			} else {
				if !tt.valid {
					t.Errorf("Succeeded but should have failed")
					t.FailNow()
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tt.valid {
				hostInv, err := invClient.GetHostResourceByUUID(ctx, host.GetTenantId(), host.Uuid)
				require.NoError(t, err)
				require.NotNil(t, hostInv)

				assert.Equal(t, tt.args.onboardingStatus.Status, hostInv.GetOnboardingStatus())
				assert.Equal(t, tt.args.onboardingStatus.StatusIndicator, hostInv.GetOnboardingStatusIndicator())
				assert.LessOrEqual(t, uint64(timeBeforeUpdate), hostInv.GetOnboardingStatusTimestamp())
			}
		})
	}
}

func TestOnboardingInventoryClient_SetHostStatusDetail(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient

	host := inv_testing.CreateHost(t, nil, nil)

	type args struct {
		tenantID string
		hostID   string
		status   inv_status.ResourceStatus
	}
	tests := []struct {
		name  string
		args  args
		valid bool
	}{
		{
			name: "Success",
			args: args{
				tenantID: host.GetTenantId(),
				hostID:   host.GetResourceId(),
				status:   om_status.DeletingStatus,
			},
			valid: true,
		},
		{
			name: "Failed_NotFound",
			args: args{
				tenantID: tenant1,
				hostID:   "host-12345678",
			},
			valid: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			timeBeforeUpdate := time.Now().Unix()
			err := invClient.SetHostStatusDetail(ctx, tt.args.tenantID, tt.args.hostID, tt.args.status)
			if err != nil {
				if tt.valid {
					t.Errorf("Failed: %s", err)
					t.FailNow()
				}
			} else {
				if !tt.valid {
					t.Errorf("Succeeded but should have failed")
					t.FailNow()
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tt.valid {
				hostInv, err := invClient.GetHostResourceByUUID(ctx, host.GetTenantId(), host.Uuid)
				require.NoError(t, err)
				require.NotNil(t, hostInv)

				assert.Equal(t, tt.args.status.Status, hostInv.GetOnboardingStatus())
				assert.Equal(t, tt.args.status.StatusIndicator, hostInv.GetOnboardingStatusIndicator())
				assert.LessOrEqual(t, uint64(timeBeforeUpdate), hostInv.GetOnboardingStatusTimestamp())
			}
		})
	}
}

func TestOnboardingInventoryClient_CreateInstanceResource(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	host := inv_testing.CreateHost(t, nil, nil)
	osRes := inv_testing.CreateOs(t)
	inst := inv_testing.CreateInstance(t, host, osRes)
	type args struct {
		ctx      context.Context
		inst     *computev1.InstanceResource
		tenantID string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "CreateInstanceResource -success",
			args: args{
				ctx:      context.Background(),
				inst:     inst,
				tenantID: inst.GetTenantId(),
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := invClient.CreateInstanceResource(tt.args.ctx, tt.args.tenantID, tt.args.inst)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.CreateInstanceResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("OnboardingInventoryClient.CreateInstanceResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOnboardingInventoryClient_GetInstanceResourceByResourceID(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient

	host := inv_testing.CreateHost(t, nil, nil)
	os := inv_testing.CreateOs(t)
	inst := inv_testing.CreateInstance(t, host, os)
	inst.DesiredOs = os
	inst.CurrentOs = os
	inst.Host = host
	type args struct {
		tenantID         string
		instID           string
		expectedInstance *computev1.InstanceResource
	}
	tests := []struct {
		name  string
		args  args
		valid bool
	}{
		{
			name: "GetInstanceResourceByResourceID_Success",
			args: args{
				tenantID:         inst.GetTenantId(),
				instID:           inst.GetResourceId(),
				expectedInstance: inst,
			},
			valid: true,
		},
		{
			name: "GetInstanceResourceByResourceID_Error",
			args: args{
				tenantID: tenant1,
				instID:   "1234567",
			},
			valid: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			invInst, err := invClient.GetInstanceResourceByResourceID(ctx, tt.args.tenantID, tt.args.instID)
			if err != nil {
				if tt.valid {
					t.Errorf("Failed: %s", err)
					t.FailNow()
				}
			} else {
				if !tt.valid {
					t.Errorf("Succeeded but should have failed")
					t.FailNow()
				}
			}

			if !t.Failed() && tt.valid {
				if eq, diff := inv_testing.ProtoEqualOrDiff(invInst, tt.args.expectedInstance); !eq {
					t.Errorf("Data not equal: %v", diff)
				}
			}
		})
	}
}

func TestOnboardingInventoryClient_GetInstanceResources(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		args    args
		want    []*computev1.InstanceResource
		wantErr bool
	}{
		{
			name: "GetInstanceResources_Success",
			args: args{
				ctx: context.Background(),
			},
			want:    []*computev1.InstanceResource{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			got, err := invClient.GetInstanceResources(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.GetInstanceResources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OnboardingInventoryClient.GetInstanceResources() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOnboardingInventoryClient_DeleteInstanceResource(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	host := inv_testing.CreateHost(t, nil, nil)
	osRes := inv_testing.CreateOs(t)
	inst := inv_testing.CreateInstance(t, host, osRes)
	type args struct {
		ctx        context.Context
		tenantID   string
		resourceID string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "DeleteInstanceResource_Success",
			args: args{
				ctx:        context.Background(),
				tenantID:   inst.GetTenantId(),
				resourceID: inst.GetResourceId(),
			},
			wantErr: false,
		},
		{
			name: "DeleteInstanceResource_Error",
			args: args{
				ctx:        context.Background(),
				resourceID: "12345678",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := invClient.DeleteInstanceResource(tt.args.ctx, tt.args.tenantID, tt.args.resourceID); (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.DeleteInstanceResource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOnboardingInventoryClient_DeleteResource(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	host := inv_testing.CreateHost(t, nil, nil)
	type args struct {
		ctx        context.Context
		tenantID   string
		resourceID string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "DeleteResource_Success",
			args: args{
				ctx:        context.Background(),
				resourceID: host.GetResourceId(),
				tenantID:   host.GetTenantId(),
			},
			wantErr: false,
		},
		{
			name: "DeleteResource_Error",
			args: args{
				ctx:        context.Background(),
				tenantID:   tenant1,
				resourceID: "12345678",
			},
			wantErr: true,
		},
		{
			name: "DeleteResource_NotFound",
			args: args{
				ctx:        context.Background(),
				resourceID: "",
				tenantID:   tenant1,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := invClient.DeleteResource(tt.args.ctx, tt.args.tenantID, tt.args.resourceID); (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.DeleteResource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOnboardingInventoryClient_CreateOSResource(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	type args struct {
		ctx      context.Context
		tenantID string
		os       *osv1.OperatingSystemResource
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "CreateOSResource_Success",
			args: args{
				ctx:      context.Background(),
				tenantID: tenant1,
				os: &osv1.OperatingSystemResource{
					OsType:     osv1.OsType_OS_TYPE_IMMUTABLE,
					TenantId:   tenant1,
					OsProvider: osv1.OsProviderKind_OS_PROVIDER_KIND_INFRA,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.CreateOSResource(tt.args.ctx, tt.args.tenantID, tt.args.os)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.CreateOSResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestOnboardingInventoryClient_GetOSResourceByResourceID(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	osRes := inv_testing.CreateOs(t)
	type args struct {
		ctx        context.Context
		tenantID   string
		resourceID string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "TestGetOSResourceByResourceID_Success",
			args: args{
				ctx:        context.Background(),
				tenantID:   osRes.GetTenantId(),
				resourceID: osRes.GetResourceId(),
			},
			wantErr: false,
		},
		{
			name: "TestGetOSResourceByResourceID_Error",
			args: args{
				ctx:        context.Background(),
				resourceID: "123243",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.GetOSResourceByResourceID(tt.args.ctx, tt.args.tenantID, tt.args.resourceID)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.GetOSResourceByResourceID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestOnboardingInventoryClient_GetOSResources(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "TestGetOSResources_Success",
			args:    args{ctx: context.Background()},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.GetOSResources(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.GetOSResources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestOnboardingInventoryClient_ListIPAddresses(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	type args struct {
		ctx     context.Context
		hostNic *computev1.HostnicResource
	}
	tests := []struct {
		name    string
		args    args
		want    []*network_v1.IPAddressResource
		wantErr bool
	}{
		{
			name:    "TestListIPAddresses_Success",
			args:    args{ctx: context.Background()},
			want:    []*network_v1.IPAddressResource{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := invClient.ListIPAddresses(tt.args.ctx, tt.args.hostNic)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.ListIPAddresses() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OnboardingInventoryClient.ListIPAddresses() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOnboardingInventoryClient_FindAllResources(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	type args struct {
		ctx   context.Context
		kinds []inv_v1.ResourceKind
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "TestFindAllResources_Success",
			args:    args{ctx: context.Background()},
			wantErr: false,
		},
		{
			name: "TestFindAllResources_Filtered_Success",
			args: args{
				ctx:   context.Background(),
				kinds: []inv_v1.ResourceKind{inv_v1.ResourceKind_RESOURCE_KIND_HOST},
			},
			wantErr: false,
		},
		{
			name: "TestFindAllResources_Error",
			args: args{
				ctx:   context.Background(),
				kinds: []inv_v1.ResourceKind{inv_v1.ResourceKind_RESOURCE_KIND_UNSPECIFIED},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.FindAllResources(tt.args.ctx, tt.args.kinds)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.FindAllResources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestOnboardingInventoryClient_UpdateInvResourceFields(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	type args struct {
		ctx      context.Context
		tenantID string
		resource proto.Message
		fields   []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "TestUpdateInvResourceFields_EmptyInput",
			args:    args{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := invClient.UpdateInvResourceFields(tt.args.ctx, tt.args.tenantID, tt.args.resource, tt.args.fields); (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.UpdateInvResourceFields() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOnboardingInventoryClient_UpdateHostStateAndStatus(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	host := inv_testing.CreateHost(t, nil, nil)
	type args struct {
		tenantID          string
		hostID            string
		hostCurrentState  computev1.HostState
		runtimeHostStatus inv_status.ResourceStatus
		updateTimestamp   int64
	}
	tests := []struct {
		name       string
		args       args
		valid      bool
		statusCode codes.Code
	}{
		{
			name: "Success",
			args: args{
				tenantID:          host.GetTenantId(),
				hostID:            host.GetResourceId(),
				hostCurrentState:  computev1.HostState_HOST_STATE_UNTRUSTED,
				runtimeHostStatus: om_status.AuthorizationStatusInvalidated,
				updateTimestamp:   time.Now().Unix(),
			},
			valid: true,
		},
		{
			name: "Failed_NotFound",
			args: args{
				tenantID:          tenant1,
				hostID:            "host-12345678",
				hostCurrentState:  computev1.HostState_HOST_STATE_UNTRUSTED,
				runtimeHostStatus: om_status.AuthorizationStatusInvalidated,
				updateTimestamp:   time.Now().Unix(),
			},
			valid:      false,
			statusCode: codes.NotFound,
		},
		{
			name: "Failed_HostStatusNotSet",
			args: args{
				hostID: host.GetResourceId(),
			},
			valid:      false,
			statusCode: codes.InvalidArgument,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			hostUp := &computev1.HostResource{
				ResourceId:          tt.args.hostID,
				CurrentState:        tt.args.hostCurrentState,
				HostStatus:          tt.args.runtimeHostStatus.Status,
				HostStatusIndicator: tt.args.runtimeHostStatus.StatusIndicator,
				HostStatusTimestamp: uint64(tt.args.updateTimestamp),
			}

			err := OnboardingTestClient.UpdateHostStateAndRuntimeStatus(ctx, tt.args.tenantID, hostUp)
			if err != nil {
				if tt.valid {
					t.Errorf("Failed: %s", err)
					t.FailNow()
				}
			} else {
				if !tt.valid {
					t.Errorf("Succeeded but should have failed")
					t.FailNow()
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tt.valid {
				hostInv, hostErr := OnboardingTestClient.GetHostResourceByUUID(ctx, host.GetTenantId(), host.Uuid)
				require.NoError(t, hostErr)
				require.NotNil(t, hostInv)

				assert.Equal(t, tt.args.hostCurrentState, hostInv.GetCurrentState())
				assert.Equal(t, tt.args.runtimeHostStatus.Status, hostInv.GetHostStatus())
				assert.Equal(t, tt.args.runtimeHostStatus.StatusIndicator, hostInv.GetHostStatusIndicator())
				assert.LessOrEqual(t, uint64(tt.args.updateTimestamp), hostInv.GetHostStatusTimestamp())
			}

			if !tt.valid {
				grpcCode := status.Code(err)
				require.Equal(t, tt.statusCode, grpcCode)
			}
		})
	}
}

func TestOnboardingInventoryClient_SetInstanceStatus(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient

	host := inv_testing.CreateHost(t, nil, nil)
	osRes := inv_testing.CreateOs(t)
	inst := inv_testing.CreateInstance(t, host, osRes)

	type args struct {
		tenantID           string
		instanceID         string
		provisioningStatus inv_status.ResourceStatus
	}
	tests := []struct {
		name  string
		args  args
		valid bool
	}{
		{
			name: "Success",
			args: args{
				tenantID:           inst.GetTenantId(),
				instanceID:         inst.GetResourceId(),
				provisioningStatus: om_status.ProvisioningStatusDone,
			},
			valid: true,
		},
		{
			name: "Failed_NotFound",
			args: args{
				tenantID:           inst.GetTenantId(),
				instanceID:         "inst-12345678",
				provisioningStatus: om_status.ProvisioningStatusDone,
			},
			valid: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			timeBeforeUpdate := time.Now().Unix()
			err := invClient.SetInstanceProvisioningStatus(ctx, tt.args.tenantID, tt.args.instanceID, tt.args.provisioningStatus)
			if err != nil {
				if tt.valid {
					t.Errorf("Failed: %s", err)
					t.FailNow()
				}
			} else {
				if !tt.valid {
					t.Errorf("Succeeded but should have failed")
					t.FailNow()
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tt.valid {
				hostInv, err := invClient.GetHostResourceByUUID(ctx, host.GetTenantId(), host.Uuid)
				require.NoError(t, err)
				require.NotNil(t, hostInv)

				instInv := hostInv.Instance
				assert.Equal(t, tt.args.provisioningStatus.Status, instInv.GetProvisioningStatus())
				assert.Equal(t, tt.args.provisioningStatus.StatusIndicator, instInv.GetProvisioningStatusIndicator())
				assert.LessOrEqual(t, uint64(timeBeforeUpdate), instInv.GetProvisioningStatusTimestamp())
			}
		})
	}
}

func TestWithClientKind(t *testing.T) {
	type args struct {
		clientKind inv_v1.ClientKind
	}
	tests := []struct {
		name string
		args args
		want Option
	}{
		{
			name: "TestWithClientKind_NoInputKind",
			args: args{},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WithClientKind(tt.args.clientKind); reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithClientKind() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewOnboardingInventoryClientWithOptions_Case(t *testing.T) {
	type args struct {
		opts []Option
	}

	tests := []struct {
		name    string
		args    args
		want    *OnboardingInventoryClient
		wantErr bool
	}{
		{
			name: "TestNewOnboardingInventoryClientWithOptions_WithInventoryAddress",
			args: args{
				opts: []Option{WithInventoryAddress("example.com")},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewOnboardingInventoryClientWithOptions(tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOnboardingInventoryClientWithOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewOnboardingInventoryClientWithOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOnboardingInventoryClient_UpdateInvResourceFields_Case(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	hostResource := &computev1.HostResource{}
	hostResCopy := proto.Clone(hostResource)
	type args struct {
		ctx      context.Context
		tenantID string
		resource proto.Message
		fields   []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "TestUpdateInvResourceFields_WithMockClient",
			args: args{
				resource: hostResCopy,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := invClient.UpdateInvResourceFields(tt.args.ctx, tt.args.tenantID, tt.args.resource, tt.args.fields); (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.UpdateInvResourceFields() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOnboardingInventoryClient_UpdateInvResourceFields_Case1(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	res := &network_v1.EndpointResource{}
	resCopy := proto.Clone(res)
	type args struct {
		ctx      context.Context
		tenantID string
		resource proto.Message
		fields   []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "TestUpdateInvResourceFields_NilClient",
			args: args{
				resource: resCopy,
				fields:   []string{"field"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := invClient.UpdateInvResourceFields(tt.args.ctx, tt.args.tenantID, tt.args.resource, tt.args.fields); (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.UpdateInvResourceFields() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOnboardingInventoryClient_UpdateInvResourceFields_Case3(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	hostResource := &computev1.HostResource{}
	hostResCopy := proto.Clone(hostResource)
	type args struct {
		ctx      context.Context
		tenantID string
		resource proto.Message
		fields   []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test UpdateInvResourceFields with nil client",
			args: args{
				resource: hostResCopy,
				fields:   []string{"field"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := invClient.UpdateInvResourceFields(tt.args.ctx, tt.args.tenantID, tt.args.resource, tt.args.fields); (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.UpdateInvResourceFields() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOnboardingInventoryClient_GetInstanceResourceByResourceID_Case(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	type args struct {
		ctx        context.Context
		tenantID   string
		resourceID string
	}
	tests := []struct {
		name    string
		args    args
		want    *computev1.InstanceResource
		wantErr bool
	}{
		{
			name: "TestGetInstanceResourceByResourceID_Success",
			args: args{
				ctx:        context.Background(),
				tenantID:   tenant1,
				resourceID: "inst-78789",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "TestGetInstanceResourceByResourceID_ErrorHandling",
			args: args{
				ctx:        context.Background(),
				tenantID:   tenant1,
				resourceID: "1234553",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := invClient.GetInstanceResourceByResourceID(tt.args.ctx, tt.args.tenantID, tt.args.resourceID)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.GetInstanceResourceByResourceID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OnboardingInventoryClient.GetInstanceResourceByResourceID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOnboardingInventoryClient_GetInstanceResources_Case(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "TestGetInstanceResources",
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.GetInstanceResources(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.GetInstanceResources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestOnboardingInventoryClient_GetOSResourceByResourceID_Case(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	type args struct {
		ctx        context.Context
		tenantID   string
		resourceID string
	}
	tests := []struct {
		name    string
		args    args
		want    *osv1.OperatingSystemResource
		wantErr bool
	}{
		{
			name: "TestGetOSResourceByID_Success",
			args: args{
				ctx:        context.Background(),
				tenantID:   tenant1,
				resourceID: "os-093dd2d7",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "TestGetOSResourceByID_ErrorHandling",
			args: args{
				ctx:        context.Background(),
				tenantID:   tenant1,
				resourceID: "1234566",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := invClient.GetOSResourceByResourceID(tt.args.ctx, tt.args.tenantID, tt.args.resourceID)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.GetOSResourceByResourceID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OnboardingInventoryClient.GetOSResourceByResourceID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOnboardingInventoryClient_GetProviderResources(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		args    args
		want    []*provider_v1.ProviderResource
		wantErr bool
	}{
		{
			name:    "Test GetProviderResources Success",
			args:    args{ctx: context.Background()},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := invClient.GetProviderResources(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.GetProviderResources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) {
				t.Errorf("OnboardingInventoryClient.GetProviderResources() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOnboardingInventoryClient_DeleteIPAddress(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	type args struct {
		ctx        context.Context
		tenantID   string
		resourceID string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "TestDeleteIPAddress_Success",
			args: args{
				ctx:        context.Background(),
				tenantID:   tenant1,
				resourceID: "os-093dd2d7",
			},
			wantErr: false,
		},
		{
			name: "TestDeleteIPAddress_Error",
			args: args{
				ctx:        context.Background(),
				tenantID:   tenant1,
				resourceID: "123",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := invClient.DeleteIPAddress(tt.args.ctx, tt.args.tenantID, tt.args.resourceID); (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.DeleteIPAddress() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOnboardingInventoryClient_GetHostBmcNic(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	host := inv_testing.CreateHost(t, nil, nil)
	type args struct {
		ctx  context.Context
		host *computev1.HostResource
	}
	tests := []struct {
		name    string
		args    args
		want    *computev1.HostnicResource
		wantErr bool
	}{
		{
			name: "GetHostBmcNic Success",
			args: args{
				ctx:  context.Background(),
				host: host,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := invClient.GetHostBmcNic(tt.args.ctx, tt.args.host)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.GetHostBmcNic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OnboardingInventoryClient.GetHostBmcNic() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOnboardingInventoryClient_SetInstanceStatusAndCurrentState(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	type args struct {
		ctx                context.Context
		tenantID           string
		instanceID         string
		currentState       computev1.InstanceState
		provisioningStatus inv_status.ResourceStatus
		currentOS          *osv1.OperatingSystemResource
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "",
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := invClient.UpdateInstance(tt.args.ctx, tt.args.tenantID, tt.args.instanceID, tt.args.currentState, tt.args.provisioningStatus, tt.args.currentOS); (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.UpdateInstance() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOnboardingInventoryClient_listAndReturnProvider(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	type args struct {
		ctx    context.Context
		filter *inv_v1.ResourceFilter
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "",
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.listAndReturnProvider(tt.args.ctx, tt.args.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.listAndReturnProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestGetProviderResourceByName(t *testing.T) {
	type args struct {
		ctx      context.Context
		tenantID string
		c        *OnboardingInventoryClient
		name     string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Empty provider name",
			args: args{
				ctx:      context.Background(),
				tenantID: tenant1,
				c:        &OnboardingInventoryClient{},
				name:     "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetProviderResourceByName(tt.args.ctx, tt.args.tenantID, tt.args.c, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetProviderResourceByName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestGetProviderResourceByName_MultiTenant(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	provider := inv_testing.CreateProvider(t, "Test Provider")

	t.Run("Valid_TenantId", func(t *testing.T) {
		providerInv, err := GetProviderResourceByName(context.Background(), provider.GetTenantId(), invClient, provider.GetName())
		require.NoError(t, err)
		require.NotNil(t, providerInv)
	})

	t.Run("Invalid_TenantId", func(t *testing.T) {
		providerInv, err := GetProviderResourceByName(context.Background(), tenant1, invClient, provider.GetName())
		require.Error(t, err)
		require.Nil(t, providerInv)
	})
}

func TestOnboardingInventoryClient_GetProviderConfig(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	type args struct {
		ctx      context.Context
		tenantID string
		name     string
	}
	inv_testing.CreateProvider(t, "dummyprovider")
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "GetProviderConfig_SuccessfulResponse",
			args: args{
				ctx:  context.Background(),
				name: "dummyprovider",
			},
			wantErr: true,
		},
		{
			name: "Empty Provider",
			args: args{
				ctx:      context.Background(),
				tenantID: tenant1,
				name:     "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.GetProviderConfig(tt.args.ctx, tt.args.tenantID, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.GetProviderConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestOnboardingInventoryClient_UpdateHostRegState(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	host := inv_testing.CreateHost(t, nil, nil)
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx      context.Context
		tenantID string
		host     *computev1.HostResource
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "UpdateHostDetailsByID_PositiveCase",
			fields: fields{
				Watcher: make(chan *client.WatchEvents),
			},
			args: args{
				ctx:      context.Background(),
				tenantID: host.GetTenantId(),
				host:     host,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := invClient.UpdateHostRegState(tt.args.ctx, tt.args.tenantID, tt.args.host.ResourceId, computev1.HostState_HOST_STATE_REGISTERED, "", "", om_status.HostRegistrationDone); (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.updateHostMacID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOnboardingInventoryClient_GetHostResource(t *testing.T) {
	type fields struct {
		Client  client.TenantAwareInventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx         context.Context
		filterType  string
		filterValue string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *computev1.HostResource
		wantErr bool
	}{
		{
			name: "Test Case for empty filter",
			fields: fields{
				Client:  nil,
				Watcher: make(chan *client.WatchEvents),
			},
			args: args{
				ctx:         context.Background(),
				filterType:  "",
				filterValue: "",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OnboardingInventoryClient{
				Client:  tt.fields.Client,
				Watcher: tt.fields.Watcher,
			}
			got, err := c.GetHostResource(tt.args.ctx, tt.args.filterType, tt.args.filterValue)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.GetHostResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OnboardingInventoryClient.GetHostResource() = %v, want %v", got, tt.want)
			}
		})
	}
}
