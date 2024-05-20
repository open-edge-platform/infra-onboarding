// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel
package main

import (
	// import dependencies

	"flag"
	"net"
	"os"

	"github.com/fsnotify/fsnotify"
	"google.golang.org/grpc"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/internal/dkammgr"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/api/dkammgr/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/config"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
)

var (
	zlog     = logging.GetLogger("MIDKAMMain")
	servaddr = flag.String(config.ServerAddress, config.Port, config.ServerAddressDescription)
)

func main() {

	// Create a certificate watcher
	watcher, watcherErr := SetWatcher()
	if watcherErr != nil {
		zlog.MiSec().Fatal().Err(watcherErr).Msgf("Failed to set watcher.")
		return
	}
	defer watcher.Close()

	//Download OS image
	downloadErr := dkammgr.DownloadOS()
	if downloadErr != nil {
		zlog.MiSec().Fatal().Err(downloadErr).Msgf("Error downloading and converting OS image")
		return
	}
	zlog.MiSec().Info().Msg("Completed dkammgr.DownloadOS")

	//Download release manifest.yaml file.
	artifactsErr := dkammgr.DownloadArtifacts()
	if artifactsErr != nil {
		zlog.MiSec().Error().Err(artifactsErr).Msgf("Error downloading file")

	}
	zlog.MiSec().Info().Msg("Completed dkammgr.DownloadArtifacts")

	//Donwload and sign iPXE
	signedIPXE, pxeErr := dkammgr.BuildSignIpxe()
	if pxeErr != nil {
		zlog.MiSec().Error().Err(pxeErr).Msgf("Failed to sign MicroOS %v", pxeErr)
		return
	}
	zlog.MiSec().Info().Msg("Completed dkammgr.BuildSignIpxe")

	if signedIPXE {
		zlog.MiSec().Info().Msgf("Signed MicroOS and moved to PVC")
	}

	// Download and sign MicroOS.
	signed, signerr := dkammgr.SignMicroOS()
	if signerr != nil {
		zlog.MiSec().Error().Err(signerr).Msgf("Failed to sign MicroOS")
	}
	zlog.MiSec().Info().Msg("Completed dkammgr.SignMicroOS")

	if signed {
		zlog.MiSec().Info().Msg("Signed MicroOS and moved to PVC")
	}

	// Set the port for DKAM Manager
	lis, err := net.Listen("tcp", *servaddr)
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error listening with TCP on address %s", *servaddr)
	}

	// Create a new gRPC server
	grpc_server := grpc.NewServer()

	// Register gRPC service implementation with the server
	pb.RegisterDkamServiceServer(grpc_server, &dkammgr.Service{})
	zlog.Info().Msgf("Starting gRPC server on port %s", *servaddr)
	if err := grpc_server.Serve(lis); err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Failed to serve: %v", err)
	}

	select {}
}

func SetWatcher() (*fsnotify.Watcher, error) {
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
