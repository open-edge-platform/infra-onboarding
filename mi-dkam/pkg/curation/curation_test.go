// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package curation

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
					SourceIp: "kind.internal",
					Ports:    "6443,10250",
					IpVer:    "ipv4",
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
					SourceIp: "",
					IpVer:    "",
					Protocol: "tcp",
					Ports:    "2379,2380,6443,9345,10250,5473",
				},
				{
					SourceIp: "",
					IpVer:    "",
					Protocol: "",
					Ports:    "7946",
				},
				{
					SourceIp: "",
					IpVer:    "",
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
					SourceIp: "",
					IpVer:    "",
					Protocol: "tcp",
					Ports:    "2379,2380,6443,9345,10250,5473",
				},
				{
					SourceIp: "",
					IpVer:    "",
					Protocol: "",
					Ports:    "7946",
				},
				{
					SourceIp: "",
					IpVer:    "",
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
		expectedUfwCommand string
	}{
		"empty": {
			ufwRule:            Rule{},
			expectedUfwCommand: "echo Firewall rule not set 0",
		},
		"rule1": {
			ufwRule: Rule{
				SourceIp: "kind.internal",
				Ports:    "6443,10250",
				IpVer:    "ipv4",
				Protocol: "tcp",
			},
			expectedUfwCommand: "ufw allow from $(dig +short kind.internal | tail -n1) to any port 6443,10250 proto tcp",
		},
		"rule2": {
			ufwRule: Rule{
				SourceIp: "",
				IpVer:    "",
				Protocol: "tcp",
				Ports:    "2379,2380,6443,9345,10250,5473",
			},
			expectedUfwCommand: "ufw allow in to any port 2379,2380,6443,9345,10250,5473 proto tcp",
		},
		"rule3": {
			ufwRule: Rule{
				SourceIp: "",
				IpVer:    "",
				Protocol: "",
				Ports:    "7946",
			},
			expectedUfwCommand: "ufw allow in to any port 7946",
		},
		"rule4": {
			ufwRule: Rule{
				SourceIp: "",
				IpVer:    "",
				Protocol: "udp",
				Ports:    "123",
			},
			expectedUfwCommand: "ufw allow in to any port 123 proto udp",
		},
		"rule5": {
			ufwRule: Rule{
				SourceIp: "kind.internal",
				Ports:    "",
				IpVer:    "ipv4",
				Protocol: "tcp",
			},
			expectedUfwCommand: "ufw allow from $(dig +short kind.internal | tail -n1) proto tcp",
		},
		"rule6": {
			ufwRule: Rule{
				SourceIp: "kind.internal",
				Ports:    "",
				IpVer:    "ipv4",
				Protocol: "",
			},
			expectedUfwCommand: "ufw allow from $(dig +short kind.internal | tail -n1)",
		},
		"rule7": {
			ufwRule: Rule{
				SourceIp: "kind.internal",
				Ports:    "1234",
				IpVer:    "ipv4",
				Protocol: "",
			},
			expectedUfwCommand: "ufw allow from $(dig +short kind.internal | tail -n1) to any port 1234",
		},
		"rule8": {
			ufwRule: Rule{
				SourceIp: "",
				IpVer:    "",
				Protocol: "abc",
				Ports:    "",
			},
			expectedUfwCommand: "echo Firewall rule not set 0",
		},
		"rule9": {
			ufwRule: Rule{
				SourceIp: "0000:000::00",
				Ports:    "6443,10250",
				IpVer:    "ipv4",
				Protocol: "tcp",
			},
			expectedUfwCommand: "ufw allow from 0000:000::00 to any port 6443,10250 proto tcp",
		},
	}
	for tcname, tc := range tests {
		t.Run(tcname, func(t *testing.T) {
			ufwCommand := GenerateUFWCommand(tc.ufwRule)
			assert.Equal(t, tc.expectedUfwCommand, ufwCommand)
		})
	}
}

func Test_GetCuratedScript(t *testing.T) {
	os.Setenv("NETIP", "static")
	dir := config.PVC
	os.MkdirAll(dir, 0755)
	dummyData := `#!/bin/bash
	enable_netipplan
        install_intel_CAcertificates
# Add your installation commands here
`
	err := os.WriteFile(dir+"/installer.sh", []byte(dummyData), 0755)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}
	filename, version := GetCuratedScript("profile", "platform")

	// Check if the returned filename matches the expected format
	expectedFilename := config.PVC + "/" + "installer.sh"
	if filename != expectedFilename {
		t.Errorf("Expected filename '%s', but got '%s'", expectedFilename, filename)
	}
	if len(version) == 0 {
		t.Errorf("Version not found")
	}
	defer func() {
		os.Unsetenv("NETIP")
		os.RemoveAll(dir)
	}()
}

