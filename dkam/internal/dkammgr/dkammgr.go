// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// Package dkammgr provides functionality for downloading, signing, and managing kernel artifacts.
package dkammgr

import (
	"context"
	"os"
	"path/filepath"

	osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/config"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/download"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/script/vpro"
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

// CurateVProInstaller curates vPro installer script for Ubuntu and copies it to PVC
func CurateVProInstaller() error {
	infraConfig := config.GetInfraConfig()

	zlog.InfraSec().Info().Msg("Curating vPro installer for Ubuntu")

	curatedScript, err := vpro.CurateVProInstaller(infraConfig, osv1.OsType_OS_TYPE_MUTABLE)
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msg("Failed to curate vPro installer")
		return err
	}

	// Write to PVC (/data)
	destPath := filepath.Join(config.PVC, "Installer")
	err = os.WriteFile(destPath, []byte(curatedScript), 0o755)
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msgf("Failed to write vPro installer to %s", destPath)
		return err
	}

	zlog.InfraSec().Info().Msgf("Successfully curated and copied vPro installer to %s", destPath)

	// Restrict access: allow only wget from /data/Installer, deny other folders in /data
	installerDir := filepath.Dir(destPath)
	if err := os.Chmod(installerDir, 0o755); err != nil {
		zlog.InfraSec().Error().Err(err).Msgf("Failed to set permissions on %s", installerDir)
	}
	// Deny access to other folders in /data except 'Installer'
	entries, err := os.ReadDir(config.PVC)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() && entry.Name() != "Installer" {
				denyPath := filepath.Join(config.PVC, entry.Name())
				if err := os.Chmod(denyPath, 0); err != nil {
					zlog.InfraSec().Error().Err(err).Msgf("Failed to restrict access to %s", denyPath)
				}
			}
		}
	}

	return nil
}
