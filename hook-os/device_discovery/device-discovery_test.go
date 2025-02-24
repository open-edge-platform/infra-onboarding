// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"strconv"
	"testing"
)

var (
	requiredVars = []string{
		"onboarding_manager_svc",
		"onboarding_stream_svc",
		"OBM_PORT",
		"KEYCLOAK_URL",
	}

	optionalVars = []string{
		"EXTRA_HOSTS",
	}
)

func FuzzTestDeviceDiscoveryEnv(f *testing.F) {

	f.Add("/tmp/envfile", "/tmp/kernelfile")

	f.Fuzz(func(t *testing.T, envConfigPath string, kernelArgsFilePath string) {

		// Load environment variables from env_config
		if err := loadEnvConfig(envConfigPath); err != nil {
			t.Logf("Failed to load env_config: %v", err)
		} else {
			t.Error("Expected error to be returned")
		}

		// // Check and load the environment variables
		envVars, err := readEnvVars(requiredVars, optionalVars)
		if err != nil {
			t.Log("Error:", err)
		} else {
			t.Error("Expected error to be returned.")
		}

		_, err = strconv.Atoi(envVars["OBM_PORT"])
		if err != nil {
			t.Logf("Error converting port to integer: %v\n", err)
		} else {
			t.Error("Expected error to be returned.")
		}

		cfg, err := parseKernelArguments(kernelArgsFilePath)
		if err != nil {
			t.Logf("Error parsing kernel arguments: %v\n", err)
		} else {
			t.Error("Expected error to be returned.")
		}

		_, err = getSerialNumber()
		if err != nil {
			t.Logf("Error getting serial number: %v\n", err)
		} else {
			t.Error("Expected error to be returned.")
		}

		_, err = getUUID()
		if err != nil {
			t.Logf("Error getting uuid: %v\n", err)
		} else {
			t.Error("Expected error to be returned.")
		}

		_, err = getIPAddress(cfg.workerID)
		if err != nil {
			t.Logf("Error getting Ip address: %v\n", err)
		} else {
			t.Error("Expected error to be returned")
		}

	})

}
