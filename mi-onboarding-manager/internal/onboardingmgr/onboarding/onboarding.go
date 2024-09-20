/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	dkam "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/api/dkammgr/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/errors"
	logging "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/logging"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/policy/rbac"
)

var (
	clientName = "Onboarding"
	zlog       = logging.GetLogger(clientName)
)

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

	var err error
	if enableAuth {
		zlog.Info().Msgf("Authentication is enabled, starting RBAC server for Onboarding manager")
		// start OPA server with policies
		_, err = rbac.New(rbacRules)
		if err != nil {
			zlog.Fatal().Msg("Failed to start RBAC OPA server")
		}
	}
}

func GetOSResourceFromDkamService(ctx context.Context, repoURL, sha256, profilename, installedPackages string,
	platform, kernelCommand string, osType string,
) (*dkam.GetENProfileResponse, error) {
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
	dkamConn, err := grpc.NewClient(dkamAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("Failed to connect to DKAM server %s, retry in next iteration...", dkamAddr)
		return nil, inv_errors.Errorf("Failed to connect to DKAM server")
	}

	defer dkamConn.Close()

	// Create an instance of DkamServiceClient using the connection
	dkamClient := dkam.NewDkamServiceClient(dkamConn)
	response, err := dkamClient.GetENProfile(ctx, &dkam.GetENProfileRequest{
		RepoUrl:           repoURL,
		Sha256:            sha256,
		InstalledPackages: installedPackages,
		KernelCommand:     kernelCommand,
		ProfileName:       profilename,
		Platform:          platform,
		OsType:            osType,
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
