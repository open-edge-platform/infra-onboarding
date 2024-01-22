/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/
package maestro

import (
	"context"
	"sync"
	"time"

	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/os/v1"
	inv_client "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/client"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

var (
	clientName = "OnboardingInventoryClient"
	zlog       = logging.GetLogger(clientName)
)

const (
	DefaultTimeout = 3 * time.Second
)

// List resources by the provided filter. Filter is done only on fields that are set (not default values of the
// resources). Note that this function will NOT return an error if an object is not found.
func ListAllResources(
	ctx context.Context,
	c inv_client.InventoryClient,
	filter *inv_v1.ResourceFilter,
) ([]*inv_v1.Resource, error) {
	zlog.Debug().Msgf("listAllResources")
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()
	// we agreed to not return a NotFound error to avoid too many 'Not Found'
	// responses to the consumer of our external APIs.
	objs, err := c.ListAll(ctx, filter.GetResource(), filter.GetFieldMask())
	if err != nil && !inv_errors.IsNotFound(err) {
		zlog.MiSec().MiErr(err).Msgf("Unable to listAll %v", filter)
		return nil, err
	}
	for _, v := range objs {
		if err = v.ValidateAll(); err != nil {
			zlog.MiSec().MiErr(err).Msgf("Invalid input, validation has failed: %v", v)
			return nil, inv_errors.Wrap(err)
		}
	}
	return objs, nil
}

func NewInventoryClient(wg *sync.WaitGroup, addr string) (inv_client.InventoryClient, chan *inv_client.WatchEvents, error) {
	ctx := context.Background()
	resourceKinds := []inv_v1.ResourceKind{
		inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE,
		inv_v1.ResourceKind_RESOURCE_KIND_HOST,
		inv_v1.ResourceKind_RESOURCE_KIND_OS,
	}
	eventCh := make(chan *inv_client.WatchEvents)

	cfg := inv_client.InventoryClientConfig{
		Name:                      clientName,
		Address:                   addr,
		Events:                    eventCh,
		EnableRegisterRetry:       false,
		AbortOnUnknownClientError: true,
		ClientKind:                inv_v1.ClientKind_CLIENT_KIND_RESOURCE_MANAGER,
		ResourceKinds:             resourceKinds,
		EnableTracing:             true,
		Wg:                        wg,
		SecurityCfg: &inv_client.SecurityConfig{
			Insecure: true,
		},
	}

	invClient, err := inv_client.NewInventoryClient(ctx, cfg)
	if err != nil {
		return nil, eventCh, err
	}

	zlog.MiSec().Info().Msgf("Inventory client started")
	return invClient, eventCh, nil
}

func CreateHostResource(ctx context.Context, c inv_client.InventoryClient, uuid string, hostres *computev1.HostResource) (string, error) {
	hostres.Uuid = uuid
	resreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: hostres,
		},
	}
	res, err := c.Create(ctx, resreq)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("Failed create host resource with %v \n", hostres)
		return "", err
	}
	zlog.Info().Msgf("New Host ID : %s \n", res.ResourceId)
	hostres.ResourceId = res.ResourceId

	return res.ResourceId, nil
}

func CreateHostnicResource(ctx context.Context, c inv_client.InventoryClient, HostResourceId string, hostNicres *computev1.HostnicResource) (string, error) {

	zlog.Info().Msgf("Create Hostnic Resource is %v\n", hostNicres)

	hostres, err := GetHostResourceByResourceID(ctx, c, HostResourceId)
	zlog.Debug().Msgf("Hostresource read is %v\n", hostres)

	hostNicres.Host = hostres
	resreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Hostnic{
			Hostnic: hostNicres,
		},
	}

	res, err := c.Create(ctx, resreq)
	if err != nil {
		return "", err
	}
	return res.ResourceId, nil
}

func CreateInstanceResource(ctx context.Context, c inv_client.InventoryClient, inst *computev1.InstanceResource, hostID, osID string) (string, error) {

	// Set the host ID in the instance resource's Host field
	inst.Host = &computev1.HostResource{

		ResourceId: hostID,
	}
	inst.Os = &osv1.OperatingSystemResource{

		ResourceId: osID,
	}
	resreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Instance{
			Instance: inst,
		},
	}

	res, err := c.Create(ctx, resreq)
	if err != nil {
		return "", err
	}
	return res.ResourceId, nil
}

