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
		ProvisionerIp     string
		ImType            string
		DpsScopeId        string
		DpsRegistrationId string
		DpsSymmKey        string
		Guid              string
	}

	ArtifactData struct {
		BkcUrl        string
		BkcBasePkgUrl string
	}
	Groupinfo struct {
		Group     string
		Version   string
		namespace string
	}
	CustomerInfo struct {
		DpsScopeId        string
		DpsRegistrationId string
		DpsSymmKey        string
	}
)
