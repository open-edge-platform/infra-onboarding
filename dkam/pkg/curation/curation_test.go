// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package curation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/curation"
)

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
