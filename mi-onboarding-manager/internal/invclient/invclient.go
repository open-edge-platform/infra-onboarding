// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

package invclient

import (
	"context"
	"fmt"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	network_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/network/v1"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/os/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/client"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"sync"
	"time"
)

const (
	DefaultTimeout = 3 * time.Second
)

var (
	clientName = "OnboardingInventoryClient"
	zlog       = logging.GetLogger(clientName)
)

type OnboardingInventoryClient struct {
	Client  client.InventoryClient
	Watcher chan *client.WatchEvents
}

type Options struct {
	InventoryAddress string
	EnableTracing    bool
}

type Option func(*Options)

// WithInventoryAddress sets the Inventory Address.
func WithInventoryAddress(invAddr string) Option {
	return func(options *Options) {
		options.InventoryAddress = invAddr
	}
}

// WithEnableTracing enables tracing.
func WithEnableTracing(enableTracing bool) Option {
	return func(options *Options) {
		options.EnableTracing = enableTracing
	}
}

// NewOnboardingInventoryClientWithOptions creates a client by instantiating a new Inventory client.
func NewOnboardingInventoryClientWithOptions(opts ...Option) (*OnboardingInventoryClient, error) {
	ctx := context.Background()
	var options Options
	for _, opt := range opts {
		opt(&options)
	}

	wg := sync.WaitGroup{}
	eventsWatcher := make(chan *client.WatchEvents)
	cfg := client.InventoryClientConfig{
		Name:                      clientName,
		Address:                   options.InventoryAddress,
		EnableRegisterRetry:       false,
		AbortOnUnknownClientError: true,
		// TODO: add security credentials
		SecurityCfg: &client.SecurityConfig{
			Insecure: true,
			CaPath:   "",
			CertPath: "",
			KeyPath:  "",
		},
		Events:     eventsWatcher,
		ClientKind: inv_v1.ClientKind_CLIENT_KIND_RESOURCE_MANAGER,
		ResourceKinds: []inv_v1.ResourceKind{
			inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE,
			inv_v1.ResourceKind_RESOURCE_KIND_HOST,
			inv_v1.ResourceKind_RESOURCE_KIND_OS,
		},
		Wg:            &wg,
		EnableTracing: options.EnableTracing,
	}

	invClient, err := client.NewInventoryClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	zlog.MiSec().Info().Msgf("Inventory client started")
	return NewOnboardingInventoryClient(invClient, eventsWatcher)
}

func NewOnboardingInventoryClient(
	invClient client.InventoryClient, watcher chan *client.WatchEvents,
) (*OnboardingInventoryClient, error) {
	cli := &OnboardingInventoryClient{
		Client:  invClient,
		Watcher: watcher,
	}
	return cli, nil
}

func (c *OnboardingInventoryClient) Close() {
	if err := c.Client.Close(); err != nil {
		zlog.MiSec().MiErr(err).Msgf("")
	}
	zlog.MiSec().Info().Msgf("Inventory client stopped")
}

// List resources by the provided filter. Filter is done only on fields that are set (not default values of the
// resources). Note that this function will NOT return an error if an object is not found.
func (c *OnboardingInventoryClient) listAllResources(
	ctx context.Context,
	filter *inv_v1.ResourceFilter,
) ([]*inv_v1.Resource, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	// we agreed to not return a NotFound error to avoid too many 'Not Found'
	// responses to the consumer of our external APIs.
	objs, err := c.Client.List(ctx, filter)
	if err != nil && !inv_errors.IsNotFound(err) {
		zlog.MiSec().MiErr(err).Msgf("Unable to listAll %v", filter)
		return nil, err
	}
	resources := make([]*inv_v1.Resource, 0, len(objs.Resources))
	for _, v := range objs.Resources {
		if v.GetResource() != nil {
			if err = v.GetResource().ValidateAll(); err != nil {
				zlog.MiSec().MiErr(err).Msgf("Invalid input, validation has failed: %v", v)
				return nil, inv_errors.Wrap(err)
			}
			resources = append(resources, v.GetResource())
		}
	}
	return resources, nil
}

