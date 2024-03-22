// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package curation

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
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
	}
	for tcname, tc := range tests {
		t.Run(tcname, func(t *testing.T) {
			ufwCommand := GenerateUFWCommand(tc.ufwRule)
			assert.Equal(t, tc.expectedUfwCommand, ufwCommand)
		})
	}
}
