// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cloudinit_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	osv1 "github.com/intel/infra-core/inventory/v2/pkg/api/os/v1"
	"github.com/intel/infra-onboarding/dkam/pkg/config"
	dkam_testing "github.com/intel/infra-onboarding/dkam/testing"
	"github.com/intel/infra-onboarding/onboarding-manager/pkg/cloudinit"
)

var (
	currentDir  string
	projectRoot string
)

func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("Error getting current directory: %v", err))
	}
	projectRoot = filepath.Dir(filepath.Dir(wd))
	currentDir = wd

	cleanupFunc := dkam_testing.StartTestReleaseService("test-profile")

	run := m.Run()
	cleanupFunc()
	os.Exit(run)
}

//nolint:funlen // it consists of required test cases.
func TestGenerateFromInfraConfig(t *testing.T) {
	dkam_testing.PrepareTestInfraConfig(t)
	dkam_testing.PrepareTestCaCertificateFile(t)
	baseConfig := config.GetInfraConfig()
	baseConfig.ENDebianPackagesRepo = "test.deb"

	type args struct {
		options             cloudinit.CloudInitOptions
		infraConfigOverride func(config.InfraConfig) config.InfraConfig
	}
	tests := []struct {
		name                   string
		args                   args
		expectedOutputFileName string
		wantErr                bool
	}{
		{
			name: "Success_Base_ImmutableOS",
			args: args{
				options: cloudinit.CloudInitOptions{
					Mode:   "dev",
					OsType: osv1.OsType_OS_TYPE_IMMUTABLE,
				},
				infraConfigOverride: func(infraConfig config.InfraConfig) config.InfraConfig {
					newCfg := infraConfig
					newCfg.ClusterURL = "cluster.kind.internal"
					newCfg.ExtraHosts = strings.Split("1.1.1.1 a.test,2.2.2.2 b.test", ",")
					newCfg.FirewallCfgAllow = `
[
    {
        "sourceIp": "",
        "ports": "80,443",
        "ipVer": "",
        "protocol": "tcp"
    }
]`
					return newCfg
				},
			},
			expectedOutputFileName: "expected-installer-01.cfg",
			wantErr:                false,
		},
		{
			name: "Success_Base_MutableOS",
			args: args{
				options: cloudinit.CloudInitOptions{
					Mode:   "dev",
					OsType: osv1.OsType_OS_TYPE_MUTABLE,
				},
				infraConfigOverride: func(infraConfig config.InfraConfig) config.InfraConfig {
					newCfg := infraConfig
					newCfg.ClusterURL = "cluster.kind.internal"
					newCfg.ExtraHosts = strings.Split("1.1.1.1 a.test,2.2.2.2 b.test", ",")
					newCfg.FirewallCfgAllow = `
[
    {
        "sourceIp": "",
        "ports": "80,443",
        "ipVer": "",
        "protocol": "tcp"
    }
]`
					return newCfg
				},
			},
			expectedOutputFileName: "expected-installer-02.cfg",
			wantErr:                false,
		},
		{
			name: "Success_NotKindInternal",
			args: args{
				options: cloudinit.CloudInitOptions{
					Mode:   "dev",
					OsType: osv1.OsType_OS_TYPE_IMMUTABLE,
				},
				// cluster.test by default
			},
			expectedOutputFileName: "expected-installer-03.cfg",
			wantErr:                false,
		},
		{
			name: "Success_ProdMode",
			args: args{
				options: cloudinit.CloudInitOptions{
					Mode:   "prod",
					OsType: osv1.OsType_OS_TYPE_IMMUTABLE,
				},
			},
			expectedOutputFileName: "expected-installer-04.cfg",
			wantErr:                false,
		},
		{
			name: "Success_ProdMode_MutableOS",
			args: args{
				options: cloudinit.CloudInitOptions{
					Mode:   "prod",
					OsType: osv1.OsType_OS_TYPE_MUTABLE,
				},
			},
			expectedOutputFileName: "expected-installer-05.cfg",
			wantErr:                false,
		},
		{
			name: "Success_NoProxies",
			args: args{
				options: cloudinit.CloudInitOptions{
					Mode:   "prod",
					OsType: osv1.OsType_OS_TYPE_MUTABLE,
				},
				infraConfigOverride: func(infraConfig config.InfraConfig) config.InfraConfig {
					newCfg := infraConfig
					newCfg.ENProxyHTTP = ""
					newCfg.ENProxyHTTPS = ""
					newCfg.ENProxyNoProxy = ""
					newCfg.ENProxySocks = ""
					newCfg.ENProxyFTP = ""
					return newCfg
				},
			},
			expectedOutputFileName: "expected-installer-06.cfg",
			wantErr:                false,
		},
		{
			name: "Success_SelectedProxies",
			args: args{
				options: cloudinit.CloudInitOptions{
					Mode:   "prod",
					OsType: osv1.OsType_OS_TYPE_IMMUTABLE,
				},
				infraConfigOverride: func(infraConfig config.InfraConfig) config.InfraConfig {
					newCfg := infraConfig
					newCfg.ENProxyFTP = ""
					newCfg.ENProxySocks = ""
					return newCfg
				},
			},
			expectedOutputFileName: "expected-installer-07.cfg",
			wantErr:                false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.SetInfraConfig(baseConfig)
			if tt.args.infraConfigOverride != nil {
				newCfg := tt.args.infraConfigOverride(config.GetInfraConfig())
				config.SetInfraConfig(newCfg)
			}

			got, err := cloudinit.GenerateFromInfraConfig(tt.args.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateFromInfraConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				fileData, err := os.ReadFile(currentDir + "/testout/" + tt.expectedOutputFileName)
				require.NoError(t, err)

				require.Equal(t, got, string(fileData))
			}
		})
	}
}
