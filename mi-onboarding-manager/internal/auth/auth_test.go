// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package auth

import (
	"context"
	"flag"
	"testing"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/common"
)

func Test_auth_init(t *testing.T) {
	type args struct {
		ctx             context.Context
		disableCredMgmt bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test Case enabled",
			args: args{
				ctx:             context.Background(),
				disableCredMgmt: false,
			},
			wantErr: true,
		},
		{
			name: "Test Case disabled",
			args: args{
				ctx:             context.Background(),
				disableCredMgmt: true,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			common.FlagDisableCredentialsManagement = flag.Bool(tt.name, tt.args.disableCredMgmt, "")
			if err := Init(); (err != nil) != tt.wantErr {
				t.Errorf("auth.Init() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	// ensure the default value for the other tests
	common.FlagDisableCredentialsManagement = flag.Bool("", false, "")
}
