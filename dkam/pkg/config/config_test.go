// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/intel/infra-onboarding/dkam/pkg/config"
	dkam_testing "github.com/intel/infra-onboarding/dkam/testing"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	cleanupFunc := dkam_testing.StartTestReleaseService("profile")
	defer cleanupFunc()

	run := m.Run()
	os.Exit(run)
}

func TestRead(t *testing.T) {
	t.Run("Unsupported config type", func(t *testing.T) {
		*config.FlagConfigFilePath = "/tmp/invalid"
		err := config.Read()
		require.Error(t, err)
	})

	t.Run("Invalid path", func(t *testing.T) {
		*config.FlagConfigFilePath = "/tmp/invalid.yaml"
		err := config.Read()
		require.Error(t, err)
	})

	testConfig := config.InfraConfig{
		ENManifestTag: dkam_testing.CorrectTestManifestTag,
		ENProxyHTTP:   "test",
	}
	f, err := os.CreateTemp(os.TempDir(), "infraconfig_*.yaml")
	require.NoError(t, err)
	defer os.RemoveAll(f.Name())

	out, err := yaml.Marshal(&testConfig)
	require.NoError(t, err)
	_, err = f.Write(out)
	require.NoError(t, err)

	t.Run("Success", func(t *testing.T) {
		*config.FlagConfigFilePath = f.Name()

		err = config.Read()
		require.NoError(t, err)

		got := config.GetInfraConfig()
		require.Equal(t, testConfig.ENManifestTag, got.ENManifestTag)
		require.Equal(t, testConfig.ENProxyHTTP, got.ENProxyHTTP)
		require.NotEmpty(t, got.ENManifest)
	})

	t.Run("UpdateConfig", func(t *testing.T) {
		testConfig.ENProxyHTTPS = "new proxy"
		out, err = yaml.Marshal(&testConfig)
		require.NoError(t, err)
		_, err = f.WriteAt(out, 0) // overwrite file
		require.NoError(t, err)

		// give time for config refresh
		time.Sleep(1 * time.Second)

		got := config.GetInfraConfig()
		fmt.Println(got)
		require.Equal(t, testConfig.ENManifestTag, got.ENManifestTag)
		require.Equal(t, testConfig.ENProxyHTTP, got.ENProxyHTTP)
		require.Equal(t, testConfig.ENProxyHTTPS, got.ENProxyHTTPS)
		require.NotEmpty(t, got.ENManifest)
	})
}

func TestDownloadENManifest(t *testing.T) {
	t.Run("InvalidTag", func(t *testing.T) {
		got, err := config.DownloadENManifest("invalidTag")
		require.Error(t, err)
		require.Nil(t, got)
	})

	t.Run("Success", func(t *testing.T) {
		got, err := config.DownloadENManifest(dkam_testing.CorrectTestManifestTag)
		require.NoError(t, err)
		require.NotEmpty(t, got)
	})
}

func TestSetGetInfraConfig(t *testing.T) {
	testInfraConfig := config.InfraConfig{
		ENManifestTag: "test",
	}
	config.SetInfraConfig(testInfraConfig)
	got := config.GetInfraConfig()
	require.Equal(t, testInfraConfig, got)
}
