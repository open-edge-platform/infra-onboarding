// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package testing

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-onboarding/dkam/internal/env"
	"github.com/open-edge-platform/infra-onboarding/dkam/internal/flag"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/config"
)

const (
	TestManifestRepo       = "test-manifest-repo"
	CorrectTestManifestTag = "correct"
	EmptyTestManifestTag   = "empty"
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

func EnableLegacyModeForTesting(t *testing.T) {
	t.Helper()
	*flag.LegacyMode = true
	t.Cleanup(func() {
		*flag.LegacyMode = false // restore default mode
	})
}

func StartTestReleaseService(testProfileName string) func() {
	config.SetInfraConfig(config.InfraConfig{
		ENManifestRepo:     TestManifestRepo,
		ENAgentManifestTag: CorrectTestManifestTag,
	})
	infraConfig := config.GetInfraConfig()

	expectedFileContent := "GOOD TEST!"
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
  - name: caddy
    version: 2.7.6
  - name: inbc-program
    version: 4.2.8.2-1
  - name: inbm-configuration-agent
    version: 4.2.8.2-1
  - name: inbm-cloudadapter-agent
    version: 4.2.8.2-1
  - name: inbm-diagnostic-agent
    version: 4.2.8.2-1
  - name: inbm-dispatcher-agent
    version: 4.2.8.2-1
  - name: inbm-telemetry-agent
    version: 4.2.8.2-1
  - name: mqtt
    version: 4.2.8.2-1
  - name: tpm-provision
    version: 4.2.8.2-1
  - name: trtl
    version: 4.2.8.2-1
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
		ENAgentManifestTag:                    "latest-dev",
		InfraURL:                              "infra.test:443",
		ClusterURL:                            "cluster.test:443",
		UpdateURL:                             "update.test:443",
		ReleaseServiceURL:                     "rs.test:443",
		LogsObservabilityURL:                  "logs.test:443",
		MetricsObservabilityURL:               "metrics.test:443",
		KeycloakURL:                           "keycloak.test:443",
		TelemetryURL:                          "telemetry.test:443",
		RegistryURL:                           "registry.test",
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
					Name:    "trtl",
					Version: "1.0.0",
				},
				{
					Name:    "inbm-cloudadapter-agent",
					Version: "1.0.0",
				},
				{
					Name:    "inbm-dispatcher-agent",
					Version: "1.0.0",
				},
				{
					Name:    "inbm-configuration-agent",
					Version: "1.0.0",
				},
				{
					Name:    "inbm-telemetry-agent",
					Version: "1.0.0",
				},
				{
					Name:    "inbm-diagnostic-agent",
					Version: "1.0.0",
				},
				{
					Name:    "mqtt",
					Version: "1.0.0",
				},
				{
					Name:    "tpm-provision",
					Version: "1.0.0",
				},
				{
					Name:    "inbc-program",
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
