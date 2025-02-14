/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	logging "github.com/intel/infra-core/inventory/v2/pkg/logging"
	"github.com/intel/infra-core/inventory/v2/pkg/policy/rbac"
	"github.com/intel/infra-onboarding/onboarding-manager/internal/invclient"
)

var (
	clientName = "Onboarding"
	zlog       = logging.GetLogger(clientName)
)

func InitOnboarding(invClient *invclient.OnboardingInventoryClient, _ string, enableAuth bool, rbacRules string) {
	if invClient == nil {
		zlog.Debug().Msgf("Warning: invClient is nil")
		return
	}

	var err error
	if enableAuth {
		zlog.Info().Msgf("Authentication is enabled, starting RBAC server for Onboarding manager")
		// start OPA server with policies
		_, err = rbac.New(rbacRules)
		if err != nil {
			zlog.Fatal().Msg("Failed to start RBAC OPA server")
		}
	}
}
