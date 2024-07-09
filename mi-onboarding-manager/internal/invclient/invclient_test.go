// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

package invclient

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	om_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/status"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	network_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/network/v1"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/os/v1"
	provider_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/provider/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/client"
	inv_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/status"
	inv_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
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
		invClient client.InventoryClient
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
	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	type args struct {
		ctx  context.Context
		host *computev1.HostResource
	}
	tests := []struct {
		name  string
		args  args
		valid bool
	}{
		{
			name: "UpdatingHostResource",
			args: args{
				ctx:  context.Background(),
				host: host,
			},
			valid: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err := invClient.UpdateHostResource(ctx, tt.args.host)
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
				hostInv, err := invClient.GetHostResourceByUUID(ctx, host.Uuid)
				require.NoError(t, err)
				require.NotNil(t, hostInv)
			}
		})
	}
}

func TestOnboardingInventoryClient_GetHostResources(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
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
				hostInv, err := invClient.GetHostResourceByUUID(ctx, host.Uuid)
				require.NoError(t, err)
				require.NotNil(t, hostInv)
			}
		})
	}
}

func TestOnboardingInventoryClient_FindAllInstances(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient

	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	type args struct {
		hostID         string
		instanceStatus []string
		details        string
		ctx            context.Context
	}
	tests := []struct {
		name  string
		args  args
		want  []string
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
	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	type args struct {
		hostID           string
		hostStatus       computev1.HostStatus
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
				hostID:           host.GetResourceId(),
				hostStatus:       computev1.HostStatus_HOST_STATUS_UNSPECIFIED,
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
			hostInv, err := invClient.GetHostResourceByResourceID(ctx, host.ResourceId)
			require.NoError(t, err)
			require.NotNil(t, hostInv)
			assert.Equal(t, tt.args.hostStatus, hostInv.GetLegacyHostStatus())
		})
	}
	t.Run("Invalid Resource Id", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		hostInv, err := invClient.GetHostResourceByResourceID(ctx, "12345")
		require.Error(t, err)
		require.Nil(t, hostInv)
	})
}

func TestOnboardingInventoryClient_CreateHostResource(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	type args struct {
		hostID           string
		hostStatus       computev1.HostStatus
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
				hostID:           host.GetResourceId(),
				hostStatus:       computev1.HostStatus_HOST_STATUS_ONBOARDING,
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
			err := invClient.SetHostStatus(ctx, tt.args.hostID, tt.args.hostStatus, tt.args.details, tt.args.onboardingStatus)
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
				hostInv, err := invClient.CreateHostResource(ctx, &computev1.HostResource{
					Uuid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
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
	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	type args struct {
		hostID           string
		hostStatus       computev1.HostStatus
		details          string
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
				hostID:           host.GetResourceId(),
				hostStatus:       computev1.HostStatus_HOST_STATUS_ONBOARDING,
				details:          "some detail",
				onboardingStatus: om_status.OnboardingStatusInProgress,
				uuid:             "123",
			},
			want:  nil,
			valid: false,
		},
		{
			name: "MissingUUID_ResourceRetrievalError",
			args: args{
				hostID:           host.GetResourceId(),
				hostStatus:       computev1.HostStatus_HOST_STATUS_ONBOARDING,
				details:          "some detail",
				onboardingStatus: om_status.OnboardingStatusInProgress,
			},
			want:  nil,
			valid: true,
		},
		{
			name: "ValidUUID_ResourceRetrievalSuccess",
			args: args{
				hostID:           host.GetResourceId(),
				hostStatus:       computev1.HostStatus_HOST_STATUS_ONBOARDING,
				details:          "some detail",
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
				hostID:           host.GetResourceId(),
				hostStatus:       computev1.HostStatus_HOST_STATUS_ONBOARDING,
				details:          "some detail",
				onboardingStatus: om_status.OnboardingStatusInProgress,
				uuid:             "123",
			},
			want:  nil,
			valid: true,
		},
		{
			name: "ValidUUID_ResourceRetrievalError",
			args: args{
				hostID:           host.GetResourceId(),
				hostStatus:       computev1.HostStatus_HOST_STATUS_ONBOARDING,
				details:          "some detail",
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

			invClient.SetHostStatus(ctx, tt.args.hostID, tt.args.hostStatus, tt.args.details, tt.args.onboardingStatus)
			if !t.Failed() && tt.valid {
				hostInv, err := invClient.GetHostResourceByUUID(ctx, host.Uuid)
				require.NoError(t, err)
				require.NotNil(t, hostInv)

				assert.Equal(t, tt.args.hostStatus, hostInv.GetLegacyHostStatus())
				assert.Equal(t, tt.args.onboardingStatus.Status, hostInv.GetOnboardingStatus())
				assert.Equal(t, tt.args.onboardingStatus.StatusIndicator, hostInv.GetOnboardingStatusIndicator())
			}
		})
	}
}

func TestOnboardingInventoryClient_DeleteHostResource(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	type args struct {
		hostID           string
		hostStatus       computev1.HostStatus
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
				hostID:           host.GetResourceId(),
				hostStatus:       computev1.HostStatus_HOST_STATUS_ONBOARDING,
				details:          "some detail",
				onboardingStatus: om_status.OnboardingStatusInProgress,
			},
			valid: true,
		},
		{
			name: "Failed_NotFound",
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
			if !t.Failed() && tt.valid {
				err := invClient.DeleteHostResource(ctx, host.ResourceId)
				require.NoError(t, err)

			}
		})
	}
}

