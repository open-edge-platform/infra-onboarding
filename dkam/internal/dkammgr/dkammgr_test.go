// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package dkammgr_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	osv1 "github.com/intel/infra-core/inventory/v2/pkg/api/os/v1"
	"github.com/intel/infra-core/inventory/v2/pkg/logging"
	inv_testing "github.com/intel/infra-core/inventory/v2/pkg/testing"
	"github.com/intel/infra-onboarding/dkam/internal/dkammgr"
	"github.com/intel/infra-onboarding/dkam/pkg/config"
	dkam_testing "github.com/intel/infra-onboarding/dkam/testing"
)

var (
	projectRoot string
	zlog        = logging.GetLogger("DKAM-Mgr")
)

func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	config.PVC, err = os.MkdirTemp(os.TempDir(), "test_pvc")
	if err != nil {
		panic(fmt.Sprintf("Error creating temp directory: %v", err))
	}

	projectRoot = filepath.Dir(filepath.Dir(wd))
	policyPath := projectRoot + "/out"
	migrationsDir := projectRoot + "/out"

	cleanupFunc := dkam_testing.StartTestReleaseService("profile")

	inv_testing.StartTestingEnvironment(policyPath, "", migrationsDir)
	run := m.Run()
	cleanupFunc()
	inv_testing.StopTestingEnvironment()

	os.Exit(run)
}

func TestDownloadArtifacts(t *testing.T) {
	dkam_testing.PrepareTestReleaseFile(t, projectRoot)
	// Create a UploadBaseImageRequest

	err := dkammgr.DownloadArtifacts(context.Background())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGetCuratedScript(t *testing.T) {
	dkam_testing.EnableLegacyModeForTesting(t)
	dkam_testing.PrepareTestInfraConfig(t)
	dkam_testing.PrepareTestReleaseFile(t, projectRoot)
	dkam_testing.PrepareTestCaCertificateFile(t)

	dir := config.PVC
	mkdirerr := os.MkdirAll(dir, 0o755)
	if mkdirerr != nil {
		fmt.Println("Error creating dir:", mkdirerr)
	}
	mkdirerr = os.MkdirAll(config.DownloadPath, 0o755)
	if mkdirerr != nil {
		fmt.Println("Error creating dir:", mkdirerr)
	}
	currentDir, err := os.Getwd()
	if err != nil {
		zlog.InfraSec().Fatal().Err(err).Msgf("Error getting current working directory: %v", err)
		return
	}
	zlog.InfraSec().Info().Msgf("Current dir %s", currentDir)
	parentDir := filepath.Join(currentDir, "..", "..")
	config.ScriptPath = parentDir + "/pkg/script"
	dummyData := `#!/bin/bash
	enable_netipplan
        install_intel_CAcertificates
# Add your installation commands here
`
	err = os.WriteFile(dir+"/installer.sh", []byte(dummyData), 0o600)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}
	defer func() {
		os.Remove(dir + "/installer.sh")
	}()

	osr := &osv1.OperatingSystemResource{
		ProfileName: "profile",
		OsType:      osv1.OsType_OS_TYPE_MUTABLE,
	}
	err = dkammgr.GetCuratedScript(context.TODO(), osr)

	// Check if the returned filename matches the expected format
	assert.NoError(t, err)
}

func TestGetMode(t *testing.T) {
	// Save the original value of MODE so that it can be restored later
	originalMode := os.Getenv("MODE")

	// Defer the restoration of the original value
	defer func() {
		os.Setenv("MODE", originalMode)
	}()

	tests := []struct {
		name         string
		testMode     string
		expectedMode string
	}{
		{"Mode is set", "production", "production"},
		{"Mode is not set", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the test value for MODE
			t.Setenv("MODE", tt.testMode)

			result := dkammgr.GetMODE()
			if result != tt.expectedMode {
				t.Errorf("Expected %v, but got %v", tt.expectedMode, result)
			}
		})
	}
}

//nolint:dupl // this is for SignMicroOS.
func TestSignMicroOS(t *testing.T) {
	currentDir, err := os.Getwd()
	if err != nil {
		zlog.InfraSec().Fatal().Err(err).Msgf("Error getting current working directory: %v", err)
		return
	}
	zlog.InfraSec().Info().Msgf("Current dir %s", currentDir)
	parentDir := filepath.Join(currentDir, "..", "..")
	config.ScriptPath = parentDir + "/pkg/script"

	// Call the function you want to test
	result, err := dkammgr.SignMicroOS()

	// Check if the result matches the expected value
	if result != true {
		t.Errorf("Expected result to be true, got %t", result)
	}

	// Check if the error is nil
	if err != nil {
		t.Errorf("Expected error to be nil, got %v", err)
	}
}

//nolint:dupl // this is for BuildSignIpxe.
func TestBuildSignIpxe1(t *testing.T) {
	currentDir, err := os.Getwd()
	if err != nil {
		zlog.InfraSec().Fatal().Err(err).Msgf("Error getting current working directory: %v", err)
		return
	}
	zlog.InfraSec().Info().Msgf("Current dir %s", currentDir)
	parentDir := filepath.Join(currentDir, "..", "..")
	config.ScriptPath = parentDir + "/pkg/script"

	// Call the function you want to test
	result, err := dkammgr.BuildSignIpxe()

	// Check if the result matches the expected value
	if result != true {
		t.Errorf("Expected result to be true, got %t", result)
	}

	// Check if the error is nil
	if err != nil {
		t.Errorf("Expected error to be nil, got %v", err)
	}
}
