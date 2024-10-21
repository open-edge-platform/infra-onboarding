// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package artifact

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	google_rpc "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/util"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/api"
	om_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/status"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/compute/v1"
	inventoryv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/inventory/v1"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/os/v1"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/logging"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/policy/rbac"
	inv_tenant "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/tenant"
)

const (
	DefaultTimeout = 3 * time.Second
)

var (
	name = "NodeArtifactService"
	zlog = logging.GetLogger(name)

	hostResID string

	UpdateHostFieldmask = &fieldmaskpb.FieldMask{
		Paths: []string{
			computev1.HostResourceFieldBmcKind,
			computev1.HostResourceFieldBmcIp,
			computev1.HostResourceFieldSerialNumber,
			computev1.HostResourceFieldPxeMac,
		},
	}
	CompareHostFieldmask = &fieldmaskpb.FieldMask{
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
	NodeArtifactService struct {
		pb.UnimplementedNodeArtifactServiceNBServer
		invClient *invclient.OnboardingInventoryClient
		// TODO: remove this later https://jira.devtools.intel.com/browse/LPIO-1829
		invClientAPI *invclient.OnboardingInventoryClient
		rbac         *rbac.Policy
		authEnabled  bool
	}
)

// NewArtifactService is a constructor function.
func NewArtifactService(invClient *invclient.OnboardingInventoryClient, inventoryAdr string, enableTracing bool,
	enableAuth bool, rbacRules string,
) (*NodeArtifactService, error) {
	if invClient == nil {
		return nil, inv_errors.Errorf("invClient is nil in NewArtifactService")
	}

	var rbacPolicy *rbac.Policy
	var err error
	if enableAuth {
		zlog.Info().Msgf("Authentication is enabled, starting RBAC server for Artifact Service")
		// start OPA server with policies
		rbacPolicy, err = rbac.New(rbacRules)
		if err != nil {
			zlog.Fatal().Msg("Failed to start RBAC OPA server")
		}
	}

	var invClientAPI *invclient.OnboardingInventoryClient
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
		rbac:         rbacPolicy,
		authEnabled:  enableAuth,
	}, nil
}

func CopyNodeReqToNodeData(payload []*pb.NodeData, tenantID string) ([]*computev1.HostResource, error) {
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
				TenantId:     tenantID,
			}
			zlog.Debug().Msgf("Adding HostResource: %v", hostres)
			hosts = append(hosts, hostres)
		}
	}

	zlog.Debug().Msgf("Generates a list of hosts of length=%d", len(hosts))

	return hosts, nil
}

// sendStreamErrorResponse to send an error response on the stream.
func sendStreamErrorResponse(stream pb.NodeArtifactServiceNB_OnboardNodeStreamServer,
	code codes.Code, message string,
) error {
	response := &pb.OnboardStreamResponse{
		Status: &google_rpc.Status{
			Code:    int32(code), // #nosec G115
			Message: message,
		},
		NodeState: pb.OnboardStreamResponse_UNSPECIFIED,
	}
	return sendOnboardStreamResponse(stream, response)
}

// sendOnboardStreamResponse send a response on the stream.
func sendOnboardStreamResponse(stream pb.NodeArtifactServiceNB_OnboardNodeStreamServer,
	response *pb.OnboardStreamResponse,
) error {
	if err := stream.Send(response); err != nil {
		zlog.Error().Err(err).Msg("Failed to send response on the stream")
		return err
	}
	return nil
}

// receiveFromStream receive a message from the stream.
func (s *NodeArtifactService) receiveFromStream(stream pb.NodeArtifactServiceNB_OnboardNodeStreamServer) (
	*pb.OnboardStreamRequest, error,
) {
	zlog.Info().Msgf("OnboardNodeStream started: receiveFromStream")
	req, err := stream.Recv()
	if errors.Is(err, io.EOF) {
		zlog.Info().Msgf("OnboardNodeStream client has closed the stream")
		return nil, io.EOF
	}
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("OnboardNodeStream error receiving from stream: %v", err)
		return nil, err
	}
	return req, nil
}