func TestOnboardingInventoryClient_SetHostStatus(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient

	host := inv_testing.CreateHost(t, nil, nil, nil, nil)

	type args struct {
		hostID           string
		hostStatus       computev1.HostStatus
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
				hostID:           host.GetResourceId(),
				hostStatus:       computev1.HostStatus_HOST_STATUS_ONBOARDING,
				details:          "some detail",
				onboardingStatus: om_status.OnboardingStatusInProgress,
			},
			valid: true,
		},
		{
			name: "Failed_NotFound",
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

			timeBeforeUpdate := time.Now().Unix()
			err := invClient.SetHostStatus(ctx, tt.args.hostID, tt.args.hostStatus, tt.args.details, tt.args.onboardingStatus)
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
				hostInv, err := invClient.GetHostResourceByUUID(ctx, host.Uuid)
				require.NoError(t, err)
				require.NotNil(t, hostInv)

				assert.Equal(t, tt.args.hostStatus, hostInv.GetLegacyHostStatus())
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

	host := inv_testing.CreateHost(t, nil, nil, nil, nil)

	type args struct {
		hostID       string
		statusDetail string
		status       inv_status.ResourceStatus
	}
	tests := []struct {
		name  string
		args  args
		valid bool
	}{
		{
			name: "Success",
			args: args{
				hostID:       host.GetResourceId(),
				statusDetail: "some detail",
				status:       om_status.DeletingStatus,
			},
			valid: true,
		},
		{
			name: "Failed_NotFound",
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

			timeBeforeUpdate := time.Now().Unix()
			err := invClient.SetHostStatusDetail(ctx, tt.args.hostID, tt.args.statusDetail, tt.args.status)
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
				hostInv, err := invClient.GetHostResourceByUUID(ctx, host.Uuid)
				require.NoError(t, err)
				require.NotNil(t, hostInv)

				assert.Equal(t, tt.args.statusDetail, hostInv.GetProviderStatusDetail())
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
	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	osRes := inv_testing.CreateOs(t)
	inst := inv_testing.CreateInstance(t, host, osRes)
	type args struct {
		ctx  context.Context
		inst *computev1.InstanceResource
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
				ctx:  context.Background(),
				inst: inst,
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := invClient.CreateInstanceResource(tt.args.ctx, tt.args.inst)
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

	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	os := inv_testing.CreateOs(t)
	inst := inv_testing.CreateInstance(t, host, os)
	type args struct {
		instID     string
		instStatus computev1.InstanceStatus
	}
	tests := []struct {
		name  string
		args  args
		valid bool
	}{
		{
			name: "GetInstanceResourceByResourceID_Success",
			args: args{
				instID:     inst.GetResourceId(),
				instStatus: computev1.InstanceStatus_INSTANCE_STATUS_UNSPECIFIED,
			},
			valid: true,
		},
		{
			name: "GetInstanceResourceByResourceID_Error",
			args: args{
				instID:     "1234567",
				instStatus: computev1.InstanceStatus_INSTANCE_STATUS_UNSPECIFIED,
			},
			valid: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			InstanceResource, err := invClient.GetInstanceResourceByResourceID(ctx, tt.args.instID)
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
				assert.Equal(t, tt.args.instStatus, InstanceResource.GetStatus())
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
	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	osRes := inv_testing.CreateOs(t)
	inst := inv_testing.CreateInstance(t, host, osRes)
	type args struct {
		ctx        context.Context
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
			if err := invClient.DeleteInstanceResource(tt.args.ctx, tt.args.resourceID); (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.DeleteInstanceResource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOnboardingInventoryClient_DeleteResource(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	type args struct {
		ctx        context.Context
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
			},
			wantErr: false,
		},
		{
			name: "DeleteResource_Error",
			args: args{
				ctx:        context.Background(),
				resourceID: "12345678",
			},
			wantErr: true,
		},
		{
			name: "DeleteResource_NotFound",
			args: args{
				ctx:        context.Background(),
				resourceID: "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := invClient.DeleteResource(tt.args.ctx, tt.args.resourceID); (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.DeleteResource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOnboardingInventoryClient_CreateOSResource(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	type args struct {
		ctx context.Context
		os  *osv1.OperatingSystemResource
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "CreateOSResource_Success",
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.CreateOSResource(tt.args.ctx, tt.args.os)
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
			_, err := invClient.GetOSResourceByResourceID(tt.args.ctx, tt.args.resourceID)
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
			if err := invClient.UpdateInvResourceFields(tt.args.ctx, tt.args.resource, tt.args.fields); (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.UpdateInvResourceFields() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOnboardingInventoryClient_UpdateHostStateAndStatus(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	type args struct {
		hostID               string
		hostCurrentState     computev1.HostState
		runtimeHostStatus    inv_status.ResourceStatus
		updateTimestamp      int64
		legacyHostStatus     computev1.HostStatus
		providerStatus       string
		providerStatusDetail string
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
				hostID:               host.GetResourceId(),
				hostCurrentState:     computev1.HostState_HOST_STATE_UNTRUSTED,
				runtimeHostStatus:    om_status.AuthorizationStatusInvalidated,
				legacyHostStatus:     computev1.HostStatus_HOST_STATUS_INVALIDATED,
				providerStatus:       "some status",
				providerStatusDetail: "some detail",
				updateTimestamp:      time.Now().Unix(),
			},
			valid: true,
		},
		{
			name: "Failed_NotFound",
			args: args{
				hostID:               "host-12345678",
				hostCurrentState:     computev1.HostState_HOST_STATE_UNTRUSTED,
				runtimeHostStatus:    om_status.AuthorizationStatusInvalidated,
				legacyHostStatus:     computev1.HostStatus_HOST_STATUS_INVALIDATED,
				providerStatus:       "some status",
				providerStatusDetail: "some detail",
				updateTimestamp:      time.Now().Unix(),
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
				ResourceId:           tt.args.hostID,
				CurrentState:         tt.args.hostCurrentState,
				LegacyHostStatus:     tt.args.legacyHostStatus,
				ProviderStatus:       tt.args.providerStatus,
				ProviderStatusDetail: tt.args.providerStatusDetail,
				HostStatus:           tt.args.runtimeHostStatus.Status,
				HostStatusIndicator:  tt.args.runtimeHostStatus.StatusIndicator,
				HostStatusTimestamp:  uint64(tt.args.updateTimestamp),
			}

			err := OnboardingTestClient.UpdateHostStateAndRuntimeStatus(ctx, hostUp)
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
				hostInv, hostErr := OnboardingTestClient.GetHostResourceByUUID(ctx, host.Uuid)
				require.NoError(t, hostErr)
				require.NotNil(t, hostInv)

				assert.Equal(t, tt.args.hostCurrentState, hostInv.GetCurrentState())
				assert.Equal(t, tt.args.legacyHostStatus, hostInv.GetLegacyHostStatus())
				assert.Equal(t, tt.args.providerStatus, hostInv.GetProviderStatus())
				assert.Equal(t, tt.args.providerStatusDetail, hostInv.GetProviderStatusDetail())
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

	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	osRes := inv_testing.CreateOs(t)
	inst := inv_testing.CreateInstance(t, host, osRes)

	type args struct {
		instanceID         string
		instanceStatus     computev1.InstanceStatus
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
				instanceID:         inst.GetResourceId(),
				instanceStatus:     computev1.InstanceStatus_INSTANCE_STATUS_PROVISIONED,
				provisioningStatus: om_status.ProvisioningStatusDone,
			},
			valid: true,
		},
		{
			name: "Failed_NotFound",
			args: args{
				instanceID:         "inst-12345678",
				instanceStatus:     computev1.InstanceStatus_INSTANCE_STATUS_PROVISIONED,
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
			err := invClient.SetInstanceStatus(ctx, tt.args.instanceID, tt.args.instanceStatus, tt.args.provisioningStatus)
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
				hostInv, err := invClient.GetHostResourceByUUID(ctx, host.Uuid)
				require.NoError(t, err)
				require.NotNil(t, hostInv)

				instInv := hostInv.Instance
				assert.Equal(t, tt.args.instanceStatus, instInv.GetStatus())
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
			if err := invClient.UpdateInvResourceFields(tt.args.ctx, tt.args.resource, tt.args.fields); (err != nil) != tt.wantErr {
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
			if err := invClient.UpdateInvResourceFields(tt.args.ctx, tt.args.resource, tt.args.fields); (err != nil) != tt.wantErr {
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
			if err := invClient.UpdateInvResourceFields(tt.args.ctx, tt.args.resource, tt.args.fields); (err != nil) != tt.wantErr {
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
				resourceID: "inst-78789",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "TestGetInstanceResourceByResourceID_ErrorHandling",
			args: args{
				ctx:        context.Background(),
				resourceID: "1234553",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := invClient.GetInstanceResourceByResourceID(tt.args.ctx, tt.args.resourceID)
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
				resourceID: "os-093dd2d7",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "TestGetOSResourceByID_ErrorHandling",
			args: args{
				ctx:        context.Background(),
				resourceID: "1234566",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := invClient.GetOSResourceByResourceID(tt.args.ctx, tt.args.resourceID)
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
				resourceID: "os-093dd2d7",
			},
			wantErr: false,
		},
		{
			name: "TestDeleteIPAddress_Error",
			args: args{
				ctx:        context.Background(),
				resourceID: "123",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := invClient.DeleteIPAddress(tt.args.ctx, tt.args.resourceID); (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.DeleteIPAddress() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOnboardingInventoryClient_GetHostBmcNic(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
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
		instanceID         string
		currentState       computev1.InstanceState
		instanceStatus     computev1.InstanceStatus
		provisioningStatus inv_status.ResourceStatus
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
			if err := invClient.SetInstanceStatusAndCurrentState(tt.args.ctx, tt.args.instanceID, tt.args.currentState, tt.args.instanceStatus, tt.args.provisioningStatus); (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.SetInstanceStatusAndCurrentState() error = %v, wantErr %v", err, tt.wantErr)
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
		ctx  context.Context
		c    *OnboardingInventoryClient
		name string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Empty provider name",
			args: args{
				ctx:  context.Background(),
				c:    &OnboardingInventoryClient{},
				name: "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetProviderResourceByName(tt.args.ctx, tt.args.c, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetProviderResourceByName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestOnboardingInventoryClient_GetProviderConfig(t *testing.T) {
	CreateOnboardingClientForTesting(t)
	invClient := OnboardingTestClient
	type args struct {
		ctx  context.Context
		name string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "Empty Provider",
			args:    args{
				ctx:  context.Background(),
				name: "",
			},
			wantErr: true,
		},

	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.GetProviderConfig(tt.args.ctx, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.GetProviderConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

