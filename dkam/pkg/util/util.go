// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"
	"os"
	"path/filepath"

	osv1 "github.com/intel/infra-core/inventory/v2/pkg/api/os/v1"
	inv_errors "github.com/intel/infra-core/inventory/v2/pkg/errors"
	"github.com/intel/infra-core/inventory/v2/pkg/logging"
	"github.com/intel/infra-onboarding/dkam/pkg/config"
)

var zlog = logging.GetLogger("DKAMUtil")

func GetLocalInstallerPath(osType osv1.OsType) (string, error) {
	switch osType {
	case osv1.OsType_OS_TYPE_MUTABLE:
		return config.ScriptPath + "/Installer", nil
	case osv1.OsType_OS_TYPE_IMMUTABLE:
		return config.ScriptPath + "/Installer.cfg", nil
	default:
		invErr := inv_errors.Errorf("Unsupported OS type %v, may result in wrong local installer path",
			osType)
		zlog.InfraSec().Error().Err(invErr).Msg("")
		return "", invErr
	}
}

// GetInstallerLocation returns a relative, standard path where OS installation artifacts are stored.
// We assume that installation artifacts are unique per OS resource as scripts may change over time for OS profiles.
// That's why we use OS resource ID to uniquely identify the installation artifact.
// NOTE1: This may lead to some duplication of files if multiple OS resources use the same installer scripts,
// we may use profile_version+profile_name (once profile_version is populated) to save space on the PV.
// NOTE2: We should make sure that installation artifacts doesn't include any tenant-specific information.
// Multiple tenants should be able to share the same installation artifacts (see NOTE1).
func GetInstallerLocation(osResource *osv1.OperatingSystemResource, rootDir string) (string, error) {
	// profileIdentifier is a unique identifier of OS profile. For now we use OS resource ID instead of
	// profile_name+profile_version to uniqely identify installation artifacts until we fully integrate profile_version.
	profileIdentifier := osResource.GetResourceId()

	installerPath := fmt.Sprintf("%s/OSArtifacts/%s/installer", rootDir, profileIdentifier)

	switch osResource.GetOsType() {
	case osv1.OsType_OS_TYPE_IMMUTABLE:
		installerPath += ".cfg"
	case osv1.OsType_OS_TYPE_MUTABLE:
		installerPath += ".sh"
	default:
		invErr := inv_errors.Errorf("Unsupported OS type %v, may result in wrong installation artifacts path",
			osResource.GetOsType())
		zlog.InfraSec().Error().Err(invErr).Msg("")
		return "", invErr
	}

	return installerPath, nil
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil // path exists
	}
	if os.IsNotExist(err) {
		return false, nil // path does not exist
	}
	return false, err // an error occurred (other than not existing)
}

func GetReleaseFilePathIfExists() (string, error) {
	releaseFilePath := filepath.Join(config.DownloadPath, "tmp", config.ReleaseVersion+".yaml")
	exists, err := PathExists(releaseFilePath)
	if err != nil {
		invErr := inv_errors.Errorf("Failed to check if path %s exists: %s", releaseFilePath, err)
		zlog.InfraSec().Err(invErr).Msg("")
		return "", invErr
	}

	if !exists {
		invErr := inv_errors.Errorf("The release file not found under path %s", releaseFilePath)
		zlog.InfraSec().Err(invErr).Msg("")
		return "", invErr
	}

	zlog.InfraSec().Debug().Msgf("Release file found under path %s", releaseFilePath)

	return releaseFilePath, nil
}
