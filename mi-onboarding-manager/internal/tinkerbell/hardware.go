// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package tinkerbell

import (
	"context"

	tink "github.com/tinkerbell/tink/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/errors"
)

func NewHardware(name, ns, id, device, ip, gateway, osResourceID string) *tink.Hardware {
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
						OsSlug:  osResourceID, // passing OS resource id
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
						LeaseTime:   leaseTime86400,
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

// TODO (LPIO-1865): We can probably optimize it.
// Instead of doing GET+CREATE we can try CREATE and check if resource already exists.
func CreateHardwareIfNotExists(ctx context.Context, k8sCli client.Client, k8sNamespace string,
	deviceInfo utils.DeviceInfo, osResourceID string,
) error {
	hwInfo := NewHardware(
		GetTinkHardwareName(deviceInfo.GUID),
		k8sNamespace,
		deviceInfo.HwMacID,
		deviceInfo.DiskType, deviceInfo.HwIP, deviceInfo.Gateway, osResourceID)

	obj := &tink.Hardware{}
	err := k8sCli.Get(ctx, client.ObjectKeyFromObject(hwInfo), obj)
	if err != nil && errors.IsNotFound(err) {
		zlog.Debug().Msgf("Creating new Tinkerbell hardware %s for host %s.", hwInfo.Name, deviceInfo.GUID)

		createErr := k8sCli.Create(ctx, hwInfo)
		if createErr != nil {
			zlog.MiSec().MiErr(err).Msgf("")
			return inv_errors.Errorf("Failed to create Tinkerbell hardware %s", hwInfo.Name)
		}

		return nil
	}

	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("")
		// some other error that may need retry
		return inv_errors.Errorf("Failed to check if Tinkerbell hardware %s exists.", hwInfo.Name)
	}

	zlog.Debug().Msgf("Tinkerbell hardware %s for host %s already exists.", hwInfo.Name, deviceInfo.GUID)

	// already exists, do not return error
	return nil
}

func DeleteHardwareForHostIfExist(ctx context.Context, k8sNamespace, hostUUID string) error {
	zlog.Debug().Msgf("Deleting DI workflow resources for host %s", hostUUID)

	kubeClient, err := K8sClientFactory()
	if err != nil {
		return err
	}

	hw := &tink.Hardware{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Hardware",
			APIVersion: "tinkerbell.org/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "machine-" + hostUUID,
			Namespace: k8sNamespace,
		},
	}

	if err = kubeClient.Delete(ctx, hw); err != nil && !errors.IsNotFound(err) {
		zlog.MiSec().MiErr(err).Msg("")
		zlog.Debug().Msgf("Failed to delete Tink hardware resources for host %s", hostUUID)
		return inv_errors.Errorf("Failed to delete Tink hardware resources for host")
	}

	return nil
}
