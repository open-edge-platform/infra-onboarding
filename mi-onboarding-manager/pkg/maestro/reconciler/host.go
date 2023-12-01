package reconciler

import (
	"context"

	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_client "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/client"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/pkg/maestro"
	rec_v2 "github.com/onosproject/onos-lib-go/pkg/controller/v2"
)

type HostReconciler struct {
	invClient inv_client.InventoryClient
}

func NewHostReconciler(c inv_client.InventoryClient) *HostReconciler {
	return &HostReconciler{
		invClient: c,
	}
}

func (hr *HostReconciler) Reconcile(ctx context.Context, request rec_v2.Request[ResourceID]) rec_v2.Directive[ResourceID] {
	resourceID := request.ID.String()
	log.Infof("Reconciling host %s", resourceID)

	inst, err := maestro.GetHostResourceByResourceID(ctx, hr.invClient, resourceID)
	if err != nil {
		log.Errorf("Failed to get host %s %v", resourceID, err)
	}
	if directive := handleInventoryError(err, request); directive != nil {
		return directive
	}

	if inst.DesiredState == inst.CurrentState {
		log.Infof("Host %s reconciliation skipped", resourceID)
		return request.Ack()
	}

	return hr.reconcileInstance(ctx, request, inst)
}

func (hr *HostReconciler) reconcileInstance(
	ctx context.Context,
	request rec_v2.Request[ResourceID],
	inst *computev1.HostResource,
) rec_v2.Directive[ResourceID] {
	id := inst.GetResourceId()
	log.Infof("Reconciling Host with ID: %s current state: %v desired state: %v",
		id, inst.GetCurrentState(), inst.GetDesiredState())

	if inst.GetDesiredState() == computev1.HostState_HOST_STATE_DELETED {
		log.Infof("Deleting host ID %s (set current status to Deleted)", id)
		err := maestro.DeleteHostResource(ctx, hr.invClient, &computev1.HostResource{ResourceId: id})
		if err != nil {
			log.Errorf("Failed to update host with ID: %s", id)
		}
		if directive := handleInventoryError(err, request); directive != nil {
			return directive
		}
		log.Infof("Host with ID %v has been deleted", id)
		return request.Ack()
	}

	return request.Ack()
}
