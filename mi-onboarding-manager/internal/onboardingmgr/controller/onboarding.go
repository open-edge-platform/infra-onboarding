/*
   Copyright (C) 2023 Intel Corporation
   SPDX-License-Identifier: Apache-2.0
*/

package controller

import (
	"context"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/config"
	repository "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/persistence"
)

type (
	InventoryManager struct {
		nodeRepo     repository.Repository
		artifactRepo repository.Repository
	}
)

func NewInventoryManager(conf *config.Config) *InventoryManager {
	im := &InventoryManager{}
	//repository.InitDB(conf)
	im.nodeRepo = repository.GetNodeRepository()
	im.artifactRepo = repository.GetArtifactRepository()
	return im
}

func (im InventoryManager) AddArtifacts(ctx context.Context, data []repository.ArtifactData) ([]repository.ArtifactData, error) {
	return im.artifactRepo.CreateArtifacts(ctx, data)
}

func (im InventoryManager) UpdateArtifacts(ctx context.Context, data []repository.ArtifactData) error {
	return im.artifactRepo.UpdateArtifacts(ctx, data)
}

func (im InventoryManager) DeleteArtifacts(ctx context.Context, ids []string) error {
	return im.artifactRepo.DeleteArtifacts(ctx, ids)
}

func (im InventoryManager) GetArtifacts(ctx context.Context, data repository.ArtifactData) ([]*repository.ArtifactData, error) {
	return im.artifactRepo.GetArtifacts(ctx, data)
}

func (im InventoryManager) AddNodes(ctx context.Context, data []repository.NodeData) ([]repository.NodeData, error) {
	return im.nodeRepo.CreateNodes(ctx, data)
}

func (im InventoryManager) UpdateNodes(ctx context.Context, data []repository.NodeData) error {
	return im.nodeRepo.UpdateNodes(ctx, data)
}

func (im InventoryManager) DeleteNodes(ctx context.Context, ids []string) error {
	return im.nodeRepo.DeleteNodes(ctx, ids)
}

func (im InventoryManager) GetNodes(ctx context.Context, data repository.NodeData) ([]*repository.NodeData, error) {
	return im.nodeRepo.GetNodes(ctx, data)
}
