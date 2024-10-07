// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

package reconcilers

import (
	"context"

	"google.golang.org/grpc/codes"
	grpc_status "google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	rec_v2 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-app.lib-go/pkg/controller/v2"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/common"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/env"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	onboarding "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/onboarding"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/onbworkflowclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/util"
	om_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/status"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/compute/v1"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/os/v1"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/logging"
	inv_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/status"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/tracing"
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
	request rec_v2.Request[ReconcilerID],
) rec_v2.Directive[ReconcilerID] {
	if ir.enableTracing {
		ctx = tracing.StartTrace(ctx, "MIOnboardingManager", "InstanceReconciler")
		defer tracing.StopTrace(ctx)
	}
	resourceID := request.ID.GetResourceID()
	tenantID := request.ID.GetTenantID()
	zlogInst.Info().Msgf("Reconciling Instance resourceID (%s) and tenantID (%s)", resourceID, tenantID)

	instance, err := ir.invClient.GetInstanceResourceByResourceID(ctx, tenantID, resourceID)
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
	zlogInst.Debug().Msgf("Updating Host %s onboarding status: %q", newHost.GetUuid(), newHost.GetOnboardingStatus())

	if !util.IsSameHostStatus(oldInstance.GetHost(), newHost) {
		if err := ir.invClient.SetHostOnboardingStatus(
			ctx, newHost.GetTenantId(), newHost.GetResourceId(),
			inv_status.New(newHost.GetOnboardingStatus(), newHost.GetOnboardingStatusIndicator())); err != nil {
			zlogInst.MiSec().MiErr(err).Msgf("Failed to update host %s status", newHost.GetResourceId())
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
			zlogInst.MiSec().MiErr(err).Msgf("Failed to update instance %s status", newInstance.GetResourceId())
		}
	}
}

