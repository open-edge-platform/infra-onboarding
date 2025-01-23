// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package config

const (
	ReleaseVersion         = "latest-dev"
	DownloadPath           = "/tmp"
	BootsCaCertificateFile = "/etc/ssl/boots-ca-cert/ca.crt"
)

// As variable to allow changes in tests
var (
	PVC                   = "/data"
	OrchCACertificateFile = "/etc/ssl/orch-ca-cert/ca.crt"
	ScriptPath            = "/home/appuser/pkg/script"
	ENManifestRepo        = "one-intel-edge/edge-node/en/manifest"
	HookOSRepo            = "one-intel-edge/edge-node/file/provisioning-hook-os"
	ProfileScriptRepo     = "one-intel-edge/edge-node/file/profile-scripts/"
)
