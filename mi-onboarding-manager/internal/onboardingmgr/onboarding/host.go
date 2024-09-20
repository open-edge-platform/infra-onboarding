/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	"context"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	om_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/status"
	inv_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/status"
)

func UpdateHostStatusByHostGUID(ctx context.Context,
	invClient *invclient.OnboardingInventoryClient,
	hostUUID string, statusDetails string,
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

	zlog.Debug().Msgf("Update host resc (%v) status", hostResc.ResourceId)
	zlog.Debug().Msgf("Update Host (%v) onboarding status: %v", hostResc.ResourceId, onboardingStatus)

	if err = invClient.SetHostOnboardingStatus(ctx, hostResc.GetResourceId(), onboardingStatus); err != nil {
		zlog.MiSec().MiError("Failed to update host resource info").Msg("UpdateHostStatusByHostGUID")
		return err
	}

	return nil
}