func GetInstanceResources(ctx context.Context, c inv_client.InventoryClient) (hostres []*computev1.InstanceResource, err error) {
	filter, err := util.GetFilterFromSetResource(&inv_v1.Resource{
		Resource: &inv_v1.Resource_Instance{
			Instance: &computev1.InstanceResource{},
		},
	})
	if err != nil {
		return nil, err
	}
	resources, err := ListAllResources(ctx, c, filter)
	if err != nil {
		return nil, err
	}
	return util.GetSpecificResourceList[*computev1.InstanceResource](resources)
}

func GetHostResources(ctx context.Context, c inv_client.InventoryClient) (hostres []*computev1.HostResource, err error) {
	filter, err := util.GetFilterFromSetResource(&inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: &computev1.HostResource{},
		},
	})
	if err != nil {
		return nil, err
	}
	resources, err := ListAllResources(ctx, c, filter)
	if err != nil {
		return nil, err
	}
	return util.GetSpecificResourceList[*computev1.HostResource](resources)
}

func GetHostResourceByResourceID(ctx context.Context, c inv_client.InventoryClient, resourceID string) (*computev1.HostResource, error) {
	getresresp, err := c.Get(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	host := getresresp.GetResource().GetHost()

	if validateErr := host.ValidateAll(); validateErr != nil {
		return nil, inv_errors.Wrap(validateErr)
	}

	return host, nil
}

func GetInstanceResourceByResourceID(ctx context.Context, c inv_client.InventoryClient, resourceID string) (*computev1.InstanceResource, error) {
	getresresp, err := c.Get(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	inst := getresresp.GetResource().GetInstance()

	if validateErr := inst.ValidateAll(); validateErr != nil {
		return nil, inv_errors.Wrap(validateErr)
	}

	return inst, nil
}

func GetHostResourceByUUID(
	ctx context.Context,
	c inv_client.InventoryClient,
	uuid string,
) (*computev1.HostResource, error) {
	if uuid == "" {
		err := inv_errors.Errorfc(codes.InvalidArgument, "empty UUID")
		return nil, err
	}

	filter, err := util.GetFilterFromSetResource(&inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: &computev1.HostResource{
				Uuid: uuid,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return listOneHost(ctx, c, filter)
}

func DeleteHostResourceByResourceID(ctx context.Context, c inv_client.InventoryClient, resourceID string) error {
	zlog.Info().Msgf("Delete Hostusb: %v", resourceID)

	_, err := c.Delete(ctx, resourceID)
	if inv_errors.IsNotFound(err) {
		zlog.MiSec().MiErr(err).Msgf("Not found while HostResource delete, dropping err: resourceID=%s", resourceID)
		return nil
	}
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("Failed delete Host resource %s", resourceID)
		return err
	}

	return err
}

func DeleteHostResource(ctx context.Context, c inv_client.InventoryClient, host *computev1.HostResource) error {
	h := &computev1.HostResource{
		ResourceId:   host.GetResourceId(),
		CurrentState: computev1.HostState_HOST_STATE_DELETED,
	}

	err := UpdateHostCurrentState(ctx, c, h)
	if err != nil {
		return err
	}

	return nil
}

func DeleteInstanceResource(ctx context.Context, c inv_client.InventoryClient, inst *computev1.InstanceResource) error {
	h := &computev1.InstanceResource{
		ResourceId:   inst.GetResourceId(),
		CurrentState: computev1.InstanceState_INSTANCE_STATE_DELETED,
	}

	err := UpdateInstanceCurrentState(ctx, c, h)
	if err != nil {
		return err
	}

	return nil
}

func UpdateHostResource(ctx context.Context, c inv_client.InventoryClient, host *computev1.HostResource) error {

	return UpdateInvResourceFields(ctx, c, host, []string{
		computev1.HostResourceFieldKind,
		computev1.HostResourceFieldCurrentState,
		computev1.HostResourceFieldHardwareKind,
		computev1.HostResourceFieldMemoryBytes,
		computev1.HostResourceFieldCpuModel,
		computev1.HostResourceFieldCpuSockets,
		computev1.HostResourceFieldCpuCores,
		computev1.HostResourceFieldCpuCapabilities,
		computev1.HostResourceFieldCpuArchitecture,
		computev1.HostResourceFieldCpuThreads,
		computev1.HostResourceFieldMgmtIp,
		computev1.HostResourceFieldBmcKind,
		computev1.HostResourceFieldBmcIp,
		computev1.HostResourceFieldBmcUsername,
		computev1.HostResourceFieldBmcPassword,
		computev1.HostResourceFieldPxeMac,
		computev1.HostResourceFieldHostname,
		computev1.HostResourceFieldProductName,
		computev1.HostResourceFieldBiosVersion,
		computev1.HostResourceFieldBiosReleaseDate,
		computev1.HostResourceFieldBiosVendor,
		computev1.HostResourceFieldMetadata,
		computev1.HostResourceFieldSerialNumber,
		computev1.HostResourceFieldUuid,
	})

}

// UpdateHostnic updates an existing Hostnic resource info in Inventory, except state and other fields not allowed from RM.
func UpdateHostnic(ctx context.Context, c inv_client.InventoryClient, hostNic *computev1.HostnicResource) error {
	zlog.Debug().Msgf("Update Hostnic: %v", hostNic)

	err := UpdateInvResourceFields(ctx, c, hostNic, []string{
		computev1.HostnicResourceFieldKind,
		computev1.HostnicResourceFieldDeviceName,
		computev1.HostnicResourceEdgeHost,
		computev1.HostnicResourceFieldPciIdentifier,
		computev1.HostnicResourceFieldMacAddr,
		computev1.HostnicResourceFieldSriovEnabled,
		computev1.HostnicResourceFieldSriovVfsNum,
		computev1.HostnicResourceFieldSriovVfsTotal,
		computev1.HostnicResourceFieldPeerName,
		computev1.HostnicResourceFieldPeerDescription,
		computev1.HostnicResourceFieldPeerMac,
		computev1.HostnicResourceFieldPeerMgmtIp,
		computev1.HostnicResourceFieldPeerPort,
		computev1.HostnicResourceFieldSupportedLinkMode,
		computev1.HostnicResourceFieldAdvertisingLinkMode,
		computev1.HostnicResourceFieldCurrentSpeedBps,
		computev1.HostnicResourceFieldCurrentDuplex,
		computev1.HostnicResourceFieldFeatures,
		computev1.HostnicResourceFieldMtu,
		computev1.HostnicResourceFieldLinkState,
		computev1.HostnicResourceFieldBmcInterface,
	})
	if err != nil {
		zlog.MiSec().MiError("Failed update Hostnic resource %v", hostNic).Msg("UpdateHostnic")
		return err
	}
	return nil
}
func GetInventoryResourceAndID(resource proto.Message) (*inv_v1.Resource, string, error) {
	var (
		invResource   = &inv_v1.Resource{}
		invResourceID string
	)
	if resource == nil {
		err := inv_errors.Errorfc(codes.InvalidArgument, "no resource provided")
		return nil, invResourceID, err
	}

	switch res := resource.(type) {
	case *computev1.HostResource:
		invResource.Resource = &inv_v1.Resource_Host{
			Host: res,
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
	case *computev1.HostnicResource:
		invResource.Resource = &inv_v1.Resource_Hostnic{
			Hostnic: res,
		}
		invResourceID = res.GetResourceId()
	default:
		err := inv_errors.Errorfc(codes.InvalidArgument, "unsupported resource type: %t", resource)
		return nil, invResourceID, err
	}

	return invResource, invResourceID, nil
}

func UpdateHostCurrentState(ctx context.Context, c inv_client.InventoryClient, host *computev1.HostResource) error {
	return UpdateInvResourceFields(ctx, c, host, []string{
		"current_state",
	})
}

func UpdateInstanceCurrentState(ctx context.Context, c inv_client.InventoryClient, inst *computev1.InstanceResource) error {
	return UpdateInvResourceFields(ctx, c, inst, []string{
		"current_state",
	})
}

func UpdateInvResourceFields(ctx context.Context, c inv_client.InventoryClient, resource proto.Message, fields []string) error {
	if resource == nil {
		return inv_errors.Errorfc(codes.InvalidArgument, "no resource provided")
	}

	if len(fields) == 0 {
		return nil
	}

	resCopy := proto.Clone(resource)
	invResource, invResourceID, err := GetInventoryResourceAndID(resCopy)
	if err != nil {
		return err
	}

	fieldMask, err := fieldmaskpb.New(resCopy, fields...)
	if err != nil {
		return inv_errors.Wrap(err)
	}

	err = util.ValidateMaskAndFilterMessage(resCopy, fieldMask, true)
	if err != nil {
		return err
	}

	_, err = c.Update(ctx, invResourceID, fieldMask, invResource)
	return err
}

func listOneHost(
	ctx context.Context,
	c inv_client.InventoryClient,
	filter *inv_v1.ResourceFilter,
) (*computev1.HostResource, error) {
	resources, err := ListAllResources(ctx, c, filter)
	if err != nil {
		return nil, err
	}

	if len(resources) == 0 {
		return nil, inv_errors.Errorfc(codes.NotFound, "no resources found")
	}
	if len(resources) > 1 {
		return nil, inv_errors.Errorfc(codes.Internal, "returned multiple (%d) resources", len(resources))
	}

	host := resources[0].GetHost()
	if host == nil {
		err = inv_errors.Errorfc(codes.Internal, "empty Host resource")
		return nil, err
	}

	return host, nil
}

func CreateOsResource(ctx context.Context, c inv_client.InventoryClient, osr *osv1.OperatingSystemResource) (string, error) {
	resreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Os{
			Os: osr,
		},
	}
	res, err := c.Create(ctx, resreq)
	if err != nil {
		return "", err
	}
	return res.ResourceId, nil
}

func GetOsResourceById(ctx context.Context, c inv_client.InventoryClient, resourceID string) (*osv1.OperatingSystemResource, error) {
	osGetRes, err := c.Get(ctx, resourceID)
	if err != nil {
		return nil, err
	}
	osRes := osGetRes.GetResource().GetOs()

	if validateosErr := osRes.ValidateAll(); validateosErr != nil {
		return nil, inv_errors.Wrap(validateosErr)
	}

	return osRes, nil
}

func GetOsResources(ctx context.Context, c inv_client.InventoryClient) (osres []*osv1.OperatingSystemResource, err error) {
	filter, err := util.GetFilterFromSetResource(&inv_v1.Resource{
		Resource: &inv_v1.Resource_Os{
			Os: &osv1.OperatingSystemResource{},
		},
	})
	if err != nil {
		return nil, err
	}
	resources, err := ListAllResources(ctx, c, filter)
	if err != nil {
		return nil, err
	}
	return util.GetSpecificResourceList[*osv1.OperatingSystemResource](resources)
}

func DeleteOsResource(ctx context.Context, c inv_client.InventoryClient, resourceID string) error {
	delResp, err := c.Delete(ctx, resourceID)
	if err != nil {
		return err
	}

	if delValidateErr := delResp.ValidateAll(); delValidateErr != nil {
		return inv_errors.Wrap(delValidateErr)
	}
	return nil
}

func UpdateOsResource(ctx context.Context, c inv_client.InventoryClient, osr *osv1.OperatingSystemResource) error {
	fieldMask := &fieldmaskpb.FieldMask{
		Paths: []string{
			"repo_url",
		},
	}

	res := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Os{
			Os: &osv1.OperatingSystemResource{
				RepoUrl: osr.RepoUrl,
			},
		},
	}
	_, err := c.Update(
		ctx,
		osr.ResourceId,
		fieldMask,
		res,
	)
	if err != nil {
		return err
	}

	return nil
}

func FindAllResources(ctx context.Context, c inv_client.InventoryClient, kinds []inv_v1.ResourceKind) ([]string, error) {
	fmk := &fieldmaskpb.FieldMask{Paths: []string{}}
	var allResources []string
	for _, kind := range kinds {
		res, err := util.GetResourceFromKind(kind)
		if err != nil {
			return nil, err
		}
		resources, err := c.FindAll(ctx, res, fmk)
		if err != nil {
			return nil, err
		}
		allResources = append(allResources, resources...)
	}
	return allResources, nil
}
