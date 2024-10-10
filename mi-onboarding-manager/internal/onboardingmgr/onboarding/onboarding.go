/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	logging "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/logging"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/policy/rbac"
)

var (
	clientName = "Onboarding"
	zlog       = logging.GetLogger(clientName)
)

//nolint:tagliatelle // Renaming the json keys may effect while unmarshalling/marshaling so, used nolint.
type ResponseData struct {
	To2CompletedOn string `json:"to2CompletedOn"`
	To0Expiry      string `json:"to0Expiry"`
}

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
