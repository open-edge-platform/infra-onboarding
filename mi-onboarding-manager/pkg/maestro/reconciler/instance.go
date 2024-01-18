/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/
package reconciler

import (
	"context"

	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/os/v1"
	inv_client "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/client"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/onboardingmgr"
	onboarding "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/onboarding"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/pkg/maestro"
	rec_v2 "github.com/onosproject/onos-lib-go/pkg/controller/v2"
)

type InstanceReconciler struct {
	invClient inv_client.InventoryClient
}
type OnboardingManager struct {
	pb.OnBoardingEBServer
}

func NewInstanceReconciler(c inv_client.InventoryClient) *InstanceReconciler {
	return &InstanceReconciler{
		invClient: c,
	}
}

func (ir *InstanceReconciler) Reconcile(ctx context.Context, request rec_v2.Request[ResourceID]) rec_v2.Directive[ResourceID] {
	resourceID := request.ID.String()
	zlog.MiSec().Info().Msgf("Reconciling instance : %s", resourceID)

	inst, err := getInstanceByID(ctx, ir.invClient, resourceID)
	if err != nil {
		zlog.Err(err).Msgf("Failed to get instance : %s", resourceID)
	}
	if directive := handleInventoryError(err, request); directive != nil {
		return directive
	}

	if inst.DesiredState == inst.CurrentState {
		zlog.MiSec().Info().Msgf("Instance %s reconciliation skipped", resourceID)
		return request.Ack()
	}

	return ir.reconcileInstance(ctx, request, inst)
}

func (ir *InstanceReconciler) reconcileInstance(
	ctx context.Context,
	request rec_v2.Request[ResourceID],
	inst *computev1.InstanceResource,
) rec_v2.Directive[ResourceID] {
	id := inst.GetResourceId()
	zlog.Debug().Msgf("Reconciling instance with ID: %s current state: %v desired state: %v",
		id, inst.GetCurrentState(), inst.GetDesiredState())

	onboardingMgr := &onboarding.OnboardingManager{}

	if inst.GetDesiredState() == computev1.InstanceState_INSTANCE_STATE_RUNNING {

		//Getting the host details for the id
		hostID := inst.GetHost().GetResourceId()
		host, err := GetHostDetailsByResourceID(ctx, ir.invClient, hostID)
		if err != nil {
			zlog.Err(err).Msgf("Failed to get host details for instance ID : %s", host.GetResourceId())
			return request.Ack()
		}
		zlog.MiSec().Info().Msgf("Host details associated with Instance id %v", host)
		osd, oserr := maestro.GetOsResourceById(ctx, ir.invClient, inst.Os.GetResourceId())
		if oserr != nil {
			zlog.Err(oserr).Msgf("Failed to get os details for instance ID : %s", inst.Os.GetResourceId())
			return request.Ack()
		}
		onboardingRequest, err := onboarding.ConvertInstanceForOnboarding([]*computev1.InstanceResource{inst}, []*osv1.OperatingSystemResource{osd}, host)
		if err != nil {
			zlog.Err(err).Msgf("Failed to convert instance for onboarding")
			return request.Ack()
		}

		zlog.MiSec().Info().Msgf("onboarding request: %v", onboardingRequest)

		if len(onboardingRequest) > 0 {
			response, oberr := onboardingMgr.StartOnboarding(ctx, onboardingRequest[0])
			if oberr != nil {
				zlog.Err(oberr).Msgf("Failed to start onboard for the instance ID : %s", id)
				return request.Ack()
			}
			if response.Status == "Success" {
				err := ir.updateInstance(ctx, id)
				if err != nil {
					zlog.Err(err).Msgf("Failed to update instance with ID : %s", id)
					return request.Ack()
				}
			} else {
				zlog.Err(err).Msgf("Failed to update instance for the ID : %s", inst.GetResourceId())
				return request.Ack()
			}
		} else {
			zlog.Err(err).Msgf("Failed to start onboarding for instance ID : %s", inst.GetResourceId())
			return request.Ack()
		}

	}

	if inst.GetDesiredState() == computev1.InstanceState_INSTANCE_STATE_DELETED {
		zlog.MiSec().Info().Msgf("Deleting instance ID %s (set current status to Deleted)", id)
		err := ir.deleteInstance(ctx, id)
		if err != nil {
			zlog.Err(err).Msgf("Failed to update instance with ID: %s", id)
		}
		if directive := handleInventoryError(err, request); directive != nil {
			return directive
		}
		zlog.Debug().Msgf("Instance with ID %v has been deleted", id)
		return request.Ack()
	}

	return request.Ack()
}

func (ir *InstanceReconciler) updateInstance(
	ctx context.Context,
	id string,
) error {
	instRes := computev1.InstanceResource{
		ResourceId:   id,
		CurrentState: computev1.InstanceState_INSTANCE_STATE_RUNNING,
	}

	err := updateInstanceCurrentState(ctx, ir.invClient, &instRes)
	return err
}

func (ir *InstanceReconciler) deleteInstance(
	ctx context.Context,
	id string,
) error {
	instRes := computev1.InstanceResource{
		ResourceId:   id,
		CurrentState: computev1.InstanceState_INSTANCE_STATE_DELETED,
	}

	err := updateInstanceCurrentState(ctx, ir.invClient, &instRes)
	return err
}

func updateInstanceCurrentState(ctx context.Context, c inv_client.InventoryClient, inst *computev1.InstanceResource) error {
	return maestro.UpdateInvResourceFields(ctx, c, inst, []string{
		"current_state",
	})
}

func getInstanceByID(ctx context.Context, c inv_client.InventoryClient, resourceID string) (*computev1.InstanceResource, error) {
	res, err := c.Get(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	inst := res.GetResource().GetInstance()
	if err := inst.ValidateAll(); err != nil {
		return nil, inv_errors.Wrap(err)
	}

	return inst, nil
}

func GetHostDetailsByResourceID(ctx context.Context, c inv_client.InventoryClient, hostID string) (*computev1.HostResource, error) {
	host, err := c.Get(ctx, hostID)
	if err != nil {
		log.Errorf("Failed to get host details for host ID %s: %v", hostID, err)
		return nil, err
	}

	inst := host.GetResource().GetHost()
	if err := inst.ValidateAll(); err != nil {
		return nil, inv_errors.Wrap(err)
	}
	return inst, nil
}
