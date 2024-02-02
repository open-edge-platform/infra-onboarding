// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package reconcilers

import (
	"context"
	"errors"
	"reflect"
	"testing"

	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	v14 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/invclient"
	onboarding "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/onboarding"
	rec_v2 "github.com/onosproject/onos-lib-go/pkg/controller/v2"
	"github.com/stretchr/testify/mock"
)

func TestNewHostReconciler(t *testing.T) {
	type args struct {
		c *invclient.OnboardingInventoryClient
	}
	tests := []struct {
		name string
		args args
		want *HostReconciler
	}{
		{
			name: "Positive",
			args: args{
				c: &invclient.OnboardingInventoryClient{},
			},
			want: &HostReconciler{
				invClient: &invclient.OnboardingInventoryClient{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewHostReconciler(tt.args.c); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewHostReconciler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHostReconciler_Reconcile_Case1(t *testing.T) {
	type fields struct {
		invClient *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx     context.Context
		request rec_v2.Request[ResourceID]
	}
	testRequest := rec_v2.Request[ResourceID]{
		ID: ResourceID("test-id"),
	}
	// testRequestDre := rec_v2.Directive[ResourceID]{
	// 	// ID: ResourceID("test-id"),
	// }
	mockInvClient := &onboarding.MockInventoryClient{}
	mockInvClient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{}, errors.New("err"))
	mockHost := &v14.HostResource{
		DesiredState: computev1.HostState_HOST_STATE_UNSPECIFIED,
		CurrentState: computev1.HostState_HOST_STATE_UNSPECIFIED,
	}
	mockResource := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: mockHost,
		},
	}
	mockInvClient1 := &onboarding.MockInventoryClient{}
	mockInvClient1.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource,
	}, nil)
	mockHost2 := &v14.HostResource{
		DesiredState: computev1.HostState_HOST_STATE_PROVISIONED,
		CurrentState: computev1.HostState_HOST_STATE_UNSPECIFIED,
	}
	mockResource2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: mockHost2,
		},
	}
	mockInvClient2 := &onboarding.MockInventoryClient{}
	mockInvClient2.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource2,
	}, nil)
	mockInvClient2.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	//=================================
	mockHost3 := &v14.HostResource{
		ResourceId:   "host-084d9b08",
		DesiredState: computev1.HostState_HOST_STATE_DELETED,
		// CurrentState: computev1.HostState_HOST_STATE_UNSPECIFIED,
		HostNics: []*computev1.HostnicResource{{ResourceId: "hostnic-084d9b08"}},
	}
	mockResource3 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: mockHost3,
		},
	}
	mockInvClient3 := &onboarding.MockInventoryClient{}
	mockInvClient3.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource3,
	}, nil)
	mockInvClient3.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	// mockResources3 := &inv_v1.ListResourcesResponse{
	// 	Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource3}},
	// }
	mockResources3 := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource3}},
	}
	mockInvClient3.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources3, nil)
	tests := []struct {
		name   string
		fields fields
		args   args
		want   rec_v2.Directive[ResourceID]
	}{
		{
			name: "TestCase1",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{
					Client: mockInvClient,
				},
			},
			args: args{
				ctx:     context.TODO(),
				request: testRequest,
			},
			want: testRequest.Ack(),
		},
		{
			name: "TestCase2",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{
					Client: mockInvClient1,
				},
			},
			args: args{
				ctx:     context.TODO(),
				request: testRequest,
			},
			want: testRequest.Ack(),
		},
		{
			name: "TestCase3",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{
					Client: mockInvClient2,
				},
			},
			args: args{
				ctx:     context.TODO(),
				request: testRequest,
			},
			want: testRequest.Ack(),
		},
		//==
		// {
		// 	name: "TestCase4",
		// 	fields: fields{
		// 		invClient: &invclient.OnboardingInventoryClient{
		// 			Client: mockInvClient3,
		// 		},
		// 	},
		// 	args: args{
		// 		ctx:     context.TODO(),
		// 		request: testRequest,
		// 	},
		// 	want: testRequest.Ack(),
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := &HostReconciler{
				invClient: tt.fields.invClient,
			}
			if got := hr.Reconcile(tt.args.ctx, tt.args.request); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HostReconciler.Reconcile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHostReconciler_deleteHost(t *testing.T) {
	type fields struct {
		invClient *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx  context.Context
		host *computev1.HostResource
	}
	mockInvClient := &onboarding.MockInventoryClient{}
	mockInvClient.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	mockInvClient1 := &onboarding.MockInventoryClient{}
	mockInvClient1.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, errors.New("err"))
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test Case 1",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{
					Client: mockInvClient,
				},
			},
			args:    args{ctx: context.Background()},
			wantErr: false,
		},
		{
			name: "Test Case 2",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{
					Client: mockInvClient1,
				},
			},
			args:    args{ctx: context.Background()},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := &HostReconciler{
				invClient: tt.fields.invClient,
			}
			if err := hr.deleteHost(tt.args.ctx, tt.args.host); (err != nil) != tt.wantErr {
				t.Errorf("HostReconciler.deleteHost() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHostReconciler_deleteHostGpuByHost(t *testing.T) {
	type fields struct {
		invClient *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx     context.Context
		hostres *computev1.HostResource
	}
	mockInvClient := &onboarding.MockInventoryClient{}

	mockInvClient.On("Delete", mock.Anything, mock.Anything).Return(&inv_v1.DeleteResourceResponse{}, nil)
	mockInvClient1 := &onboarding.MockInventoryClient{}

	mockInvClient1.On("Delete", mock.Anything, mock.Anything).Return(&inv_v1.DeleteResourceResponse{}, errors.New("err"))
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test Case 1",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{
					Client: mockInvClient,
				},
			},
			args: args{
				ctx: context.Background(),
				hostres: &computev1.HostResource{
					HostGpus: []*v14.HostgpuResource{
						{},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Test Case 2",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{
					Client: mockInvClient1,
				},
			},
			args: args{
				ctx: context.Background(),
				hostres: &computev1.HostResource{
					HostGpus: []*v14.HostgpuResource{{}},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := &HostReconciler{
				invClient: tt.fields.invClient,
			}
			if err := hr.deleteHostGpuByHost(tt.args.ctx, tt.args.hostres); (err != nil) != tt.wantErr {
				t.Errorf("HostReconciler.deleteHostGpuByHost() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHostReconciler_deleteHostNicByHost(t *testing.T) {
	type fields struct {
		invClient *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx     context.Context
		hostres *computev1.HostResource
	}
	mockInvClient := &onboarding.MockInventoryClient{}
	mockInvClient.On("Delete", mock.Anything, mock.Anything).Return(&inv_v1.DeleteResourceResponse{}, nil)
	mockInvClient.On("List", mock.Anything, mock.Anything).Return(&inv_v1.ListResourcesResponse{}, nil)
	mockInvClient1 := &onboarding.MockInventoryClient{}
	mockInvClient1.On("Delete", mock.Anything, mock.Anything).Return(&inv_v1.DeleteResourceResponse{}, errors.New("err"))
	mockInvClient1.On("List", mock.Anything, mock.Anything).Return(&inv_v1.ListResourcesResponse{}, errors.New("err"))
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test Case 1",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{
					Client: mockInvClient,
				},
			},
			args: args{
				ctx: context.Background(),
				hostres: &computev1.HostResource{
					HostNics: []*computev1.HostnicResource{{}},
				},
			},
			wantErr: false,
		},
		{
			name: "Test Case 2",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{
					Client: mockInvClient1,
				},
			},
			args: args{
				ctx: context.Background(),
				hostres: &computev1.HostResource{
					HostNics: []*computev1.HostnicResource{{}},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := &HostReconciler{
				invClient: tt.fields.invClient,
			}
			if err := hr.deleteHostNicByHost(tt.args.ctx, tt.args.hostres); (err != nil) != tt.wantErr {
				t.Errorf("HostReconciler.deleteHostNicByHost() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHostReconciler_deleteIPsByHostNic(t *testing.T) {
	type fields struct {
		invClient *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx     context.Context
		hostNic *computev1.HostnicResource
	}
	resource := &inv_v1.Resource{
		// Resource: &computev1.HostnicResource{},
	}
	mockResources := []*inv_v1.GetResourceResponse{{Resource: resource}}
	mockInvClient := &onboarding.MockInventoryClient{}
	mockInvClient.On("Delete", mock.Anything, mock.Anything).Return(&inv_v1.DeleteResourceResponse{}, nil)
	mockInvClient.On("List", mock.Anything, mock.Anything).Return(&inv_v1.ListResourcesResponse{}, nil)
	mockInvClient1 := &onboarding.MockInventoryClient{}
	mockInvClient1.On("Delete", mock.Anything, mock.Anything).Return(&inv_v1.DeleteResourceResponse{}, nil)
	mockInvClient1.On("List", mock.Anything, mock.Anything).Return(&inv_v1.ListResourcesResponse{Resources: mockResources}, nil)
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test Case 1",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{
					Client: mockInvClient,
				},
			},
			args: args{
				ctx:     context.Background(),
				hostNic: &v14.HostnicResource{},
			},
			wantErr: false,
		},
		{
			name: "Test Case 1",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{
					Client: mockInvClient1,
				},
			},
			args: args{
				ctx:     context.Background(),
				hostNic: &v14.HostnicResource{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := &HostReconciler{
				invClient: tt.fields.invClient,
			}
			if err := hr.deleteIPsByHostNic(tt.args.ctx, tt.args.hostNic); (err != nil) != tt.wantErr {
				t.Errorf("HostReconciler.deleteIPsByHostNic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHostReconciler_deleteHostStorageByHost(t *testing.T) {
	type fields struct {
		invClient *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx     context.Context
		hostres *computev1.HostResource
	}
	mockInvClient := &onboarding.MockInventoryClient{}
	mockInvClient.On("Delete", mock.Anything, mock.Anything).Return(&inv_v1.DeleteResourceResponse{}, nil)
	mockInvClient.On("List", mock.Anything, mock.Anything).Return(&inv_v1.ListResourcesResponse{}, nil)
	mockInvClient1 := &onboarding.MockInventoryClient{}
	mockInvClient1.On("Delete", mock.Anything, mock.Anything).Return(&inv_v1.DeleteResourceResponse{}, errors.New("err"))
	mockInvClient1.On("List", mock.Anything, mock.Anything).Return(&inv_v1.ListResourcesResponse{}, errors.New("err"))
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test Case 1",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{
					Client: mockInvClient,
				},
			},
			args: args{
				ctx: context.Background(),
				hostres: &computev1.HostResource{
					HostStorages: []*v14.HoststorageResource{{}},
				},
			},
			wantErr: false,
		},
		{
			name: "Test Case 2",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{
					Client: mockInvClient1,
				},
			},
			args: args{
				ctx: context.Background(),
				hostres: &computev1.HostResource{
					HostStorages: []*v14.HoststorageResource{{}},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := &HostReconciler{
				invClient: tt.fields.invClient,
			}
			if err := hr.deleteHostStorageByHost(tt.args.ctx, tt.args.hostres); (err != nil) != tt.wantErr {
				t.Errorf("HostReconciler.deleteHostStorageByHost() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHostReconciler_deleteHostUsbByHost(t *testing.T) {
	type fields struct {
		invClient *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx  context.Context
		host *computev1.HostResource
	}
	mockInvClient := &onboarding.MockInventoryClient{}
	mockInvClient.On("Delete", mock.Anything, mock.Anything).Return(&inv_v1.DeleteResourceResponse{}, nil)
	mockInvClient.On("List", mock.Anything, mock.Anything).Return(&inv_v1.ListResourcesResponse{}, nil)
	mockInvClient1 := &onboarding.MockInventoryClient{}
	mockInvClient1.On("Delete", mock.Anything, mock.Anything).Return(&inv_v1.DeleteResourceResponse{}, errors.New("err"))
	mockInvClient1.On("List", mock.Anything, mock.Anything).Return(&inv_v1.ListResourcesResponse{}, errors.New("err"))
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
				ctx: context.Background(),
				host: &computev1.HostResource{
					HostUsbs: []*v14.HostusbResource{{}},
				},
			},
			wantErr: false,
		},
		{
			name: "Test Case 1",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{
					Client: mockInvClient1,
				},
			},
			args: args{
				ctx: context.Background(),
				host: &computev1.HostResource{
					HostUsbs: []*v14.HostusbResource{{}},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := &HostReconciler{
				invClient: tt.fields.invClient,
			}
			if err := hr.deleteHostUsbByHost(tt.args.ctx, tt.args.host); (err != nil) != tt.wantErr {
				t.Errorf("HostReconciler.deleteHostUsbByHost() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHostReconciler_Reconcile(t *testing.T) {
	type fields struct {
		invClient *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx     context.Context
		request rec_v2.Request[ResourceID]
	}
	mockHost3 := &v14.HostResource{
		ResourceId:   "host-084d9b08",
		DesiredState: computev1.HostState_HOST_STATE_DELETED,
		HostNics: []*computev1.HostnicResource{{ResourceId: "hostnic-084d9b08"}},
	}
	mockResource3 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: mockHost3,
		},
	}
	mockInvClient3 := &onboarding.MockInventoryClient{}
	mockInvClient3.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource3,
	}, nil)
	mockInvClient3.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	mockResources3 := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource3}},
	}
	mockInvClient3.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources3, nil)
	testRequest := rec_v2.Request[ResourceID]{
		ID: ResourceID("test-id"),
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   rec_v2.Directive[ResourceID]
	}{
		{
			name: "TestCase4",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{
					Client: mockInvClient3,
				},
			},
			args: args{
				ctx:     context.TODO(),
				request: testRequest,
			},
			want: testRequest.Ack(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := &HostReconciler{
				invClient: tt.fields.invClient,
			}
			if got := hr.Reconcile(tt.args.ctx, tt.args.request); reflect.DeepEqual(got, tt.want) {
				t.Errorf("HostReconciler.Reconcile() = %v, want %v", got, tt.want)
			}
		})
	}
}

