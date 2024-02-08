/*
   Copyright (C) 2023 Intel Corporation
   SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"reflect"
	"testing"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/onboardingmgr"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

type MockOnBoardingEBClient struct {
	mock.Mock
}

func (m *MockOnBoardingEBClient) StartOnboarding(ctx context.Context, req *pb.OnboardingRequest,
	_ ...grpc.CallOption,
) (*pb.OnboardingResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*pb.OnboardingResponse), args.Error(1)
}

func TestOnboardingTest(t *testing.T) {
	type args struct {
		client pb.OnBoardingEBClient
	}
	mockClient := &MockOnBoardingEBClient{}
	mockClient.On("StartOnboarding", mock.Anything, mock.Anything).Return(&pb.OnboardingResponse{}, nil)
	tests := []struct {
		name    string
		args    args
		want    *pb.OnboardingResponse
		wantErr bool
	}{
		{
			name: "Test Case 1",
			args: args{
				client: mockClient,
			},
			want:    &pb.OnboardingResponse{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := OnboardingTest(tt.args.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardingTest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OnboardingTest() = %v, want %v", got, tt.want)
			}
		})
	}
}