func Test_GetCuratedScript_Case(t *testing.T) {
	os.Setenv("MODE", "prod")
	dir := config.PVC
	os.MkdirAll(dir, 0755)
	dummyData := `#!/bin/bash
	enable_netipplan
        install_intel_CAcertificates
# Add your installation commands here
`
	err := os.WriteFile(dir+"/installer.sh", []byte(dummyData), 0755)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}
	filename, version := GetCuratedScript("profile", "platform")

	// Check if the returned filename matches the expected format
	expectedFilename := config.PVC + "/" + "installer.sh"
	if filename != expectedFilename {
		t.Errorf("Expected filename '%s', but got '%s'", expectedFilename, filename)
	}
	if len(version) == 0 {
		t.Errorf("Version not found")
	}
	defer func() {
		os.Unsetenv("MODE")
		os.RemoveAll(dir)
	}()
}

func Test_GetCuratedScript_Case1(t *testing.T) {
	os.Setenv("ORCH_CLUSTER", "kind.internal")
	dir := config.PVC
	os.MkdirAll(dir, 0755)
	dummyData := `#!/bin/bash
	enable_netipplan
        install_intel_CAcertificates
# Add your installation commands here
`
	err := os.WriteFile(dir+"/installer.sh", []byte(dummyData), 0755)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}
	filename, version := GetCuratedScript("profile", "platform")

	// Check if the returned filename matches the expected format
	expectedFilename := config.PVC + "/" + "installer.sh"
	if filename != expectedFilename {
		t.Errorf("Expected filename '%s', but got '%s'", expectedFilename, filename)
	}
	if len(version) == 0 {
		t.Errorf("Version not found")
	}
	defer func() {
		os.Unsetenv("ORCH_CLUSTER")
		os.RemoveAll(dir)
	}()
}

func Test_GetCuratedScript_Case2(t *testing.T) {
	os.Setenv("SOCKS_PROXY", "proxy")
	dir := config.PVC
	os.MkdirAll(dir, 0755)
	dummyData := `#!/bin/bash
	enable_netipplan
        install_intel_CAcertificates
# Add your installation commands here
`
	err := os.WriteFile(dir+"/installer.sh", []byte(dummyData), 0755)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}
	filename, version := GetCuratedScript("profile", "platform")

	// Check if the returned filename matches the expected format
	expectedFilename := config.PVC + "/" + "installer.sh"
	if filename != expectedFilename {
		t.Errorf("Expected filename '%s', but got '%s'", expectedFilename, filename)
	}
	if len(version) == 0 {
		t.Errorf("Version not found")
	}
	defer func() {
		os.Unsetenv("SOCKS_PROXY")
		os.RemoveAll(dir)
	}()
}

func Test_GetCuratedScript_Case3(t *testing.T) {
	dir := config.PVC
	os.MkdirAll(dir, 0755)
	dummyData := `#!/bin/bash
	enable_netipplan
        install_intel_CAcertificates
# Add your installation commands here
`
	err := os.WriteFile(dir+"/installer.sh", []byte(dummyData), 0755)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}
	originalDir, _ := os.Getwd()
	result := strings.Replace(originalDir, "curation", "script/tmp", -1)
	res := filepath.Join(result, "latest-dev.yaml")
	if err := os.MkdirAll(filepath.Dir(res), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	src := strings.Replace(originalDir, "curation", "script/latest-dev.yaml", -1)
	CopyFile(src, res)
	os.Setenv("NETIP", "static")
	filename, version := GetCuratedScript("profile", "platform")
	expectedFilename := config.PVC + "/" + "installer.sh"
	if filename != expectedFilename {
		t.Errorf("Expected filename '%s', but got '%s'", expectedFilename, filename)
	}
	if len(version) == 0 {
		t.Errorf("Version not found")
	}
	defer func() {
		os.Unsetenv("NETIP")
		CopyFile(res, src)
		os.Remove(res)
		os.RemoveAll(dir)
	}()
}

