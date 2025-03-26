// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package curation

import (
	"io/fs"
	"os"
	"path/filepath"

	osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/config"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/curation"
)

var zlog = logging.GetLogger("LocalCuration")

const (
	fileMode  = 0o755
	writeMode = 0o600
)

func CurateLegacyScript() error {
	targetInstallerScriptPath := config.PVC + "/Installer.sh"
	localInstallerScript := config.ScriptPath + "/Installer"

	templateVariables, err := curation.GetCommonInfraTemplateVariables(config.GetInfraConfig(), osv1.OsType_OS_TYPE_MUTABLE)
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msg("Failed to get template variables for curation")
		return err
	}

	tmplScript, err := os.ReadFile(localInstallerScript)
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msgf(
			"Failed to read template of installation script from path %v", localInstallerScript)
		return err
	}

	curatedScriptData, createErr := curation.CurateFromTemplate(string(tmplScript), templateVariables)
	if createErr != nil {
		zlog.InfraSec().Error().Msgf("Error checking path %v", createErr)
		return createErr
	}

	writeErr := writeFileToPath(targetInstallerScriptPath, []byte(curatedScriptData))
	if writeErr != nil {
		zlog.InfraSec().Error().Err(writeErr).Msgf("Failed to write file to path %s", targetInstallerScriptPath)
		return writeErr
	}

	zlog.Info().Msg("Legacy installer script written to PV")

	return nil
}

func writeFileToPath(filePath string, content []byte) error {
	zlog.Debug().Msgf("Writing data to path %s", filePath)

	err := os.MkdirAll(filepath.Dir(filePath), fs.FileMode(fileMode))
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msg("")
		return inv_errors.Errorf("Failed to create sub-directories to save file")
	}

	err = os.WriteFile(filePath, content, fs.FileMode(writeMode))
	if err != nil {
		errMsg := "Failed save the data to output path"
		zlog.Error().Err(err).Msg(errMsg)
		return inv_errors.Errorf("%s", errMsg)
	}

	return nil
}
