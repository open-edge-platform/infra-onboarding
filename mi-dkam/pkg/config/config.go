// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package config

const (
	ServerAddress            = "serverAddress"
	ServerAddressDescription = "The endpoint address of this component to serve on. " +
		"It should have the following format <IP address>:<port>."
	Port            = "0.0.0.0:5581"
	Ubuntuversion   = "jammy"
	Arch            = "amd64"
	Release         = "prod"
	ProdHarbor      = "harbor.edgeorch.net"
	DevHarbor       = "amr-registry.caas.intel.com"
	AuthServer      = "integration-dev.maestro.intel.com"
	ReleaseVersion  = "latest-dev"
	PVC             = "/data/"
	Tag             = "manifest"
	PreintTag       = "pre-int/manifest"
	Artifact        = "one-intel-edge/edgenode/en/manifest"
	ImageUrl        = "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img"
	ImageFileName   = "jammy-server-cloudimg-amd64.raw.gz"
	RSProxy         = "http://rs-proxy.rs-proxy.svc.cluster.local:8081/v2/one-intel-edge/edge-node/file/provisioning-hook-os/"
	RSProxyManifest = "http://rs-proxy.rs-proxy.svc.cluster.local:8081/v2/one-intel-edge/edge-node/en/"
)
