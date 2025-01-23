// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package util

import (
	"fmt"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/pkg/config"
	"os"
	"path/filepath"

	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/errors"

	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/api/os/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/logging"
)

var zlog = logging.GetLogger("DKAMUtil")

// getOSImageLocation generates a relative, standard path based on user inputs.
// fileName should also include file extension (e.g., someFileName.raw.gz).
func getOSImageLocation(os *osv1.OperatingSystemResource, rootDir string, fileName string) string {
	return rootDir + "/OSImage/" + os.GetSha256() + "/" + fileName
}

// GetOSImageLocation generates a relative, standard path where OS image is stored.
// File name is generated based on the OperatingSystemResource.
func GetOSImageLocation(os *osv1.OperatingSystemResource, rootDir string) string {
	// Immutable OS images will be downloaded directly from RS, so return the same path.
	// FIXME: commented out for now, because we still use PV for immutable OS images in M2 demo.
	//if os.GetOsType() == osv1.OsType_OS_TYPE_IMMUTABLE {
	//	return os.GetRepoUrl()
	//}

	fileName := os.GetProfileName()

	switch os.GetOsType() {
	case osv1.OsType_OS_TYPE_IMMUTABLE:
		zlog.MiSec().Info().Msgf("OS image URL: %v", rootDir+os.GetImageUrl())
		return rootDir + os.GetImageUrl()
	case osv1.OsType_OS_TYPE_MUTABLE:
		fileName += ".raw.gz"
	default:
		zlog.MiSec().Error().Msgf("Unsupported OS type %v, may result in wrong OS image path", os.GetOsType())
	}

	return getOSImageLocation(os, rootDir, fileName)
}

// GetOSImageLocationWithCustomFilename returns a relative, standard path where OS image is stored, using a custom filename,
func GetOSImageLocationWithCustomFilename(os *osv1.OperatingSystemResource, rootDir string, fileName string) string {
	return getOSImageLocation(os, rootDir, fileName)
}

// GetInstallerLocation returns a relative, standard path where OS installation artifacts are stored.
// We assume that installation artifacts are unique per OS resource as scripts may change over time for OS profiles.
// That's why we use OS resource ID to uniquely identify the installation artifact.
// NOTE1: This may lead to some duplication of files if multiple OS resources use the same installer scripts,
// we may use profile_version+profile_name (once profile_version is populated) to save space on the PV.
// NOTE2: We should make sure that installation artifacts doesn't include any tenant-specific information.
// Multiple tenants should be able to share the same installation artifacts (see NOTE1).
func GetInstallerLocation(os *osv1.OperatingSystemResource, rootDir string) (string, error) {
	// profileIdentifier is a unique identifier of OS profile. For now we use OS resource ID instead of
	// profile_name+profile_version to uniqely identify installation artifacts until we fully integrate profile_version.
	profileIdentifier := os.GetResourceId()

	installerPath := fmt.Sprintf("%s/OSArtifacts/%s/installer", rootDir, profileIdentifier)

	switch os.GetOsType() {
	case osv1.OsType_OS_TYPE_IMMUTABLE:
		installerPath += ".cfg"
	case osv1.OsType_OS_TYPE_MUTABLE:
		installerPath += ".sh"
	default:
		invErr := inv_errors.Errorf("Unsupported OS type %v, may result in wrong installation artifacts path",
			os.GetOsType())
		zlog.MiSec().Error().Err(invErr).Msg("")
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
		zlog.MiSec().Err(invErr).Msg("")
		return "", invErr
	}

	if !exists {
		invErr := inv_errors.Errorf("The release file not found under path %s", releaseFilePath)
		zlog.MiSec().Err(invErr).Msg("")
		return "", invErr
	}

	zlog.MiSec().Debug().Msgf("Release file found under path %s", releaseFilePath)

	return releaseFilePath, nil
}
