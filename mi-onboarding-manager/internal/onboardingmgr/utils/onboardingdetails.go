/*
   Copyright (C) 2023 Intel Corporation
   SPDX-License-Identifier: Apache-2.0
*/

package utils

type (
	DeviceInfo struct {
		HwSerialID        string
		HwMacID           string
		HwIP              string
		DiskType          string
		LoadBalancerIP    string
		Gateway           string
		RootfspartNo      string
		Rootfspart        string
		ClientImgName     string
		ProvisionerIP     string
		ImType            string
		DpsScopeID        string
		DpsRegistrationID string
		DpsSymmKey        string
		GUID              string
		SecurityFeature   uint32
		FdoGUID           string
		ClientID          string
		ClientSecret      string
		FdoOwnerDNS       string
		FdoMfgDNS         string
		FdoOwnerPort      string
		FdoMfgPort        string
		FdoRvPort         string
	}

	ArtifactData struct {
		BkcURL        string
		BkcBasePkgURL string
	}
	Groupinfo struct {
		Group   string
		Version string
	}
	CustomerInfo struct {
		DpsScopeID        string
		DpsRegistrationID string
		DpsSymmKey        string
	}
)

const (
	ProdBkc     = "prod_bkc"
	ProdJammy   = "prod_jammy"
	ProdFocal   = "prod_focal"
	ProdFocalMs = "prod_focal-ms"

	ImgTypeBkc     = "bkc"
	ImgTypeJammy   = "jammy"
	ImgTypeFocal   = "focal"
	ImgTypeFocalMs = "focal-ms"
)
