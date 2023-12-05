/*
   Copyright (C) 2023 Intel Corporation
   SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"flag"
	"net"
	"time"

	"os"
	"os/signal"
	"sync"
	"syscall"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/onboardingmgr"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/config"
	inventory "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/controller"
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

type server struct {
	pb.UnimplementedNodeArtifactServiceNBServer
	pb.UnimplementedNodeServiceBBServer
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

func (s *server) CreateNodes(ctx context.Context, req *pb.NodeRequest) (*pb.NodeResponse, error) {

	log.Info("CreateNodes")
	result := "SUCCESS"

	data, _ := CopyNodeReqtoNodetData(req.Payload)

	log.Infof("CreateNodes %d", len(data))

	data, err := manager.AddNodes(ctx, data)

	if err != nil {
		log.Errorf("%v", err)
		result = "FAIL"
		return nil, err
	}

	for i := range data {
		log.Infof("ID - %s", data[i].ID)
	}

	req.Payload, err = CopyNodeDatatoNodeResp(data, result)

	return &pb.NodeResponse{Payload: req.Payload}, err
}

func (s *server) DeleteNodes(ctx context.Context, req *pb.NodeRequest) (*pb.NodeResponse, error) {
	// Handle the RPC and return a response
	//TODO to be implemented
	result := "SUCCESS"
	log.Info("DeleteNodes")

	dat, _ := CopyNodeReqtoNodetData(req.Payload)

	//1. Get the nodes satisfying the query
	nodedata, err := manager.GetNodes(ctx, dat[0])

	if err != nil {
		log.Infof("%v", err)
		result = "FAIL"
		return nil, err
	}

	//2. Populate the query
	var ID []string

	for j := range nodedata {
		//Get Node details for each of the node id
		ID = append(ID, nodedata[j].ID)
		log.Infof("ID %s j %d", ID[j], j)
		log.Infof("Nodedata[j].FwArtID %s", nodedata[j].FwArtID)
	}

	//3. Delete the nodes
	err = manager.DeleteNodes(ctx, ID)

	if err != nil {
		log.Infof("%v", err)
		result = "FAIL"
		return nil, err
	}

	req.Payload = nil
	if result == "SUCCESS" {
		req.Payload = append(req.Payload, &pb.NodeData{Result: 0})
	} else {
		req.Payload = append(req.Payload, &pb.NodeData{Result: 1})
	}

	return &pb.NodeResponse{Payload: req.Payload}, nil
}

func (s *server) GetNodes(ctx context.Context, req *pb.NodeRequest) (*pb.NodeResponse, error) {
	// Handle the RPC and return a response
	log.Info("GetNodes")

	result := "SUCCESS"

	dat, _ := CopyNodeReqtoNodetData(req.Payload)

	data, err := manager.GetNodes(ctx, dat[0])

	log.Info(dat[0].PlatformType)

	if err != nil {
		log.Errorf("%v", err)
		result = "FAIL"
		return nil, err
	}

	var resp pb.NodeResponse

	newSlice := make([]repository.NodeData, len(data))

	// Copy the values from the source slice to the destination slice
	for i, e := range data {
		if e == nil {
			continue
		}
		newSlice[i] = *e
	}

	resp.Payload, err = CopyNodeDatatoNodeResp(newSlice, result)

	return &pb.NodeResponse{Payload: resp.Payload}, err
}

func (s *server) UpdateNodes(ctx context.Context, req *pb.NodeRequest) (*pb.NodeResponse, error) {
	// Handle the RPC and return a response
	result := "SUCCESS"

	log.Info("UpdateNodes")

	for j := range req.Payload {
		//Get Node details for each of the node id
		var data repository.NodeData
		data.ID = req.Payload[j].NodeId
		log.Infof("ID %s j %d", req.Payload[j].NodeId, j)
		nodedata, err := manager.GetNodes(ctx, data)

		if err != nil {
			log.Errorf("%v", err)
			result = "FAIL"
			return nil, err
		}

		newSlice := make([]repository.NodeData, len(nodedata))

		// Copy the values from the source slice to the destination slice
		//TODO need to populate all parameters
		for i, e := range nodedata {
			if e == nil {
				continue
			}
			newSlice[i] = *e
			//populate the update parameters to the nodedata.
			if req.Payload[j].FwArtifactId != "" {
				newSlice[i].FwArtID = req.Payload[j].FwArtifactId

			}

			if req.Payload[j].OsArtifactId != "" {
				newSlice[i].OsArtID = req.Payload[j].OsArtifactId
			}

			if req.Payload[j].PlatArtifactId != "" {
				newSlice[i].PlatformArtID = req.Payload[j].PlatArtifactId
			}

			if req.Payload[j].HwId != "" {
				newSlice[i].HwID = req.Payload[j].HwId
			}

			if req.Payload[j].PlatformType != "" {
				newSlice[i].PlatformType = req.Payload[j].PlatformType
			}

			if req.Payload[j].DeviceType != "" {
				newSlice[i].DeviceType = req.Payload[j].DeviceType
			}
		}

		err = manager.UpdateNodes(ctx, newSlice)

		if err != nil {
			log.Errorf("%v", err)
			result = "FAIL"
			return nil, err
		}

	}

	req.Payload = nil
	if result == "SUCCESS" {
		req.Payload = append(req.Payload, &pb.NodeData{Result: 0})
	} else {
		req.Payload = append(req.Payload, &pb.NodeData{Result: 1})
	}
	return &pb.NodeResponse{Payload: req.Payload}, nil
}

func main() {
	config.Load()
	conf := config.GetConfig()
	manager = inventory.NewInventoryManager(conf)

	_, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()

	listenAddr := ":50052" // Set the port
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterNodeArtifactServiceNBServer(s, &server{})
	pb.RegisterNodeServiceBBServer(s, &server{})

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

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigChan
		log.Info("Shutdown server")
		close(termChan)
		nbHandler.Stop()
		invClient.Close()
	}()

	wg.Wait()

	// Gracefully stop the gRPC server
	<-stop
	s.GracefulStop()

	log.Info("gRPC server stopped")
}
