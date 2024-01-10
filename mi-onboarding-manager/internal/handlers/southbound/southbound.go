// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package southbound

import (
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/onboardingmgr"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/handlers/southbound/artifact"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/onboarding"
	"google.golang.org/grpc"
	"net"
)

// Misc variables.
var (
	loggerName = "OnboardingSBHandler"
	zlog       = logging.GetLogger(loggerName)
)

type SBHandlerConfig struct {
	ServerAddress string
	EnableTracing bool
}

type SBHandler struct {
	cfg SBHandlerConfig

	server *grpc.Server
}

func NewSBHandler(config SBHandlerConfig) (*SBHandler, error) {
	return &SBHandler{
		cfg: config,
	}, nil
}

func (sbh *SBHandler) Start() error {
	lis, err := net.Listen("tcp", sbh.cfg.ServerAddress)
	if err != nil {
		return err
	}

	sbh.server = grpc.NewServer()
	pb.RegisterNodeArtifactServiceNBServer(sbh.server, &artifact.NodeArtifactService{})
	pb.RegisterOnBoardingEBServer(sbh.server, &onboarding.OnboardingManager{})

	//Run go routine to start the gRPC server
	go func() {
		if err := sbh.server.Serve(lis); err != nil {
			zlog.MiSec().Fatal().Err(err).Msgf("Error listening with TCP: %s", lis.Addr().String())
		}
	}()

	zlog.MiSec().Info().Msgf("SB handler started")
	return nil
}

func (sbh *SBHandler) Stop() {
	sbh.server.Stop()
	zlog.MiSec().Info().Msgf("SB handler stopped")
}
