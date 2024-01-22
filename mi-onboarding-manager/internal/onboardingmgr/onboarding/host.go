/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	"context"

	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	inv_client "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/client"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/util"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/pkg/maestro"
	"google.golang.org/grpc/codes"
)

func UpdateHostStatusByHostGuid(ctx context.Context,
	invClient inv_client.InventoryClient,
	serialnum string, hoststatus computev1.HostStatus,
) error {
	zlog.Info().Msgf("UpdateHostStatusByHostGuid")
	/* Check if any node with the serial num exists or not */
	hostResc, err := GetHostResourceBySN(ctx, invClient, serialnum)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("Node Doesn't Exist")
		return err
	} else {
		zlog.Debug().Msgf("Node and its Host Resource Exist")
		zlog.Debug().Msgf("GetHostResourceBySN = %v", hostResc)
	}

	hostStatusName := computev1.HostStatus_name[int32(hoststatus)]
	zlog.Debug().Msgf("Update host resc (%v) status: %v", hostResc.ResourceId,
		hostStatusName)

	if err = SetHostStatus(ctx, invClient, hostResc.GetResourceId(), hoststatus); err != nil {
		zlog.MiSec().MiError("Failed to update host resource info").Msg("UpdateHostStatusByHostGuid")
		return err
	}

	return nil
}

func SetHostStatus(ctx context.Context, c inv_client.InventoryClient, resourceID string, hostStatus computev1.HostStatus,
) error {
	zlog.Info().Msgf("SetHostStatus")
	updateHost := &computev1.HostResource{
		ResourceId: resourceID,
		HostStatus: hostStatus,
	}

	return UpdateHostStatus(ctx, c, updateHost)
}

func UpdateHostStatus(ctx context.Context, c inv_client.InventoryClient, host *computev1.HostResource) error {
	zlog.Info().Msgf("UpdateHostStatus")
	return maestro.UpdateInvResourceFields(ctx, c, host, []string{
		"host_status",
	})
}

func GetHostResourceBySN(
	ctx context.Context,
	c inv_client.InventoryClient,
	serial string,
) (*computev1.HostResource, error) {
	zlog.Debug().Msgf("Obtaining Host resource by its SN (%s)", serial)
	// FIXME: remove this check and make sure it is covered by validateAll function
	if serial == "" {
		err := inv_errors.Errorfc(codes.InvalidArgument, "Empty serial number")
		zlog.MiSec().MiErr(err).Msg("get host resource by SN with empty serial NO.")
		return nil, err
	}

	filter, err := util.GetFilterFromSetResource(&inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: &computev1.HostResource{
				SerialNumber: serial,
			},
		},
	})
	if err != nil {
		zlog.MiSec().MiErr(err).Msg("Failed to get filter from a Host resource by Serial Number")
		return nil, err
	}
	return listAndReturnHost(ctx, c, filter)
}

func listAndReturnHost(
	ctx context.Context,
	c inv_client.InventoryClient,
	filter *inv_v1.ResourceFilter,
) (*computev1.HostResource, error) {
	zlog.Debug().Msgf("listAndReturnHost")
	resources, err := maestro.ListAllResources(ctx, c, filter)
	if err != nil {
		zlog.MiSec().MiErr(err).Msg("Failed to listAllResources")
		return nil, err
	}

	if len(resources) == 0 {
		zlog.Debug().Msgf("the length is 0")
		return nil, inv_errors.Errorfc(codes.NotFound, "No Resources found")
	}
	zlog.Debug().Msgf("the length is %d", len(resources))

	hostres := resources[0].GetHost()
	if hostres == nil {
		err = inv_errors.Errorfc(codes.Internal, "Empty Host resource")
		zlog.MiSec().MiErr(err).Msg("Inventory returned an empty Host resource")
		return nil, err
	}

	return hostres, nil
}
