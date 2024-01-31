// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package util

import (
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func IsSameHost(
	originalHostres *computev1.HostResource,
	updatedHostres *computev1.HostResource,
	fieldmask *fieldmaskpb.FieldMask,
) (bool, error) {
	// firstly, cloning Host resource to avoid changing its content
	clonedHostres := proto.Clone(originalHostres)
	// with the fieldmask we are filtering out the fields we don't need
	err := util.ValidateMaskAndFilterMessage(clonedHostres, fieldmask, true)
	if err != nil {
		return false, err
	}

	return proto.Equal(clonedHostres, updatedHostres), nil
}

func IsSameHostnic(
	originalHostnic *computev1.HostnicResource,
	updatedHostnic *computev1.HostnicResource,
	fieldmask *fieldmaskpb.FieldMask,
) (bool, error) {
	// firstly, cloning Host resource to avoid changing its content
	clonedHostres := proto.Clone(originalHostnic)
	// with the fieldmask we are filtering out the fields we don't need
	err := util.ValidateMaskAndFilterMessage(clonedHostres, fieldmask, true)
	if err != nil {
		return false, err
	}

	return proto.Equal(clonedHostres, updatedHostnic), nil
}

func GetBmcNicsFromHost(
	host *computev1.HostResource,
) ([]*computev1.HostnicResource, error) {
	bmcNics := make([]*computev1.HostnicResource, 0)
	for _, hostNic := range host.HostNics {
		if hostNic.BmcInterface == true {
			bmcNics = append(bmcNics, hostNic)
		}
	}

	if len(bmcNics) == 0 {
		return nil, inv_errors.Errorfc(codes.NotFound,
			"No BMC interfaces found for Host %s", host.ResourceId)
	}

	return bmcNics, nil
}
