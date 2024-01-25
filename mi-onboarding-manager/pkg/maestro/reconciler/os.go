/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/
package reconciler

import (
	"context"
	"os"

	dkam "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/api/grpc/dkammgr"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/os/v1"
	inv_client "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/client"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
	onboarding "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/onboarding"
	rec_v2 "github.com/onosproject/onos-lib-go/pkg/controller/v2"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

var (
	clientName = "OnboardingInventoryClient"
	zlog       = logging.GetLogger(clientName)
)

// OsReconciler is responsible for reconciling operating system instances.
type OsReconciler struct {
	invClient inv_client.InventoryClient
}

// NewOsReconciler creates a new OsReconciler instance with the given InventoryClient.
func NewOsReconciler(c inv_client.InventoryClient) *OsReconciler {
	return &OsReconciler{
		invClient: c,
	}
}

// Reconcile is responsible for reconciling operating system instances based on the provided request.
func (osr *OsReconciler) Reconcile(ctx context.Context, request rec_v2.Request[ResourceID]) rec_v2.Directive[ResourceID] {
	resourceID := request.ID.String()
	zlog.MiSec().Info().Msgf("Reconciling os instance : %s", resourceID)
	osre, err := getOsInstanceByID(ctx, osr.invClient, resourceID)
	if err != nil {
		zlog.Err(err).Msgf("Failed to get os instance : %s", resourceID)
	}
	if osdirective := handleInventoryError(err, request); osdirective != nil {
		return osdirective
	}
	return osr.reconcileOsInstance(ctx, request, osre)
}

func (osr *OsReconciler) reconcileOsInstance(
	ctx context.Context,
	request rec_v2.Request[ResourceID],
	osinst *osv1.OperatingSystemResource,
) rec_v2.Directive[ResourceID] {
	id := osinst.GetResourceId()
	zlog.MiSec().Info().Msgf("Reconciling os instance with ID : %s", id)

	disableFeatureX := os.Getenv("DISABLE_FEATUREX")

	if disableFeatureX == "true" {
		response, err := onboarding.GetOSResourceFromDkamService(ctx)
		if err != nil {
			zlog.Err(err).Msgf("Failed to trigger DKAM for os instance ID : %s", id)
			return request.Ack()
		}
		err = osr.updateOsInstance(ctx, osr.invClient, osinst, response, id)
		if err != nil {
			zlog.Err(err).Msgf("Failed to update os instance with ID : %s", id)
		}
	}
	return request.Ack()
}

func (osr *OsReconciler) updateOsInstance(ctx context.Context, c inv_client.InventoryClient, osre *osv1.OperatingSystemResource, dkam *dkam.GetArtifactsResponse, id string) error {
	fieldMask := &fieldmaskpb.FieldMask{
		Paths: []string{
			osv1.OperatingSystemResourceFieldRepoUrl,
		},
	}
	res := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Os{
			Os: &osv1.OperatingSystemResource{
				RepoUrl: dkam.ManifestFile,
			},
		},
	}
	_, err := c.Update(
		ctx,
		osre.ResourceId,
		fieldMask,
		res,
	)
	if err != nil {
		return err
	}
	return nil
}

func getOsInstanceByID(ctx context.Context, c inv_client.InventoryClient, resourceID string) (*osv1.OperatingSystemResource, error) {
	res, err := c.Get(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	osre := res.GetResource().GetOs()
	if err := osre.ValidateAll(); err != nil {
		return nil, inv_errors.Wrap(err)
	}

	return osre, nil
}
