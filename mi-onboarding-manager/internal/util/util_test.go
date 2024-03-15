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
		hoststatus       computev1.HostStatus
		statusDetails    string
		onboardingStatus inv_status.ResourceStatus
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test Case",
			args: args{
				instance: &computev1.InstanceResource{
					Host: &computev1.HostResource{
						ResourceId: "host-084d9b08",
					},
				},
				hoststatus: computev1.HostStatus_HOST_STATUS_BOOT_FAILED,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			PopulateHostStatus(tt.args.instance, tt.args.hoststatus, tt.args.statusDetails, tt.args.onboardingStatus)
		})
	}
}

func TestPopulateHostStatusDetail(t *testing.T) {
	type args struct {
		instance      *computev1.InstanceResource
		statusDetails string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test Case",
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
			PopulateHostStatusDetail(tt.args.instance, tt.args.statusDetails)
		})
	}
}

func TestPopulateInstanceStatusAndCurrentState(t *testing.T) {
	type args struct {
		instance           *computev1.InstanceResource
		currentState       computev1.InstanceState
		instancestatus     computev1.InstanceStatus
		provisioningStatus inv_status.ResourceStatus
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test Case",
			args: args{
				instance: &computev1.InstanceResource{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			PopulateInstanceStatusAndCurrentState(tt.args.instance, tt.args.currentState, tt.args.instancestatus, tt.args.provisioningStatus)
		})
	}
}

