/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	"context"

	"google.golang.org/grpc/codes"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/errors"
	inv_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/status"
)

func UpdateInstanceStatusByGUID(ctx context.Context,
	invClient *invclient.OnboardingInventoryClient,
	hostUUID string, provisioningStatus inv_status.ResourceStatus,
) error {
	zlog.Info().Msg("UpdateInstanceStatusByGUID")

	hostResc, err := invClient.GetHostResourceByUUID(ctx, hostUUID)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("Node Doesn't Exist")
		return err
	}
	zlog.Debug().Msgf("Node and its Host Resource Exist")
	zlog.Debug().Msgf("GetHostResourceBySN = %v", hostResc)

	instanceResc := hostResc.GetInstance()
	if instanceResc == nil {
		err = inv_errors.Errorfc(codes.NotFound, "Instance Doesn't Exist")
		zlog.MiSec().MiErr(err).Msgf(hostUUID)
		return err
	}
	zlog.Debug().Msgf("Node and its Instance Resource Exist")
	zlog.Debug().Msgf("GetInstanceResourceBySN = %v", instanceResc)

	zlog.Debug().Msgf("Update Instance resc (%v) status", instanceResc.ResourceId)
	zlog.Debug().Msgf("Update Instance (%v) provisioning status: %v", instanceResc.ResourceId, provisioningStatus)

	if err = invClient.SetInstanceProvisioningStatus(ctx, instanceResc.GetResourceId(), provisioningStatus); err != nil {
		zlog.MiSec().MiErr(err).Msgf("Failed to update status of Instance %v", instanceResc.ResourceId)
		return err
	}

	return nil
}
