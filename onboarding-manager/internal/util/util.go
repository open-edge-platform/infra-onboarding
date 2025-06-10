// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"encoding/json"

	"google.golang.org/grpc/codes"

	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	_ "github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging" // include to pass tests with -globalLogLevel
	inv_status "github.com/open-edge-platform/infra-core/inventory/v2/pkg/status"
	om_status "github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/status"
)

const (
	IsStandaloneMetadataKey = "standalone-node"
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

func PopulateInstanceStatus(
	instance *computev1.InstanceResource,
	instanceStatus inv_status.ResourceStatus,
) {
	instance.InstanceStatus = instanceStatus.Status
	instance.InstanceStatusIndicator = instanceStatus.StatusIndicator
	instance.InstanceStatusDetail = ""
}

func PopulateInstanceUpdateStatus(
	instance *computev1.InstanceResource,
	updateStatus inv_status.ResourceStatus,
) {
	instance.UpdateStatus = updateStatus.Status
	instance.UpdateStatusIndicator = updateStatus.StatusIndicator
	instance.UpdateStatusDetail = ""
}

func PopulateInstanceTrustedAttestationStatus(
	instance *computev1.InstanceResource,
	trustedAttestationStatus inv_status.ResourceStatus,
) {
	instance.TrustedAttestationStatus = trustedAttestationStatus.Status
	instance.TrustedAttestationStatusIndicator = trustedAttestationStatus.StatusIndicator
}

func PopulateInstanceIdleStatus(
	instance *computev1.InstanceResource,
) {
	PopulateInstanceStatus(instance, om_status.InstanceStatusUnknown)
	PopulateInstanceProvisioningStatus(instance, om_status.ProvisioningStatusUnknown)
	PopulateInstanceUpdateStatus(instance, om_status.UpdateStatusUnknown)
	PopulateInstanceTrustedAttestationStatus(instance, om_status.TrustedAttestationStatusUnknown)
}

func PopulateCurrentOS(instance *computev1.InstanceResource, osResourceID string) {
	instance.CurrentOs = &osv1.OperatingSystemResource{ResourceId: osResourceID}
}

func IsStandalone(instance *computev1.InstanceResource) (bool, error) {
	if instance.GetDesiredOs() == nil {
		return false, nil
	}

	osMetadata := instance.GetDesiredOs().GetMetadata()

	var jsonMap map[string]string
	err := json.Unmarshal([]byte(osMetadata), &jsonMap)
	if err != nil {
		return false, inv_errors.Errorfc(codes.InvalidArgument, "Failed to parse JSON map: %v", err)
	}

	isStandaloneMdValue, exists := jsonMap[IsStandaloneMetadataKey]
	if !exists {
		// treat as non-standalone if metadata not included
		return false, nil
	}

	return isStandaloneMdValue == "true", nil
}
