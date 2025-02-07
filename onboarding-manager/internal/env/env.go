// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"flag"
	"os"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/logging"
)

const (
	envHTTPProxy     = "EN_HTTP_PROXY"
	envHTTPSProxy    = "EN_HTTPS_PROXY"
	envNoProxy       = "EN_NO_PROXY"
	envNameservers   = "EN_NAMESERVERS"
	envImageType     = "IMAGE_TYPE"
	envK8sNamespace  = "MI_K8S_NAMESPACE"
	envDkamMode      = "EN_DKAMMODE"
	envUserName      = "EN_USERNAME"
	envPassWord      = "EN_PASSWORD"
	envTinkerVersion = "TINKER_VERSION"

	defaultK8sNamespace = "orch-infa"
)

var (
	ENProxyHTTP   = os.Getenv(envHTTPProxy)
	ENProxyHTTPS  = os.Getenv(envHTTPSProxy)
	ENProxyNo     = os.Getenv(envNoProxy)
	ENNameservers = os.Getenv(envNameservers)
	ENDkamMode    = os.Getenv(envDkamMode)
	ENUserName    = os.Getenv(envUserName)
	ENPassWord    = os.Getenv(envPassWord)

	K8sNamespace = GetEnvWithDefault(envK8sNamespace, defaultK8sNamespace)

	TinkerActionVersion = os.Getenv(envTinkerVersion)

	FlagEnforceCloudInit = flag.Bool("enforceCloudInit", false,
		"Set to true to always use cloud-init to provision Day0/Day1 EN configuration")
)

var zlog = logging.GetLogger("Env")

func GetEnvWithDefault(key, defaultVal string) string {
	v, found := os.LookupEnv(key)
	if found && v != "" {
		return v
	}
	zlog.Warn().Msgf("%s env var is not set, using default image type: %s",
		envImageType, defaultVal)
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
