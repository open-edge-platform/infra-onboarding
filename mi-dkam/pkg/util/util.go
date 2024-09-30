// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package util

import (
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/os/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/logging"
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
		fileName += ".raw.xz" // We (EIM) control TiberOS extensions so it will always be .raw.xz
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
