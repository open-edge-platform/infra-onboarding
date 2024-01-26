/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	"context"
	"errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/invclient"

	"github.com/apex/log"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
)

func UpdateInstanceStatusByGuid(ctx context.Context,
	invClient *invclient.OnboardingInventoryClient,
	hostUUID string, instancestatus computev1.InstanceStatus,
) error {
	log.Infof("UpdateInstanceStatusByGuid")

	hostResc, err := invClient.GetHostResourceByUUID(ctx, hostUUID)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("Node Doesn't Exist")
		return err
	} else {
		zlog.Debug().Msgf("Node and its Host Resource Exist")
		zlog.Debug().Msgf("GetHostResourceBySN = %v", hostResc)
	}

	instanceResc := hostResc.GetInstance()
	if instanceResc == nil {
		err := errors.New("Instance Doesn't Exist")
		zlog.MiSec().MiErr(err).Msgf(hostUUID)
		return err
	}
	zlog.Debug().Msgf("Node and its Instance Resource Exist")
	zlog.Debug().Msgf("GetInstanceResourceBySN = %v", instanceResc)

	instanceStatusName := computev1.InstanceStatus_name[int32(instancestatus)]
	zlog.Debug().Msgf("Update Instance resc (%v) status: %v", instanceResc.ResourceId,
		instanceStatusName)

	if err = invClient.SetInstanceStatus(ctx, instanceResc.GetResourceId(), instancestatus); err != nil {
		zlog.MiSec().MiError("Failed to update Instance resource info").Msg("UpdateInstanceStatusByGuid")
		return err
	}

	return nil
}
