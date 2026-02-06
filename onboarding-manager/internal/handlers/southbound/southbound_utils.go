// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package southbound

import (
	"context"
	"net"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/metrics"
	inv_tenant "github.com/open-edge-platform/infra-core/inventory/v2/pkg/tenant"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/handlers/southbound/grpcserver"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/invclient"
	pb "github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/api/onboardingmgr/v1"
)

// Misc variables.
var (
	loggerName = "OnboardingSBHandler"
	zlog       = logging.GetLogger(loggerName)
)

// SBHandlerConfig provides functionality for onboarding management.
type SBHandlerConfig struct {
	ServerAddress    string
	InventoryAddress string
	EnableTracing    bool
	EnableMetrics    bool
	MetricsAddress   string
	EnableAuth       bool
	RBAC             string
}

// SBHandlerNioConfig provides functionality for onboarding management.
// Nio config.
type SBHandlerNioConfig struct {
	ServerAddressNio string
	InventoryAddress string
	EnableTracing    bool
}

// SBNioHandler provides functionality for onboarding management.
// Nio Handler.
type SBNioHandler struct {
	invClient *invclient.OnboardingInventoryClient
	cfg       SBHandlerNioConfig
	lis       net.Listener
	server    *grpc.Server
}

// SBHandler provides functionality for onboarding management.
type SBHandler struct {
	invClient *invclient.OnboardingInventoryClient
	cfg       SBHandlerConfig

	lis    net.Listener
	server *grpc.Server
}

// NewSBHandler performs operations for onboarding management.
func NewSBHandler(invClient *invclient.OnboardingInventoryClient, config SBHandlerConfig) (*SBHandler, error) {
	lis, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", config.ServerAddress)
	if err != nil {
		return nil, err
	}

	return NewSBHandlerWithListener(lis, invClient, config), nil
}

// NewSBHandlerWithListener performs operations for onboarding management.
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
	interactiveOnboardingService, err := grpcserver.NewInteractiveOnboardingService(
		sbh.invClient,
		sbh.cfg.InventoryAddress, sbh.cfg.EnableTracing, sbh.cfg.EnableAuth, sbh.cfg.RBAC)
	if err != nil {
		return err
	}
	srvOpts := make([]grpc.ServerOption, 0, 1)
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
	pb.RegisterInteractiveOnboardingServiceServer(sbh.server, interactiveOnboardingService)

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
			zlog.InfraSec().Fatal().Err(err).Msgf("Error listening with TCP: %s", sbh.lis.Addr().String())
		}
	}()

	zlog.InfraSec().Info().Msgf("SB handler started")
	return nil
}

// Stop performs operations for the receiver.
func (sbh *SBHandler) Stop() {
	sbh.server.Stop()
	zlog.InfraSec().Info().Msgf("SB handler stopped")
}

// NewSBNioHandler performs operations for onboarding management.
func NewSBNioHandler(invClient *invclient.OnboardingInventoryClient,
	config SBHandlerNioConfig,
) (*SBNioHandler, error) {
	lis, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", config.ServerAddressNio)
	if err != nil {
		return nil, err
	}
	return NewSBNioHandlerWithListener(lis, invClient, config), nil
}

// NewSBNioHandlerWithListener performs operations for onboarding management.
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

// Start performs operations for the receiver.
// start SB Nio server.
func (sbhnio *SBNioHandler) Start() error {
	interactiveOnboardingService, err := grpcserver.NewNonInteractiveOnboardingService(sbhnio.invClient,
		sbhnio.cfg.InventoryAddress, sbhnio.cfg.EnableTracing)
	if err != nil {
		return err
	}
	var srvOpts []grpc.ServerOption
	sbhnio.server = grpc.NewServer(srvOpts...)
	pb.RegisterNonInteractiveOnboardingServiceServer(sbhnio.server, interactiveOnboardingService)
	// Register reflection service on gRPC server.
	reflection.Register(sbhnio.server)
	// Run go routine to start the gRPC server.
	go func() {
		if err := sbhnio.server.Serve(sbhnio.lis); err != nil {
			zlog.InfraSec().Fatal().Err(err).Msgf("Error listening with TCP: %s", sbhnio.lis.Addr().String())
		}
	}()

	zlog.InfraSec().Info().Msgf("SB NIO handler started")
	return nil
}

// Stop performs operations for the receiver.
func (sbhnio *SBNioHandler) Stop() {
	sbhnio.server.Stop()
	zlog.InfraSec().Info().Msgf("SB NIO handler stopped")
}
