// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package testing

import (
	"fmt"
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

	inv_v1 "github.com/intel/infra-core/inventory/v2/pkg/api/inventory/v1"
	"github.com/intel/infra-core/inventory/v2/pkg/logging"
	inv_testing "github.com/intel/infra-core/inventory/v2/pkg/testing"
	"github.com/intel/infra-onboarding/dkam/internal/env"
	"github.com/intel/infra-onboarding/dkam/internal/invclient"
	"github.com/intel/infra-onboarding/dkam/pkg/config"
)

const (
	fileMode = 0o755

	TestManifestRepo       = "test-manifest-repo"
	CorrectTestManifestTag = "correct"
	EmptyTestManifestTag   = "empty"
)

var (
	clientName = inv_testing.ClientType("TestDKAMInventoryClient")
	zlog       = logging.GetLogger("DKAM-Manager-Testing")
	InvClient  *invclient.DKAMInventoryClient
	mu         sync.Mutex
)

func exampleManifest(digest string, fileLen int) string {
	return fmt.Sprintf(`
		{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json",
		"config":{"mediaType":"application/vnd.intel.ensp.en",
		"digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2},
		"layers":[{
			"mediaType":"application/vnd.oci.image.layer.v1.tar",
			"digest":"`+digest+`",
			"size":%d,
			"annotations":{"org.opencontainers.image.title":"`+digest+`"}
		}],
		"annotations":{"org.opencontainers.image.created":"2024-03-26T10:32:25Z"}}`, fileLen)
}

