// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package reconcilers

import (
	"context"
	"fmt"
	"time"

	rec_v2 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-app.lib-go/pkg/controller/v2"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	om_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/status"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/auth"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/flags"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/tracing"
)

const loggerName = "HostReconciler"

var zlogHost = logging.GetLogger(loggerName)

type HostReconciler struct {
	invClient     *invclient.OnboardingInventoryClient
	enableTracing bool
}

func NewHostReconciler(c *invclient.OnboardingInventoryClient, enableTracing bool) *HostReconciler {
	return &HostReconciler{
		invClient:     c,
		enableTracing: enableTracing,
	}
}

func (hr *HostReconciler) Reconcile(ctx context.Context,
	request rec_v2.Request[ResourceID],
) rec_v2.Directive[ResourceID] {
	if hr.enableTracing {
		ctx = tracing.StartTrace(ctx, "MIOnboardingManager", "HostReconciler")
		defer tracing.StopTrace(ctx)
	}
	resourceID := request.ID.String()
	zlogHost.Info().Msgf("Reconciling Host %s", resourceID)

	host, err := hr.invClient.GetHostResourceByResourceID(ctx, resourceID)
	if directive := HandleInventoryError(err, request); directive != nil {
		return directive
	}

	// Forbid Host provisioning with defined Provider. Such Host should be reconciled within Provider-specific RM.
	if host.GetProvider() != nil {
		zlogHost.Info().Msgf("Host %s should be reconciled within other vendor-specific RM (%s)",
			host.GetResourceId(), host.GetProvider().GetName())
		return request.Ack()
	}

	if host.DesiredState == host.CurrentState {
		zlogHost.Debug().Msgf("Host %s reconciliation skipped", resourceID)
		return request.Ack()
	}

	return hr.reconcileHost(ctx, request, host)
}

func (hr *HostReconciler) reconcileHost(
	ctx context.Context,
	request rec_v2.Request[ResourceID],
	host *computev1.HostResource,
) rec_v2.Directive[ResourceID] {
	zlogHost.Debug().Msgf("Reconciling host with ID %s, with Current state: %v, Desired state: %v.",
		host.GetResourceId(), host.GetCurrentState(), host.GetDesiredState())

	if host.GetDesiredState() == computev1.HostState_HOST_STATE_DELETED {
		err := hr.deleteHost(ctx, host)
		if directive := HandleInventoryError(err, request); directive != nil {
			return directive
		}
		zlogHost.Debug().Msgf("Host %v has been deleted", host.GetResourceId())
		return request.Ack()
	}

	if host.GetDesiredState() == computev1.HostState_HOST_STATE_UNTRUSTED {
		err := hr.invalidateHost(ctx, host)
		if directive := HandleInventoryError(err, request); directive != nil {
			return directive
		}
		zlogHost.Debug().Msgf("Host %v has been unauthorized", host.GetResourceId())
		return request.Ack()
	}

	return request.Ack()
}

func (hr *HostReconciler) checkIfInstanceIsAssociated(ctx context.Context, host *computev1.HostResource) error {
	if host.GetInstance() != nil {
		reconcErr := inv_errors.Errorf("Instance %s is still assigned to host %s, waiting for Instance to be deleted first",
			host.GetInstance().GetResourceId(), host.GetResourceId())
		zlogHost.Warn().Err(reconcErr).Msg("")

		details := fmt.Sprintf("waiting on %s deletion", host.GetInstance().GetResourceId())
		err := hr.invClient.SetHostStatusDetail(ctx, host.GetResourceId(),
			om_status.LegacyHostStatusDeletingWithDetails(details),
			om_status.ModernHostStatusDeletingWithDetails(details))
		if err != nil {
			// log debug message only in the case of failure
			zlogHost.Debug().Err(err).Msgf("Failed update status detail for host %s", host.GetResourceId())
		}

		return reconcErr
	}

	return nil
}

