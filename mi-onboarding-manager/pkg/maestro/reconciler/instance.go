/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/
package reconciler

import (
	"context"

	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
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
	log.Infof("Reconciling instance %s", resourceID)

	inst, err := getInstanceByID(ctx, ir.invClient, resourceID)
	if err != nil {
		log.Errorf("Failed to get instance %s %v", resourceID, err)
	}
	if directive := handleInventoryError(err, request); directive != nil {
		return directive
	}

	if inst.DesiredState == inst.CurrentState {
		log.Infof("Instance %s reconciliation skipped", resourceID)
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
	log.Infof("Reconciling instance with ID: %s current state: %v desired state: %v",
		id, inst.GetCurrentState(), inst.GetDesiredState())

	onboardingMgr := &onboarding.OnboardingManager{}

	//TODO : Instance state value has to be confirmed.
	if inst.GetCurrentState() == computev1.InstanceState_INSTANCE_STATE_INSTALLED {

		// Trigger DKAM with the necessary details,
		response, err := onboarding.GetOSResourceFromDkam(ctx, inst)
		if err != nil {
			log.Errorf("Failed to trigger DKAM for instance ID %s: %v", id, err)
			return request.Ack()
		}

		//Getting the host details for the id
		hostID := inst.GetHost().GetResourceId()
		host, err := GetHostDetailsByResourceID(ctx, ir.invClient, hostID)
		if err != nil {
			log.Errorf("Failed to get host details for instance ID %s: %v", host.GetResourceId(), err)
			return request.Ack()
		}
		log.Infof("Host details associated with Instance id %v", host)

		onboardingRequest, err := onboarding.ConvertInstanceForOnboarding([]*computev1.InstanceResource{inst}, host, response)
		if err != nil {
			log.Errorf("Failed to convert instance for onboarding: %v", err)
			return request.Ack()
		}

		log.Infof("onboarding request: %v", onboardingRequest)

		if len(onboardingRequest) > 0 {
			_, err = onboardingMgr.StartOnboarding(ctx, onboardingRequest[0])
			err := ir.updateInstance(ctx, id)
			if err != nil {
				log.Errorf("Failed to update instance with ID: %s,  %s", id, err)
			}
		} else {
			log.Errorf("Failed to start onboarding for instance ID %s: %v", inst.GetResourceId(), err)
		}

	}

	if inst.GetDesiredState() == computev1.InstanceState_INSTANCE_STATE_DELETED {
		log.Infof("Deleting instance ID %s (set current status to Deleted)", id)
		err := ir.deleteInstance(ctx, id)
		if err != nil {
			log.Errorf("Failed to update instance with ID: %s", id)
		}
		if directive := handleInventoryError(err, request); directive != nil {
			return directive
		}
		log.Infof("Instance with ID %v has been deleted", id)
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