func (c *OnboardingInventoryClient) findAllResources(ctx context.Context, kind inv_v1.ResourceKind) ([]string, error) {
	fmk := &fieldmaskpb.FieldMask{Paths: []string{}}
	res, err := util.GetResourceFromKind(kind)
	if err != nil {
		return nil, err
	}

	resources, err := c.Client.FindAll(ctx, res, fmk)
	if err != nil {
		return nil, err
	}

	return resources, nil
}

func (c *OnboardingInventoryClient) getResourceByID(ctx context.Context, resourceID string) (*inv_v1.GetResourceResponse, error) {
	getresresp, err := c.Client.Get(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	return getresresp, nil
}

func (c *OnboardingInventoryClient) createResource(ctx context.Context, resource *inv_v1.Resource) (string, error) {
	res, err := c.Client.Create(ctx, resource)
	if err != nil {
		return "", err
	}
	return res.ResourceId, nil
}

func (c *OnboardingInventoryClient) UpdateInvResourceFields(ctx context.Context, resource proto.Message, fields []string) error {
	if resource == nil {
		return inv_errors.Errorfc(codes.InvalidArgument, "no resource provided")
	}

	if len(fields) == 0 {
		return nil
	}

	resCopy := proto.Clone(resource)
	invResource, invResourceID, err := getInventoryResourceAndID(resCopy)
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

	_, err = c.Client.Update(ctx, invResourceID, fieldMask, invResource)
	return err
}

func (c *OnboardingInventoryClient) listAndReturnHost(
	ctx context.Context,
	filter *inv_v1.ResourceFilter,
) (*computev1.HostResource, error) {
	resources, err := c.listAllResources(ctx, filter)
	if err != nil {
		return nil, err
	}
	err = util.CheckListOutputIsSingular(resources)
	if err != nil {
		zlog.MiSec().MiErr(err).Msg("Obtained non-singular Host resource")
		return nil, err
	}
	host := resources[0].GetHost()
	if host == nil {
		err = inv_errors.Errorfc(codes.Internal, "Empty Host resource")
		zlog.MiSec().MiErr(err).Msg("Inventory returned an empty Host resource")
		return nil, err
	}

	return host, nil
}

func (c *OnboardingInventoryClient) FindAllInstances(ctx context.Context) ([]string, error) {
	return c.findAllResources(ctx, inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE)
}

func (c *OnboardingInventoryClient) CreateHostResource(ctx context.Context, host *computev1.HostResource) (string, error) {
	return c.createResource(ctx, &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: host,
		},
	})
}

func (c *OnboardingInventoryClient) GetHostResources(ctx context.Context) (hostres []*computev1.HostResource, err error) {
	filter := &inv_v1.ResourceFilter{
		Resource: &inv_v1.Resource{
			Resource: &inv_v1.Resource_Host{},
		},
	}

	resources, err := c.listAllResources(ctx, filter)
	if err != nil {
		return nil, err
	}
	return util.GetSpecificResourceList[*computev1.HostResource](resources)
}

