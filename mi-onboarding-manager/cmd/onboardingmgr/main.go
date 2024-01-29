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
	"time"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/client"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/flags"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/oam"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/tracing"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/onboardingmgr"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/handlers/controller"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/handlers/southbound"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/invclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/config"
	inventory "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/controller"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/onboarding"
)

var (
	name = "MiOnboardingRM"
	zlog = logging.GetLogger(name + "Main")

	dkamAddr         = flag.String("dkamaddr", "localhost:5581", "DKAM server address to connect to")
	serverAddress    = flag.String(flags.ServerAddress, "0.0.0.0:50054", flags.ServerAddressDescription)
	inventoryAddress = flag.String(client.InventoryAddress, "localhost:50051", client.InventoryAddressDescription)
	oamServerAddress = flag.String(oam.OamServerAddress, "", oam.OamServerAddressDescription)
	enableTracing    = flag.Bool(tracing.EnableTracing, false, tracing.EnableTracingDescription)
	traceURL         = flag.String(tracing.TraceURL, "", tracing.TraceURLDescription)

	wg        = sync.WaitGroup{}
	readyChan = make(chan bool, 1)
	termChan  = make(chan bool, 1)
	sigChan   = make(chan os.Signal, 1)
)

const (
	DefaultTimeout = 3 * time.Second
)

var (
	manager *inventory.InventoryManager
)

type OnboardingEB struct {
	pb.UnimplementedOnBoardingEBServer
}

var (
	Project   = "frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service"
	RepoURL   = fmt.Sprintf("https://github.com/intel-innersource/%s.git", Project)
	Version   = "<unset>"
	Revision  = "<unset>"
	BuildDate = "<unset>"
)

func printSummary() {
	zlog.Info().Msgf("Starting IFM Onboarding Manager")
	zlog.MiSec().Info().Msgf("RepoURL: %s, Version: %s, Revision: %s, BuildDate: %s\n", RepoURL, Version, Revision, BuildDate)
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

func main() {
	// Print a summary of the build
	printSummary()
	flag.Parse()

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

	config.Load()
	conf := config.GetConfig()
	manager = inventory.NewInventoryManager(conf)

	invClient, err := invclient.NewOnboardingInventoryClientWithOptions(
		invclient.WithInventoryAddress(*inventoryAddress),
		invclient.WithEnableTracing(*enableTracing),
	)
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Unable to start onboarding inventory client")
	}

	onboarding.InitOnboarding(invClient, *dkamAddr)

	onboardingController, err := controller.New(invClient)
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Unable to create onboarding controller")
	}

	err = onboardingController.Start()
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Unable to start onboarding controller")
	}

	sbHandler, err := southbound.NewSBHandler(invClient, southbound.SBHandlerConfig{
		ServerAddress: *serverAddress,
		EnableTracing: *enableTracing,
	})
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Unable to create southbound handler")
	}

	err = sbHandler.Start()
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
