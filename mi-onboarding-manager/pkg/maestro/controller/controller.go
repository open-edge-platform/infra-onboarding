/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/
package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	inv_client "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/client"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/util"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/pkg/maestro"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/pkg/maestro/reconciler"
	rec_v2 "github.com/onosproject/onos-lib-go/pkg/controller/v2"
	"github.com/pkg/errors"
)

var (
	loggerName = "OnboardingNBHandler"
	zlog       = logging.GetLogger(loggerName)
)

const (
	defaultTickerPeriod = 3 * time.Second
	parallelism         = 1
)

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

	zlog.MiSec().Info().Msgf("NB handler started")
	return nil
}

func (nbh *NBHandler) Stop() {
	close(nbh.stop)
	nbh.wg.Wait()
	for _, ctrl := range nbh.controllers {
		ctrl.Stop()
	}
	zlog.MiSec().Info().Msgf("NB handler stopped")
}

func (nbh *NBHandler) controlLoop(ctx context.Context) {
	ticker := time.NewTicker(defaultTickerPeriod)
	defer ticker.Stop()

	for {
		select {
		case ev, ok := <-nbh.invEvents:
			if !ok {
				zlog.MiSec().Fatal().Msg("gRPC stream with inventory closed")
				return
			}
			if !nbh.filterEvent(ev.Event) {
				zlog.Debug().Msgf("Event %v is not allowed by filter", ev.Event)
				continue
			}
			nbh.reconcileResource(ev.Event.ResourceId)
		case <-ticker.C:
			if err := nbh.reconcileAll(ctx); err != nil {
				zlog.MiSec().MiErr(err).Msgf("full reconciliation failed")
			}
		case <-nbh.stop:
			nbh.wg.Done()
			return
		}
	}
}

func (nbh *NBHandler) filterEvent(event *inv_v1.SubscribeEventsResponse) bool {
	zlog.Debug().Msgf("New inventory event received. ResourceID=%v, Kind=%s", event.ResourceId, event.EventKind)
	if err := event.ValidateAll(); err != nil {
		zlog.MiSec().MiErr(err).Msgf("Invalid event received: %s", event.ResourceId)
		return false
	}

	expectedKind, err := util.GetResourceKindFromResourceID(event.ResourceId)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("Unknown resource kind for ID %s.", event.ResourceId)
		return false
	}
	filter, ok := nbh.filters[expectedKind]
	if !ok {
		zlog.Debug().Msgf("No filter found for resource kind %s, accepting all events", expectedKind)
		return true
	}
	return filter(event)
}

func (nbh *NBHandler) reconcileAll(ctx context.Context) error {
	zlog.Info().Msg("Reconciling all instances")
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

	zlog.Debug().Msgf("Reconciling resource (%s) of kind=%s", resourceID, expectedKind)

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
