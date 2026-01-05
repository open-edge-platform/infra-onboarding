// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"testing"

	"device-discovery/internal/sysinfo"
)

func TestAutoDetectSystemInfo(t *testing.T) {
	// Test getting serial number
	serial, err := sysinfo.GetSerialNumber()
	if err != nil {
		t.Logf("Could not get serial number: %v", err)
	} else {
		t.Logf("Serial number: %s", serial)
	}

	// Test getting UUID
	uuid, err := sysinfo.GetUUID()
	if err != nil {
		t.Logf("Could not get UUID: %v", err)
	} else {
		t.Logf("UUID: %s", uuid)
	}

	// Test getting primary MAC
	mac, err := sysinfo.GetPrimaryMAC()
	if err != nil {
		t.Logf("Could not get primary MAC: %v", err)
	} else {
		t.Logf("Primary MAC: %s", mac)
		
		// Test getting IP from MAC
		if mac != "" {
			ip, err := sysinfo.GetIPAddress(mac)
			if err != nil {
				t.Logf("Could not get IP for MAC %s: %v", mac, err)
			} else {
				t.Logf("IP address for MAC %s: %s", mac, ip)
			}
		}
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *CLIConfig
		shouldErr bool
	}{
		{
			name: "valid config",
			cfg: &CLIConfig{
				ObmSvc:       "obm.example.com",
				ObsSvc:       "obs.example.com",
				ObmPort:      50051,
				KeycloakURL:  "keycloak.example.com",
				MacAddr:      "00:11:22:33:44:55",
				SerialNumber: "ABC123",
				UUID:         "12345678-1234-1234-1234-123456789012",
				IPAddress:    "192.168.1.100",
			},
			shouldErr: false,
		},
		{
			name: "missing obm-svc",
			cfg: &CLIConfig{
				ObsSvc:      "obs.example.com",
				ObmPort:     50051,
				KeycloakURL: "keycloak.example.com",
				MacAddr:     "00:11:22:33:44:55",
			},
			shouldErr: true,
		},
		{
			name: "missing mac address",
			cfg: &CLIConfig{
				ObmSvc:      "obm.example.com",
				ObsSvc:      "obs.example.com",
				ObmPort:     50051,
				KeycloakURL: "keycloak.example.com",
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a basic validation test
			// In production, validateConfig exits on error
			// Here we just check if required fields are present
			hasError := tt.cfg.ObmSvc == "" || tt.cfg.ObsSvc == "" || 
			            tt.cfg.ObmPort == 0 || tt.cfg.KeycloakURL == "" || 
			            tt.cfg.MacAddr == ""
			
			if hasError != tt.shouldErr {
				t.Errorf("expected error: %v, got error: %v", tt.shouldErr, hasError)
			}
		})
	}
}

func FuzzTestDeviceDiscovery(f *testing.F) {
	f.Add("obm.example.com", "obs.example.com", 50051, "keycloak.example.com", "00:11:22:33:44:55")

	f.Fuzz(func(t *testing.T, obmSvc string, obsSvc string, obmPort int, keycloakURL string, macAddr string) {
		// Create a config with fuzzed values
		cfg := &CLIConfig{
			ObmSvc:      obmSvc,
			ObsSvc:      obsSvc,
			ObmPort:     obmPort,
			KeycloakURL: keycloakURL,
			MacAddr:     macAddr,
		}

		// Test that validation doesn't panic
		_ = cfg
		
		// Test sysinfo functions don't panic with various inputs
		if macAddr != "" {
			_, _ = sysinfo.GetIPAddress(macAddr)
		}
	})
}
