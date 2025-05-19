// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"

	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/handlers/controller/reconcilers"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/invclient"
	rec_v2 "github.com/open-edge-platform/orch-library/go/pkg/controller/v2"
)

var (
	loggerName = "OnboardingController"
	zlog       = logging.GetLogger(loggerName)

	// a default interval for periodic reconciliation.
	// Periodic reconciliation guarantees events are handled even
	// if our notification won't deliver an event.
	// Since we are not confident about the reliability of the current reconciliation,
	// we set quite frequent periodic reconciliation (10m), but it should be increased in the future.
	defaultTickerPeriod = 10 * time.Minute
)

const (
	parallelism = 1
)

type Filter func(event *inv_v1.SubscribeEventsResponse) bool

type OnboardingController struct {
	invClient   *invclient.OnboardingInventoryClient
	filters     map[inv_v1.ResourceKind]Filter
	controllers map[inv_v1.ResourceKind]*rec_v2.Controller[reconcilers.ReconcilerID]
	wg          *sync.WaitGroup
	stop        chan bool
}

func New(
	invClient *invclient.OnboardingInventoryClient,
	enableTracing bool,
) (*OnboardingController, error) {
	controllers := make(map[inv_v1.ResourceKind]*rec_v2.Controller[reconcilers.ReconcilerID])
	filters := make(map[inv_v1.ResourceKind]Filter)

	hostRcnl := reconcilers.NewHostReconciler(invClient, enableTracing)
	hostCtrl := rec_v2.NewController[reconcilers.ReconcilerID](
		hostRcnl.Reconcile, rec_v2.WithParallelism(parallelism))
	controllers[inv_v1.ResourceKind_RESOURCE_KIND_HOST] = hostCtrl
	filters[inv_v1.ResourceKind_RESOURCE_KIND_HOST] = hostEventFilter

	instRcnl := reconcilers.NewInstanceReconciler(invClient, enableTracing)
	instCtrl := rec_v2.NewController[reconcilers.ReconcilerID](
		instRcnl.Reconcile, rec_v2.WithParallelism(parallelism))
	controllers[inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE] = instCtrl
	filters[inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE] = instanceEventFilter
	return &OnboardingController{
		invClient:   invClient,
		filters:     filters,
		controllers: controllers,
		wg:          &sync.WaitGroup{},
		stop:        make(chan bool),
	}, nil
}

func (obc *OnboardingController) Start() error {
	if err := obc.reconcileAll(); err != nil {
		return err
	}

	obc.wg.Add(1)
	go obc.controlLoop()

	zlog.InfraSec().Info().Msgf("Onboarding controller started")
	return nil
}

func (obc *OnboardingController) Stop() {
	close(obc.stop)
	obc.wg.Wait()
	for _, ctrl := range obc.controllers {
		ctrl.Stop()
	}
	zlog.InfraSec().Info().Msgf("Onboarding controller stopped")
}

func (obc *OnboardingController) controlLoop() {
	zlog.InfraSec().Info().Msgf("Reconciliation ticker period set to %f minutes", defaultTickerPeriod.Minutes())
	ticker := time.NewTicker(defaultTickerPeriod)
	defer ticker.Stop()

	for {
		select {
		case ev, ok := <-obc.invClient.Watcher:
			if !ok {
				zlog.InfraSec().Fatal().Msg("gRPC stream with inventory closed")
				return
			}
			obc.handleInventoryEvent(ev)
		case ev, ok := <-obc.invClient.InternalWatcher:
			if !ok {
				zlog.InfraSec().Fatal().Msg("internal events channel closed")
				continue
			}
			obc.handleInternalEvent(ev)

		case <-ticker.C:
			if err := obc.reconcileAll(); err != nil {
				zlog.InfraSec().InfraErr(err).Msgf("full reconciliation failed")
			}
		case <-obc.stop:
			obc.wg.Done()
			return
		}
	}
}

func (obc *OnboardingController) handleInternalEvent(event *client.ResourceTenantIDCarrier) {
	zlog.Debug().Msgf("Internal event [tenantID=%s, resourceID=%s]", event.TenantId, event.ResourceId)

	// Propagate the even directly to the reconciler, we don't filter any event, we don't have.
	if err := obc.reconcileResource(event.TenantId, event.ResourceId); err != nil {
		zlog.InfraSec().InfraErr(err).Msgf("reconciliation resource failed")
	}
}

func (obc *OnboardingController) handleInventoryEvent(event *client.WatchEvents) {
	zlog.Debug().Msgf("Inventory event: event=%v", event.Event)

	if !obc.filterEvent(event.Event) {
		zlog.Debug().Msgf("Event %v is not allowed by filter", event.Event)
		return
	}
	tID, resID, err := util.GetResourceKeyFromResource(event.Event.GetResource())
	if err != nil {
		zlog.InfraSec().Err(err).Msgf("Failed to get resource key from event: event=%v", event.Event)
		return
	}
	if err := obc.reconcileResource(tID, resID); err != nil {
		zlog.InfraSec().InfraErr(err).Msgf("reconciliation resource failed")
	}
}

func (obc *OnboardingController) filterEvent(event *inv_v1.SubscribeEventsResponse) bool {
	zlog.Debug().Msgf("New inventory event received. ResourceID=%v, Kind=%s", event.ResourceId, event.EventKind)
	if err := validator.ValidateMessage(event); err != nil {
		zlog.InfraSec().InfraErr(err).Msgf("Invalid event received: %s", event.ResourceId)
		return false
	}

	expectedKind, err := util.GetResourceKindFromResourceID(event.ResourceId)
	if err != nil {
		zlog.InfraSec().InfraErr(err).Msgf("Unknown resource kind for ID %s.", event.ResourceId)
		return false
	}
	filter, ok := obc.filters[expectedKind]
	if !ok {
		zlog.Debug().Msgf("No filter found for resource kind %s, accepting all events", expectedKind)
		return false
	}

	return filter(event)
}

func (obc *OnboardingController) reconcileAll() error {
	zlog.Debug().Msgf("Reconciling all resources")

	// Use context.WithTimeout to set a timeout for the operation
	ctx, cancel := context.WithTimeout(context.Background(), *invclient.ReconcileTimeout)
	defer cancel()

	resourceKinds := []inv_v1.ResourceKind{
		inv_v1.ResourceKind_RESOURCE_KIND_HOST,
		inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE,
	}
	ids, err := obc.invClient.FindAllResources(ctx, resourceKinds)
	if err != nil && !inv_errors.IsNotFound(err) {
		return err
	}

	for _, id := range ids {
		err = obc.reconcileResource(id.GetTenantId(), id.GetResourceId())
		if err != nil {
			return err
		}
	}

	return nil
}

func (obc *OnboardingController) reconcileResource(tenantID, resourceID string) error {
	expectedKind, err := util.GetResourceKindFromResourceID(resourceID)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unknown resource kind for resource ID %s", resourceID))
	}

	zlog.Debug().Msgf("Reconciling resource (%s) of kind=%s",
		fmt.Sprintf("[tenantID=%s, resourceID=%s]", tenantID, resourceID), expectedKind)

	controller, ok := obc.controllers[expectedKind]
	if !ok {
		return fmt.Errorf("unknown resource controller for kind %s with ID %s",
			expectedKind, fmt.Sprintf("[tenantID=%s, resourceID=%s]", tenantID, resourceID))
	}

	if err = controller.Reconcile(reconcilers.NewReconcilerID(tenantID, resourceID)); err != nil {
		zlog.Err(err).Msgf("Error while reconciling resource")
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
