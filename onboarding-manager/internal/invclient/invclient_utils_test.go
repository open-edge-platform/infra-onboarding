// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package invclient_test

import (
	"testing"

	inv_v1 "github.com/intel/infra-core/inventory/v2/pkg/api/inventory/v1"
	"github.com/intel/infra-core/inventory/v2/pkg/logging"
	inv_testing "github.com/intel/infra-core/inventory/v2/pkg/testing"
	"github.com/intel/infra-onboarding/onboarding-manager/internal/invclient"
)

const (
	testClientName = "TestOnboardingInventoryClient"
	loggerName     = "TestLogger"
)

var (
	zlogTest             = logging.GetLogger(loggerName)
	OnboardingTestClient *invclient.OnboardingInventoryClient
)

func CreateOnboardingClientForTesting(tb testing.TB) {
	tb.Helper()
	var err error
	resourceKinds := []inv_v1.ResourceKind{
		inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE,
		inv_v1.ResourceKind_RESOURCE_KIND_HOST,
		inv_v1.ResourceKind_RESOURCE_KIND_OS,
	}
	err = inv_testing.CreateClient(testClientName, inv_v1.ClientKind_CLIENT_KIND_RESOURCE_MANAGER, resourceKinds, "")
	if err != nil {
		zlogTest.Fatal().Err(err).Msg("Cannot create onboarding invclient client")
	}

	OnboardingTestClient, err = invclient.NewOnboardingInventoryClient(
		inv_testing.TestClients[testClientName].GetTenantAwareInventoryClient(), inv_testing.TestClientsEvents[testClientName])
	if err != nil {
		zlogTest.Fatal().Err(err).Msg("Cannot create onboarding invclient client")
	}
	tb.Cleanup(func() {
		OnboardingTestClient.Close()
		delete(inv_testing.TestClients, testClientName)
		delete(inv_testing.TestClientsEvents, testClientName)
	})
}
