// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package artifact

import (
	"context"
	"time"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/invclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/onboardingmgr"
	repository "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/persistence"
)

var (
	name      = "NodeArtifactService"
	zlog      = logging.GetLogger(name)
	hostResID string
)

const (
	DefaultTimeout = 3 * time.Second
)

var HostFieldmask = &fieldmaskpb.FieldMask{
	Paths: []string{
		computev1.HostResourceFieldBmcKind,
		computev1.HostResourceFieldBmcIp,
		computev1.HostResourceFieldSerialNumber,
		computev1.HostResourceFieldUuid,
	},
}

var HostnicFieldmask = &fieldmaskpb.FieldMask{
	Paths: []string{
		computev1.HostnicResourceFieldDeviceName,
		computev1.HostnicResourceEdgeHost,
		computev1.HostnicResourceFieldMacAddr,
		computev1.HostnicResourceFieldBmcInterface,
	},
}

type NodeArtifactService struct {
	pb.UnimplementedNodeArtifactServiceNBServer
	invClient *invclient.OnboardingInventoryClient
}

// NewArtifactService is a constructor function.
func NewArtifactService(invClient *invclient.OnboardingInventoryClient) (*NodeArtifactService, error) {
	if invClient == nil {
		return nil, inv_errors.Errorf("invClient is nil in NewArtifactService")
	}

	return &NodeArtifactService{
		invClient: invClient,
	}, nil
}

func CopyNodeReqtoNodetData(payload []*pb.NodeData) ([]*computev1.HostResource, []*computev1.HostnicResource, error) {
	zlog.Info().Msgf("CopyNodeReqtoNodetData")

	zlog.Debug().Msgf("%d", len(payload))
	hosts := make([]*computev1.HostResource, 0)
	hostNics := make([]*computev1.HostnicResource, 0)
	for _, s := range payload {
		for _, hwData := range s.Hwdata {
			hostres := &computev1.HostResource{
				BmcKind:      computev1.BaremetalControllerKind_BAREMETAL_CONTROLLER_KIND_PDU,
				BmcIp:        hwData.SutIp,
				SerialNumber: hwData.Serialnum,
				Uuid:         hwData.Uuid,
			}

			/* TODO: Implement multiple NIC resources for each Host Resource/Node
			 * some changes might be required in type HwData struct in protobuf (onboarding.pb.go) file
			 *
			 * TODO: This is just for test purporse only.
			 *       Need to change it either for
			 *       multiple host resource based on the pdctl command input
			 *		 create a Jira ticket and address this before GA release
			 */
			hostnic := &computev1.HostnicResource{
				Host:         hostres,
				MacAddr:      hwData.MacId,
				DeviceName:   hwData.HostNicDevName,
				BmcInterface: true,
			}

			zlog.Debug().Msgf("MAC is %s \n", hwData.MacId)
			zlog.Debug().Msgf("sut ip is %s \n", hwData.SutIp)
			zlog.Debug().Msgf("uuid is %s \n", hwData.Uuid)
			zlog.Debug().Msgf("serial num is %s \n", hwData.Serialnum)
			zlog.Debug().Msgf("bmc ip is %s \n", hwData.BmcIp)
			zlog.Debug().Msgf("Host nic dev name is %s \n", hwData.HostNicDevName)
			zlog.Debug().Msgf("bmc interface is %t \n", hwData.BmcInterface)
			hosts = append(hosts, hostres)
			hostNics = append(hostNics, hostnic)
		}
	}

	zlog.Debug().Msgf("%d", len(hosts))

	return hosts, hostNics, nil
}

func CopyNodeDatatoNodeResp(payload []repository.NodeData, result string) ([]*pb.NodeData, error) {
	zlog.Info().Msg("CopyNodeDatatoNodeResp")
	data := make([]*pb.NodeData, 0)
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

	/* Copy node data from user */
	hostresdata, hostNics, _ := CopyNodeReqtoNodetData(req.Payload)
	// TODO: CopyNodeReqtoNodetData currently returns a list of Host/Hostnic resources with just a single element.
	//  We should change it to either multiple Host/Hostnic resources returned or return a single resource.
	host := hostresdata[0]
	hostNic := hostNics[0]
	/* Check if any node with the UUID exists already */
	/* TODO: Need to check this hostresdata array for all the serial numbers existence
	 *		 already in the system
	 */
	_, err := s.invClient.GetHostResourceByUUID(ctx, hostresdata[0].Uuid)

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
	hostResID, err = s.invClient.CreateHostResource(ctx, host)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("Cannot create Host resource: %v", host)
		return nil, err
	}
	zlog.Debug().Msgf("CreateHostResource ID = %s", hostResID)

	/* TODO: This is just for test purporse only.
	 *       Need to change it either for
	 *       multiple host resource based on the pdctl command input
	 *		 create a Jira ticket and address this before GA release
	 */
	hostNic.Host.ResourceId = hostResID
	hostNicID, err := s.invClient.CreateHostNICResource(ctx, hostNic)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("Cannot create Hostnic resource: %v", hostNic)
		return nil, err
	}
	zlog.Debug().Msgf("CreateHostNicResource ID = %s", hostNicID)

	return &pb.NodeResponse{Payload: req.Payload}, nil
}

