// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package util

import (
	"os"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/status"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/util"
)

const (
	EnvImageType     = "IMAGE_TYPE"
	EnvDiskType      = "DISK_PARTITION"
	EnvImgURL        = "IMG_URL"
	EnvProvisionerIP = "PD_IP"
	EnvOverlayURL    = "OVERLAY_URL"
	EnvFdoMfgDNS     = "FDO_MFG_URL"
	EnvFdoMfgPort    = "FDO_MFG_PORT"
	EnvFdoOwnerDNS   = "FDO_OWNER_URL"
	EnvFdoOwnerPort  = "FDO_OWNER_PORT"
	EnvFdoRvPort     = "FDO_RV_PORT"
	EnvK8sNamespace  = "MI_K8S_NAMESPACE"

	defaultOwnerURL  = "mi-fdo-owner"
	defaultOwnerPort = "58042"
	defaultMfgURL    = "mi-fdo-mfg"
	defaultMfgPort   = "58039"
)

var (
	DiskType      = os.Getenv(EnvDiskType)
	ImgURL        = os.Getenv(EnvImgURL)
	ProvisionerIP = os.Getenv(EnvProvisionerIP)
	OverlayURL    = os.Getenv(EnvOverlayURL)
	FdoRvPort     = os.Getenv(EnvFdoRvPort)
	FdoMfgDNS     = GetEnvWithDefault(EnvFdoMfgDNS, defaultMfgURL)
	FdoMfgPort    = GetEnvWithDefault(EnvFdoMfgPort, defaultMfgPort)
	FdoOwnerDNS   = GetEnvWithDefault(EnvFdoOwnerDNS, defaultOwnerURL)
	FdoOwnerPort  = GetEnvWithDefault(EnvFdoOwnerPort, defaultOwnerPort)
	K8sNamespace  = GetEnvWithDefault(EnvK8sNamespace, "maestro-iaas-system")
)

func GetEnvWithDefault(key, defaultVal string) func() string {
	return func() string {
		v, found := os.LookupEnv(key)
		if found && v != "" {
			return v
		}
		return defaultVal
	}
}

func IsSameHostStatus(
	oldHost *computev1.HostResource,
	newHost *computev1.HostResource,
) bool {
	return oldHost.LegacyHostStatus == newHost.LegacyHostStatus && //nolint:staticcheck // this field will be deprecated soon
		oldHost.ProviderStatusDetail == newHost.ProviderStatusDetail && //nolint:staticcheck // this field will be deprecated soon
		oldHost.OnboardingStatusIndicator == newHost.OnboardingStatusIndicator &&
		oldHost.OnboardingStatus == newHost.OnboardingStatus
}

func IsSameInstanceStatusAndState(
	oldInstance *computev1.InstanceResource,
	newInstance *computev1.InstanceResource,
) bool {
	return oldInstance.Status == newInstance.Status && //nolint:staticcheck // this field will be deprecated soon
		oldInstance.CurrentState == newInstance.CurrentState &&
		oldInstance.ProvisioningStatus == newInstance.ProvisioningStatus &&
		oldInstance.ProvisioningStatusIndicator == newInstance.ProvisioningStatusIndicator
}

func PopulateHostStatus(
	instance *computev1.InstanceResource,
	hoststatus computev1.HostStatus,
	statusDetails string,
	onboardingStatus inv_status.ResourceStatus,
) {
	host := instance.GetHost()                // eager-loaded
	host.LegacyHostStatus = hoststatus        //nolint:staticcheck // this field will be deprecated soon
	host.ProviderStatusDetail = statusDetails //nolint:staticcheck // this field will be deprecated soon
	host.OnboardingStatus = onboardingStatus.Status
	host.OnboardingStatusIndicator = onboardingStatus.StatusIndicator
}

func PopulateHostStatusDetail(
	instance *computev1.InstanceResource,
	statusDetails string,
) {
	host := instance.GetHost() // eager-loaded

	host.ProviderStatusDetail = statusDetails //nolint:staticcheck // this field will be deprecated soon
}

func PopulateInstanceStatusAndCurrentState(
	instance *computev1.InstanceResource,
	currentState computev1.InstanceState,
	instancestatus computev1.InstanceStatus,
	provisioningStatus inv_status.ResourceStatus,
) {
	instance.CurrentState = currentState
	instance.Status = instancestatus //nolint:staticcheck // this field will be deprecated soon
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
