// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package southbound

import (
	"net"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/logging"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/metrics"
	inv_tenant "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/tenant"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/onboarding-manager/internal/handlers/southbound/artifact"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/onboarding-manager/internal/invclient"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/onboarding-manager/pkg/api"
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
	EnableMetrics    bool
	MetricsAddress   string
	EnableAuth       bool
	RBAC             string
}

// Nio config.
type SBHandlerNioConfig struct {
	ServerAddressNio string
	InventoryAddress string
	EnableTracing    bool
}

// Nio Handler.
type SBNioHandler struct {
	invClient *invclient.OnboardingInventoryClient
	cfg       SBHandlerNioConfig
	lis       net.Listener
	server    *grpc.Server
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

// Start IO server.
func (sbh *SBHandler) Start() error {
	nodeArtifactService, err := artifact.NewArtifactService(sbh.invClient,
		sbh.cfg.InventoryAddress, sbh.cfg.EnableTracing, sbh.cfg.EnableAuth, sbh.cfg.RBAC)
	if err != nil {
		return err
	}
	var srvOpts []grpc.ServerOption
	var unaryInter []grpc.UnaryServerInterceptor
	unaryInter = append(unaryInter, inv_tenant.GetExtractTenantIDInterceptor(inv_tenant.GetOnboardingRoles()))
	srvMetrics := metrics.GetServerMetricsWithLatency()
	cliMetrics := metrics.GetClientMetricsWithLatency()
	if sbh.cfg.EnableMetrics {
		zlog.Info().Msgf("Metrics exporter Enable with address %s", sbh.cfg.MetricsAddress)
		unaryInter = append(unaryInter, srvMetrics.UnaryServerInterceptor())
	}
	srvOpts = append(srvOpts, grpc.ChainUnaryInterceptor(unaryInter...))
	sbh.server = grpc.NewServer(srvOpts...)
	pb.RegisterNodeArtifactServiceNBServer(sbh.server, nodeArtifactService)

	// Register reflection service on gRPC server.
	reflection.Register(sbh.server)
	if sbh.cfg.EnableMetrics {
		// Register metrics
		srvMetrics.InitializeMetrics(sbh.server)
		metrics.StartMetricsExporter([]prometheus.Collector{cliMetrics, srvMetrics},
			metrics.WithListenAddress(sbh.cfg.MetricsAddress))
	}
	// Run go routine to start the gRPC server.
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

func NewSBNioHandler(invClient *invclient.OnboardingInventoryClient,
	config SBHandlerNioConfig,
) (*SBNioHandler, error) {
	lis, err := net.Listen("tcp", config.ServerAddressNio)
	if err != nil {
		return nil, err
	}
	return NewSBNioHandlerWithListener(lis, invClient, config), nil
}

func NewSBNioHandlerWithListener(listener net.Listener,
	invClient *invclient.OnboardingInventoryClient,
	config SBHandlerNioConfig,
) *SBNioHandler {
	return &SBNioHandler{
		invClient: invClient,
		cfg:       config,
		lis:       listener,
	}
}

// start SB Nio server.
func (sbhnio *SBNioHandler) Start() error {
	nodeArtifactService, err := artifact.NewNonInteractiveOnboardingService(sbhnio.invClient,
		sbhnio.cfg.InventoryAddress, sbhnio.cfg.EnableTracing)
	if err != nil {
		return err
	}
	var srvOpts []grpc.ServerOption
	sbhnio.server = grpc.NewServer(srvOpts...)
	pb.RegisterNonInteractiveOnboardingServiceServer(sbhnio.server, nodeArtifactService)
	// Register reflection service on gRPC server.
	reflection.Register(sbhnio.server)
	// Run go routine to start the gRPC server.
	go func() {
		if err := sbhnio.server.Serve(sbhnio.lis); err != nil {
			zlog.MiSec().Fatal().Err(err).Msgf("Error listening with TCP: %s", sbhnio.lis.Addr().String())
		}
	}()

	zlog.MiSec().Info().Msgf("SB NIO handler started")
	return nil
}

func (sbhnio *SBNioHandler) Stop() {
	sbhnio.server.Stop()
	zlog.MiSec().Info().Msgf("SB NIO handler stopped")
}
