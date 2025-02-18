// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package curation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	dkam_testing "github.com/intel/infra-onboarding/dkam/testing"

	osv1 "github.com/intel/infra-core/inventory/v2/pkg/api/os/v1"
	"github.com/intel/infra-onboarding/dkam/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testOSProfileName = "test-profile"
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
	config.ScriptPath = strings.Replace(currentDir, "curation", "script", -1)
	config.PVC, err = os.MkdirTemp(os.TempDir(), "test_pvc")
	if err != nil {
		panic(fmt.Sprintf("Error creating temp directory: %v", err))
	}

	cleanupFunc := dkam_testing.StartTestReleaseService(testOSProfileName)
	defer cleanupFunc()

	run := m.Run()
	os.Exit(run)
}

func Test_ParseJSONUfwRules(t *testing.T) {
	tests := map[string]struct {
		jsonUfw     string
		expectedUfw []FirewallRule
		valid       bool
	}{
		"wrongStringUfw": {
			jsonUfw: "test_wrong_JSON",
			valid:   false,
		},
		"emptyStringUfw": {
			jsonUfw:     "",
			expectedUfw: make([]FirewallRule, 0),
			valid:       true,
		},
		"emptyListUfw": {
			jsonUfw:     "[]",
			expectedUfw: make([]FirewallRule, 0),
			valid:       true,
		},
		"singleUfwRule": {
			jsonUfw: `[{"sourceIp":"kind.internal", "ipVer": "ipv4", "protocol": "tcp", "ports": "6443,10250"}]`,
			expectedUfw: []FirewallRule{
				{
					SourceIP: "kind.internal",
					Ports:    "6443,10250",
					IPVer:    "ipv4",
					Protocol: "tcp",
				},
			},
			valid: true,
		},
		"multipleUfwRule": {
			jsonUfw: `[	
	{"sourceIp":"", "ipVer": "", "protocol": "tcp", "ports": "2379,2380,6443,9345,10250,5473"},
    {"sourceIp":"", "ipVer": "", "protocol": "", "ports": "7946"},
    {"sourceIp":"", "ipVer": "", "protocol": "udp", "ports": "123"}
]`,
			expectedUfw: []FirewallRule{
				{
					SourceIP: "",
					IPVer:    "",
					Protocol: "tcp",
					Ports:    "2379,2380,6443,9345,10250,5473",
				},
				{
					SourceIP: "",
					IPVer:    "",
					Protocol: "",
					Ports:    "7946",
				},
				{
					SourceIP: "",
					IPVer:    "",
					Protocol: "udp",
					Ports:    "123",
				},
			},
			valid: true,
		},
		"multipleUfwRuleOmitEmpty": {
			jsonUfw: `[	
	{"protocol": "tcp", "ports": "2379,2380,6443,9345,10250,5473"},
    {"ports": "7946"},
    {"protocol": "udp", "ports": "123"}
]`,
			expectedUfw: []FirewallRule{
				{
					SourceIP: "",
					IPVer:    "",
					Protocol: "tcp",
					Ports:    "2379,2380,6443,9345,10250,5473",
				},
				{
					SourceIP: "",
					IPVer:    "",
					Protocol: "",
					Ports:    "7946",
				},
				{
					SourceIP: "",
					IPVer:    "",
					Protocol: "udp",
					Ports:    "123",
				},
			},
			valid: true,
		},
	}

	for tcname, tc := range tests {
		t.Run(tcname, func(t *testing.T) {
			parsedRules, err := ParseJSONFirewallRules(tc.jsonUfw)
			if !tc.valid {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedUfw, parsedRules)
			}
		})
	}
}

