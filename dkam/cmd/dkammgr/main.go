// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel
package main

import (
	// import dependencies

	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/client"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/logging"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/metrics"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/oam"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/tracing"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/internal/dkammgr"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/internal/handlers/controller"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/internal/invclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/pkg/config"
)

var (
	name = "MIDKAMMain"
	zlog = logging.GetLogger(name + "Main")

	inventoryAddress = flag.String(client.InventoryAddress, "localhost:50051", client.InventoryAddressDescription)
	wg               = sync.WaitGroup{}
	oamServerAddress = flag.String(oam.OamServerAddress, "", oam.OamServerAddressDescription)
	enableTracing    = flag.Bool(tracing.EnableTracing, false, tracing.EnableTracingDescription)
	traceURL         = flag.String(tracing.TraceURL, "", tracing.TraceURLDescription)
	enableMetrics    = flag.Bool(metrics.EnableMetrics, false, metrics.EnableMetricsDescription)
	metricsAddress   = flag.String(metrics.MetricsAddress, metrics.MetricsAddressDefault, metrics.MetricsAddressDescription)
	readyChan        = make(chan bool, 1)
	termChan         = make(chan bool, 1)
	sigChan          = make(chan os.Signal, 1)
)

var (
	Project   = "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam"
	RepoURL   = fmt.Sprintf("https://github.com/intel-innersource/%s.git", Project)
	Version   = "<unset>"
	Revision  = "<unset>"
	BuildDate = "<unset>"
)

func printSummary() {
	zlog.Info().Msg("Starting DKAM")
	zlog.MiSec().Info().Msgf("RepoURL: %s, Version: %s, Revision: %s, BuildDate: %s\n",
		RepoURL, Version, Revision, BuildDate)
}

func main() {

	watcher, watcherErr := SetWatcher()
	if watcherErr != nil {
		zlog.MiSec().Fatal().Err(watcherErr).Msgf("Failed to set watcher.")
		return
	}
	defer watcher.Close()

	// Print a summary of the build
	printSummary()
	flag.Parse()

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

	if *enableMetrics {
		startMetricsServer()
	}

	if err := GetArtifacts(context.Background()); err != nil {
		zlog.MiSec().Fatal().Err(err).Msg("Failed to get artifacts")
	}

	go func() {
		defer wg.Done()
		if err := BuildBinaries(); err != nil {
			zlog.MiSec().Fatal().Err(err).Msg("Failed to get artifacts")
		}
	}()

	invClient, err := invclient.NewDKAMInventoryClientWithOptions(
		invclient.WithInventoryAddress(*inventoryAddress),
		invclient.WithEnableTracing(*enableTracing),
		invclient.WithEnableMetrics(*enableMetrics),
	)
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Unable to start onboarding inventory client")
	}

	dkamController, err := controller.New(invClient, *enableTracing)
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Unable to create onboarding controller")
	}

	err = dkamController.Start()
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Unable to start onboarding controller")
	}

	setupOamServerAndSetReady(*enableTracing, *oamServerAddress)

	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	<-sigChan // blocking

	// Terminate Onboarding Manager when termination signal received
	close(termChan)
	dkamController.Stop()
	invClient.Close()

	//wg.Done()
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

func startMetricsServer() {
	metrics.StartMetricsExporter([]prometheus.Collector{metrics.GetClientMetricsWithLatency()},
		metrics.WithListenAddress(*metricsAddress))
}

func setupOamServerAndSetReady(enableTracing bool, oamServerAddress string) {
	zlog.Info().Msg("Inside setupOamServerAndSetReady...")
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

func GetArtifacts(ctx context.Context) error {
	outDir := filepath.Join(config.DownloadPath, "tmp")
	// 0. cleanup
	os.RemoveAll(outDir)
	zlog.MiSec().Info().Msg("Get all artifacts...")
	// Download release manifest.yaml file.
	artifactsErr := dkammgr.DownloadArtifacts(ctx)
	if artifactsErr != nil {
		zlog.MiSec().Fatal().Err(artifactsErr).Msgf("Error downloading file %v", artifactsErr)
		return artifactsErr
	}
	return nil
}

func BuildBinaries() error {

	// Donwload and sign iPXE
	signedIPXE, pxeErr := dkammgr.BuildSignIpxe()
	if pxeErr != nil {
		zlog.MiSec().Fatal().Err(pxeErr).Msgf("Failed to sign MicroOS %v", pxeErr)
		return pxeErr
	}
	if signedIPXE {
		zlog.MiSec().Info().Msg("Signed IPXE and moved to PVC")
	}

	// Download and sign MicroOS.
	signed, signerr := dkammgr.SignMicroOS()
	if signerr != nil {
		zlog.MiSec().Fatal().Err(signerr).Msgf("Failed to sign MicroOS %v", signerr)
		return signerr
	}
	if signed {
		zlog.MiSec().Info().Msg("Signed MicroOS and moved to PVC")
	}
	return nil
}

func SetWatcher() (*fsnotify.Watcher, error) {
	zlog.MiSec().Info().Msg("Enable watcher...")
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Failed to create a watcher")
	}

	orchCACertificateFile := config.OrchCACertificateFile
	bootsCaCertificateFile := config.BootsCaCertificateFile

	// Add the certificate files to the watcher
	addFileToWatcher(watcher, orchCACertificateFile)
	addFileToWatcher(watcher, bootsCaCertificateFile)

	// Watch for events
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					zlog.MiSec().Info().Msg("Certificate file changed. Rebuilding iPXE and microOS...")
					zlog.MiSec().Fatal().Err(err).Msgf("Restart DKAM: %v", err)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				zlog.MiSec().Fatal().Err(err).Msgf("Error:%v", err)
			}
		}
	}()

	return watcher, nil

}

func addFileToWatcher(watcher *fsnotify.Watcher, filename string) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		zlog.MiSec().Fatal().Err(err).Msgf("File does not exist:%s, error: %v", filename, err)
		return
	}
	if err := watcher.Add(filename); err != nil {
		zlog.MiSec().Error().Msgf("Failed to add file to watcher:%s, error: %v", filename, err)
	} else {
		zlog.MiSec().Info().Msgf("Watcher added for file:%s", filename)
	}
}
