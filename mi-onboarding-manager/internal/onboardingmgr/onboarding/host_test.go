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

func TestUpdateHostStatusByHostGuid(t *testing.T) {
	type args struct {
		ctx        context.Context
		invClient  *invclient.OnboardingInventoryClient
		hostUUID   string
		hoststatus computev1.HostStatus
	}
	host := &computev1.HostResource{
		ResourceId: "host-084d9b08",
		Uuid:       "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
	}
	mockResource2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: host,
		},
	}
	MockInvClient := &MockInventoryClient{}
	mockResources := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource2}},
	}
	MockInvClient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil)
	MockInvClient.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	MockInvClient1 := &MockInventoryClient{}
	MockInvClient1.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil)
	MockInvClient1.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, errors.New("err"))
	MockInvClient2 := &MockInventoryClient{}
	MockInvClient2.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, errors.New("err"))
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
				hostUUID:   "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
				hoststatus: computev1.HostStatus_HOST_STATUS_RUNNING,
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
				hostUUID:   "mockhostUUID",
				hoststatus: computev1.HostStatus_HOST_STATUS_RUNNING,
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
				hostUUID:   "mockhostUUID",
				hoststatus: computev1.HostStatus_HOST_STATUS_RUNNING,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := UpdateHostStatusByHostGuid(tt.args.ctx, tt.args.invClient, tt.args.hostUUID, tt.args.hoststatus); (err != nil) != tt.wantErr {
				t.Errorf("UpdateHostStatusByHostGuid() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

