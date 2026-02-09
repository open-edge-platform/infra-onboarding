// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// Package status provides functionality for onboarding management.
package status

import (
	"fmt"

	statusv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/status/v1"
	inv_status "github.com/open-edge-platform/infra-core/inventory/v2/pkg/status"
)

var (
	// LegacyHostStatusDeleting defines a configuration value.
	LegacyHostStatusDeleting = "Deleting"

	// InstanceStatusUnknown defines a configuration value.
	// resource statuses for Instance.
	InstanceStatusUnknown = inv_status.New("Unknown", statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED)
	// ProvisioningStatusUnknown defines a configuration value.
	ProvisioningStatusUnknown = inv_status.New("Unknown", statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED)
	// ProvisioningStatusInProgress defines a configuration value.
	ProvisioningStatusInProgress = inv_status.New("Provisioning In Progress",
		statusv1.StatusIndication_STATUS_INDICATION_IN_PROGRESS)
	// ProvisioningStatusFailed defines a configuration value.
	ProvisioningStatusFailed = inv_status.New("Provisioning Failed", statusv1.StatusIndication_STATUS_INDICATION_ERROR)
	// ProvisioningStatusDone defines a configuration value.
	ProvisioningStatusDone = inv_status.New("Provisioned", statusv1.StatusIndication_STATUS_INDICATION_IDLE)
	// UpdateStatusUnknown defines a configuration value.
	UpdateStatusUnknown = inv_status.New("Unknown", statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED)
	// TrustedAttestationStatusUnknown defines a configuration value.
	TrustedAttestationStatusUnknown = inv_status.New("Unknown", statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED)

	// InitializationDone defines a configuration value.
	// resource statuses for Host.
	InitializationDone = inv_status.New("Device initialized", statusv1.StatusIndication_STATUS_INDICATION_IDLE)
	// InitializationFailed defines a configuration value.
	InitializationFailed = inv_status.New("Device initialization failed",
		statusv1.StatusIndication_STATUS_INDICATION_ERROR)
	// OnboardingStatusUnknown defines a configuration value.
	OnboardingStatusUnknown = inv_status.New("Unknown", statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED)
	// OnboardingStatusDone defines a configuration value.
	OnboardingStatusDone = inv_status.New("Onboarded", statusv1.StatusIndication_STATUS_INDICATION_IDLE)
	// OnboardingStatusFailed defines a configuration value.
	OnboardingStatusFailed = inv_status.New("Onboarding Failed", statusv1.StatusIndication_STATUS_INDICATION_ERROR)
	// AuthorizationStatusInvalidated defines a configuration value.
	AuthorizationStatusInvalidated = inv_status.New("Invalidated", statusv1.StatusIndication_STATUS_INDICATION_IDLE)

	// HostRegistrationUnknown defines a configuration value.
	HostRegistrationUnknown = inv_status.New("Unknown", statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED)
	// HostRegistrationDone defines a configuration value.
	HostRegistrationDone = inv_status.New("Host is Registered", statusv1.StatusIndication_STATUS_INDICATION_IDLE)
	// HostRegistrationInFailed defines a configuration value.
	HostRegistrationInFailed = inv_status.New("Host Registration Failed", statusv1.StatusIndication_STATUS_INDICATION_ERROR)

	// HostStatusRebooting defines a configuration value.
	HostStatusRebooting = inv_status.New("Rebooting", statusv1.StatusIndication_STATUS_INDICATION_IN_PROGRESS)

	// DeletingStatus defines a configuration value.
	DeletingStatus = inv_status.New("Deleting", statusv1.StatusIndication_STATUS_INDICATION_IN_PROGRESS)
)

// WithDetails performs operations for onboarding management.
func WithDetails(status inv_status.ResourceStatus, details string) inv_status.ResourceStatus {
	return inv_status.New(fmt.Sprintf("%s: %s", status.Status, details), status.StatusIndicator)
}

// LegacyHostStatusDeletingWithDetails performs operations for onboarding management.
func LegacyHostStatusDeletingWithDetails(detail string) string {
	return fmt.Sprintf("%s: %s", LegacyHostStatusDeleting, detail)
}

// ModernHostStatusDeletingWithDetails performs operations for onboarding management.
func ModernHostStatusDeletingWithDetails(detail string) inv_status.ResourceStatus {
	return inv_status.New(LegacyHostStatusDeletingWithDetails(detail), statusv1.StatusIndication_STATUS_INDICATION_IN_PROGRESS)
}

// NewStatusWithDetails performs operations for onboarding management.
func NewStatusWithDetails(baseStatus inv_status.ResourceStatus, details string) inv_status.ResourceStatus {
	if details == "" {
		return baseStatus
	}
	return inv_status.ResourceStatus{
		Status:          fmt.Sprintf("%s: %s", baseStatus.Status, details),
		StatusIndicator: baseStatus.StatusIndicator,
	}
}

// NewHostRegistrationUUIDFailed performs operations for onboarding management.
func NewHostRegistrationUUIDFailed() inv_status.ResourceStatus {
	return inv_status.New("Host Registration Failed due to mismatch of UUID, Reported UUID is",
		statusv1.StatusIndication_STATUS_INDICATION_ERROR)
}

// NewHostRegistrationSerialNumFailed performs operations for onboarding management.
func NewHostRegistrationSerialNumFailed() inv_status.ResourceStatus {
	return inv_status.New("Host Registration Failed due to mismatch of Serial Number, Reported Serial Number is",
		statusv1.StatusIndication_STATUS_INDICATION_ERROR)
}

// HostRegistrationUUIDFailedWithDetails performs operations for onboarding management.
func HostRegistrationUUIDFailedWithDetails(detail string) inv_status.ResourceStatus {
	return inv_status.New(
		fmt.Sprintf("%s: %s", NewHostRegistrationUUIDFailed().Status, detail),
		statusv1.StatusIndication_STATUS_INDICATION_ERROR,
	)
}

// HostRegistrationSerialNumFailedWithDetails performs operations for onboarding management.
func HostRegistrationSerialNumFailedWithDetails(detail string) inv_status.ResourceStatus {
	return inv_status.New(
		fmt.Sprintf("%s: %s", NewHostRegistrationSerialNumFailed().Status, detail),
		statusv1.StatusIndication_STATUS_INDICATION_ERROR,
	)
}
