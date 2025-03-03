// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	google_rpc "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	computev1 "github.com/intel/infra-core/inventory/v2/pkg/api/compute/v1"
	inventoryv1 "github.com/intel/infra-core/inventory/v2/pkg/api/inventory/v1"
	osv1 "github.com/intel/infra-core/inventory/v2/pkg/api/os/v1"
	inv_errors "github.com/intel/infra-core/inventory/v2/pkg/errors"
	"github.com/intel/infra-core/inventory/v2/pkg/logging"
	"github.com/intel/infra-core/inventory/v2/pkg/policy/rbac"
	inv_status "github.com/intel/infra-core/inventory/v2/pkg/status"
	inv_tenant "github.com/intel/infra-core/inventory/v2/pkg/tenant"
	"github.com/intel/infra-onboarding/onboarding-manager/internal/invclient"
	"github.com/intel/infra-onboarding/onboarding-manager/internal/onboarding"
	onboarding_types "github.com/intel/infra-onboarding/onboarding-manager/internal/onboarding/types"
	pb "github.com/intel/infra-onboarding/onboarding-manager/pkg/api"
	om_status "github.com/intel/infra-onboarding/onboarding-manager/pkg/status"
)

const (
	DefaultTimeout = 3 * time.Second
)

var (
	name = "InteractiveOnboardingService"
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

type InventoryClientService struct {
	invClient    *invclient.OnboardingInventoryClient
	invClientAPI *invclient.OnboardingInventoryClient
}

type (
	InteractiveOnboardingService struct {
		pb.UnimplementedInteractiveOnboardingServiceServer
		InventoryClientService
		rbac        *rbac.Policy
		authEnabled bool
	}
)

type (
	NonInteractiveOnboardingService struct {
		pb.UnimplementedNonInteractiveOnboardingServiceServer
		InventoryClientService
	}
)

// NewInteractiveOnboardingService to start the gRPC server - IO.
func NewInteractiveOnboardingService(invClient *invclient.OnboardingInventoryClient, inventoryAdr string, enableTracing bool,
	enableAuth bool, rbacRules string,
) (*InteractiveOnboardingService, error) {
	if invClient == nil {
		return nil, inv_errors.Errorf("invClient is nil in NewInteractiveOnboardingService")
	}

	var rbacPolicy *rbac.Policy
	var err error
	if enableAuth {
		zlog.Info().Msgf("Authentication is enabled, starting RBAC server for InteractiveOnboarding Service")
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
		// TODO: remove this later https://jira.devtools.intel.com/browse/ITEP-1829
		invClientAPI, err = invclient.NewOnboardingInventoryClientWithOptions(
			invclient.WithInventoryAddress(inventoryAdr),
			invclient.WithEnableTracing(enableTracing),
			invclient.WithClientKind(inventoryv1.ClientKind_CLIENT_KIND_API),
		)
		if err != nil {
			return nil, inv_errors.Errorf("Unable to start onboarding inventory API server client %v", err)
		}
	}

	return &InteractiveOnboardingService{
		InventoryClientService: InventoryClientService{
			invClient:    invClient,
			invClientAPI: invClientAPI,
		},
		rbac:        rbacPolicy,
		authEnabled: enableAuth,
	}, nil
}

// NewNonInteractiveOnboardingService to start the gRPC server - NIO.
func NewNonInteractiveOnboardingService(invClient *invclient.OnboardingInventoryClient, inventoryAdr string,
	enableTracing bool,
) (*NonInteractiveOnboardingService, error) {
	if invClient == nil {
		return nil, inv_errors.Errorf("invClient is nil in NonInteractiveOnboardingService")
	}

	var invClientAPI *invclient.OnboardingInventoryClient
	var err error
	if inventoryAdr == "" {
		zlog.Warn().Msg("Unable to start onboarding inventory API server client, empty inventory address")
	} else {
		// TODO: remove this later https://jira.devtools.intel.com/browse/ITEP-1829
		invClientAPI, err = invclient.NewOnboardingInventoryClientWithOptions(
			invclient.WithInventoryAddress(inventoryAdr),
			invclient.WithEnableTracing(enableTracing),
			invclient.WithClientKind(inventoryv1.ClientKind_CLIENT_KIND_API),
		)
		if err != nil {
			return nil, inv_errors.Errorf("Unable to start onboarding inventory API server client %v", err)
		}
	}
	return &NonInteractiveOnboardingService{
		InventoryClientService: InventoryClientService{
			invClient:    invClient,
			invClientAPI: invClientAPI,
		},
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
func sendStreamErrorResponse(stream pb.NonInteractiveOnboardingService_OnboardNodeStreamServer,
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
func sendOnboardStreamResponse(stream pb.NonInteractiveOnboardingService_OnboardNodeStreamServer,
	response *pb.OnboardStreamResponse,
) error {
	if err := stream.Send(response); err != nil {
		zlog.Error().Err(err).Msg("Failed to send response on the stream")
		return err
	}
	return nil
}

// receiveFromStream receive a message from the stream.
func (s *NonInteractiveOnboardingService) receiveFromStream(stream pb.NonInteractiveOnboardingService_OnboardNodeStreamServer) (
	*pb.OnboardStreamRequest, error,
) {
	zlog.Info().Msgf("OnboardNodeStream started: receiveFromStream")
	req, err := stream.Recv()
	if errors.Is(err, io.EOF) {
		zlog.Info().Msgf("OnboardNodeStream client has closed the stream")
		return nil, io.EOF
	}
	if err != nil {
		zlog.InfraSec().InfraErr(err).Msgf("OnboardNodeStream error receiving from stream: %v", err)
		return nil, err
	}
	return req, nil
}

// handleRegisteredState  processes the REGISTERED state.
func (s *NonInteractiveOnboardingService) handleRegisteredState(stream pb.NonInteractiveOnboardingService_OnboardNodeStreamServer,
	hostInv *computev1.HostResource, req *pb.OnboardStreamRequest,
) error {
	response := &pb.OnboardStreamResponse{
		Status:    &google_rpc.Status{Code: int32(codes.OK)},
		NodeState: pb.OnboardStreamResponse_REGISTERED,
		ProjectId: hostInv.GetTenantId(),
	}
	if err := sendOnboardStreamResponse(stream, response); err != nil {
		return err
	}

	// Update the host details, state, and registration status
	err := s.invClient.UpdateHostRegState(context.Background(),
		hostInv.GetTenantId(),
		hostInv.ResourceId,
		computev1.HostState_HOST_STATE_REGISTERED, req.HostIp,
		req.MacId,
		inv_status.New(om_status.HostRegistrationDone.Status, om_status.HostRegistrationDone.StatusIndicator))
	if err != nil {
		zlog.Error().Err(err).Msgf("Update failed for host resource id %v", hostInv.ResourceId)
		return err
	}
	return nil
}

// handleRegisteredState processes the ONBOARDED state.
func (s *NonInteractiveOnboardingService) handleOnboardedState(stream pb.NonInteractiveOnboardingService_OnboardNodeStreamServer,
	hostInv *computev1.HostResource, req *pb.OnboardStreamRequest,
) error {
	clientID, clientSecret, err := onboarding.FetchClientSecret(context.Background(), hostInv.GetTenantId(), hostInv.Uuid)
	if err != nil {
		zlog.Error().Err(err).Msg("Failed to fetch client id and secret from keycloak")
		return err
	}
	zlog.Debug().Msgf("Host Desired state : %v\n, client ID: %v\n client secret: %v\n",
		hostInv.DesiredState, clientID, clientSecret)
	if err := sendOnboardStreamResponse(stream, &pb.OnboardStreamResponse{
		NodeState:    pb.OnboardStreamResponse_ONBOARDED,
		Status:       &google_rpc.Status{Code: int32(codes.OK)},
		ClientId:     clientID,
		ClientSecret: clientSecret,
		ProjectId:    hostInv.GetTenantId(),
	}); err != nil {
		zlog.Error().Err(err).Msg("Failed to send response client Id and secret on the stream")
		return err
	}

	// make the current state to ONBOARDED and onboarding status to ONBOARDED
	errUpdatehostStatus := s.invClient.UpdateHostCurrentStateNOnboardStatus(context.Background(),
		hostInv.GetTenantId(), hostInv.ResourceId, req.HostIp, req.MacId, computev1.HostState_HOST_STATE_ONBOARDED,
		inv_status.New(om_status.OnboardingStatusDone.Status,
			om_status.OnboardingStatusDone.StatusIndicator))
	if errUpdatehostStatus != nil {
		zlog.Error().Err(errUpdatehostStatus).Msg("Failed to update host current status to ONBOARDED")
		return errUpdatehostStatus
	}
	OnboardingStatusTimestamp := uint64(time.Now().Unix()) // #nosec G115
	zlog.Info().Msgf("Instrumentation Info: Host Onboarded Successfully on %d\n", OnboardingStatusTimestamp)
	// closes the stream after sending the final response
	return nil
}

// handleDefaultState processes the UNSPECIFIED state.
func (s *NonInteractiveOnboardingService) handleDefaultState(
	stream pb.NonInteractiveOnboardingService_OnboardNodeStreamServer,
) error {
	return sendOnboardStreamResponse(stream, &pb.OnboardStreamResponse{
		Status: &google_rpc.Status{
			Code:    int32(codes.FailedPrecondition),
			Message: "The node state is unspecified",
		},
		NodeState: pb.OnboardStreamResponse_UNSPECIFIED,
	})
}

//nolint:cyclop,funlen // reason: function is long due to necessary logic; cyclomatic complexity is high due to necessary handling
func (s *NonInteractiveOnboardingService) getHostResource(req *pb.OnboardStreamRequest) (*computev1.HostResource, error) {
	var hostResource *computev1.HostResource
	var serialNumberMatch, uuidMatch bool
	var hostResourceByUUID, hostResourceBySN *computev1.HostResource

	// Check if UUID is provided
	if req.Uuid != "" {
		var errUUID error
		hostResourceByUUID, errUUID = s.invClient.GetHostResource(context.Background(), computev1.HostResourceFieldUuid, req.Uuid)
		if errUUID != nil {
			if inv_errors.IsNotFound(errUUID) {
				zlog.Debug().Msgf("Node doesn't exist for UUID: %v", req.Uuid)
				zlog.Error().Err(errUUID).Msgf("Node doesn't exist for UUID")
			} else {
				zlog.Debug().Msgf("Error retrieving host resource by UUID: %v", req.Uuid)
				zlog.Error().Err(errUUID).Msgf("Error retrieving host resource by UUID")
				return nil, inv_errors.Errorfc(codes.Internal, "Error retrieving host resource by UUID")
			}
		} else {
			uuidMatch = true
			hostResource = hostResourceByUUID
			zlog.Debug().Msgf("Node exists for UUID %v", req.Uuid)

			// Check the associated serial number
			if hostResource.SerialNumber == "" {
				zlog.Debug().Msgf("Proceeding with registration for UUID %v with no Serial Number in inventory", req.Uuid)
				return hostResource, nil
			}
		}
	}

	// Check if Serial Number is provided
	if req.Serialnum != "" {
		var errSN error
		hostResourceBySN, errSN = s.invClient.GetHostResource(
			context.Background(),
			computev1.HostResourceFieldSerialNumber,
			req.Serialnum,
		)
		if errSN != nil {
			if inv_errors.IsNotFound(errSN) {
				zlog.Debug().Msgf("Node doesn't exist for serial number: %v", req.Serialnum)
				zlog.Error().Err(errSN).Msgf("Node doesn't exist for serial number")
			} else {
				zlog.Debug().Msgf("Error retrieving host resource by serial number: %v", req.Serialnum)
				zlog.Error().Err(errSN).Msgf("Error retrieving host resource by serial number")
				return nil, inv_errors.Errorfc(codes.Internal, "Error retrieving host resource by serial number")
			}
		} else {
			serialNumberMatch = true
			if hostResource == nil {
				hostResource = hostResourceBySN
			}
			zlog.Debug().Msgf("Node exists for serial number %v", req.Serialnum)

			if hostResource.Uuid == "" {
				hostResource.Uuid = req.Uuid
				errUpdate := s.invClient.UpdateHostResource(context.Background(), hostResource.GetTenantId(), hostResource)
				if errUpdate != nil {
					zlog.Error().Err(errUpdate).Msgf("failed to updated the host resource uuid: %v", errUpdate)
					return nil, inv_errors.Errorfc(codes.Internal, "failed to updated the host resource uuid")
				}
				zlog.Debug().Msgf("Proceeding with registration for Serial Number %v with no UUID in inventory", req.Serialnum)
				return hostResource, nil
			}
		}
	}

	// Handle mismatches between the two resources
	if uuidMatch && serialNumberMatch {
		// Ensure both resources are not nil
		if hostResourceByUUID != nil && hostResourceBySN != nil {
			if hostResourceByUUID.ResourceId != hostResourceBySN.ResourceId {
				zlog.Debug().Msgf("Mismatch: UUID %v and Serial Number %v refer to different resources", req.Uuid, req.Serialnum)
				return nil, inv_errors.Errorfc(codes.InvalidArgument, "UUID and Serial Number refer to different resources")
			}
			// Set hostResource to one of them (either works)
			hostResource = hostResourceByUUID // or hostResourceBySN, both are the same in this case
		} else {
			zlog.Debug().Msg("One of the resources is nil while checking for UUID and Serial Number match")
			return nil, inv_errors.Errorfc(codes.Internal, "Error: One of the host resources is nil")
		}
	}

	// Handle cases based on matches found
	if (uuidMatch && !serialNumberMatch) || (!uuidMatch && serialNumberMatch) {
		var detail string
		var status inv_status.ResourceStatus
		var errorType string

		if !serialNumberMatch {
			detail = req.Serialnum
			status = om_status.HostRegistrationSerialNumFailedWithDetails(detail)
			errorType = computev1.HostResourceFieldSerialNumber
			zlog.Error().Msgf("Node doesn't exist for serial number: %v", detail)
		} else {
			detail = req.Uuid
			status = om_status.HostRegistrationUUIDFailedWithDetails(detail)
			errorType = computev1.HostResourceFieldUuid
			zlog.Debug().Msgf("Node doesn't exist for UUID: %v", detail)
			zlog.Error().Msgf("Node doesn't exist for UUID")
		}

		// Update host details if hostResource is not nil
		if hostResource != nil {
			if updateErr := s.invClient.UpdateHostRegState(context.Background(), hostResource.GetTenantId(),
				hostResource.ResourceId, hostResource.CurrentState, "", "", status,
			); updateErr != nil {
				return nil, updateErr
			}
		}
		// Return a NotFound error with relevant details
		return nil, inv_errors.Errorfc(codes.NotFound, "Node doesn't exist for %s: %v", errorType, detail)
	}

	if uuidMatch && serialNumberMatch {
		zlog.Debug().Msgf("Both UUID and Serial Number match: %v", hostResource)
		return hostResource, nil // Return the matched host resource
	}

	// If both UUID and serial number are not found
	if !uuidMatch && !serialNumberMatch {
		zlog.Info().Msg("Device not found for provided UUID and Serial Number")
		return nil, inv_errors.Errorfc(
			codes.NotFound,
			"Device not found for both UUID and Serial Number",
		)
	}

	return hostResource, nil
}

//nolint:cyclop,funlen // reason: cyclomatic complexity is high due to necessary handling
func (s *NonInteractiveOnboardingService) OnboardNodeStream(
	stream pb.NonInteractiveOnboardingService_OnboardNodeStreamServer,
) error {
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

		// Retrieves the host resource based on UUID or Serial Number.
		hostInv, err = s.getHostResource(req)
		if err != nil {
			if inv_errors.IsNotFound(err) {
				zlog.Error().Err(err).Msg("Device not found")
				if errdevNotFound := sendStreamErrorResponse(stream, codes.NotFound,
					"Device not found"); errdevNotFound != nil {
					zlog.Error().Err(errdevNotFound).Msg("Failed to send 'NotFound' error response on the stream")
					return errdevNotFound
				}
				continue
			}
			zlog.Error().Err(err).Msg("Internal server error")
			if errInternal := sendStreamErrorResponse(stream, codes.Internal,
				"Internal server error"); errInternal != nil {
				zlog.Error().Err(errInternal).Msg("Failed to send 'Internal' error response on the stream")
				return errInternal
			}
			return nil // Close the stream
		}

		// 2. If the UUID is found but the current state is ONBOARDED or ERROR,
		// the OM sends a FAILED_PRECONDITION
		if hostInv.CurrentState == computev1.HostState_HOST_STATE_ONBOARDED ||
			hostInv.CurrentState == computev1.HostState_HOST_STATE_ERROR {
			zlog.Debug().Msgf("Node already exists for UUID %v and node current state %v",
				req.Uuid, hostInv.CurrentState)
			// Send a failure response indicating the node is already onboarded or provisioned.
			return sendStreamErrorResponse(stream, codes.FailedPrecondition,
				fmt.Sprintf("Node is already %s", hostInv.CurrentState.String()))
		}

		zlog.Debug().Msgf("Node %v exists in inventory. Desired state: %v, Current state: %v",
			hostInv.Uuid, hostInv.DesiredState, hostInv.CurrentState)

		// 3. If the DesiredState is not REGISTERED, or ONBOARDED,
		// the OM sends a FAILURE_UNSPECIFIED response and returns an error, closing the stream

		// send the response to EN based on host Desired state
		switch hostInv.DesiredState {
		// The host is in the REGISTERED state.
		// Allow the EN to retry but do not close the stream.
		// Assume SI initalially configure desiredstate as REGISTERED
		case computev1.HostState_HOST_STATE_REGISTERED:
			if err := s.handleRegisteredState(stream, hostInv, req); err != nil {
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
			if err := s.handleOnboardedState(stream, hostInv, req); err != nil {
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

//nolint:funlen,cyclop // reason: function is long due to necessary logic; cyclomatic complexity is high due to necessary handling
func (s *InteractiveOnboardingService) CreateNodes(ctx context.Context, req *pb.NodeRequest) (*pb.NodeResponse, error) {
	zlog.Info().Msgf("CreateNodes")
	if validationErr := req.Validate(); validationErr != nil {
		zlog.InfraSec().InfraErr(validationErr).Msgf("Request does not match the expected regex pattern %v", validationErr)
		return nil, validationErr
	}

	if s.authEnabled {
		// checking if JWT contains write permission
		if !s.rbac.IsRequestAuthorized(ctx, rbac.CreateKey) {
			err := inv_errors.Errorfc(codes.PermissionDenied, "Request is blocked by RBAC")
			zlog.InfraSec().InfraErr(err).Msgf("Request CreateNodes is not authenticated")
			return nil, err
		}
	}

	tenantID, present := inv_tenant.GetTenantIDFromContext(ctx)
	if !present {
		// This should never happen! Interceptor should either fail or set it!
		err := inv_errors.Errorfc(codes.Unauthenticated, "Tenant ID is not present in context")
		zlog.InfraSec().InfraErr(err).Msg("Request CreateNodes is not authenticated")
		return nil, err
	}
	zlog.Debug().Msgf("CreateNodes: tenantID=%s", tenantID)

	/* Copy node data from user */
	hostresdata, err := CopyNodeReqToNodeData(req.Payload, tenantID)
	if err != nil {
		zlog.InfraSec().InfraErr(err).Msgf("CopyNodeReqToNodeData error: %v", err)
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
	// Print the Host onboarded time for Instrumentation
	zlog.Info().Msgf("Instrumentation Info: Host Onboarded Successfully on %d\n",
		host.OnboardingStatusTimestamp)
	var hostInv *computev1.HostResource
	hostInv, err = s.invClient.GetHostResourceByUUID(ctx, tenantID, host.Uuid)
	switch {
	case inv_errors.IsNotFound(err):
		zlog.Debug().Msgf("Create op : Node Doesn't Exist for GUID %s and tID=%s\n",
			host.Uuid, tenantID)
	case err == nil:
		zlog.Debug().Msgf("Create op : Node and its Host Resource Already Exist for GUID %s, tID=%s \n",
			host.Uuid, tenantID)
		// UUID found and check for the serial number matches
		if hostInv.SerialNumber != host.SerialNumber {
			zlog.Debug().Msgf("Serial number mismatch for GUID %s, updating host resource", host.Uuid)
			// Update the host resource with the correct serial number
			// and all required fields host registration and onboarding status for serial id mismatch
			hostInv.SerialNumber = host.SerialNumber
			hostInv.BmcIp = host.BmcIp
			hostInv.PxeMac = host.PxeMac
			hostInv.CurrentState = computev1.HostState_HOST_STATE_ONBOARDED
			if updateErr := s.invClient.UpdateHostResourceStatus(ctx, hostInv.GetTenantId(), hostInv.ResourceId, hostInv,
				inv_status.New(om_status.OnboardingStatusDone.Status, om_status.OnboardingStatusDone.StatusIndicator),
				inv_status.New(om_status.HostRegistrationUnknown.Status,
					om_status.HostRegistrationUnknown.StatusIndicator)); updateErr != nil {
				zlog.InfraSec().InfraErr(updateErr).Msgf("Failed to update Host resource: %v tID=%s", hostInv, tenantID)
				return nil, updateErr
			}
		}
		if ztErr := s.startZeroTouch(ctx, tenantID, hostInv.ResourceId); ztErr != nil {
			zlog.InfraSec().InfraErr(ztErr).Msgf("startZeroTouch error: %v", ztErr)
			return nil, ztErr
		}
		return &pb.NodeResponse{Payload: req.Payload, ProjectId: hostInv.GetTenantId()}, nil
	case err != nil:
		zlog.Debug().Msgf("Create op :Failed CreateNodes() for GUID %s tID=%s \n", host.Uuid, tenantID)
		zlog.InfraSec().InfraErr(err).Msgf("Create op :Failed CreateNodes()\n")
		return nil, err
	}
	// UUID not found, create a new host
	hostResID, err = s.invClient.CreateHostResource(ctx, tenantID, host)
	if err != nil {
		zlog.InfraSec().InfraErr(err).Msgf("Cannot create Host resource: %v tID=%s", host, tenantID)
		return nil, err
	}
	zlog.Debug().Msgf("CreateHostResource ID = %s and tID=%s", hostResID, tenantID)

	if err := s.startZeroTouch(ctx, tenantID, hostResID); err != nil {
		zlog.InfraSec().InfraErr(err).Msgf("startZeroTouch error: %v", err)
		return nil, err
	}

	return &pb.NodeResponse{Payload: req.Payload, ProjectId: hostInv.GetTenantId()}, nil
}

func (s *InventoryClientService) startZeroTouch(ctx context.Context, tenantID, hostResID string) error {
	zlog.Debug().Msgf("Starting zero touch for host ID %s  tenant ID %s...", hostResID, tenantID)
	zlog.Info().Msgf("Starting zero touch for host")

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
	pconf, err := s.invClient.GetProviderConfig(ctx, tenantID, onboarding_types.DefaultProviderName)
	if err != nil {
		zlog.Err(err).Msgf("Failed to get provider configuration")
		return nil
	}

	// if AutoProvision is set, create an Instance for the Host with the OS set to the value of the default OS
	return s.checkNCreateInstance(ctx, tenantID, *pconf, host)
}

func (s *InventoryClientService) checkNCreateInstance(ctx context.Context, tenantID string,
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
		}
		osRes, err := s.invClientAPI.GetOSResourceByResourceID(ctx, tenantID, pconf.DefaultOs)
		if err != nil {
			zlog.Debug().Msgf("Failed to GetOSResourceByResourceID for host resource (uuid=%s)", hostResID)
			zlog.Err(err).Msgf("Failed to GetOSResourceByResourceID for host resource")
			return err
		}
		instance.SecurityFeature = osRes.GetSecurityFeature()

		if _, err := s.invClientAPI.CreateInstanceResource(ctx, tenantID, instance); err != nil {
			zlog.Debug().Msgf("Failed to CreateInstanceResource for host resource (uuid=%s),tID=%s", hostResID, tenantID)
			zlog.Err(err).Msgf("Failed to CreateInstanceResource for host resource uuid,tID")
			return err
		}
	}

	return nil
}
