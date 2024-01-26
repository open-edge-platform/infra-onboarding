// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package artifact

import (
	"context"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/invclient"
	"time"

	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/onboardingmgr"
	repository "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/persistence"
)

var (
	name = "NodeArtifactService"
	zlog = logging.GetLogger(name)
)

const (
	DefaultTimeout = 3 * time.Second
)

var hostResID string
var hostNicResID string

type NodeArtifactService struct {
	pb.UnimplementedNodeArtifactServiceNBServer
	invClient *invclient.OnboardingInventoryClient
}

// TODO: this should be removed and the internal invClient of NodeArtifactService should be used.
//
//	This requires to pass NodeArtifactService instance to SB Handler.
var ginvClient *invclient.OnboardingInventoryClient

// InitNodeArtifactService is a constructor function
func InitNodeArtifactService(invClient *invclient.OnboardingInventoryClient) *NodeArtifactService {
	if invClient == nil {
		zlog.Debug().Msgf("Warning: invClient is nil in InitNodeArtifactService")
		// Return an error or handle the nil case appropriately
		return nil
	}
	ginvClient = invClient

	return &NodeArtifactService{
		invClient: invClient,
	}
}

func CopyNodeReqtoNodetData(payload []*pb.NodeData) ([]computev1.HostResource, error) {

	zlog.Info().Msgf("CopyNodeReqtoNodetData")

	zlog.Debug().Msgf("%d", len(payload))
	var data []computev1.HostResource
	for i, s := range payload {
		hostres := computev1.HostResource{
			BmcKind:      computev1.BaremetalControllerKind_BAREMETAL_CONTROLLER_KIND_PDU,
			BmcIp:        s.Hwdata[i].SutIp,
			SerialNumber: s.Hwdata[i].Serialnum,
			Uuid:         s.Hwdata[i].Uuid,
		}

		/* TODO: Implement multiple NIC resources for each Host Resource/Node
		 * some changes might be required in type HwData struct in protobuf (onboarding.pb.go) file
		 *
		 * TODO: This is just for test purporse only.
		 *       Need to change it either for
		 *       multiple host resource based on the pdctl command input
		 *		 create a Jira ticket and address this before GA release
		 */
		hostnic := computev1.HostnicResource{
			MacAddr:      s.Hwdata[i].MacId,
			DeviceName:   s.Hwdata[i].HostNicDevName,
			BmcInterface: s.Hwdata[i].BmcInterface,
		}
		hostres.HostNics = append(hostres.HostNics, &hostnic)
		zlog.Debug().Msgf("MAC is %s \n", s.Hwdata[i].MacId)
		zlog.Debug().Msgf("sut ip is %s \n", s.Hwdata[i].SutIp)
		zlog.Debug().Msgf("uuid is %s \n", s.Hwdata[i].Uuid)
		zlog.Debug().Msgf("serial num is %s \n", s.Hwdata[i].Serialnum)
		zlog.Debug().Msgf("bmc ip is %s \n", s.Hwdata[i].BmcIp)
		zlog.Debug().Msgf("Host nic dev name is %s \n", s.Hwdata[i].HostNicDevName)
		zlog.Debug().Msgf("bmc interface is %t \n", s.Hwdata[i].BmcInterface)
		data = append(data, hostres)
	}

	zlog.Debug().Msgf("%d", len(data))

	return data, nil
}

func CopyNodeDatatoNodeResp(payload []repository.NodeData, result string) ([]*pb.NodeData, error) {
	zlog.Info().Msg("CopyNodeDatatoNodeResp")
	var data []*pb.NodeData
	for _, s := range payload {
		art2 := pb.NodeData{
			NodeId:          s.ID,
			HwId:            s.HwID,
			FwArtifactId:    s.FwArtID,
			OsArtifactId:    s.OsArtID,
			AppArtifactId:   s.AppArtID,
			PlatArtifactId:  s.PlatformArtID,
			PlatformType:    s.PlatformType,
			DeviceType:      s.DeviceType,
			DeviceInfoAgent: s.DeviceInfoAgent,
			DeviceStatus:    s.DeviceStatus,
		}
		if result == "SUCCESS" {
			art2.Result = 0
		} else {
			art2.Result = 1
		}
		data = append(data, &art2)
	}
	return data, nil
}

