// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

package reconcilers

import (
	"context"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/common"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/onboarding"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/onbworkflowclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/util"
	om_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/status"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	inv_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/status"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/tracing"
	"google.golang.org/grpc/codes"
	grpc_status "google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"os"
	"strings"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/api"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"

	rec_v2 "github.com/onosproject/onos-lib-go/pkg/controller/v2"
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
type OnboardingManager struct {
	pb.OnBoardingEBServer
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

	if instance.CurrentState == computev1.InstanceState_INSTANCE_STATE_ERROR {
		// current_state set to ERROR by previous reconciliation cycles
		// We don't have auto-recovery mechanisms. The previous reconciliation cycle should
		// set providerStatusDetail to provide feedback to user.
		// ATM I (Tomasz) believe that a user should delete via UI and re-configure host again, once the issue is fixed (e.g., wrong BIOS settings, etc.)
		zlogInst.Warn().Msgf("Current state of Instance %s is ERROR. Reconciliation won't happen until the Instance is re-created.", instance.GetResourceId())
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
	newInstance *computev1.InstanceResource) {

	newHost := newInstance.GetHost()
	zlogInst.Debug().Msgf("Updating Host %s status with %s, status details: %s, onboarding status: %q",
		newHost.GetUuid(), newHost.GetLegacyHostStatus(), newHost.GetProviderStatusDetail(), newHost.GetOnboardingStatus())

	if !util.IsSameHostStatus(oldInstance.GetHost(), newHost) {
		if err := ir.invClient.SetHostStatus(
			ctx,
			newHost.GetResourceId(),
			newHost.GetLegacyHostStatus(),
			newHost.GetProviderStatusDetail(),
			inv_status.New(newHost.GetOnboardingStatus(), newHost.GetOnboardingStatusIndicator())); err != nil {
			zlogInst.MiSec().MiErr(err).Msgf("Failed to update host %s status", newHost.GetResourceId())
		}
	}

	zlogInst.Debug().Msgf("Updating Instance %s with state %s and status %s, provisioning status: %q",
		newInstance.GetResourceId(), newInstance.GetCurrentState(), newInstance.GetStatus(), newInstance.GetProvisioningStatus())

	if !util.IsSameInstanceStatusAndState(oldInstance, newInstance) {
		if err := ir.invClient.SetInstanceStatusAndCurrentState(
			ctx,
			newInstance.GetResourceId(),
			newInstance.GetCurrentState(),
			newInstance.GetStatus(),
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
		instance.GetResourceId(), instance.GetCurrentState(), instance.GetDesiredState(), host.GetLegacyHostStatus())

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
	bmcNic *computev1.HostnicResource, artifactInfo utils.ArtifactData) utils.DeviceInfo {
	host := instance.GetHost() // eager-loaded

	deviceInfo := utils.DeviceInfo{
		GUID:            host.GetUuid(),
		HwSerialID:      host.GetSerialNumber(),
		HwMacID:         bmcNic.GetMacAddr(), // TODO: from hostnics, maybe we can add "bmc_mac" to Host object in Inventory to avoid querying hostnics?
		HwIP:            host.GetBmcIp(),
		SecurityFeature: uint32(instance.GetSecurityFeature()),
		DiskType:        os.Getenv("DISK_PARTITION"),
		LoadBalancerIP:  os.Getenv("IMG_URL"),
		Gateway:         onboarding.GenerateGatewayFromBaseIP(host.GetBmcIp()),
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
	if len(invURL) > 1 {
		overlayURL = invURL[1]
	}

	sutIP := instance.GetHost().GetBmcIp()
	osURL = utils.ReplaceHostIP(osURL, sutIP)
	overlayURL = utils.ReplaceHostIP(overlayURL, sutIP)

	return utils.ArtifactData{
		BkcURL:        osURL,
		BkcBasePkgURL: overlayURL,
	}, nil
}

func (ir *InstanceReconciler) tryProvisionInstance(ctx context.Context, instance *computev1.InstanceResource) error {
	bmcNic, err := ir.invClient.GetHostBmcNic(ctx, instance.GetHost())
	if err != nil {
		return err
	}

	artifactInfo, err := convertInstanceToArtifactInfo(instance)
	if err != nil {
		return err
	}

	deviceInfo := convertInstanceToDeviceInfo(instance, bmcNic, artifactInfo)
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
	if err = onbworkflowclient.CheckStatusOrRunDIWorkflow(ctx, deviceInfo, instance); err != nil {
		return err
	}

	// 2. Check status of FDO, we won't progress to next steps until TO2 is completed
	if err = onbworkflowclient.CheckTO2StatusOrRunFDOActions(ctx, deviceInfo, instance); err != nil {
		return err
	}

	// 3. Check status of Prod Workflow and initiate it if not running
	if err = onbworkflowclient.CheckStatusOrRunProdWorkflow(ctx, deviceInfo, instance); err != nil {
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

	if err := onbworkflowclient.DeleteProdWorkflowResourcesIfExist(ctx, instance.GetHost().GetUuid()); err != nil {
		return err
	}

	if *common.FlagEnableDeviceInitialization {
		if err := onbworkflowclient.DeleteDIWorkflowResourcesIfExist(ctx, instance.GetHost().GetUuid()); err != nil {
			return err
		}
	}

	if err := onbworkflowclient.DeleteTinkHardwareForHostIfExist(ctx, instance.GetHost().GetUuid()); err != nil {
		return err
	}

	return nil
}
