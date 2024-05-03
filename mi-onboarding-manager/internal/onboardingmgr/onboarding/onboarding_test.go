/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	"context"
	"reflect"
	"testing"

	dkam "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/api/dkammgr/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	onboarding "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/onboarding/onboardingmocks"
)

const rbacRules = "../../../rego/authz.rego"

func TestInitOnboarding(t *testing.T) {
	type args struct {
		invClient  *invclient.OnboardingInventoryClient
		dkamAddr   string
		enableAuth bool
		rbac       string
	}
	mockInvClient := &onboarding.MockInventoryClient{}
	inputargs := args{
		invClient: &invclient.OnboardingInventoryClient{
			Client: mockInvClient,
		},
		enableAuth: true,
		rbac:       rbacRules,
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
			InitOnboarding(tt.args.invClient, tt.args.dkamAddr, tt.args.enableAuth, tt.args.rbac)
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