func Test_GenerateUFWCommand(t *testing.T) {
	tests := map[string]struct {
		ufwRule            FirewallRule
		expectedUfwCommand []string
	}{
		"empty": {
			ufwRule:            FirewallRule{},
			expectedUfwCommand: []string{},
		},
		"rule1": {
			ufwRule: FirewallRule{
				SourceIP: "kind.internal",
				Ports:    "6443,10250",
				IPVer:    "ipv4",
				Protocol: "tcp",
			},
			expectedUfwCommand: []string{"ufw allow from $(dig +short kind.internal | tail -n1) to any port 6443,10250 proto tcp"},
		},
		"rule2": {
			ufwRule: FirewallRule{
				SourceIP: "",
				IPVer:    "",
				Protocol: "tcp",
				Ports:    "2379,2380,6443,9345,10250,5473",
			},
			expectedUfwCommand: []string{"ufw allow in to any port 2379,2380,6443,9345,10250,5473 proto tcp"},
		},
		"rule3": {
			ufwRule: FirewallRule{
				SourceIP: "",
				IPVer:    "",
				Protocol: "",
				Ports:    "7946",
			},
			expectedUfwCommand: []string{"ufw allow in to any port 7946"},
		},
		"rule4": {
			ufwRule: FirewallRule{
				SourceIP: "",
				IPVer:    "",
				Protocol: "udp",
				Ports:    "123",
			},
			expectedUfwCommand: []string{"ufw allow in to any port 123 proto udp"},
		},
		"rule5": {
			ufwRule: FirewallRule{
				SourceIP: "kind.internal",
				Ports:    "",
				IPVer:    "ipv4",
				Protocol: "tcp",
			},
			expectedUfwCommand: []string{"ufw allow from $(dig +short kind.internal | tail -n1) proto tcp"},
		},
		"rule6": {
			ufwRule: FirewallRule{
				SourceIP: "kind.internal",
				Ports:    "",
				IPVer:    "ipv4",
				Protocol: "",
			},
			expectedUfwCommand: []string{"ufw allow from $(dig +short kind.internal | tail -n1)"},
		},
		"rule7": {
			ufwRule: FirewallRule{
				SourceIP: "kind.internal",
				Ports:    "1234",
				IPVer:    "ipv4",
				Protocol: "",
			},
			expectedUfwCommand: []string{"ufw allow from $(dig +short kind.internal | tail -n1) to any port 1234"},
		},
		"rule8": {
			ufwRule: FirewallRule{
				SourceIP: "",
				IPVer:    "",
				Protocol: "abc",
				Ports:    "",
			},
			expectedUfwCommand: []string{},
		},
		"rule9": {
			ufwRule: FirewallRule{
				SourceIP: "0000:000::00",
				Ports:    "6443,10250",
				IPVer:    "ipv4",
				Protocol: "tcp",
			},
			expectedUfwCommand: []string{"ufw allow from 0000:000::00 to any port 6443,10250 proto tcp"},
		},
	}
	for tcname, tc := range tests {
		t.Run(tcname, func(t *testing.T) {
			ufwCommands := GenerateUFWCommands(tc.ufwRule)
			assert.Equal(t, tc.expectedUfwCommand, ufwCommands)
		})
	}
}

func Test_GetCuratedScript(t *testing.T) {
	dkam_testing.PrepareTestInfraConfig(t)
	dkam_testing.PrepareTestReleaseFile(t, projectRoot)
	dkam_testing.PrepareTestCaCertificateFile(t)

	dummyData := `#!/bin/bash
	enable_netipplan
        install_intel_CAcertificates
# Add your installation commands here
`

	err := os.WriteFile(config.PVC+"/installer.sh", []byte(dummyData), 0o755)
	require.NoError(t, err)
	defer func() {
		os.Remove(config.PVC + "/installer.sh")
	}()

	osr := &osv1.OperatingSystemResource{
		ProfileName: testOSProfileName,
		OsType:      osv1.OsType_OS_TYPE_MUTABLE,
	}

	err = CurateScript(context.TODO(), osr)
	require.NoError(t, err)
}

