/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	"context"
	"os"
	"reflect"
	"testing"

	dkam "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/api/dkammgr/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
)

const rbacRules = "../../../rego/authz.rego"

func TestInitOnboarding(t *testing.T) {
	type args struct {
		invClient  *invclient.OnboardingInventoryClient
		dkamAddr   string
		enableAuth bool
		rbac       string
	}
	inputargs := args{
		invClient:  &invclient.OnboardingInventoryClient{},
		enableAuth: true,
		rbac:       rbacRules,
	}
	inputargs1 := args{
		invClient:  nil,
		enableAuth: true,
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "SuccessfulInitialization",
			args: inputargs,
		},
		{
			name: "MissingInventoryClient",
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
		ctx               context.Context
		repoURL           string
		sha256            string
		profilename       string
		installedPackages string
		platform          string
		kernelCommand     string
	}
	tests := []struct {
		name    string
		args    args
		want    *dkam.GetENProfileResponse
		wantErr bool
	}{
		{
			name: "Test Case with empty host and port",
			args: args{
				ctx: context.TODO(),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetOSResourceFromDkamService(tt.args.ctx, tt.args.repoURL, tt.args.sha256, tt.args.profilename, tt.args.installedPackages, tt.args.platform, tt.args.kernelCommand)
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

func TestGetOSResourceFromDkamService_Case1(t *testing.T) {
	os.Setenv("DKAMHOST", "00.00.00.000")
	os.Setenv("DKAMPORT", "00.00.00.000")
	type args struct {
		ctx               context.Context
		repoURL           string
		sha256            string
		profilename       string
		installedPackages string
		platform          string
		kernelCommand     string
	}
	tests := []struct {
		name    string
		args    args
		want    *dkam.GetENProfileResponse
		wantErr bool
	}{
		{
			name: "Test Case with host and port",
			args: args{
				ctx: context.TODO(),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetOSResourceFromDkamService(tt.args.ctx, tt.args.repoURL, tt.args.sha256, tt.args.profilename, tt.args.installedPackages, tt.args.platform, tt.args.kernelCommand)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetOSResourceFromDkamService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetOSResourceFromDkamService() = %v, want %v", got, tt.want)
			}
		})
	}
	defer func() {
		os.Unsetenv("DKAMHOST")
		os.Unsetenv("DKAMPORT")
	}()
}
