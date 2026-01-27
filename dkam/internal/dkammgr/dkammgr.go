// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// Package dkammgr provides functionality for downloading, signing, and managing kernel artifacts.
package dkammgr

import (
	"context"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/config"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/download"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/signing"
)

var zlog = logging.GetLogger("DKAM-Mgr")

// DownloadArtifacts downloads all required artifacts from the release service.
func DownloadArtifacts(ctx context.Context) error {
	zlog.InfraSec().Info().Msgf("Manifest Tag: %s", config.GetInfraConfig().ENAgentManifestTag)

	zlog.InfraSec().Info().Msg("Download artifacts")

	downloaded, downloadErr := download.DownloadMicroOS(ctx)
	if downloadErr != nil {
		zlog.InfraSec().Info().Msgf("Failed to download MicroOS %v", downloadErr)
		return downloadErr
	}
	if downloaded {
		zlog.InfraSec().Info().Msg("Downloaded successfully")
	}

	return nil
}

// SignMicroOS signs the MicroOS kernel image.
func SignMicroOS() (bool, error) {
	signed, err := signing.SignMicroOS()
	if err != nil {
		zlog.InfraSec().Info().Msgf("Failed to sign MicroOS %v", err)
		return false, err
	}
	if signed {
		zlog.InfraSec().Info().Msgf("Signed MicroOS and moved to PVC")
	}

	return true, nil
}

// BuildSignIpxe builds and signs the iPXE boot loader.
func BuildSignIpxe() (bool, error) {
	signed, err := signing.BuildSignIpxe()
	if err != nil {
		zlog.InfraSec().Info().Msgf("Failed to build and sign iPXE %v", err)
		return false, err
	}
	if signed {
		zlog.InfraSec().Info().Msgf("Build, Signed iPXE and moved to PVC")
	}
	return true, nil
}
