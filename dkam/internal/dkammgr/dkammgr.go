// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package dkammgr

import (
	"context"
	"os"

	osv1 "github.com/intel/infra-core/inventory/v2/pkg/api/os/v1"
	"github.com/intel/infra-core/inventory/v2/pkg/logging"
	"github.com/intel/infra-onboarding/dkam/pkg/config"
	"github.com/intel/infra-onboarding/dkam/pkg/curation"
	"github.com/intel/infra-onboarding/dkam/pkg/download"
	"github.com/intel/infra-onboarding/dkam/pkg/signing"
	"github.com/intel/infra-onboarding/dkam/pkg/util"
)

var (
	zlog = logging.GetLogger("DKAM-Mgr")
	file string
)

func DownloadArtifacts(ctx context.Context) error {
	MODE := GetMODE()
	zlog.InfraSec().Info().Msgf("Mode of deployment: %s", MODE)
	zlog.InfraSec().Info().Msgf("Manifest Tag: %s", config.GetInfraConfig().ENManifestTag)

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

func DownloadOS(ctx context.Context, osRes *osv1.OperatingSystemResource) error {
	zlog.Info().Msgf("Inside DownloadOS...")

	if osRes.GetOsProvider() != osv1.OsProviderKind_OS_PROVIDER_KIND_INFRA {
		zlog.Debug().Msgf("Skipping OS download for %s due to OS provider kind: %s",
			osRes.GetResourceId(), osRes.GetOsProvider().String())
		return nil
	}
	if osRes.GetOsType() == osv1.OsType_OS_TYPE_IMMUTABLE {
		zlog.Debug().Msgf("Skipping OS download for OS type: %s", osRes.GetOsType())
		return nil
	}

	imageURL := osRes.GetImageUrl()
	zlog.Info().Msgf("imageURL %s", imageURL)
	targetDir := config.PVC

	zlog.Info().Msgf("Download Ubuntu OS")

	file = util.GetOSImageLocation(osRes, targetDir)
	// Check if the compressed raw image file already exists
	if _, err := os.Stat(file); os.IsNotExist(err) {
		// Download the image
		if err := download.DownloadUbuntuImage(ctx, osRes, targetDir); err != nil {
			zlog.InfraSec().Error().Err(err).Msgf("Error downloading image:%v", err)
			return err
		}
	} else {
		zlog.InfraSec().Info().Msgf("Compressed raw image file already exists: %s", file)
	}

	zlog.InfraSec().Info().Msg("OS Image downloaded and move to PVC")
	return nil
}
