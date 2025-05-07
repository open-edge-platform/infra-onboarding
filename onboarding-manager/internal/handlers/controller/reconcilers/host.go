// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package reconcilers

import (
	"context"
	"fmt"
	"time"

	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	kk_auth "github.com/open-edge-platform/infra-core/inventory/v2/pkg/auth"
	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/tracing"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/invclient"
	om_status "github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/status"
	rec_v2 "github.com/open-edge-platform/orch-library/go/pkg/controller/v2"
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
	request rec_v2.Request[ReconcilerID],
) rec_v2.Directive[ReconcilerID] {
	if hr.enableTracing {
		ctx = tracing.StartTrace(ctx, "InfraOnboardingManager", "HostReconciler")
		defer tracing.StopTrace(ctx)
	}
	resourceID := request.ID.GetResourceID()
	tenantID := request.ID.GetTenantID()
	zlogHost.Info().Msgf("Reconciling Host")
	zlogHost.Debug().Msgf("Reconciling Host resourceID %s and tenantID %s", resourceID, tenantID)

	host, err := hr.invClient.GetHostResourceByResourceID(ctx, tenantID, resourceID)
	if directive := HandleInventoryError(err, request); directive != nil {
		return directive
	}

	// Forbid Host provisioning with defined Provider. Such Host should be reconciled within Provider-specific RM.
	if host.GetProvider() != nil {
		zlogHost.Info().Msgf("Host should be reconciled within other vendor-specific RM (%s)", host.GetProvider().GetName())
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
	request rec_v2.Request[ReconcilerID],
	host *computev1.HostResource,
) rec_v2.Directive[ReconcilerID] {
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
		reconcErr := inv_errors.Errorf("Instance is still assigned to host, waiting for Instance to be deleted first")
		zlogHost.Debug().Err(reconcErr).Msgf("Instance %s  host %s", host.GetInstance().GetResourceId(), host.GetResourceId())
		zlogHost.Warn().Err(reconcErr).Msg("")

		details := fmt.Sprintf("waiting on %s deletion", host.GetInstance().GetResourceId())
		err := hr.invClient.SetHostStatusDetail(ctx, host.GetTenantId(), host.GetResourceId(),
			om_status.ModernHostStatusDeletingWithDetails(details))
		if err != nil {
			// log debug message only in the case of failure
			zlogHost.Debug().Err(err).Msgf("Failed update status detail for host %s", host.GetResourceId())
		}

		return reconcErr
	}

	return nil
}

func (hr *HostReconciler) deleteHost(
	ctx context.Context,
	host *computev1.HostResource,
) error {
	zlogHost.Debug().Msgf("Deleting host ID %s (set current status Deleted)\n", host.GetResourceId())

	// if a host has still relationship with Instance, do not proceed with deletion.
	if err := hr.checkIfInstanceIsAssociated(ctx, host); err != nil {
		return err
	}

	if err := hr.invClient.SetHostStatusDetail(ctx, host.GetTenantId(), host.GetResourceId(),
		om_status.DeletingStatus); err != nil {
		// log debug message only in the case of failure
		zlogHost.Debug().Err(err).Msgf("Failed to update status detail for host %s", host.GetResourceId())
	}

	// if the current state is Untrusted, host certificates are already revoked
	if host.GetCurrentState() != computev1.HostState_HOST_STATE_UNTRUSTED {
		if err := kk_auth.RevokeHostCredentials(ctx, host.GetTenantId(), host.GetUuid()); err != nil {
			return err
		}
	}

	// following functions are only modifying current state
	// we continue to delete other host objects in case of not found errors
	if err := hr.deleteHostNicByHost(ctx, host); err != nil {
		zlogHost.InfraSec().InfraError("Failed to delete host nic resource of Host").Msg("deleteHost")
		return err
	}

	if err := hr.deleteHostStorageByHost(ctx, host); err != nil {
		zlogHost.InfraSec().InfraError("Failed to delete host storage resource of Host").Msg("deleteHost")
		return err
	}

	if err := hr.deleteHostUsbByHost(ctx, host); err != nil {
		zlogHost.InfraSec().InfraError("Failed to delete host usb resource of Host").Msg("deleteHost")
		return err
	}

	if err := hr.deleteHostGpuByHost(ctx, host); err != nil {
		zlogHost.InfraSec().InfraError("Failed to delete host gpu resource of Host").Msg("deleteHost")
		return err
	}

	err := hr.invClient.DeleteHostResource(ctx, host.GetTenantId(), host.GetResourceId())
	if err != nil {
		zlogHost.InfraSec().InfraError("Failed to delete Host").Msg("deleteHost")
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
		err := hr.invClient.DeleteResource(ctx, gpu.GetTenantId(), gpu.GetResourceId())
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
		err := hr.invClient.DeleteResource(ctx, nic.GetTenantId(), nic.GetResourceId())
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
		err := hr.invClient.DeleteIPAddress(ctx, ip.GetTenantId(), ip.GetResourceId())
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
		err := hr.invClient.DeleteResource(ctx, disk.GetTenantId(), disk.GetResourceId())
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
		err := hr.invClient.DeleteResource(ctx, usb.GetTenantId(), usb.GetResourceId())
		if err != nil {
			return err
		}
	}

	return nil
}

func (hr *HostReconciler) invalidateHost(ctx context.Context, host *computev1.HostResource) error {
	zlogHost.Debug().Msgf("Invalidating Host %s", host.GetResourceId())

	if err := kk_auth.RevokeHostCredentials(ctx, host.GetTenantId(), host.GetUuid()); err != nil {
		return err
	}

	untrustedHost := computev1.HostResource{
		ResourceId:          host.GetResourceId(),
		CurrentState:        computev1.HostState_HOST_STATE_UNTRUSTED,
		HostStatus:          om_status.AuthorizationStatusInvalidated.Status,
		HostStatusIndicator: om_status.AuthorizationStatusInvalidated.StatusIndicator, // #nosec G115
		HostStatusTimestamp: uint64(time.Now().Unix()),                                // #nosec G115
	}

	// Although Onboarding Manager should not update host_status that is updated by HRM,
	// the host authorization status (being a host_status) must be updated by OM, because
	// OM is the only source of truth for state reconciliation. Anyway, this operation is safe to
	// HRM because once the state is moved to UNTRUSTED, HRM won't perform any runtime status update.
	if err := hr.invClient.UpdateHostStateAndRuntimeStatus(ctx, host.GetTenantId(), &untrustedHost); err != nil {
		zlogHost.InfraSec().InfraError("Failed to update host state and status").Msg("invalidateHost")
		return err
	}

	zlogHost.InfraSec().Info().Msgf("Host %s is invalidated", host.GetHostname())
	return nil
}
