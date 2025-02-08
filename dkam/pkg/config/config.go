// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"flag"
)

const (
	ReleaseVersion         = "latest-dev"
	DownloadPath           = "/tmp"
	BootsCaCertificateFile = "/etc/ssl/boots-ca-cert/ca.crt"
)

// As a variable to allow changes in tests.
var (
	PVC                   = "/data"
	OrchCACertificateFile = "/etc/ssl/orch-ca-cert/ca.crt"
	ScriptPath            = "/home/appuser/pkg/script"
	ENManifestRepo        = "one-intel-edge/edge-node/en/manifest"
	HookOSRepo            = "one-intel-edge/edge-node/file/provisioning-hook-os"
	ProfileScriptRepo     = "one-intel-edge/edge-node/file/profile-scripts/"

	FlagEnforceCloudInit = flag.Bool("enforceCloudInit", false,
		"Set to true to always use cloud-init to provision Day0/Day1 EN configuration")
)