// handleRegisteredState  processes the REGISTERED state.
func (s *NodeArtifactService) handleRegisteredState(stream pb.NodeArtifactServiceNB_OnboardNodeStreamServer,
	hostInv *computev1.HostResource,
) error {
	response := &pb.OnboardStreamResponse{
		Status:    &google_rpc.Status{Code: int32(codes.OK)},
		NodeState: pb.OnboardStreamResponse_REGISTERED,
	}
	if err := sendOnboardStreamResponse(stream, response); err != nil {
		return err
	}
	// make the current state to registered
	err := s.invClient.UpdateHostCurrentState(context.Background(), hostInv.GetTenantId(),
		hostInv.ResourceId, computev1.HostState_HOST_STATE_REGISTERED)
	if err != nil {
		return err
	}
	return nil
}

// handleRegisteredState processes the ONBOARDED state.
func (s *NodeArtifactService) handleOnboardedState(stream pb.NodeArtifactServiceNB_OnboardNodeStreamServer,
	hostInv *computev1.HostResource,
) error {
	clientID, clientSecret, err := utils.FetchClientSecret(context.Background(), hostInv.GetTenantId(), hostInv.Uuid)
	if err != nil {
		zlog.Error().Err(err).Msg("Failed to fetch client id and secret from keycloak")
		return err
	}
	zlog.Info().Msgf("Host Desired state : %v\n, client ID: %v\n client secret: %v\n",
		hostInv.DesiredState, clientID, clientSecret)
	if err := sendOnboardStreamResponse(stream, &pb.OnboardStreamResponse{
		NodeState:    pb.OnboardStreamResponse_ONBOARDED,
		Status:       &google_rpc.Status{Code: int32(codes.OK)},
		ClientId:     clientID,
		ClientSecret: clientSecret,
	}); err != nil {
		zlog.Error().Err(err).Msg("Failed to send response client Id and secret on the stream")
		return err
	}
	// make the current state to ONBOARDED.
	errUpdatehostStatus := s.invClient.UpdateHostCurrentState(context.Background(), hostInv.GetTenantId(),
		hostInv.ResourceId, computev1.HostState_HOST_STATE_ONBOARDED)
	if errUpdatehostStatus != nil {
		zlog.Error().Err(errUpdatehostStatus).Msg("Failed to update host current status to ONBOARDED")
		return errUpdatehostStatus
	}
	// closes the stream after sending the final response
	return nil
}

// handleDefaultState processes the UNSPECIFIED state.
func (s *NodeArtifactService) handleDefaultState(stream pb.NodeArtifactServiceNB_OnboardNodeStreamServer) error {
	return sendOnboardStreamResponse(stream, &pb.OnboardStreamResponse{
		Status: &google_rpc.Status{
			Code:    int32(codes.Unknown),
			Message: "The node state is unspecified or unknown.",
		},
		NodeState: pb.OnboardStreamResponse_UNSPECIFIED,
	})
}

