/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding_test

import (
	"testing"

	"github.com/intel/infra-onboarding/onboarding-manager/internal/invclient"
	"github.com/intel/infra-onboarding/onboarding-manager/internal/onboardingmgr/onboarding"
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
		t.Run(tt.name, func(_ *testing.T) {
			onboarding.InitOnboarding(tt.args.invClient, tt.args.dkamAddr, tt.args.enableAuth, tt.args.rbac)
		})
	}
}