func Test_GetCuratedScript_Case4(t *testing.T) {
	dir := config.PVC
	os.MkdirAll(dir, 0755)
	dummyData := `#!/bin/bash
	enable_netipplan
        install_intel_CAcertificates
# Add your installation commands here
`
	err := os.WriteFile(dir+"/installer.sh", []byte(dummyData), 0755)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}
	originalDir, _ := os.Getwd()
	result := strings.Replace(originalDir, "curation", "script/tmp", -1)
	res := filepath.Join(result, "latest-dev.yaml")
	if err := os.MkdirAll(filepath.Dir(res), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	src := strings.Replace(originalDir, "curation", "script/latest-dev.yaml", -1)
	CopyFile(src, res)
	os.Setenv("NETIP", "static")
	direc := dir + "/tmp/"
	os.MkdirAll(direc, 0755)
	os.Create(direc + "latest-dev.yaml")
	CopyFile(src, direc+"latest-dev.yaml")
	filename, version := GetCuratedScript("profile", "platform")
	expectedFilename := config.PVC + "/" + "installer.sh"
	if filename != expectedFilename {
		t.Errorf("Expected filename '%s', but got '%s'", expectedFilename, filename)
	}
	if len(version) == 0 {
		t.Errorf("Version not found")
	}
	defer func() {
		os.Unsetenv("NETIP")
		CopyFile(res, src)
		os.Remove(res)
		os.RemoveAll(dir)
	}()
}

