/*
   Copyright (C) 2023 Intel Corporation
   SPDX-License-Identifier: Apache-2.0
*/

package controller

import (
	"context"
	"reflect"
	"testing"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/config"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/persistence"
	"github.com/stretchr/testify/mock"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateNodes(ctx context.Context, data []persistence.NodeData) ([]persistence.NodeData, error) {
	args := m.Called(ctx, data)
	return args.Get(0).([]persistence.NodeData), args.Error(1)
}

func (m *MockRepository) UpdateNodes(ctx context.Context, data []persistence.NodeData) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *MockRepository) GetNodes(ctx context.Context, data persistence.NodeData) ([]*persistence.NodeData, error) {
	args := m.Called(ctx, data)
	return args.Get(0).([]*persistence.NodeData), args.Error(1)
}

func (m *MockRepository) DeleteNodes(ctx context.Context, ids []string) error {
	args := m.Called(ctx, ids)
	return args.Error(0)
}

func (m *MockRepository) CreateArtifacts(ctx context.Context,
	data []persistence.ArtifactData,
) ([]persistence.ArtifactData, error) {
	args := m.Called(ctx, data)
	return args.Get(0).([]persistence.ArtifactData), args.Error(1)
}

func (m *MockRepository) UpdateArtifacts(ctx context.Context, data []persistence.ArtifactData) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *MockRepository) GetArtifacts(ctx context.Context, data persistence.ArtifactData) ([]*persistence.ArtifactData, error) {
	args := m.Called(ctx, data)
	return args.Get(0).([]*persistence.ArtifactData), args.Error(1)
}

func (m *MockRepository) DeleteArtifacts(ctx context.Context, ids []string) error {
	args := m.Called(ctx, ids)
	return args.Error(0)
}

