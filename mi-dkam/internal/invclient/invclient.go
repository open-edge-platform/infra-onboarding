// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

package invclient

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	uuid_lib "github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/inventory/v1"
	network_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/network/v1"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/os/v1"
	provider_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/provider/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/client"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/logging"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/util"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/validator"
)

const (
	DefaultTimeout = 3 * time.Second
)

var (
	clientName = "DKAMInventoryClient"
	zlog       = logging.GetLogger(clientName)
)

type DKAMInventoryClient struct {
	Client  client.InventoryClient
	Watcher chan *client.WatchEvents
}

type Options struct {
	InventoryAddress string
	EnableTracing    bool
	ClientKind       inv_v1.ClientKind
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

func WithClientKind(clientKind inv_v1.ClientKind) Option {
	return func(options *Options) {
		options.ClientKind = clientKind
	}
}

// NewDKAMInventoryClientWithOptions creates a client by instantiating a new Inventory client.
func NewDKAMInventoryClientWithOptions(opts ...Option) (*DKAMInventoryClient, error) {
	ctx := context.Background()
	options := Options{
		ClientKind: inv_v1.ClientKind_CLIENT_KIND_RESOURCE_MANAGER,
	}
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
		ClientKind: options.ClientKind,
		ResourceKinds: []inv_v1.ResourceKind{
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
	return NewDKAMInventoryClient(invClient, eventsWatcher)
}

func NewDKAMInventoryClient(
	invClient client.InventoryClient, watcher chan *client.WatchEvents,
) (*DKAMInventoryClient, error) {
	cli := &DKAMInventoryClient{
		Client:  invClient,
		Watcher: watcher,
	}
	return cli, nil
}

func (c *DKAMInventoryClient) Close() {
	if err := c.Client.Close(); err != nil {
		zlog.MiSec().MiErr(err).Msgf("")
	}
	zlog.MiSec().Info().Msgf("Inventory client stopped")
}

// List resources by the provided filter. Filter is done only on fields that are set (not default values of the
// resources). Note that this function will NOT return an error if an object is not found.
func (c *DKAMInventoryClient) listAllResources(
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
			if err = validator.ValidateMessage(v.GetResource()); err != nil {
				zlog.MiSec().MiErr(err).Msgf("Invalid input, validation has failed: %v", v)
				return nil, inv_errors.Wrap(err)
			}
			resources = append(resources, v.GetResource())
		}
	}
	return resources, nil
}

