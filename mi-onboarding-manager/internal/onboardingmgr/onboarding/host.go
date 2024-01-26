/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	"context"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/invclient"

	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
)

func UpdateHostStatusByHostGuid(ctx context.Context,
	invClient *invclient.OnboardingInventoryClient,
	hostUUID string, hoststatus computev1.HostStatus,
) error {
	zlog.Info().Msgf("UpdateHostStatusByHostGuid")
	/* Check if any host with the UUID exists or not */
	hostResc, err := invClient.GetHostResourceByUUID(ctx, hostUUID)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("Node Doesn't Exist")
		return err
	} else {
		zlog.Debug().Msgf("Node and its Host Resource Exist")
		zlog.Debug().Msgf("GetHostResourceByUUID = %v", hostResc)
	}

	hostStatusName := computev1.HostStatus_name[int32(hoststatus)]
	zlog.Debug().Msgf("Update host resc (%v) status: %v", hostResc.ResourceId,
		hostStatusName)

	if err = invClient.SetHostStatus(ctx, hostResc.GetResourceId(), hoststatus); err != nil {
		zlog.MiSec().MiError("Failed to update host resource info").Msg("UpdateHostStatusByHostGuid")
		return err
	}

	return nil
}