func StartTestReleaseService(testProfileName string) func() {
	config.SetInfraConfig(config.InfraConfig{
		ENManifestRepo: TestManifestRepo,
		ENManifestTag:  CorrectTestManifestTag,
	})
	infraConfig := config.GetInfraConfig()

	expectedFileContent := "GOOD TEST!"
	expectedTestManifest := `
metadata:
  schemaVersion: 0.2.1
  release: 3.0.0-dev
  bma:
  debs:
    - description: Node Agent
      repository: Release Service debian repository
      package: node-agent
      registry: Release Service OCI registry
      ociArtifact: one-intel-edge/edge-node/deb/node-agent
      version: 1.4.4
      publishToRepo: true
    - description: Cluster Agent
      repository: Release Service debian repository
      package: cluster-agent
      registry: Release Service OCI registry
      ociArtifact: one-intel-edge/edge-node/deb/cluster-agent
      version: 1.3.11
      publishToRepo: true
    - description: Hardware Discovery Agent
      repository: Release Service debian repository
      package: hardware-discovery-agent
      registry: Release Service OCI registry
      ociArtifact: one-intel-edge/edge-node/deb/hardware-discovery-agent
      version: 1.4.5
  provisioning:
  files:
    - description: iPXE binary & script
      server: Release Service OCI registry
      path: one-intel-edge/edge-node/file/provisioning-ipxe-efi-app
      version: 1.0.1
    - description: MicroOS image
      server: Release Service OCI registry
      path: one-intel-edge/edge-node/file/provisioning-hook-os
      version: 1.6.0
`

	mux := http.NewServeMux()

	testManifestDigestCorrect := "TEST_MANIFEST_DIGEST_CORRECT"
	testManifestDigestEmpty := "TEST_MANIFEST_DIGEST_EMPTY"
	testProfileManifest := "TEST_PROFILE_MANIFEST"

	mux.HandleFunc("/v2/"+infraConfig.ENManifestRepo+"/manifests/"+CorrectTestManifestTag,
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			// return example manifest
			w.Write([]byte(exampleManifest(testManifestDigestCorrect, len(expectedTestManifest))))
		})
	mux.HandleFunc("/v2/"+infraConfig.ENManifestRepo+"/blobs/"+testManifestDigestCorrect,
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(expectedTestManifest))
		})

	mux.HandleFunc("/v2/"+infraConfig.ENManifestRepo+"/manifests/"+EmptyTestManifestTag,
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			// return example manifest
			w.Write([]byte(exampleManifest(testManifestDigestEmpty, 1)))
		})
	mux.HandleFunc("/v2/"+infraConfig.ENManifestRepo+"/blobs/"+testManifestDigestEmpty,
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			// empty data
		})

	// test handlers for profile script
	mux.HandleFunc("/v2/"+env.ProfileScriptRepo+testProfileName+"/manifests/1.0.2",
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			// return example manifest
			w.Write([]byte(exampleManifest(testProfileManifest, len(expectedFileContent))))
		})
	mux.HandleFunc("/v2/"+env.ProfileScriptRepo+testProfileName+"/blobs/"+testProfileManifest,
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

//nolint:funlen // test helper
func PrepareTestInfraConfig(_ *testing.T) {
	testConfig := config.InfraConfig{
		ENManifestTag:                         "latest-dev",
		InfraURL:                              "infra.test:443",
		ClusterURL:                            "cluster.test:443",
		UpdateURL:                             "update.test:443",
		ReleaseServiceURL:                     "rs.test:443",
		LogsObservabilityURL:                  "logs.test:443",
		MetricsObservabilityURL:               "metrics.test:443",
		KeycloakURL:                           "keycloak.test:443",
		TelemetryURL:                          "telemetry.test:443",
		RegistryURL:                           "registry.test:443",
		FileServerURL:                         "fs.test:443",
		ProvisioningService:                   "provisioning.test:443",
		ProvisioningServerURL:                 "provisioning.test:443",
		TinkServerURL:                         "tink.test:443",
		OnboardingURL:                         "onboarding.test:443",
		OnboardingStreamURL:                   "onboarding-stream.test:443",
		CDN:                                   "cdn.test:443",
		SystemConfigFsInotifyMaxUserInstances: 1,
		SystemConfigVmOverCommitMemory:        1,
		SystemConfigKernelPanicOnOops:         1,
		SystemConfigKernelPanic:               1,
		ENProxyHTTP:                           "http-proxy.test",
		ENProxyHTTPS:                          "https-proxy.test",
		ENProxyFTP:                            "ftp-proxy.test",
		ENProxyNoProxy:                        "no-proxy.test",
		ENProxySocks:                          "socks.test",
		NetIP:                                 "dynamic",
		NTPServers:                            []string{"ntp1.org", "ntp2.org"},
		DNSServers:                            []string{"1.1.1.1"},
		ExtraHosts:                            []string{},
		FirewallReqAllow:                      "",
		FirewallCfgAllow:                      "",
		ENManifest: config.ENManifest{
			Packages: struct {
				Debians []string `yaml:"deb_packages"`
			}{},
			BMA: struct {
				Debs []config.AgentsVersion `yaml:"debs"`
			}{
				Debs: []config.AgentsVersion{
					{
						Package: "node-agent",
						Version: "1.0.0",
					},
					{
						Package: "caddy",
						Version: "1.0.0",
					},
					{
						Package: "hardware-discovery-agent",
						Version: "1.0.0",
					},
					{
						Package: "cluster-agent",
						Version: "1.0.0",
					},
					{
						Package: "platform-observability-agent",
						Version: "1.0.0",
					},
					{
						Package: "platform-telemetry-agent",
						Version: "1.0.0",
					},
					{
						Package: "trtl",
						Version: "1.0.0",
					},
					{
						Package: "inbm-cloudadapter-agent",
						Version: "1.0.0",
					},
					{
						Package: "inbm-dispatcher-agent",
						Version: "1.0.0",
					},
					{
						Package: "inbm-configuration-agent",
						Version: "1.0.0",
					},
					{
						Package: "inbm-telemetry-agent",
						Version: "1.0.0",
					},
					{
						Package: "inbm-diagnostic-agent",
						Version: "1.0.0",
					},
					{
						Package: "mqtt",
						Version: "1.0.0",
					},
					{
						Package: "tpm-provision",
						Version: "1.0.0",
					},
					{
						Package: "inbc-program",
						Version: "1.0.0",
					},
					{
						Package: "platform-update-agent",
						Version: "1.0.0",
					},
				},
			},
			Metadata: struct {
				//nolint:tagliatelle // test file
				DebianRepositories []struct {
					Name         string `yaml:"name"`
					URL          string `yaml:"url"`
					Architecture string `yaml:"architecture"`
					Key          string `yaml:"key"`
					Section      string `yaml:"section"`
					Distribution string `yaml:"distribution"`
					Root         string `yaml:"root"`
					ThirdParty   bool   `yaml:"thirdParty"`
					AuthType     string `yaml:"authType"`
				} `yaml:"debianRepositories"`
			}{
				//nolint:tagliatelle // test file
				DebianRepositories: []struct {
					Name         string `yaml:"name"`
					URL          string `yaml:"url"`
					Architecture string `yaml:"architecture"`
					Key          string `yaml:"key"`
					Section      string `yaml:"section"`
					Distribution string `yaml:"distribution"`
					Root         string `yaml:"root"`
					ThirdParty   bool   `yaml:"thirdParty"`
					AuthType     string `yaml:"authType"`
				}{
					{
						Distribution: "test-distro",
					},
				},
			},
		},
	}
	config.SetInfraConfig(testConfig)
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