//nolint:funlen,cyclop // reason: function is long due to necessary logic; cyclomatic complexity is high due to necessary handling
func (s *NodeArtifactService) OnboardNodeStream(stream pb.NodeArtifactServiceNB_OnboardNodeStreamServer) error {
	zlog.Info().Msgf("OnboardNodeStream started")

	var hostInv *computev1.HostResource
	var startZeroTouchAfterClose bool // Flag to start zero touch after closing the stream

	// Start zero-touch process when the stream closes
	defer func() {
		if startZeroTouchAfterClose && hostInv != nil {
			go func() {
				ctx := context.Background()
				// Start the zero-touch process.
				if err := s.startZeroTouch(ctx, hostInv.GetTenantId(), hostInv.GetResourceId()); err != nil {
					zlog.Error().Err(err).Msg("Failed to start zero touch process")
				}
			}()
		}
	}()

	for {
		// Receive a message from the stream
		req, err := s.receiveFromStream(stream)
		if err != nil {
			return err
		}

		// Validate the stream request using the generated Validate method
		if reqValidateerr := req.Validate(); reqValidateerr != nil {
			return sendStreamErrorResponse(stream, codes.InvalidArgument, reqValidateerr.Error())
		}

		// Check if the Serial Number already exists when UUID is empty
		if req.Uuid == "" {
			hostResource, errGetSN := s.invClient.GetHostResourceBySerailNumber(context.Background(),
				req.Serialnum)
			if errGetSN == nil {
				// If serial number exists in the system.
				zlog.Info().Msgf("Node %v exists for serial number %v", hostResource.Uuid, req.Serialnum)
				return sendStreamErrorResponse(stream, codes.AlreadyExists, "Serial number already exists")
			}
			zlog.Error().Err(errGetSN).Msgf("Error retrieving host resource by serial number: %v", req.Serialnum)
			return sendStreamErrorResponse(stream, codes.Internal, errGetSN.Error())
		}

		// Process the request based on sequence diagram
		// Process the request and look up the host state using
		// the UUID from the request.

		// 1. If the UUID provided by the EN is not found in the inventory
		hostInv, err = s.invClient.GetHostResourceByUUID(context.Background(), req.Uuid)
		tenantID := hostInv.GetTenantId()
		hostInv.CurrentState = computev1.HostState_HOST_STATE_ONBOARDED
		hostInv.OnboardingStatus = om_status.OnboardingStatusDone.Status
		hostInv.OnboardingStatusIndicator = om_status.OnboardingStatusDone.StatusIndicator
		hostInv.OnboardingStatusTimestamp = uint64(time.Now().Unix()) // #nosec G115

		if err != nil {
			zlog.Info().Msgf("Node Doesn't Exist for UUID %v\n", req.Uuid)
			// The server sends the Device "NotFound" grpc code over the stream
			// and continues to keep the stream open
			// The EN close the stream from its side and then proceed to invoke
			// the unary gRPC API for IO processing
			if errdevNotFound := sendStreamErrorResponse(stream, codes.NotFound, err.Error()); errdevNotFound != nil {
				zlog.Error().Err(errdevNotFound).Msg("Failed to send 'NotFound' error response on the stream")
				return errdevNotFound
			}
			continue // The stream remains open, waiting for the client to close it
		}

		// Only update MAC ID if it's empty or differs from the incoming request
		if hostInv.PxeMac == "" || hostInv.PxeMac != req.GetMacId() {
			macAddress := req.GetMacId()
			errupdatemacStatus := s.invClient.UpdateHostMacID(context.Background(), tenantID, hostInv.ResourceId, macAddress)
			if errupdatemacStatus != nil {
				zlog.Error().Err(errupdatemacStatus).Msg("Failed to update host MAC ID")
				return errupdatemacStatus
			}
			zlog.Info().Msgf("MAC ID updated to %s for resource %s", macAddress, hostInv.ResourceId)
		} else {
			zlog.Info().Msgf("MAC ID %s already set for resource %s, skipping update.",
				hostInv.PxeMac, hostInv.ResourceId)
		}

		// Only update hostip if it's empty or differs from the incoming request
		if hostInv.BmcIp == "" || hostInv.BmcIp != req.GetHostIp() {
			bmcIP := req.GetHostIp()
			errUpdateBmcStatus := s.invClient.UpdateHostIP(context.Background(), tenantID, hostInv.ResourceId, bmcIP)
			if errUpdateBmcStatus != nil {
				zlog.Error().Err(errUpdateBmcStatus).Msg("Failed to update host BMC IP")
				return errUpdateBmcStatus
			}
			zlog.Info().Msgf("Host IP updated to %s for resource %s", bmcIP, hostInv.ResourceId)
		} else {
			zlog.Info().Msgf("Host IP %s already set for resource %s, skipping update.",
				hostInv.BmcIp, hostInv.ResourceId)
		}

		// 2. If the UUID is found but the current state is ONBOARDED or ERROR,
		// the OM sends a FAILED_PRECONDITION
		if hostInv.CurrentState == computev1.HostState_HOST_STATE_ONBOARDED ||
			hostInv.CurrentState == computev1.HostState_HOST_STATE_ERROR {
			zlog.Info().Msgf("Node already exists for UUID %v and node current state %v",
				req.Uuid, hostInv.CurrentState)
			// Send a failure response indicating the node is already onboarded or provisioned.
			return sendStreamErrorResponse(stream, codes.FailedPrecondition,
				fmt.Sprintf("Node is already %s", hostInv.CurrentState.String()))
		}

		zlog.Info().Msgf("Node %v exists in inventory. Desired state: %v, Current state: %v",
			hostInv.Uuid, hostInv.DesiredState, hostInv.CurrentState)

		// 3. If the DesiredState is not REGISTERED, or ONBOARDED,
		// the OM sends a FAILURE_UNSPECIFIED response and returns an error, closing the stream

		// send the response to EN based on host Desired state
		switch hostInv.DesiredState {
		// The host is in the REGISTERED state.
		// Allow the EN to retry but do not close the stream.
		// Assume SI initalially configure desiredstate as REGISTERED
		case computev1.HostState_HOST_STATE_REGISTERED:
			if err := s.handleRegisteredState(stream, hostInv); err != nil {
				return err
			}
			// continue to keep the stream open when the EN is in the REGISTERED state,
			// allowing for retries without closing the stream
			continue

		case computev1.HostState_HOST_STATE_ONBOARDED:
			/*
				If the DesiredState is ONBOARDED, the server proceeds with onboarding,
				communicates with Keycloak to create EN secrets, sends a SUCCESS response
				with the client_id and client_secret, and then returns nil, closing the stream
			*/
			if err := s.handleOnboardedState(stream, hostInv); err != nil {
				return err
			}
			startZeroTouchAfterClose = true
			return nil // Close the stream

		default:
			// For other states, send an error.
			if err := s.handleDefaultState(stream); err != nil {
				return err
			}
			return nil // Close the stream
		}
	}
}

