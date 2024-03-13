// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

package invclient

import (
	"context"
	"errors"
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
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/client/cache"
	inv_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/status"
	inv_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type MockInventoryClient struct {
	mock.Mock
}

func (m *MockInventoryClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockInventoryClient) List(ctx context.Context, filter *inv_v1.ResourceFilter) (*inv_v1.ListResourcesResponse, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(*inv_v1.ListResourcesResponse), args.Error(1)
}

func (m *MockInventoryClient) ListAll(ctx context.Context, filter *inv_v1.ResourceFilter) ([]*inv_v1.Resource, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*inv_v1.Resource), args.Error(1)
}

func (m *MockInventoryClient) Find(ctx context.Context, filter *inv_v1.ResourceFilter) (*inv_v1.FindResourcesResponse, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(*inv_v1.FindResourcesResponse), args.Error(1)
}

func (m *MockInventoryClient) FindAll(ctx context.Context, filter *inv_v1.ResourceFilter) ([]string, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockInventoryClient) Get(ctx context.Context, id string) (*inv_v1.GetResourceResponse, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*inv_v1.GetResourceResponse), args.Error(1)
}

func (m *MockInventoryClient) Create(ctx context.Context, resource *inv_v1.Resource) (*inv_v1.CreateResourceResponse, error) {
	args := m.Called(ctx, resource)
	return args.Get(0).(*inv_v1.CreateResourceResponse), args.Error(1)
}

func (m *MockInventoryClient) Update(ctx context.Context, id string,
	mask *fieldmaskpb.FieldMask, resource *inv_v1.Resource,
) (*inv_v1.UpdateResourceResponse, error) {
	args := m.Called(ctx, id, mask, resource)
	return args.Get(0).(*inv_v1.UpdateResourceResponse), args.Error(1)
}

func (m *MockInventoryClient) Delete(ctx context.Context, id string) (*inv_v1.DeleteResourceResponse, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*inv_v1.DeleteResourceResponse), args.Error(1)
}

func (m *MockInventoryClient) UpdateSubscriptions(ctx context.Context, kinds []inv_v1.ResourceKind) error {
	args := m.Called(ctx, kinds)
	return args.Error(0)
}

func (m *MockInventoryClient) ListInheritedTelemetryProfiles(ctx context.Context,
	inheritBy *inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy,
	filter string,
	orderBy string,
	limit, offset uint32,
) (*inv_v1.ListInheritedTelemetryProfilesResponse, error) {
	args := m.Called(ctx, inheritBy, filter, orderBy, limit, offset)
	return args.Get(0).(*inv_v1.ListInheritedTelemetryProfilesResponse), args.Error(1)
}

func (m *MockInventoryClient) TestingOnlySetClient(invClient inv_v1.InventoryServiceClient) {
	m.Called(invClient)
}

func (m *MockInventoryClient) TestGetClientCache() *cache.InventoryCache {
	m.Called()
	return nil
}

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
			name: "TestCase",
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
			name: "TestCase",
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
			name:    "Test Case",
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
			name:    "Test Case",
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
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	mockClient := &MockInventoryClient{}
	mockClient.On("Close").Return(nil)
	mockClient1 := &MockInventoryClient{}
	mockClient1.On("Close").Return(errors.New("err"))
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "Positive",
			fields: fields{
				Client:  mockClient,
				Watcher: make(chan *client.WatchEvents),
			},
		},
		{
			name: "Negative",
			fields: fields{
				Client:  mockClient1,
				Watcher: make(chan *client.WatchEvents),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OnboardingInventoryClient{
				Client:  tt.fields.Client,
				Watcher: tt.fields.Watcher,
			}
			c.Close()
		})
	}
}

