// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package dkammgr

import (
	"context"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/config"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/download"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/signing"
)

var zlog = logging.GetLogger("DKAM-Mgr")

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

func SignMicroOS() (bool, error) {
	signed, err := signing.SignUOS()
	if err != nil {
		zlog.InfraSec().Info().Msgf("Failed to sign MicroOS %v", err)
		return false, err
	}
	if signed {
		zlog.InfraSec().Info().Msgf("Signed MicroOS and moved to PVC")
	}

	return true, nil
}

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
