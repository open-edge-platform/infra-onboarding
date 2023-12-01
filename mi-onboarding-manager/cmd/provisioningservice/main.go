/*
   Copyright (C) 2023 Intel Corporation
   SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"log"
	"net"
	"os"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/provisioningproto"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/provisioningservice/onboarding"
	"google.golang.org/grpc"
)

// main function to start onboarding service
func main() {
	onb_addr := os.Getenv("MGR_HOST")
	onb_port := os.Getenv("ONBMGR_PORT")

	if onb_addr == "" || onb_port == "" {
		log.Printf("Invalid environment variables MGR_HOST and ONBMGR_PORT please export")
		os.Exit(1)
	}

	onb_server := onb_addr + ":" + onb_port
	// Start listener
	lis, err := net.Listen("tcp", onb_server)
	// lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterOnBoardingEBServer(grpcServer, &onboarding.OnboardingManager{})
	log.Printf("Server started at %v", lis.Addr())

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}
}