//nolint:cyclop // complexity is 11
func (hr *HostReconciler) deleteHost(
	ctx context.Context,
	host *computev1.HostResource,
) error {
	zlogHost.Info().Msgf("Deleting host ID %s (set current status Deleted)\n", host.GetResourceId())

	// if a host has still relationship with Instance, do not proceed with deletion.
	if err := hr.checkIfInstanceIsAssociated(ctx, host); err != nil {
		return err
	}

	// We need to set provider status detail to Deleting.
	// Since we don't have Deleting status, a possible state would be:
	// - legacy_host_status = RUNNING
	// - provider_status_detail = "Deleting: ..."
	// In such a case, UI will show message stored in provider_status_detail,
	// but if we clear the provider_status_detail field, a RUNNING status will be shown causing a poor UX.
	// Note that this behavior is valid only for legacy host status,
	// but we should keep the same behavior with modern statuses - otherwise, UI won't have any modern status to show.
	//nolint:staticcheck // provider status detail will be deprecated post-24.03.
	if host.GetProviderStatusDetail() != om_status.LegacyHostStatusDeleting {
		err := hr.invClient.SetHostStatusDetail(ctx, host.GetResourceId(),
			om_status.LegacyHostStatusDeleting, om_status.DeletingStatus)
		if err != nil {
			// log debug message only in the case of failure
			zlogHost.Debug().Err(err).Msgf("Failed update status detail for host %s", host.GetResourceId())
		}
	}

	// if the current state is Untrusted, host certificates are already revoked
	if host.GetCurrentState() != computev1.HostState_HOST_STATE_UNTRUSTED {
		if err := hr.revokeHostCredentials(ctx, host.GetUuid()); err != nil {
			return err
		}
	}

	// following functions are only modifying current state
	// we continue to delete other host objects in case of not found errors
	if err := hr.deleteHostNicByHost(ctx, host); err != nil {
		zlogHost.MiSec().MiError("Failed to delete host nic resource of Host (%s)", host.GetResourceId()).Msg("deleteHost")
		return err
	}

	if err := hr.deleteHostStorageByHost(ctx, host); err != nil {
		zlogHost.MiSec().MiError("Failed to delete host storage resource of Host (%s)",
			host.GetResourceId()).Msg("deleteHost")
		return err
	}

	if err := hr.deleteHostUsbByHost(ctx, host); err != nil {
		zlogHost.MiSec().MiError("Failed to delete host usb resource of Host (%s)", host.GetResourceId()).Msg("deleteHost")
		return err
	}

	if err := hr.deleteHostGpuByHost(ctx, host); err != nil {
		zlogHost.MiSec().MiError("Failed to delete host gpu resource of Host (%s)", host.GetResourceId()).Msg("deleteHost")
		return err
	}

	err := hr.invClient.DeleteHostResource(ctx, host.GetResourceId())
	if err != nil {
		zlogHost.MiSec().MiError("Failed to delete Host %s", host.GetResourceId()).Msg("deleteHost")
		// inventory error will be handled by upper layer
		return err
	}

	return nil
}

func (hr *HostReconciler) deleteHostGpuByHost(ctx context.Context, hostres *computev1.HostResource) error {
	// eager loaded from Host
	gpus := hostres.GetHostGpus()

	for _, gpu := range gpus {
		zlogHost.Debug().Msgf("Deleting host GPU with ID=%s", gpu.GetResourceId())
		err := hr.invClient.DeleteResource(ctx, gpu.GetResourceId())
		if err != nil {
			return err
		}
	}

	return nil
}

func (hr *HostReconciler) deleteHostNicByHost(ctx context.Context, hostres *computev1.HostResource) error {
	// eager loaded from Host
	nics := hostres.GetHostNics()

	for _, nic := range nics {
		// Firstly the IPAddresses due to the strong relation with nic
		if err := hr.deleteIPsByHostNic(ctx, nic); err != nil {
			return err
		}

		zlogHost.Debug().Msgf("Deleting host NIC with ID=%s", nic.GetResourceId())
		err := hr.invClient.DeleteResource(ctx, nic.GetResourceId())
		if err != nil {
			return err
		}
	}

	return nil
}