func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	if err := os.MkdirAll(filepath.Dir(src), 0755); err != nil {
		return err
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

func Test_copyFile(t *testing.T) {
	type args struct {
		src string
		dst string
	}
	wd, _ := os.Getwd()
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				src: "",
			},
			wantErr: true,
		},
		{
			name: "Test Case1",
			args: args{
				src: wd,
				dst: "",
			},
			wantErr: true,
		},
		{
			name: "Test Case 2",
			args: args{
				src: wd,
				dst: wd + "dummy",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := copyFile(tt.args.src, tt.args.dst); (err != nil) != tt.wantErr {
				t.Errorf("copyFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	defer func() {
		os.Remove(wd + "dummy")
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
	originalDir, _ := os.Getwd()
	result := strings.Replace(originalDir, "curation", "script/tmp", -1)
	res := filepath.Join(result, "latest-dev.yaml")
	if err := os.MkdirAll(filepath.Dir(res), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	src := strings.Replace(originalDir, "curation", "script/latest-dev.yaml", -1)
	CopyFile(src, res)
	dummyData := `#!/bin/bash
	enable_netipplan
        install_intel_CAcertificates
# Add your installation commands here
`
	err := os.WriteFile(res, []byte(dummyData), 0644)
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

func TestCreateOverlayScript(t *testing.T) {
	originalDir, _ := os.Getwd()
	src := strings.Replace(originalDir, "curation", "script", -1)
	dir := src + "/Installer"
	os.MkdirAll(dir, 0755)
	dataDir := config.PVC
	os.MkdirAll(dataDir, 0755)
	dummyData := `#!/bin/bash
	enable_netipplan
        install_intel_CAcertificates
# Add your installation commands here
`
	err := os.WriteFile(dataDir+"/installer.sh", []byte(dummyData), 0755)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}
	result := strings.Replace(originalDir, "curation", "script/tmp", -1)
	dst := filepath.Join(result, "Installer")
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	srcs := strings.Replace(originalDir, "curation", "script/Installer", -1)
	CopyFile(srcs, dst)
	type args struct {
		pwd     string
		profile string
		MODE    string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test Case",
			args: args{
				pwd: originalDir,
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CreateOverlayScript(tt.args.pwd, tt.args.profile, tt.args.MODE); got == tt.want {
				t.Errorf("CreateOverlayScript() = %v, want %v", got, tt.want)
			}
		})
	}
	defer func() {
		os.Remove(dst)
		os.RemoveAll(dataDir)
		CopyFile(dst, srcs)
	}()
}

func TestCreateOverlayScript_Case(t *testing.T) {
	os.Setenv("FIREWALL_REQ_ALLOW", `{
		"sourceIp": "000.000.0.000",
		"ports": "00,000",
		"ipVer": "0000",
		"protocol": "000"
	  }`,
	)
	os.Setenv("FIREWALL_CFG_ALLOW", `{
		"sourceIp": "000.000.0.000",
		"ports": "00,000",
		"ipVer": "0000",
		"protocol": "000"
	  }`,
	)
	originalDir, _ := os.Getwd()
	src := strings.Replace(originalDir, "curation", "script", -1)
	dir := src + "/Installer"
	os.MkdirAll(dir, 0755)
	dataDir := config.PVC
	os.MkdirAll(dataDir, 0755)
	dummyData := `#!/bin/bash
	enable_netipplan
        install_intel_CAcertificates
# Add your installation commands here
`
	err := os.WriteFile(dataDir+"/installer.sh", []byte(dummyData), 0755)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}
	result := strings.Replace(originalDir, "curation", "script/tmp", -1)
	dst := filepath.Join(result, "Installer")
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	srcs := strings.Replace(originalDir, "curation", "script/Installer", -1)
	CopyFile(srcs, dst)
	type args struct {
		pwd     string
		profile string
		MODE    string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test Case",
			args: args{
				pwd: originalDir,
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CreateOverlayScript(tt.args.pwd, tt.args.profile, tt.args.MODE); got == tt.want {
				t.Errorf("CreateOverlayScript() = %v, want %v", got, tt.want)
			}
		})
	}
	defer func() {
		os.Unsetenv("FIREWALL_REQ_ALLOW")
		os.Unsetenv("FIREWALL_CFG_ALLOW")
		os.Remove(dst)
		os.RemoveAll(dataDir)
		CopyFile(dst, srcs)
	}()
}

func TestCreateOverlayScript_Case1(t *testing.T) {
	os.Setenv("FIREWALL_REQ_ALLOW", `[
		{
		  "sourceIp": "000.000.0.000",
		  "ports": "00,000",
		  "ipVer": "0000",
		  "protocol": "0000"
		}
	  ]`)
	os.Setenv("FIREWALL_CFG_ALLOW", `[
		{
		  "sourceIp": "000.000.0.000",
		  "ports": "00,000",
		  "ipVer": "0000",
		  "protocol": "0000"
		}
	  ]`)

	originalDir, _ := os.Getwd()
	src := strings.Replace(originalDir, "curation", "script", -1)
	dir := src + "/Installer"
	os.MkdirAll(dir, 0755)
	dataDir := config.PVC
	os.MkdirAll(dataDir, 0755)
	dummyData := `#!/bin/bash
	enable_netipplan
        install_intel_CAcertificates
# Add your installation commands here
`
	err := os.WriteFile(dataDir+"/installer.sh", []byte(dummyData), 0755)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}
	result := strings.Replace(originalDir, "curation", "script/tmp", -1)
	dst := filepath.Join(result, "Installer")
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	srcs := strings.Replace(originalDir, "curation", "script/Installer", -1)
	CopyFile(srcs, dst)
	type args struct {
		pwd     string
		profile string
		MODE    string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test Case",
			args: args{
				pwd: originalDir,
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CreateOverlayScript(tt.args.pwd, tt.args.profile, tt.args.MODE); got == tt.want {
				t.Errorf("CreateOverlayScript() = %v, want %v", got, tt.want)
			}
		})
	}
	defer func() {
		os.Unsetenv("FIREWALL_REQ_ALLOW")
		os.Unsetenv("FIREWALL_CFG_ALLOW")
		os.RemoveAll(dst)
		os.RemoveAll(dataDir)
		CopyFile(dst, srcs)
	}()
}

func TestCreateOverlayScript_Case2(t *testing.T) {
	originalDir, _ := os.Getwd()
	src := strings.Replace(originalDir, "curation", "script", -1)
	dir := src + "/Installer"
	os.MkdirAll(dir, 0755)
	dataDir := config.PVC
	os.MkdirAll(dataDir, 0755)
	dummyData := `#!/bin/bash
	enable_netipplan
        install_intel_CAcertificates
# Add your installation commands here
`
	err := os.WriteFile(dataDir+"/installer.sh", []byte(dummyData), 0755)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}
	result := strings.Replace(originalDir, "curation", "script/tmp", -1)
	dst := filepath.Join(result, "Installer")
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	srcs := strings.Replace(originalDir, "curation", "script/Installer", -1)
	CopyFile(srcs, dst)
	type args struct {
		pwd     string
		profile string
		MODE    string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test Case",
			args: args{
				pwd: originalDir,
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CreateOverlayScript(tt.args.pwd, tt.args.profile, tt.args.MODE); got == tt.want {
				t.Errorf("CreateOverlayScript() = %v, want %v", got, tt.want)
			}
		})
	}
	defer func() {
		os.RemoveAll(dst)
		os.RemoveAll(dataDir)
		CopyFile(dst, srcs)
	}()
}

