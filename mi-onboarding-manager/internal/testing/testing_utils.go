// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

package testing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/inventory/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/logging"
	inv_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/status"
	inv_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/testing"
)

var (
	clientName inv_testing.ClientType = "TestOnboardingInventoryClient"
	zlog                              = logging.GetLogger("Onboarding-Manager-Testing")
	InvClient  *invclient.OnboardingInventoryClient
)

// CreateInventoryOnboardingClientForTesting is an helper function to create a new client.
func CreateInventoryOnboardingClientForTesting() {
	resourceKinds := []inv_v1.ResourceKind{
		inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE,
		inv_v1.ResourceKind_RESOURCE_KIND_HOST,
		inv_v1.ResourceKind_RESOURCE_KIND_OS,
	}
	err := inv_testing.CreateClient(clientName, inv_v1.ClientKind_CLIENT_KIND_RESOURCE_MANAGER, resourceKinds, "")
	if err != nil {
		zlog.Fatal().Err(err).Msg("Cannot create Inventory OnboardingRM client")
	}

	InvClient, err = invclient.NewOnboardingInventoryClient(inv_testing.TestClients[clientName].GetTenantAwareInventoryClient(),
		inv_testing.TestClientsEvents[clientName])
	if err != nil {
		zlog.Fatal().Err(err).Msg("Cannot create Inventory OnboardingRM client")
	}
}

func DeleteInventoryOnboardingClientForTesting() {
	InvClient.Close()
	time.Sleep(1 * time.Second)
	delete(inv_testing.TestClients, clientName)
	delete(inv_testing.TestClientsEvents, clientName)
}

func AssertHost(
	tb testing.TB,
	tenantID string,
	resID string,
	expectedDesiredState computev1.HostState,
	expectedCurrentState computev1.HostState,
	expectedHostStatus inv_status.ResourceStatus,
) {
	tb.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	gresp, err := inv_testing.TestClients[inv_testing.APIClient].GetTenantAwareInventoryClient().Get(ctx, tenantID, resID)
	require.NoError(tb, err)
	host := gresp.GetResource().GetHost()
	assert.Equal(tb, expectedDesiredState, host.GetDesiredState())
	assert.Equal(tb, expectedCurrentState, host.GetCurrentState())
	assert.Equal(tb, expectedHostStatus.Status, host.GetHostStatus())
	assert.Equal(tb, expectedHostStatus.StatusIndicator, host.GetHostStatusIndicator())
}

func AssertHostOnboardingStatus(tb testing.TB, resID string, expectedOnboardingStatus inv_status.ResourceStatus) {
	tb.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	gresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, resID)
	require.NoError(tb, err)
	host := gresp.GetResource().GetHost()
	assert.Equal(tb, expectedOnboardingStatus.Status, host.GetOnboardingStatus())
	assert.Equal(tb, expectedOnboardingStatus.StatusIndicator, host.GetOnboardingStatusIndicator())
}

func AssertInstance(
	tb testing.TB,
	tenantID string,
	resID string,
	expectedDesiredState computev1.InstanceState,
	expectedCurrentState computev1.InstanceState,
	expectedStatus inv_status.ResourceStatus,
) {
	tb.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	gresp, err := inv_testing.TestClients[inv_testing.APIClient].GetTenantAwareInventoryClient().Get(ctx, tenantID, resID)
	require.NoError(tb, err)

	instance := gresp.GetResource().GetInstance()

	assert.Equal(tb, expectedDesiredState, instance.GetDesiredState())
	assert.Equal(tb, expectedCurrentState, instance.GetCurrentState())
	assert.Equal(tb, expectedStatus.Status, instance.GetInstanceStatus())
	assert.Equal(tb, expectedStatus.StatusIndicator, instance.GetInstanceStatusIndicator())
}