//nolint:cyclop // May effect the functionality now, need to simplify this in future
func (s *NodeArtifactService) CreateNodes(ctx context.Context, req *pb.NodeRequest) (*pb.NodeResponse, error) {
	zlog.Info().Msgf("CreateNodes")
	if s.authEnabled {
		// checking if JWT contains write permission
		if !s.rbac.IsRequestAuthorized(ctx, rbac.CreateKey) {
			err := inv_errors.Errorfc(codes.PermissionDenied, "Request is blocked by RBAC")
			zlog.MiSec().MiErr(err).Msgf("Request CreateNodes is not authenticated")
			return nil, err
		}
	}

	tenantID, present := inv_tenant.GetTenantIDFromContext(ctx)
	if !present {
		// This should never happen! Interceptor should either fail or set it!
		err := inv_errors.Errorfc(codes.Unauthenticated, "Tenant ID is not present in context")
		zlog.MiSec().MiErr(err).Msg("Request CreateNodes is not authenticated")
		return nil, err
	}
	zlog.Debug().Msgf("CreateNodes: tenantID=%s", tenantID)

	/* Copy node data from user */
	hostresdata, err := CopyNodeReqToNodeData(req.Payload, tenantID)
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
	// IO path - set the current state to ONBOARDED
	host.CurrentState = computev1.HostState_HOST_STATE_ONBOARDED
	host.OnboardingStatus = om_status.OnboardingStatusDone.Status
	host.OnboardingStatusIndicator = om_status.OnboardingStatusDone.StatusIndicator
	host.OnboardingStatusTimestamp = uint64(time.Now().Unix()) // #nosec G115

	hostInv, err := s.invClient.GetHostResourceByUUID(ctx, host.Uuid)

	switch {
	case inv_errors.IsNotFound(err):
		zlog.Info().Msgf("Create op : Node Doesn't Exist for GUID %s and tID=%s\n", host.Uuid, tenantID)

	case err == nil:
		zlog.Debug().Msgf("Create op : Node and its Host Resource Already Exist for GUID %s, tID=%s \n", host.Uuid, tenantID)
		if ztErr := s.startZeroTouch(ctx, tenantID, hostInv.ResourceId); ztErr != nil {
			zlog.MiSec().MiErr(ztErr).Msgf("startZeroTouch error: %v", ztErr)
			return nil, ztErr
		}
		return &pb.NodeResponse{Payload: req.Payload}, nil

	case err != nil:
		zlog.MiSec().MiErr(err).Msgf("Create op :Failed CreateNodes() for GUID %s tID=%s \n", host.Uuid, tenantID)
		return nil, err
	}

	hostResID, err = s.invClient.CreateHostResource(ctx, tenantID, host)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("Cannot create Host resource: %v tID=%s", host, tenantID)
		return nil, err
	}
	zlog.Debug().Msgf("CreateHostResource ID = %s and tID=%s", hostResID, tenantID)

	if err := s.startZeroTouch(ctx, tenantID, hostResID); err != nil {
		zlog.MiSec().MiErr(err).Msgf("startZeroTouch error: %v", err)
		return nil, err
	}

	return &pb.NodeResponse{Payload: req.Payload}, nil
}

