/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	"context"

	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/status"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/invclient"
	om_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/pkg/status"
)

func UpdateHostStatusByHostGUID(ctx context.Context,
	invClient *invclient.OnboardingInventoryClient,
	hostUUID string, hoststatus computev1.HostStatus, statusDetails string,
	onboardingStatus inv_status.ResourceStatus,
) error {
	zlog.Info().Msgf("UpdateHostStatusByHostGUID")
	/* Check if any host with the UUID exists or not */
	hostResc, err := invClient.GetHostResourceByUUID(ctx, hostUUID)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("Node Doesn't Exist")
		return err
	}
	zlog.Debug().Msgf("Node and its Host Resource Exist")
	zlog.Debug().Msgf("GetHostResourceByUUID = %v", hostResc)

	if statusDetails != "" {
		onboardingStatus = om_status.WithDetails(onboardingStatus, statusDetails)
	}

	zlog.Debug().Msgf("Update host resc (%v) status: %v", hostResc.ResourceId,
		hoststatus)
	zlog.Debug().Msgf("Update Host (%v) onboarding status: %v", hostResc.ResourceId, onboardingStatus)

	if err = invClient.SetHostStatus(ctx, hostResc.GetResourceId(), hoststatus, statusDetails, onboardingStatus); err != nil {
		zlog.MiSec().MiError("Failed to update host resource info").Msg("UpdateHostStatusByHostGUID")
		return err
	}

	return nil
}
