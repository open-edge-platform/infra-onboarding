// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package southbound_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	inv_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/testing"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/onboardingmgr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	hostGUID = "BFD3B398-9A4B-480D-AB53-4050ED108F5C"
	hostSN   = "87654321"
)

func TestSouthbound_CreateNodes(t *testing.T) {
	// already exists, don't create
	t.Run("AlreadyExists", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
		assert.Equal(t, computev1.BaremetalControllerKind_BAREMETAL_CONTROLLER_KIND_PDU, hostInv.GetBmcKind())
		require.Len(t, hostInv.GetHostNics(), 1)
	})
}

func TestSouthbound_DeleteNodes(t *testing.T) {
	t.Run("Error_CannotGetHostByUUID", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

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
		// should update both Host and Hostnic
		_, err = OMTestClient.UpdateNodes(ctx, nodeReq)
		require.NoError(t, err)

		// do update again, there should be no request to Inventory now (check logs)
		_, err = OMTestClient.UpdateNodes(ctx, nodeReq)
		require.NoError(t, err)

		// change MAC address again, only Hostnic should be updated
		nodeReq.Payload[0].Hwdata[0].MacId = "aa:bb:cc:dd:ee:gg"
		_, err = OMTestClient.UpdateNodes(ctx, nodeReq)
		require.NoError(t, err)
	})

	t.Run("Error_MoreThanOneBmcNic", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

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

		nodeReq.Payload[0].Hwdata = append(nodeReq.Payload[0].Hwdata, &pb.HwData{
			MacId:          "90:49:fa:07:6c:ff",
			SutIp:          "10.10.1.20",
			Serialnum:      hostSN,
			Uuid:           uuid.NewString(),
			BmcIp:          "10.10.10.20",
			BmcInterface:   true,
			HostNicDevName: "bmc1",
		})

		_, err = OMTestClient.UpdateNodes(ctx, nodeReq)
		require.Error(t, err)
	})
}