func Test_GetCuratedScript_Case(t *testing.T) {
	dkam_testing.PrepareTestReleaseFile(t, projectRoot)
	dkam_testing.PrepareTestCaCertificateFile(t)

	os.MkdirAll(config.DownloadPath, 0o755)

	os.Setenv("MODE", "dev")
	os.Setenv("EN_HTTP_PROXY", "proxy")
	os.Setenv("EN_HTTPS_PROXY", "proxy")
	os.Setenv("EN_NO_PROXY", "proxy")
	os.Setenv("EN_FTP_PROXY", "proxy")
	os.Setenv("EN_SOCKS_PROXY", "proxy")
	os.Setenv("ORCH_CLUSTER", "kind.internal")
	defer func() {
		os.Unsetenv("ORCH_CLUSTER")
		os.Unsetenv("MODE")
		os.Unsetenv("EN_HTTP_PROXY")
		os.Unsetenv("EN_HTTPS_PROXY")
		os.Unsetenv("EN_NO_PROXY")
		os.Unsetenv("EN_FTP_PROXY")
		os.Unsetenv("EN_SOCKS_PROXY")
		os.Unsetenv("MODE")
		os.Unsetenv("FIREWALL_REQ_ALLOW")
	}()

	osr := &osv1.OperatingSystemResource{
		ProfileName: testOSProfileName,
		OsType:      osv1.OsType_OS_TYPE_IMMUTABLE,
	}
	os.Setenv("FIREWALL_REQ_ALLOW", `[
    {
        "sourceIp": "192.168.1.1",
        "ports": "80",
        "ipVer": "",
        "protocol": "tcp"
    },
    {
        "sourceIp": "192.168.1.1",
        "ports": "53,123,161",
        "ipVer": "",
        "protocol": "udp"
    },
    {
        "sourceIp": "example.com",
        "ports": "443",
        "ipVer": "",
        "protocol": "tcp"
    },
    {
        "sourceIp": "",
        "ports": "22",
        "ipVer": "",
        "protocol": "tcp"
    },
    {
        "sourceIp": "",
        "ports": "53,123",
        "ipVer": "",
        "protocol": "udp"
    },
    {
        "sourceIp": "192.168.1.1",
        "ports": "8080",
        "ipVer": "",
        "protocol": ""
    },
    {
        "sourceIp": "",
        "ports": "80,443",
        "ipVer": "",
        "protocol": ""
    },
    {
        "sourceIp": "",
        "ports": "",
        "ipVer": "",
        "protocol": ""
    },
    {
        "sourceIp": "example.com",
        "ports": "80,443",
        "ipVer": "",
        "protocol": ""
    },
	{
        "sourceIp": "192.168.1.1",
        "ports": "",
        "ipVer": "",
        "protocol": "tcp"
    },
    {
        "sourceIp": "",
        "ports": "",
        "ipVer": "",
        "protocol": "udp"
    },
	{
        "sourceIp": "192.168.1.1",
        "ports": "",
        "ipVer": "",
        "protocol": ""
    }
]
`)

	err := CurateScript(context.TODO(), osr)
	assert.NoError(t, err)
}

func Test_GetCuratedScript_Case1(t *testing.T) {
	dkam_testing.PrepareTestReleaseFile(t, projectRoot)
	dkam_testing.PrepareTestCaCertificateFile(t)

	os.Setenv("ORCH_CLUSTER", "kind.internal")
	os.MkdirAll(config.DownloadPath, 0o755)

	dummyData := `#!/bin/bash
	enable_netipplan
        install_intel_CAcertificates
# Add your installation commands here
`
	err := os.WriteFile(config.PVC+"/installer.sh", []byte(dummyData), 0o755)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}

	os.Setenv("ORCH_CLUSTER", "kind.internal")
	defer os.Unsetenv("ORCH_CLUSTER")
	osr := &osv1.OperatingSystemResource{
		ProfileName: testOSProfileName,
		OsType:      osv1.OsType_OS_TYPE_MUTABLE,
	}
	err = CurateScript(context.TODO(), osr)

	assert.NoError(t, err)
	defer func() {
		os.Unsetenv("ORCH_CLUSTER")
		os.Remove(config.PVC + "/installer.sh")
	}()
}

func Test_GetCuratedScript_Case2(t *testing.T) {
	dkam_testing.PrepareTestReleaseFile(t, projectRoot)
	dkam_testing.PrepareTestCaCertificateFile(t)

	os.Setenv("SOCKS_PROXY", "proxy")
	os.MkdirAll(config.DownloadPath, 0o755)

	dummyData := `#!/bin/bash
	enable_netipplan
        install_intel_CAcertificates
# Add your installation commands here
`
	err := os.WriteFile(config.PVC+"/installer.sh", []byte(dummyData), 0o755)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}

	os.Setenv("ORCH_CLUSTER", "kind.internal")
	defer os.Unsetenv("ORCH_CLUSTER")
	osr := &osv1.OperatingSystemResource{
		ProfileName: testOSProfileName,
		OsType:      osv1.OsType_OS_TYPE_MUTABLE,
	}
	err = CurateScript(context.TODO(), osr)

	assert.NoError(t, err)

	defer func() {
		os.Unsetenv("SOCKS_PROXY")
		os.Remove(config.PVC + "/installer.sh")
	}()
}

