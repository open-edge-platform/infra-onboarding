// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package curation

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	dkam_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/testing"

	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/api/os/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/pkg/config"
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
	cleanupFunc := dkam_testing.StartTestReleaseService(testOSProfileName)
	defer cleanupFunc()
	if err != nil {
		panic(fmt.Sprintf("Error creating temp directory: %v", err))
	}
	run := m.Run()
	os.Exit(run)
}

func Test_ParseJSONUfwRules(t *testing.T) {
	tests := map[string]struct {
		jsonUfw     string
		expectedUfw []Rule
		valid       bool
	}{
		"wrongStringUfw": {
			jsonUfw: "test_wrong_JSON",
			valid:   false,
		},
		"emptyStringUfw": {
			jsonUfw:     "",
			expectedUfw: make([]Rule, 0),
			valid:       true,
		},
		"emptyListUfw": {
			jsonUfw:     "[]",
			expectedUfw: make([]Rule, 0),
			valid:       true,
		},
		"singleUfwRule": {
			jsonUfw: `[{"sourceIp":"kind.internal", "ipVer": "ipv4", "protocol": "tcp", "ports": "6443,10250"}]`,
			expectedUfw: []Rule{
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
			expectedUfw: []Rule{
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
			expectedUfw: []Rule{
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
			parsedRules, err := ParseJSONUfwRules(tc.jsonUfw)
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
		ufwRule            Rule
		expectedUfwCommand []string
	}{
		"empty": {
			ufwRule:            Rule{},
			expectedUfwCommand: []string{},
		},
		"rule1": {
			ufwRule: Rule{
				SourceIP: "kind.internal",
				Ports:    "6443,10250",
				IPVer:    "ipv4",
				Protocol: "tcp",
			},
			expectedUfwCommand: []string{"ufw allow from $(dig +short kind.internal | tail -n1) to any port 6443,10250 proto tcp"},
		},
		"rule2": {
			ufwRule: Rule{
				SourceIP: "",
				IPVer:    "",
				Protocol: "tcp",
				Ports:    "2379,2380,6443,9345,10250,5473",
			},
			expectedUfwCommand: []string{"ufw allow in to any port 2379,2380,6443,9345,10250,5473 proto tcp"},
		},
		"rule3": {
			ufwRule: Rule{
				SourceIP: "",
				IPVer:    "",
				Protocol: "",
				Ports:    "7946",
			},
			expectedUfwCommand: []string{"ufw allow in to any port 7946"},
		},
		"rule4": {
			ufwRule: Rule{
				SourceIP: "",
				IPVer:    "",
				Protocol: "udp",
				Ports:    "123",
			},
			expectedUfwCommand: []string{"ufw allow in to any port 123 proto udp"},
		},
		"rule5": {
			ufwRule: Rule{
				SourceIP: "kind.internal",
				Ports:    "",
				IPVer:    "ipv4",
				Protocol: "tcp",
			},
			expectedUfwCommand: []string{"ufw allow from $(dig +short kind.internal | tail -n1) proto tcp"},
		},
		"rule6": {
			ufwRule: Rule{
				SourceIP: "kind.internal",
				Ports:    "",
				IPVer:    "ipv4",
				Protocol: "",
			},
			expectedUfwCommand: []string{"ufw allow from $(dig +short kind.internal | tail -n1)"},
		},
		"rule7": {
			ufwRule: Rule{
				SourceIP: "kind.internal",
				Ports:    "1234",
				IPVer:    "ipv4",
				Protocol: "",
			},
			expectedUfwCommand: []string{"ufw allow from $(dig +short kind.internal | tail -n1) to any port 1234"},
		},
		"rule8": {
			ufwRule: Rule{
				SourceIP: "",
				IPVer:    "",
				Protocol: "abc",
				Ports:    "",
			},
			expectedUfwCommand: []string{},
		},
		"rule9": {
			ufwRule: Rule{
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
	dkam_testing.PrepareTestReleaseFile(t, projectRoot)
	dkam_testing.PrepareTestCaCertificateFile(t)

	os.Setenv("NETIP", "static")

	os.MkdirAll(config.DownloadPath, 0o755)
	os.Setenv("ORCH_CLUSTER", "kind.internal")
	defer os.Unsetenv("ORCH_CLUSTER")
	dummyData := `#!/bin/bash
	enable_netipplan
        install_intel_CAcertificates
# Add your installation commands here
`
	os.Setenv("EN_HTTP_PROXY", "proxy")
	os.Setenv("EN_HTTPS_PROXY", "proxy")
	os.Setenv("EN_NO_PROXY", "proxy")
	os.Setenv("EN_FTP_PROXY", "proxy")
	os.Setenv("EN_SOCKS_PROXY", "proxy")
	defer func() {
		os.Unsetenv("NETIP")
		os.Unsetenv("EN_HTTP_PROXY")
		os.Unsetenv("EN_HTTPS_PROXY")
		os.Unsetenv("EN_NO_PROXY")
		os.Unsetenv("EN_FTP_PROXY")
		os.Unsetenv("EN_SOCKS_PROXY")
	}()

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

func TestGetReleaseArtifactList(t *testing.T) {
	type args struct {
		filePath string
	}
	tests := []struct {
		name    string
		args    args
		want    Config
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				filePath: "",
			},
			want:    Config{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetReleaseArtifactList(tt.args.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetReleaseArtifactList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestGetReleaseArtifactList_NegativeCase(t *testing.T) {
	result := strings.Replace(currentDir, "curation", "script/tmp", -1)
	res := filepath.Join(result, "latest-dev.yaml")
	if err := os.MkdirAll(filepath.Dir(res), 0o755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	src := strings.Replace(currentDir, "curation", "script/latest-dev.yaml", -1)
	dkam_testing.CopyFile(src, res)
	dummyData := `#!/bin/bash
	enable_netipplan
        install_intel_CAcertificates
# Add your installation commands here
`
	err := os.WriteFile(res, []byte(dummyData), 0o644)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	type args struct {
		filePath string
	}
	tests := []struct {
		name    string
		args    args
		want    Config
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				filePath: res,
			},
			want:    Config{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetReleaseArtifactList(tt.args.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetReleaseArtifactList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
	defer func() {
		os.Remove(res)
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
	testCaCert := "-----BEGIN CERTIFICATE-----\nMIIFYTCCA0mgAwIBAgIRAKbmACDFdpXWP890dsFbSaUwDQYJKoZIhvcNAQELBQAw\nKTELMAkGA1UEBhMCVVMxGjAYBgNVBAoTEUludGVsIENvcnBvcmF0aW9uMB4XDTI1\nMDEwMjE1NDgyNFoXDTI3MDEwMjE1NDgyNFowKTELMAkGA1UEBhMCVVMxGjAYBgNV\nBAoTEUludGVsIENvcnBvcmF0aW9uMIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIIC\nCgKCAgEAn4WF0VQity1EJWfltgSMGbRDFaX8ML97tcIY8vtWRrLgYYkxN/OUcI4k\n3NUO9gofa1sZ9FsNqyyPozQtJjd67PnngG5IJgNqjUEVUpwezk0AjuopLxH138NA\n3PebgKHztYHGA1K69QVBwuvI8PvuJ5ic37YUj4qH4djQdwwlpEMAM3l9OST2Mk64\nk7yXFkP79bx33Q01q5zreQ6WvzDl5a17mFDjotUhKh0udR4XKn+/8hBEs28ohBZa\nl4zXIqbw1V1T8baQdJsB5VaItlXJ40IWhYuCh5NtW71toFcePWP/ef+LwjvwZYo0\niPB72mxoRACmL8z7vpFD71Sdn6mBDhI34wMXYhwLtU+P7ySGvsS2PQWHggYsQdbv\n74IBtIBVmt5nrjTAJzaWKDQSz2u3iJvct/CnfS8wp4xjH53qeSqp0ToPwqNy0iQF\n/z2uLn//eIyc//pbe59zehrN09SqxCGJhQr7CdoxO/PC6ZtemKg28WSl8iSaBvvp\njGS7Kaj5RPuRN3Ms1RymJV0FuMMdNfy7kn4KmPYVl4y6CRcD+orCDj3RTsycEP6c\nf/ulT2NzX6k8wJBq3yOjUim54HaPnSbUEmmhJnnrHyVqtHOv4IgJgr/F5Bjyv8+o\nHp5uyL/oidLaj7bO2rsNWrkJawpuWs3gr+aheesYHzBvKJMttmMCAwEAAaOBgzCB\ngDAOBgNVHQ8BAf8EBAMCAqQwEwYDVR0lBAwwCgYIKwYBBQUHAwEwDwYDVR0TAQH/\nBAUwAwEB/zAdBgNVHQ4EFgQUP5dseOrhR0SEA1eJXvDazoI/5ogwKQYDVR0RBCIw\nIIINa2luZC5pbnRlcm5hbIIPKi5raW5kLmludGVybmFsMA0GCSqGSIb3DQEBCwUA\nA4ICAQBfyjP6vxpMHSPHn1Buz4+TZa8an0knz+0iAuLUYyUSBTNw6JCwcvQ0fVIZ\ngYXjyJZ16KhpvvygoL78CR72aC/TayeJHAwtsGWLLh1PXXtdZ36x/5SoedVLbChA\nD0HFRaFYzSlMd5yja8ECYUKv+qyb5WhhE+8qAct2h7BHG2RqzGru8U8I52WIXE1O\nWUb+EL+4TbWc3ARNpFER9HAy3ZXuUQax+tcPVSRDFpGcAFULjRFz8MyJ1hp9h3eX\nHwbitJvn/tmEP2tuIPUNN4yrYP3fpFJhjIYOrR2e7OaVRdJZMyw6vFHsceRNw0mv\n3O4Fa/O3bO9v9p/PHJlQMBqo8Tx8wYYnsRtpTxipwleKxBv+NtQw11g8Twh23ngQ\nh6O1i3pKst5eB6IJV5s5tXHdMKj2tk0iJcZ/BuZk9iRWguSgX3Qyb+1eUgbn4Ypa\nrLySufMbDv+LzxwTvQ7xWjVerhgiD7PAxNl0vCAN+rvpUonhDdBtjN8PIG0cjCRx\nwlEyL3+eQa58bIAxDRc97UmxUdhbjKGcL5E9JMR8t6XpFj+UKiW90zxx7ckLgyh+\n3+6q7nWtoAGeX1kZqBCc9idN1+0wp9F3xlG7VoVCtAf8rhNRA1l/3rpHb5ACfSrb\nFUiNbT1AeeTP59OwEBIOHNWAq58TBYTVItg7Sjqc6L2eavSWwA==\n-----END CERTIFICATE-----"
	baseClusterVariables := map[string]interface{}{
		"MODE":                           "dev",
		"ORCH_CLUSTER":                   "cluster.kind.internal",
		"ORCH_INFRA":                     "infra.test",
		"ORCH_UPDATE":                    "update.test",
		"ORCH_PLATFORM_OBS_HOST":         "obs.test",
		"ORCH_PLATFORM_OBS_PORT":         "1234",
		"ORCH_PLATFORM_OBS_METRICS_HOST": "metrics.test",
		"ORCH_PLATFORM_OBS_METRICS_PORT": "5678",
		"ORCH_TELEMETRY_HOST":            "telemetry.test",
		"ORCH_TELEMETRY_PORT":            "1234",
		"KEYCLOAK_URL":                   "keycloak.test",
		"KEYCLOAK_FQDN":                  "keycloak.test",
		"RELEASE_TOKEN_URL":              "release-svc.test",
		"RELEASE_FQDN":                   "release-svc.test",
		"ORCH_APT_PORT":                  "1234",
		"ORCH_IMG_PORT":                  "1234",
		"FILE_SERVER":                    "file-server.test",
		"IMG_REGISTRY_URL":               "registry-svc.test",
		"NTP_SERVERS":                    "ntp1.test,ntp2.test",
		"EN_HTTP_PROXY":                  "http-proxy.test",
		"EN_HTTPS_PROXY":                 "https-proxy.test",
		"EN_NO_PROXY":                    "no-proxy.test",
		"EN_FTP_PROXY":                   "ftp-proxy.test",
		"EN_SOCKS_PROXY":                 "socks-server.test",

		"CA_CERT": testCaCert,

		"IS_TIBEROS":        true,
		"FIREWALL_PROVIDER": "iptables",
		"FIREWALL_RULES": []string{
			"iptables -A INPUT -p tcp --dport 80 -j ACCEPT",
			"iptables -A INPUT -p tcp --dport 443 -j ACCEPT",
		},

		"EXTRA_HOSTS": strings.Split("1.1.1.1 a.test,2.2.2.2 b.test", ","),
	}

	copyBaseVariables := func() map[string]interface{} {
		templVarMap := make(map[string]interface{})
		for k, v := range baseClusterVariables {
			templVarMap[k] = v
		}
		return templVarMap
	}

	type args struct {
		templateVariables map[string]interface{}
	}
	tests := []struct {
		name                   string
		args                   args
		expectedOutputFileName string
		wantErr                bool
	}{
		{
			name: "Success_Base",
			args: args{
				templateVariables: baseClusterVariables,
			},
			expectedOutputFileName: "expected-installer-01.cfg",
			wantErr:                false,
		},
		{
			name: "Success_NotKindInternal",
			args: args{
				templateVariables: func() map[string]interface{} {
					templVarMap := copyBaseVariables()
					templVarMap["ORCH_CLUSTER"] = "cluster.not-kind.internal"
					return templVarMap
				}(),
			},
			expectedOutputFileName: "expected-installer-02.cfg",
			wantErr:                false,
		},
		{
			name: "Success_ProdMode",
			args: args{
				templateVariables: func() map[string]interface{} {
					templVarMap := copyBaseVariables()
					templVarMap["MODE"] = "prod"
					return templVarMap
				}(),
			},
			expectedOutputFileName: "expected-installer-03.cfg",
			wantErr:                false,
		},
		{
			name: "Success_NoProxies",
			args: args{
				templateVariables: func() map[string]interface{} {
					templVarMap := copyBaseVariables()
					templVarMap["EN_HTTP_PROXY"] = ""
					templVarMap["EN_HTTPS_PROXY"] = ""
					templVarMap["EN_NO_PROXY"] = ""
					templVarMap["EN_FTP_PROXY"] = ""
					templVarMap["EN_SOCKS_PROXY"] = ""
					return templVarMap
				}(),
			},
			expectedOutputFileName: "expected-installer-04.cfg",
			wantErr:                false,
		},
		{
			name: "Success_SelectedProxies",
			args: args{
				templateVariables: func() map[string]interface{} {
					templVarMap := copyBaseVariables()
					templVarMap["EN_FTP_PROXY"] = ""
					templVarMap["EN_SOCKS_PROXY"] = ""
					return templVarMap
				}(),
			},
			expectedOutputFileName: "expected-installer-05.cfg",
			wantErr:                false,
		},
		{
			name: "Success_NoIptablesRules",
			args: args{
				templateVariables: func() map[string]interface{} {
					templVarMap := copyBaseVariables()
					templVarMap["FIREWALL_RULES"] = []string{}
					return templVarMap
				}(),
			},
			expectedOutputFileName: "expected-installer-06.cfg",
			wantErr:                false,
		},
		{
			name: "Failed_MissingTemplateVariables",
			args: args{
				templateVariables: func() map[string]interface{} {
					templVarMap := copyBaseVariables()
					delete(templVarMap, "MODE")
					return templVarMap
				}(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			templatePath := strings.Replace(currentDir, "curation", "script", -1)
			templatePath = filepath.Join(templatePath, "Installer.cfg")
			got, err := CurateScriptFromTemplate(templatePath, tt.args.templateVariables)
			if (err != nil) != tt.wantErr {
				t.Errorf("CurateScriptFromTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				fileData, err := os.ReadFile(currentDir + "/testout/" + tt.expectedOutputFileName)
				require.NoError(t, err)

				require.Equal(t, got, string(fileData))
			}
		})
	}

	// test missing input file
	t.Run("Failed_MissingInputFile", func(t *testing.T) {
		_, err := CurateScriptFromTemplate("", baseClusterVariables)
		require.Error(t, err)
	})

	t.Run("Failed_InvalidTemplateFile", func(t *testing.T) {
		_, err := CurateScriptFromTemplate(currentDir+"/testdata/invalid-template", baseClusterVariables)
		require.Error(t, err)
	})
}
