// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package status

import (
	"fmt"

	statusv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/api/status/v1"
	inv_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/status"
)

var (
	LegacyHostStatusDeleting = "Deleting"

	// resource statuses for Instance.
	ProvisioningStatusUnknown    = inv_status.New("Unknown", statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED)
	ProvisioningStatusInProgress = inv_status.New("Provisioning In Progress",
		statusv1.StatusIndication_STATUS_INDICATION_IN_PROGRESS)
	ProvisioningStatusFailed = inv_status.New("Provisioning Failed", statusv1.StatusIndication_STATUS_INDICATION_ERROR)
	ProvisioningStatusDone   = inv_status.New("Provisioned", statusv1.StatusIndication_STATUS_INDICATION_IDLE)

	// resource statuses for Host.
	InitializationInProgress = inv_status.New("Device initializing",
		statusv1.StatusIndication_STATUS_INDICATION_IN_PROGRESS)
	InitializationDone   = inv_status.New("Device initialized", statusv1.StatusIndication_STATUS_INDICATION_IDLE)
	InitializationFailed = inv_status.New("Device initialization failed",
		statusv1.StatusIndication_STATUS_INDICATION_ERROR)
	OnboardingStatusBooting        = inv_status.New("Booting", statusv1.StatusIndication_STATUS_INDICATION_IN_PROGRESS)
	OnboardingStatusInProgress     = inv_status.New("Onboarding", statusv1.StatusIndication_STATUS_INDICATION_IN_PROGRESS)
	OnboardingStatusDone           = inv_status.New("Onboarded", statusv1.StatusIndication_STATUS_INDICATION_IDLE)
	OnboardingStatusFailed         = inv_status.New("Onboarding Failed", statusv1.StatusIndication_STATUS_INDICATION_ERROR)
	AuthorizationStatusInvalidated = inv_status.New("Invalidated", statusv1.StatusIndication_STATUS_INDICATION_IDLE)

	HostRegistrationUnknown  = inv_status.New("Unknown", statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED)
	HostRegistrationDone     = inv_status.New("Host is Registered", statusv1.StatusIndication_STATUS_INDICATION_IDLE)
	HostRegistrationInFailed = inv_status.New("Host Registration Failed", statusv1.StatusIndication_STATUS_INDICATION_ERROR)

	DeletingStatus = inv_status.New("Deleting", statusv1.StatusIndication_STATUS_INDICATION_IN_PROGRESS)
)

func WithDetails(status inv_status.ResourceStatus, details string) inv_status.ResourceStatus {
	return inv_status.New(fmt.Sprintf("%s: %s", status.Status, details), status.StatusIndicator)
}

func LegacyHostStatusDeletingWithDetails(detail string) string {
	return fmt.Sprintf("%s: %s", LegacyHostStatusDeleting, detail)
}

func ModernHostStatusDeletingWithDetails(detail string) inv_status.ResourceStatus {
	return inv_status.New(LegacyHostStatusDeletingWithDetails(detail), statusv1.StatusIndication_STATUS_INDICATION_IN_PROGRESS)
}

func NewStatusWithDetails(baseStatus inv_status.ResourceStatus, details string) inv_status.ResourceStatus {
	if details == "" {
		return baseStatus
	}
	return inv_status.ResourceStatus{
		Status:          fmt.Sprintf("%s: %s", baseStatus.Status, details),
		StatusIndicator: baseStatus.StatusIndicator,
	}
}

func NewHostRegistrationUUIDFailed() inv_status.ResourceStatus {
	return inv_status.New("Host Registration Failed due to mismatch of UUID, Reported UUID is",
		statusv1.StatusIndication_STATUS_INDICATION_ERROR)
}

func NewHostRegistrationSerialNumFailed() inv_status.ResourceStatus {
	return inv_status.New("Host Registration Failed due to mismatch of Serial Number, Reported Serial Number is",
		statusv1.StatusIndication_STATUS_INDICATION_ERROR)
}

func HostRegistrationUUIDFailedWithDetails(detail string) inv_status.ResourceStatus {
	return inv_status.New(
		fmt.Sprintf("%s: %s", NewHostRegistrationUUIDFailed().Status, detail),
		statusv1.StatusIndication_STATUS_INDICATION_ERROR,
	)
}

func HostRegistrationSerialNumFailedWithDetails(detail string) inv_status.ResourceStatus {
	return inv_status.New(
		fmt.Sprintf("%s: %s", NewHostRegistrationSerialNumFailed().Status, detail),
		statusv1.StatusIndication_STATUS_INDICATION_ERROR,
	)
}
