// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package util

import (
	"testing"

	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/status"
)

func TestPopulateHostStatus(t *testing.T) {
	type args struct {
		instance         *computev1.InstanceResource
		onboardingStatus inv_status.ResourceStatus
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "TestPopulateHostStatus_BootFailed",
			args: args{
				instance: &computev1.InstanceResource{
					Host: &computev1.HostResource{
						ResourceId: "host-084d9b08",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			PopulateHostOnboardingStatus(tt.args.instance, tt.args.onboardingStatus)
		})
	}
}

func TestPopulateInstanceStatusAndCurrentState(t *testing.T) {
	type args struct {
		instance           *computev1.InstanceResource
		currentState       computev1.InstanceState
		provisioningStatus inv_status.ResourceStatus
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "TestPopulateInstanceStatusAndCurrentState_WithInstance",
			args: args{
				instance: &computev1.InstanceResource{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			PopulateInstanceStatusAndCurrentState(tt.args.instance, tt.args.currentState, tt.args.provisioningStatus)
		})
	}
}
