// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"os"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

const (
	envHookOSRepo    = "RS_HOOK_OS_REPO"
	envHookOSVersion = "HOOK_OS_VERSION"
)

var (
	// TODO: pass all hook os version and bare metal agent
	//  versions via configmap or override values to dkam.
	HookOSRepo    = os.Getenv(envHookOSRepo)
	HookOSVersion = os.Getenv(envHookOSVersion)
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
	HookOSRepo = MustGetEnv(envHookOSRepo)
	HookOSVersion = MustGetEnv(envHookOSVersion)
}
