/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	"context"
	"errors"
	"reflect"
	"testing"

	dkam "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/api/grpc/dkammgr"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/api"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	"github.com/stretchr/testify/mock"
)

func TestInitOnboarding(t *testing.T) {
	type args struct {
		invClient *invclient.OnboardingInventoryClient
		dkamAddr  string
	}
	mockInvClient := &MockInventoryClient{}
	inputargs := args{
		invClient: &invclient.OnboardingInventoryClient{
			Client: mockInvClient,
		},
	}
	inputargs1 := args{
		invClient: nil,
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "positive",
			args: inputargs,
		},
		{
			name: "negative",
			args: inputargs1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitOnboarding(tt.args.invClient, tt.args.dkamAddr)
		})
	}
}

func TestGetOSResourceFromDkamService(t *testing.T) {
	type args struct {
		ctx         context.Context
		profilename string
		platform    string
	}
	tests := []struct {
		name    string
		args    args
		want    *dkam.GetArtifactsResponse
		wantErr bool
	}{
		{
			name: "TestCase1",
			args: args{
				ctx: context.TODO(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "TestCase2",
			args: args{
				ctx: context.TODO(),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetOSResourceFromDkamService(tt.args.ctx, tt.args.profilename, tt.args.platform)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetOSResourceFromDkamService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetOSResourceFromDkamService() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandleSecureBootMismatch(t *testing.T) {
	type args struct {
		ctx context.Context
		req *pb.SecureBootResponse
	}
	mockClient := &MockInventoryClient{}
	_invClient = &invclient.OnboardingInventoryClient{
		Client: mockClient,
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				ctx: context.Background(),
				req: &pb.SecureBootResponse{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := IsSecureBootConfigAtEdgeNodeMismatch(tt.args.ctx, tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf("HandleSecureBootMismatch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandleSecureBootMismatch_Case(t *testing.T) {
	type args struct {
		ctx context.Context
		req *pb.SecureBootResponse
	}
	mockClient := &MockInventoryClient{}
	mockHost := &computev1.HostResource{
		ResourceId: "host-084d9b08",
		Instance: &computev1.InstanceResource{
			ResourceId: "inst-084d9b08",
		},
	}
	mockResource := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: mockHost,
		},
	}
	mockResources := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource}},
	}
	mockClient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil)
	mockClient.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	_invClient = &invclient.OnboardingInventoryClient{
		Client: mockClient,
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				ctx: context.Background(),
				req: &pb.SecureBootResponse{
					Guid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := IsSecureBootConfigAtEdgeNodeMismatch(tt.args.ctx, tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf("HandleSecureBootMismatch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandleSecureBootMismatch_Case1(t *testing.T) {
	type args struct {
		ctx context.Context
		req *pb.SecureBootResponse
	}
	mockClient := &MockInventoryClient{}
	mockHost := &computev1.HostResource{
		ResourceId: "host-084d9b08",
		Instance: &computev1.InstanceResource{
			ResourceId: "inst-084d9b08",
		},
	}
	mockResource := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: mockHost,
		},
	}
	mockResources := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource}},
	}
	mockClient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil)
	mockClient.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, errors.New("err"))
	_invClient = &invclient.OnboardingInventoryClient{
		Client: mockClient,
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				ctx: context.Background(),
				req: &pb.SecureBootResponse{
					Guid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := IsSecureBootConfigAtEdgeNodeMismatch(tt.args.ctx, tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf("HandleSecureBootMismatch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandleSecureBootMismatch_Case2(t *testing.T) {
	type args struct {
		ctx context.Context
		req *pb.SecureBootResponse
	}
	mockClient := &MockInventoryClient{}
	mockHost := &computev1.HostResource{
		ResourceId: "host-084d9b08",
		Instance: &computev1.InstanceResource{
			ResourceId: "inst-084d9b08",
		},
	}
	mockResource := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: mockHost,
		},
	}
	mockResources := &inv_v1.ListResourcesResponse{
		Resources: []*inv_v1.GetResourceResponse{{Resource: mockResource}},
	}
	mockClient.On("List", mock.Anything, mock.Anything, mock.Anything).Return(mockResources, nil)
	mockClient.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil).Once()
	mockClient.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, errors.New("err")).Once()
	_invClient = &invclient.OnboardingInventoryClient{
		Client: mockClient,
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				ctx: context.Background(),
				req: &pb.SecureBootResponse{
					Guid: "9fa8a788-f9f8-434a-8620-bbed2a12b0ad",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := IsSecureBootConfigAtEdgeNodeMismatch(tt.args.ctx, tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf("HandleSecureBootMismatch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOnboardingManager_SecureBootStatus(t *testing.T) {
	type fields struct {
		OnBoardingSBServer pb.OnBoardingSBServer
	}
	type args struct {
		ctx context.Context
		req *pb.SecureBootStatRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.SecureBootResponse
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				ctx: context.Background(),
				req: &pb.SecureBootStatRequest{},
			},
			want:    &pb.SecureBootResponse{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &OnboardingManager{
				OnBoardingSBServer: tt.fields.OnBoardingSBServer,
			}
			got, err := s.SecureBootStatus(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingManager.SecureBootStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OnboardingManager.SecureBootStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}
