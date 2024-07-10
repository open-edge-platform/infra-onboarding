// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package southbound

import (
	"net"
	"sync"

	"google.golang.org/grpc"

	dkammgr "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/internal/dkammgr"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/internal/invclient"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/api/dkammgr/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
)

// Misc variables.
var (
	loggerName = "DKAMSBHandler"
	zlog       = logging.GetLogger(loggerName)
)

type SBHandlerConfig struct {
	ServerAddress    string
	InventoryAddress string
	EnableTracing    bool
	EnableAuth       bool
	RBAC             string
}

type SBHandler struct {
	invClient *invclient.DKAMInventoryClient
	cfg       SBHandlerConfig
	wg        *sync.WaitGroup
	lis       net.Listener
	server    *grpc.Server
}

func NewSBHandler(invClient *invclient.DKAMInventoryClient, config SBHandlerConfig) (*SBHandler, error) {
	lis, err := net.Listen("tcp", config.ServerAddress)
	if err != nil {
		return nil, err
	}
	zlog.MiSec().Info().Msgf("DKAM server started at %s", config.ServerAddress)
	return NewSBHandlerWithListener(lis, invClient, config), nil
}

func NewSBHandlerWithListener(listener net.Listener,
	invClient *invclient.DKAMInventoryClient,
	config SBHandlerConfig,
) *SBHandler {
	return &SBHandler{
		invClient: invClient,
		cfg:       config,
		wg:        &sync.WaitGroup{},
		lis:       listener,
	}
}

func (sbh *SBHandler) Start() error {
	nodeArtifactService, err := dkammgr.NewDKAMService(sbh.invClient,
		sbh.cfg.InventoryAddress, sbh.cfg.EnableTracing, sbh.cfg.EnableAuth, sbh.cfg.RBAC)
	if err != nil {
		return err
	}

	sbh.server = grpc.NewServer()
	pb.RegisterDkamServiceServer(sbh.server, nodeArtifactService)

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