func (m *MockRepository) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestNewInventoryManager(t *testing.T) {
	type args struct {
		conf *config.Config
	}
	tests := []struct {
		name string
		args args
		want *InventoryManager
	}{
		{
			name: "Test with valid config",
			args: args{
				conf: &config.Config{},
			},
			want: &InventoryManager{
				nodeRepo:     nil,
				artifactRepo: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewInventoryManager(tt.args.conf); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewInventoryManager() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInventoryManager_AddArtifacts(t *testing.T) {
	type fields struct {
		nodeRepo     persistence.Repository
		artifactRepo persistence.Repository
	}
	mockRepo := &MockRepository{}
	mockRepo.On("CreateArtifacts", mock.Anything, mock.Anything).Return([]persistence.ArtifactData{}, nil)
	type args struct {
		ctx  context.Context
		data []persistence.ArtifactData
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []persistence.ArtifactData
		wantErr bool
	}{
		{
			name: "TestCase",
			fields: fields{
				nodeRepo:     mockRepo,
				artifactRepo: mockRepo,
			},
			args: args{
				ctx: context.Background(),
				data: []persistence.ArtifactData{
					{ID: "1", Name: "Artifact1"},
					{ID: "2", Name: "Artifact2"},
				},
			},
			want:    []persistence.ArtifactData{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := InventoryManager{
				nodeRepo:     tt.fields.nodeRepo,
				artifactRepo: tt.fields.artifactRepo,
			}
			got, err := im.AddArtifacts(tt.args.ctx, tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("InventoryManager.AddArtifacts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InventoryManager.AddArtifacts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInventoryManager_UpdateArtifacts(t *testing.T) {
	type fields struct {
		nodeRepo     persistence.Repository
		artifactRepo persistence.Repository
	}
	mockRepo := &MockRepository{}
	mockRepo.On("UpdateArtifacts", mock.Anything, mock.Anything).Return(nil)
	type args struct {
		ctx  context.Context
		data []persistence.ArtifactData
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "TestCase",
			fields: fields{
				nodeRepo:     mockRepo,
				artifactRepo: mockRepo,
			},
			args: args{
				ctx: context.Background(),
				data: []persistence.ArtifactData{
					{ID: "1", Name: "Artifact1"},
					{ID: "2", Name: "Artifact2"},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := InventoryManager{
				nodeRepo:     tt.fields.nodeRepo,
				artifactRepo: tt.fields.artifactRepo,
			}
			if err := im.UpdateArtifacts(tt.args.ctx, tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("InventoryManager.UpdateArtifacts() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInventoryManager_DeleteArtifacts(t *testing.T) {
	type fields struct {
		nodeRepo     persistence.Repository
		artifactRepo persistence.Repository
	}
	mockRepo := &MockRepository{}
	mockRepo.On("DeleteArtifacts", mock.Anything, mock.Anything).Return(nil)
	type args struct {
		ctx context.Context
		ids []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "TestCase",
			fields: fields{
				nodeRepo:     mockRepo,
				artifactRepo: mockRepo,
			},
			args: args{
				ctx: context.Background(),
				ids: []string{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := InventoryManager{
				nodeRepo:     tt.fields.nodeRepo,
				artifactRepo: tt.fields.artifactRepo,
			}
			if err := im.DeleteArtifacts(tt.args.ctx, tt.args.ids); (err != nil) != tt.wantErr {
				t.Errorf("InventoryManager.DeleteArtifacts() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInventoryManager_GetArtifacts(t *testing.T) {
	type fields struct {
		nodeRepo     persistence.Repository
		artifactRepo persistence.Repository
	}
	type args struct {
		ctx  context.Context
		data persistence.ArtifactData
	}
	mockRepo := &MockRepository{}
	mockRepo.On("GetArtifacts", mock.Anything, mock.Anything).Return([]*persistence.ArtifactData{}, nil)
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*persistence.ArtifactData
		wantErr bool
	}{
		{
			name: "TestCase",
			fields: fields{
				nodeRepo:     mockRepo,
				artifactRepo: mockRepo,
			},
			args: args{
				ctx:  context.Background(),
				data: persistence.ArtifactData{},
			},
			want:    []*persistence.ArtifactData{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := InventoryManager{
				nodeRepo:     tt.fields.nodeRepo,
				artifactRepo: tt.fields.artifactRepo,
			}
			got, err := im.GetArtifacts(tt.args.ctx, tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("InventoryManager.GetArtifacts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InventoryManager.GetArtifacts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInventoryManager_AddNodes(t *testing.T) {
	type fields struct {
		nodeRepo     persistence.Repository
		artifactRepo persistence.Repository
	}
	type args struct {
		ctx  context.Context
		data []persistence.NodeData
	}
	mockRepo := &MockRepository{}
	mockRepo.On("CreateNodes", mock.Anything, mock.Anything).Return([]persistence.NodeData{}, nil)
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []persistence.NodeData
		wantErr bool
	}{
		{
			name: "TestCase",
			fields: fields{
				nodeRepo:     mockRepo,
				artifactRepo: mockRepo,
			},
			args: args{
				ctx:  context.Background(),
				data: []persistence.NodeData{},
			},
			want:    []persistence.NodeData{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := InventoryManager{
				nodeRepo:     tt.fields.nodeRepo,
				artifactRepo: tt.fields.artifactRepo,
			}
			got, err := im.AddNodes(tt.args.ctx, tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("InventoryManager.AddNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InventoryManager.AddNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInventoryManager_UpdateNodes(t *testing.T) {
	type fields struct {
		nodeRepo     persistence.Repository
		artifactRepo persistence.Repository
	}
	mockRepo := &MockRepository{}
	mockRepo.On("UpdateNodes", mock.Anything, mock.Anything).Return(nil)
	type args struct {
		ctx  context.Context
		data []persistence.NodeData
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "TestCase",
			fields: fields{
				nodeRepo:     mockRepo,
				artifactRepo: mockRepo,
			},
			args: args{
				ctx:  context.Background(),
				data: []persistence.NodeData{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := InventoryManager{
				nodeRepo:     tt.fields.nodeRepo,
				artifactRepo: tt.fields.artifactRepo,
			}
			if err := im.UpdateNodes(tt.args.ctx, tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("InventoryManager.UpdateNodes() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInventoryManager_DeleteNodes(t *testing.T) {
	type fields struct {
		nodeRepo     persistence.Repository
		artifactRepo persistence.Repository
	}
	mockRepo := &MockRepository{}
	mockRepo.On("DeleteNodes", mock.Anything, mock.Anything).Return(nil)
	type args struct {
		ctx context.Context
		ids []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "TestCase",
			fields: fields{
				nodeRepo:     mockRepo,
				artifactRepo: mockRepo,
			},
			args: args{
				ctx: context.Background(),
				ids: []string{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := InventoryManager{
				nodeRepo:     tt.fields.nodeRepo,
				artifactRepo: tt.fields.artifactRepo,
			}
			if err := im.DeleteNodes(tt.args.ctx, tt.args.ids); (err != nil) != tt.wantErr {
				t.Errorf("InventoryManager.DeleteNodes() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInventoryManager_GetNodes(t *testing.T) {
	type fields struct {
		nodeRepo     persistence.Repository
		artifactRepo persistence.Repository
	}
	type args struct {
		ctx  context.Context
		data persistence.NodeData
	}
	mockRepo := &MockRepository{}
	mockRepo.On("GetNodes", mock.Anything, mock.Anything).Return([]*persistence.NodeData{}, nil)
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*persistence.NodeData
		wantErr bool
	}{
		{
			name: "TestCase",
			fields: fields{
				nodeRepo:     mockRepo,
				artifactRepo: mockRepo,
			},
			args: args{
				ctx:  context.Background(),
				data: persistence.NodeData{},
			},
			want:    []*persistence.NodeData{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := InventoryManager{
				nodeRepo:     tt.fields.nodeRepo,
				artifactRepo: tt.fields.artifactRepo,
			}
			got, err := im.GetNodes(tt.args.ctx, tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("InventoryManager.GetNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InventoryManager.GetNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}
