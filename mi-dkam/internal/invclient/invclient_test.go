// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

package invclient

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/google/uuid"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/os/v1"
	provider_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/provider/v1"
	statusv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/status/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/client"
	inv_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/status"
	inv_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/testing"
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

func TestNewDKAMInventoryClientWithOptions(t *testing.T) {
	type args struct {
		opts []Option
	}
	tests := []struct {
		name    string
		args    args
		want    *DKAMInventoryClient
		wantErr bool
	}{
		{
			name:    "WithOutOptions",
			args:    args{},
			want:    nil,
			wantErr: true,
		},
		{
			name: "WithOptions",
			args: args{
				opts: []Option{WithInventoryAddress("example.com")},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDKAMInventoryClientWithOptions(tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDKAMInventoryClientWithOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDKAMInventoryClientWithOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestNewDKAMInventoryClient(t *testing.T) {
	type args struct {
		invClient client.InventoryClient
		watcher   chan *client.WatchEvents
	}
	tests := []struct {
		name    string
		args    args
		want    *DKAMInventoryClient
		wantErr bool
	}{
		{
			name:    "CreatingNewOnboardingInventoryClient",
			args:    args{},
			want:    &DKAMInventoryClient{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDKAMInventoryClient(tt.args.invClient, tt.args.watcher)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDKAMInventoryClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDKAMInventoryClient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDKAMInventoryClient_Close(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	invClient.Close()
}

func TestDKAMInventoryClient_UpdateHostResource(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	type args struct {
		ctx  context.Context
		host *computev1.HostResource
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Updating host",
			args: args{
				ctx:  context.Background(),
				host: host,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := invClient.UpdateHostResource(tt.args.ctx, tt.args.host); (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.UpdateHostResource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDKAMInventoryClient_GetHostResources(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name        string
		args        args
		wantHostres []*computev1.HostResource
		wantErr     bool
	}{
		{
			name: "GetHostResources",
			args: args{
				ctx: context.Background(),
			},
			wantHostres: nil,
			wantErr:     false,
		},
		{
			name: "GetHostResources",
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
			},
			wantHostres: nil,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.GetHostResources(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.GetHostResources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestDKAMInventoryClient_GetHostResourceByResourceID(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	type args struct {
		ctx        context.Context
		resourceID string
	}
	tests := []struct {
		name    string
		args    args
		want    *computev1.HostResource
		wantErr bool
	}{
		{
			name: "GetHostResourceByResourceID",
			args: args{
				ctx:        context.Background(),
				resourceID: host.ResourceId,
			},
			want:    host,
			wantErr: false,
		},
		{
			name: "InvalidResourceID",
			args: args{
				ctx:        context.Background(),
				resourceID: "",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.GetHostResourceByResourceID(tt.args.ctx, tt.args.resourceID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.GetHostResourceByResourceID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestDKAMInventoryClient_GetHostBmcNic(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
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
			name: "GetHostBmcNic",
			args: args{
				ctx:  context.Background(),
				host: host,
			},
			want:    &computev1.HostnicResource{},
			wantErr: true,
		},
		{
			name: "GetHostBmcNic Failure",
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
				host: host,
			},
			want:    &computev1.HostnicResource{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.GetHostBmcNic(tt.args.ctx, tt.args.host)
			if (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.GetHostBmcNic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestDKAMInventoryClient_GetHostResourceByUUID(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	type args struct {
		ctx  context.Context
		uuid string
	}
	tests := []struct {
		name    string
		args    args
		want    *computev1.HostResource
		wantErr bool
	}{
		{
			name: "GetHostResourceByUUID",
			args: args{
				ctx:  context.Background(),
				uuid: host.Uuid,
			},
			want:    &computev1.HostResource{},
			wantErr: false,
		},
		{
			name: "InvalidUUID",
			args: args{
				ctx:  context.Background(),
				uuid: "",
			},
			want:    &computev1.HostResource{},
			wantErr: true,
		},
		{
			name: "InvalidUUID",
			args: args{
				ctx:  context.Background(),
				uuid: uuid.New().String(),
			},
			want:    &computev1.HostResource{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.GetHostResourceByUUID(tt.args.ctx, tt.args.uuid)
			if (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.GetHostResourceByUUID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestDKAMInventoryClient_UpdateHostStateAndRuntimeStatus(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	type args struct {
		ctx  context.Context
		host *computev1.HostResource
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "UpdateHostStateAndRuntimeStatus",
			args: args{
				ctx:  context.Background(),
				host: host,
			},
			wantErr: true,
		},
		{
			name: "UpdateHostStateAndRuntimeStatus Successful",
			args: args{
				ctx: context.Background(),
				host: &computev1.HostResource{
					HostStatus:          "status",
					HostStatusTimestamp: 123,
					HostStatusIndicator: statusv1.StatusIndication_STATUS_INDICATION_IDLE,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := invClient.UpdateHostStateAndRuntimeStatus(tt.args.ctx, tt.args.host); (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.UpdateHostStateAndRuntimeStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDKAMInventoryClient_SetHostStatus(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	type args struct {
		ctx              context.Context
		hostID           string
		hostStatus       computev1.HostStatus
		statusDetails    string
		onboardingStatus inv_status.ResourceStatus
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "SetHostStatus",
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := invClient.SetHostStatus(tt.args.ctx, tt.args.hostID, tt.args.hostStatus, tt.args.statusDetails, tt.args.onboardingStatus); (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.SetHostStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDKAMInventoryClient_SetHostStatusDetail(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	type args struct {
		ctx              context.Context
		hostID           string
		statusDetail     string
		onboardingStatus inv_status.ResourceStatus
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "SetHostStatusDetail",
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := invClient.SetHostStatusDetail(tt.args.ctx, tt.args.hostID, tt.args.statusDetail, tt.args.onboardingStatus); (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.SetHostStatusDetail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDKAMInventoryClient_DeleteHostResource(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
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
			name: "DeleteHostResource",
			args: args{
				ctx:        context.Background(),
				resourceID: host.ResourceId,
			},
			wantErr: false,
		},
		{
			name: "DeleteHostResource Failure",
			args: args{
				ctx:        context.Background(),
				resourceID: "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := invClient.DeleteHostResource(tt.args.ctx, tt.args.resourceID); (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.DeleteHostResource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDKAMInventoryClient_CreateInstanceResource(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	os := inv_testing.CreateOs(t)
	instance := inv_testing.CreateInstance(t, host, os)
	type args struct {
		ctx  context.Context
		inst *computev1.InstanceResource
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "CreateInstanceResource",
			args: args{
				ctx:  context.Background(),
				inst: instance,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.CreateInstanceResource(tt.args.ctx, tt.args.inst)
			if (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.CreateInstanceResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

		})
	}
}

func TestDKAMInventoryClient_GetInstanceResourceByResourceID(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	os := inv_testing.CreateOs(t)
	instance := inv_testing.CreateInstance(t, host, os)
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
			name: "GetInstanceResourceByResourceID",
			args: args{
				ctx:        context.Background(),
				resourceID: instance.ResourceId,
			},
			want:    &computev1.InstanceResource{},
			wantErr: false,
		},
		{
			name: "GetInstanceResourceByResourceID Failure",
			args: args{
				ctx:        context.Background(),
				resourceID: "",
			},
			want:    &computev1.InstanceResource{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.GetInstanceResourceByResourceID(tt.args.ctx, tt.args.resourceID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.GetInstanceResourceByResourceID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestDKAMInventoryClient_GetInstanceResources(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
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
			name: "GetInstanceResources",
			args: args{
				ctx: context.Background(),
			},
			want:    []*computev1.InstanceResource{},
			wantErr: false,
		},
		{
			name: "GetInstanceResources Failure",
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
			},
			want:    []*computev1.InstanceResource{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.GetInstanceResources(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.GetInstanceResources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestDKAMInventoryClient_UpdateInstanceCurrentState(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	type args struct {
		ctx      context.Context
		instance *computev1.InstanceResource
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "UpdateInstanceCurrentState",
			args: args{
				ctx:      context.Background(),
				instance: &computev1.InstanceResource{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := invClient.UpdateInstanceCurrentState(tt.args.ctx, tt.args.instance); (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.UpdateInstanceCurrentState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDKAMInventoryClient_FindAllInstances(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "FindAllInstances",
			args: args{
				ctx: context.Background(),
			},
			want:    []string{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.FindAllInstances(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.FindAllInstances() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestDKAMInventoryClient_CreateHostResource(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	type args struct {
		ctx  context.Context
		host *computev1.HostResource
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "CreateHostResource Failure",
			args: args{
				ctx:  context.Background(),
				host: host,
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.CreateHostResource(tt.args.ctx, tt.args.host)
			if (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.CreateHostResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestDKAMInventoryClient_SetInstanceStatus(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	type args struct {
		ctx                context.Context
		instanceID         string
		instanceStatus     computev1.InstanceStatus
		provisioningStatus inv_status.ResourceStatus
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "SetInstanceStatus",
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := invClient.SetInstanceStatus(tt.args.ctx, tt.args.instanceID, tt.args.instanceStatus, tt.args.provisioningStatus); (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.SetInstanceStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDKAMInventoryClient_SetInstanceStatusAndCurrentState(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
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
			name: "SetInstanceStatusAndCurrentState",
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := invClient.SetInstanceStatusAndCurrentState(tt.args.ctx, tt.args.instanceID, tt.args.currentState, tt.args.instanceStatus, tt.args.provisioningStatus); (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.SetInstanceStatusAndCurrentState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDKAMInventoryClient_DeleteInstanceResource(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	os := inv_testing.CreateOs(t)
	instance := inv_testing.CreateInstance(t, host, os)
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
			name: "DeleteInstanceResource Failure",
			args: args{
				ctx:        context.Background(),
				resourceID: "",
			},
			wantErr: true,
		},
		{
			name: "DeleteInstanceResource",
			args: args{
				ctx:        context.Background(),
				resourceID: instance.ResourceId,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := invClient.DeleteInstanceResource(tt.args.ctx, tt.args.resourceID); (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.DeleteInstanceResource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDKAMInventoryClient_DeleteResource(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
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
			name: "DeleteResource Failure",
			args: args{
				ctx:        context.Background(),
				resourceID: "",
			},
			wantErr: true,
		},
		{
			name: "DeleteResource",
			args: args{
				ctx:        context.Background(),
				resourceID: host.ResourceId,
			},
			wantErr: false,
		},
		{
			name: "Invalid Resource Id",
			args: args{
				ctx:        context.Background(),
				resourceID: "host-084d9b08",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := invClient.DeleteResource(tt.args.ctx, tt.args.resourceID); (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.DeleteResource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDKAMInventoryClient_CreateOSResource(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	type args struct {
		ctx context.Context
		os  *osv1.OperatingSystemResource
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "CreateOSResource",
			args: args{
				ctx: context.Background(),
				os:  &osv1.OperatingSystemResource{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.CreateOSResource(tt.args.ctx, tt.args.os)
			if (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.CreateOSResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestDKAMInventoryClient_GetOSResourceByResourceID(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	os := inv_testing.CreateOs(t)
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
			name: "GetOSResourceByResourceID Failure",
			args: args{
				ctx:        context.Background(),
				resourceID: "",
			},
			wantErr: true,
		},
		{
			name: "GetOSResourceByResourceID",
			args: args{
				ctx:        context.Background(),
				resourceID: os.ResourceId,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.GetOSResourceByResourceID(tt.args.ctx, tt.args.resourceID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.GetOSResourceByResourceID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestDKAMInventoryClient_GetOSResources(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		args    args
		want    []*osv1.OperatingSystemResource
		wantErr bool
	}{
		{
			name: "GetOSResources",
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
		{
			name: "GetOSResources Failure",
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.GetOSResources(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.GetOSResources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestDKAMInventoryClient_ListIPAddresses(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	type args struct {
		ctx     context.Context
		hostNic *computev1.HostnicResource
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ListIPAddresses",
			args: args{
				ctx:     context.Background(),
				hostNic: &computev1.HostnicResource{},
			},
			wantErr: false,
		},
		{
			name: "ListIPAddresses Failure",
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
				hostNic: &computev1.HostnicResource{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.ListIPAddresses(tt.args.ctx, tt.args.hostNic)
			if (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.ListIPAddresses() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestDKAMInventoryClient_FindAllResources(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
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
			name: "FindAllResources",
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
		{
			name: "FindAllResources_Success",
			args: args{
				ctx:   context.Background(),
				kinds: []inv_v1.ResourceKind{inv_v1.ResourceKind_RESOURCE_KIND_HOST},
			},
			wantErr: false,
		},
		{
			name: "FindAllResources_Error",
			args: args{
				ctx:   context.Background(),
				kinds: []inv_v1.ResourceKind{inv_v1.ResourceKind_RESOURCE_KIND_UNSPECIFIED},
			},
			wantErr: true,
		},
		{
			name: "FindAllResources Failure",
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
				kinds: []inv_v1.ResourceKind{inv_v1.ResourceKind_RESOURCE_KIND_HOST},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.FindAllResources(tt.args.ctx, tt.args.kinds)
			if (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.FindAllResources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestDKAMInventoryClient_GetProviderResources(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "GetProviderResources",
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
		{
			name: "GetProviderResources Failure",
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.GetProviderResources(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.GetProviderResources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestDKAMInventoryClient_DeleteIPAddress(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
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
			name: "DeleteIPAddress Failure",
			args: args{
				ctx:        context.Background(),
				resourceID: "",
			},
			wantErr: true,
		},
		{
			name: "DeleteIPAddress",
			args: args{
				ctx:        context.Background(),
				resourceID: host.ResourceId,
			},
			wantErr: false,
		},
		{
			name: "Invalid ResourceId",
			args: args{
				ctx:        context.Background(),
				resourceID: "host-084d9b08",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := invClient.DeleteIPAddress(tt.args.ctx, tt.args.resourceID); (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.DeleteIPAddress() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetProviderResourceByName(t *testing.T) {
	type args struct {
		ctx  context.Context
		c    *DKAMInventoryClient
		name string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "GetProviderResourceByName Failure",
			args: args{
				ctx:  context.Background(),
				c:    &DKAMInventoryClient{},
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

func TestDKAMInventoryClient_listAndReturnProvider(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	type args struct {
		ctx    context.Context
		filter *inv_v1.ResourceFilter
	}
	osr := inv_testing.CreateOs(t)
	tests := []struct {
		name    string
		args    args
		want    *provider_v1.ProviderResource
		wantErr bool
	}{
		{
			name: "listAndReturnProvider",
			args: args{
				ctx:    context.Background(),
				filter: &inv_v1.ResourceFilter{},
			},
			want:    &provider_v1.ProviderResource{},
			wantErr: true,
		},
		{
			name: "listAndReturnProvider case",
			args: args{
				ctx: context.Background(),
				filter: &inv_v1.ResourceFilter{
					Resource: &inv_v1.Resource{Resource: &inv_v1.Resource_Os{}},
					Filter:   fmt.Sprintf("%s = %q", osv1.OperatingSystemResourceFieldResourceId, osr.ResourceId),
				},
			},
			want:    &provider_v1.ProviderResource{},
			wantErr: true,
		},
		{
			name: "listAndReturnProvider case with name",
			args: args{
				ctx: context.Background(),
				filter: &inv_v1.ResourceFilter{
					Resource: &inv_v1.Resource{Resource: &inv_v1.Resource_Os{}},
					Filter:   fmt.Sprintf("%s = %q", provider_v1.ProviderResourceFieldName, "fm_onboarding"),
				},
			},
			want:    &provider_v1.ProviderResource{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.listAndReturnProvider(tt.args.ctx, tt.args.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.listAndReturnProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestDKAMInventoryClient_GetProviderConfig(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	inv_testing.CreateProvider(t, "fm_onboarding")
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
			name: "GetProviderConfig",
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
		{
			name: "GetProviderConfig with name",
			args: args{
				ctx:  context.Background(),
				name: "fm_onboarding",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.GetProviderConfig(tt.args.ctx, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.GetProviderConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestDKAMInventoryClient_UpdateInvResourceFields(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
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
			name: "UpdateInvResourceFields with empty fields",
			args: args{
				ctx:      context.Background(),
				resource: hostResCopy,
				fields:   []string{},
			},
			wantErr: false,
		},
		{
			name: "UpdateInvResourceFields",
			args: args{
				ctx:      context.Background(),
				resource: hostResCopy,
				fields:   []string{"field"},
			},
			wantErr: true,
		},
		{
			name: "Nil Resourcefields",
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
		{
			name: "Invalid Resource",
			args: args{
				ctx:      context.Background(),
				resource: proto.Clone(&provider_v1.ProviderResource{}),
				fields:   []string{"field"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := invClient.UpdateInvResourceFields(tt.args.ctx, tt.args.resource, tt.args.fields); (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.UpdateInvResourceFields() error = %v, wantErr %v", err, tt.wantErr)
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
		want Options
	}{
		{
			name: "EnablingTracing",
			args: args{enableTracing: true},
			want: Options{EnableTracing: true},
		},
		{
			name: "DisablingTracing",
			args: args{enableTracing: false},
			want: Options{EnableTracing: false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &Options{}
			got := WithEnableTracing(tt.args.enableTracing)
			got(options)
			if !reflect.DeepEqual(*options, tt.want) {
				t.Errorf("WithEnableTracing() = %v, want %v", *options, tt.want)
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
		want Options
	}{
		{
			name: "EmptyClientKind",
			args: args{},
			want: Options{},
		},
		{
			name: "WithClientKind",
			args: args{
				clientKind: inv_v1.ClientKind_CLIENT_KIND_API,
			},
			want: Options{ClientKind: inv_v1.ClientKind_CLIENT_KIND_API},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &Options{}
			got := WithClientKind(tt.args.clientKind)
			got(options)
			if !reflect.DeepEqual(*options, tt.want) {
				t.Errorf("WithClientKind() = %v, want %v", *options, tt.want)
			}
		})
	}
}

func TestDKAMInventoryClient_listAndReturnHost(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	type args struct {
		ctx    context.Context
		filter *inv_v1.ResourceFilter
	}
	osr := inv_testing.CreateOs(t)
	filter := &inv_v1.ResourceFilter{}
	tests := []struct {
		name    string
		args    args
		want    *computev1.HostResource
		wantErr bool
	}{
		{
			name: "listAndReturnHost test case failure",
			args: args{
				ctx:    context.Background(),
				filter: filter,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "listAndReturnHost case",
			args: args{
				ctx: context.Background(),
				filter: &inv_v1.ResourceFilter{
					Resource: &inv_v1.Resource{Resource: &inv_v1.Resource_Os{}},
					Filter:   fmt.Sprintf("%s = %q", osv1.OperatingSystemResourceFieldResourceId, osr.ResourceId),
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := invClient.listAndReturnHost(tt.args.ctx, tt.args.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.listAndReturnHost() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DKAMInventoryClient.listAndReturnHost() = %v, want %v", got, tt.want)
			}
		})
	}
}

