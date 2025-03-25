// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package reconcilers

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	grpc_status "google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	inv_status "github.com/open-edge-platform/infra-core/inventory/v2/pkg/status"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/tracing"
	dkam_util "github.com/open-edge-platform/infra-onboarding/dkam/pkg/util"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/env"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/invclient"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/onboarding"
	onboarding_types "github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/onboarding/types"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/util"
	om_status "github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/status"
	rec_v2 "github.com/open-edge-platform/orch-library/go/pkg/controller/v2"
)

const (
	instanceReconcilerLoggerName = "InstanceReconciler"

	TinkStackURLTemplate = "http://%s/tink-stack"
)

// Misc variables.
var (
	zlogInst = logging.GetLogger(instanceReconcilerLoggerName)
)

type InstanceReconciler struct {
	invClient     *invclient.OnboardingInventoryClient
	enableTracing bool
}

func NewInstanceReconciler(c *invclient.OnboardingInventoryClient, enableTracing bool) *InstanceReconciler {
	return &InstanceReconciler{
		invClient:     c,
		enableTracing: enableTracing,
	}
}

func (ir *InstanceReconciler) Reconcile(ctx context.Context,
	request rec_v2.Request[ReconcilerID],
) rec_v2.Directive[ReconcilerID] {
	if ir.enableTracing {
		ctx = tracing.StartTrace(ctx, "InfraOnboardingManager", "InstanceReconciler")
		defer tracing.StopTrace(ctx)
	}
	resourceID := request.ID.GetResourceID()
	tenantID := request.ID.GetTenantID()
	zlogInst.Info().Msgf("Reconciling Instance")
	zlogInst.Debug().Msgf("Reconciling Instance resourceID (%s) and tenantID (%s)", resourceID, tenantID)

	instance, err := ir.invClient.GetInstanceResourceByResourceID(ctx, tenantID, resourceID)
	if directive := HandleInventoryError(err, request); directive != nil {
		return directive
	}

	// the only allowed path from the ERROR state is DELETED
	if directive := ir.handleErrorState(instance, request); directive != nil {
		return directive
	}

	// Forbid Instance provisioning with defined Provider. Such Instance should be reconciled within Provider-specific RM.
	if directive := ir.handleProviderSpecificRM(instance, request); directive != nil {
		return directive
	}

	if directive := ir.handleMatchingStates(ctx, instance, request, resourceID); directive != nil {
		return directive
	}

	if directive := ir.handleHostOnboarded(instance, request); directive != nil {
		return directive
	}

	return ir.reconcileInstance(ctx, request, instance)
}

func (ir *InstanceReconciler) handleHostOnboarded(instance *computev1.InstanceResource, request rec_v2.Request[ReconcilerID],
) rec_v2.Directive[ReconcilerID] {
	if instance.GetDesiredState() != computev1.InstanceState_INSTANCE_STATE_RUNNING ||
		instance.GetHost().GetCurrentState() == computev1.HostState_HOST_STATE_ONBOARDED {
		// Proceed with provisioning only if the host is already onboarded.
		return nil
	}
	zlogInst.Info().Msgf("Host is not yet onboarded. Reconciliation will be skipped until the host is onboarded. hostUUID=%s",
		instance.GetHost().GetUuid(),
	)
	// TODO: currently we ack the request, but we should consider retrying the reconciliation, for example for a fixed
	//  amount of times.
	return request.Ack()
}

func (ir *InstanceReconciler) handleErrorState(instance *computev1.InstanceResource, request rec_v2.Request[ReconcilerID],
) rec_v2.Directive[ReconcilerID] {
	if instance.GetProvisioningStatusIndicator() == om_status.ProvisioningStatusFailed.StatusIndicator &&
		instance.DesiredState != computev1.InstanceState_INSTANCE_STATE_DELETED {
		// ProvisioningStatusIndicator is set to ERROR by previous reconciliation cycles
		// We don't have auto-recovery mechanisms. The previous reconciliation cycle should
		// set providerStatusDetail to provide feedback to user.
		// ATM I (Tomasz) believe that a user should delete via UI and re-configure host again,
		// once the issue is fixed (e.g., wrong BIOS settings, etc.)
		zlogInst.Warn().Msgf(
			"Provisioning status is failed. Reconciliation won't happen until the Instance is re-created.")
		return request.Ack()
	}
	return nil
}

