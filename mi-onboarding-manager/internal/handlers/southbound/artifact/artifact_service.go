// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package artifact

import (
	"context"
	"encoding/json"
	"time"

	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/util"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/api"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inventoryv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/os/v1"
	provider_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/provider/v1"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
)

const (
	DefaultTimeout      = 3 * time.Second
	DefaultProviderName = "fm_onboarding"
)

var (
	name = "NodeArtifactService"
	zlog = logging.GetLogger(name)

	hostResID string

	HostFieldmask = &fieldmaskpb.FieldMask{
		Paths: []string{
			computev1.HostResourceFieldBmcKind,
			computev1.HostResourceFieldBmcIp,
			computev1.HostResourceFieldSerialNumber,
			computev1.HostResourceFieldUuid,
			computev1.HostResourceFieldPxeMac,
		},
	}
)

type (
	//nolint:tagliatelle // Renaming the json keys may effect while unmarshalling/marshaling so, used nolint.
	ProviderConfig struct {
		// the Resource ID of the OS profile
		DefaultOs     string `json:"defaultOs"`
		AutoProvision bool   `json:"autoProvision"`
	}

	NodeArtifactService struct {
		pb.UnimplementedNodeArtifactServiceNBServer
		invClient *invclient.OnboardingInventoryClient
		// TODO: remove this later https://jira.devtools.intel.com/browse/LPIO-1829
		invClientAPI *invclient.OnboardingInventoryClient
	}
)

// NewArtifactService is a constructor function.
func NewArtifactService(invClient *invclient.OnboardingInventoryClient, inventoryAdr string, enableTracing bool,
) (*NodeArtifactService, error) {
	if invClient == nil {
		return nil, inv_errors.Errorf("invClient is nil in NewArtifactService")
	}

	var (
		invClientAPI *invclient.OnboardingInventoryClient
		err          error
	)
	if inventoryAdr == "" {
		zlog.Warn().Msg("Unable to start onboarding inventory API server client, empty inventory address")
	} else {
		// TODO: remove this later https://jira.devtools.intel.com/browse/LPIO-1829
		invClientAPI, err = invclient.NewOnboardingInventoryClientWithOptions(
			invclient.WithInventoryAddress(inventoryAdr),
			invclient.WithEnableTracing(enableTracing),
			invclient.WithClientKind(inventoryv1.ClientKind_CLIENT_KIND_API),
		)
		if err != nil {
			return nil, inv_errors.Errorf("Unable to start onboarding inventory API server client %v", err)
		}
	}

	return &NodeArtifactService{
		invClient:    invClient,
		invClientAPI: invClientAPI,
	}, nil
}

func CopyNodeReqToNodeData(payload []*pb.NodeData) ([]*computev1.HostResource, error) {
	zlog.Info().Msgf("CopyNodeReqToNodeData")

	zlog.Debug().Msgf("Parsing NodeData of length=%d", len(payload))
	hosts := make([]*computev1.HostResource, 0)
	for _, s := range payload {
		for _, hwData := range s.Hwdata {
			hostres := &computev1.HostResource{
				BmcKind:      computev1.BaremetalControllerKind_BAREMETAL_CONTROLLER_KIND_PDU,
				BmcIp:        hwData.SutIp,
				SerialNumber: hwData.Serialnum,
				Uuid:         hwData.Uuid,
				PxeMac:       hwData.MacId,
			}
			zlog.Debug().Msgf("Adding HostResource: %v", hostres)
			hosts = append(hosts, hostres)
		}
	}

	zlog.Debug().Msgf("Generates a list of hosts of length=%d", len(hosts))

	return hosts, nil
}

func (s *NodeArtifactService) CreateNodes(ctx context.Context, req *pb.NodeRequest) (*pb.NodeResponse, error) {
	zlog.Info().Msgf("CreateNodes")

	/* Copy node data from user */
	hostresdata, err := CopyNodeReqToNodeData(req.Payload)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("CopyNodeReqToNodeData error: %v", err)
		return nil, err
	}
	// TODO: CopyNodeReqToNodeData currently returns a list of Host resources with just a single element.
	//  We should change it to either multiple Host resources returned or return a single resource.
	host := hostresdata[0]
	/* Check if any node with the UUID exists already */
	/* TODO: Need to check this hostresdata array for all the serial numbers existence
	 *		 already in the system
	 */
	_, err = s.invClient.GetHostResourceByUUID(ctx, host.Uuid)

	switch {
	case inv_errors.IsNotFound(err):
		zlog.MiSec().MiErr(err).Msgf("Create op : Node Doesn't Exist for GUID %s\n", host.Uuid)

	case err == nil:
		zlog.Debug().Msgf("Create op : Node and its Host Resource Already Exist for GUID %s \n", host.Uuid)
		if ztErr := s.startZeroTouch(ctx, host.ResourceId); ztErr != nil {
			zlog.MiSec().MiErr(ztErr).Msgf("startZeroTouch error: %v", ztErr)
			return nil, ztErr
		}
		return &pb.NodeResponse{Payload: req.Payload}, nil

	case err != nil:
		zlog.MiSec().MiErr(err).Msgf("Create op :Failed CreateNodes() for GUID %s\n", host.Uuid)
		return nil, err
	}

	hostResID, err = s.invClient.CreateHostResource(ctx, host)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("Cannot create Host resource: %v", host)
		return nil, err
	}
	zlog.Debug().Msgf("CreateHostResource ID = %s", hostResID)

	if err := s.startZeroTouch(ctx, hostResID); err != nil {
		zlog.MiSec().MiErr(err).Msgf("startZeroTouch error: %v", err)
		return nil, err
	}

	return &pb.NodeResponse{Payload: req.Payload}, nil
}