func (s *NodeArtifactService) DeleteNodes(ctx context.Context, req *pb.NodeRequest) (*pb.NodeResponse, error) {
	zlog.Info().Msgf("DeleteNodes")
	if s.authEnabled {
		// checking if JWT contains valid claim
		if !s.rbac.IsRequestAuthorized(ctx, rbac.DeleteKey) {
			err := inv_errors.Errorfc(codes.PermissionDenied, "Request is blocked by RBAC")
			zlog.MiSec().MiErr(err).Msgf("Request DeleteNodes is not authenticated")
			return nil, err
		}
	}

	tenantID, present := inv_tenant.GetTenantIDFromContext(ctx)
	if !present {
		// This should never happen! Interceptor should either fail or set it!
		err := inv_errors.Errorfc(codes.Unauthenticated, "Tenant ID is not present in context")
		zlog.MiSec().MiErr(err).Msg("Request DeleteNodes is not authenticated")
		return nil, err
	}

	zlog.Debug().Msgf("DeleteNodes: tenantID=%s", tenantID)
	hostresdata, err := CopyNodeReqToNodeData(req.Payload, tenantID)
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
		zlog.MiSec().MiErr(err).Msgf("Delete op : Node Doesn't Exist for GUID %s ,tID=%s\n", hostresdata[0].Uuid, tenantID)
		return &pb.NodeResponse{Payload: req.Payload}, nil

	case err == nil:
		zlog.Debug().Msgf("Delete op : Node and its Host Resource Already Exist for GUID %s ,tID=%s \n",
			hostresdata[0].Uuid, tenantID)

	case err != nil:
		zlog.MiSec().MiErr(err).Msgf("Delete op : Failed DeleteNodes() for GUID %s,tID=%s\n", hostresdata[0].Uuid, tenantID)
		return nil, err
	}

	/* copy and update the fetched resource id of the given serial number
	 *  to the pre-existing host resource
	 */
	hostResID = hostresget.ResourceId
	hostresdata[0].ResourceId = hostResID

	err = s.invClient.DeleteHostResource(ctx, tenantID, hostResID)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("\nDeleteHostResource()  Error : %v\n", err)
		return nil, err
	}

	return &pb.NodeResponse{Payload: req.Payload}, nil
}

