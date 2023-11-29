package maestro

import (
	"context"
	"sync"

	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	inv_client "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/client"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/util"
	"github.com/intel-sandbox/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/pkg/logger"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

var log = logger.GetLogger()

func NewInventoryClient(ctx context.Context, wg *sync.WaitGroup, addr string) (inv_client.InventoryClient, chan *inv_client.WatchEvents, error) {
	log.Info("Init Inv client")
	resourceKinds := []inv_v1.ResourceKind{
		inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE,
		inv_v1.ResourceKind_RESOURCE_KIND_HOST,
		inv_v1.ResourceKind_RESOURCE_KIND_OS,
	}
	eventCh := make(chan *inv_client.WatchEvents)

	cfg := inv_client.InventoryClientConfig{
		Name:                      "onboarding_manager",
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

	client, err := inv_client.NewInventoryClient(ctx, cfg)
	if err != nil {
		log.Errorf("Failed to create new inventory client %v", err)
		return nil, nil, err
	}
	return client, eventCh, nil
}

func FindAllResources(ctx context.Context, c inv_client.InventoryClient, kind inv_v1.ResourceKind) ([]string, error) {
	fmk := &fieldmaskpb.FieldMask{Paths: []string{}}
	res, err := util.GetResourceFromKind(kind)
	if err != nil {
		return nil, err
	}

	resources, err := c.FindAll(ctx, res, fmk)
	if err != nil {
		return nil, err
	}

	return resources, nil
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
		return "", err
	}
	return res.ResourceId, nil
}

func CreateInstanceResource(ctx context.Context, c inv_client.InventoryClient, inst *computev1.InstanceResource) (string, error) {
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
	resources, err := listAllResources(ctx, c, filter)
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
	resources, err := listAllResources(ctx, c, filter)
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
		"kind",
		"description",
		"current_state",
		"hardware_kind",
		"memory_bytes",
		"cpu_model",
		"cpu_sockets",
		"cpu_cores",
		"cpu_capabilities",
		"cpu_architecture",
		"cpu_threads",
		"gpu_pci_id",
		"gpu_product",
		"gpu_vendor",
		"mgmt_ip",
		"bmc_kind",
		"bmc_ip",
		"bmc_username",
		"bmc_password",
		"pxe_mac",
		"hostname",
		"product_name",
		"bios_version",
		"bios_release_date",
		"bios_vendor",
		"metadata",
	})
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
	resources, err := listAllResources(ctx, c, filter)
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

func listAllResources(
	ctx context.Context,
	c inv_client.InventoryClient,
	filter *inv_v1.ResourceFilter,
) ([]*inv_v1.Resource, error) {
	objs, err := c.ListAll(ctx, filter.GetResource(), filter.GetFieldMask())
	if err != nil && !inv_errors.IsNotFound(err) {
		return nil, err
	}
	for _, v := range objs {
		if err = v.ValidateAll(); err != nil {
			return nil, inv_errors.Wrap(err)
		}
	}
	return objs, nil
}
