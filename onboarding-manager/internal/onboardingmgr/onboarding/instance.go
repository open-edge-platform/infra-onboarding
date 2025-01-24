/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	"context"

	"google.golang.org/grpc/codes"

	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/errors"
	inv_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/status"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/onboarding-manager/internal/invclient"
)

func UpdateInstanceStatusByGUID(ctx context.Context,
	tenantID string,
	invClient *invclient.OnboardingInventoryClient,
	hostUUID string, provisioningStatus inv_status.ResourceStatus,
) error {
	zlog.Info().Msg("UpdateInstanceStatusByGUID")

	hostResc, err := invClient.GetHostResourceByUUID(ctx, tenantID, hostUUID)
	if err != nil {
		zlog.InfraSec().InfraErr(err).Msg("Node Doesn't Exist")
		return err
	}
	zlog.Debug().Msg("Node and its Host Resource Exist")
	zlog.Debug().Msgf("GetHostResourceBySN = %v", hostResc)

	instanceResc := hostResc.GetInstance()
	if instanceResc == nil {
		err = inv_errors.Errorfc(codes.NotFound, "Instance Doesn't Exist")
		zlog.Debug().Msgf("Host UUID: %s", hostUUID)
		zlog.InfraSec().InfraErr(err).Msgf("Host UUID")
		return err
	}
	zlog.Debug().Msg("Node and its Instance Resource Exist")
	zlog.Debug().Msgf("GetInstanceResourceBySN = %v", instanceResc)

	zlog.Debug().Msgf("Update Instance resc (%v) status", instanceResc.ResourceId)
	zlog.Debug().Msgf("Update Instance (%v) provisioning status: %v", instanceResc.ResourceId, provisioningStatus)

	if err = invClient.SetInstanceProvisioningStatus(ctx, tenantID,
		instanceResc.GetResourceId(), provisioningStatus); err != nil {
		zlog.InfraSec().InfraErr(err).Msgf("Failed to update status of Instance")
		return err
	}

	return nil
}
