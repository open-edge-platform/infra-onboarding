// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package config

const (
	ServerAddress            = "serverAddress"
	ServerAddressDescription = "The endpoint address of this component to serve on. " +
		"It should have the following format <IP address>:<port>."
	Port           = "0.0.0.0:5581"
	Ubuntuversion  = "jammy"
	Arch           = "amd64"
	Release        = "prod"
	ProdFileServer = "files.edgeorch.net"
	DevFileServer  = "files-rs.internal.ledgepark.intel.com"
	ProdHarbor     = "harbor.edgeorch.net"
	DevHarbor      = "amr-registry.caas.intel.com"
	AuthServer     = "demo2.maestro.intel.com"
	ReleaseVersion = "24.03"
	GPGKey         = "ledgepark-debian-signing-key-gpg-non-prod.pem"
	PVC            = "/data/"
)