func (s *NodeArtifactService) CreateNodes(ctx context.Context, req *pb.NodeRequest) (*pb.NodeResponse, error) {

	zlog.Info().Msgf("CreateNodes")
	var err error

	if ginvClient == nil {
		// Handle the case when ginvClient is nil
		zlog.Debug().Msgf("ginvClient is nil \n")
		return nil, nil
	}

	/* Copy node data from user */
	hostresdata, _ := CopyNodeReqtoNodetData(req.Payload)

	/* Check if any node with the UUID exists already */
	/* TODO: Need to check this hostresdata array for all the serial numbers existence
	 *		 already in the system
	 */
	_, err = ginvClient.GetHostResourceByUUID(ctx, hostresdata[0].Uuid)

	switch {
	case inv_errors.IsNotFound(err):
		zlog.MiSec().MiErr(err).Msgf("Create op : Node Doesn't Exist for GUID %s\n", hostresdata[0].Uuid)

	case err == nil:
		zlog.Debug().Msgf("Create op : Node and its Host Resource Already Exist for GUID %s \n", hostresdata[0].Uuid)
		return &pb.NodeResponse{Payload: req.Payload}, nil

	case err != nil:
		zlog.MiSec().MiErr(err).Msgf("Create op :Failed CreateNodes() for GUID %s\n", hostresdata[0].Uuid)
		return nil, err
	}

	/* TODO: Need to change it either to single host resource creation or
	 *       multiple host resource based on the pdctl command input
	 */

	hostResID, err = ginvClient.CreateHostResource(ctx, &hostresdata[0])
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("CreateNodes() : CreateHostResource() Error : %v\n", err)
	}
	zlog.Debug().Msgf("\nCreateHostResource ID = %s\n", hostResID)

	/* TODO: This is just for test purporse only.
	 *       Need to change it either for
	 *       multiple host resource based on the pdctl command input
	 *		 create a Jira ticket and address this before GA release
	 */
	hostresdata[0].HostNics[0].Host = &hostresdata[0]
	hostNicID, err := ginvClient.CreateHostNICResource(ctx, hostresdata[0].HostNics[0])
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("CreateNodes() : CreateHostnicResource() Error : %v\n", err)
	}
	zlog.Debug().Msgf("\nCreateHostNicResource ID = %s\n", hostNicID)

	// TODO (LPIO-1740): this is not needed, to remove
	hostres, err := ginvClient.GetHostResourceByResourceID(ctx, hostResID)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("\nGetHostResourceByResourceID() Error : %v\n", err)
	}
	zlog.Debug().Msgf("\n GetHostResourceByResourceID in CreateNodes()= %v \n", hostres)

	return &pb.NodeResponse{Payload: req.Payload}, nil
}

func (s *NodeArtifactService) DeleteNodes(ctx context.Context, req *pb.NodeRequest) (*pb.NodeResponse, error) {
	zlog.Info().Msgf("DeleteNodes")
	hostresdata, _ := CopyNodeReqtoNodetData(req.Payload)

	/* TODO: Need to change it either to single host resource creation or
	 *       multiple host resource based on the pdctl command input
	 */
	/* Check if any node with the serial num exists or not */
	hostresget, err := ginvClient.GetHostResourceByUUID(ctx, hostresdata[0].Uuid)

	switch {
	case inv_errors.IsNotFound(err):
		zlog.MiSec().MiErr(err).Msgf("Delete op : Node Doesn't Exist for GUID %s\n", hostresdata[0].Uuid)
		return nil, nil

	case err == nil:
		zlog.Debug().Msgf("Delete op : Node and its Host Resource Already Exist for GUID %s \n", hostresdata[0].Uuid)

	case err != nil:
		zlog.MiSec().MiErr(err).Msgf("Delete op : Failed DeleteNodes() for GUID %s\n", hostresdata[0].Uuid)
		return nil, err
	}

	/* copy and update the fetched resource id of the given serial number
	 *  to the pre-existing host resource
	 */
	hostResID = hostresget.ResourceId
	hostresdata[0].ResourceId = hostResID

	err = ginvClient.DeleteHostResource(ctx, hostResID)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("\nDeleteHostResource() Error : %v\n", err)
	}

	// TODO (LPIO-1740): this is not needed, to remove
	hostres, err := ginvClient.GetHostResourceByResourceID(ctx, hostResID)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("\nGetHostResourceByResourceID() Error : %v\n", err)
	}
	zlog.Debug().Msgf("\n GetHostResourceByResourceID in DeleteNodes()= %v \n", hostres)

	return &pb.NodeResponse{Payload: req.Payload}, nil
}