func (ir *InstanceReconciler) handleProviderSpecificRM(instance *computev1.InstanceResource, request rec_v2.Request[ReconcilerID],
) rec_v2.Directive[ReconcilerID] {
	if instance.GetHost() != nil && instance.GetHost().GetProvider() != nil {
		zlogInst.Info().Msgf("Instance should be reconciled within other vendor-specific RM (%s)",
			instance.GetHost().GetProvider().GetName())
		return request.Ack()
	}
	return nil
}

func (ir *InstanceReconciler) handleMatchingStates(ctx context.Context, instance *computev1.InstanceResource,
	request rec_v2.Request[ReconcilerID], resourceID string,
) rec_v2.Directive[ReconcilerID] {
	if instance.DesiredState == instance.CurrentState {
		// HRM may already update the state to RUNNING before provisioning is done (see NEX-15924).
		// In such case, we let reconciler complete the provisioning process and clean up resources.
		// TODO (NEX-16077): a clean solution should be to update provisioning status and clean resources
		//  based on events from Tinkerbell CRDs.
		if instance.GetCurrentState() == computev1.InstanceState_INSTANCE_STATE_RUNNING &&
			instance.GetProvisioningStatusIndicator() != om_status.ProvisioningStatusDone.StatusIndicator &&
			instance.GetProvisioningStatus() != om_status.ProvisioningStatusDone.Status {
			zlogInst.Info().Msgf("Instance (%s) is in RUNNING state but provisioning status is not done."+
				" Forcing reconciliation to finish provisioning.",
				resourceID)
			return ir.reconcileInstance(ctx, request, instance)
		}

		zlogInst.Debug().Msgf("Instance (%s) reconciliation skipped - states current (%s) desired (%s)",
			resourceID, instance.CurrentState, instance.DesiredState)
		return request.Ack()
	}
	return nil
}

func (ir *InstanceReconciler) updateHostInstanceStatusAndCurrentState(
	ctx context.Context,
	oldInstance *computev1.InstanceResource,
	newInstance *computev1.InstanceResource,
) {
	newHost := newInstance.GetHost()
	zlogInst.Debug().Msgf("Updating Host %s resourceID %s onboarding status: %q",
		newHost.GetUuid(), newHost.GetResourceId(), newHost.GetOnboardingStatus())

	if !util.IsSameHostStatus(oldInstance.GetHost(), newHost) {
		if err := ir.invClient.SetHostOnboardingStatus(
			ctx, newHost.GetTenantId(), newHost.GetResourceId(),
			inv_status.New(newHost.GetOnboardingStatus(), newHost.GetOnboardingStatusIndicator())); err != nil {
			zlogInst.InfraSec().InfraErr(err).Msgf("Failed to update host status")
		}
	}

	zlogInst.Debug().Msgf("Updating Instance %s with state %s, provisioning status: %q",
		newInstance.GetResourceId(), newInstance.GetCurrentState(),
		newInstance.GetProvisioningStatus())

	if !util.IsSameInstanceStatusAndState(oldInstance, newInstance) || oldInstance.CurrentOs != newInstance.CurrentOs {
		if err := ir.invClient.UpdateInstance(
			ctx,
			newInstance.GetTenantId(),
			newInstance.GetResourceId(),
			newInstance.GetCurrentState(),
			inv_status.New(newInstance.GetProvisioningStatus(), newInstance.GetProvisioningStatusIndicator()),
			newInstance.GetCurrentOs(),
		); err != nil {
			zlogInst.InfraSec().InfraErr(err).Msgf("Failed to update instance status")
		}
	}
}

