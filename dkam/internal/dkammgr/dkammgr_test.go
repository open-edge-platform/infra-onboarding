// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package dkammgr_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-onboarding/dkam/internal/dkammgr"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/config"
	dkam_testing "github.com/open-edge-platform/infra-onboarding/dkam/testing"
)

var zlog = logging.GetLogger("DKAM-Mgr")

func TestMain(m *testing.M) {
	var err error
	config.PVC, err = os.MkdirTemp(os.TempDir(), "test_pvc")
	if err != nil {
		panic(fmt.Sprintf("Error creating temp directory: %v", err))
	}

	cleanupFunc := dkam_testing.StartTestReleaseService("profile")

	run := m.Run()
	cleanupFunc()

	os.Exit(run)
}

func TestDownloadArtifacts(t *testing.T) {
	err := dkammgr.DownloadArtifacts(context.Background())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
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
