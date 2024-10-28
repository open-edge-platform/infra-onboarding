// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

package invclient

import (
	"context"
	"encoding/json"
	"flag"
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
	statusv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/status/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/client"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/logging"
	inv_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/status"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/util"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/validator"
)

var inventoryTimeout = flag.Duration("invTimeout", DefaultInventoryTimeout, "Inventory API calls timeout")

const MaxSerialNumberLength = 36
const (
	DefaultInventoryTimeout = 5 * time.Second
	ReconcileDefaultTimeout = 5 * time.Minute // Longer timeout for reconciling all resources
)

var ReconcileTimeout = flag.Duration(
	"timeoutReconcileAll",
	ReconcileDefaultTimeout,
	"Timeout used when reconciling all resources",
)

var (
	clientName = "OnboardingInventoryClient"
	zlog       = logging.GetLogger(clientName)
)

type OnboardingInventoryClient struct {
	Client  client.TenantAwareInventoryClient
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

// NewOnboardingInventoryClientWithOptions creates a client by instantiating a new Inventory client.
func NewOnboardingInventoryClientWithOptions(opts ...Option) (*OnboardingInventoryClient, error) {
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
			inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE,
			inv_v1.ResourceKind_RESOURCE_KIND_HOST,
			inv_v1.ResourceKind_RESOURCE_KIND_OS,
		},
		Wg:            &wg,
		EnableTracing: options.EnableTracing,
	}

	invClient, err := client.NewTenantAwareInventoryClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	zlog.MiSec().Info().Msgf("Inventory client started")
	return NewOnboardingInventoryClient(invClient, eventsWatcher)
}

func NewOnboardingInventoryClient(
	invClient client.TenantAwareInventoryClient, watcher chan *client.WatchEvents,
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
	ctx, cancel := context.WithTimeout(ctx, *inventoryTimeout)
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

func (c *OnboardingInventoryClient) getResourceByID(ctx context.Context, tenantID, resourceID string,
) (*inv_v1.GetResourceResponse, error) {
	getresresp, err := c.Client.Get(ctx, tenantID, resourceID)
	if err != nil {
		return nil, err
	}

	return getresresp, nil
}

func (c *OnboardingInventoryClient) createResource(ctx context.Context,
	tenantID string, resource *inv_v1.Resource,
) (string, error) {
	res, err := c.Client.Create(ctx, tenantID, resource)
	if err != nil {
		return "", err
	}
	_, rID, err := util.GetResourceKeyFromResource(res)
	if err != nil {
		// This error should never happen
		zlog.MiSec().MiErr(err).Msgf("this error should never happen")
		return "", err
	}
	return rID, nil
}

func (c *OnboardingInventoryClient) UpdateInvResourceFields(ctx context.Context, tenantID string,
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

	_, err = c.Client.Update(ctx, tenantID, invResourceID, fieldMask, invResource)
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

func (c *OnboardingInventoryClient) FindAllInstances(ctx context.Context) ([]*client.ResourceTenantIDCarrier, error) {
	return c.FindAllResources(ctx, []inv_v1.ResourceKind{inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE})
}

func (c *OnboardingInventoryClient) CreateHostResource(ctx context.Context, tenantID string,
	host *computev1.HostResource,
) (string, error) {
	return c.createResource(ctx, tenantID, &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: host,
		},
	})
}

