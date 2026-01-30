// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package main

import (
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

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/metrics"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/oam"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/tracing"
	"github.com/open-edge-platform/infra-onboarding/dkam/internal/dkammgr"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/config"
)

var (
	name = "InfraDKAM"
	zlog = logging.GetLogger(name + "Main")

	wg               = sync.WaitGroup{}
	oamServerAddress = flag.String(oam.OamServerAddress, "", oam.OamServerAddressDescription)
	enableTracing    = flag.Bool(tracing.EnableTracing, false, tracing.EnableTracingDescription)
	traceURL         = flag.String(tracing.TraceURL, "", tracing.TraceURLDescription)
	enableMetrics    = flag.Bool(metrics.EnableMetrics, false, metrics.EnableMetricsDescription)
	metricsAddress   = flag.String(
		metrics.MetricsAddress,
		metrics.MetricsAddressDefault,
		metrics.MetricsAddressDescription,
	)
	readyChan = make(chan bool, 1)
	termChan  = make(chan bool, 1)
	sigChan   = make(chan os.Signal, 1)
)

var (
	Project   = "infra-onboarding/dkam"
	RepoURL   = fmt.Sprintf("https://github.com/open-edge-platform/%s.git", Project)
	Version   = "<unset>"
	Revision  = "<unset>"
	BuildDate = "<unset>"
)

func printSummary() {
	zlog.Info().Msg("Starting DKAM")
	zlog.InfraSec().Info().Msgf("RepoURL: %s, Version: %s, Revision: %s, BuildDate: %s\n",
		RepoURL, Version, Revision, BuildDate)
}

func main() {
	watcher, watcherErr := SetWatcher()
	if watcherErr != nil {
		zlog.InfraSec().Fatal().Err(watcherErr).Msgf("Failed to set watcher.")
		return
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			zlog.InfraSec().Error().Err(err).Msg("Failed to close watcher")
		}
	}()

	// Print a summary of the build
	printSummary()
	flag.Parse()
	if err := config.Read(); err != nil {
		zlog.InfraSec().Fatal().Err(err).Msgf("Failed to read config")
	}

	setupTracingIfEnabled()

	if *enableMetrics {
		startMetricsServer()
	}

	if err := GetArtifacts(context.Background()); err != nil {
		zlog.InfraSec().Fatal().Err(err).Msg("Failed to get artifacts")
	}

	go func() {
		defer wg.Done()
		if err := BuildBinaries(); err != nil {
			zlog.InfraSec().Fatal().Err(err).Msg("Failed to get artifacts")
		}
	}()

	setupOamServerAndSetReady(*enableTracing, *oamServerAddress)

	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	<-sigChan // blocking
	close(termChan)
}

func setupTracingIfEnabled() {
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
				zlog.InfraSec().Fatal().Err(err).Msg("Cannot start Inventory OAM gRPC server")
			}
		}()
		readyChan <- true
	}
}

func GetArtifacts(ctx context.Context) error {
	outDir := filepath.Join(config.DownloadPath, "tmp")
	// 0. cleanup
	_ = os.RemoveAll(outDir)
	zlog.InfraSec().Info().Msg("Get all artifacts...")
	// Download release manifest.yaml file.
	artifactsErr := dkammgr.DownloadArtifacts(ctx)
	if artifactsErr != nil {
		zlog.InfraSec().Fatal().Err(artifactsErr).Msgf("Error downloading file %v", artifactsErr)
		return artifactsErr
	}
	return nil
}

func BuildBinaries() error {
	// Download and sign iPXE
	signedIPXE, pxeErr := dkammgr.BuildSignIpxe()
	if pxeErr != nil {
		zlog.InfraSec().Fatal().Err(pxeErr).Msgf("Failed to sign MicroOS %v", pxeErr)
		return pxeErr
	}
	if signedIPXE {
		zlog.InfraSec().Info().Msg("Signed IPXE and moved to PVC")
	}

	// Download and sign MicroOS.
	signed, signerr := dkammgr.SignMicroOS()
	if signerr != nil {
		zlog.InfraSec().Fatal().Err(signerr).Msgf("Failed to sign MicroOS %v", signerr)
		return signerr
	}
	if signed {
		zlog.InfraSec().Info().Msg("Signed MicroOS and moved to PVC")
	}
	return nil
}

func SetWatcher() (*fsnotify.Watcher, error) {
	zlog.InfraSec().Info().Msg("Enable watcher...")
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		zlog.InfraSec().Fatal().Err(err).Msgf("Failed to create a watcher")
		return nil, fmt.Errorf("failed to create a watcher: %w", err)
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
					zlog.InfraSec().Info().Msg("Certificate file changed. Rebuilding iPXE and microOS...")
					zlog.InfraSec().Fatal().Err(err).Msgf("Restart DKAM: %v", err)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				zlog.InfraSec().Fatal().Err(err).Msgf("Error:%v", err)
			}
		}
	}()

	return watcher, nil
}

func addFileToWatcher(watcher *fsnotify.Watcher, filename string) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		zlog.InfraSec().Fatal().Err(err).Msgf("File does not exist:%s, error: %v", filename, err)
		return
	}
	if err := watcher.Add(filename); err != nil {
		zlog.InfraSec().Error().Msgf("Failed to add file to watcher:%s, error: %v", filename, err)
	} else {
		zlog.InfraSec().Info().Msgf("Watcher added for file:%s", filename)
	}
}
