// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package testing

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	computev1 "github.com/intel/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/intel/infra-core/inventory/v2/pkg/api/inventory/v1"
	"github.com/intel/infra-core/inventory/v2/pkg/client"
	"github.com/intel/infra-core/inventory/v2/pkg/logging"
	inv_status "github.com/intel/infra-core/inventory/v2/pkg/status"
	inv_testing "github.com/intel/infra-core/inventory/v2/pkg/testing"
	"github.com/intel/infra-onboarding/onboarding-manager/internal/invclient"
)

var (
	clientName inv_testing.ClientType = "TestOnboardingInventoryClient"
	zlog                              = logging.GetLogger("Onboarding-Manager-Testing")
	InvClient  *invclient.OnboardingInventoryClient
	mu         sync.Mutex
)

// CreateInventoryOnboardingClientForTesting is an helper function to create a new client.
func CreateInventoryOnboardingClientForTesting() {
	mu.Lock()
	defer mu.Unlock()
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
		inv_testing.TestClientsEvents[clientName], make(chan *client.ResourceTenantIDCarrier))
	if err != nil {
		zlog.Fatal().Err(err).Msg("Cannot create Inventory OnboardingRM client")
	}
}

func DeleteInventoryOnboardingClientForTesting() {
	mu.Lock()
	defer mu.Unlock()
	InvClient.Close()
	time.Sleep(1 * time.Second)
	delete(inv_testing.TestClients, clientName)
	delete(inv_testing.TestClientsEvents, clientName)
}

//nolint:dupl // This is for AssertHost.
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

//nolint:dupl // This is for AssertHost.
func AssertInstance(
	tb testing.TB,
	tenantID string,
	resID string,
	expectedDesiredState computev1.InstanceState,
	expectedCurrentState computev1.InstanceState,
	expectedProvisioningStatus inv_status.ResourceStatus,
) {
	tb.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	gresp, err := inv_testing.TestClients[inv_testing.APIClient].GetTenantAwareInventoryClient().Get(ctx, tenantID, resID)
	require.NoError(tb, err)

	instance := gresp.GetResource().GetInstance()

	assert.Equal(tb, expectedDesiredState, instance.GetDesiredState())
	assert.Equal(tb, expectedCurrentState, instance.GetCurrentState())
	assert.Equal(tb, expectedProvisioningStatus.Status, instance.GetProvisioningStatus())
	assert.Equal(tb, expectedProvisioningStatus.StatusIndicator, instance.GetProvisioningStatusIndicator())
}
