// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package southbound

import (
	"net"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/handlers/southbound/artifact"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/onboarding"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/api"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
	"google.golang.org/grpc"
)

// Misc variables.
var (
	loggerName = "OnboardingSBHandler"
	zlog       = logging.GetLogger(loggerName)
)

type SBHandlerConfig struct {
	ServerAddress    string
	InventoryAddress string
	EnableTracing    bool
}

type SBHandler struct {
	invClient *invclient.OnboardingInventoryClient
	cfg       SBHandlerConfig

	lis    net.Listener
	server *grpc.Server
}

func NewSBHandler(invClient *invclient.OnboardingInventoryClient, config SBHandlerConfig) (*SBHandler, error) {
	lis, err := net.Listen("tcp", config.ServerAddress)
	if err != nil {
		return nil, err
	}

	return NewSBHandlerWithListener(lis, invClient, config), nil
}

func NewSBHandlerWithListener(listener net.Listener,
	invClient *invclient.OnboardingInventoryClient,
	config SBHandlerConfig,
) *SBHandler {
	return &SBHandler{
		invClient: invClient,
		cfg:       config,
		lis:       listener,
	}
}

func (sbh *SBHandler) Start() error {
	nodeArtifactService, err := artifact.NewArtifactService(sbh.invClient,
		sbh.cfg.InventoryAddress, sbh.cfg.EnableTracing)
	if err != nil {
		return err
	}

	sbh.server = grpc.NewServer()
	pb.RegisterNodeArtifactServiceNBServer(sbh.server, nodeArtifactService)
	pb.RegisterOnBoardingEBServer(sbh.server, &onboarding.OnboardingManager{})

	// Run go routine to start the gRPC server
	go func() {
		if err := sbh.server.Serve(sbh.lis); err != nil {
			zlog.MiSec().Fatal().Err(err).Msgf("Error listening with TCP: %s", sbh.lis.Addr().String())
		}
	}()

	zlog.MiSec().Info().Msgf("SB handler started")
	return nil
}

func (sbh *SBHandler) Stop() {
	sbh.server.Stop()
	zlog.MiSec().Info().Msgf("SB handler stopped")
}
