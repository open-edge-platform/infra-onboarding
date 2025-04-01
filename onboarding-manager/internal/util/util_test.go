// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package util_test

import (
	"testing"

	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_status "github.com/open-edge-platform/infra-core/inventory/v2/pkg/status"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/util"
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
		t.Run(tt.name, func(_ *testing.T) {
			util.PopulateHostOnboardingStatus(tt.args.instance, tt.args.onboardingStatus)
		})
	}
}

func TestPopulateInstanceProvisioningStatus(t *testing.T) {
	type args struct {
		instance           *computev1.InstanceResource
		provisioningStatus inv_status.ResourceStatus
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "TestPopulateInstanceProvisioningStatus_WithInstance",
			args: args{
				instance: &computev1.InstanceResource{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			util.PopulateInstanceProvisioningStatus(tt.args.instance, tt.args.provisioningStatus)
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
		t.Run(tt.name, func(_ *testing.T) {
			util.PopulateInstanceStatusAndCurrentState(tt.args.instance, tt.args.currentState, tt.args.provisioningStatus)
		})
	}
}
