/*
   Copyright (C) 2023 Intel Corporation
   SPDX-License-Identifier: Apache-2.0
*/

package utils

const (
	ImgTypeBkc          = "prod_bkc"
	ImgTypeJammy        = "prod_jammy"
	ImgTypeFocal        = "prod_focal"
	ImgTypeFocalMs      = "prod_focal-ms"
	DefaultProviderName = "fm_onboarding"
)

type (
	// DeviceInfo is an internal representation of host info and its metadata that is used during the onboarding process.
	DeviceInfo struct {
		// HwSerialID serial number of a host.
		HwSerialID string
		// HwMacID MAC address of the management NIC of a host.
		HwMacID string
		// HwIP IP address of the management NIC of a host.
		HwIP string
		// DiskType disk type of a host.
		DiskType string
		// OSImageURL a URL pointing to the OS location on the EN's reverse proxy.
		OSImageURL string
		// Gateway IP gateway of a local subnet where a host is located.
		Gateway string
		// InstallerScriptURL a URL pointing to the location of OS Installer script on the EN's reverse proxy.
		InstallerScriptURL string
		// Rootfspart a number of rootfs partition
		Rootfspart string
		// ClientImgName a name of the OS image used for a host
		ClientImgName string
		// ImgType an OS image type used for a host
		ImgType string
		// GUID UUID identifier of a host
		GUID string
		// SecurityFeature security flags for a host
		SecurityFeature uint32
		// FdoGUID UUID identifier of a host used by FDO (different from GUID)
		FdoGUID string
		// AuthClientID a client ID of a host used by authorization service (e.g., Keycloak)
		AuthClientID string
		// AuthClientSecret a client secret of a host used to by authorization service (e.g., Keycloak)
		AuthClientSecret string
		// TinkerVersion a version of tink-worker image to used by EN for uOS onboarding
		TinkerVersion string
		// Hostname a host name set in the OS
		Hostname string
		// CustomerID stores a unique identifier of a customer. Used for license check.
		CustomerID string
	}
)
