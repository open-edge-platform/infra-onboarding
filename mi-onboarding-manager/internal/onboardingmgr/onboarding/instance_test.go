/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	"context"
	"testing"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	om_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/testing"
	om_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/status"
	inv_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/status"
	inv_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/testing"
)

func TestUpdateInstanceStatusByGuid(t *testing.T) {
	type args struct {
		ctx                context.Context
		invClient          *invclient.OnboardingInventoryClient
		hostUUID           string
		provisioningStatus inv_status.ResourceStatus
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	host := inv_testing.CreateHost(t, nil, nil)
	hostRes := inv_testing.CreateHost(t, nil, nil)
	osRes := inv_testing.CreateOs(t)
	inv_testing.CreateInstance(t, hostRes, osRes)
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Instance Doesn't Exist",
			args: args{
				ctx:                context.TODO(),
				invClient:          om_testing.InvClient,
				hostUUID:           host.Uuid,
				provisioningStatus: om_status.ProvisioningStatusFailed,
			},
			wantErr: true,
		},
		{
			name: "InvalidUUIDError",
			args: args{
				ctx:                context.TODO(),
				invClient:          om_testing.InvClient,
				hostUUID:           "mockhostUUID",
				provisioningStatus: om_status.ProvisioningStatusFailed,
			},
			wantErr: true,
		},

		{
			name: "ListResourcesError",
			args: args{
				ctx:                context.TODO(),
				invClient:          om_testing.InvClient,
				hostUUID:           "mockhostUUID",
				provisioningStatus: om_status.ProvisioningStatusFailed,
			},
			wantErr: true,
		},
		{
			name: "ListResourcesEmpty",
			args: args{
				ctx:                context.TODO(),
				invClient:          om_testing.InvClient,
				hostUUID:           "mockhostUUID",
				provisioningStatus: om_status.ProvisioningStatusFailed,
			},
			wantErr: true,
		},
		{
			name: "UpdateInstanceNoInstance",
			args: args{
				ctx:                context.TODO(),
				invClient:          om_testing.InvClient,
				hostUUID:           "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
				provisioningStatus: om_status.ProvisioningStatusFailed,
			},
			wantErr: true,
		},
		{
			name: "SuccessfulStatusUpdate",
			args: args{
				ctx:                context.TODO(),
				invClient:          om_testing.InvClient,
				hostUUID:           hostRes.Uuid,
				provisioningStatus: om_status.ProvisioningStatusFailed,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := UpdateInstanceStatusByGUID(tt.args.ctx, tt.args.invClient, tt.args.hostUUID,
				tt.args.provisioningStatus); (err != nil) != tt.wantErr {
				t.Errorf("UpdateInstanceStatusByGUID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