func (ir *InstanceReconciler) reconcileInstance(
	ctx context.Context,
	request rec_v2.Request[ReconcilerID],
	instance *computev1.InstanceResource,
) rec_v2.Directive[ReconcilerID] {
	instanceID := instance.GetResourceId()

	zlogInst.Info().Msgf("Reconciling Instance with ID %s, with Current state: %v, Desired state: %v",
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
		zlogInst.MiSec().Info().Msgf("Deleting instance ID %s (set current status to Deleted)", instanceID)

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

//nolint:funlen // May effect the functionality, need to simplify this in future
func convertInstanceToDeviceInfo(instance *computev1.InstanceResource,
	provider invclient.ProviderConfig,
) (utils.DeviceInfo, error) {
	host := instance.GetHost() // eager-loaded

	if instance.GetDesiredOs() == nil {
		// this should not happen but just in case
		return utils.DeviceInfo{}, inv_errors.Errorfc(codes.InvalidArgument,
			"Instance %s doesn't have any OS associated", instance.GetResourceId())
	}

	repoURL := instance.GetDesiredOs().GetImageUrl()
	imageSha256 := instance.GetDesiredOs().GetSha256()
	profileName := instance.GetDesiredOs().GetProfileName()
	installedPackages := instance.GetDesiredOs().GetInstalledPackages()
	kernalCommand := instance.GetDesiredOs().GetKernelCommand()
	platform := instance.GetDesiredOs().GetArchitecture()
	osType := instance.GetDesiredOs().GetOsType()
	zlogInst.Info().Msgf("----------------------From DeviceInfo -------------------\n")
	zlogInst.Info().Msgf("repoURL is %s\n", repoURL)
	zlogInst.Info().Msgf("sha256 is %s\n", imageSha256)
	zlogInst.Info().Msgf("profileName is %s\n", profileName)
	zlogInst.Info().Msgf("installedPackages is %s\n", installedPackages)
	zlogInst.Info().Msgf("kernalCommand is %s\n", kernalCommand)
	zlogInst.Info().Msgf("platform is %s\n", platform)
	zlogInst.Info().Msgf("os type is %s\n", osType.String())

	response, err := onboarding.GetOSResourceFromDkamService(context.Background(), repoURL, imageSha256,
		profileName, installedPackages, platform, kernalCommand, osType.String())
	if err != nil {
		invError := inv_errors.Errorfc(grpc_status.Code(err), "Failed to trigger DKAM for OS instance. Error: %v", err)
		zlogInst.Err(invError).Msg("Error triggering DKAM for OS instance")
		return utils.DeviceInfo{}, invError
	}

	osLocationURL := response.GetOsUrl()
	installerScriptURL := response.GetOverlayscriptUrl()
	tinkerVersion := env.TinkerActionVersion

	sutIP := instance.GetHost().GetBmcIp()
	osLocationURL = utils.ReplaceHostIP(osLocationURL, sutIP)
	installerScriptURL = utils.ReplaceHostIP(installerScriptURL, sutIP)

	zlogInst.Info().Msgf("----------------------From DKAM start-------------------\n")
	zlogInst.Info().Msgf("osLocationURL is %s\n", osLocationURL)
	zlogInst.Info().Msgf("installerScriptURL is %s\n", installerScriptURL)
	zlogInst.Info().Msgf("tinkerVersion is %s\n", tinkerVersion)

	zlogInst.Info().Msgf("sutIP is %s\n", sutIP)
	zlogInst.Info().Msgf("utils.ReplaceHostIP: osLocationURL is %s\n", osLocationURL)
	zlogInst.Info().Msgf("installerScriptURL is %s\n", osLocationURL)
	zlogInst.Info().Msgf("imageSha256 is %s\n", imageSha256)

	deviceInfo := utils.DeviceInfo{
		GUID:               host.GetUuid(),
		HwSerialID:         host.GetSerialNumber(),
		HwMacID:            host.GetPxeMac(),
		HwIP:               host.GetBmcIp(),
		Hostname:           host.GetResourceId(),                  // we use resource ID as hostname to uniquely identify a host
		SecurityFeature:    uint32(instance.GetSecurityFeature()), // #nosec G115
		ImgType:            env.ImgType,
		OSImageURL:         env.ImgURL,
		OsImageSHA256:      imageSha256,
		DiskType:           env.DiskType,
		Rootfspart:         utils.CalculateRootFS(env.ImgType, env.DiskType),
		InstallerScriptURL: env.InstallerScriptURL,
		TinkerVersion:      tinkerVersion,
		ClientImgName:      ClientImgName,
		CustomerID:         provider.CustomerID,
		ENProductKeyIDs:    provider.ENProductKeyIDs,
		OsType:             osType.String(),
	}

	if osType == osv1.OsType_OS_TYPE_IMMUTABLE {
		deviceInfo.ImgType = utils.ImgTypeTiberOs
		// TODO: Fix the correct env image type based on OS type in charts
		env.ImgType = utils.ImgTypeTiberOs
	} else {
		deviceInfo.ImgType = utils.ImgTypeBkc
		// TODO: Fix the correct env image type based on OS type in charts
		env.ImgType = utils.ImgTypeBkc
	}

	// Adding additional checks.
	if osLocationURL == "" || installerScriptURL == "" || tinkerVersion == "" {
		// Create an error from the gRPC status code
		err := inv_errors.Errorfr(inv_errors.Reason_OPERATION_IN_PROGRESS, "Installation artifacts are not yet ready")
		return utils.DeviceInfo{}, err
	}

	deviceInfo.OSImageURL = osLocationURL
	deviceInfo.InstallerScriptURL = installerScriptURL

	zlogInst.Info().Msgf("----------------------At the end prints-------------------\n")
	zlogInst.Info().Msgf("OSImageURL is %s\n", deviceInfo.OSImageURL)
	zlogInst.Info().Msgf("InstallerScriptURL is %s\n", deviceInfo.InstallerScriptURL)
	zlogInst.Info().Msgf("ImgType is %s\n", deviceInfo.ImgType)
	zlogInst.Info().Msgf("DiskType is %s\n", deviceInfo.DiskType)
	zlogInst.Info().Msgf("OsType is %s\n", deviceInfo.OsType)
	zlogInst.Info().Msgf("SecurityFeature is %d\n", deviceInfo.SecurityFeature)
	zlogInst.Info().Msgf("SecurityFeature is %s\n", instance.GetSecurityFeature().String())
	zlogInst.Info().Msgf("ClientImgName is %s\n", deviceInfo.ClientImgName)
	zlogInst.Info().Msgf("CustomerID is %s\n", deviceInfo.CustomerID)
	zlogInst.Info().Msgf("ENProductKeyIDs is %s\n", deviceInfo.ENProductKeyIDs)
	zlogInst.Info().Msgf("OsImageSHA256 is %s\n", deviceInfo.OsImageSHA256)

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
		instance.GetResourceId(), instance.GetDesiredOs().GetName())

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
	if err := onbworkflowclient.RunFDOActions(ctx, instance.GetTenantId(), &deviceInfo); err != nil {
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

	util.PopulateInstanceStatusAndCurrentState(instance, computev1.InstanceState_INSTANCE_STATE_RUNNING,
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
