// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package secrets

import (
	"context"
	"flag"
	"testing"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/common"
)

func TestInit(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "Test Case",
			args:    args{ctx: context.Background()},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Init(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Init() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetClientID(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "Test Case",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetClientID(); got != tt.want {
				t.Errorf("GetClientID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetClientSecret(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "Test Case",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetClientSecret(); got != tt.want {
				t.Errorf("GetClientSecret() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_secretService_init(t *testing.T) {
	common.FlagDisableCredentialsManagement = flag.Bool("name", true, "")
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		ss      *secretService
		args    args
		wantErr bool
	}{
		{
			name: "test case",
			ss:   &secretService{},
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := &secretService{}
			if err := ss.init(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("secretService.init() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	defer func(){
		common.FlagDisableCredentialsManagement = flag.Bool("", false, "")
	}()
}