func (c *DKAMInventoryClient) getResourceByID(ctx context.Context, resourceID string) (*inv_v1.GetResourceResponse, error) {
	getresresp, err := c.Client.Get(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	return getresresp, nil
}

func (c *DKAMInventoryClient) createResource(ctx context.Context, resource *inv_v1.Resource) (string, error) {
	res, err := c.Client.Create(ctx, resource)
	if err != nil {
		return "", err
	}
	return res.ResourceId, nil
}

func (c *DKAMInventoryClient) UpdateInvResourceFields(ctx context.Context,
	resource proto.Message, fields []string,
) error {
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

func (c *DKAMInventoryClient) listAndReturnHost(
	ctx context.Context,
	filter *inv_v1.ResourceFilter,
) (*computev1.HostResource, error) {
	resources, err := c.listAllResources(ctx, filter)
	if err != nil {
		return nil, err
	}
	err = util.CheckListOutputIsSingular(resources)
	if err != nil {
		zlog.Info().Msgf("Obtained non-singular Host resource")
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

func (c *DKAMInventoryClient) FindAllInstances(ctx context.Context) ([]string, error) {
	return c.FindAllResources(ctx, []inv_v1.ResourceKind{inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE})
}

func (c *DKAMInventoryClient) CreateHostResource(ctx context.Context,
	host *computev1.HostResource,
) (string, error) {
	return c.createResource(ctx, &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: host,
		},
	})
}

func (c *DKAMInventoryClient) GetHostResources(ctx context.Context,
) (hostres []*computev1.HostResource, err error) {
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

func (c *DKAMInventoryClient) GetHostResourceByResourceID(ctx context.Context,
	resourceID string,
) (*computev1.HostResource, error) {
	resp, err := c.getResourceByID(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	host := resp.GetResource().GetHost()

	if validateErr := validator.ValidateMessage(host); validateErr != nil {
		return nil, inv_errors.Wrap(validateErr)
	}

	return host, nil
}

func (c *DKAMInventoryClient) GetHostBmcNic(ctx context.Context, host *computev1.HostResource,
) (*computev1.HostnicResource, error) {
	filter := &inv_v1.ResourceFilter{
		Resource: &inv_v1.Resource{
			Resource: &inv_v1.Resource_Hostnic{},
		},
		Filter: fmt.Sprintf("%s.%s = %q AND %s = true",
			computev1.HostnicResourceEdgeHost,
			computev1.HostResourceFieldResourceId,
			host.GetResourceId(),
			computev1.HostnicResourceFieldBmcInterface),
	}

	resources, err := c.listAllResources(ctx, filter)
	if err != nil {
		return nil, err
	}

	hostNics, err := util.GetSpecificResourceList[*computev1.HostnicResource](resources)
	if err != nil {
		return nil, err
	}

	if len(hostNics) == 0 {
		return nil, inv_errors.Errorfc(codes.NotFound,
			"No BMC interfaces found for Host %s", host.ResourceId)
	}

	if len(hostNics) > 1 {
		zlog.Warn().Msgf("More than one BMC interface found for host %s, using the first NIC from the list.",
			host.GetResourceId())
	}

	return hostNics[0], nil
}

func (c *DKAMInventoryClient) GetHostResourceByUUID(
	ctx context.Context,
	uuid string,
) (*computev1.HostResource, error) {
	_, err := uuid_lib.Parse(uuid)
	// additional check for length is needed because .Parse() accepts other non-standard format (see function docs).
	if err != nil || len(uuid) != 36 {
		return nil, inv_errors.Errorfc(codes.InvalidArgument, "invalid UUID")
	}

	filter := &inv_v1.ResourceFilter{
		Resource: &inv_v1.Resource{
			Resource: &inv_v1.Resource_Host{},
		},
		Filter: fmt.Sprintf("%s = %q", computev1.HostResourceFieldUuid, uuid),
	}

	return c.listAndReturnHost(ctx, filter)
}

func (c *DKAMInventoryClient) UpdateHostResource(ctx context.Context, host *computev1.HostResource) error {
	return c.UpdateInvResourceFields(ctx, host, []string{
		computev1.HostResourceFieldKind,
		computev1.HostResourceFieldName,
		computev1.HostResourceFieldSerialNumber,
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

func (c *DKAMInventoryClient) updateHostCurrentState(ctx context.Context, host *computev1.HostResource) error {
	return c.UpdateInvResourceFields(ctx, host, []string{
		computev1.HostResourceFieldCurrentState,
	})
}

func (c *DKAMInventoryClient) DeleteHostResource(ctx context.Context, resourceID string) error {
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

func (c *DKAMInventoryClient) CreateInstanceResource(ctx context.Context,
	inst *computev1.InstanceResource,
) (string, error) {
	return c.createResource(ctx, &inv_v1.Resource{
		Resource: &inv_v1.Resource_Instance{
			Instance: inst,
		},
	})
}

func (c *DKAMInventoryClient) GetInstanceResourceByResourceID(ctx context.Context,
	resourceID string,
) (*computev1.InstanceResource, error) {
	resp, err := c.getResourceByID(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	inst := resp.GetResource().GetInstance()

	if validateErr := validator.ValidateMessage(inst); validateErr != nil {
		return nil, inv_errors.Wrap(validateErr)
	}

	return inst, nil
}

func (c *DKAMInventoryClient) GetInstanceResources(ctx context.Context) ([]*computev1.InstanceResource, error) {
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

func (c *DKAMInventoryClient) UpdateInstanceCurrentState(ctx context.Context,
	instance *computev1.InstanceResource,
) error {
	return c.UpdateInvResourceFields(ctx, instance, []string{
		computev1.HostResourceFieldCurrentState,
	})
}

func (c *DKAMInventoryClient) DeleteInstanceResource(ctx context.Context, resourceID string) error {
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

func (c *DKAMInventoryClient) DeleteResource(ctx context.Context, resourceID string) error {
	zlog.Debug().Msgf("Delete resource: %v", resourceID)

	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()
	_, err := c.Client.Delete(ctx, resourceID)
	if inv_errors.IsNotFound(err) {
		zlog.Debug().Msgf("Not found while deleting resource, dropping err: resourceID=%s", resourceID)
		return nil
	}
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("Failed to delete resource: resourceID=%s", resourceID)
		return err
	}
	return err
}

func (c *DKAMInventoryClient) CreateOSResource(ctx context.Context,
	os *osv1.OperatingSystemResource,
) (string, error) {
	return c.createResource(ctx, &inv_v1.Resource{
		Resource: &inv_v1.Resource_Os{
			Os: os,
		},
	})
}

func (c *DKAMInventoryClient) GetOSResourceByResourceID(ctx context.Context,
	resourceID string,
) (*osv1.OperatingSystemResource, error) {
	resp, err := c.getResourceByID(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	inst := resp.GetResource().GetOs()

	if validateErr := validator.ValidateMessage(inst); validateErr != nil {
		return nil, inv_errors.Wrap(validateErr)
	}

	return inst, nil
}

func (c *DKAMInventoryClient) GetOSResources(ctx context.Context) ([]*osv1.OperatingSystemResource, error) {
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
func (c *DKAMInventoryClient) ListIPAddresses(ctx context.Context, hostNic *computev1.HostnicResource) (
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

func (c *DKAMInventoryClient) FindAllResources(ctx context.Context,
	kinds []inv_v1.ResourceKind,
) ([]string, error) {
	var allResources []string
	for _, kind := range kinds {
		res, err := util.GetResourceFromKind(kind)
		if err != nil {
			return nil, err
		}
		filter := &inv_v1.ResourceFilter{
			Resource: res,
		}
		resources, err := c.Client.FindAll(ctx, filter)
		if err != nil {
			return nil, err
		}
		allResources = append(allResources, resources...)
	}
	return allResources, nil
}

func (c *DKAMInventoryClient) GetProviderResources(ctx context.Context) ([]*provider_v1.ProviderResource, error) {
	filter := &inv_v1.ResourceFilter{
		Resource: &inv_v1.Resource{
			Resource: &inv_v1.Resource_Provider{},
		},
	}
	zlog.Info().Msgf("GetProviderResources Filter: %v", filter)
	resources, err := c.listAllResources(ctx, filter)
	if err != nil {
		return nil, err
	}
	return util.GetSpecificResourceList[*provider_v1.ProviderResource](resources)
}

// DeleteIPAddress deletes an existing IP address resource in Inventory
// by setting to DELETED the current state of the resource.
func (c *DKAMInventoryClient) DeleteIPAddress(ctx context.Context, resourceID string) error {
	zlog.Debug().Msgf("Delete IPAddress: %v", resourceID)

	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()
	ipAddress := &network_v1.IPAddressResource{
		ResourceId:   resourceID,
		CurrentState: network_v1.IPAddressState_IP_ADDRESS_STATE_DELETED,
	}

	err := c.UpdateInvResourceFields(ctx, ipAddress, []string{network_v1.IPAddressResourceFieldCurrentState})
	if inv_errors.IsNotFound(err) {
		zlog.Debug().Msgf("Not found while IP address delete, dropping err: resourceID=%s", resourceID)
		return nil
	}
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("Failed delete IPAddress resource %s", resourceID)
		return err
	}

	return err
}

func (c *DKAMInventoryClient) listAndReturnProvider(
	ctx context.Context,
	filter *inv_v1.ResourceFilter,
) (*provider_v1.ProviderResource, error) {
	resources, err := c.listAllResources(ctx, filter)
	if err != nil {
		return nil, err
	}
	err = util.CheckListOutputIsSingular(resources)
	if err != nil {
		zlog.Info().Msgf("Obtained non-singular provider resource")
		return nil, err
	}
	provider := resources[0].GetProvider()
	if provider == nil {
		err = inv_errors.Errorfc(codes.Internal, "Empty provider resource")
		zlog.MiSec().MiErr(err).Msg("Inventory returned an empty provider resource")
		return nil, err
	}

	return provider, nil
}

func GetProviderResourceByName(
	ctx context.Context,
	c *DKAMInventoryClient,
	name string,
) (*provider_v1.ProviderResource, error) {
	zlog.Info().Msgf("Obtaining Provider resource by its name (%s)", name)
	if name == "" {
		err := inv_errors.Errorfc(codes.InvalidArgument, "Empty provider name")
		zlog.MiSec().MiErr(err).Msg("Empty provider name obtained at the input of the function")
		return nil, err
	}

	filter := &inv_v1.ResourceFilter{
		Resource: &inv_v1.Resource{Resource: &inv_v1.Resource_Provider{}},
		Filter:   fmt.Sprintf("%s = %q", provider_v1.ProviderResourceFieldName, name),
	}
	return c.listAndReturnProvider(ctx, filter)
}

//nolint:tagliatelle // Renaming the json keys may effect while unmarshalling/marshaling so, used nolint.
type ProviderConfig struct {
	DefaultOs     string `json:"defaultOs"`
	AutoProvision bool   `json:"autoProvision"`
	CustomerID    string `json:"customerID"`
}

func (c *DKAMInventoryClient) GetProviderConfig(
	ctx context.Context,
	name string,
) (*ProviderConfig, error) {
	// Get the provider resource by name
	provider, err := GetProviderResourceByName(ctx, c, name)
	if err != nil {
		return nil, err
	}

	var pconf ProviderConfig
	// Unmarshal provider config JSON into pconf
	if err := json.Unmarshal([]byte(provider.Config), &pconf); err != nil {
		zlog.MiErr(err).Msgf("failed to unmarshal ProviderConfig")
		return nil, inv_errors.Wrap(err)
	}

	return &pconf, nil
}