func TestOnboardingInventoryClient_UpdateHostResource(t *testing.T) {
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx  context.Context
		host *computev1.HostResource
	}
	mockClient := &MockInventoryClient{}
	mockClient.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test Case 1",
			fields: fields{
				Client: mockClient,
			},
			args: args{
				ctx:  context.Background(),
				host: &computev1.HostResource{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OnboardingInventoryClient{
				Client:  tt.fields.Client,
				Watcher: tt.fields.Watcher,
			}
			if err := c.UpdateHostResource(tt.args.ctx, tt.args.host); (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.UpdateHostResource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOnboardingInventoryClient_GetHostResources(t *testing.T) {
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx context.Context
	}
	resource := inv_v1.Resource{
		Resource: &inv_v1.Resource_Instance{
			Instance: &computev1.InstanceResource{
				ResourceId: "inst-78789",
			},
		},
	}
	mockResource := []*inv_v1.GetResourceResponse{{
		Resource: &resource,
	}}
	mockClient := &MockInventoryClient{}
	mockClient.On("List", mock.Anything, mock.Anything).Return(&inv_v1.ListResourcesResponse{}, nil)
	mockClient1 := &MockInventoryClient{}
	mockClient1.On("List", mock.Anything, mock.Anything).Return(&inv_v1.ListResourcesResponse{}, errors.New("err"))
	mockClient2 := &MockInventoryClient{}
	mockClient2.On("List", mock.Anything, mock.Anything).Return(&inv_v1.ListResourcesResponse{Resources: mockResource}, nil)
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantHostres []*computev1.HostResource
		wantErr     bool
	}{
		{
			name: "Test Case 1",
			fields: fields{
				Client: mockClient,
			},
			args: args{
				ctx: context.Background(),
			},
			wantHostres: []*computev1.HostResource{},
			wantErr:     false,
		},
		{
			name: "Test Case 2",
			fields: fields{
				Client: mockClient1,
			},
			args: args{
				ctx: context.Background(),
			},
			wantHostres: nil,
			wantErr:     true,
		},
		{
			name: "Test Case 3",
			fields: fields{
				Client: mockClient2,
			},
			args: args{
				ctx: context.Background(),
			},
			wantHostres: nil,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OnboardingInventoryClient{
				Client:  tt.fields.Client,
				Watcher: tt.fields.Watcher,
			}
			gotHostres, err := c.GetHostResources(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.GetHostResources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotHostres, tt.wantHostres) {
				t.Errorf("OnboardingInventoryClient.GetHostResources() = %v, want %v", gotHostres, tt.wantHostres)
			}
		})
	}
}

