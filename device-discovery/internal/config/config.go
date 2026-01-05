// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	TokenFolder             = "/dev/shm"
	EnvConfigPath           = "/etc/hook/env_config"
	ExtraHostsFile          = "/etc/hosts"
	AccessTokenFile         = TokenFolder + "/idp_access_token"
	ReleaseTokenFile        = TokenFolder + "/release_token"
	KeycloakTokenURL        = "/realms/master/protocol/openid-connect/token"
	ReleaseTokenURL         = "/token"
	ClientCredentialsFolder = "/dev/shm/"
	ClientIDPath            = ClientCredentialsFolder + "/client_id"
	ClientSecretPath        = ClientCredentialsFolder + "/client_secret"
	KernelArgsFilePath      = "/proc/cmdline"
	CaCertPath              = "/etc/idp/server_cert.pem"
	ProjectIDPath           = ClientCredentialsFolder + "/project_id"
)

// UpdateHosts updates /etc/hosts with extra host mappings.
func UpdateHosts(extraHosts string) error {
	// Update hosts if they were provided
	if extraHosts != "" {
		// Replace commas with newlines and remove double quotes
		extraHostsNeeded := strings.ReplaceAll(extraHosts, ",", "\n")
		extraHostsNeeded = strings.ReplaceAll(extraHostsNeeded, "\"", "")

		// Append to /etc/hosts
		hostsFile := "/etc/hosts"
		err := os.WriteFile(hostsFile, []byte(extraHostsNeeded), os.ModeAppend|0644)
		if err != nil {
			return fmt.Errorf("error updating /etc/hosts: %w", err)
		}

		fmt.Println("Adding extra host mappings completed")
	}
	return nil
}

// LoadEnvConfig loads environment variables from a configuration file.
func LoadEnvConfig(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 && line[0] != '#' {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				os.Setenv(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			}
		}
	}
	return scanner.Err()
}

// SaveToFile writes data to the specified file path with the given permissions.
func SaveToFile(path, data string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	// Use io.Writer interface to write data
	_, err = io.WriteString(file, data)
	return err
}

// ReadEnvVars checks if all required environment variables are set and returns an error if any are missing.
func ReadEnvVars(requiredVars []string, optionalVars []string) (map[string]string, error) {
	envVars := make(map[string]string)

	// Process required environment variables
	for _, key := range requiredVars {
		value, exists := os.LookupEnv(key)
		if !exists || value == "" {
			return nil, fmt.Errorf("environment variable %s is missing", key)
		}
		envVars[key] = value
	}

	// Process optional environment variables
	for _, key := range optionalVars {
		value, exists := os.LookupEnv(key)
		if !exists || value == "" {
			continue // Skip if the optional variable doesn't exist or is empty
		}
		envVars[key] = value
	}

	return envVars, nil
}

// CreateTempScript creates a temporary script file with the given content.
func CreateTempScript(scriptContent []byte) (*os.File, error) {
	tmpfile, err := os.CreateTemp("", "client-auth.sh")
	if err != nil {
		return nil, fmt.Errorf("error creating temporary file: %w", err)
	}

	if _, err := tmpfile.Write(scriptContent); err != nil {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
		return nil, fmt.Errorf("error writing to temporary file: %w", err)
	}

	if err := tmpfile.Chmod(0700); err != nil {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
		return nil, fmt.Errorf("error setting permissions on temporary file: %w", err)
	}

	return tmpfile, nil
}
