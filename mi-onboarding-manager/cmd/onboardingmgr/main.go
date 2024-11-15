// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/env"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/handlers/controller"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/handlers/southbound"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/onboarding"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/auth"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/client"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/flags"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/logging"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/oam"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/policy/rbac"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/secretprovider"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/tracing"
)

const envNameOnboardingCredentialsSecretName = "ONBOARDING_CREDENTIALS_SECRET_NAME"

const (
	ServerAddressNio = "serverAddressNio"
)

var (
	name = "MiOnboardingRM"
	zlog = logging.GetLogger(name + "Main")

	dkamAddr         = flag.String("dkamaddr", "localhost:5581", "DKAM server address to connect to")
	serverAddress    = flag.String(flags.ServerAddress, "0.0.0.0:50054", flags.ServerAddressDescription)
	serverAddressNio = flag.String(ServerAddressNio, "0.0.0.0:50055", "grpc server address for nio")
	inventoryAddress = flag.String(client.InventoryAddress, "localhost:50051", client.InventoryAddressDescription)
	oamServerAddress = flag.String(oam.OamServerAddress, "", oam.OamServerAddressDescription)
	enableTracing    = flag.Bool(tracing.EnableTracing, false, tracing.EnableTracingDescription)
	traceURL         = flag.String(tracing.TraceURL, "", tracing.TraceURLDescription)
	enableAuth       = flag.Bool(rbac.EnableAuth, true, rbac.EnableAuthDescription)
	rbacRules        = flag.String(rbac.RbacRules, "/rego/authz.rego", rbac.RbacRulesDescription)
	// see also internal/common/flags.go for other flags.

	wg        = sync.WaitGroup{}
	readyChan = make(chan bool, 1)
	termChan  = make(chan bool, 1)
	sigChan   = make(chan os.Signal, 1)
)

var (
	Project   = "frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service"
	RepoURL   = fmt.Sprintf("https://github.com/intel-innersource/%s.git", Project)
	Version   = "<unset>"
	Revision  = "<unset>"
	BuildDate = "<unset>"
)

func printSummary() {
	zlog.Info().Msgf("Starting IFM Onboarding Manager")
	zlog.MiSec().Info().Msgf("RepoURL: %s, Version: %s, Revision: %s, BuildDate: %s\n",
		RepoURL, Version, Revision, BuildDate)
}

func setupTracing(traceURL string) func(context.Context) error {
	cleanup, exportErr := tracing.NewTraceExporterHTTP(traceURL, name, nil)
	if exportErr != nil {
		zlog.Err(exportErr).Msg("Error creating trace exporter")
	}
	if cleanup != nil {
		zlog.Info().Msgf("Tracing enabled %s", traceURL)
	} else {
		zlog.Info().Msg("Tracing disabled")
	}
	return cleanup
}

func setupOamServerAndSetReady(enableTracing bool, oamServerAddress string) {
	if oamServerAddress != "" {
		// Add oam grpc server
		wg.Add(1)
		go func() {
			if err := oam.StartOamGrpcServer(termChan, readyChan, &wg, oamServerAddress, enableTracing); err != nil {
				zlog.MiSec().Fatal().Err(err).Msg("Cannot start Inventory OAM gRPC server")
			}
		}()
		readyChan <- true
	}
}

//nolint:funlen,cyclop // it's a main, complexity is 11
func main() {
	// Print a summary of the build
	printSummary()
	flag.Parse()

	env.MustEnsureRequired()

	// Startup order, respecting deps
	// 1. Setup tracing
	// 2. Start Inventory client
	// 3. Start OnboardingController and the reconcilers
	// 4. Start southbound handler
	// 5. Start the OAM server
	if *enableTracing {
		cleanup := setupTracing(*traceURL)
		if cleanup != nil {
			defer func() {
				err := cleanup(context.Background())
				if err != nil {
					zlog.Err(err).Msg("Error in tracing cleanup")
				}
			}()
		}
	}

	invClient, err := invclient.NewOnboardingInventoryClientWithOptions(
		invclient.WithInventoryAddress(*inventoryAddress),
		invclient.WithEnableTracing(*enableTracing),
	)
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Unable to start onboarding inventory client")
	}

	onboarding.InitOnboarding(invClient, *dkamAddr, *enableAuth, *rbacRules)

	onboardingCredentialsSecretName := os.Getenv(envNameOnboardingCredentialsSecretName)
	if onboardingCredentialsSecretName == "" {
		invErr := inv_errors.Errorf("%s env variable is not set, using default value", envNameOnboardingCredentialsSecretName)
		zlog.MiSec().Fatal().Err(invErr).Msgf("")
	}

	if initErr := secretprovider.Init(context.Background(), []string{onboardingCredentialsSecretName}); initErr != nil {
		zlog.MiSec().Fatal().Err(initErr).Msgf("Unable to initialize required secrets")
	}

	if authInitErr := auth.Init(); authInitErr != nil {
		zlog.MiSec().Fatal().Err(authInitErr).Msgf("Unable to initialize auth service")
	}

	onboardingController, err := controller.New(invClient, *enableTracing)
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Unable to create onboarding controller")
	}

	err = onboardingController.Start()
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Unable to start onboarding controller")
	}

	// SB handler for IO.
	sbHandler, err := southbound.NewSBHandler(invClient, southbound.SBHandlerConfig{
		ServerAddress:    *serverAddress,
		EnableTracing:    *enableTracing,
		InventoryAddress: *inventoryAddress,
		EnableAuth:       *enableAuth,
		RBAC:             *rbacRules,
	})
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Unable to create southbound handler")
	}

	// start SB IO handler.
	err = sbHandler.Start()
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Unable to start southbound handler")
	}

	// SB handler for NIO.
	sbnioHandler, err := southbound.NewSBNioHandler(invClient, southbound.SBHandlerNioConfig{
		ServerAddressNio: *serverAddressNio,
		EnableTracing:    *enableTracing,
		InventoryAddress: *inventoryAddress,
	})
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Unable to create southbound handler")
	}
	// start SB NIO handler.
	err = sbnioHandler.Start()
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Unable to start southbound handler")
	}

	setupOamServerAndSetReady(*enableTracing, *oamServerAddress)

	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	<-sigChan // blocking

	// Terminate Onboarding Manager when termination signal received
	close(termChan)
	sbHandler.Stop()
	onboardingController.Stop()
	invClient.Close()

	wg.Done()
}