func (hr *HostReconciler) deleteIPsByHostNic(ctx context.Context, hostNic *computev1.HostnicResource) error {
	// IPs are not eager loaded
	nicIPs, err := hr.invClient.ListIPAddresses(ctx, hostNic)
	if err != nil {
		return err
	}

	for _, ip := range nicIPs {
		zlogHost.Debug().Msgf("Deleting IP address with ID=%s", ip.GetResourceId())
		err := hr.invClient.DeleteIPAddress(ctx, ip.GetResourceId())
		if err != nil {
			return err
		}
	}

	return nil
}

func (hr *HostReconciler) deleteHostStorageByHost(ctx context.Context, hostres *computev1.HostResource) error {
	// eager loaded from Host
	disks := hostres.GetHostStorages()

	for _, disk := range disks {
		zlogHost.Debug().Msgf("Deleting host storage with ID=%s", disk.GetResourceId())
		err := hr.invClient.DeleteResource(ctx, disk.GetResourceId())
		if err != nil {
			return err
		}
	}

	return nil
}

func (hr *HostReconciler) deleteHostUsbByHost(ctx context.Context, host *computev1.HostResource) error {
	usbs := host.GetHostUsbs()

	for _, usb := range usbs {
		zlogHost.Debug().Msgf("Deleting host USB with ID=%s", usb.GetResourceId())
		err := hr.invClient.DeleteResource(ctx, usb.GetResourceId())
		if err != nil {
			return err
		}
	}

	return nil
}

func (hr *HostReconciler) revokeHostCredentials(ctx context.Context, uuid string) error {
	if *flags.FlagDisableCredentialsManagement {
		zlogHost.Warn().Msgf("disableCredentialsManagement flag is set to false, " +
			"skip credentials revocation")
		return nil
	}

	authService, err := auth.AuthServiceFactory(ctx)
	if err != nil {
		return err
	}
	defer authService.Logout(ctx)

	if revokeErr := authService.RevokeCredentialsByUUID(ctx, uuid); revokeErr != nil && !inv_errors.IsNotFound(revokeErr) {
		zlogHost.MiSec().MiError("Failed to revoke credentials of host %s.", uuid).Msg("revokeHostCredentials")
		return inv_errors.Wrap(revokeErr)
	}

	return nil
}

func (hr *HostReconciler) invalidateHost(ctx context.Context, host *computev1.HostResource) error {
	zlogHost.Debug().Msgf("Invalidating Host %s", host.GetResourceId())

	if err := hr.revokeHostCredentials(ctx, host.GetUuid()); err != nil {
		return err
	}

	untrustedHost := computev1.HostResource{
		ResourceId:          host.GetResourceId(),
		CurrentState:        computev1.HostState_HOST_STATE_UNTRUSTED,
		LegacyHostStatus:    computev1.HostStatus_HOST_STATUS_INVALIDATED,
		ProviderStatus:      computev1.HostStatus_name[int32(computev1.HostStatus_HOST_STATUS_INVALIDATED)],
		HostStatus:          om_status.AuthorizationStatusInvalidated.Status,
		HostStatusIndicator: om_status.AuthorizationStatusInvalidated.StatusIndicator,
		HostStatusTimestamp: uint64(time.Now().Unix()),
	}

	// Although Onboarding Manager should not update host_status that is updated by HRM,
	// the host authorization status (being a host_status) must be updated by OM, because
	// OM is the only source of truth for state reconciliation. Anyway, this operation is safe to
	// HRM because once the state is moved to UNTRUSTED, HRM won't perform any runtime status update.
	if err := hr.invClient.UpdateHostStateAndRuntimeStatus(ctx, &untrustedHost); err != nil {
		zlogHost.MiSec().MiError("Failed to update host state and status").Msg("invalidateHost")
		return err
	}

	zlogHost.MiSec().Info().Msgf("Host %s is invalidated", host.GetHostname())
	return nil
}