func (c *OnboardingInventoryClient) GetHostResources(ctx context.Context,
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

func (c *OnboardingInventoryClient) GetHostResourceByResourceID(ctx context.Context, tenantID string,
	resourceID string,
) (*computev1.HostResource, error) {
	resp, err := c.getResourceByID(ctx, tenantID, resourceID)
	if err != nil {
		return nil, err
	}

	host := resp.GetResource().GetHost()

	if validateErr := validator.ValidateMessage(host); validateErr != nil {
		return nil, inv_errors.Wrap(validateErr)
	}

	return host, nil
}

func (c *OnboardingInventoryClient) GetHostBmcNic(ctx context.Context, host *computev1.HostResource,
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

func (c *OnboardingInventoryClient) GetHostResourceByUUID(
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

func (c *OnboardingInventoryClient) UpdateHostResource(ctx context.Context, tenantID string, host *computev1.HostResource) error {
	return c.UpdateInvResourceFields(ctx, tenantID, host, []string{
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

func (c *OnboardingInventoryClient) UpdateHostStateAndRuntimeStatus(ctx context.Context,
	tenantID string, host *computev1.HostResource,
) error {
	if host.HostStatus == "" || host.HostStatusTimestamp == 0 ||
		host.HostStatusIndicator == statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED {
		err := inv_errors.Errorfc(codes.InvalidArgument, "Missing mandatory host status fields during host status update")
		zlog.MiSec().MiErr(err).Msgf("Cannot update host status of %v", host)
		return err
	}

	return c.UpdateInvResourceFields(ctx, tenantID, host, []string{
		computev1.HostResourceFieldCurrentState,
		computev1.HostResourceFieldHostStatus,
		computev1.HostResourceFieldHostStatusIndicator,
		computev1.HostResourceFieldHostStatusTimestamp,
	})
}

func (c *OnboardingInventoryClient) updateHostCurrentState(ctx context.Context,
	tenantID string, host *computev1.HostResource,
) error {
	return c.UpdateInvResourceFields(ctx, tenantID, host, []string{
		computev1.HostResourceFieldCurrentState,
	})
}

func (c *OnboardingInventoryClient) SetHostOnboardingStatus(ctx context.Context, tenantID string, hostID string,
	onboardingStatus inv_status.ResourceStatus,
) error {
	updateHost := &computev1.HostResource{
		ResourceId:                hostID,
		OnboardingStatus:          onboardingStatus.Status,
		OnboardingStatusIndicator: onboardingStatus.StatusIndicator,
		OnboardingStatusTimestamp: uint64(time.Now().Unix()), // #nosec G115
	}

	return c.UpdateInvResourceFields(ctx, tenantID, updateHost, []string{
		computev1.HostResourceFieldOnboardingStatus,
		computev1.HostResourceFieldOnboardingStatusIndicator,
		computev1.HostResourceFieldOnboardingStatusTimestamp,
	})
}

func (c *OnboardingInventoryClient) SetHostStatusDetail(ctx context.Context, tenantID string,
	hostID string, onboardingStatus inv_status.ResourceStatus,
) error {
	updateHost := &computev1.HostResource{
		ResourceId:                hostID,
		OnboardingStatus:          onboardingStatus.Status,
		OnboardingStatusIndicator: onboardingStatus.StatusIndicator,
		OnboardingStatusTimestamp: uint64(time.Now().Unix()), // #nosec G115
	}

	zlog.Info().Msgf("Updateing host status %v", updateHost)

	return c.UpdateInvResourceFields(ctx, tenantID, updateHost, []string{
		computev1.HostResourceFieldOnboardingStatus,
		computev1.HostResourceFieldOnboardingStatusIndicator,
		computev1.HostResourceFieldOnboardingStatusTimestamp,
	})
}

func (c *OnboardingInventoryClient) DeleteHostResource(ctx context.Context, tenantID, resourceID string) error {
	h := &computev1.HostResource{
		ResourceId:   resourceID,
		CurrentState: computev1.HostState_HOST_STATE_DELETED,
	}

	err := c.updateHostCurrentState(ctx, tenantID, h)
	if err != nil {
		return err
	}

	return nil
}

func (c *OnboardingInventoryClient) CreateInstanceResource(ctx context.Context, tenantID string,
	inst *computev1.InstanceResource,
) (string, error) {
	return c.createResource(ctx, tenantID, &inv_v1.Resource{
		Resource: &inv_v1.Resource_Instance{
			Instance: inst,
		},
	})
}

func (c *OnboardingInventoryClient) GetInstanceResourceByResourceID(ctx context.Context, tenantID string,
	resourceID string,
) (*computev1.InstanceResource, error) {
	resp, err := c.getResourceByID(ctx, tenantID, resourceID)
	if err != nil {
		return nil, err
	}

	inst := resp.GetResource().GetInstance()

	if validateErr := validator.ValidateMessage(inst); validateErr != nil {
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

func (c *OnboardingInventoryClient) UpdateInstanceCurrentState(ctx context.Context, tenantID string,
	instance *computev1.InstanceResource,
) error {
	return c.UpdateInvResourceFields(ctx, tenantID, instance, []string{
		computev1.InstanceResourceFieldCurrentState,
	})
}

func (c *OnboardingInventoryClient) SetInstanceProvisioningStatus(ctx context.Context, tenantID string, instanceID string,
	provisioningStatus inv_status.ResourceStatus,
) error {
	updateInstance := &computev1.InstanceResource{
		ResourceId:                  instanceID,
		ProvisioningStatus:          provisioningStatus.Status,
		ProvisioningStatusIndicator: provisioningStatus.StatusIndicator,
		ProvisioningStatusTimestamp: uint64(time.Now().Unix()), // #nosec G115
	}

	return c.UpdateInvResourceFields(ctx, tenantID, updateInstance, []string{
		computev1.InstanceResourceFieldProvisioningStatus,
		computev1.InstanceResourceFieldProvisioningStatusIndicator,
		computev1.InstanceResourceFieldProvisioningStatusTimestamp,
	})
}

func (c *OnboardingInventoryClient) updateHostMacID(ctx context.Context, tenantID string, host *computev1.HostResource) error {
	return c.UpdateInvResourceFields(ctx, tenantID, host, []string{
		computev1.HostResourceFieldPxeMac,
	})
}

func (c *OnboardingInventoryClient) UpdateHostMacID(ctx context.Context, tenantID string, resourceID string,
	macid string,
) error {
	h := &computev1.HostResource{
		ResourceId: resourceID,
		PxeMac:     macid,
	}

	err := c.updateHostMacID(ctx, tenantID, h)
	if err != nil {
		return err
	}

	return nil
}

func (c *OnboardingInventoryClient) updateHostIP(ctx context.Context, tenantID string, host *computev1.HostResource) error {
	return c.UpdateInvResourceFields(ctx, tenantID, host, []string{
		computev1.HostResourceFieldBmcIp,
	})
}

func (c *OnboardingInventoryClient) UpdateHostIP(ctx context.Context, tenantID string, resourceID string,
	hostIP string,
) error {
	h := &computev1.HostResource{
		ResourceId: resourceID,
		BmcIp:      hostIP,
	}

	err := c.updateHostIP(ctx, tenantID, h)
	if err != nil {
		return err
	}

	return nil
}

func (c *OnboardingInventoryClient) UpdateInstance(ctx context.Context, tenantID string, instanceID string,
	currentState computev1.InstanceState,
	provisioningStatus inv_status.ResourceStatus,
	currentOS *osv1.OperatingSystemResource,
) error {
	updateInstance := &computev1.InstanceResource{
		ResourceId:                  instanceID,
		CurrentState:                currentState,
		ProvisioningStatus:          provisioningStatus.Status,
		ProvisioningStatusIndicator: provisioningStatus.StatusIndicator,
		ProvisioningStatusTimestamp: uint64(time.Now().Unix()), // #nosec G115
		CurrentOs:                   currentOS,
	}

	return c.UpdateInvResourceFields(ctx, tenantID, updateInstance, []string{
		computev1.InstanceResourceFieldCurrentState,
		computev1.InstanceResourceFieldProvisioningStatus,
		computev1.InstanceResourceFieldProvisioningStatusIndicator,
		computev1.InstanceResourceFieldProvisioningStatusTimestamp,
		computev1.InstanceResourceEdgeCurrentOs,
	})
}

func (c *OnboardingInventoryClient) DeleteInstanceResource(ctx context.Context, tenantID, resourceID string) error {
	inst := &computev1.InstanceResource{
		ResourceId:   resourceID,
		CurrentState: computev1.InstanceState_INSTANCE_STATE_DELETED,
	}

	err := c.UpdateInstanceCurrentState(ctx, tenantID, inst)
	if err != nil {
		return err
	}

	return nil
}

func (c *OnboardingInventoryClient) DeleteResource(ctx context.Context, tenantID, resourceID string) error {
	zlog.Debug().Msgf("Delete resource: %v", resourceID)

	ctx, cancel := context.WithTimeout(ctx, *inventoryTimeout)
	defer cancel()
	_, err := c.Client.Delete(ctx, tenantID, resourceID)
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

func (c *OnboardingInventoryClient) CreateOSResource(ctx context.Context, tenantID string,
	os *osv1.OperatingSystemResource,
) (string, error) {
	return c.createResource(ctx, tenantID, &inv_v1.Resource{
		Resource: &inv_v1.Resource_Os{
			Os: os,
		},
	})
}

func (c *OnboardingInventoryClient) GetOSResourceByResourceID(ctx context.Context, tenantID string,
	resourceID string,
) (*osv1.OperatingSystemResource, error) {
	resp, err := c.getResourceByID(ctx, tenantID, resourceID)
	if err != nil {
		return nil, err
	}

	inst := resp.GetResource().GetOs()

	if validateErr := validator.ValidateMessage(inst); validateErr != nil {
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

func (c *OnboardingInventoryClient) FindAllResources(ctx context.Context,
	kinds []inv_v1.ResourceKind,
) ([]*client.ResourceTenantIDCarrier, error) {
	var allResources []*client.ResourceTenantIDCarrier
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

func (c *OnboardingInventoryClient) GetProviderResources(ctx context.Context) ([]*provider_v1.ProviderResource, error) {
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
func (c *OnboardingInventoryClient) DeleteIPAddress(ctx context.Context, tenantID, resourceID string) error {
	zlog.Debug().Msgf("Delete IPAddress: %v", resourceID)

	ctx, cancel := context.WithTimeout(ctx, *inventoryTimeout)
	defer cancel()
	ipAddress := &network_v1.IPAddressResource{
		ResourceId:   resourceID,
		CurrentState: network_v1.IPAddressState_IP_ADDRESS_STATE_DELETED,
	}

	err := c.UpdateInvResourceFields(ctx, tenantID, ipAddress, []string{network_v1.IPAddressResourceFieldCurrentState})
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

func (c *OnboardingInventoryClient) listAndReturnProvider(
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
	c *OnboardingInventoryClient,
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
	DefaultOs       string `json:"defaultOs"`
	AutoProvision   bool   `json:"autoProvision"`
	CustomerID      string `json:"customerID"`
	ENProductKeyIDs string `json:"enProductKeyIDs"`
}

//nolint:tagliatelle // Renaming the json keys may effect while unmarshalling/marshaling so, used nolint.
type LicenseProviderConfig struct {
	CustomerID      string `json:"customerID"`
	ENProductKeyIDs string `json:"enProductKeyIDs"`
}

func (c *OnboardingInventoryClient) GetProviderConfig(
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

func (c *OnboardingInventoryClient) GetLicenseProviderConfig(
	ctx context.Context,
	name string,
) (*LicenseProviderConfig, error) {
	// Get the provider resource by name
	provider, err := GetProviderResourceByName(ctx, c, name)
	if err != nil {
		return nil, err
	}

	var pconf LicenseProviderConfig
	// Unmarshal provider config JSON into pconf
	if err := json.Unmarshal([]byte(provider.Config), &pconf); err != nil {
		zlog.MiErr(err).Msgf("failed to unmarshal ProviderConfig")
		return nil, inv_errors.Wrap(err)
	}

	return &pconf, nil
}

func (c *OnboardingInventoryClient) UpdateHostCurrentState(ctx context.Context, tenantID string, resourceID string,
	hostCurrentState computev1.HostState,
) error {
	h := &computev1.HostResource{
		ResourceId:   resourceID,
		CurrentState: hostCurrentState,
	}

	err := c.updateHostCurrentState(ctx, tenantID, h)
	if err != nil {
		return err
	}

	return nil
}

func (c *OnboardingInventoryClient) UpdateHostCurrentStateNOnboardStatus(ctx context.Context, tenantID string, resourceID string,
	hostCurrentState computev1.HostState, onboardingStatus inv_status.ResourceStatus,
) error {
	updateHost := &computev1.HostResource{
		ResourceId:                resourceID,
		CurrentState:              hostCurrentState,
		OnboardingStatus:          onboardingStatus.Status,
		OnboardingStatusIndicator: onboardingStatus.StatusIndicator,
		OnboardingStatusTimestamp: uint64(time.Now().Unix()), // #nosec G115
	}

	return c.UpdateInvResourceFields(ctx, tenantID, updateHost, []string{
		computev1.HostResourceFieldCurrentState,
		computev1.HostResourceFieldOnboardingStatus,
		computev1.HostResourceFieldOnboardingStatusIndicator,
		computev1.HostResourceFieldOnboardingStatusTimestamp,
	})
}

func (c *OnboardingInventoryClient) GetHostResourceBySerailNumber(
	ctx context.Context,
	serialNumber string,
) (*computev1.HostResource, error) {
	// additional check for length is needed because .Parse() accepts other non-standard format (see function docs).
	if len(serialNumber) != MaxSerialNumberLength {
		return nil, inv_errors.Errorfc(codes.InvalidArgument, "invalid SerialNumber")
	}

	filter := &inv_v1.ResourceFilter{
		Resource: &inv_v1.Resource{
			Resource: &inv_v1.Resource_Host{},
		},
		Filter: fmt.Sprintf("%s = %q", computev1.HostResourceFieldSerialNumber, serialNumber),
	}

	return c.listAndReturnHost(ctx, filter)
}