func (c *OnboardingInventoryClient) GetHostResourceByResourceID(ctx context.Context, resourceID string) (*computev1.HostResource, error) {
	resp, err := c.getResourceByID(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	host := resp.GetResource().GetHost()

	if validateErr := host.ValidateAll(); validateErr != nil {
		return nil, inv_errors.Wrap(validateErr)
	}

	return host, nil
}

func (c *OnboardingInventoryClient) GetHostResourceByUUID(
	ctx context.Context,
	uuid string,
) (*computev1.HostResource, error) {
	if uuid == "" {
		err := inv_errors.Errorfc(codes.InvalidArgument, "empty UUID")
		return nil, err
	}

	filter := &inv_v1.ResourceFilter{
		Resource: &inv_v1.Resource{
			Resource: &inv_v1.Resource_Host{},
		},
		Filter: fmt.Sprintf("host.uuid=%q", uuid),
	}

	return c.listAndReturnHost(ctx, filter)
}

func (c *OnboardingInventoryClient) UpdateHostResource(ctx context.Context, host *computev1.HostResource) error {
	return c.UpdateInvResourceFields(ctx, host, []string{
		computev1.HostResourceFieldKind,
		computev1.HostResourceFieldName,
		computev1.HostResourceFieldCurrentState,
		computev1.HostResourceFieldMgmtIp,
		computev1.HostResourceFieldBmcKind,
		computev1.HostResourceFieldBmcIp,
		computev1.HostResourceFieldBmcUsername,
		computev1.HostResourceFieldBmcPassword,
		computev1.HostResourceFieldPxeMac,
		computev1.HostResourceFieldHostname,
		// other host fields are updated by Host Resource Manager
	})
}

func (c *OnboardingInventoryClient) CreateHostNICResource(ctx context.Context, hostNIC *computev1.HostnicResource) (string, error) {
	return c.createResource(ctx, &inv_v1.Resource{
		Resource: &inv_v1.Resource_Hostnic{
			Hostnic: hostNIC,
		},
	})
}

func (c *OnboardingInventoryClient) UpdateHostNIC(ctx context.Context, hostNIC *computev1.HostnicResource) error {
	return c.UpdateInvResourceFields(ctx, hostNIC, []string{
		computev1.HostnicResourceFieldKind,
		computev1.HostnicResourceFieldDeviceName,
		computev1.HostnicResourceEdgeHost,
		computev1.HostnicResourceFieldMacAddr,
		computev1.HostnicResourceFieldPeerMgmtIp,
		computev1.HostnicResourceFieldBmcInterface,
		// other host NIC fields are updated by Host Resource Manager
	})
}

func (c *OnboardingInventoryClient) updateHostCurrentState(ctx context.Context, host *computev1.HostResource) error {
	return c.UpdateInvResourceFields(ctx, host, []string{
		computev1.HostResourceFieldCurrentState,
	})
}

func (c *OnboardingInventoryClient) SetHostStatus(ctx context.Context, hostID string, hostStatus computev1.HostStatus) error {
	updateHost := &computev1.HostResource{
		ResourceId: hostID,
		HostStatus: hostStatus,
	}

	return c.UpdateInvResourceFields(ctx, updateHost, []string{
		computev1.HostResourceFieldHostStatus,
	})
}

func (c *OnboardingInventoryClient) DeleteHostResource(ctx context.Context, resourceID string) error {
	h := &computev1.HostResource{
		ResourceId:   resourceID,
		CurrentState: computev1.HostState_HOST_STATE_DELETED,
	}

	err := c.updateHostCurrentState(ctx, h)
	if err != nil {
		return err
	}

	return nil
}

func (c *OnboardingInventoryClient) CreateInstanceResource(ctx context.Context, inst *computev1.InstanceResource) (string, error) {
	return c.createResource(ctx, &inv_v1.Resource{
		Resource: &inv_v1.Resource_Instance{
			Instance: inst,
		},
	})
}

func (c *OnboardingInventoryClient) GetInstanceResourceByResourceID(ctx context.Context, resourceID string) (*computev1.InstanceResource, error) {
	resp, err := c.getResourceByID(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	inst := resp.GetResource().GetInstance()

	if validateErr := inst.ValidateAll(); validateErr != nil {
		return nil, inv_errors.Wrap(validateErr)
	}

	return inst, nil
}

func (c *OnboardingInventoryClient) GetInstanceResources(ctx context.Context) ([]*computev1.InstanceResource, error) {
	filter := &inv_v1.ResourceFilter{
		Resource: &inv_v1.Resource{
			Resource: &inv_v1.Resource_Instance{},
		},
	}

	resources, err := c.listAllResources(ctx, filter)
	if err != nil {
		return nil, err
	}
	return util.GetSpecificResourceList[*computev1.InstanceResource](resources)
}

func (c *OnboardingInventoryClient) UpdateInstanceResource(ctx context.Context, inst *computev1.InstanceResource) error {
	return c.UpdateInvResourceFields(ctx, inst, []string{
		computev1.InstanceResourceFieldCurrentState,
		computev1.InstanceResourceFieldKind,
		computev1.InstanceResourceFieldStatus,
		computev1.InstanceResourceFieldStatusDetail,
		computev1.InstanceResourceFieldVmCpuCores,
		computev1.InstanceResourceFieldVmStorageBytes,
		computev1.InstanceResourceFieldVmMemoryBytes,
		computev1.InstanceResourceFieldName,
	})
}

func (c *OnboardingInventoryClient) UpdateInstanceCurrentState(ctx context.Context, instance *computev1.InstanceResource) error {
	return c.UpdateInvResourceFields(ctx, instance, []string{
		computev1.HostResourceFieldCurrentState,
	})
}

func (c *OnboardingInventoryClient) SetInstanceStatus(ctx context.Context, instanceID string, instanceStatus computev1.InstanceStatus) error {
	updateInstance := &computev1.InstanceResource{
		ResourceId: instanceID,
		Status:     instanceStatus,
	}

	return c.UpdateInvResourceFields(ctx, updateInstance, []string{
		computev1.InstanceResourceFieldStatus,
	})
}

func (c *OnboardingInventoryClient) DeleteInstanceResource(ctx context.Context, resourceID string) error {
	inst := &computev1.InstanceResource{
		ResourceId:   resourceID,
		CurrentState: computev1.InstanceState_INSTANCE_STATE_DELETED,
	}

	err := c.UpdateInstanceCurrentState(ctx, inst)
	if err != nil {
		return err
	}

	return nil
}

func (c *OnboardingInventoryClient) DeleteResource(ctx context.Context, resourceID string) error {
	zlog.Debug().Msgf("Delete resource: %v", resourceID)

	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()
	_, err := c.Client.Delete(ctx, resourceID)
	if inv_errors.IsNotFound(err) {
		zlog.Debug().Msgf("Not found while HostGPU delete, dropping err: resourceID=%s", resourceID)
		return nil
	}
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("Failed delete Hostgpu resource %s", resourceID)
		return err
	}
	return err
}

func (c *OnboardingInventoryClient) CreateOSResource(ctx context.Context, os *osv1.OperatingSystemResource) (string, error) {
	return c.createResource(ctx, &inv_v1.Resource{
		Resource: &inv_v1.Resource_Os{
			Os: os,
		},
	})
}

func (c *OnboardingInventoryClient) GetOSResourceByResourceID(ctx context.Context, resourceID string) (*osv1.OperatingSystemResource, error) {
	resp, err := c.getResourceByID(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	inst := resp.GetResource().GetOs()

	if validateErr := inst.ValidateAll(); validateErr != nil {
		return nil, inv_errors.Wrap(validateErr)
	}

	return inst, nil
}

func (c *OnboardingInventoryClient) GetOSResources(ctx context.Context) ([]*osv1.OperatingSystemResource, error) {
	filter := &inv_v1.ResourceFilter{
		Resource: &inv_v1.Resource{
			Resource: &inv_v1.Resource_Os{},
		},
	}

	resources, err := c.listAllResources(ctx, filter)
	if err != nil {
		return nil, err
	}
	return util.GetSpecificResourceList[*osv1.OperatingSystemResource](resources)
}

// ListIPAddresses returns the list of IP addresses associated to the nic.
func (c *OnboardingInventoryClient) ListIPAddresses(ctx context.Context, hostNic *computev1.HostnicResource) (
	[]*network_v1.IPAddressResource, error,
) {
	zlog.Debug().Msgf("List IPAddress associated to: %v", hostNic)

	filter := &inv_v1.ResourceFilter{
		Resource: &inv_v1.Resource{
			Resource: &inv_v1.Resource_Ipaddress{},
		},
		Filter: fmt.Sprintf("has(%s) AND %s.%s = %q", network_v1.IPAddressResourceEdgeNic,
			network_v1.IPAddressResourceEdgeNic, computev1.HostnicResourceFieldResourceId, hostNic.GetResourceId()),
	}
	resources, err := c.listAllResources(ctx, filter)
	if err != nil {
		return nil, err
	}
	return util.GetSpecificResourceList[*network_v1.IPAddressResource](resources)
}

func (c *OnboardingInventoryClient) FindAllResources(ctx context.Context, kinds []inv_v1.ResourceKind) ([]string, error) {
	fmk := &fieldmaskpb.FieldMask{Paths: []string{}}
	var allResources []string
	for _, kind := range kinds {
		res, err := util.GetResourceFromKind(kind)
		if err != nil {
			return nil, err
		}
		resources, err := c.Client.FindAll(ctx, res, fmk)
		if err != nil {
			return nil, err
		}
		allResources = append(allResources, resources...)
	}
	return allResources, nil
}
