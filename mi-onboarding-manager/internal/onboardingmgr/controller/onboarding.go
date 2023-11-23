/*
   Copyright (C) 2023 Intel Corporation
   SPDX-License-Identifier: Apache-2.0
*/

package controller

import (
	"context"

	"github.com/intel-sandbox/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/config"
	repository "github.com/intel-sandbox/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/persistence"
)

type (
	InventoryManager struct {
		nodeRepo     repository.Repository
		artifactRepo repository.Repository
	}
)

func NewInventoryManager(conf *config.Config) *InventoryManager {
	im := &InventoryManager{}
	repository.InitDB(conf)
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

func (im InventoryManager) AddProfiles(ctx context.Context, data []repository.ProfileData) ([]repository.ProfileData, error) {
	return im.artifactRepo.CreateProfiles(ctx, data)
}

func (im InventoryManager) UpdateProfiles(ctx context.Context, data []repository.ProfileData) error {
	return im.artifactRepo.UpdateProfiles(ctx, data)
}

func (im InventoryManager) DeleteProfiles(ctx context.Context, ids []string) error {
	return im.artifactRepo.DeleteProfiles(ctx, ids)
}

func (im InventoryManager) GetProfiles(ctx context.Context, data repository.ProfileData) ([]*repository.ProfileData, error) {
	return im.artifactRepo.GetProfiles(ctx, data)
}

func (im InventoryManager) AddGroups(ctx context.Context, data []repository.GroupData) ([]repository.GroupData, error) {
	return im.artifactRepo.CreateGroups(ctx, data)
}

func (im InventoryManager) UpdateGroups(ctx context.Context, data []repository.GroupData) error {
	return im.artifactRepo.UpdateGroups(ctx, data)
}

func (im InventoryManager) DeleteGroups(ctx context.Context, ids []string) error {
	return im.artifactRepo.DeleteGroups(ctx, ids)
}

func (im InventoryManager) GetGroups(ctx context.Context, data repository.GroupData) ([]*repository.GroupData, error) {
	return im.artifactRepo.GetGroups(ctx, data)
}
