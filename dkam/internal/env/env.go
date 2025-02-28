// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"os"

	"github.com/intel/infra-core/inventory/v2/pkg/logging"
)

const (
	envProfileScriptsRepo = "RS_PROFILE_SCRIPTS_REPO"
	envHookOSRepo         = "RS_HOOK_OS_REPO"
	envHookOSVersion      = "HOOK_OS_VERSION"
)

var (
	// TODO: pass all hook os version and bare metal agent
	//  versions via configmap or override values to dkam.
	HookOSRepo        = os.Getenv(envHookOSRepo)
	ProfileScriptRepo = os.Getenv(envProfileScriptsRepo)
	HookOSVersion     = os.Getenv(envHookOSVersion)
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
	ProfileScriptRepo = MustGetEnv(envProfileScriptsRepo)
	HookOSVersion = MustGetEnv(envHookOSVersion)
}
