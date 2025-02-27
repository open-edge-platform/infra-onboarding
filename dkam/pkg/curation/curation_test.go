// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package curation_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	osv1 "github.com/intel/infra-core/inventory/v2/pkg/api/os/v1"
	"github.com/intel/infra-onboarding/dkam/pkg/config"
	"github.com/intel/infra-onboarding/dkam/pkg/curation"
	dkam_testing "github.com/intel/infra-onboarding/dkam/testing"
)

const (
	testOSProfileName = "test-profile"
	dummyData         = `#!/bin/bash
	enable_netipplan
        install_intel_CAcertificates
# Add your installation commands here
`
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
	config.ScriptPath = strings.ReplaceAll(currentDir, "curation", "script")
	config.PVC, err = os.MkdirTemp(os.TempDir(), "test_pvc")
	if err != nil {
		panic(fmt.Sprintf("Error creating temp directory: %v", err))
	}

	cleanupFunc := dkam_testing.StartTestReleaseService(testOSProfileName)

	run := m.Run()
	cleanupFunc()
	os.Exit(run)
}

func Test_ParseJSONUfwRules(t *testing.T) {
	tests := map[string]struct {
		jsonUfw     string
		expectedUfw []curation.FirewallRule
		valid       bool
	}{
		"wrongStringUfw": {
			jsonUfw: "test_wrong_JSON",
			valid:   false,
		},
		"emptyStringUfw": {
			jsonUfw:     "",
			expectedUfw: make([]curation.FirewallRule, 0),
			valid:       true,
		},
		"emptyListUfw": {
			jsonUfw:     "[]",
			expectedUfw: make([]curation.FirewallRule, 0),
			valid:       true,
		},
		"singleUfwRule": {
			jsonUfw: `[{"sourceIp":"kind.internal", "ipVer": "ipv4", "protocol": "tcp", "ports": "6443,10250"}]`,
			expectedUfw: []curation.FirewallRule{
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
			expectedUfw: []curation.FirewallRule{
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
			expectedUfw: []curation.FirewallRule{
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
			parsedRules, err := curation.ParseJSONFirewallRules(tc.jsonUfw)
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
		ufwRule            curation.FirewallRule
		expectedUfwCommand []string
	}{
		"empty": {
			ufwRule:            curation.FirewallRule{},
			expectedUfwCommand: []string{},
		},
		"rule1": {
			ufwRule: curation.FirewallRule{
				SourceIP: "kind.internal",
				Ports:    "6443,10250",
				IPVer:    "ipv4",
				Protocol: "tcp",
			},
			expectedUfwCommand: []string{
				"ufw allow from $(dig +short kind.internal | tail -n1) to any port 6443,10250 proto tcp",
			},
		},
		"rule2": {
			ufwRule: curation.FirewallRule{
				SourceIP: "",
				IPVer:    "",
				Protocol: "tcp",
				Ports:    "2379,2380,6443,9345,10250,5473",
			},
			expectedUfwCommand: []string{"ufw allow in to any port 2379,2380,6443,9345,10250,5473 proto tcp"},
		},
		"rule3": {
			ufwRule: curation.FirewallRule{
				SourceIP: "",
				IPVer:    "",
				Protocol: "",
				Ports:    "7946",
			},
			expectedUfwCommand: []string{"ufw allow in to any port 7946"},
		},
		"rule4": {
			ufwRule: curation.FirewallRule{
				SourceIP: "",
				IPVer:    "",
				Protocol: "udp",
				Ports:    "123",
			},
			expectedUfwCommand: []string{"ufw allow in to any port 123 proto udp"},
		},
		"rule5": {
			ufwRule: curation.FirewallRule{
				SourceIP: "kind.internal",
				Ports:    "",
				IPVer:    "ipv4",
				Protocol: "tcp",
			},
			expectedUfwCommand: []string{"ufw allow from $(dig +short kind.internal | tail -n1) proto tcp"},
		},
		"rule6": {
			ufwRule: curation.FirewallRule{
				SourceIP: "kind.internal",
				Ports:    "",
				IPVer:    "ipv4",
				Protocol: "",
			},
			expectedUfwCommand: []string{"ufw allow from $(dig +short kind.internal | tail -n1)"},
		},
		"rule7": {
			ufwRule: curation.FirewallRule{
				SourceIP: "kind.internal",
				Ports:    "1234",
				IPVer:    "ipv4",
				Protocol: "",
			},
			expectedUfwCommand: []string{"ufw allow from $(dig +short kind.internal | tail -n1) to any port 1234"},
		},
		"rule8": {
			ufwRule: curation.FirewallRule{
				SourceIP: "",
				IPVer:    "",
				Protocol: "abc",
				Ports:    "",
			},
			expectedUfwCommand: []string{},
		},
		"rule9": {
			ufwRule: curation.FirewallRule{
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
			ufwCommands := curation.GenerateUFWCommands(tc.ufwRule)
			assert.Equal(t, tc.expectedUfwCommand, ufwCommands)
		})
	}
}

func Test_GetCuratedScript(t *testing.T) {
	dkam_testing.PrepareTestInfraConfig(t)
	dkam_testing.PrepareTestReleaseFile(t, projectRoot)
	dkam_testing.PrepareTestCaCertificateFile(t)

	err := os.WriteFile(config.PVC+"/installer.sh", []byte(dummyData), 0o600)
	require.NoError(t, err)
	defer func() {
		os.Remove(config.PVC + "/installer.sh")
	}()

	osr := &osv1.OperatingSystemResource{
		ProfileName: testOSProfileName,
		OsType:      osv1.OsType_OS_TYPE_MUTABLE,
	}

	err = curation.CurateScript(context.TODO(), osr)
	require.NoError(t, err)
}

func Test_GetCuratedScript_Case(t *testing.T) {
	dkam_testing.PrepareTestReleaseFile(t, projectRoot)
	dkam_testing.PrepareTestCaCertificateFile(t)

	mkdirerr := os.MkdirAll(config.DownloadPath, 0o755)
	if mkdirerr != nil {
		fmt.Println("Error creating dir:", mkdirerr)
	}

	t.Setenv("MODE", "dev")
	t.Setenv("EN_HTTP_PROXY", "proxy")
	t.Setenv("EN_HTTPS_PROXY", "proxy")
	t.Setenv("EN_NO_PROXY", "proxy")
	t.Setenv("EN_FTP_PROXY", "proxy")
	t.Setenv("EN_SOCKS_PROXY", "proxy")
	t.Setenv("ORCH_CLUSTER", "kind.internal")
	osr := &osv1.OperatingSystemResource{
		ProfileName: testOSProfileName,
		OsType:      osv1.OsType_OS_TYPE_IMMUTABLE,
	}
	t.Setenv("FIREWALL_REQ_ALLOW", `[
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

	err := curation.CurateScript(context.TODO(), osr)
	assert.NoError(t, err)
}

func Test_GetCuratedScript_Case1(t *testing.T) {
	dkam_testing.PrepareTestReleaseFile(t, projectRoot)
	dkam_testing.PrepareTestCaCertificateFile(t)

	t.Setenv("ORCH_CLUSTER", "kind.internal")
	mkdirerr := os.MkdirAll(config.DownloadPath, 0o755)
	if mkdirerr != nil {
		fmt.Println("Error creating dir:", mkdirerr)
	}

	err := os.WriteFile(config.PVC+"/installer.sh", []byte(dummyData), 0o600)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}

	t.Setenv("ORCH_CLUSTER", "kind.internal")
	defer os.Unsetenv("ORCH_CLUSTER")
	osr := &osv1.OperatingSystemResource{
		ProfileName: testOSProfileName,
		OsType:      osv1.OsType_OS_TYPE_MUTABLE,
	}
	defer func() {
		os.Unsetenv("ORCH_CLUSTER")
		os.Remove(config.PVC + "/installer.sh")
	}()
	err = curation.CurateScript(context.TODO(), osr)
	assert.NoError(t, err)
}

func Test_GetCuratedScript_Case2(t *testing.T) {
	dkam_testing.PrepareTestReleaseFile(t, projectRoot)
	dkam_testing.PrepareTestCaCertificateFile(t)

	t.Setenv("SOCKS_PROXY", "proxy")
	mkdirerr := os.MkdirAll(config.DownloadPath, 0o755)
	if mkdirerr != nil {
		fmt.Println("Error creating dir:", mkdirerr)
	}

	err := os.WriteFile(config.PVC+"/installer.sh", []byte(dummyData), 0o600)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}

	t.Setenv("ORCH_CLUSTER", "kind.internal")
	osr := &osv1.OperatingSystemResource{
		ProfileName: testOSProfileName,
		OsType:      osv1.OsType_OS_TYPE_MUTABLE,
	}
	defer func() {
		os.Remove(config.PVC + "/installer.sh")
	}()
	err = curation.CurateScript(context.TODO(), osr)
	assert.NoError(t, err)
}

func Test_GetCuratedScript_Case3(t *testing.T) {
	dkam_testing.PrepareTestReleaseFile(t, projectRoot)
	dkam_testing.PrepareTestCaCertificateFile(t)

	mkdirerr := os.MkdirAll(config.DownloadPath, 0o755)
	if mkdirerr != nil {
		fmt.Println("Error creating dir:", mkdirerr)
	}

	err := os.WriteFile(config.PVC+"/installer.sh", []byte(dummyData), 0o600)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}
	result := strings.ReplaceAll(currentDir, "curation", "script/tmp")
	res := filepath.Join(result, "latest-dev.yaml")
	if err = os.MkdirAll(filepath.Dir(res), 0o755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	src := strings.ReplaceAll(currentDir, "curation", "script/latest-dev.yaml")
	copyFileErr := dkam_testing.CopyFile(src, res)
	if copyFileErr != nil {
		fmt.Println("Error copying file:", copyFileErr)
	}
	t.Setenv("NETIP", "static")
	t.Setenv("ORCH_CLUSTER", "kind.internal")
	osr := &osv1.OperatingSystemResource{
		ProfileName: testOSProfileName,
		OsType:      osv1.OsType_OS_TYPE_MUTABLE,
	}
	defer func() {
		copyFileErr = dkam_testing.CopyFile(res, src)
		if copyFileErr != nil {
			fmt.Println("Error copying file:", copyFileErr)
		}
		os.Remove(res)
		os.Remove(config.PVC + "/installer.sh")
	}()
	err = curation.CurateScript(context.TODO(), osr)
	assert.NoError(t, err)
}

func Test_GetCuratedScript_Case4(t *testing.T) {
	dkam_testing.PrepareTestReleaseFile(t, projectRoot)
	dkam_testing.PrepareTestCaCertificateFile(t)

	mkdirerr := os.MkdirAll(config.DownloadPath, 0o755)
	if mkdirerr != nil {
		fmt.Println("Error creating dir:", mkdirerr)
	}

	err := os.WriteFile(config.PVC+"/installer.sh", []byte(dummyData), 0o600)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}

	result := strings.ReplaceAll(currentDir, "curation", "script/tmp")
	res := filepath.Join(result, "latest-dev.yaml")
	if err = os.MkdirAll(filepath.Dir(res), 0o755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	src := strings.ReplaceAll(currentDir, "curation", "script/latest-dev.yaml")
	copyFileErr := dkam_testing.CopyFile(src, res)
	if copyFileErr != nil {
		fmt.Println("Error copying file:", copyFileErr)
	}
	t.Setenv("NETIP", "static")
	direc := config.PVC + "/tmp/"
	mkdirerr = os.MkdirAll(direc, 0o755)
	if mkdirerr != nil {
		fmt.Println("Error creating dir:", mkdirerr)
	}
	_, creationErr := os.Create(direc + "latest-dev.yaml")
	if creationErr != nil {
		fmt.Println("Error copying file:", creationErr)
	}
	copyFileErr = dkam_testing.CopyFile(src, direc+"latest-dev.yaml")
	if copyFileErr != nil {
		fmt.Println("Error copying file:", copyFileErr)
	}
	t.Setenv("ORCH_CLUSTER", "kind.internal")
	osr := &osv1.OperatingSystemResource{
		ProfileName: testOSProfileName,
		OsType:      osv1.OsType_OS_TYPE_MUTABLE,
	}
	defer func() {
		copyFileErr = dkam_testing.CopyFile(res, src)
		if copyFileErr != nil {
			fmt.Println("Error copying file:", copyFileErr)
		}
		os.Remove(res)
		os.Remove(config.PVC + "/installer.sh")
	}()
	err = curation.CurateScript(context.TODO(), osr)
	assert.NoError(t, err)
}

func TestGetCuratedScript(t *testing.T) {
	dkam_testing.PrepareTestReleaseFile(t, projectRoot)
	dkam_testing.PrepareTestCaCertificateFile(t)
	t.Setenv("ORCH_CLUSTER", "kind.internal")

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
				osType:  osv1.OsType_OS_TYPE_MUTABLE,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			osr := &osv1.OperatingSystemResource{
				ProfileName: tt.args.profile,
				OsType:      tt.args.osType,
			}
			if err := curation.CurateScript(context.TODO(), osr); (err != nil) != tt.wantErr {
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
		got, err := curation.CurateFromTemplate("{{ .TEST_1 }} {{ .TEST_2 }}", templateVars)
		require.NoError(t, err)
		require.Equal(t, "test test", got)
	})

	t.Run("Failed_MissingVariable", func(t *testing.T) {
		_, err := curation.CurateFromTemplate("{{ .TEST_1 }} {{ .TEST_3 }}", templateVars)
		require.Error(t, err)
	})

	t.Run("Failed_InvalidTemplate", func(t *testing.T) {
		_, err := curation.CurateFromTemplate("{{ .TEST_1", templateVars)
		require.Error(t, err)
	})
}
