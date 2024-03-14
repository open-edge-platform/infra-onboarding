// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

package reconcilers

import (
	"context"
	"os"
	"strings"

	rec_v2 "github.com/onosproject/onos-lib-go/pkg/controller/v2"
	"google.golang.org/grpc/codes"
	grpc_status "google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/common"
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

func convertInstanceToDeviceInfo(instance *computev1.InstanceResource, artifactInfo utils.ArtifactData) utils.DeviceInfo {
	host := instance.GetHost() // eager-loaded

	deviceInfo := utils.DeviceInfo{
		GUID:            host.GetUuid(),
		HwSerialID:      host.GetSerialNumber(),
		HwMacID:         host.GetPxeMac(),
		HwIP:            host.GetBmcIp(),
		SecurityFeature: uint32(instance.GetSecurityFeature()),
		DiskType:        os.Getenv("DISK_PARTITION"),
		LoadBalancerIP:  os.Getenv("IMG_URL"),
		Gateway:         utils.GenerateGatewayFromBaseIP(host.GetBmcIp()),
		ProvisionerIP:   os.Getenv("PD_IP"),
		ImType:          os.Getenv("IMAGE_TYPE"),
		RootfspartNo:    os.Getenv("OVERLAY_URL"),
		FdoMfgDNS:       os.Getenv("FDO_MFG_URL"),
		FdoOwnerDNS:     os.Getenv("FDO_OWNER_URL"),
		FdoMfgPort:      os.Getenv("FDO_MFG_PORT"),
		FdoOwnerPort:    os.Getenv("FDO_OWNER_PORT"),
		FdoRvPort:       os.Getenv("FDO_RV_PORT"),
	}

	deviceInfo.Rootfspart = utils.CalculateRootFS(deviceInfo.ImType, deviceInfo.DiskType)
	deviceInfo.TinkerVersion = artifactInfo.TinkerVersion

	switch deviceInfo.ImType {
	case utils.ProdBkc:
		deviceInfo.ClientImgName = "jammy-server-cloudimg-amd64.raw.gz"
		deviceInfo.ImType = utils.ImgTypeBkc
		deviceInfo.LoadBalancerIP = artifactInfo.BkcURL
		deviceInfo.RootfspartNo = artifactInfo.BkcBasePkgURL
	case utils.ProdFocal:
		deviceInfo.ClientImgName = "focal-server-cloudimg-amd64.raw.gz"
		deviceInfo.ImType = utils.ImgTypeFocal
	case utils.ProdFocalMs:
		deviceInfo.ImType = utils.ImgTypeFocalMs
	default:
		deviceInfo.ClientImgName = "jammy-server-cloudimg-amd64.raw.gz"
		deviceInfo.ImType = utils.ImgTypeJammy
	}

	return deviceInfo
}

func convertInstanceToArtifactInfo(instance *computev1.InstanceResource) (utils.ArtifactData, error) {
	const checkInvURLLength = 2
	if instance.GetOs() == nil {
		// this should not happen but just in case
		return utils.ArtifactData{}, inv_errors.Errorfc(codes.InvalidArgument,
			"Instance %s doesn't have any OS associated", instance.GetResourceId())
	}
	repoURL := instance.GetOs().GetRepoUrl()
	invURL := strings.Split(repoURL, ";")

	if len(invURL) == 0 {
		return utils.ArtifactData{}, inv_errors.Errorfc(codes.InvalidArgument,
			"Invalid format of OS repo url: %s", repoURL)
	}

	osURL := invURL[0]
	if !utils.IsValidOSURLFormat(osURL) {
		return utils.ArtifactData{}, inv_errors.Errorfc(codes.InvalidArgument,
			"Invalid format of OS url: %s", osURL)
	}

	var overlayURL string
	var tinkerVersion string
	if len(invURL) > 1 {
		overlayURL = invURL[1]
	}

	if len(invURL) > checkInvURLLength {
		tinkerVersion = invURL[2]
	}

	sutIP := instance.GetHost().GetBmcIp()
	osURL = utils.ReplaceHostIP(osURL, sutIP)
	overlayURL = utils.ReplaceHostIP(overlayURL, sutIP)

	return utils.ArtifactData{
		BkcURL:        osURL,
		BkcBasePkgURL: overlayURL,
		TinkerVersion: tinkerVersion,
	}, nil
}

func (ir *InstanceReconciler) tryProvisionInstance(ctx context.Context, instance *computev1.InstanceResource) error {
	artifactInfo, err := convertInstanceToArtifactInfo(instance)
	if err != nil {
		return err
	}

	deviceInfo := convertInstanceToDeviceInfo(instance, artifactInfo)
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

	// 1. Check status of DI workflow and initiate it if not running
	if diErr := onbworkflowclient.CheckStatusOrRunDIWorkflow(ctx, deviceInfo, instance); diErr != nil {
		return diErr
	}

	// 2. Check status of FDO, we won't progress to next steps until TO2 is completed
	if fdoErr := onbworkflowclient.CheckTO2StatusOrRunFDOActions(ctx, deviceInfo, instance); fdoErr != nil {
		return fdoErr
	}

	// 3. Check status of Prod Workflow and initiate it if not running
	if prodErr := onbworkflowclient.CheckStatusOrRunProdWorkflow(ctx, deviceInfo, instance); prodErr != nil {
		return prodErr
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

	if err := onbworkflowclient.DeleteProdWorkflowResourcesIfExist(ctx, instance.GetHost().GetUuid()); err != nil {
		return err
	}

	if *common.FlagEnableDeviceInitialization {
		if err := onbworkflowclient.DeleteDIWorkflowResourcesIfExist(ctx, instance.GetHost().GetUuid()); err != nil {
			return err
		}
	}

	return onbworkflowclient.DeleteTinkHardwareForHostIfExist(ctx, instance.GetHost().GetUuid())
}
