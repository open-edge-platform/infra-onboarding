/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	"context"
	"errors"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"

	dkam "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/api/dkammgr/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/api"
	om_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/status"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	logging "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/policy/rbac"
)

var (
	clientName  = "Onboarding"
	zlog        = logging.GetLogger(clientName)
	_invClient  *invclient.OnboardingInventoryClient
	rbacPolicy  *rbac.Policy
	authEnabled = false
)

type Manager struct {
	pb.OnBoardingSBServer
}

//nolint:tagliatelle // Renaming the json keys may effect while unmarshalling/marshaling so, used nolint.
type ResponseData struct {
	To2CompletedOn string `json:"to2CompletedOn"`
	To0Expiry      string `json:"to0Expiry"`
}

func InitOnboarding(invClient *invclient.OnboardingInventoryClient, _ string, enableAuth bool, rbacRules string) {
	if invClient == nil {
		zlog.Debug().Msgf("Warning: invClient is nil")
		return
	}
	_invClient = invClient

	var err error
	if enableAuth {
		zlog.Info().Msgf("Authentication is enabled, starting RBAC server for Onboarding manager")
		// start OPA server with policies
		rbacPolicy, err = rbac.New(rbacRules)
		if err != nil {
			zlog.Fatal().Msg("Failed to start RBAC OPA server")
		}
	}
	authEnabled = enableAuth
}

func GetOSResourceFromDkamService(ctx context.Context, profilename, platform string) (*dkam.GetArtifactsResponse, error) {
	// Get the DKAM manager host and port
	host := os.Getenv("DKAMHOST")
	port := os.Getenv("DKAMPORT")

	if host == "" || port == "" {
		err := inv_errors.Errorf("DKAM endpoint is not set")
		zlog.MiErr(err).Msgf("")
		return nil, err
	}

	// Dial DKAM Manager API
	dkamAddr := fmt.Sprintf("%s:%s", host, port)

	// Create a gRPC connection to DKAM server
	dkamConn, err := grpc.Dial(dkamAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("Failed to connect to DKAM server %s, retry in next iteration...", dkamAddr)
		return nil, inv_errors.Errorf("Failed to connect to DKAM server")
	}

	defer dkamConn.Close()

	// Create an instance of DkamServiceClient using the connection
	dkamClient := dkam.NewDkamServiceClient(dkamConn)
	response, err := dkamClient.GetArtifacts(ctx, &dkam.GetArtifactsRequest{
		ProfileName: profilename,
		Platform:    platform,
	})
	if err != nil {
		zlog.Err(err).Msg("Failed to get software details from DKAM")
		return nil, err
	}
	if response == nil {
		responseErr := inv_errors.Errorf("DKAM response is nil")
		zlog.MiErr(responseErr).Msg("")
		return nil, responseErr
	}

	zlog.Debug().Msgf("Software details successfully obtained from DKAM: %v", response)

	return response, nil
}

func IsSecureBootConfigAtEdgeNodeMismatch(ctx context.Context, req *pb.SecureBootResponse) error {
	zlog.Info().Msgf("IsSecureBootConfigAtEdgeNodeMismatch")

	// Getting details from Host for now using GUID/UUID
	instanceDetails, err := _invClient.GetHostResourceByUUID(ctx, req.Guid)
	if err != nil {
		zlog.Err(err).Msg("Failed to get Instance Details")
		return err // Return the error if failed to get instance details
	}
	// Check if SecureBootStatus mismatches
	if ((instanceDetails.Instance.GetSecurityFeature().String() == "SECURITY_FEATURE_SECURE_BOOT_AND_FULL_DISK_ENCRYPTION") &&
		(req.Result.String() == "FAILURE")) ||
		((instanceDetails.Instance.GetSecurityFeature().String() == "SECURITY_FEATURE_UNSPECIFIED") &&
			(req.Result.String() == "SUCCESS")) ||
		((instanceDetails.Instance.GetSecurityFeature().String() == "SECURITY_FEATURE_NONE") &&
			(req.Result.String() == "SUCCESS")) {
		// If there's a mismatch, update the instance status to INSTANCE_STATE_ERROR
		err := UpdateInstanceStatusByGUID(ctx, _invClient, req.Guid, computev1.InstanceStatus_INSTANCE_STATUS_ERROR,
			om_status.OnboardingStatusFailed)
		if err != nil {
			zlog.Err(err).Msg("Failed to Update the instance status")
			return err
		}

		// Update host status with fail status and statusDetails
		err = UpdateHostStatusByHostGUID(ctx, _invClient, req.Guid, computev1.HostStatus_HOST_STATUS_BOOT_FAILED,
			"SecureBoot status mismatch", om_status.OnboardingStatusFailed)
		if err != nil {
			zlog.Err(err).Msg("Failed to Update the host status")
			return err
		}

		// Return an error indicating SecureBoot status mismatch
		return errors.New("SecureBoot status mismatch")
	}

	zlog.Info().Msgf("IsSecureBootConfigAtEdgeNodeMismatch(): SB flags matched")

	// If there's no error and no mismatch, return nil
	return nil
}

func (s *Manager) SecureBootStatus(ctx context.Context, req *pb.SecureBootStatRequest) (*pb.SecureBootResponse, error) {
	zlog.Info().Msgf("------- SecureBootStatus() ----------------\n")
	if authEnabled {
		// this request requires read and write permissions
		if !rbacPolicy.IsRequestAuthorized(ctx, rbac.GetKey) || !rbacPolicy.IsRequestAuthorized(ctx, rbac.UpdateKey) {
			err := inv_errors.Errorfc(codes.Unauthenticated, "Request is blocked by RBAC")
			zlog.MiSec().MiErr(err).Msgf("Request SecureBootStatus is not authenticated")
			return nil, err
		}
	}
	resp := &pb.SecureBootResponse{
		Guid:   req.Guid,
		Result: pb.SecureBootResponse_Status(req.Result),
	}
	err := IsSecureBootConfigAtEdgeNodeMismatch(ctx, resp)
	if err != nil {
		return resp, errors.New("SecureBoot Status mismatch")
	}
	return resp, nil
}
