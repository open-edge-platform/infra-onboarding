// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

package testing

import (
	"time"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/internal/invclient"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/inventory/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/logging"
	inv_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/testing"
)

var (
	clientName = inv_testing.ClientType("TestDKAMInventoryClient")
	zlog       = logging.GetLogger("DKAM-Manager-Testing")
	InvClient  *invclient.DKAMInventoryClient
)

// CreateInventoryDKAMClientForTesting is an helper function to create a new client.
func CreateInventoryDKAMClientForTesting() {
	resourceKinds := []inv_v1.ResourceKind{
		inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE,
		inv_v1.ResourceKind_RESOURCE_KIND_HOST,
		inv_v1.ResourceKind_RESOURCE_KIND_OS,
	}
	err := inv_testing.CreateClient(clientName, inv_v1.ClientKind_CLIENT_KIND_RESOURCE_MANAGER, resourceKinds, "")
	if err != nil {
		zlog.Fatal().Err(err).Msg("Cannot create Inventory DKAMRM client")
	}

	InvClient, err = invclient.NewDKAMInventoryClient(
		inv_testing.TestClients[clientName].GetTenantAwareInventoryClient(),
		inv_testing.TestClientsEvents[clientName])
	if err != nil {
		zlog.Fatal().Err(err).Msg("Cannot create Inventory DKAMRM client")
	}
}

func DeleteInventoryDKAMClientForTesting() {
	InvClient.Close()
	time.Sleep(1 * time.Second)
	delete(inv_testing.TestClients, clientName)
	delete(inv_testing.TestClientsEvents, clientName)
}

// func AssertHost(
// 	tb testing.TB,
// 	resID string,
// 	expectedDesiredState computev1.HostState,
// 	expectedCurrentState computev1.HostState,
// 	expectedLegacyStatus computev1.HostStatus,
// 	expectedProviderStatusDetail string,
// 	expectedHostStatus inv_status.ResourceStatus,
// ) {
// 	tb.Helper()

// 	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
// 	defer cancel()

// 	gresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, resID)
// 	require.NoError(tb, err)
// 	host := gresp.GetResource().GetHost()
// 	assert.Equal(tb, expectedDesiredState, host.GetDesiredState())
// 	assert.Equal(tb, expectedCurrentState, host.GetCurrentState())
// 	//nolint:staticcheck // legacy host status will be deprecated post-24.03.
// 	assert.Equal(tb, expectedLegacyStatus, host.GetLegacyHostStatus())
// 	//nolint:staticcheck // this field will be deprecated soon
// 	assert.Equal(tb, expectedProviderStatusDetail, host.GetProviderStatusDetail())
// 	assert.Equal(tb, expectedHostStatus.Status, host.GetHostStatus())
// 	assert.Equal(tb, expectedHostStatus.StatusIndicator, host.GetHostStatusIndicator())
// }

// func AssertHostDKAMStatus(tb testing.TB, resID string, expectedDKAMStatus inv_status.ResourceStatus) {
// 	tb.Helper()

// 	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
// 	defer cancel()

// 	gresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, resID)
// 	require.NoError(tb, err)
// 	host := gresp.GetResource().GetHost()
// 	assert.Equal(tb, expectedDKAMStatus.Status, host.GetDKAMStatus())
// 	assert.Equal(tb, expectedDKAMStatus.StatusIndicator, host.GetDKAMStatusIndicator())
// }

// func AssertInstance(
// 	tb testing.TB,
// 	resID string,
// 	expectedDesiredState computev1.InstanceState,
// 	expectedCurrentState computev1.InstanceState,
// 	expectedStatus computev1.InstanceStatus,
// ) {
// 	tb.Helper()

// 	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
// 	defer cancel()

// 	gresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, resID)
// 	require.NoError(tb, err)

// 	instance := gresp.GetResource().GetInstance()

// 	assert.Equal(tb, expectedDesiredState, instance.GetDesiredState())
// 	assert.Equal(tb, expectedCurrentState, instance.GetCurrentState())
// 	//nolint:staticcheck // legacy host status will be deprecated post-24.03.
// 	assert.Equal(tb, expectedStatus, instance.GetStatus())
// }
