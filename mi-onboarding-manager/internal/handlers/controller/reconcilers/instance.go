// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

package reconcilers

import (
	"context"

	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/os/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/invclient"

	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/onboardingmgr"
	onboarding "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/onboarding"
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
	invClient *invclient.OnboardingInventoryClient
}
type OnboardingManager struct {
	pb.OnBoardingEBServer
}

func NewInstanceReconciler(c *invclient.OnboardingInventoryClient) *InstanceReconciler {
	return &InstanceReconciler{
		invClient: c,
	}
}

func (ir *InstanceReconciler) Reconcile(ctx context.Context, request rec_v2.Request[ResourceID]) rec_v2.Directive[ResourceID] {
	resourceID := request.ID.String()
	zlogInst.Info().Msgf("Reconciling Instance (%s)", resourceID)

	instance, err := ir.invClient.GetInstanceResourceByResourceID(ctx, resourceID)
	if directive := HandleInventoryError(err, request); directive != nil {
		return directive
	}

	if instance.DesiredState == instance.CurrentState {
		zlogInst.Debug().Msgf("Instance (%s) reconciliation skipped", resourceID)
		return request.Ack()
	}

	return ir.reconcileInstance(ctx, request, instance)
}

func (ir *InstanceReconciler) reconcileInstance(
	ctx context.Context,
	request rec_v2.Request[ResourceID],
	instance *computev1.InstanceResource,
) rec_v2.Directive[ResourceID] {
	instanceID := instance.GetResourceId()
	zlogInst.Debug().Msgf("Reconciling Instance with ID %s, with Current state: %v, Desired state: %v.",
		instance.GetResourceId(), instance.GetCurrentState(), instance.GetDesiredState())

	onboardingMgr := &onboarding.OnboardingManager{}

	// TODO: we should also check if there is no onboarding in progress
	if instance.GetDesiredState() == computev1.InstanceState_INSTANCE_STATE_RUNNING {
		// no need to query Host from Inventory, eager loaded from Instance
		host := instance.GetHost()
		zlogInst.MiSec().Debug().Msgf("Host details associated with Instance id %v Resource %v", host, host.ResourceId)
		// no need to query OS from Inventory, eager loaded from Instance
		os := instance.GetOs()

		host, err := ir.invClient.GetHostResourceByResourceID(ctx, host.ResourceId)
		if err != nil {
			zlogInst.MiSec().MiErr(err).Msgf("Failed to Get Host Resource by ID")
			return request.Ack()
		}
		onboardingRequest, err := onboarding.ConvertInstanceForOnboarding(
			[]*computev1.InstanceResource{instance}, []*osv1.OperatingSystemResource{os}, host)
		if err != nil {
			zlogInst.MiSec().MiErr(err).Msgf("Failed to convert instance for onboarding")
			return request.Ack()
		}

		zlogInst.MiSec().Debug().Msgf("onboarding request: %v", onboardingRequest)

		if len(onboardingRequest) > 0 {
			response, oberr := onboardingMgr.StartOnboarding(ctx, onboardingRequest[0])
			if oberr != nil {
				zlogInst.MiSec().MiErr(oberr).Msgf("Failed to start onboard for the instance ID : %s", instanceID)
				return request.Ack()
			}

			if response.Status == "Success" {
				updateErr := ir.updateInstance(ctx, instanceID)
				if updateErr != nil {
					zlogInst.MiSec().MiErr(updateErr).Msgf("Failed to update instance with ID : %s", instanceID)
					return request.Ack()
				}
			} else {
				zlogInst.MiSec().Error().Msgf("Failed to start onboarding, response status=%s", response.Status)
				return request.Ack()
			}
		} else {
			zlogInst.MiSec().Error().Msg("Failed to start onboarding, empty onboarding request list")
			return request.Ack()
		}
	}

	if instance.GetDesiredState() == computev1.InstanceState_INSTANCE_STATE_DELETED {
		zlogInst.MiSec().Info().Msgf("Deleting instance ID %s (set current status to Deleted)", instanceID)
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

func (ir *InstanceReconciler) updateInstance(
	ctx context.Context,
	id string,
) error {
	inst := &computev1.InstanceResource{
		ResourceId:   id,
		CurrentState: computev1.InstanceState_INSTANCE_STATE_RUNNING,
	}

	err := ir.invClient.UpdateInstanceCurrentState(ctx, inst)
	return err
}
