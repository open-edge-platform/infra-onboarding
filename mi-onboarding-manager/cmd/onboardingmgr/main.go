/*
   Copyright (C) 2023 Intel Corporation
   SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"flag"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/onboardingmgr"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/config"
	inventory "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/controller"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/onboarding"
	repository "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/persistence"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/pkg/logger"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/pkg/maestro"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/pkg/maestro/controller"

	"google.golang.org/grpc"
)

var (
	log        = logger.GetLogger()
	invSvrAddr = flag.String("invsvraddr", "localhost:50051", "Inventory server address to connect to")
)
var manager *inventory.InventoryManager

type NodeArtifactService struct {
	pb.UnimplementedNodeArtifactServiceNBServer
}

type OnboardingEB struct {
	pb.UnimplementedOnBoardingEBServer
}

func CopyNodeReqtoNodetData(payload []*pb.NodeData) ([]repository.NodeData, error) {

	log.Info("CopyNodeReqtoNodetData")
	log.Infof("%d", len(payload))
	var data []repository.NodeData
	for _, s := range payload {
		art := repository.NodeData{
			ID:           s.NodeId,
			HwID:         s.HwId,
			PlatformType: s.PlatformType,
			DeviceType:   s.DeviceType,
		}
		log.Infof("HwID %d", s.HwId)
		log.Infof("NodeId %d", s.NodeId)
		data = append(data, art)
	}

	log.Infof("%d", len(data))
	return data, nil
}

func CopyNodeDatatoNodeResp(payload []repository.NodeData, result string) ([]*pb.NodeData, error) {
	log.Info("CopyNodeDatatoNodeResp")
	var data []*pb.NodeData
	for _, s := range payload {
		art2 := pb.NodeData{
			NodeId:          s.ID,
			HwId:            s.HwID,
			FwArtifactId:    s.FwArtID,
			OsArtifactId:    s.OsArtID,
			AppArtifactId:   s.AppArtID,
			PlatArtifactId:  s.PlatformArtID,
			PlatformType:    s.PlatformType,
			DeviceType:      s.DeviceType,
			DeviceInfoAgent: s.DeviceInfoAgent,
			DeviceStatus:    s.DeviceStatus,
		}
		if result == "SUCCESS" {
			art2.Result = 0
		} else {
			art2.Result = 1
		}
		data = append(data, &art2)
	}
	return data, nil
}

func (s *NodeArtifactService) CreateNodes(ctx context.Context, req *pb.NodeRequest) (*pb.NodeResponse, error) {

	log.Info("CreateNodes")

	return &pb.NodeResponse{Payload: req.Payload}, nil
}

func (s *NodeArtifactService) DeleteNodes(ctx context.Context, req *pb.NodeRequest) (*pb.NodeResponse, error) {
	// Handle the RPC and return a response
	//TODO to be implemented
	return &pb.NodeResponse{Payload: req.Payload}, nil
}

func (s *NodeArtifactService) GetNodes(ctx context.Context, req *pb.NodeRequest) (*pb.NodeResponse, error) {
	// Handle the RPC and return a response
	log.Info("GetNodes")

	return &pb.NodeResponse{Payload: req.Payload}, nil
}

func (s *NodeArtifactService) UpdateNodes(ctx context.Context, req *pb.NodeRequest) (*pb.NodeResponse, error) {
	// Handle the RPC and return a response

	log.Info("UpdateNodes")

	return &pb.NodeResponse{Payload: req.Payload}, nil
}

func main() {
	config.Load()
	conf := config.GetConfig()
	manager = inventory.NewInventoryManager(conf)

	_, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()

	listenAddr := ":50054" // Set the port
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterNodeArtifactServiceNBServer(s, &NodeArtifactService{})
	pb.RegisterOnBoardingEBServer(s, &onboarding.OnboardingManager{})
	//Run go routine to start the gRPC server
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatal("gRPC server failed:")
		}
	}()

	// Set up a signal handler to catch termination signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	flag.Parse()
	wg := &sync.WaitGroup{}
	ctx := context.Background()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	termChan := make(chan bool)

	invClient, invEvents, err := maestro.NewInventoryClient(ctx, wg, termChan, *invSvrAddr)
	if err != nil {
		log.Fatalf("failed to create NewInventoryClient: %v", err)
	}

	nbHandler, err := controller.NewNBHandler(invClient, invEvents)
	if err != nil {
		log.Fatalf("failed to create NBHandler: %v", err)
	}

	err = nbHandler.Start()
	if err != nil {
		log.Fatalf("failed to start NBHandler: %v", err)
	}

	go func() {
		<-sigChan
		log.Info("Shutdown server")
		close(termChan)
		nbHandler.Stop()
		invClient.Close()
	}()

	wg.Wait()

	//Gracefully stop the gRPC server
	<-stop
	s.GracefulStop()

	log.Info("gRPC server stopped")
}
