// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package testing provides test utilities and helpers for DKAM testing.
package testing

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/config"
)

const (
	// TestManifestRepo is the test manifest repository name.
	TestManifestRepo = "test-manifest-repo"
	// CorrectTestManifestTag is the tag for correct test manifests.
	CorrectTestManifestTag = "correct"
	// EmptyTestManifestTag is the tag for empty test manifests.
	EmptyTestManifestTag = "empty"
	// TestMicroOSfileName is the filename for test micro OS files.
	TestMicroOSfileName = "test-uos-file"
)

func exampleManifest(digest string, fileLen int) string {
	return fmt.Sprintf(`
		{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json",
		"config":{"mediaType":"application/vnd.intel.hookos.file",
		"digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2},
		"layers":[{
			"mediaType":"application/vnd.oci.image.layer.v1.tar",
			"digest":"`+digest+`",
			"size":%d,
			"annotations":{"org.opencontainers.image.title":"`+digest+`"}
		}],
		"annotations":{"org.opencontainers.image.created":"2025-03-18T16:44:00Z"}}`, fileLen)
}

// StartTestReleaseService starts a test HTTP server that simulates the release service.
//
//nolint:funlen // Test setup function with multiple endpoints
func StartTestReleaseService() func() {
	config.SetInfraConfig(config.InfraConfig{
		ENManifestRepo:     TestManifestRepo,
		ENAgentManifestTag: CorrectTestManifestTag,
	})
	infraConfig := config.GetInfraConfig()

	expectedTestManifest := `
repository:
  codename: 3.0
  component: main
packages:
  - name: cluster-agent
    version: 1.5.8
  - name: hardware-discovery-agent
    version: 1.5.3
  - name: node-agent
    version: 1.5.8
  - name: platform-observability-agent
    version: 1.7.2
  - name: platform-telemetry-agent
    version: 1.2.4
  - name: platform-update-agent
    version: 1.3.4
  - name: platform-manageability-agent
    version: 1.2.4
  - name: caddy
    version: 2.7.6
  - name: in-band-manageability
    version: 1.0.2
`
	mux := http.NewServeMux()

	testManifestDigestCorrect := "TEST_MANIFEST_DIGEST_CORRECT"
	testManifestDigestEmpty := "TEST_MANIFEST_DIGEST_EMPTY"

	mux.HandleFunc("/v2/"+infraConfig.ENManifestRepo+"/manifests/"+CorrectTestManifestTag,
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			// return example manifest
			_, _ = w.Write([]byte(exampleManifest(testManifestDigestCorrect, len(expectedTestManifest))))
		})
	mux.HandleFunc("/v2/"+infraConfig.ENManifestRepo+"/blobs/"+testManifestDigestCorrect,
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(expectedTestManifest))
		})

	mux.HandleFunc("/v2/"+infraConfig.ENManifestRepo+"/manifests/"+EmptyTestManifestTag,
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			// return example manifest
			_, _ = w.Write([]byte(exampleManifest(testManifestDigestEmpty, 1)))
		})
	mux.HandleFunc("/v2/"+infraConfig.ENManifestRepo+"/blobs/"+testManifestDigestEmpty,
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			// empty data
		})
	mux.HandleFunc("/"+TestMicroOSfileName,
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			// return test data
			_, _ = w.Write([]byte("testdata"))
		})

	svr := httptest.NewServer(mux)
	config.SetInfraConfig(config.InfraConfig{
		ENManifestRepo:     TestManifestRepo,
		ENAgentManifestTag: CorrectTestManifestTag,
		CDN:                svr.URL,
		EMBImageURL:        TestMicroOSfileName,
	})

	testRegistryEndpoint, _ := strings.CutPrefix(svr.URL, "http://")

	_ = os.Setenv("RSPROXY_ADDRESS", testRegistryEndpoint+"/")

	return func() {
		_ = os.Unsetenv("RSPROXY_ADDRESS")
		svr.Close()
	}
}

// PrepareTestCaCertificateFile creates a temporary CA certificate file for testing.
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
		_ = os.RemoveAll(tmpDir)
		config.OrchCACertificateFile = originalCaCertificatePath
	})
}

// PrepareTestInfraConfig sets up test infrastructure configuration.
//
//nolint:funlen // Test setup function with comprehensive config
func PrepareTestInfraConfig(_ *testing.T) {
	testConfig := config.InfraConfig{
		ENAgentManifestTag:                    "latest-dev",
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
		ManageabilityURL:                      "manageability.test:443",
		RPSAddress:                            "rps.test",
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
		FirewallReqAllow:                      "",
		FirewallCfgAllow:                      "",
		ENManifest: config.ENManifest{
			Repository: config.Repository{
				Codename:  "1.0",
				Component: "main",
			},
			Packages: []config.AgentsVersion{
				{
					Name:    "node-agent",
					Version: "1.0.0",
				},
				{
					Name:    "caddy",
					Version: "1.0.0",
				},
				{
					Name:    "hardware-discovery-agent",
					Version: "1.0.0",
				},
				{
					Name:    "cluster-agent",
					Version: "1.0.0",
				},
				{
					Name:    "platform-observability-agent",
					Version: "1.0.0",
				},
				{
					Name:    "platform-telemetry-agent",
					Version: "1.0.0",
				},
				{
					Name:    "platform-manageability-agent",
					Version: "1.0.0",
				},
				{
					Name:    "in-band-manageability",
					Version: "1.0.0",
				},
				{
					Name:    "platform-update-agent",
					Version: "1.0.0",
				},
			},
		},
	}
	config.SetInfraConfig(testConfig)
}
