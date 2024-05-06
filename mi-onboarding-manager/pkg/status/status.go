// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package status

import (
	"fmt"

	statusv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/status/v1"
	inv_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/status"
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
	OnboardingStatusFailed         = inv_status.New("Error", statusv1.StatusIndication_STATUS_INDICATION_ERROR)
	AuthorizationStatusInvalidated = inv_status.New("Invalidated", statusv1.StatusIndication_STATUS_INDICATION_IDLE)

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
