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

	osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/config"
	dkam_testing "github.com/open-edge-platform/infra-onboarding/dkam/testing"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/cloudinit"
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
	baseConfig.ENFilesRsRoot = "test"
	baseConfig.DNSServers = []string{"1.1.1.1", "2.2.2.2"}

	const testHostname = "test-hostname"
	const testTenantID = "test-tenantid"
	const testClientID = "test-client-id"
	const testClientSecret = "test-client-secret"
	const testHostMAC = "aa:bb:cc:dd:ee:ff"
	const clusterURL = "cluster.kind.internal"

	type args struct {
		template            string
		options             []cloudinit.Option
		infraConfigOverride func(config.InfraConfig) config.InfraConfig
	}
	tests := []struct {
		name                   string
		args                   args
		expectedOutputFileName string
		wantErr                bool
	}{
		{
			name: "Success_StandaloneEN",
			args: args{
				options: []cloudinit.Option{
					cloudinit.WithOSType(osv1.OsType_OS_TYPE_IMMUTABLE),
					cloudinit.WithHostname(testHostname),
					cloudinit.WithHostMACAddress(testHostMAC),
					cloudinit.WithRunAsStandalone(),
				},
			},
			expectedOutputFileName: "expected-installer-13.cfg",
			wantErr:                false,
		},
		{
			name: "Success_StandaloneEN_WithLocalAccount",
			args: args{
				options: []cloudinit.Option{
					cloudinit.WithOSType(osv1.OsType_OS_TYPE_IMMUTABLE),
					cloudinit.WithHostname(testHostname),
					cloudinit.WithHostMACAddress(testHostMAC),
					cloudinit.WithRunAsStandalone(),
					cloudinit.WithLocalAccount("user", "sshkey"),
				},
			},
			expectedOutputFileName: "expected-installer-14.cfg",
			wantErr:                false,
		},
		{
			name: "Success_Base_ImmutableOS",
			args: args{
				options: []cloudinit.Option{
					cloudinit.WithDevMode("user", "pass"),
					cloudinit.WithOSType(osv1.OsType_OS_TYPE_IMMUTABLE),
					cloudinit.WithHostname(testHostname),
					cloudinit.WithTenantID(testTenantID),
					cloudinit.WithClientCredentials(testClientID, testClientSecret),
					cloudinit.WithHostMACAddress(testHostMAC),
				},
				infraConfigOverride: func(infraConfig config.InfraConfig) config.InfraConfig {
					newCfg := infraConfig
					newCfg.ClusterURL = clusterURL
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
				options: []cloudinit.Option{
					cloudinit.WithDevMode("user", "pass"),
					cloudinit.WithOSType(osv1.OsType_OS_TYPE_MUTABLE),
					cloudinit.WithHostname(testHostname),
					cloudinit.WithTenantID(testTenantID),
					cloudinit.WithClientCredentials(testClientID, testClientSecret),
					cloudinit.WithHostMACAddress(testHostMAC),
				},
				infraConfigOverride: func(infraConfig config.InfraConfig) config.InfraConfig {
					newCfg := infraConfig
					newCfg.ClusterURL = clusterURL
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
				options: []cloudinit.Option{
					cloudinit.WithDevMode("user", "pass"),
					cloudinit.WithOSType(osv1.OsType_OS_TYPE_IMMUTABLE),
					cloudinit.WithHostname(testHostname),
					cloudinit.WithTenantID(testTenantID),
					cloudinit.WithClientCredentials(testClientID, testClientSecret),
					cloudinit.WithHostMACAddress(testHostMAC),
				},
				// cluster.test by default
			},
			expectedOutputFileName: "expected-installer-03.cfg",
			wantErr:                false,
		},
		{
			name: "Success_ProdMode",
			args: args{
				options: []cloudinit.Option{
					cloudinit.WithOSType(osv1.OsType_OS_TYPE_IMMUTABLE),
					cloudinit.WithHostname(testHostname),
					cloudinit.WithTenantID(testTenantID),
					cloudinit.WithClientCredentials(testClientID, testClientSecret),
					cloudinit.WithHostMACAddress(testHostMAC),
				},
			},
			expectedOutputFileName: "expected-installer-04.cfg",
			wantErr:                false,
		},
		{
			name: "Success_ProdMode_MutableOS",
			args: args{
				options: []cloudinit.Option{
					cloudinit.WithOSType(osv1.OsType_OS_TYPE_MUTABLE),
					cloudinit.WithHostname(testHostname),
					cloudinit.WithTenantID(testTenantID),
					cloudinit.WithClientCredentials(testClientID, testClientSecret),
					cloudinit.WithHostMACAddress(testHostMAC),
				},
			},
			expectedOutputFileName: "expected-installer-05.cfg",
			wantErr:                false,
		},
		{
			name: "Success_MutableOS_WithStaticIP",
			args: args{
				options: []cloudinit.Option{
					cloudinit.WithOSType(osv1.OsType_OS_TYPE_MUTABLE),
					cloudinit.WithHostname(testHostname),
					cloudinit.WithTenantID(testTenantID),
					cloudinit.WithHostMACAddress(testHostMAC),
					cloudinit.WithClientCredentials(testClientID, testClientSecret),
					cloudinit.WithPreserveIP("10.10.0.10", []string{"1.1.1.1", "2.2.2.2"}),
				},
			},
			expectedOutputFileName: "expected-installer-09.cfg",
			wantErr:                false,
		},
		{
			name: "Success_ImutableOS_WithStaticIP",
			args: args{
				options: []cloudinit.Option{
					cloudinit.WithOSType(osv1.OsType_OS_TYPE_IMMUTABLE),
					cloudinit.WithHostname(testHostname),
					cloudinit.WithTenantID(testTenantID),
					cloudinit.WithHostMACAddress(testHostMAC),
					cloudinit.WithClientCredentials(testClientID, testClientSecret),
					cloudinit.WithPreserveIP("10.10.0.10", []string{"1.1.1.1", "2.2.2.2"}),
				},
			},
			expectedOutputFileName: "expected-installer-10.cfg",
			wantErr:                false,
		},
		{
			name: "Success_NoProxies",
			args: args{
				options: []cloudinit.Option{
					cloudinit.WithOSType(osv1.OsType_OS_TYPE_MUTABLE),
					cloudinit.WithHostname(testHostname),
					cloudinit.WithTenantID(testTenantID),
					cloudinit.WithClientCredentials(testClientID, testClientSecret),
					cloudinit.WithHostMACAddress(testHostMAC),
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
				options: []cloudinit.Option{
					cloudinit.WithOSType(osv1.OsType_OS_TYPE_IMMUTABLE),
					cloudinit.WithHostname(testHostname),
					cloudinit.WithTenantID(testTenantID),
					cloudinit.WithClientCredentials(testClientID, testClientSecret),
					cloudinit.WithHostMACAddress(testHostMAC),
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
		{
			name: "Success_NoDNSServers",
			args: args{
				options: []cloudinit.Option{
					cloudinit.WithOSType(osv1.OsType_OS_TYPE_MUTABLE),
					cloudinit.WithHostname(testHostname),
					cloudinit.WithTenantID(testTenantID),
					cloudinit.WithClientCredentials(testClientID, testClientSecret),
					cloudinit.WithHostMACAddress(testHostMAC),
				},
				infraConfigOverride: func(infraConfig config.InfraConfig) config.InfraConfig {
					newCfg := infraConfig
					newCfg.DNSServers = []string{}
					return newCfg
				},
			},
			expectedOutputFileName: "expected-installer-08.cfg",
			wantErr:                false,
		},
		{
			name: "Failed_MissingTenantID",
			args: args{
				options: []cloudinit.Option{
					cloudinit.WithOSType(osv1.OsType_OS_TYPE_IMMUTABLE),
					cloudinit.WithHostname(testHostname),
					cloudinit.WithClientCredentials(testClientID, testClientSecret),
					cloudinit.WithHostMACAddress(testHostMAC),
				},
			},
			wantErr: true,
		},
		{
			name: "Failed_MissingHostname",
			args: args{
				options: []cloudinit.Option{
					cloudinit.WithOSType(osv1.OsType_OS_TYPE_IMMUTABLE),
					cloudinit.WithTenantID(testTenantID),
					cloudinit.WithClientCredentials(testClientID, testClientSecret),
					cloudinit.WithHostMACAddress(testHostMAC),
				},
			},
			wantErr: true,
		},
		{
			name: "Failed_InvalidOSType",
			args: args{
				options: []cloudinit.Option{
					cloudinit.WithOSType(osv1.OsType_OS_TYPE_UNSPECIFIED),
					cloudinit.WithTenantID(testTenantID),
					cloudinit.WithHostname(testHostname),
					cloudinit.WithClientCredentials(testClientID, testClientSecret),
					cloudinit.WithHostMACAddress(testHostMAC),
				},
			},
			wantErr: true,
		},
		{
			name: "Failed_DevModeNoUser",
			args: args{
				options: []cloudinit.Option{
					cloudinit.WithOSType(osv1.OsType_OS_TYPE_IMMUTABLE),
					cloudinit.WithTenantID(testTenantID),
					cloudinit.WithHostname(testHostname),
					cloudinit.WithDevMode("", ""),
					cloudinit.WithClientCredentials(testClientID, testClientSecret),
					cloudinit.WithHostMACAddress(testHostMAC),
				},
			},
			wantErr: true,
		},
		{
			name: "Failed_NoClientCredentials",
			args: args{
				options: []cloudinit.Option{
					cloudinit.WithOSType(osv1.OsType_OS_TYPE_IMMUTABLE),
					cloudinit.WithHostname(testHostname),
					cloudinit.WithTenantID(testTenantID),
					cloudinit.WithHostMACAddress(testHostMAC),
				},
			},
			wantErr: true,
		},
		{
			name: "Failed_NoHostMAC",
			args: args{
				options: []cloudinit.Option{
					cloudinit.WithOSType(osv1.OsType_OS_TYPE_IMMUTABLE),
					cloudinit.WithHostname(testHostname),
					cloudinit.WithTenantID(testTenantID),
					cloudinit.WithClientCredentials(testClientID, testClientSecret),
				},
			},
			wantErr: true,
		},
		{
			name: "Dummy testTemplate",
			args: args{
				template: "test-template.cfg",
				options: []cloudinit.Option{
					cloudinit.WithDevMode("user", "pass"),
					cloudinit.WithOSType(osv1.OsType_OS_TYPE_IMMUTABLE),
					cloudinit.WithHostname(testHostname),
					cloudinit.WithTenantID(testTenantID),
					cloudinit.WithClientCredentials(testClientID, testClientSecret),
					cloudinit.WithHostMACAddress(testHostMAC),
				},
			},
			expectedOutputFileName: "test-template.cfg",
			wantErr:                false,
		},
		{
			name: "Success_Base_ImmutableOS_WithLocal",
			args: args{
				options: []cloudinit.Option{
					cloudinit.WithLocalAccount("user", "sshkey"),
					cloudinit.WithOSType(osv1.OsType_OS_TYPE_IMMUTABLE),
					cloudinit.WithHostname(testHostname),
					cloudinit.WithTenantID(testTenantID),
					cloudinit.WithClientCredentials(testClientID, testClientSecret),
					cloudinit.WithHostMACAddress(testHostMAC),
				},
				infraConfigOverride: func(infraConfig config.InfraConfig) config.InfraConfig {
					newCfg := infraConfig
					newCfg.ClusterURL = clusterURL
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
			expectedOutputFileName: "expected-installer-11.cfg",
			wantErr:                false,
		},
		{
			name: "Success_Base_ImmutableOS_WithLocal_And_DevAccount",
			args: args{
				options: []cloudinit.Option{
					cloudinit.WithDevMode("user", "pass"),
					cloudinit.WithLocalAccount("user", "sshkey"),
					cloudinit.WithOSType(osv1.OsType_OS_TYPE_IMMUTABLE),
					cloudinit.WithHostname(testHostname),
					cloudinit.WithTenantID(testTenantID),
					cloudinit.WithClientCredentials(testClientID, testClientSecret),
					cloudinit.WithHostMACAddress(testHostMAC),
				},
				infraConfigOverride: func(infraConfig config.InfraConfig) config.InfraConfig {
					newCfg := infraConfig
					newCfg.ClusterURL = clusterURL
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
			expectedOutputFileName: "expected-installer-12.cfg",
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

			got, err := cloudinit.GenerateFromInfraConfig(tt.args.template, config.GetInfraConfig(), tt.args.options...)
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
