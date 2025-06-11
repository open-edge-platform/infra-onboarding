// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"os"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

const (
	envUOSRepo                 = "UOS_REPO"
	envUOSVersion              = "UOS_VERSION"
	envNameRsFilesProxyAddress = "RSPROXY_FILES_ADDRESS"
)

var (
	// TODO: pass all hook os version and bare metal agent
	//  versions via configmap or override values to dkam.
	UOSRepo        = os.Getenv(envUOSRepo)
	UOSVersion     = os.Getenv(envUOSVersion)
	RSProxyAddress = os.Getenv(envNameRsFilesProxyAddress)
)

var zlog = logging.GetLogger("Env")

func MustGetEnv(key string) string {
	v, found := os.LookupEnv(key)
	if found && v != "" {
		zlog.Debug().Msgf("Found env var %s = %s", key, v)
		return v
	}

	zlog.Fatal().Msgf("Mandatory env var %s is not set or empty!", key)
	return ""
}

func MustEnsureRequired() {
	UOSRepo = MustGetEnv(envUOSRepo)
	UOSVersion = MustGetEnv(envUOSVersion)
	RSProxyAddress = MustGetEnv(envNameRsFilesProxyAddress)
}
