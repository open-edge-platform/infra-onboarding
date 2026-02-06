// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// Package env provides functionality for onboarding management.
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
)

var (
	// ENDkamMode defines a configuration value.
	ENDkamMode = os.Getenv(envDkamMode)
	// ENUserName defines a configuration value.
	ENUserName = os.Getenv(envUserName)
	// ENPassWord defines a configuration value.
	ENPassWord = os.Getenv(envPassWord)

	// K8sNamespace defines a configuration value.
	K8sNamespace = os.Getenv(envK8sNamespace)

	// TinkerActionVersion defines a configuration value.
	TinkerActionVersion = os.Getenv(envTinkerVersion)
	// TinkerArtifactName defines a configuration value.
	TinkerArtifactName = os.Getenv(envTinkerArtifactName)
)

var zlog = logging.GetLogger("Env")

// MustGetEnv performs operations for onboarding management.
func MustGetEnv(key string) string {
	v, found := os.LookupEnv(key)
	if found && v != "" {
		zlog.Debug().Msgf("Found env var %s = %s", key, v)
		return v
	}

	zlog.Fatal().Msgf("Mandatory env var %s is not set or empty!", key)
	return ""
}

// MustEnsureRequired performs operations for onboarding management.
func MustEnsureRequired() {
	TinkerActionVersion = MustGetEnv(envTinkerVersion)
	TinkerArtifactName = MustGetEnv(envTinkerArtifactName)
	K8sNamespace = MustGetEnv(envK8sNamespace)
}
