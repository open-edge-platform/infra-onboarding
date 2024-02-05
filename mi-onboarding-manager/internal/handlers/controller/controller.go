// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/handlers/controller/reconcilers"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/invclient"

	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/util"
	rec_v2 "github.com/onosproject/onos-lib-go/pkg/controller/v2"
	"github.com/pkg/errors"
)

var (
	loggerName = "OnboardingController"
	zlog       = logging.GetLogger(loggerName)
)

const (
	defaultTickerPeriod = 3 * time.Second
	parallelism         = 1
)

type Filter func(event *inv_v1.SubscribeEventsResponse) bool

type OnboardingController struct {
	invClient   *invclient.OnboardingInventoryClient
	filters     map[inv_v1.ResourceKind]Filter
	controllers map[inv_v1.ResourceKind]*rec_v2.Controller[reconcilers.ResourceID]
	wg          *sync.WaitGroup
	stop        chan bool
}

func New(
	invClient *invclient.OnboardingInventoryClient,
) (*OnboardingController, error) {
	controllers := make(map[inv_v1.ResourceKind]*rec_v2.Controller[reconcilers.ResourceID])
	filters := make(map[inv_v1.ResourceKind]Filter)

	hostRcnl := reconcilers.NewHostReconciler(invClient)
	hostCtrl := rec_v2.NewController[reconcilers.ResourceID](
		hostRcnl.Reconcile, rec_v2.WithParallelism(parallelism))
	controllers[inv_v1.ResourceKind_RESOURCE_KIND_HOST] = hostCtrl
	filters[inv_v1.ResourceKind_RESOURCE_KIND_HOST] = hostEventFilter

	instRcnl := reconcilers.NewInstanceReconciler(invClient)
	instCtrl := rec_v2.NewController[reconcilers.ResourceID](
		instRcnl.Reconcile, rec_v2.WithTimeout(3*time.Hour), rec_v2.WithParallelism(parallelism))
	controllers[inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE] = instCtrl
	filters[inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE] = instanceEventFilter
	osRcnl := reconcilers.NewOsReconciler(invClient)
	osCtrl := rec_v2.NewController[reconcilers.ResourceID](
		osRcnl.Reconcile, rec_v2.WithParallelism(parallelism))
	controllers[inv_v1.ResourceKind_RESOURCE_KIND_OS] = osCtrl
	filters[inv_v1.ResourceKind_RESOURCE_KIND_OS] = osEventFilter

	return &OnboardingController{
		invClient:   invClient,
		filters:     filters,
		controllers: controllers,
		wg:          &sync.WaitGroup{},
		stop:        make(chan bool),
	}, nil
}

func (obc *OnboardingController) Start() error {
	ctx := context.Background()
	if err := obc.reconcileAll(ctx); err != nil {
		return err
	}

	obc.wg.Add(1)
	go obc.controlLoop(ctx)

	zlog.MiSec().Info().Msgf("Onboarding controller started")
	return nil
}

func (obc *OnboardingController) Stop() {
	close(obc.stop)
	obc.wg.Wait()
	for _, ctrl := range obc.controllers {
		ctrl.Stop()
	}
	zlog.MiSec().Info().Msgf("Onboarding controller stopped")
}

func (obc *OnboardingController) controlLoop(ctx context.Context) {
	ticker := time.NewTicker(defaultTickerPeriod)
	defer ticker.Stop()

	for {
		select {
		case ev, ok := <-obc.invClient.Watcher:
			if !ok {
				zlog.MiSec().Fatal().Msg("gRPC stream with inventory closed")
				return
			}
			if !obc.filterEvent(ev.Event) {
				zlog.Debug().Msgf("Event %v is not allowed by filter", ev.Event)
				continue
			}
			obc.reconcileResource(ev.Event.ResourceId)
		case <-ticker.C:
			if err := obc.reconcileAll(ctx); err != nil {
				zlog.MiSec().MiErr(err).Msgf("full reconciliation failed")
			}
		case <-obc.stop:
			obc.wg.Done()
			return
		}
	}
}

func (obc *OnboardingController) filterEvent(event *inv_v1.SubscribeEventsResponse) bool {
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
	filter, ok := obc.filters[expectedKind]
	if !ok {
		zlog.Debug().Msgf("No filter found for resource kind %s, accepting all events", expectedKind)
		return true
	}
	return filter(event)
}

func (obc *OnboardingController) reconcileAll(ctx context.Context) error {
	zlog.Debug().Msgf("Reconciling all instances")

	resourceKinds := []inv_v1.ResourceKind{inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE, inv_v1.ResourceKind_RESOURCE_KIND_OS}
	ids, err := obc.invClient.FindAllResources(ctx, resourceKinds)
	if err != nil && !inv_errors.IsNotFound(err) {
		return err
	}

	for _, id := range ids {
		err = obc.reconcileResource(id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (obc *OnboardingController) reconcileResource(resourceID string) error {
	expectedKind, err := util.GetResourceKindFromResourceID(resourceID)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unknown resource kind for resource ID %s", resourceID))
	}

	zlog.Debug().Msgf("Reconciling resource (%s) of kind=%s", resourceID, expectedKind)

	controller, ok := obc.controllers[expectedKind]
	if !ok {
		return fmt.Errorf("unknown resource controller for kind %s with ID %s", expectedKind, resourceID)
	}

	if err = controller.Reconcile(reconcilers.ResourceID(resourceID)); err != nil {
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

func osEventFilter(event *inv_v1.SubscribeEventsResponse) bool {
	return event.EventKind == inv_v1.SubscribeEventsResponse_EVENT_KIND_UPDATED ||
		event.EventKind == inv_v1.SubscribeEventsResponse_EVENT_KIND_CREATED ||
		event.EventKind == inv_v1.SubscribeEventsResponse_EVENT_KIND_DELETED
}
