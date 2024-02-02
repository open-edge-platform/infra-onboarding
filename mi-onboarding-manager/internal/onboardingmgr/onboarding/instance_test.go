/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	"context"
	"errors"
	"testing"

	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/invclient"
	"github.com/stretchr/testify/mock"
)

func TestUpdateInstanceStatusByGuid(t *testing.T) {
	type args struct {
		ctx            context.Context
		invClient      *invclient.OnboardingInventoryClient
		hostUUID       string
		instancestatus computev1.InstanceStatus
	}
	MockInvClient := &MockInventoryClient{}
	host := &computev1.HostResource{
		ResourceId: "host-084d9b08",
		Instance: &computev1.InstanceResource{
			ResourceId: "inst-084d9b08",
		},
	}
	mockResource1 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: host,
		},
	}
	mockResources := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource1}},
	}
	host2 := &computev1.HostResource{
		ResourceId: "host-084d9b08",
	}
	mockResource2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: host2,
		},
	}
	mockResources1 := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource2}},
	}
	host3 := &computev1.HostResource{
		ResourceId: "host-084d9b08",
		Instance:nil,
	}
	mockResource3 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: host3,
		},
	}
	mockResources3 := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource3}},
	}
	MockInvClient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil)
	MockInvClient.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	MockInvClient1 := &MockInventoryClient{}
	MockInvClient1.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil)
	MockInvClient1.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, errors.New("err"))
	MockInvClient2 := &MockInventoryClient{}
	MockInvClient2.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, errors.New("err"))
	MockInvClient3 := &MockInventoryClient{}
	MockInvClient3.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources1, nil)
	MockInvClient3.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	MockInvClient4 := &MockInventoryClient{}
	MockInvClient4.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources3, nil)
	MockInvClient4.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Positive",
			args: args{
				ctx: context.TODO(),
				invClient: &invclient.OnboardingInventoryClient{
					Client: MockInvClient,
				},
				hostUUID:       "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
				instancestatus: computev1.InstanceStatus_INSTANCE_STATUS_ERROR,
			},
			wantErr: false,
		},
		{
			name: "Negative",
			args: args{
				ctx: context.TODO(),
				invClient: &invclient.OnboardingInventoryClient{
					Client: MockInvClient1,
				},
				hostUUID:       "mockhostUUID",
				instancestatus: computev1.InstanceStatus_INSTANCE_STATUS_ERROR,
			},
			wantErr: true,
		},

		{
			name: "Negative1",
			args: args{
				ctx: context.TODO(),
				invClient: &invclient.OnboardingInventoryClient{
					Client: MockInvClient2,
				},
				hostUUID:       "mockhostUUID",
				instancestatus: computev1.InstanceStatus_INSTANCE_STATUS_ERROR,
			},
			wantErr: true,
		},
		{
			name: "Negative2",
			args: args{
				ctx: context.TODO(),
				invClient: &invclient.OnboardingInventoryClient{
					Client: MockInvClient3,
				},
				hostUUID:       "mockhostUUID",
				instancestatus: computev1.InstanceStatus_INSTANCE_STATUS_ERROR,
			},
			wantErr: true,
		},
		{
			name: "Negative3",
			args: args{
				ctx: context.TODO(),
				invClient: &invclient.OnboardingInventoryClient{
					Client: MockInvClient4,
				},
				hostUUID:       "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
				instancestatus: computev1.InstanceStatus_INSTANCE_STATUS_ERROR,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := UpdateInstanceStatusByGuid(tt.args.ctx, tt.args.invClient, tt.args.hostUUID, tt.args.instancestatus); (err != nil) != tt.wantErr {
				t.Errorf("UpdateInstanceStatusByGuid() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

