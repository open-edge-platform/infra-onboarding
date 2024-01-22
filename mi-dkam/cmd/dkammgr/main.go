// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel
package main

import (
	// import dependencies

	"flag"
	"net"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/api/grpc/dkammgr"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/internal/dkammgr"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/config"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/logging"
	"google.golang.org/grpc"
)

var (
	zlog     = logging.GetLogger("MIDKAMMain")
	servaddr = flag.String(config.ServerAddress, config.Port, config.ServerAddressDescription)
)

func main() {
	//Download release manifest.yaml file.
	err := dkammgr.DownloadArtifacts()
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error downloading file")
		return
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
}