func TestCreateOverlayScript_Case4(t *testing.T) {
	os.Setenv("FIREWALL_REQ_ALLOW", `[
		{
		  "sourceIp": "000.000.0.000",
		  "ports": "00,000",
		  "ipVer": "0000",
		  "protocol": "0000"
		}
	  ]`)
	os.Setenv("FIREWALL_CFG_ALLOW", `[
		{
		  "sourceIp": "000.000.0.000",
		  "ports": "00,000",
		  "ipVer": "0000",
		  "protocol": "0000"
		}
	  ]`)

	originalDir, _ := os.Getwd()
	src := strings.Replace(originalDir, "curation", "script", -1)
	dir := src + "/Installer"
	os.MkdirAll(dir, 0755)
	dataDir := config.PVC
	os.MkdirAll(dataDir, 0755)
	dummyData := `#!/bin/bash
	enable_netipplan
        install_intel_CAcertificates
# Add your installation commands here
`
	err := os.WriteFile(dataDir+"/installer.sh", []byte(dummyData), 0755)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}
	result := strings.Replace(originalDir, "curation", "script/tmp", -1)
	dst := filepath.Join(result, "Installer")
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	srcs := strings.Replace(originalDir, "curation", "script/Installer", -1)
	CopyFile(srcs, dst)
	type args struct {
		pwd     string
		profile string
		MODE    string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test Case",
			args: args{
				pwd: originalDir,
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CreateOverlayScript(tt.args.pwd, tt.args.profile, tt.args.MODE); got == tt.want {
				t.Errorf("CreateOverlayScript() = %v, want %v", got, tt.want)
			}
		})
	}
	defer func() {
		os.Unsetenv("FIREWALL_REQ_ALLOW")
		os.Unsetenv("FIREWALL_CFG_ALLOW")
		os.RemoveAll(dst)
		os.RemoveAll(dataDir)
		CopyFile(dst, srcs)
	}()
}

func TestCreateOverlayScript_Case3(t *testing.T) {
	originalDir, _ := os.Getwd()
	src := strings.Replace(originalDir, "curation", "script", -1)
	dir := src + "/Installer"
	os.MkdirAll(dir, 0755)
	dataDir := config.PVC
	os.MkdirAll(dataDir, 0755)
	dummyData := `#!/bin/bash
	enable_netipplan
        install_intel_CAcertificates
# Add your installation commands here
`
	err := os.WriteFile(dataDir+"/installer.sh", []byte(dummyData), 0755)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}
	result := strings.Replace(originalDir, "curation", "script/tmp", -1)
	dst := filepath.Join(result, "Installer")
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	srcs := strings.Replace(originalDir, "curation", "script/Installer", -1)
	CopyFile(srcs, dst)

	path := "/etc/ssl/orch-ca-cert/ca.crt"
	err2 := os.MkdirAll("/etc/ssl/orch-ca-cert", 0755)
	if err2 != nil {
		fmt.Println("Error creating directories:", err2)
		return
	}
	file, err3 := os.Create(path)
	if err3 != nil {
		fmt.Println("Error creating file:", err3)
		return
	}
	defer file.Close()
	type args struct {
		pwd     string
		profile string
		MODE    string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test Case",
			args: args{
				pwd:  originalDir,
				MODE: "dev",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CreateOverlayScript(tt.args.pwd, tt.args.profile, tt.args.MODE); got == tt.want {
				t.Errorf("CreateOverlayScript() = %v, want %v", got, tt.want)
			}
		})
	}
	defer func() {
		os.RemoveAll(dst)
		os.RemoveAll(dataDir)
		CopyFile(dst, srcs)
		os.RemoveAll(path)
	}()
}

func TestAddProxies(t *testing.T) {
	type args struct {
		fileName  string
		newLines  []string
		beginLine string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Invalid file name",
			args: args{
				fileName: "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AddProxies(tt.args.fileName, tt.args.newLines, tt.args.beginLine)
		})
	}
}

func TestAddFirewallRules(t *testing.T) {
	type args struct {
		fileName string
		newLines []string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Invalid file name",
			args: args{
				fileName: "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AddFirewallRules(tt.args.fileName, tt.args.newLines)
		})
	}
}
