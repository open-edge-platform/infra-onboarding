// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"os"

	"github.com/intel/infra-core/inventory/v2/pkg/logging"
)

const (
	envK8sNamespace  = "MI_K8S_NAMESPACE"
	envDkamMode      = "EN_DKAMMODE"
	envUserName      = "EN_USERNAME"
	envPassWord      = "EN_PASSWORD"
	envTinkerVersion = "TINKER_VERSION"

	defaultK8sNamespace = "orch-infa"
)

var (
	ENDkamMode = os.Getenv(envDkamMode)
	ENUserName = os.Getenv(envUserName)
	ENPassWord = os.Getenv(envPassWord)

	K8sNamespace = GetEnvWithDefault(envK8sNamespace, defaultK8sNamespace)

	TinkerActionVersion = os.Getenv(envTinkerVersion)
)

var zlog = logging.GetLogger("Env")

func GetEnvWithDefault(key, defaultVal string) string {
	v, found := os.LookupEnv(key)
	if found && v != "" {
		return v
	}
	zlog.Warn().Msgf("%s env var is not set, using default image type: %s",
		key, defaultVal)
	return defaultVal
}

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
	TinkerActionVersion = MustGetEnv(envTinkerVersion)
}
