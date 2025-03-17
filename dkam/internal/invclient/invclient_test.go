// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package invclient_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/google/uuid"

	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-onboarding/dkam/internal/invclient"
)

func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	projectRoot := filepath.Dir(filepath.Dir(wd))
	policyPath := projectRoot + "/out"
	migrationsDir := projectRoot + "/out"

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
		want invclient.Option
	}{
		{
			name: "ProvidingInventoryAddress",
			args: args{},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := invclient.WithInventoryAddress(tt.args.invAddr); reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithInventoryAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewDKAMInventoryClientWithOptions(t *testing.T) {
	type args struct {
		opts []invclient.Option
	}
	tests := []struct {
		name    string
		args    args
		want    *invclient.DKAMInventoryClient
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
				opts: []invclient.Option{invclient.WithInventoryAddress("example.com")},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := invclient.NewDKAMInventoryClientWithOptions(tt.args.opts...)
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
		invClient client.TenantAwareInventoryClient
		watcher   chan *client.WatchEvents
	}
	tests := []struct {
		name    string
		args    args
		want    *invclient.DKAMInventoryClient
		wantErr bool
	}{
		{
			name:    "CreatingNewOnboardingInventoryClient",
			args:    args{},
			want:    &invclient.DKAMInventoryClient{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := invclient.NewDKAMInventoryClient(tt.args.invClient, tt.args.watcher)
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

func TestDKAMInventoryClient_GetHostResourceByUUID(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	host := inv_testing.CreateHost(t, nil, nil)
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

func TestDKAMInventoryClient_GetOSResourceByResourceID(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	osRes := inv_testing.CreateOs(t)
	type args struct {
		ctx        context.Context
		resourceID string
		tenantID   string
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
				tenantID:   osRes.GetTenantId(),
			},
			wantErr: true,
		},
		{
			name: "GetOSResourceByResourceID",
			args: args{
				ctx:        context.Background(),
				resourceID: osRes.ResourceId,
				tenantID:   osRes.GetTenantId(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invClient.GetOSResourceByResourceID(tt.args.ctx, tt.args.tenantID, tt.args.resourceID)
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

func TestGetProviderResourceByName(t *testing.T) {
	type args struct {
		ctx  context.Context
		c    *invclient.DKAMInventoryClient
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
				c:    &invclient.DKAMInventoryClient{},
				name: "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invclient.GetProviderResourceByName(tt.args.ctx, tt.args.c, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetProviderResourceByName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestDKAMInventoryClient_GetProviderConfig(t *testing.T) {
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	inv_testing.CreateProvider(t, "infra_onboarding")
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
				name: "infra_onboarding",
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

func TestWithEnableTracing(t *testing.T) {
	type args struct {
		enableTracing bool
	}
	tests := []struct {
		name string
		args args
		want invclient.Options
	}{
		{
			name: "EnablingTracing",
			args: args{enableTracing: true},
			want: invclient.Options{EnableTracing: true},
		},
		{
			name: "DisablingTracing",
			args: args{enableTracing: false},
			want: invclient.Options{EnableTracing: false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &invclient.Options{}
			got := invclient.WithEnableTracing(tt.args.enableTracing)
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
		want invclient.Options
	}{
		{
			name: "EmptyClientKind",
			args: args{},
			want: invclient.Options{},
		},
		{
			name: "WithClientKind",
			args: args{
				clientKind: inv_v1.ClientKind_CLIENT_KIND_API,
			},
			want: invclient.Options{ClientKind: inv_v1.ClientKind_CLIENT_KIND_API},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &invclient.Options{}
			got := invclient.WithClientKind(tt.args.clientKind)
			got(options)
			if !reflect.DeepEqual(*options, tt.want) {
				t.Errorf("WithClientKind() = %v, want %v", *options, tt.want)
			}
		})
	}
}

func TestDKAMInventoryClient_ListAllResources(t *testing.T) {
	var s inv_v1.ResourceKind
	CreateDkamClientForTesting(t)
	invClient := DkamTestClient
	type fields struct {
		Client  client.TenantAwareInventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx   context.Context
		kinds []inv_v1.ResourceKind
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*inv_v1.Resource
		wantErr bool
	}{
		{
			name: "Test case",
			fields: fields{
				Client: invClient.Client,
			},
			args: args{
				ctx:   context.Background(),
				kinds: []inv_v1.ResourceKind{s},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &invclient.DKAMInventoryClient{
				Client:  tt.fields.Client,
				Watcher: tt.fields.Watcher,
			}
			got, err := c.ListAllResources(tt.args.ctx, tt.args.kinds)
			if (err != nil) != tt.wantErr {
				t.Errorf("DKAMInventoryClient.ListAllResources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DKAMInventoryClient.ListAllResources() = %v, want %v", got, tt.want)
			}
		})
	}
}