func TestOnboardingInventoryClient_FindAllInstances(t *testing.T) {
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx context.Context
	}
	mockClient := &MockInventoryClient{}
	mockClient.On("FindAll", mock.Anything, mock.Anything, mock.Anything).Return([]string{}, nil)
	mockClient1 := &MockInventoryClient{}
	mockClient1.On("FindAll", mock.Anything, mock.Anything, mock.Anything).Return([]string{}, errors.New("err"))
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "Test Case 1",
			fields: fields{
				Client: mockClient,
			},
			args: args{
				ctx: context.Background(),
			},
			want:    nil,
			wantErr: false,
		},

		{
			name: "Test Case 2",
			fields: fields{
				Client: mockClient1,
			},
			args: args{
				ctx: context.Background(),
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
			got, err := c.FindAllInstances(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.FindAllInstances() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OnboardingInventoryClient.FindAllInstances() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOnboardingInventoryClient_GetHostResourceByResourceID(t *testing.T) {
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx        context.Context
		resourceID string
	}
	mockClient := &MockInventoryClient{}
	mockClient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{}, errors.New("err"))
	resource := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: &computev1.HostResource{
				ResourceId: "inst-78789",
			},
		},
	}
	mockClient1 := &MockInventoryClient{}
	mockClient1.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{Resource: resource}, nil)
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *computev1.HostResource
		wantErr bool
	}{
		{
			name: "Test Case 1",
			fields: fields{
				Client: mockClient,
			},
			args: args{
				ctx:        context.Background(),
				resourceID: "id",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Test Case 2",
			fields: fields{
				Client: mockClient1,
			},
			args: args{
				ctx:        context.Background(),
				resourceID: "id",
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
			got, err := c.GetHostResourceByResourceID(tt.args.ctx, tt.args.resourceID)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.GetHostResourceByResourceID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OnboardingInventoryClient.GetHostResourceByResourceID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOnboardingInventoryClient_CreateHostResource(t *testing.T) {
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx  context.Context
		host *computev1.HostResource
	}
	mockClient := &MockInventoryClient{}
	mockClient.On("Create", mock.Anything, mock.Anything).Return(&inv_v1.CreateResourceResponse{}, nil)
	mockClient1 := &MockInventoryClient{}
	mockClient1.On("Create", mock.Anything, mock.Anything).Return(&inv_v1.CreateResourceResponse{}, errors.New("err"))
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Test Case 1",
			fields: fields{
				Client: mockClient,
			},
			args: args{
				ctx: context.Background(),
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "Test Case 1",
			fields: fields{
				Client: mockClient1,
			},
			args: args{
				ctx: context.Background(),
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OnboardingInventoryClient{
				Client:  tt.fields.Client,
				Watcher: tt.fields.Watcher,
			}
			got, err := c.CreateHostResource(tt.args.ctx, tt.args.host)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.CreateHostResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("OnboardingInventoryClient.CreateHostResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOnboardingInventoryClient_GetHostResourceByUUID(t *testing.T) {
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx  context.Context
		uuid string
	}
	mockClient := &MockInventoryClient{}
	mockClient.On("List", mock.Anything, mock.Anything).Return(&inv_v1.ListResourcesResponse{}, nil)
	resource := inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: &computev1.HostResource{
				ResourceId: "host-084d9b08",
			},
		},
	}
	mockResource := []*inv_v1.GetResourceResponse{{
		Resource: &resource,
	}}
	mockClient1 := &MockInventoryClient{}
	mockClient1.On("List", mock.Anything, mock.Anything).Return(&inv_v1.ListResourcesResponse{Resources: mockResource}, nil)
	resource1 := inv_v1.Resource{}
	mockResource1 := []*inv_v1.GetResourceResponse{{
		Resource: &resource1,
	}}
	mockClient2 := &MockInventoryClient{}
	mockClient2.On("List", mock.Anything, mock.Anything).Return(&inv_v1.ListResourcesResponse{Resources: mockResource1}, nil)

	mockClient3 := &MockInventoryClient{}
	mockClient3.On("List", mock.Anything, mock.Anything).Return(&inv_v1.ListResourcesResponse{}, errors.New("err"))
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *computev1.HostResource
		wantErr bool
	}{
		{
			name: "Test Case 1",
			fields: fields{
				Client: mockClient,
			},
			args: args{
				ctx:  context.Background(),
				uuid: "123",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Test Case 2",
			fields: fields{
				Client: mockClient,
			},
			args: args{
				ctx: context.Background(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Test Case 3",
			fields: fields{
				Client: mockClient1,
			},
			args: args{
				ctx:  context.Background(),
				uuid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
			},
			want: &computev1.HostResource{
				ResourceId: "host-084d9b08",
			},
			wantErr: false,
		},
		{
			name: "Test Case 4",
			fields: fields{
				Client: mockClient2,
			},
			args: args{
				ctx:  context.Background(),
				uuid: "123",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Test Case 5",
			fields: fields{
				Client: mockClient3,
			},
			args: args{
				ctx:  context.Background(),
				uuid: "123",
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
			got, err := c.GetHostResourceByUUID(tt.args.ctx, tt.args.uuid)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.GetHostResourceByUUID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OnboardingInventoryClient.GetHostResourceByUUID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOnboardingInventoryClient_DeleteHostResource(t *testing.T) {
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx        context.Context
		resourceID string
	}
	mockClient := &MockInventoryClient{}
	mockClient.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)

	mockClient1 := &MockInventoryClient{}
	mockClient1.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, errors.New("err"))

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			fields: fields{
				Client: mockClient,
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
		{
			name: "Test Case 1",
			fields: fields{
				Client: mockClient1,
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OnboardingInventoryClient{
				Client:  tt.fields.Client,
				Watcher: tt.fields.Watcher,
			}
			if err := c.DeleteHostResource(tt.args.ctx, tt.args.resourceID); (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.DeleteHostResource() error = %v, wantErr %v", err, tt.wantErr)
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

			timeBeforeUpdate := time.Now().UTC()
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
				timeNow, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST",
					hostInv.GetOnboardingStatusTimestamp())
				require.NoError(t, err)
				assert.False(t, timeNow.UTC().Before(timeBeforeUpdate))
			}
		})
	}
}

func TestOnboardingInventoryClient_CreateInstanceResource(t *testing.T) {
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx  context.Context
		inst *computev1.InstanceResource
	}
	mockClient := &MockInventoryClient{}
	mockClient.On("Create", mock.Anything, mock.Anything).Return(&inv_v1.CreateResourceResponse{}, nil)
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Test Case 1",
			fields: fields{
				Client: mockClient,
			},
			args: args{
				ctx: context.Background(),
			},
			want:    "",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OnboardingInventoryClient{
				Client:  tt.fields.Client,
				Watcher: tt.fields.Watcher,
			}
			got, err := c.CreateInstanceResource(tt.args.ctx, tt.args.inst)
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
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx        context.Context
		resourceID string
	}
	resource := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: &computev1.HostResource{
				ResourceId: "inst-78789",
			},
		},
	}
	mockClient := &MockInventoryClient{}
	mockClient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{Resource: resource}, nil)
	mockClient1 := &MockInventoryClient{}
	mockClient1.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{Resource: resource}, errors.New("err"))
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *computev1.InstanceResource
		wantErr bool
	}{
		{
			name: "Test Case 1",
			fields: fields{
				Client: mockClient,
			},
			args: args{
				ctx: context.Background(),
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "Test Case 2",
			fields: fields{
				Client: mockClient1,
			},
			args: args{
				ctx: context.Background(),
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
			got, err := c.GetInstanceResourceByResourceID(tt.args.ctx, tt.args.resourceID)
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

func TestOnboardingInventoryClient_GetInstanceResources(t *testing.T) {
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx context.Context
	}
	mockClient := &MockInventoryClient{}
	mockClient.On("List", mock.Anything, mock.Anything).Return(&inv_v1.ListResourcesResponse{}, nil)
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*computev1.InstanceResource
		wantErr bool
	}{
		{
			name: "Test Case 1",
			fields: fields{
				Client: mockClient,
			},
			args: args{
				ctx: context.Background(),
			},
			want:    []*computev1.InstanceResource{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OnboardingInventoryClient{
				Client:  tt.fields.Client,
				Watcher: tt.fields.Watcher,
			}
			got, err := c.GetInstanceResources(tt.args.ctx)
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

func TestOnboardingInventoryClient_UpdateInstanceResource(t *testing.T) {
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx  context.Context
		inst *computev1.InstanceResource
	}
	mockClient := &MockInventoryClient{}
	mockClient.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test Case 1",
			fields: fields{
				Client: mockClient,
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OnboardingInventoryClient{
				Client:  tt.fields.Client,
				Watcher: tt.fields.Watcher,
			}
			if err := c.UpdateInstanceResource(tt.args.ctx, tt.args.inst); (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.UpdateInstanceResource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOnboardingInventoryClient_DeleteInstanceResource(t *testing.T) {
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx        context.Context
		resourceID string
	}
	mockClient := &MockInventoryClient{}
	mockClient.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	mockClient1 := &MockInventoryClient{}
	mockClient1.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, errors.New("err"))
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test Case 1",
			fields: fields{
				Client: mockClient,
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
		{
			name: "Test Case 2",
			fields: fields{
				Client: mockClient1,
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OnboardingInventoryClient{
				Client:  tt.fields.Client,
				Watcher: tt.fields.Watcher,
			}
			if err := c.DeleteInstanceResource(tt.args.ctx, tt.args.resourceID); (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.DeleteInstanceResource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOnboardingInventoryClient_DeleteResource(t *testing.T) {
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx        context.Context
		resourceID string
	}
	mockClient := &MockInventoryClient{}
	mockClient.On("Delete", mock.Anything, mock.Anything).Return(&inv_v1.DeleteResourceResponse{}, nil)
	mockClient1 := &MockInventoryClient{}
	mockClient1.On("Delete", mock.Anything, mock.Anything).Return(&inv_v1.DeleteResourceResponse{}, errors.New("err"))
	mockClient2 := &MockInventoryClient{}
	mockClient2.On("Delete", mock.Anything, mock.Anything).Return(&inv_v1.DeleteResourceResponse{},
		status.Error(codes.NotFound, "Node not found"))
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test Case 1",
			fields: fields{
				Client: mockClient,
			},
			args:    args{ctx: context.Background()},
			wantErr: false,
		},
		{
			name: "Test Case 2",
			fields: fields{
				Client: mockClient1,
			},
			args:    args{ctx: context.Background()},
			wantErr: true,
		},
		{
			name: "Test Case 3",
			fields: fields{
				Client: mockClient2,
			},
			args:    args{ctx: context.Background()},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OnboardingInventoryClient{
				Client:  tt.fields.Client,
				Watcher: tt.fields.Watcher,
			}
			if err := c.DeleteResource(tt.args.ctx, tt.args.resourceID); (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.DeleteResource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOnboardingInventoryClient_CreateOSResource(t *testing.T) {
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx context.Context
		os  *osv1.OperatingSystemResource
	}
	mockClient := &MockInventoryClient{}
	mockClient.On("Create", mock.Anything, mock.Anything).Return(&inv_v1.CreateResourceResponse{}, nil)
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Test case 1",
			fields: fields{
				Client: mockClient,
			},
			args:    args{ctx: context.Background()},
			want:    "",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OnboardingInventoryClient{
				Client:  tt.fields.Client,
				Watcher: tt.fields.Watcher,
			}
			got, err := c.CreateOSResource(tt.args.ctx, tt.args.os)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.CreateOSResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("OnboardingInventoryClient.CreateOSResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOnboardingInventoryClient_GetOSResourceByResourceID(t *testing.T) {
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx        context.Context
		resourceID string
	}
	mockClient := &MockInventoryClient{}
	mockClient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{}, nil)
	mockClient1 := &MockInventoryClient{}
	mockClient1.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{}, errors.New("err"))
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *osv1.OperatingSystemResource
		wantErr bool
	}{
		{
			name: "Test Case 1",
			fields: fields{
				Client: mockClient,
			},
			args:    args{ctx: context.Background()},
			want:    nil,
			wantErr: false,
		},
		{
			name: "Test Case 2",
			fields: fields{
				Client: mockClient1,
			},
			args:    args{ctx: context.Background()},
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
			got, err := c.GetOSResourceByResourceID(tt.args.ctx, tt.args.resourceID)
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

func TestOnboardingInventoryClient_GetOSResources(t *testing.T) {
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx context.Context
	}
	mockClient := &MockInventoryClient{}
	mockClient.On("List", mock.Anything, mock.Anything).Return(&inv_v1.ListResourcesResponse{}, nil)
	mockClient1 := &MockInventoryClient{}
	mockClient1.On("List", mock.Anything, mock.Anything).Return(&inv_v1.ListResourcesResponse{}, errors.New("err"))
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*osv1.OperatingSystemResource
		wantErr bool
	}{
		{
			name: "Test Case 1",
			fields: fields{
				Client: mockClient,
			},
			args:    args{ctx: context.Background()},
			want:    []*osv1.OperatingSystemResource{},
			wantErr: false,
		},
		{
			name: "Test Case 2",
			fields: fields{
				Client: mockClient1,
			},
			args:    args{ctx: context.Background()},
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
			got, err := c.GetOSResources(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.GetOSResources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OnboardingInventoryClient.GetOSResources() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOnboardingInventoryClient_ListIPAddresses(t *testing.T) {
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx     context.Context
		hostNic *computev1.HostnicResource
	}
	mockClient := &MockInventoryClient{}
	mockClient.On("List", mock.Anything, mock.Anything).Return(&inv_v1.ListResourcesResponse{}, nil)
	mockClient1 := &MockInventoryClient{}
	mockClient1.On("List", mock.Anything, mock.Anything).Return(&inv_v1.ListResourcesResponse{}, errors.New("err"))
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*network_v1.IPAddressResource
		wantErr bool
	}{
		{
			name: "Test Case 1",
			fields: fields{
				Client: mockClient,
			},
			args:    args{ctx: context.Background()},
			want:    []*network_v1.IPAddressResource{},
			wantErr: false,
		},
		{
			name: "Test Case 2",
			fields: fields{
				Client: mockClient1,
			},
			args:    args{ctx: context.Background()},
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
			got, err := c.ListIPAddresses(tt.args.ctx, tt.args.hostNic)
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
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx   context.Context
		kinds []inv_v1.ResourceKind
	}
	mockClient := &MockInventoryClient{}
	mockClient1 := &MockInventoryClient{}
	mockClient1.On("FindAll", mock.Anything, mock.Anything, mock.Anything).Return([]string{}, nil)
	mockClient2 := &MockInventoryClient{}
	mockClient2.On("FindAll", mock.Anything, mock.Anything, mock.Anything).Return([]string{}, errors.New("err"))
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "Test Case 1",
			fields: fields{
				Client: mockClient,
			},
			args:    args{ctx: context.Background()},
			want:    nil,
			wantErr: false,
		},
		{
			name: "Test Case 2",
			fields: fields{
				Client: mockClient1,
			},
			args: args{
				ctx:   context.Background(),
				kinds: []inv_v1.ResourceKind{inv_v1.ResourceKind_RESOURCE_KIND_HOST},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "Test Case 3",
			fields: fields{
				Client: mockClient2,
			},
			args: args{
				ctx:   context.Background(),
				kinds: []inv_v1.ResourceKind{inv_v1.ResourceKind_RESOURCE_KIND_HOST},
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
			got, err := c.FindAllResources(tt.args.ctx, tt.args.kinds)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.FindAllResources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OnboardingInventoryClient.FindAllResources() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOnboardingInventoryClient_UpdateInvResourceFields(t *testing.T) {
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx      context.Context
		resource proto.Message
		fields   []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test case 1",
			fields: fields{
				Client: &MockInventoryClient{},
			},
			args:    args{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OnboardingInventoryClient{
				Client:  tt.fields.Client,
				Watcher: tt.fields.Watcher,
			}
			if err := c.UpdateInvResourceFields(tt.args.ctx, tt.args.resource, tt.args.fields); (err != nil) != tt.wantErr {
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
		updateTimestamp      time.Time
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
				updateTimestamp:      time.Now().UTC(),
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
				updateTimestamp:      time.Now().UTC(),
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
				HostStatusTimestamp:  tt.args.updateTimestamp.String(),
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
				hostInv, err1 := OnboardingTestClient.GetHostResourceByUUID(ctx, host.Uuid)
				require.NoError(t, err1)
				require.NotNil(t, hostInv)

				assert.Equal(t, tt.args.hostCurrentState, hostInv.GetCurrentState())
				assert.Equal(t, tt.args.legacyHostStatus, hostInv.GetLegacyHostStatus())
				assert.Equal(t, tt.args.providerStatus, hostInv.GetProviderStatus())
				assert.Equal(t, tt.args.providerStatusDetail, hostInv.GetProviderStatusDetail())
				assert.Equal(t, tt.args.runtimeHostStatus.Status, hostInv.GetHostStatus())
				assert.Equal(t, tt.args.runtimeHostStatus.StatusIndicator, hostInv.GetHostStatusIndicator())
				timeNow, err2 := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST",
					hostInv.GetHostStatusTimestamp())
				require.NoError(t, err2)
				assert.False(t, timeNow.UTC().Before(tt.args.updateTimestamp))
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

			timeBeforeUpdate := time.Now().UTC()
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
				timeNow, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST",
					instInv.GetProvisioningStatusTimestamp())
				require.NoError(t, err)
				assert.False(t, timeNow.UTC().Before(timeBeforeUpdate))
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
			name: "Test Case",
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
			name: "Test Case",
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
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	hostResource := &computev1.HostResource{}
	hostResCopy := proto.Clone(hostResource)
	type args struct {
		ctx      context.Context
		resource proto.Message
		fields   []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test case 1",
			fields: fields{
				Client: &MockInventoryClient{},
			},
			args: args{
				resource: hostResCopy,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OnboardingInventoryClient{
				Client:  tt.fields.Client,
				Watcher: tt.fields.Watcher,
			}
			if err := c.UpdateInvResourceFields(tt.args.ctx, tt.args.resource, tt.args.fields); (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.UpdateInvResourceFields() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOnboardingInventoryClient_UpdateInvResourceFields_Case1(t *testing.T) {
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	// hostResource := &computev1.HostResource{}
	// hostResCopy := proto.Clone(hostResource)
	res := &network_v1.EndpointResource{}
	resCopy := proto.Clone(res)
	type args struct {
		ctx      context.Context
		resource proto.Message
		fields   []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test case 1",
			fields: fields{
				Client: &MockInventoryClient{},
			},
			args: args{
				resource: resCopy,
				fields:   []string{"field"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OnboardingInventoryClient{
				Client:  tt.fields.Client,
				Watcher: tt.fields.Watcher,
			}
			if err := c.UpdateInvResourceFields(tt.args.ctx, tt.args.resource, tt.args.fields); (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.UpdateInvResourceFields() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOnboardingInventoryClient_UpdateInvResourceFields_Case3(t *testing.T) {
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	hostResource := &computev1.HostResource{}
	hostResCopy := proto.Clone(hostResource)
	type args struct {
		ctx      context.Context
		resource proto.Message
		fields   []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test case 1",
			fields: fields{
				Client: &MockInventoryClient{},
			},
			args: args{
				resource: hostResCopy,
				fields:   []string{"field"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OnboardingInventoryClient{
				Client:  tt.fields.Client,
				Watcher: tt.fields.Watcher,
			}
			if err := c.UpdateInvResourceFields(tt.args.ctx, tt.args.resource, tt.args.fields); (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.UpdateInvResourceFields() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOnboardingInventoryClient_GetInstanceResourceByResourceID_Case(t *testing.T) {
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx        context.Context
		resourceID string
	}
	resource := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Instance{
			Instance: &computev1.InstanceResource{
				ResourceId: "inst-78789",
			},
		},
	}
	mockClient := &MockInventoryClient{}
	mockClient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{Resource: resource}, nil)
	mockClient1 := &MockInventoryClient{}
	mockClient1.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{Resource: resource}, errors.New("err"))
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *computev1.InstanceResource
		wantErr bool
	}{
		{
			name: "Test Case 1",
			fields: fields{
				Client: mockClient,
			},
			args: args{
				ctx: context.Background(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Test Case 2",
			fields: fields{
				Client: mockClient1,
			},
			args: args{
				ctx: context.Background(),
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
			got, err := c.GetInstanceResourceByResourceID(tt.args.ctx, tt.args.resourceID)
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
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx context.Context
	}
	mockClient := &MockInventoryClient{}
	mockClient.On("List", mock.Anything, mock.Anything).Return(&inv_v1.ListResourcesResponse{}, errors.New("err"))
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*computev1.InstanceResource
		wantErr bool
	}{
		{
			name: "Test Case 1",
			fields: fields{
				Client: mockClient,
			},
			args: args{
				ctx: context.Background(),
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
			got, err := c.GetInstanceResources(tt.args.ctx)
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

func TestOnboardingInventoryClient_GetOSResourceByResourceID_Case(t *testing.T) {
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx        context.Context
		resourceID string
	}
	mockClient := &MockInventoryClient{}
	mockClient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: &inv_v1.Resource{
			Resource: &inv_v1.Resource_Os{
				Os: &osv1.OperatingSystemResource{
					ResourceId: "123",
				},
			},
		},
	}, nil)
	mockClient1 := &MockInventoryClient{}
	mockClient1.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{}, errors.New("err"))
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *osv1.OperatingSystemResource
		wantErr bool
	}{
		{
			name: "Test Case 1",
			fields: fields{
				Client: mockClient,
			},
			args:    args{ctx: context.Background()},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Test Case 2",
			fields: fields{
				Client: mockClient1,
			},
			args:    args{ctx: context.Background()},
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
			got, err := c.GetOSResourceByResourceID(tt.args.ctx, tt.args.resourceID)
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
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx context.Context
	}
	mockClient := &MockInventoryClient{}
	mockClient.On("List", mock.Anything, mock.Anything).Return(&inv_v1.ListResourcesResponse{}, errors.New("err"))
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*provider_v1.ProviderResource
		wantErr bool
	}{
		{
			name: "Test Case",
			fields: fields{
				Client: mockClient,
			},
			args:    args{ctx: context.Background()},
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
			got, err := c.GetProviderResources(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.GetProviderResources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OnboardingInventoryClient.GetProviderResources() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOnboardingInventoryClient_DeleteIPAddress(t *testing.T) {
	type fields struct {
		Client  client.InventoryClient
		Watcher chan *client.WatchEvents
	}
	type args struct {
		ctx        context.Context
		resourceID string
	}
	mockClient := &MockInventoryClient{}
	mockClient.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	mockClient1 := &MockInventoryClient{}
	mockClient1.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, errors.New("err"))
	mockClient2 := &MockInventoryClient{}
	mockClient2.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, status.Error(codes.NotFound, "Node not found"))
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			fields: fields{
				Client: mockClient,
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
		{
			name: "Test Case",
			fields: fields{
				Client: mockClient1,
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
		{
			name: "Test Case",
			fields: fields{
				Client: mockClient2,
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OnboardingInventoryClient{
				Client:  tt.fields.Client,
				Watcher: tt.fields.Watcher,
			}
			if err := c.DeleteIPAddress(tt.args.ctx, tt.args.resourceID); (err != nil) != tt.wantErr {
				t.Errorf("OnboardingInventoryClient.DeleteIPAddress() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
