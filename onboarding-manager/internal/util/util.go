// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	computev1 "github.com/intel/infra-core/inventory/v2/pkg/api/compute/v1"
	osv1 "github.com/intel/infra-core/inventory/v2/pkg/api/os/v1"
	inv_errors "github.com/intel/infra-core/inventory/v2/pkg/errors"
	inv_status "github.com/intel/infra-core/inventory/v2/pkg/status"
	"github.com/intel/infra-core/inventory/v2/pkg/util"
	"github.com/intel/infra-onboarding/onboarding-manager/internal/onboardingmgr/utils"
)

func GetImageTypeFromOsType(osType osv1.OsType) (string, error) {
	switch osType {
	case osv1.OsType_OS_TYPE_IMMUTABLE:
		return utils.ImgTypeTiberMicrovisor, nil
	case osv1.OsType_OS_TYPE_MUTABLE:
		return utils.ImgTypeUbuntu, nil
	default:
		return "", inv_errors.Errorf("Unknown OS type %T", osType)
	}
}

func IsSameHostStatus(
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

func PopulateHostOnboardingStatus(
	instance *computev1.InstanceResource,
	onboardingStatus inv_status.ResourceStatus,
) {
	host := instance.GetHost() // eager-loaded
	host.OnboardingStatus = onboardingStatus.Status
	host.OnboardingStatusIndicator = onboardingStatus.StatusIndicator
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

func IsSameHost(
	originalHostres *computev1.HostResource,
	updatedHostres *computev1.HostResource,
	fieldmask *fieldmaskpb.FieldMask,
) (bool, error) {
	// firstly, cloning Host resource to avoid changing its content
	clonedHostres := proto.Clone(originalHostres)
	// with the fieldmask we are filtering out the fields we don't need
	err := util.ValidateMaskAndFilterMessage(clonedHostres, fieldmask, true)
	if err != nil {
		return false, err
	}

	return proto.Equal(clonedHostres, updatedHostres), nil
}

func PopulateCurrentOS(instance *computev1.InstanceResource, osResourceID string) {
	instance.CurrentOs = &osv1.OperatingSystemResource{ResourceId: osResourceID}
}
