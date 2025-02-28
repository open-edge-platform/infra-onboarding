// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package download

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	as "github.com/intel/infra-core/inventory/v2/pkg/artifactservice"
	inv_errors "github.com/intel/infra-core/inventory/v2/pkg/errors"
	"github.com/intel/infra-core/inventory/v2/pkg/logging"
	"github.com/intel/infra-onboarding/dkam/internal/env"
	"github.com/intel/infra-onboarding/dkam/pkg/config"
	"github.com/intel/infra-onboarding/dkam/pkg/util"
)

var zlog = logging.GetLogger("InfraDKAMDownload")

const fileMode = 0o755

//nolint:revive // Keeping the function name for clarity and consistency.
func DownloadMicroOS(ctx context.Context) (bool, error) {
	zlog.Info().Msgf("Inside Download and sign artifact... %s", config.DownloadPath)
	repo := env.HookOSRepo
	hookOSVersion := env.HookOSVersion
	zlog.InfraSec().Info().Msgf("Hook OS repo URL is %s and HookOS version is %s",
		repo, hookOSVersion)
	artifacts, err := as.DownloadArtifacts(ctx, repo, hookOSVersion)
	if err != nil {
		invErr := inv_errors.Errorf("Error downloading HookOS for tag %s", hookOSVersion)
		zlog.Err(invErr).Msg("")
	}

	if artifacts != nil && len(*artifacts) > 0 {
		for _, artifact := range *artifacts {
			zlog.InfraSec().Info().Msgf("Downloading artifact %s", artifact.Name)
			filePath := config.DownloadPath + "/" + artifact.Name

			err = CreateFile(filePath, &artifact)
			if err != nil {
				zlog.InfraSec().Error().Err(err).Msg("Error writing to file")
				return false, err
			}
		}
	}

	zlog.InfraSec().Info().Msg("File downloaded")
	return true, nil
}

func CreateFile(filePath string, artifact *as.Artifact) error {
	file, fileErr := os.Create(filePath)
	if fileErr != nil {
		zlog.InfraSec().Error().Err(fileErr).Msgf("Error while creating file %v", filePath)
		return fileErr
	}
	defer file.Close()

	_, err := file.Write(artifact.Data)
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msgf("Error writing to file:%v", err)
		return err
	}
	return nil
}

func MoveFile(source, destination string) error {
	exists, patherr := util.PathExists(source)
	if patherr != nil {
		zlog.InfraSec().Error().Err(patherr).Msgf("Error checking file path %v", source)
		return patherr
	}
	//nolint:revive // Ignoring due to specific need for this structure
	if exists {
		destDir := filepath.Dir(destination)
		if err := os.MkdirAll(destDir, fileMode); err != nil {
			zlog.InfraSec().Error().Err(err).Msgf("Failed to create destination dir %s", destDir)
			return err
		}

		cmd := exec.Command("mv", source, destination)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			zlog.InfraSec().Info().Msgf("error running 'mv' command: %v", err)
			return err
		} else {
			zlog.InfraSec().Info().Msgf("File %s copied to %s", source, destination)
		}
	} else {
		zlog.InfraSec().Debug().Msgf("Source file path %s does not exist", source)
	}
	return nil
}
