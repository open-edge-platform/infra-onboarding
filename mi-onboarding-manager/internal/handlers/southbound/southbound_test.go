// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package southbound_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/api"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/compute/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/errors"
	inv_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/testing"
)

var (
	hostGUID = "BFD3B398-9A4B-480D-AB53-4050ED108F5C"
	hostSN   = "87654321"
)

func TestSouthbound_CreateNodes(t *testing.T) {
	// already exists, don't create
	t.Run("AlreadyExists", func(t *testing.T) {
		ctx := createOutgoingContextWithENJWT(t)

		h1 := inv_testing.CreateHost(t, nil, nil, nil, nil)
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
		ctx, cancel := inv_testing.CreateContextWithJWT(t)
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
		ctx := createOutgoingContextWithENJWT(t)
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

func TestSouthbound_DeleteNodes(t *testing.T) {
	t.Run("Error_CannotGetHostByUUID", func(t *testing.T) {
		ctx, cancel := inv_testing.CreateContextWithJWT(t)
		defer cancel()
		nodeReq := &pb.NodeRequest{
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
		_, err := OMTestClient.DeleteNodes(ctx, nodeReq)
		require.Error(t, err)
	})

	t.Run("NotFound", func(t *testing.T) {
		ctx := createOutgoingContextWithENJWT(t)
		nodeReq := &pb.NodeRequest{
			Payload: []*pb.NodeData{
				{
					Hwdata: []*pb.HwData{
						{
							MacId:          "90:49:fa:07:6c:fd",
							SutIp:          "10.10.1.1",
							Serialnum:      hostSN,
							Uuid:           uuid.NewString(),
							BmcIp:          "10.10.10.10",
							BmcInterface:   true,
							HostNicDevName: "bmc0",
						},
					},
				},
			},
		}
		_, err := OMTestClient.DeleteNodes(ctx, nodeReq)
		require.NoError(t, err)
	})

	t.Run("Success_Delete", func(t *testing.T) {
		ctx := createOutgoingContextWithENJWT(t)

		hostUUID := uuid.NewString()
		nodeReq := &pb.NodeRequest{
			Payload: []*pb.NodeData{
				{
					Hwdata: []*pb.HwData{
						{
							MacId:          "90:49:fa:07:6c:fd",
							SutIp:          "10.10.1.1",
							Serialnum:      hostSN,
							Uuid:           hostUUID,
							BmcIp:          "10.10.10.10",
							BmcInterface:   true,
							HostNicDevName: "bmc0",
						},
					},
				},
			},
		}
		_, err := OMTestClient.CreateNodes(ctx, nodeReq)
		require.NoError(t, err)

		_, err = OMTestClient.DeleteNodes(ctx, nodeReq)
		require.NoError(t, err)

		// get Host by UUID
		// Note that Host resource should be removed by reconciler that is not running in this test case,
		// so we only check if current_state has been updated.
		hostInv := GetHostbyUUID(t, hostUUID)
		assert.Equal(t, hostInv.GetCurrentState(), computev1.HostState_HOST_STATE_DELETED)
	})
}

func TestSouthbound_UpdateNodes(t *testing.T) {
	t.Run("NotFound", func(t *testing.T) {
		ctx := createOutgoingContextWithENJWT(t)

		inUpdate := &pb.NodeRequest{
			Payload: []*pb.NodeData{
				{
					Hwdata: []*pb.HwData{
						{
							MacId:          "90:49:fa:07:6c:fd",
							SutIp:          "10.10.1.1",
							Serialnum:      hostSN,
							Uuid:           uuid.NewString(),
							BmcIp:          "10.10.10.10",
							BmcInterface:   true,
							HostNicDevName: "bmc0",
						},
					},
				},
			},
		}
		_, err := OMTestClient.UpdateNodes(ctx, inUpdate)
		require.Error(t, err)
		assert.True(t, errors.IsNotFound(err))
	})

	t.Run("Error_CannotGetHostByUUID", func(t *testing.T) {
		ctx, cancel := inv_testing.CreateContextWithJWT(t)
		defer cancel()
		inUpdate := &pb.NodeRequest{
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
		_, err := OMTestClient.UpdateNodes(ctx, inUpdate)
		require.Error(t, err)
	})

	t.Run("Success_Update", func(t *testing.T) {
		ctx := createOutgoingContextWithENJWT(t)

		nodeReq := &pb.NodeRequest{
			Payload: []*pb.NodeData{
				{
					Hwdata: []*pb.HwData{
						{
							MacId:          "90:49:fa:07:6c:fd",
							SutIp:          "10.10.1.1",
							Serialnum:      hostSN,
							Uuid:           uuid.NewString(),
							BmcIp:          "10.10.10.10",
							BmcInterface:   true,
							HostNicDevName: "bmc0",
						},
					},
				},
			},
		}
		_, err := OMTestClient.CreateNodes(ctx, nodeReq)
		require.NoError(t, err)

		// do update with new BMC IP and MAC address
		nodeReq.Payload[0].Hwdata[0].BmcIp = "10.10.10.150"
		nodeReq.Payload[0].Hwdata[0].MacId = "aa:bb:cc:dd:ee:ff"

		_, err = OMTestClient.UpdateNodes(ctx, nodeReq)
		require.NoError(t, err)

		// do update again, there should be no request to Inventory now (check logs)
		_, err = OMTestClient.UpdateNodes(ctx, nodeReq)
		require.NoError(t, err)
	})
}
