// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package tinkerbell

import (
	tink "github.com/tinkerbell/tink/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewHardware(name, ns string, id, device, ip, gateway string) *tink.Hardware {
	hw := &tink.Hardware{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Hardware",
			APIVersion: "tinkerbell.org/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: tink.HardwareSpec{
			Disks: []tink.Disk{{
				Device: device,
			}},
			Metadata: &tink.HardwareMetadata{
				Facility: &tink.MetadataFacility{
					FacilityCode: "onboarding",
				},
				Instance: &tink.MetadataInstance{
					ID:       id,
					Hostname: name,
					OperatingSystem: &tink.MetadataInstanceOperatingSystem{
						Distro:  "ubuntu",
						OsSlug:  "ubuntu_20_04",
						Version: "20.04",
					},
				},
			},
			Interfaces: []tink.Interface{
				{
					Netboot: &tink.Netboot{
						AllowPXE:      &[]bool{true}[0],
						AllowWorkflow: &[]bool{true}[0],
					},
					DHCP: &tink.DHCP{
						Arch:     "x86_64",
						Hostname: name,
						IP: &tink.IP{
							Address: ip,
							Gateway: gateway,
							Netmask: "255.255.255.0",
						},
						LeaseTime:   86400,
						MAC:         id,
						NameServers: []string{"10.248.2.1", "172.30.90.4", "10.223.45.36"},
						UEFI:        true,
					},
				},
			},
		},
	}

	return hw
}
