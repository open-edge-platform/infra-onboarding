// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

package reconcilers

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	grpc_status "google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	rec_v2 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-app.lib-go/pkg/controller/v2"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/common"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/env"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/onbworkflowclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/util"
	om_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/status"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
	inv_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/status"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/tracing"
)

const (
	instanceReconcilerLoggerName = "InstanceReconciler"
	checkInvURLLength            = 2
	ClientImgName                = "jammy-server-cloudimg-amd64.raw.gz"
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
	request rec_v2.Request[ResourceID],
) rec_v2.Directive[ResourceID] {
	if ir.enableTracing {
		ctx = tracing.StartTrace(ctx, "MIOnboardingManager", "InstanceReconciler")
		defer tracing.StopTrace(ctx)
	}
	resourceID := request.ID.String()
	zlogInst.Info().Msgf("Reconciling Instance (%s)", resourceID)

	instance, err := ir.invClient.GetInstanceResourceByResourceID(ctx, resourceID)
	if directive := HandleInventoryError(err, request); directive != nil {
		return directive
	}

	// the only allowed path from the ERROR state is DELETED
	if instance.CurrentState == computev1.InstanceState_INSTANCE_STATE_ERROR &&
		instance.DesiredState != computev1.InstanceState_INSTANCE_STATE_DELETED {
		// current_state set to ERROR by previous reconciliation cycles
		// We don't have auto-recovery mechanisms. The previous reconciliation cycle should
		// set providerStatusDetail to provide feedback to user.
		// ATM I (Tomasz) believe that a user should delete via UI and re-configure host again,
		// once the issue is fixed (e.g., wrong BIOS settings, etc.)
		zlogInst.Warn().Msgf(
			"Current state of Instance %s is ERROR. Reconciliation won't happen until the Instance is re-created.",
			instance.GetResourceId())
		return request.Ack()
	}

	// Forbid Instance provisioning with defined Provider. Such Instance should be reconciled within Provider-specific RM.
	if instance.GetProvider() != nil {
		zlogInst.Info().Msgf("Instance %s should be reconciled within other vendor-specific RM (%s)",
			instance.GetResourceId(), instance.GetProvider().GetName())
		return request.Ack()
	}

	if instance.DesiredState == instance.CurrentState {
		zlogInst.Debug().Msgf("Instance (%s) reconciliation skipped", resourceID)
		return request.Ack()
	}

	return ir.reconcileInstance(ctx, request, instance)
}

func (ir *InstanceReconciler) updateHostInstanceStatusAndCurrentState(
	ctx context.Context,
	oldInstance *computev1.InstanceResource,
	newInstance *computev1.InstanceResource,
) {
	newHost := newInstance.GetHost()
	//nolint:staticcheck // this field will be deprecated soon
	zlogInst.Debug().Msgf("Updating Host %s status with %s, status details: %s, onboarding status: %q", newHost.GetUuid(),
		newHost.GetLegacyHostStatus(), newHost.GetProviderStatusDetail(), newHost.GetOnboardingStatus())

	if !util.IsSameHostStatus(oldInstance.GetHost(), newHost) {
		if err := ir.invClient.SetHostStatus(
			ctx, newHost.GetResourceId(),
			//nolint:staticcheck // this field will be deprecated soon
			newHost.GetLegacyHostStatus(), newHost.GetProviderStatusDetail(),
			inv_status.New(newHost.GetOnboardingStatus(), newHost.GetOnboardingStatusIndicator())); err != nil {
			zlogInst.MiSec().MiErr(err).Msgf("Failed to update host %s status", newHost.GetResourceId())
		}
	}

	zlogInst.Debug().Msgf("Updating Instance %s with state %s and status %s, provisioning status: %q",
		newInstance.GetResourceId(), newInstance.GetCurrentState(),
		newInstance.GetStatus(), //nolint:staticcheck // this field will be deprecated soon
		newInstance.GetProvisioningStatus())

	if !util.IsSameInstanceStatusAndState(oldInstance, newInstance) {
		if err := ir.invClient.SetInstanceStatusAndCurrentState(
			ctx,
			newInstance.GetResourceId(),
			newInstance.GetCurrentState(),
			newInstance.GetStatus(), //nolint:staticcheck // this field will be deprecated soon
			inv_status.New(newInstance.GetProvisioningStatus(), newInstance.GetProvisioningStatusIndicator()),
		); err != nil {
			zlogInst.MiSec().MiErr(err).Msgf("Failed to update instance %s status", newInstance.GetResourceId())
		}
	}
}

