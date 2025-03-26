/*
   Copyright (C) 2023 Intel Corporation
   SPDX-License-Identifier: Apache-2.0
*/

//nolint:stylecheck,revive // use underscore for onboarding_types
package onboarding_types

import osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"

const (
	DefaultProviderName = "infra_onboarding"
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
		// OSImageURL a URL pointing to the OS location on the EN's reverse proxy.
		OSImageURL string
		// Gateway IP gateway of a local subnet where a host is located.
		Gateway string
		// ImgType an OS image type used for a host
		ImgType string
		// GUID UUID identifier of a host
		GUID string
		// SecurityFeature security flags for a host
		SecurityFeature osv1.SecurityFeature
		// AuthClientID a client ID of a host used by authorization service (e.g., Keycloak)
		AuthClientID string
		// AuthClientSecret a client secret of a host used to by authorization service (e.g., Keycloak)
		AuthClientSecret string
		// TinkerVersion a version of tink-worker image to used by EN for uOS onboarding
		TinkerVersion string
		// Hostname a host name set in the OS
		Hostname string
		// sha256 is used by EN to validate the SHA256 of OS image authenticity.
		OsImageSHA256 string
		// OsType differentiates between mutable and immutable OS
		OsType osv1.OsType
		// Tenant ID of resource
		TenantID       string
		PlatformBundle string
		// local account username
		LocalAccountUserName string
		// SSh key
		SSHKey string
	}
)
