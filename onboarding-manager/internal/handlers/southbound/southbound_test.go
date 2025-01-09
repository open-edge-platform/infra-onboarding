// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package southbound_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/onboarding-manager/pkg/api"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/api/compute/v1"
	inv_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/testing"
)

var (
	hostGUID = "BFD3B398-9A4B-480D-AB53-4050ED108F5C"
	hostSN   = "87654321"
)

const (
	tenant1 = "11111111-1111-1111-1111-111111111111"
	tenant2 = "22222222-2222-2222-2222-222222222222"
)

func TestSouthbound_CreateNodes(t *testing.T) {
	// already exists, don't create
	t.Run("AlreadyExists", func(t *testing.T) {
		dao := inv_testing.NewInvResourceDAOOrFail(t)
		ctx, cancel := inv_testing.CreateContextWithENJWT(t, tenant1)
		defer cancel()
		h1 := dao.CreateHost(t, tenant1)
		inCreate := &pb.NodeRequest{
			Payload: []*pb.NodeData{
				{
					Hwdata: []*pb.HwData{
						{
							MacId:          "90:49:fa:07:6c:fd",
							SutIp:          "10.10.1.1",
							Serialnum:      hostSN,
							Uuid:           h1.Uuid,
							BmcIp:          "10.10.10.10",
							BmcInterface:   true,
							HostNicDevName: "bmc0",
						},
					},
				},
			},
		}
		_, err := OMTestClient.CreateNodes(ctx, inCreate)
		require.NoError(t, err)
	})

	t.Run("Error_CannotGetHostByUUID", func(t *testing.T) {
		ctx, cancel := inv_testing.CreateContextWithENJWT(t, tenant1)
		defer cancel()
		inCreate := &pb.NodeRequest{
			Payload: []*pb.NodeData{
				{
					Hwdata: []*pb.HwData{
						{
							MacId:          "90:49:fa:07:6c:fd",
							SutIp:          "10.10.1.1",
							Serialnum:      hostSN,
							Uuid:           "malformed uuid",
							BmcIp:          "10.10.10.10",
							BmcInterface:   true,
							HostNicDevName: "bmc0",
						},
					},
				},
			},
		}
		_, err := OMTestClient.CreateNodes(ctx, inCreate)
		require.Error(t, err)
	})

	t.Run("Success", func(t *testing.T) {
		ctx, cancel := inv_testing.CreateContextWithENJWT(t, tenant1)
		defer cancel()
		bmcIP := "10.10.1.1"
		inCreate := &pb.NodeRequest{
			Payload: []*pb.NodeData{
				{
					Hwdata: []*pb.HwData{
						{
							MacId:          "90:49:fa:07:6c:fd",
							SutIp:          bmcIP,
							Serialnum:      hostSN,
							Uuid:           hostGUID,
							BmcIp:          "10.10.10.10",
							BmcInterface:   true,
							HostNicDevName: "bmc0",
						},
					},
				},
			},
		}
		_, err := OMTestClient.CreateNodes(ctx, inCreate)
		require.NoError(t, err)

		hostInv := GetHostbyUUID(t, hostGUID)
		assert.Equal(t, hostSN, hostInv.GetSerialNumber())
		assert.Equal(t, hostGUID, hostInv.GetUuid())
		assert.Equal(t, bmcIP, hostInv.GetBmcIp())
		assert.Equal(t, "90:49:fa:07:6c:fd", hostInv.GetPxeMac())
		assert.Equal(t, computev1.BaremetalControllerKind_BAREMETAL_CONTROLLER_KIND_PDU, hostInv.GetBmcKind())
	})
}
