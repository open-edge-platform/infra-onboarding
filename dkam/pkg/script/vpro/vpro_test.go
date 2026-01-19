// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package vpro_test

import (
	"testing"

	osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/config"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/script/vpro"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCurateVProInstaller(t *testing.T) {
	t.Run("Success_Ubuntu", func(t *testing.T) {
		// Create a mock infra configuration
		infraConfig := config.InfraConfig{
			ClusterURL:              "https://cluster.example.com",
			InfraURL:                "https://infra.example.com:9092",
			UpdateURL:               "https://update.example.com:8080",
			KeycloakURL:             "https://keycloak.example.com:8443",
			ReleaseServiceURL:       "https://release.example.com:9090",
			FileServerURL:           "files.example.com:60444",
			RegistryURL:             "registry.example.com:5000",
			LogsObservabilityURL:    "obs-logs.example.com:4317",
			MetricsObservabilityURL: "obs-metrics.example.com:4318",
			TelemetryURL:            "telemetry.example.com:8081",
			ManageabilityURL:        "manage.example.com:50051",
			RPSAddress:              "rps.example.com:8084",

			ENDebianPackagesRepo: "files-edge-orch/debian",
			ENFilesRsRoot:        "files-edge-orch",
			RSType:               "auth",

			NTPServers: []string{"ntp1.example.com", "ntp2.example.com"},

			ENProxyHTTP:    "http://proxy.example.com:3128",
			ENProxyHTTPS:   "https://proxy.example.com:3128",
			ENProxyNoProxy: "localhost,127.0.0.1",
			ENProxyFTP:     "ftp://proxy.example.com:3128",
			ENProxySocks:   "socks5://proxy.example.com:1080",

			SystemConfigVmOverCommitMemory:        1,
			SystemConfigKernelPanic:               10,
			SystemConfigKernelPanicOnOops:         1,
			SystemConfigFsInotifyMaxUserInstances: 8192,

			DisableCOProfile:   false,
			DisableO11YProfile: false,
		}

		// Curate the installer for Ubuntu (mutable OS)
		result, err := vpro.CurateVProInstaller(infraConfig, osv1.OsType_OS_TYPE_MUTABLE)

		// Assertions
		require.NoError(t, err)
		assert.NotEmpty(t, result)

		// Verify some template variables were replaced
		assert.Contains(t, result, "https://cluster.example.com")
		assert.Contains(t, result, "https://infra.example.com:9092")
		assert.Contains(t, result, "KERNEL_CONFIG_OVER_COMMIT_MEMORY=1")
		assert.Contains(t, result, "KERNEL_CONFIG_KERNEL_PANIC=10")
		assert.NotContains(t, result, "{{ .ORCH_CLUSTER }}")
		assert.NotContains(t, result, "{{ .ORCH_INFRA }}")
	})

	t.Run("Success_EMT", func(t *testing.T) {
		infraConfig := config.InfraConfig{
			InfraURL:      "https://infra.example.com:9092",
			UpdateURL:     "https://update.example.com:8080",
			FileServerURL: "files.example.com:60444",
			ENFilesRsRoot: "files-edge-orch",
			RSType:        "noauth",
		}

		// Curate for EMT (immutable OS)
		result, err := vpro.CurateVProInstaller(infraConfig, osv1.OsType_OS_TYPE_IMMUTABLE)

		require.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, `if [ "noauth" == "auth" ];`)
	})
}
