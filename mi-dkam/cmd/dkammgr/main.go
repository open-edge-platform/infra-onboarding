/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: LicenseRef-Intel
 */
package main

import (
	// import dependencies
	"net"
	"log"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/api/grpc/dkammgr"
         "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/internal/dkammgr"
	"google.golang.org/grpc"
)

const (
	port       = ":5581" //gRPC port
)

func main() {

	// Set the port for DKAM Manager
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal("Failed to listen", err)
	}

	// Create a new gRPC server 
	grpc_server := grpc.NewServer()


	// Register gRPC service implementation with the server
	pb.RegisterDkamServiceServer(grpc_server, &dkammgr.Service{})
	log.Println("Starting gRPC server on port", port)
	if err := grpc_server.Serve(lis); err != nil {
		log.Fatal("Failed to serve", err)
	}
}