func (s *NodeArtifactService) GetNodes(ctx context.Context, req *pb.NodeRequest) (*pb.NodeResponse, error) {
	zlog.Info().Msgf("GetNodes")

	guid := req.Payload[0].Hwdata[0].Uuid

	/* Check if any node with the serial num exists or not */
	hostresget, err := ginvClient.GetHostResourceByUUID(ctx, guid)

	switch {
	case inv_errors.IsNotFound(err):
		zlog.MiSec().MiErr(err).Msgf("Get op : Node Doesn't Exist for GUID %s\n", guid)
		return nil, nil

	case err == nil:
		zlog.Debug().Msgf("Get op : Node and its Host Resource Already Exist for GUID %s \n", guid)

	case err != nil:
		zlog.MiSec().MiErr(err).Msgf("Get op : Failed CreateNodes() for GUID %s\n", guid)
		return nil, err
	}

	//Copy the fetched resource id of the given serial number
	hostResID = hostresget.ResourceId

	hostres, err := ginvClient.GetHostResourceByResourceID(ctx, hostResID)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("\nGetNodes() : GetHostResourceByResourceID() Error : %v\n", err)
		return nil, err
	}
	zlog.Debug().Msgf("\n HostResource by GetNodes() = %v \n", hostres)

	if len(hostres.HostNics) == 0 {
		zlog.Info().Msgf("GetNodes() : Slice is empty \n")
	}
	return &pb.NodeResponse{Payload: req.Payload}, nil
}

func (s *NodeArtifactService) UpdateNodes(ctx context.Context, req *pb.NodeRequest) (*pb.NodeResponse, error) {

	zlog.Info().Msgf("UpdateNodes")

	hostresdata, _ := CopyNodeReqtoNodetData(req.Payload)

	/* TODO: Need to change it either to single host resource creation or
	 *       multiple host resource based on the pdctl command input
	 */
	/* Check if any node with the serial num exists already */
	hostresget, err := ginvClient.GetHostResourceByUUID(ctx, hostresdata[0].Uuid)

	switch {
	case inv_errors.IsNotFound(err):
		zlog.MiSec().MiErr(err).Msgf("Update op : Node Doesn't Exist for GUID %s\n", hostresdata[0].Uuid)
		return nil, err

	case err == nil:
		zlog.Debug().Msgf("Update op : Node and its Host Resource Already Exist for GUID %s \n", hostresdata[0].Uuid)

	case err != nil:
		zlog.MiSec().MiErr(err).Msgf("Update op : Failed CreateNodes() for GUID %s\n", hostresdata[0].Uuid)
		return nil, err
	}

	//update the fetched resource id of the given serial number
	hostResID = hostresget.ResourceId
	hostresdata[0].ResourceId = hostResID
	hostNicResID = hostresget.HostNics[0].ResourceId
	hostresdata[0].HostNics[0].ResourceId = hostNicResID

	zlog.Debug().Msgf("hostResID is %s\n", hostResID)
	zlog.Debug().Msgf("hostNicResID is %s\n", hostNicResID)

	hostres, err := ginvClient.GetHostResourceByResourceID(ctx, hostResID)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("UpdateNodes() : GetHostResourceByResourceID() Error : %v\n", err)
		return nil, err
	}
	zlog.Debug().Msgf("GetHostResource ID in UpdateNodes = %v \n", hostres)

	// TODO (LPIO-1740): we should check if Host Resource has changed. Otherwise, skip to limit load on Inventory
	err = ginvClient.UpdateHostResource(ctx, &hostresdata[0])
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("UpdateNodes() : UpdateHostResource() Error : %v\n", err)
		return nil, err
	}
	/* TODO: Add other parameters in future
	 *       Move this to a function
	 */
	nic := &computev1.HostnicResource{
		Host:         &hostresdata[0],
		MacAddr:      hostresdata[0].HostNics[0].MacAddr,
		PeerMgmtIp:   hostresdata[0].HostNics[0].PeerMgmtIp,
		DeviceName:   hostresdata[0].HostNics[0].DeviceName,
		BmcInterface: hostresdata[0].HostNics[0].BmcInterface,
	}
	nic.ResourceId = hostresdata[0].HostNics[0].ResourceId
	// TODO: (LPIO-1740): check if Host NIC has changed. Otherwise, skip to limit load on Inventory
	err = ginvClient.UpdateHostNIC(ctx, nic)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("UpdateNodes() : UpdateHostnic() Error : %v\n", err)
		return nil, err
	}

	// TODO (LPIO-1740): this is not needed, to remove
	hostres, err = ginvClient.GetHostResourceByResourceID(ctx, hostResID)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("UpdateNodes() :GetHostResourceByResourceID() Error : %v\n", err)
		return nil, err
	}

	zlog.Debug().Msgf("GetHostResource ID in UpdateNodes after updating = %v \n", hostres)

	return &pb.NodeResponse{Payload: req.Payload}, nil
}