func (ir *InstanceReconciler) reconcileInstance(
	ctx context.Context,
	request rec_v2.Request[ResourceID],
	instance *computev1.InstanceResource,
) rec_v2.Directive[ResourceID] {
	instanceID := instance.GetResourceId()
	host := instance.GetHost()

	zlogInst.Info().Msgf("Reconciling Instance with ID %s, with Current state: %v, Desired state: %v, HostState: %s",
		instance.GetResourceId(), instance.GetCurrentState(), instance.GetDesiredState(),
		host.GetLegacyHostStatus()) //nolint:staticcheck // this field will be deprecated soon

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
		zlogInst.MiSec().Info().Msgf("Deleting instance ID %s (set current status to Deleted)", instanceID)

		if err := ir.cleanupProvisioningResources(ctx, instance); err != nil {
			if directive := HandleProvisioningError(err, request); directive != nil {
				return directive
			}
		}

		err := ir.invClient.UpdateInstanceCurrentState(
			ctx,
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

	return request.Ack()
}

func convertInstanceToDeviceInfo(instance *computev1.InstanceResource,
	provider invclient.ProviderConfig,
) (utils.DeviceInfo, error) {
	host := instance.GetHost() // eager-loaded

	if instance.GetOs() == nil {
		// this should not happen but just in case
		return utils.DeviceInfo{}, inv_errors.Errorfc(codes.InvalidArgument,
			"Instance %s doesn't have any OS associated", instance.GetResourceId())
	}

	repoURL := instance.GetOs().GetRepoUrl()
	repoURLInfo := strings.Split(repoURL, ";")

	if len(repoURLInfo) == 0 {
		return utils.DeviceInfo{}, inv_errors.Errorfc(codes.InvalidArgument,
			"Invalid format of OS repo url: %s", repoURL)
	}

	osLocationURL := repoURLInfo[0]
	if !utils.IsValidOSURLFormat(osLocationURL) {
		return utils.DeviceInfo{}, inv_errors.Errorfc(codes.InvalidArgument,
			"Invalid format of OS url: %s", osLocationURL)
	}

	var (
		installerScriptURL string
		tinkerVersion      string
	)

	if len(repoURLInfo) > 1 {
		installerScriptURL = repoURLInfo[1]
	}

	if len(repoURLInfo) > checkInvURLLength {
		tinkerVersion = repoURLInfo[2]
	}

	sutIP := instance.GetHost().GetBmcIp()
	osLocationURL = utils.ReplaceHostIP(osLocationURL, sutIP)
	installerScriptURL = utils.ReplaceHostIP(installerScriptURL, sutIP)

	deviceInfo := utils.DeviceInfo{
		GUID:               host.GetUuid(),
		HwSerialID:         host.GetSerialNumber(),
		HwMacID:            host.GetPxeMac(),
		HwIP:               host.GetBmcIp(),
		Hostname:           host.GetResourceId(), // we use resource ID as hostname to uniquely identify a host
		SecurityFeature:    uint32(instance.GetSecurityFeature()),
		ImgType:            env.ImgType,
		OSImageURL:         env.ImgURL,
		DiskType:           env.DiskType,
		Rootfspart:         utils.CalculateRootFS(env.ImgType, env.DiskType),
		InstallerScriptURL: env.InstallerScriptURL,
		TinkerVersion:      tinkerVersion,
		ClientImgName:      ClientImgName,
		CustomerID:         provider.CustomerID,
	}

	if env.ImgType == utils.ImgTypeBkc {
		deviceInfo.OSImageURL = osLocationURL
		deviceInfo.InstallerScriptURL = installerScriptURL
	}

	return deviceInfo, nil
}

func (ir *InstanceReconciler) tryProvisionInstance(ctx context.Context, instance *computev1.InstanceResource) error {
	// TODO : Passing default provider name while trying to provision, need to change according to provider name and compare.
	providerConfig, err := ir.invClient.GetProviderConfig(ctx, utils.DefaultProviderName)
	if err != nil {
		zlogInst.Err(err).Msgf("Failed to get provider configuration")
		return err
	}

	deviceInfo, err := convertInstanceToDeviceInfo(instance, *providerConfig)
	if err != nil {
		return err
	}

	//nolint:errcheck // this function currently not returning any error to handle
	oldInstance := proto.Clone(instance).(*computev1.InstanceResource)

	zlogInst.Debug().Msgf("Trying to provision Instance %s with OS %s",
		instance.GetResourceId(), instance.GetOs().GetName())

	defer func() {
		// if unrecoverable error, set current_state to ERROR
		if grpc_status.Convert(err).Code() == codes.Aborted {
			instance.CurrentState = computev1.InstanceState_INSTANCE_STATE_ERROR
		}
		// should be safe to not return an error
		// if the inventory client fails, this will be eventually fixed in the next reconciliation cycle
		ir.updateHostInstanceStatusAndCurrentState(ctx, oldInstance, instance)
	}()

	// 1. Check status of DI workflow and initiate if it's not running
	if err := onbworkflowclient.CheckStatusOrRunDIWorkflow(ctx, deviceInfo, instance); err != nil {
		return err
	}

	// 2. Run FDO actions
	if err := onbworkflowclient.RunFDOActions(ctx, &deviceInfo); err != nil {
		return err
	}

	// 3. Check status of Reboot workflow and initiate if it's not running
	if err := onbworkflowclient.CheckStatusOrRunRebootWorkflow(ctx, deviceInfo, instance); err != nil {
		return err
	}

	// 4. Check status of Prod Workflow and initiate if it's not running.
	//    NOTE that Prod workflow will only start if TO2 process is completed.
	if err := onbworkflowclient.CheckStatusOrRunProdWorkflow(ctx, deviceInfo, instance); err != nil {
		return err
	}

	util.PopulateInstanceStatusAndCurrentState(instance,
		computev1.InstanceState_INSTANCE_STATE_RUNNING,
		computev1.InstanceStatus_INSTANCE_STATUS_PROVISIONED,
		om_status.ProvisioningStatusDone)

	return nil
}

func (ir *InstanceReconciler) cleanupProvisioningResources(
	ctx context.Context,
	instance *computev1.InstanceResource,
) error {
	zlogInst.Info().Msgf("Cleaning up all provisioning resources for host %s", instance.GetHost().GetUuid())

	if err := onbworkflowclient.DeleteProdWorkflowResourcesIfExist(
		ctx, instance.GetHost().GetUuid(), env.ImgType); err != nil {
		return err
	}

	if *common.FlagEnableDeviceInitialization {
		if err := onbworkflowclient.DeleteRebootWorkflowResourcesIfExist(ctx, instance.GetHost().GetUuid()); err != nil {
			return err
		}
		if err := onbworkflowclient.DeleteDIWorkflowResourcesIfExist(ctx, instance.GetHost().GetUuid()); err != nil {
			return err
		}
	}

	return onbworkflowclient.DeleteTinkHardwareForHostIfExist(ctx, instance.GetHost().GetUuid())
}
