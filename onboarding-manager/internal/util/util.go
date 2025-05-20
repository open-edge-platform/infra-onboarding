// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package util

import (
	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
	_ "github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging" // include to pass tests with -globalLogLevel
	inv_status "github.com/open-edge-platform/infra-core/inventory/v2/pkg/status"
)

func IsSameHostStatus(
	oldHost *computev1.HostResource,
	newHost *computev1.HostResource,
) bool {
	return oldHost.HostStatusIndicator == newHost.HostStatusIndicator &&
		oldHost.HostStatus == newHost.HostStatus
}

func IsSameOnboardingStatus(
	oldHost *computev1.HostResource,
	newHost *computev1.HostResource,
) bool {
	return oldHost.OnboardingStatusIndicator == newHost.OnboardingStatusIndicator &&
		oldHost.OnboardingStatus == newHost.OnboardingStatus
}

func IsSameInstanceStatusAndState(
	oldInstance *computev1.InstanceResource,
	newInstance *computev1.InstanceResource,
) bool {
	return oldInstance.CurrentState == newInstance.CurrentState &&
		oldInstance.ProvisioningStatus == newInstance.ProvisioningStatus &&
		oldInstance.ProvisioningStatusIndicator == newInstance.ProvisioningStatusIndicator
}

func PopulateHostStatus(
	instance *computev1.InstanceResource,
	hostStatus inv_status.ResourceStatus,
) {
	host := instance.GetHost() // eager-loaded
	host.HostStatus = hostStatus.Status
	host.HostStatusIndicator = hostStatus.StatusIndicator
}

func PopulateHostOnboardingStatus(
	instance *computev1.InstanceResource,
	onboardingStatus inv_status.ResourceStatus,
) {
	host := instance.GetHost() // eager-loaded
	host.OnboardingStatus = onboardingStatus.Status
	host.OnboardingStatusIndicator = onboardingStatus.StatusIndicator
}

func PopulateInstanceProvisioningStatus(
	instance *computev1.InstanceResource,
	provisioningStatus inv_status.ResourceStatus,
) {
	instance.ProvisioningStatus = provisioningStatus.Status
	instance.ProvisioningStatusIndicator = provisioningStatus.StatusIndicator
}

func PopulateInstanceStatusAndCurrentState(
	instance *computev1.InstanceResource,
	currentState computev1.InstanceState,
	provisioningStatus inv_status.ResourceStatus,
) {
	instance.CurrentState = currentState
	instance.ProvisioningStatus = provisioningStatus.Status
	instance.ProvisioningStatusIndicator = provisioningStatus.StatusIndicator
}

func PopulateCurrentOS(instance *computev1.InstanceResource, osResourceID string) {
	instance.CurrentOs = &osv1.OperatingSystemResource{ResourceId: osResourceID}
}