func (s *NodeArtifactService) DeleteNodes(ctx context.Context, req *pb.NodeRequest) (*pb.NodeResponse, error) {
	zlog.Info().Msgf("DeleteNodes")
	hostresdata, _, _ := CopyNodeReqtoNodetData(req.Payload)

	/* TODO: Need to change it either to single host resource creation or
	 *       multiple host resource based on the pdctl command input
	 */
	/* Check if any node with the serial num exists or not */
	hostresget, err := s.invClient.GetHostResourceByUUID(ctx, hostresdata[0].Uuid)

	switch {
	case inv_errors.IsNotFound(err):
		zlog.MiSec().MiErr(err).Msgf("Delete op : Node Doesn't Exist for GUID %s\n", hostresdata[0].Uuid)
		return &pb.NodeResponse{Payload: req.Payload}, nil

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

	err = s.invClient.DeleteHostResource(ctx, hostResID)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("\nDeleteHostResource() Error : %v\n", err)
		return nil, err
	}

	return &pb.NodeResponse{Payload: req.Payload}, nil
}

func (s *NodeArtifactService) GetNodes(ctx context.Context, req *pb.NodeRequest) (*pb.NodeResponse, error) {
	zlog.Info().Msgf("GetNodes")

	guid := req.Payload[0].Hwdata[0].Uuid

	/* Check if any node with the serial num exists or not */
	hostresget, err := s.invClient.GetHostResourceByUUID(ctx, guid)

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

	zlog.Debug().Msgf("\n HostResource by GetNodes() = %v \n", hostresget)

	if len(hostresget.HostNics) == 0 {
		zlog.Info().Msgf("GetNodes() : Slice is empty \n")
	}
	return &pb.NodeResponse{Payload: req.Payload}, nil
}

func (s *NodeArtifactService) UpdateNodes(ctx context.Context, req *pb.NodeRequest) (*pb.NodeResponse, error) {
	zlog.Info().Msgf("UpdateNodes")

	host, bmcNics, _ := CopyNodeReqtoNodetData(req.Payload)

	/* TODO: Need to change it either to single host resource creation or
	 *       multiple host resource based on the pdctl command input
	 */
	hostInv, err := s.invClient.GetHostResourceByUUID(ctx, host[0].Uuid)
	switch {
	case inv_errors.IsNotFound(err):
		zlog.MiSec().MiErr(err).Msgf("Update op : Node Doesn't Exist for GUID %s\n", host[0].Uuid)
		return nil, err

	case err == nil:
		zlog.Debug().Msgf("Update op : Node and its Host Resource Already Exist for GUID %s \n", host[0].Uuid)

	case err != nil:
		zlog.MiSec().MiErr(err).Msgf("Update op : Failed CreateNodes() for GUID %s\n", host[0].Uuid)
		return nil, err
	}

	doHostUpdate := false
	isSameHost, err := util.IsSameHost(hostInv, host[0], HostFieldmask)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("Failed to compare Host resources, continuing to do update anyway")
		doHostUpdate = true
	}

	if !isSameHost || doHostUpdate {
		err = s.invClient.UpdateHostResource(ctx, host[0])
		if err != nil {
			zlog.MiSec().MiErr(err).Msgf("UpdateNodes() : UpdateHostResource() Error : %v", err)
			return nil, err
		}
	} else {
		zlog.Debug().Msgf("Skipping to update Host resource due to no changes. "+
			"Original Host: %v, Updated Host: %v", hostInv, host[0])
	}

	bmcNicsInv, err := util.GetBmcNicsFromHost(hostInv)
	if err != nil {
		zlog.MiSec().MiErr(err).Msg("Cannot get BMC interfaces from Host resource. Update won't be done.")
		return &pb.NodeResponse{Payload: req.Payload}, nil
	}

	// current assumption is that there is always one BMC Hostnic resource
	// FIXME: we should revisit this assumption in future
	if len(bmcNicsInv) != 1 || len(bmcNics) != 1 {
		zlog.MiSec().MiErr(err).Msgf(
			"Cannot update BMC interfaces as the number of BMC interfaces associated with Host doesn't equal one. "+
				"Inventory BMC Nics: %v, Updated BMC Nics: %v", bmcNicsInv, bmcNics)
		return nil, inv_errors.Errorfc(codes.InvalidArgument, "Exactly one BMC interface should be provided")
	}
	originalBmcNic := bmcNicsInv[0]
	updatedBmcNic := bmcNics[0]

	doHostnicUpdate := false
	isSameHostnic, err := util.IsSameHostnic(originalBmcNic, updatedBmcNic, HostnicFieldmask)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("Failed to compare Hostnic resources, continuing to do update anyway")
		doHostnicUpdate = true
	}

	if !isSameHostnic || doHostnicUpdate {
		updatedBmcNic.ResourceId = originalBmcNic.ResourceId
		updatedBmcNic.Host.ResourceId = hostInv.ResourceId
		err = s.invClient.UpdateHostNIC(ctx, updatedBmcNic)
		if err != nil {
			zlog.MiSec().MiErr(err).Msgf("UpdateNodes() : UpdateHostnic() Error : %v", err)
			return nil, err
		}
	} else {
		zlog.Debug().Msgf("Skipping to update Hostnic resource due to no changes. "+
			"Original Hostnic: %v, Updated Hostnic: %v", originalBmcNic, updatedBmcNic)
	}

	return &pb.NodeResponse{Payload: req.Payload}, nil
}