func (ir *InstanceReconciler) reconcileInstance(
	ctx context.Context,
	request rec_v2.Request[ReconcilerID],
	instance *computev1.InstanceResource,
) rec_v2.Directive[ReconcilerID] {
	zlogInst.Debug().Msgf("Reconciling Instance with ID %s, with Current state: %v, Desired state: %v",
		instance.GetResourceId(), instance.GetCurrentState(), instance.GetDesiredState())

	if instance.GetDesiredState() == computev1.InstanceState_INSTANCE_STATE_RUNNING {
		err := ir.tryProvisionInstance(ctx, instance)
		if directive := HandleProvisioningError(err, request); directive != nil {
			return directive
		}

		if err = ir.cleanupProvisioningResources(ctx, instance); err != nil {
			// do not retry, Tinkerbell resources will eventually be deleted when Instance is deleted.
			return request.Ack()
		}

		zlogInst.Debug().Msgf("Instance (%s) has been provisioned", instance.GetResourceId())
		return request.Ack()
	}

	if instance.GetDesiredState() == computev1.InstanceState_INSTANCE_STATE_DELETED {
		zlogInst.InfraSec().Info().Msgf("Deleting instance (set current status to Deleted)")

		if err := ir.cleanupProvisioningResources(ctx, instance); err != nil {
			if directive := HandleProvisioningError(err, request); directive != nil {
				return directive
			}
		}

		err := ir.invClient.UpdateInstanceCurrentState(
			ctx,
			instance.GetTenantId(),
			&computev1.InstanceResource{
				ResourceId:   instance.GetResourceId(),
				CurrentState: computev1.InstanceState_INSTANCE_STATE_DELETED,
			},
		)
		if directive := HandleInventoryError(err, request); directive != nil {
			return directive
		}
		zlogInst.Debug().Msgf("Instance (%s) has been deleted", instance.GetResourceId())
		return request.Ack()
	}
	if instance.GetDesiredState() == computev1.InstanceState_INSTANCE_STATE_UNTRUSTED {
		err := ir.invClient.UpdateInstanceCurrentState(
			ctx,
			instance.GetTenantId(),
			&computev1.InstanceResource{
				ResourceId:   instance.GetResourceId(),
				CurrentState: computev1.InstanceState_INSTANCE_STATE_UNTRUSTED,
			},
		)
		if directive := HandleInventoryError(err, request); directive != nil {
			return directive
		}
		zlogInst.Debug().Msgf("Instance (%s) currentState has been updated to untrusted", instance.GetResourceId())
		return request.Ack()
	}
	return request.Ack()
}

func convertInstanceToDeviceInfo(instance *computev1.InstanceResource,
) (onboarding_types.DeviceInfo, error) {
	host := instance.GetHost() // eager-loaded

	if instance.GetDesiredOs() == nil {
		// this should not happen but just in case
		return onboarding_types.DeviceInfo{}, inv_errors.Errorfc(codes.InvalidArgument,
			"Instance %s doesn't have any OS associated", instance.GetResourceId())
	}

	desiredOs := instance.GetDesiredOs()

	zlogInst.Debug().Msgf("Converting Instance %s to device info. OS resource: %s",
		instance.GetResourceId(), desiredOs)

	localHostIP := "127.0.0.1"
	var osLocationURL string
	// OS and Installer location returned to EN points to a local server that proxies requests to Provisioning Nginx
	if desiredOs.GetOsType() == osv1.OsType_OS_TYPE_MUTABLE {
		zlogInst.Debug().Msgf("Pulling %s image from %s", desiredOs.GetProfileName(), desiredOs.GetImageUrl())
		osLocationURL = desiredOs.GetImageUrl()
	} else {
		// TiberMicrovisor can be pulled drirectly from Release Server or CDN Server
		zlogInst.Debug().Msgf("Pulling %s image Pulling from CDN/RS Servers", desiredOs.GetProfileName())
		osLocationURL = fmt.Sprintf("http://%s/%s", localHostIP, desiredOs.GetImageUrl())
	}

	proxyURL := fmt.Sprintf(TinkStackURLTemplate, localHostIP)
	// Installer script or Cloud init file download
	installerScriptURL, err := dkam_util.GetInstallerLocation(instance.GetDesiredOs(), proxyURL)
	if err != nil {
		return onboarding_types.DeviceInfo{}, err
	}
	tinkerVersion := env.TinkerActionVersion

	deviceInfo := onboarding_types.DeviceInfo{
		GUID:               host.GetUuid(),
		HwSerialID:         host.GetSerialNumber(),
		HwMacID:            host.GetPxeMac(),
		HwIP:               host.GetBmcIp(),
		Hostname:           host.GetResourceId(), // we use resource ID as hostname to uniquely identify a host
		SecurityFeature:    instance.GetSecurityFeature(),
		OSImageURL:         osLocationURL,
		OsImageSHA256:      desiredOs.GetSha256(),
		InstallerScriptURL: installerScriptURL,
		TinkerVersion:      tinkerVersion,
		OsType:             desiredOs.GetOsType(),
		PlatformBundle:     desiredOs.GetPlatformBundle(),
	}

	zlogInst.Debug().Msgf("DeviceInfo generated from OS resource (%s): %+v",
		instance.GetDesiredOs().GetResourceId(), deviceInfo)

	return deviceInfo, nil
}