func (s *NodeArtifactService) GetNodes(ctx context.Context, req *pb.NodeRequest) (*pb.NodeResponse, error) {
	zlog.Info().Msgf("GetNodes")
	if s.authEnabled {
		if !s.rbac.IsRequestAuthorized(ctx, rbac.GetKey) {
			err := inv_errors.Errorfc(codes.PermissionDenied, "Request is blocked by RBAC")
			zlog.MiSec().MiErr(err).Msgf("Request GetNodes is not authenticated")
			return nil, err
		}
	}

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

//nolint:cyclop // cyclomatic complexity is high due to switch statement and multiple error handling
func (s *NodeArtifactService) UpdateNodes(ctx context.Context, req *pb.NodeRequest) (*pb.NodeResponse, error) {
	zlog.Info().Msgf("UpdateNodes")
	if s.authEnabled {
		// checking if JWT contains write permissions
		if !s.rbac.IsRequestAuthorized(ctx, rbac.UpdateKey) {
			err := inv_errors.Errorfc(codes.PermissionDenied, "Request is blocked by RBAC")
			zlog.MiSec().MiErr(err).Msgf("Request UpdateNodes is not authenticated")
			return nil, err
		}
	}

	tenantID, present := inv_tenant.GetTenantIDFromContext(ctx)
	if !present {
		// This should never happen! Interceptor should either fail or set it!
		err := inv_errors.Errorfc(codes.Unauthenticated, "Tenant ID is not present in context")
		zlog.MiSec().MiErr(err).Msg("Request UpdateNodes is not authenticated")
		return nil, err
	}

	zlog.Debug().Msgf("UpdateNodes: tenantID=%s", tenantID)

	host, err := CopyNodeReqToNodeData(req.Payload, tenantID)
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
		zlog.MiSec().MiErr(err).Msgf("Update op : Node Doesn't Exist for GUID %s,tID=%s\n", host[0].Uuid, tenantID)
		return nil, err

	case err == nil:
		zlog.Debug().Msgf("Update op : Node and its Host Resource Already Exist for GUID %s ,tID=%s\n", host[0].Uuid, tenantID)

	case err != nil:
		zlog.MiSec().MiErr(err).Msgf("Update op : Failed CreateNodes() for GUID %s,tID=%s\n", host[0].Uuid, tenantID)
		return nil, err
	}

	doHostUpdate := false
	isSameHost, err := util.IsSameHost(hostInv, host[0], CompareHostFieldmask)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("Failed to compare Host resources, continuing to do update anyway")
		doHostUpdate = true
	}

	if !isSameHost || doHostUpdate {
		host[0].ResourceId = hostInv.GetResourceId()
		err = s.invClient.UpdateInvResourceFields(ctx, tenantID, host[0], UpdateHostFieldmask.Paths)
		if err != nil {
			zlog.MiSec().MiErr(err).Msgf("UpdateNodes() : UpdateHostResource() Error : %v", err)
			return nil, err
		}
	} else {
		zlog.Debug().Msgf("Skipping to update Host resource due to no changes. "+
			"Original Host: %v, Updated Host: %v ,tID=%s", hostInv, host[0], tenantID)
	}

	return &pb.NodeResponse{Payload: req.Payload}, nil
}

func (s *NodeArtifactService) startZeroTouch(ctx context.Context, tenantID, hostResID string) error {
	zlog.Info().Msgf("Starting zero touch for host ID %s  tenant ID %s...", hostResID, tenantID)

	host, err := s.invClient.GetHostResourceByResourceID(ctx, tenantID, hostResID)
	if err != nil {
		zlog.Err(err).Msgf("No host found with resource ID %s,,tID=%s", hostResID, tenantID)
		return err // Return the error to the caller
	}

	// Check if an instance has already been created for the host
	if host.Instance != nil {
		zlog.Debug().Msgf("An Instance (%s) is already created for a host %s ,tID=%s",
			host.GetInstance().GetResourceId(), host.GetResourceId(), tenantID)
		return nil
	}

	// TODO : Passing default provider name while trying to provision, need to change according to provider name and compare.
	pconf, err := s.invClient.GetProviderConfig(ctx, utils.DefaultProviderName)
	if err != nil {
		zlog.Err(err).Msgf("Failed to get provider configuration")
		return nil
	}

	// if AutoProvision is set, create an Instance for the Host with the OS set to the value of the default OS
	return s.checkNCreateInstance(ctx, tenantID, *pconf, host)
}

func (s *NodeArtifactService) checkNCreateInstance(ctx context.Context, tenantID string,
	pconf invclient.ProviderConfig, host *computev1.HostResource,
) error {
	if pconf.AutoProvision {
		instance := &computev1.InstanceResource{
			TenantId: tenantID,

			Kind:         computev1.InstanceKind_INSTANCE_KIND_METAL,
			DesiredState: computev1.InstanceState_INSTANCE_STATE_RUNNING,
			CurrentState: computev1.InstanceState_INSTANCE_STATE_UNSPECIFIED,

			Host: &computev1.HostResource{
				ResourceId: host.ResourceId,
			},
			DesiredOs: &osv1.OperatingSystemResource{
				ResourceId: pconf.DefaultOs,
			},

			SecurityFeature: osv1.SecurityFeature_SECURITY_FEATURE_SECURE_BOOT_AND_FULL_DISK_ENCRYPTION,
		}
		if _, err := s.invClientAPI.CreateInstanceResource(ctx, tenantID, instance); err != nil {
			zlog.Err(err).Msgf("Failed to CreateInstanceResource for host resource (uuid=%s),tID=%s", hostResID, tenantID)
			return err
		}
	}

	return nil
}