func (s *NodeArtifactService) DeleteNodes(ctx context.Context, req *pb.NodeRequest) (*pb.NodeResponse, error) {
	zlog.Info().Msgf("DeleteNodes")
	hostresdata, err := CopyNodeReqToNodeData(req.Payload)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("CopyNodeReqToNodeData error: %v", err)
		return nil, err
	}
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
	var tempErr error
	switch {
	case inv_errors.IsNotFound(err):
		zlog.MiSec().MiErr(err).Msgf("Get op : Node Doesn't Exist for GUID %s\n", guid)
		return nil, tempErr

	case err == nil:
		zlog.Debug().Msgf("Get op : Node and its Host Resource Already Exist for GUID %s \n", guid)

	case err != nil:
		zlog.MiSec().MiErr(err).Msgf("Get op : Failed CreateNodes() for GUID %s\n", guid)
		return nil, err
	}

	zlog.Debug().Msgf("HostResource by GetNodes() = %v", hostresget)

	return &pb.NodeResponse{Payload: req.Payload}, nil
}

func (s *NodeArtifactService) UpdateNodes(ctx context.Context, req *pb.NodeRequest) (*pb.NodeResponse, error) {
	zlog.Info().Msgf("UpdateNodes")

	host, err := CopyNodeReqToNodeData(req.Payload)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("CopyNodeReqToNodeData error: %v", err)
		return nil, err
	}

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
		host[0].ResourceId = hostInv.GetResourceId()
		err = s.invClient.UpdateInvResourceFields(ctx, host[0], HostFieldmask.Paths)
		if err != nil {
			zlog.MiSec().MiErr(err).Msgf("UpdateNodes() : UpdateHostResource() Error : %v", err)
			return nil, err
		}
	} else {
		zlog.Debug().Msgf("Skipping to update Host resource due to no changes. "+
			"Original Host: %v, Updated Host: %v", hostInv, host[0])
	}

	return &pb.NodeResponse{Payload: req.Payload}, nil
}

func (s *NodeArtifactService) startZeroTouch(ctx context.Context, hostResID string) error {
	zlog.Info().Msgf("Starting zero touch for host ID %s...", hostResID)

	host, err := s.invClient.GetHostResourceByResourceID(ctx, hostResID)
	if err != nil {
		zlog.Err(err).Msgf("Skipping, no host resource found with (uuid=%s)", hostResID)
		return nil
	}

	// an Instance has been created for the Host, skip
	if host.Instance != nil {
		zlog.Debug().Msgf("An Instance (%s) is already created for a host %s",
			host.GetInstance().GetResourceId(), host.GetResourceId())
		return nil
	}

	providers, err := s.invClient.GetProviderResources(ctx)
	if err != nil {
		zlog.Err(err).Msgf("Skipping, no provider resource found")
		return nil
	}

	var provider *provider_v1.ProviderResource
	for _, p := range providers {
		if DefaultProviderName == p.Name {
			provider = p
			break
		}
	}

	// check automatic provisioning setting
	if provider == nil || provider.Config == "" {
		zlog.Info().Msg("Skipping, no default provider resource found")
		return nil
	}

	var pconf ProviderConfig
	if err := json.Unmarshal([]byte(provider.Config), &pconf); err != nil {
		zlog.Err(err).Msgf("Failed to unmarshal ProviderConfig for host resource (uuid=%s)", hostResID)
		return err
	}

	// if AutoProvision is set, create an Instance for the Host with the OS set to the value of the default OS
	return s.checkNCreateInstance(ctx, pconf, host)
}

func (s *NodeArtifactService) checkNCreateInstance(ctx context.Context, pconf ProviderConfig, host *computev1.HostResource,
) error {
	if pconf.AutoProvision {
		instance := &computev1.InstanceResource{
			Kind:         computev1.InstanceKind_INSTANCE_KIND_METAL,
			DesiredState: computev1.InstanceState_INSTANCE_STATE_RUNNING,
			CurrentState: computev1.InstanceState_INSTANCE_STATE_UNSPECIFIED,
			Host: &computev1.HostResource{
				ResourceId: host.ResourceId,
			},
			Os: &osv1.OperatingSystemResource{
				ResourceId: pconf.DefaultOs,
			},
			SecurityFeature: osv1.SecurityFeature_SECURITY_FEATURE_SECURE_BOOT_AND_FULL_DISK_ENCRYPTION,
		}
		if _, err := s.invClientAPI.CreateInstanceResource(ctx, instance); err != nil {
			zlog.Err(err).Msgf("Failed to CreateInstanceResource for host resource (uuid=%s)", hostResID)
			return err
		}
	}

	return nil
}