func (ir *InstanceReconciler) tryProvisionInstance(ctx context.Context, instance *computev1.InstanceResource) error {
	if instance.GetDesiredOs() == nil {
		zlogInst.Warn().Msgf("No desired OS specified for instance %s, skipping provisioning.",
			instance.GetResourceId())
		return nil
	}

	if instance.GetDesiredOs().GetOsProvider() != osv1.OsProviderKind_OS_PROVIDER_KIND_INFRA {
		zlogInst.Debug().Msgf("Skipping OS provisioning for %s due to OS provider kind: %s",
			instance.GetResourceId(), instance.GetDesiredOs().GetOsProvider().String())
		return nil
	}

	deviceInfo, err := convertInstanceToDeviceInfo(instance)
	if err != nil {
		zlogInst.InfraSec().Err(err).Msgf("Failed convertInstanceToDeviceInfo - Instance %s with Host UUID %s",
			instance.GetResourceId(), instance.GetHost().GetUuid())
		return err
	}

	//nolint:errcheck // this function currently not returning any error to handle
	oldInstance := proto.Clone(instance).(*computev1.InstanceResource)

	zlogInst.Debug().Msgf("Trying to provision Instance %s with OS %s",
		instance.GetResourceId(), instance.GetDesiredOs().GetName())

	defer func() {
		// if unrecoverable error, report error provisioning status
		if grpc_status.Convert(err).Code() == codes.Aborted {
			// report error
			util.PopulateInstanceProvisioningStatus(instance, om_status.ProvisioningStatusFailed)
		}
		// should be safe to not return an error
		// if the inventory client fails, this will be eventually fixed in the next reconciliation cycle
		zlogInst.InfraSec().Err(err).Msgf("Updating Host and Instance status - Instance %s with Host UUID %s",
			instance.GetResourceId(), instance.GetHost().GetUuid())
		ir.updateHostInstanceStatusAndCurrentState(ctx, oldInstance, instance)
	}()

	// Check status of Prod Workflow and initiate if it's not running.
	if err := onboarding.CheckStatusOrRunProdWorkflow(ctx, deviceInfo, instance); err != nil {
		zlogInst.InfraSec().Err(err).Msgf("Failed CheckStatusOrRunProdWorkflow - Instance %s with Host UUID %s",
			instance.GetResourceId(), instance.GetHost().GetUuid())
		return err
	}

	util.PopulateInstanceStatusAndCurrentState(instance, computev1.InstanceState_INSTANCE_STATE_RUNNING,
		om_status.ProvisioningStatusDone)

	return nil
}

func (ir *InstanceReconciler) cleanupProvisioningResources(
	ctx context.Context,
	instance *computev1.InstanceResource,
) error {
	zlogInst.Debug().Msgf("Cleaning up all provisioning resources for host %s", instance.GetHost().GetUuid())

	if err := onboarding.DeleteProdWorkflowResourcesIfExist(ctx, instance.GetHost().GetUuid()); err != nil {
		return err
	}

	return onboarding.DeleteTinkHardwareForHostIfExist(ctx, instance.GetHost().GetUuid())
}
