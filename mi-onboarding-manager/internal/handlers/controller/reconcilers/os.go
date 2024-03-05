// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package reconcilers

import (
	"context"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/tracing"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"

	dkam "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/api/grpc/dkammgr"
	onboarding "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/onboarding"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/os/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
	rec_v2 "github.com/onosproject/onos-lib-go/pkg/controller/v2"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

var (
	clientName = "OSReconciler"
	zlogOs     = logging.GetLogger(clientName)
)

// OsReconciler is responsible for reconciling operating system instances.
type OsReconciler struct {
	invClient     *invclient.OnboardingInventoryClient
	enableTracing bool
}

// NewOsReconciler creates a new OsReconciler instance with the given InventoryClient.
func NewOsReconciler(c *invclient.OnboardingInventoryClient, enableTracing bool) *OsReconciler {
	return &OsReconciler{
		invClient:     c,
		enableTracing: enableTracing,
	}
}

// Reconcile is responsible for reconciling operating system instances based on the provided request.
func (osr *OsReconciler) Reconcile(ctx context.Context,
	request rec_v2.Request[ResourceID],
) rec_v2.Directive[ResourceID] {
	if osr.enableTracing {
		ctx = tracing.StartTrace(ctx, "MIOnboardingManager", "OsReconciler")
		defer tracing.StopTrace(ctx)
	}

	resourceID := request.ID.String()
	zlogOs.MiSec().Debug().Msgf("Reconciling os instance : %s", resourceID)
	osre, err := osr.invClient.GetOSResourceByResourceID(ctx, resourceID)
	if directive := HandleInventoryError(err, request); directive != nil {
		return directive
	}

	// skip reconciliation if Repo URL is already set.
	// In the future, we should introduce current/desired state to drive reconciliation.
	if osre.RepoUrl != "" {
		zlogInst.Debug().Msgf("OS (%s) reconciliation skipped. RepoURL set to %s",
			resourceID, osre.RepoUrl)
		return request.Ack()
	}

	return osr.reconcileOsInstance(ctx, request, osre)
}

func (osr *OsReconciler) reconcileOsInstance(
	ctx context.Context,
	request rec_v2.Request[ResourceID],
	osinst *osv1.OperatingSystemResource,
) rec_v2.Directive[ResourceID] {
	id := osinst.GetResourceId()
	zlogOs.MiSec().Info().Msgf("Reconciling OS instance with ID : %s", id)

	response, err := onboarding.GetOSResourceFromDkamService(ctx, osinst.Name, osinst.Architecture)
	if err != nil {
		zlogOs.Err(err).Msgf("Failed to trigger DKAM for os instance ID : %s", id)
		return request.Ack()
	}

	updatedOSResource, fieldmask, err := PopulateOSResourceFromDKAMResponse(response)
	if err != nil {
		return request.Ack()
	}

	// check if there are changes to OS Resource, If not, skip updating to limit number of requests to Inventory
	// This is used only once if RepoURL is not set. Once it's set this code will not be reachable,
	// but since it's not hurtful, we keep it for future usage.
	isSame, err := IsSameOSResource(osinst, updatedOSResource, fieldmask)
	if err != nil {
		return request.Ack()
	}

	if isSame {
		zlogOs.Debug().Msgf("Skipping OS Resource update for OS (%s) - no changes: %v", osinst.GetResourceId(), osinst)
		return request.Ack()
	}

	updatedOSResource.ResourceId = osinst.ResourceId

	err = osr.invClient.UpdateInvResourceFields(ctx, updatedOSResource, fieldmask.GetPaths())
	if err != nil {
		zlogOs.Err(err).Msgf("Failed to update os instance with ID : %s", id)
		return request.Ack()
	}

	return request.Ack()
}

func IsSameOSResource(
	originalOSRes *osv1.OperatingSystemResource,
	updatedOSRes *osv1.OperatingSystemResource,
	fieldmask *fieldmaskpb.FieldMask,
) (bool, error) {
	// firstly, cloning Host resource to avoid changing its content
	clonedOsRes := proto.Clone(originalOSRes)
	// with the fieldmask we are filtering out the fields we don't need
	err := util.ValidateMaskAndFilterMessage(clonedOsRes, fieldmask, true)
	if err != nil {
		return false, err
	}

	return proto.Equal(clonedOsRes, updatedOSRes), nil
}

func PopulateOSResourceFromDKAMResponse(dkamResponse *dkam.GetArtifactsResponse) (
	*osv1.OperatingSystemResource, *fieldmaskpb.FieldMask, error,
) {
	zlogOs.MiSec().Debug().Msg("Populating OS resource with DKAM artifacts")

	if dkamResponse == nil {
		zlogOs.MiSec().MiError("invalid input: DKAM response is nil").Msg("")
		return nil, nil, errors.Errorfc(codes.InvalidArgument, "invalid input: DKAM response is nil")
	}

	osr := &osv1.OperatingSystemResource{}
	fieldmask := &fieldmaskpb.FieldMask{
		Paths: []string{
			osv1.OperatingSystemResourceFieldRepoUrl,
		},
	}
	result := dkamResponse.OsUrl + ";" + dkamResponse.OverlayscriptUrl
	osr.RepoUrl = result

	return osr, fieldmask, nil
}
