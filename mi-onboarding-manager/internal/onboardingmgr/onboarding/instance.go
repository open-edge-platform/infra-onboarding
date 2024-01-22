/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	"context"
	"errors"

	"github.com/apex/log"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	inv_client "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/client"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/util"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/pkg/maestro"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func UpdateInstanceStatusByGuid(ctx context.Context,
	invClient inv_client.InventoryClient,
	serialnum string, instancestatus computev1.InstanceStatus,
) error {
	log.Infof("UpdateInstanceStatusByGuid")

	hostResc, err := GetHostResourceBySN(ctx, invClient, serialnum)
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
		zlog.MiSec().MiErr(err).Msgf(serialnum)
		return err
	}
	zlog.Debug().Msgf("Node and its Instance Resource Exist")
	zlog.Debug().Msgf("GetInstanceResourceBySN = %v", instanceResc)

	instanceStatusName := computev1.InstanceStatus_name[int32(instancestatus)]
	zlog.Debug().Msgf("Update Instance resc (%v) status: %v", instanceResc.ResourceId,
		instanceStatusName)

	if err = SetInstanceStatus(ctx, invClient, instanceResc.GetResourceId(), instancestatus); err != nil {
		zlog.MiSec().MiError("Failed to update Instance resource info").Msg("UpdateInstanceStatusByGuid")
		return err
	}

	return nil
}

func SetInstanceStatus(ctx context.Context, c inv_client.InventoryClient, resourceID string, InstanceStatus computev1.InstanceStatus,
) error {
	log.Infof("SetInstanceStatus")
	updateHost := &computev1.InstanceResource{
		ResourceId: resourceID,
		Status:     InstanceStatus,
	}

	return UpdateInstanceStatus(ctx, c, updateHost)
}

func UpdateInstanceStatus(ctx context.Context, c inv_client.InventoryClient, instance *computev1.InstanceResource) error {
	log.Infof(" UpdateInstanceStatus")
	return maestro.UpdateInvResourceFields(ctx, c, instance, []string{
		computev1.InstanceResourceFieldStatus,
	})
}

func GetInstanceResourceBySN(
	ctx context.Context,
	c inv_client.InventoryClient,
	serial string, host *computev1.HostResource,
) (*computev1.InstanceResource, error) {
	zlog.Debug().Msgf("Obtaining Instance resource by its SN (%s)", serial)
	// FIXME: remove this check and make sure it is covered by validateAll function
	if serial == "" {
		err := inv_errors.Errorfc(codes.InvalidArgument, "Empty serial number")
		zlog.MiSec().MiErr(err).Msg("get Instance resource by SN with empty serial NO.")
		return nil, err
	}

	instanceRes := &computev1.InstanceResource{
		Host: &computev1.HostResource{
			ResourceId: host.GetResourceId(),
		},
	}
	fieldmask, err := fieldmaskpb.New(instanceRes, util.BuildNestedFieldMaskFromFields("host", "resource_id"))
	if err != nil {
		wErr := inv_errors.Wrap(err)
		zlog.MiSec().MiErr(wErr).Msgf("failed to build fieldmask for getting instance by host ID. HostNicId=%s",
			host.GetResourceId())
		return nil, wErr
	}
	filter := &inv_v1.ResourceFilter{
		Resource: &inv_v1.Resource{
			Resource: &inv_v1.Resource_Instance{
				Instance: instanceRes,
			},
		},
		FieldMask: fieldmask,
	}
	if err != nil {
		zlog.MiSec().MiErr(err).Msg("Failed to get filter from a Instance resource by Serial Number")
		return nil, err
	}
	return listAndReturnInstance(ctx, c, filter)
}

func listAndReturnInstance(
	ctx context.Context,
	c inv_client.InventoryClient,
	filter *inv_v1.ResourceFilter,
) (*computev1.InstanceResource, error) {
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

	instanceRes := resources[0].GetInstance()
	if instanceRes == nil {
		err = inv_errors.Errorfc(codes.Internal, "Empty Instance resource")
		zlog.MiSec().MiErr(err).Msg("Inventory returned an empty Instnace resource")
		return nil, err
	}

	return instanceRes, nil
}
