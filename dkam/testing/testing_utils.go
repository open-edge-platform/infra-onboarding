// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package testing

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/api/inventory/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/logging"
	inv_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/testing"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/internal/invclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/pkg/config"
)

const (
	testDigest      = "TEST_DIGEST"
	testFile        = "TEST_FILE"
	exampleManifest = `
		{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json",
		"config":{"mediaType":"application/vnd.intel.ensp.en",
		"digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2},
		"layers":[{
			"mediaType":"application/vnd.oci.image.layer.v1.tar",
			"digest":"` + testDigest + `",
			"size":10,
			"annotations":{"org.opencontainers.image.title":"` + testFile + `"}
		}],
		"annotations":{"org.opencontainers.image.created":"2024-03-26T10:32:25Z"}}`
	fileMode = 0o755
)

var (
	clientName = inv_testing.ClientType("TestDKAMInventoryClient")
	zlog       = logging.GetLogger("DKAM-Manager-Testing")
	InvClient  *invclient.DKAMInventoryClient
	mu         sync.Mutex
)

func StartTestReleaseService(testProfileName string) func() {
	expectedFileContent := "GOOD TEST!"
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/"+config.ProfileScriptRepo+testProfileName+"/manifests/1.0.2",
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			// return example manifest
			w.Write([]byte(exampleManifest))
		})

	mux.HandleFunc("/v2/"+config.ProfileScriptRepo+testProfileName+"/blobs/"+testDigest,
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(expectedFileContent))
		})
	svr := httptest.NewServer(mux)

	testRegistryEndpoint, _ := strings.CutPrefix(svr.URL, "http://")

	os.Setenv("RSPROXY_ADDRESS", testRegistryEndpoint+"/")

	return func() {
		os.Unsetenv("RSPROXY_ADDRESS")
		svr.Close()
	}
}

func PrepareTestCaCertificateFile(t *testing.T) {
	t.Helper()
	tmpDir, err := os.MkdirTemp(os.TempDir(), "test_cert")
	require.NoError(t, err)
	tmpFile, err := os.CreateTemp(tmpDir, "ca_certificate")
	require.NoError(t, err)
	_, err = tmpFile.WriteString("TEST CA CONTENT")
	require.NoError(t, err)
	defer tmpFile.Close()

	originalCaCertificatePath := config.OrchCACertificateFile
	config.OrchCACertificateFile = tmpFile.Name()

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
		config.OrchCACertificateFile = originalCaCertificatePath
	})
}

func PrepareTestReleaseFile(t *testing.T, projectRoot string) {
	t.Helper()
	err := CopyFile(
		filepath.Join(projectRoot, "test", "testdata", "example-manifest-internal-rs.yaml"),
		filepath.Join(config.DownloadPath, "tmp", config.ReleaseVersion+".yaml"))
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(filepath.Join(config.DownloadPath, "tmp"))
	})
}

func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	if mkdirErr := os.MkdirAll(filepath.Dir(dst), fileMode); mkdirErr != nil {
		return mkdirErr
	}
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	return nil
}

// CreateInventoryDKAMClientForTesting is an helper function to create a new client.
func CreateInventoryDKAMClientForTesting() {
	mu.Lock()
	defer mu.Unlock()
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
