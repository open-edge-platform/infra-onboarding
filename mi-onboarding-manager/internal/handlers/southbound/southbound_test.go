// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package southbound_test

import (
	"context"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/testing"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/onboardingmgr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
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
