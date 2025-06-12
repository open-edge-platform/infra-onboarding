// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"os"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

const (
	envK8sNamespace       = "DEFAULT_K8S_NAMESPACE"
	envDkamMode           = "EN_DKAMMODE"
	envUserName           = "EN_USERNAME"
	envPassWord           = "EN_PASSWORD"
	envTinkerVersion      = "TINKER_VERSION"
	envTinkerArtifactName = "TINKER_ARTIFACT_NAME"
	envVenPartitionSupport = "VEN_PARTITION_SUPPORT"
)

var (
	ENDkamMode = os.Getenv(envDkamMode)
	ENUserName = os.Getenv(envUserName)
	ENPassWord = os.Getenv(envPassWord)

	K8sNamespace = os.Getenv(envK8sNamespace)

	TinkerActionVersion = os.Getenv(envTinkerVersion)
	TinkerArtifactName  = os.Getenv(envTinkerArtifactName)
	VenPartitionSupport = os.Getenv(envVenPartitionSupport)
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
	TinkerActionVersion = MustGetEnv(envTinkerVersion)
	TinkerArtifactName = MustGetEnv(envTinkerArtifactName)
	K8sNamespace = MustGetEnv(envK8sNamespace)
}