func Test_GetCuratedScript_Case3(t *testing.T) {
	dkam_testing.PrepareTestReleaseFile(t, projectRoot)
	dkam_testing.PrepareTestCaCertificateFile(t)

	os.MkdirAll(config.DownloadPath, 0o755)

	dummyData := `#!/bin/bash
	enable_netipplan
        install_intel_CAcertificates
# Add your installation commands here
`
	err := os.WriteFile(config.PVC+"/installer.sh", []byte(dummyData), 0o755)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}
	result := strings.Replace(currentDir, "curation", "script/tmp", -1)
	res := filepath.Join(result, "latest-dev.yaml")
	if err := os.MkdirAll(filepath.Dir(res), 0o755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	src := strings.Replace(currentDir, "curation", "script/latest-dev.yaml", -1)
	dkam_testing.CopyFile(src, res)
	os.Setenv("NETIP", "static")
	os.Setenv("ORCH_CLUSTER", "kind.internal")
	defer os.Unsetenv("ORCH_CLUSTER")
	osr := &osv1.OperatingSystemResource{
		ProfileName: testOSProfileName,
		OsType:      osv1.OsType_OS_TYPE_MUTABLE,
	}
	err = CurateScript(context.TODO(), osr)

	assert.NoError(t, err)
	defer func() {
		os.Unsetenv("NETIP")
		dkam_testing.CopyFile(res, src)
		os.Remove(res)
		os.Remove(config.PVC + "/installer.sh")
	}()
}

func Test_GetCuratedScript_Case4(t *testing.T) {
	dkam_testing.PrepareTestReleaseFile(t, projectRoot)
	dkam_testing.PrepareTestCaCertificateFile(t)

	os.MkdirAll(config.DownloadPath, 0o755)

	dummyData := `#!/bin/bash
	enable_netipplan
        install_intel_CAcertificates
# Add your installation commands here
`
	err := os.WriteFile(config.PVC+"/installer.sh", []byte(dummyData), 0o755)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}

	result := strings.Replace(currentDir, "curation", "script/tmp", -1)
	res := filepath.Join(result, "latest-dev.yaml")
	if err := os.MkdirAll(filepath.Dir(res), 0o755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	src := strings.Replace(currentDir, "curation", "script/latest-dev.yaml", -1)
	dkam_testing.CopyFile(src, res)
	os.Setenv("NETIP", "static")
	direc := config.PVC + "/tmp/"
	os.MkdirAll(direc, 0o755)
	os.Create(direc + "latest-dev.yaml")
	dkam_testing.CopyFile(src, direc+"latest-dev.yaml")
	os.Setenv("ORCH_CLUSTER", "kind.internal")
	defer os.Unsetenv("ORCH_CLUSTER")
	osr := &osv1.OperatingSystemResource{
		ProfileName: testOSProfileName,
		OsType:      osv1.OsType_OS_TYPE_MUTABLE,
	}
	err = CurateScript(context.TODO(), osr)

	assert.NoError(t, err)
	defer func() {
		os.Unsetenv("NETIP")
		dkam_testing.CopyFile(res, src)
		os.Remove(res)
		os.Remove(config.PVC + "/installer.sh")
	}()
}

func TestGetCuratedScript(t *testing.T) {
	dkam_testing.PrepareTestReleaseFile(t, projectRoot)
	dkam_testing.PrepareTestCaCertificateFile(t)
	os.Setenv("ORCH_CLUSTER", "kind.internal")
	defer os.Unsetenv("ORCH_CLUSTER")

	type args struct {
		profile string
		sha256  string
		osType  osv1.OsType
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "CurateScript test case",
			args: args{
				profile: testOSProfileName,
				sha256:  "",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			osr := &osv1.OperatingSystemResource{
				ProfileName: tt.args.profile,
				OsType:      osv1.OsType_OS_TYPE_MUTABLE,
			}
			if err := CurateScript(context.TODO(), osr); (err != nil) != tt.wantErr {
				t.Errorf("CurateScript() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCurateScriptFromTemplate(t *testing.T) {
	templateVars := map[string]interface{}{
		"TEST_1": "test",
		"TEST_2": "test",
	}

	t.Run("Success", func(t *testing.T) {
		got, err := CurateFromTemplate("{{ .TEST_1 }} {{ .TEST_2 }}", templateVars)
		require.NoError(t, err)
		require.Equal(t, "test test", got)
	})

	t.Run("Failed_MissingVariable", func(t *testing.T) {
		_, err := CurateFromTemplate("{{ .TEST_1 }} {{ .TEST_3 }}", templateVars)
		require.Error(t, err)
	})

	t.Run("Failed_InvalidTemplate", func(t *testing.T) {
		_, err := CurateFromTemplate("{{ .TEST_1", templateVars)
		require.Error(t, err)
	})
}
