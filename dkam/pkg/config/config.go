// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package config

const (
	ServerAddress            = "serverAddress"
	ServerAddressDescription = "The endpoint address of this component to serve on. " +
		"It should have the following format <IP address>:<port>."
	Port                   = "0.0.0.0:5581"
	AuthServer             = "integration-dev.maestro.intel.com"
	ReleaseVersion         = "latest-dev"
	DownloadPath           = "/tmp"
	BootsCaCertificateFile = "/etc/ssl/boots-ca-cert/ca.crt"
	TiberOSImage           = "tiberos.raw.xz"
)

// As variable to allow changes in tests
var (
	PVC                    = "/data"
	OrchCACertificateFile  = "/etc/ssl/orch-ca-cert/ca.crt"
	ScriptPath             = "/home/appuser/pkg/script"
	ENManifestRepo         = "one-intel-edge/edge-node/en/manifest"
	HookOSRepo             = "one-intel-edge/edge-node/file/provisioning-hook-os"
	ProfileScriptRepo      = "one-intel-edge/edge-node/file/profile-scripts/"
	RSProxyTiberOSManifest = "http://rs-proxy-files.rs-proxy.svc.cluster.local:8081/"
)
