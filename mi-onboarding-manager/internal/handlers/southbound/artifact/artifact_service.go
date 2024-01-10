// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package artifact

import (
	"context"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/onboardingmgr"
	repository "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/persistence"
)

var (
	name = "NodeArtifactService"
	zlog = logging.GetLogger(name)
)

type NodeArtifactService struct {
	pb.UnimplementedNodeArtifactServiceNBServer
}

func CopyNodeReqtoNodetData(payload []*pb.NodeData) ([]repository.NodeData, error) {
	zlog.Info().Msg("CopyNodeReqtoNodetData")
	zlog.Info().Msgf("%d", len(payload))
	var data []repository.NodeData
	for _, s := range payload {
		art := repository.NodeData{
			ID:           s.NodeId,
			HwID:         s.HwId,
			PlatformType: s.PlatformType,
			DeviceType:   s.DeviceType,
		}
		zlog.Info().Msgf("HwID %d", s.HwId)
		zlog.Info().Msgf("NodeId %d", s.NodeId)
		data = append(data, art)
	}

	zlog.Info().Msgf("%d", len(data))
	return data, nil
}

func CopyNodeDatatoNodeResp(payload []repository.NodeData, result string) ([]*pb.NodeData, error) {
	zlog.Info().Msg("CopyNodeDatatoNodeResp")
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
	zlog.Info().Msg("CreateNodes")

	return &pb.NodeResponse{Payload: req.Payload}, nil
}

func (s *NodeArtifactService) DeleteNodes(ctx context.Context, req *pb.NodeRequest) (*pb.NodeResponse, error) {
	// Handle the RPC and return a response
	//TODO to be implemented
	return &pb.NodeResponse{Payload: req.Payload}, nil
}

func (s *NodeArtifactService) GetNodes(ctx context.Context, req *pb.NodeRequest) (*pb.NodeResponse, error) {
	// Handle the RPC and return a response
	zlog.Info().Msg("GetNodes")

	return &pb.NodeResponse{Payload: req.Payload}, nil
}

func (s *NodeArtifactService) UpdateNodes(ctx context.Context, req *pb.NodeRequest) (*pb.NodeResponse, error) {
	// Handle the RPC and return a response
	zlog.Info().Msg("UpdateNodes")

	return &pb.NodeResponse{Payload: req.Payload}, nil
}
