package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	inv_client "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/client"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/util"
	"github.com/intel-sandbox/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/pkg/logger"
	"github.com/intel-sandbox/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/pkg/maestro"
	"github.com/intel-sandbox/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/pkg/maestro/reconciler"
	rec_v2 "github.com/onosproject/onos-lib-go/pkg/controller/v2"
	"github.com/pkg/errors"
)

const (
	defaultTickerPeriod = 30 * time.Second
	parallelism         = 1
)

var log = logger.GetLogger()

type Filter func(event *inv_v1.SubscribeEventsResponse) bool

type NBHandler struct {
	invClient   inv_client.InventoryClient
	invEvents   chan *inv_client.WatchEvents
	filters     map[inv_v1.ResourceKind]Filter
	controllers map[inv_v1.ResourceKind]*rec_v2.Controller[reconciler.ResourceID]
	wg          *sync.WaitGroup
	stop        chan bool
}

func NewNBHandler(
	invClient inv_client.InventoryClient,
	invEvents chan *inv_client.WatchEvents,
) (*NBHandler, error) {
	controllers := make(map[inv_v1.ResourceKind]*rec_v2.Controller[reconciler.ResourceID])
	filters := make(map[inv_v1.ResourceKind]Filter)

	hostRcnl := reconciler.NewHostReconciler(invClient)
	hostCtrl := rec_v2.NewController[reconciler.ResourceID](
		hostRcnl.Reconcile, rec_v2.WithParallelism(parallelism))
	controllers[inv_v1.ResourceKind_RESOURCE_KIND_HOST] = hostCtrl
	filters[inv_v1.ResourceKind_RESOURCE_KIND_HOST] = hostEventFilter

	instRcnl := reconciler.NewInstanceReconciler(invClient)
	instCtrl := rec_v2.NewController[reconciler.ResourceID](
		instRcnl.Reconcile, rec_v2.WithParallelism(parallelism))
	controllers[inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE] = instCtrl
	filters[inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE] = instanceEventFilter

	return &NBHandler{
		invClient:   invClient,
		invEvents:   invEvents,
		filters:     filters,
		controllers: controllers,
		wg:          &sync.WaitGroup{},
		stop:        make(chan bool),
	}, nil
}

func (nbh *NBHandler) Start() error {
	ctx := context.Background()
	if err := nbh.reconcileAll(ctx); err != nil {
		return err
	}

	nbh.wg.Add(1)
	go nbh.controlLoop(ctx)

	log.Info("NBHandler started")
	return nil
}

func (nbh *NBHandler) Stop() {
	close(nbh.stop)
	nbh.wg.Wait()
	for _, ctrl := range nbh.controllers {
		ctrl.Stop()
	}
	log.Info("NBHndler stopped")
}

func (nbh *NBHandler) controlLoop(ctx context.Context) {
	ticker := time.NewTicker(defaultTickerPeriod)
	defer ticker.Stop()

	for {
		select {
		case ev, ok := <-nbh.invEvents:
			if !ok {
				log.Info("GRPC inventory event stream closed")
				return
			}
			if !nbh.filterEvent(ev.Event) {
				continue
			}
			nbh.reconcileResource(ev.Event.ResourceId)
		case <-ticker.C:
			if err := nbh.reconcileAll(ctx); err != nil {
				log.Errorf("full reconciliation failed %v", err)
			}
		case <-nbh.stop:
			nbh.wg.Done()
			return
		}
	}
}

func (nbh *NBHandler) filterEvent(event *inv_v1.SubscribeEventsResponse) bool {
	log.Infof("New inventory event received with ID=%v kind=%s", event.ResourceId, event.EventKind)
	if err := event.ValidateAll(); err != nil {
		log.Errorf("invalid event received with ID=%s %s", event.ResourceId, err)
		return false
	}

	expectedKind, err := util.GetResourceKindFromResourceID(event.ResourceId)
	if err != nil {
		log.Errorf("unknown resource kind for ID %s", event.ResourceId)
		return false
	}
	filter, ok := nbh.filters[expectedKind]
	if !ok {
		log.Infof("No filter found for resource kind %s", expectedKind)
		return true
	}
	return filter(event)
}

func (nbh *NBHandler) reconcileAll(ctx context.Context) error {
	log.Info("Reconciling all instances")
	ids, err := maestro.FindAllResources(ctx, nbh.invClient, inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE)
	if err != nil && !inv_errors.IsNotFound(err) {
		return err
	}

	for _, id := range ids {
		err = nbh.reconcileResource(id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (nbh *NBHandler) reconcileResource(resourceID string) error {
	expectedKind, err := util.GetResourceKindFromResourceID(resourceID)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unknown resource kind for resource ID %s", resourceID))
	}

	log.Infof("Reconciling resource %s of kind %s", resourceID, expectedKind)

	controller, ok := nbh.controllers[expectedKind]
	if !ok {
		return fmt.Errorf("unknown resource controller for kind %s with ID %s", expectedKind, resourceID)
	}

	if err = controller.Reconcile(reconciler.ResourceID(resourceID)); err != nil {
		return err
	}
	return nil
}

func instanceEventFilter(event *inv_v1.SubscribeEventsResponse) bool {
	return event.EventKind == inv_v1.SubscribeEventsResponse_EVENT_KIND_UPDATED ||
		event.EventKind == inv_v1.SubscribeEventsResponse_EVENT_KIND_CREATED ||
		event.EventKind == inv_v1.SubscribeEventsResponse_EVENT_KIND_DELETED
}

func hostEventFilter(event *inv_v1.SubscribeEventsResponse) bool {
	return event.EventKind != inv_v1.SubscribeEventsResponse_EVENT_KIND_DELETED
}
