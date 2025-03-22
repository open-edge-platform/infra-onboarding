// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package dkammgr

import (
	"context"
	"os"

	osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/config"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/curation"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/download"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/signing"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/util"
)

var zlog = logging.GetLogger("DKAM-Mgr")

func DownloadArtifacts(ctx context.Context) error {
	MODE := GetMODE()
	zlog.InfraSec().Info().Msgf("Mode of deployment: %s", MODE)
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

func GetCuratedScript(ctx context.Context, osResource *osv1.OperatingSystemResource) error {
	scriptFileName, err := util.GetInstallerLocation(osResource, config.PVC)
	if err != nil {
		return err
	}

	installerExists, patherr := util.PathExists(scriptFileName)
	if patherr != nil {
		zlog.InfraSec().Info().Msgf("Error checking installer file path %v", patherr)
	}
	if installerExists {
		zlog.InfraSec().Info().Msg("Installer exists. Skip curation.")
	} else {
		err := curation.CurateScript(ctx, osResource)
		if err != nil {
			zlog.InfraSec().Info().Msgf("Failed curate %v", err)
			return err
		}
	}
	return nil
}

func SignMicroOS() (bool, error) {
	signed, err := signing.SignHookOS()
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

func GetMODE() string {
	return os.Getenv("MODE")
}
