// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"

	inv_v1 "github.com/intel/infra-core/inventory/v2/pkg/api/inventory/v1"
	inv_errors "github.com/intel/infra-core/inventory/v2/pkg/errors"
	"github.com/intel/infra-core/inventory/v2/pkg/logging"
	"github.com/intel/infra-core/inventory/v2/pkg/util"
	"github.com/intel/infra-core/inventory/v2/pkg/validator"
	"github.com/intel/infra-onboarding/dkam/internal/handlers/controller/reconcilers"
	"github.com/intel/infra-onboarding/dkam/internal/invclient"
	rec_v2 "github.com/intel/orch-library/go/pkg/controller/v2"
)

var (
	loggerName = "DKAMController"
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
	timeOut5m   = 5 * time.Minute
)

type Filter func(event *inv_v1.SubscribeEventsResponse) bool

type DKAMController struct {
	invClient   *invclient.DKAMInventoryClient
	filters     map[inv_v1.ResourceKind]Filter
	controllers map[inv_v1.ResourceKind]*rec_v2.Controller[reconcilers.ReconcilerID]
	wg          *sync.WaitGroup
	stop        chan bool
}

func New(
	invClient *invclient.DKAMInventoryClient,
	enableTracing bool,
) (*DKAMController, error) {
	controllers := make(map[inv_v1.ResourceKind]*rec_v2.Controller[reconcilers.ReconcilerID])
	filters := make(map[inv_v1.ResourceKind]Filter)

	osRcnl := reconcilers.NewOsReconciler(invClient, enableTracing)
	osCtrl := rec_v2.NewController[reconcilers.ReconcilerID](
		osRcnl.Reconcile, rec_v2.WithParallelism(parallelism), rec_v2.WithTimeout(timeOut5m))
	controllers[inv_v1.ResourceKind_RESOURCE_KIND_OS] = osCtrl
	filters[inv_v1.ResourceKind_RESOURCE_KIND_OS] = osEventFilter

	return &DKAMController{
		invClient:   invClient,
		filters:     filters,
		controllers: controllers,
		wg:          &sync.WaitGroup{},
		stop:        make(chan bool),
	}, nil
}

func (obc *DKAMController) Start() error {
	ctx := context.Background()
	if err := obc.reconcileAll(ctx); err != nil {
		return err
	}

	obc.wg.Add(1)
	go obc.controlLoop(ctx)

	zlog.InfraSec().Info().Msgf("DKAM controller started")
	return nil
}

func (obc *DKAMController) Stop() {
	close(obc.stop)
	obc.wg.Wait()
	for _, ctrl := range obc.controllers {
		ctrl.Stop()
	}
	zlog.InfraSec().Info().Msgf("Onboarding controller stopped")
}

func (obc *DKAMController) controlLoop(ctx context.Context) {
	ticker := time.NewTicker(defaultTickerPeriod)
	defer ticker.Stop()

	for {
		select {
		case ev, ok := <-obc.invClient.Watcher:
			if !ok {
				zlog.InfraSec().Fatal().Msg("gRPC stream with inventory closed")
				return
			}
			if !obc.filterEvent(ev.Event) {
				zlog.Debug().Msgf("Event %v is not allowed by filter", ev.Event)
				continue
			}
			if err := obc.reconcileResource(ev.Event.Resource); err != nil {
				zlog.InfraSec().InfraErr(err).Msgf("reconciliation resource failed")
			}
		case <-ticker.C:
			if err := obc.reconcileAll(ctx); err != nil {
				zlog.InfraSec().InfraErr(err).Msgf("full reconciliation failed")
			}
		case <-obc.stop:
			obc.wg.Done()
			return
		}
	}
}

func (obc *DKAMController) filterEvent(event *inv_v1.SubscribeEventsResponse) bool {
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
		return true
	}

	return filter(event)
}

func (obc *DKAMController) reconcileAll(ctx context.Context) error {
	zlog.Debug().Msgf("Reconciling all resources")

	resourceKinds := []inv_v1.ResourceKind{
		inv_v1.ResourceKind_RESOURCE_KIND_OS,
	}
	resources, err := obc.invClient.ListAllResources(ctx, resourceKinds)
	if err != nil && !inv_errors.IsNotFound(err) {
		return err
	}

	for _, resource := range resources {
		err = obc.reconcileResource(resource)
		if err != nil {
			return err
		}
	}

	return nil
}

func (obc *DKAMController) reconcileResource(resource *inv_v1.Resource) error {
	tenantID, resourceID, err := util.GetResourceKeyFromResource(resource)
	if err != nil {
		return err
	}

	expectedKind, err := util.GetResourceKindFromResourceID(resourceID)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unknown resource kind for resource ID %s", resourceID))
	}

	zlog.Debug().Msgf("Reconciling resource (%s) of kind=%s", resourceID, expectedKind)

	controller, ok := obc.controllers[expectedKind]
	if !ok {
		return fmt.Errorf("unknown resource controller for kind %s with ID %s", expectedKind, resourceID)
	}

	if err = controller.Reconcile(reconcilers.WrapReconcilerID(tenantID, resourceID)); err != nil {
		zlog.Err(err).Msgf("Error while reconciling resource ID %s", resourceID)
		return err
	}
	return nil
}

func osEventFilter(event *inv_v1.SubscribeEventsResponse) bool {
	return event.EventKind == inv_v1.SubscribeEventsResponse_EVENT_KIND_UPDATED ||
		event.EventKind == inv_v1.SubscribeEventsResponse_EVENT_KIND_CREATED ||
		event.EventKind == inv_v1.SubscribeEventsResponse_EVENT_KIND_DELETED
}
