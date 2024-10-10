/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	"testing"

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
