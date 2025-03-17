// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package invclient

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"

	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	network_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/network/v1"
	osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
)

func getInventoryResourceAndID(resource proto.Message) (*inv_v1.Resource, string, error) {
	invResource := &inv_v1.Resource{}

	if resource == nil {
		err := inv_errors.Errorfc(codes.InvalidArgument, "no resource provided")
		zlog.InfraSec().InfraErr(err).Msgf("getInventoryResourceAndID")
		return nil, "", err
	}

	invResourceID, err := setInventoryResourceAndID(resource, invResource)
	if err != nil {
		return invResource, invResourceID, err
	}

	return invResource, invResourceID, nil
}

func setInventoryResourceAndID(resource proto.Message, invResource *inv_v1.Resource) (string, error) {
	var invResourceID string
	switch res := resource.(type) {
	case *computev1.HostResource:
		invResource.Resource = &inv_v1.Resource_Host{
			Host: res,
		}
		invResourceID = res.GetResourceId()
	case *computev1.HoststorageResource:
		invResource.Resource = &inv_v1.Resource_Hoststorage{
			Hoststorage: res,
		}
		invResourceID = res.GetResourceId()
	case *computev1.HostnicResource:
		invResource.Resource = &inv_v1.Resource_Hostnic{
			Hostnic: res,
		}
		invResourceID = res.GetResourceId()
	case *computev1.HostusbResource:
		invResource.Resource = &inv_v1.Resource_Hostusb{
			Hostusb: res,
		}
		invResourceID = res.GetResourceId()
	case *computev1.HostgpuResource:
		invResource.Resource = &inv_v1.Resource_Hostgpu{
			Hostgpu: res,
		}
		invResourceID = res.GetResourceId()
	case *network_v1.IPAddressResource:
		invResource.Resource = &inv_v1.Resource_Ipaddress{
			Ipaddress: res,
		}
		invResourceID = res.GetResourceId()
	case *computev1.InstanceResource:
		invResource.Resource = &inv_v1.Resource_Instance{
			Instance: res,
		}
		invResourceID = res.GetResourceId()
	case *osv1.OperatingSystemResource:
		invResource.Resource = &inv_v1.Resource_Os{
			Os: res,
		}
		invResourceID = res.GetResourceId()
	default:
		err := inv_errors.Errorfc(codes.InvalidArgument, "unsupported resource type: %t", resource)
		zlog.InfraSec().InfraErr(err).Msg("getInventoryResourceAndID")
		return invResourceID, err
	}
	return invResourceID, nil
}
