// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package reconcilers

import (
	"context"
	"fmt"

	osv1 "github.com/intel/infra-core/inventory/v2/pkg/api/os/v1"
	"github.com/intel/infra-core/inventory/v2/pkg/logging"
	"github.com/intel/infra-core/inventory/v2/pkg/tracing"
	"github.com/intel/infra-onboarding/dkam/internal/dkammgr"
	"github.com/intel/infra-onboarding/dkam/internal/invclient"
	rec_v2 "github.com/intel/orch-library/go/pkg/controller/v2"
)

var (
	clientName = "OSReconciler"
	zlogOs     = logging.GetLogger(clientName)
)

// OsReconciler is responsible for reconciling operating system instances.
type OsReconciler struct {
	invClient     *invclient.DKAMInventoryClient
	enableTracing bool
}

// NewOsReconciler creates a new OsReconciler instance with the given InventoryClient.
func NewOsReconciler(c *invclient.DKAMInventoryClient, enableTracing bool) *OsReconciler {
	return &OsReconciler{
		invClient:     c,
		enableTracing: enableTracing,
	}
}

// Reconcile is responsible for reconciling operating system instances based on the provided request.
func (osr *OsReconciler) Reconcile(ctx context.Context,
	request rec_v2.Request[ReconcilerID],
) rec_v2.Directive[ReconcilerID] {
	if osr.enableTracing {
		ctx = tracing.StartTrace(ctx, "InfraDKAM", "OsReconciler")
		defer tracing.StopTrace(ctx)
	}

	tenantID, resourceID := UnwrapReconcilerID(request.ID)
	zlogOs.InfraSec().Debug().Msgf("Reconciling OS %s of tenant %s", resourceID, tenantID)
	osre, err := osr.invClient.GetOSResourceByResourceID(ctx, tenantID, resourceID)
	if directive := HandleInventoryError(err, request); directive != nil {
		return directive
	}

	// In the future, we should introduce current/desired state to drive reconciliation.

	return osr.reconcileOs(ctx, request, osre)
}

func (osr *OsReconciler) reconcileOs(
	ctx context.Context,
	request rec_v2.Request[ReconcilerID],
	osinst *osv1.OperatingSystemResource,
) rec_v2.Directive[ReconcilerID] {
	id := osinst.GetResourceId()
	zlogOs.InfraSec().Info().Msgf("Reconciling OS instance with ID : %s", id)
	fmt.Printf("Received AType: %v\n", osinst.OsType)
	// Download OS image
	downloadErr := dkammgr.DownloadOS(ctx, osinst)
	if downloadErr != nil {
		zlogOs.Err(downloadErr).Msgf("Error downloading and converting OS image")
		return request.Ack()
	}

	curationErr := dkammgr.GetCuratedScript(ctx, osinst)
	if curationErr != nil {
		zlogOs.Err(curationErr).Msgf("Error curating script")
		return request.Ack()
	}

	return request.Ack()
}
