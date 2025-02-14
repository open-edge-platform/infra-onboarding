/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	"context"

	inv_status "github.com/intel/infra-core/inventory/v2/pkg/status"
	"github.com/intel/infra-onboarding/onboarding-manager/internal/invclient"
	om_status "github.com/intel/infra-onboarding/onboarding-manager/pkg/status"
)

func UpdateHostStatusByHostGUID(ctx context.Context,
	tenantID string,
	invClient *invclient.OnboardingInventoryClient,
	hostUUID string, statusDetails string,
	onboardingStatus inv_status.ResourceStatus,
) error {
	zlog.Info().Msgf("UpdateHostStatusByHostGUID")

	/* Check if any host with the UUID exists or not */
	hostResc, err := invClient.GetHostResourceByUUID(ctx, tenantID, hostUUID)
	if err != nil {
		zlog.InfraSec().InfraErr(err).Msgf("Node Doesn't Exist")
		return err
	}
	zlog.Debug().Msgf("Node and its Host Resource Exist")
	zlog.Debug().Msgf("GetHostResourceByUUID = %v", hostResc)

	if statusDetails != "" {
		onboardingStatus = om_status.WithDetails(onboardingStatus, statusDetails)
	}

	zlog.Debug().Msgf("Update host resc (%v) status", hostResc.ResourceId)
	zlog.Debug().Msgf("Update Host (%v) onboarding status: %v", hostResc.ResourceId, onboardingStatus)

	if err = invClient.SetHostOnboardingStatus(ctx, tenantID,
		hostResc.GetResourceId(), onboardingStatus); err != nil {
		zlog.InfraSec().InfraError("Failed to update host resource info").Msg("UpdateHostStatusByHostGUID")
		return err
	}

	return nil
}
