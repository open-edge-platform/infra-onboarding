// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tinkerbell

import (
	"fmt"
	"os"
	"strings"

	osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/config"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/env"
	onboarding_types "github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/onboarding/types"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/cloudinit"
	platformbundleubuntu2204 "github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/platformbundle/ubuntu-22.04"
)

const (
	ActionEraseNonRemovableDisk            = "erase-non-removable-disk" //#nosec G101 -- ignore false positive.
	ActionSecureBootStatusFlagRead         = "secure-boot-status-flag-read"
	ActionStreamUbuntuImage                = "stream-ubuntu-image"
	ActionStreamEdgeMicrovisorToolKitImage = "stream-edge-microvisor-toolkit-image"
	ActionGrowPartitionInstallScript       = "grow-partition-install-script"
	ActionCreateUser                       = "create-user"
	ActionInstallScriptDownload            = "profile-pkg-and-node-agents-install-script-download"
	ActionCloudInitInstall                 = "install-cloud-init"
	ActionInstallScript                    = "service-script-for-profile-pkg-and-node-agents-install"
	ActionInstallScriptEnable              = "enable-service-script-for-profile-pkg-node-agents"
	ActionNetplan                          = "write-netplan"
	ActionNetplanConfigure                 = "update-netplan-to-make-ip-static"
	ActionNetplanService                   = "service-script-for-netplan-update"
	ActionNetplanServiceEnable             = "enable-update-netplan.service-script"
	ActionEfibootset                       = "efibootset-for-diskboot"
	ActionFdeEncryption                    = "fde-encryption"
	ActionKernelupgrade                    = "kernel-upgrade"
	ActionReboot                           = "reboot"
	ActionAddAptProxy                      = "add-apt-proxy"
	ActionCreateSecretsDirectory           = "create-node-directory" //#nosec G101 -- ignore false positive.
	ActionWriteClientID                    = "write-client-id"
	ActionWriteClientSecret                = "write-client-secret"
	ActionWriteHostname                    = "write-hostname"
	ActionSystemdNetworkOptimize           = "systemd-network-online-optimize"
	ActionDisableSnapdOptimize             = "systemd-snapd-disable-optimize"
	ActionEMTPartition                     = "emt-partition"
	ActionCloudinitDsidentity              = "cloud-init-ds-identity"
	ActionSetSeliuxRelabel                 = "set-selinux-relabel-policy"
)

const (
	timeOutMax9800 = 9800
	timeOutMax8000 = 8000
	timeOutAvg560  = 560
	timeOutAvg200  = 200
	timeOutMin90   = 90
	timeOutMin30   = 30
	leaseTime86400 = 86400

	envTinkerImageVersion     = "TINKER_IMAGE_VERSION"
	defaultTinkerImageVersion = "v1.0.0"

	envTinkActionEraseNonRemovableDiskImage = "TINKER_ERASE_NON_REMOVABLE_DISK_IMAGE"

	envTinkActionSecurebootFlagReadImage = "TINKER_SECUREBOOTFLAGREAD_IMAGE"

	envTinkActionWriteFileImage = "TINKER_WRITEFILE_IMAGE"

	envTinkActionCexecImage = "TINKER_CEXEC_IMAGE"

	envTinkActionDiskImage = "TINKER_DISK_IMAGE"

	envTinkActionEfibootImage = "TINKER_EFIBOOT_IMAGE"

	envTinkActionFdeImage = "TINKER_FDE_IMAGE"

	envTinkActionKerenlUpgradeImage = "TINKER_KERNELUPGRD_IMAGE"

	envTinkActionEMTPartitionImage = "TINKER_EMT_IMAGE_PARTITION"

	envTinkActionQemuNbdImage2DiskImage = "TINKER_QEMU_NBD_IMAGE2DISK_IMAGE"

	envDkamDevMode = "dev"

	tinkerActionEraseNonRemovableDisks = "erase_non_removable_disks"
	tinkerActionCexec                  = "cexec"
	tinkerActionFDE                    = "fde"
	tinkerActionEMTPartition           = "emt_partition"
	tinkerActionQemuNbdImage2Disk      = "qemu_nbd_image2disk"
	tinkerActionKernelUpgrade          = "kernelupgrd"
	tinkerActionEfibootset             = "efibootset"
	tinkerActionImage2Disk             = "image2disk"
	tinkerActionWritefile              = "writefile"
	tinkerActionSecurebootflag         = "securebootflag"
)

var (
	defaultEraseNonRemovableDiskImage        = getTinkerActionImage(tinkerActionEraseNonRemovableDisks)
	defaultTinkActionSecurebootFlagReadImage = getTinkerActionImage(tinkerActionSecurebootflag)
	defaultTinkActionWriteFileImage          = getTinkerActionImage(tinkerActionWritefile)
	defaultTinkActionCexecImage              = getTinkerActionImage(tinkerActionCexec)
	defaultTinkActionDiskImage               = getTinkerActionImage(tinkerActionImage2Disk)
	defaultTinkActionEfibootImage            = getTinkerActionImage(tinkerActionEfibootset)
	defaultTinkActionFdeImage                = getTinkerActionImage(tinkerActionFDE)
	defaultTinkActionKernelUpgradeImage      = getTinkerActionImage(tinkerActionKernelUpgrade)
	defaultTinkActionEMTPartitionImage       = getTinkerActionImage(tinkerActionEMTPartition)
	defaultTinkActionQemuNbdImage2DiskImage  = getTinkerActionImage(tinkerActionQemuNbdImage2Disk)
)

// if `tinkerImageVersion` is non-empty, its value is returned,
// then it tries to retrieve value from envar, otherwise default value is returned.
func getTinkerImageVersion(tinkerImageVersion string) string {
	if tinkerImageVersion != "" {
		return tinkerImageVersion
	}
	if v := os.Getenv(envTinkerImageVersion); v != "" {
		return v
	}
	return defaultTinkerImageVersion
}

func getTinkerActionImage(imageName string) string {
	return fmt.Sprintf("localhost:7443/%s/%s", env.TinkerArtifactName, imageName)
}

func tinkActionEraseNonRemovableDisk(tinkerImageVersion string) string {
	iv := getTinkerImageVersion(tinkerImageVersion)
	if v := os.Getenv(envTinkActionEraseNonRemovableDiskImage); v != "" {
		return fmt.Sprintf("%s:%s", v, iv)
	}
	return fmt.Sprintf("%s:%s", defaultEraseNonRemovableDiskImage, iv)
}

func tinkActionSecurebootFlagReadImage(tinkerImageVersion string) string {
	iv := getTinkerImageVersion(tinkerImageVersion)
	if v := os.Getenv(envTinkActionSecurebootFlagReadImage); v != "" {
		return fmt.Sprintf("%s:%s", v, iv)
	}
	return fmt.Sprintf("%s:%s", defaultTinkActionSecurebootFlagReadImage, iv)
}

func tinkActionWriteFileImage(tinkerImageVersion string) string {
	iv := getTinkerImageVersion(tinkerImageVersion)
	if v := os.Getenv(envTinkActionWriteFileImage); v != "" {
		return fmt.Sprintf("%s:%s", v, iv)
	}
	return fmt.Sprintf("%s:%s", defaultTinkActionWriteFileImage, iv)
}

func tinkActionCexecImage(tinkerImageVersion string) string {
	iv := getTinkerImageVersion(tinkerImageVersion)
	if v := os.Getenv(envTinkActionCexecImage); v != "" {
		return fmt.Sprintf("%s:%s", v, iv)
	}
	return fmt.Sprintf("%s:%s", defaultTinkActionCexecImage, iv)
}

func tinkActionDiskImage(tinkerImageVersion string) string {
	iv := getTinkerImageVersion(tinkerImageVersion)
	if v := os.Getenv(envTinkActionDiskImage); v != "" {
		return fmt.Sprintf("%s:%s", v, iv)
	}
	return fmt.Sprintf("%s:%s", defaultTinkActionDiskImage, iv)
}

func tinkActionEfibootImage(tinkerImageVersion string) string {
	iv := getTinkerImageVersion(tinkerImageVersion)
	if v := os.Getenv(envTinkActionEfibootImage); v != "" {
		return fmt.Sprintf("%s:%s", v, iv)
	}
	return fmt.Sprintf("%s:%s", defaultTinkActionEfibootImage, iv)
}

func tinkActionFdeImage(tinkerImageVersion string) string {
	iv := getTinkerImageVersion(tinkerImageVersion)
	if v := os.Getenv(envTinkActionFdeImage); v != "" {
		return fmt.Sprintf("%s:%s", v, iv)
	}
	return fmt.Sprintf("%s:%s", defaultTinkActionFdeImage, iv)
}

func tinkActionKernelupgradeImage(tinkerImageVersion string) string {
	iv := getTinkerImageVersion(tinkerImageVersion)
	if v := os.Getenv(envTinkActionKerenlUpgradeImage); v != "" {
		return fmt.Sprintf("%s:%s", v, iv)
	}
	return fmt.Sprintf("%s:%s", defaultTinkActionKernelUpgradeImage, iv)
}

func tinkActionEMTPartitionImage(tinkerImageVersion string) string {
	iv := getTinkerImageVersion(tinkerImageVersion)
	if v := os.Getenv(envTinkActionEMTPartitionImage); v != "" {
		return fmt.Sprintf("%s:%s", v, iv)
	}
	return fmt.Sprintf("%s:%s", defaultTinkActionEMTPartitionImage, iv)
}

func tinkActionQemuNbdImage2DiskImage(tinkerImageVersion string) string {
	iv := getTinkerImageVersion(tinkerImageVersion)
	if v := os.Getenv(envTinkActionQemuNbdImage2DiskImage); v != "" {
		return fmt.Sprintf("%s:%s", v, iv)
	}
	return fmt.Sprintf("%s:%s", defaultTinkActionQemuNbdImage2DiskImage, iv)
}

//nolint:funlen,cyclop // May effect the functionality, need to simplify this in future
func NewTemplateDataProdEdgeMicrovisorToolkit(name string, deviceInfo onboarding_types.DeviceInfo) ([]byte, error) {
	infraConfig := config.GetInfraConfig()
	opts := []cloudinit.Option{
		cloudinit.WithOSType(deviceInfo.OsType),
		cloudinit.WithTenantID(deviceInfo.TenantID),
		cloudinit.WithHostname(deviceInfo.Hostname),
		cloudinit.WithClientCredentials(deviceInfo.AuthClientID, deviceInfo.AuthClientSecret),
	}

	if env.ENDkamMode == envDkamDevMode {
		opts = append(opts, cloudinit.WithDevMode(env.ENUserName, env.ENPassWord))
	}

	cloudInitData, err := cloudinit.GenerateFromInfraConfig(infraConfig, opts...)
	if err != nil {
		return nil, err
	}

	wf := Workflow{
		Version:       "0.1",
		Name:          name,
		GlobalTimeout: timeOutMax9800,
		Tasks: []Task{{
			Name:       "os-installation",
			WorkerAddr: "{{.device_1}}",
			Volumes: []string{
				"/dev:/dev",
				"/dev/console:/dev/console",
				"/lib/firmware:/lib/firmware:ro",
			},
			Actions: []Action{
				{
					Name:    ActionSecureBootStatusFlagRead,
					Image:   tinkActionSecurebootFlagReadImage(deviceInfo.TinkerVersion),
					Timeout: timeOutAvg560,
					Environment: map[string]string{
						"SECURITY_FEATURE_FLAG": deviceInfo.SecurityFeature.String(),
					},
					Volumes: []string{
						"/:/host:rw",
					},
				},
				{
					Name:    ActionStreamEdgeMicrovisorToolKitImage,
					Image:   tinkActionDiskImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMax9800,
					Environment: map[string]string{
						"IMG_URL":    deviceInfo.OSImageURL,
						"COMPRESSED": "true",
						"SHA256":     deviceInfo.OsImageSHA256,
					},
					Pid: "host",
				},

				{
					Name:    ActionEMTPartition,
					Image:   tinkActionEMTPartitionImage(deviceInfo.TinkerVersion),
					Timeout: timeOutAvg560,
				},
				// TODO: remove write hostname actions once fixed in EMT image
				{
					Name:    ActionWriteHostname,
					Image:   tinkActionWriteFileImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/hostname",
						"CONTENTS": fmt.Sprintf(`
%s`, deviceInfo.Hostname),
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0755",
						"DIRMODE": "0755",
					},
				},
				{
					Name:    ActionCloudInitInstall,
					Image:   tinkActionWriteFileImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/cloud/cloud.cfg.d/infra.cfg",
						"CONTENTS":  cloudInitData,
						"UID":       "0",
						"GID":       "0",
						"MODE":      "0755",
						"DIRMODE":   "0755",
					},
				},

				{
					Name:    ActionCloudinitDsidentity,
					Image:   tinkActionWriteFileImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/cloud/ds-identify.cfg",
						"CONTENTS":  `datasource: NoCloud`,
						"UID":       "0",
						"GID":       "0",
						"MODE":      "0600",
						"DIRMODE":   "0700",
					},
				},

				{
					Name:    ActionFdeEncryption,
					Image:   tinkActionFdeImage(deviceInfo.TinkerVersion),
					Timeout: timeOutAvg560,
				},

				{
					Name:    ActionEfibootset,
					Image:   tinkActionEfibootImage(deviceInfo.TinkerVersion),
					Timeout: timeOutAvg560,
				},

				{
					Name:    ActionSetSeliuxRelabel,
					Image:   tinkActionCexecImage(deviceInfo.TinkerVersion),
					Timeout: timeOutAvg200,
					Environment: map[string]string{
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /",
					},
				},

				{
					Name:    ActionReboot,
					Image:   "public.ecr.aws/l0g8r8j6/tinkerbell/hub/reboot-action:latest",
					Timeout: timeOutMin90,
					Volumes: []string{
						"/worker:/worker",
					},
				},
			},
		}},
	}

	// FDE removal if security feature flag is not set for FDE
	// #nosec G115
	if deviceInfo.SecurityFeature ==
		osv1.SecurityFeature_SECURITY_FEATURE_SECURE_BOOT_AND_FULL_DISK_ENCRYPTION {
		for i, task := range wf.Tasks {
			for j, action := range task.Actions {
				if action.Name == ActionEMTPartition {
					// Remove the action from the slice
					wf.Tasks[i].Actions = append(wf.Tasks[i].Actions[:j], wf.Tasks[i].Actions[j+1:]...)
				}
				if action.Name == ActionSetSeliuxRelabel {
					// Remove the action from the slice
					wf.Tasks[i].Actions = append(wf.Tasks[i].Actions[:j], wf.Tasks[i].Actions[j+1:]...)
				}
			}
		}
	} else if deviceInfo.SecurityFeature ==
		osv1.SecurityFeature_SECURITY_FEATURE_NONE {
		for i, task := range wf.Tasks {
			for j, action := range task.Actions {
				if action.Name == ActionFdeEncryption {
					// Remove the action from the slice
					wf.Tasks[i].Actions = append(wf.Tasks[i].Actions[:j], wf.Tasks[i].Actions[j+1:]...)
					break
				}
			}
		}
	}

	// Create the User credentials only for dev mode and remove the action for production mode
	if env.ENDkamMode != envDkamDevMode {
		for i, task := range wf.Tasks {
			for j, action := range task.Actions {
				if action.Name == ActionCreateUser {
					// Remove the create user  from the slice
					wf.Tasks[i].Actions = append(wf.Tasks[i].Actions[:j], wf.Tasks[i].Actions[j+1:]...)
					break
				}
			}
		}
	}
	return marshalWorkflow(&wf)
}

//nolint:funlen // May effect the functionality, need to simplify this in future
func NewTemplateDataUbuntu(name string, deviceInfo onboarding_types.DeviceInfo) ([]byte, error) {
	infraConfig := config.GetInfraConfig()
	opts := []cloudinit.Option{
		cloudinit.WithOSType(deviceInfo.OsType),
		cloudinit.WithTenantID(deviceInfo.TenantID),
		cloudinit.WithHostname(deviceInfo.Hostname),
		cloudinit.WithClientCredentials(deviceInfo.AuthClientID, deviceInfo.AuthClientSecret),
	}

	if env.ENDkamMode == envDkamDevMode {
		opts = append(opts, cloudinit.WithDevMode(env.ENUserName, env.ENPassWord))
	}

	cloudInitData, err := cloudinit.GenerateFromInfraConfig(infraConfig, opts...)
	if err != nil {
		return nil, err
	}

	wf := Workflow{
		Version:       "0.1",
		Name:          name,
		GlobalTimeout: timeOutMax9800,
		Tasks: []Task{{
			Name:       "os-installation",
			WorkerAddr: "{{.device_1}}",
			Volumes: []string{
				"/dev:/dev",
				"/dev/console:/dev/console",
				"/lib/firmware:/lib/firmware:ro",
			},
			Actions: []Action{
				{
					Name:    ActionSecureBootStatusFlagRead,
					Image:   tinkActionSecurebootFlagReadImage(deviceInfo.TinkerVersion),
					Timeout: timeOutAvg560,
					Environment: map[string]string{
						"SECURITY_FEATURE_FLAG": deviceInfo.SecurityFeature.String(),
					},
					Volumes: []string{
						"/:/host:rw",
					},
				},

				{
					Name:    ActionEraseNonRemovableDisk,
					Image:   tinkActionEraseNonRemovableDisk(deviceInfo.TinkerVersion),
					Timeout: timeOutAvg560,
				},

				{
					Name:    ActionStreamUbuntuImage,
					Image:   tinkActionQemuNbdImage2DiskImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMax9800,
					Environment: map[string]string{
						"IMG_URL":     deviceInfo.OSImageURL,
						"SHA256":      deviceInfo.OsImageSHA256,
						"HTTP_PROXY":  infraConfig.ENProxyHTTP,
						"HTTPS_PROXY": infraConfig.ENProxyHTTPS,
						"NO_PROXY":    infraConfig.ENProxyNoProxy,
					},
					Pid: "host",
				},

				// TODO: Required for kernel-upgrd, we should find a way to pass env variable to kernel-upgrd action directly
				{
					Name:    ActionAddAptProxy,
					Image:   tinkActionWriteFileImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/apt/apt.conf",
						"CONTENTS": fmt.Sprintf(`
						Acquire::http::Proxy "%s";
						Acquire::https::Proxy "%s";`, infraConfig.ENProxyHTTP, infraConfig.ENProxyHTTPS),
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0755",
						"DIRMODE": "0755",
					},
				},
				{
					Name:    ActionCloudInitInstall,
					Image:   tinkActionWriteFileImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/cloud/cloud.cfg.d/infra.cfg",
						"CONTENTS":  cloudInitData,
						"UID":       "0",
						"GID":       "0",
						"MODE":      "0755",
						"DIRMODE":   "0755",
					},
				},
				{
					Name:    ActionCloudinitDsidentity,
					Image:   tinkActionWriteFileImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/cloud/ds-identify.cfg",
						"CONTENTS":  `datasource: NoCloud`,
						"UID":       "0",
						"GID":       "0",
						"MODE":      "0600",
						"DIRMODE":   "0700",
					},
				},
				{
					Name:    ActionInstallScriptDownload,
					Image:   tinkActionWriteFileImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/home/postinstall/Setup/installer.sh",
						"CONTENTS":  platformbundleubuntu2204.Installer,
						"UID":       "0",
						"GID":       "0",
						"MODE":      "0755",
						"DIRMODE":   "0755",
					},
				},
				{
					Name:    ActionInstallScript,
					Image:   tinkActionWriteFileImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/systemd/system/install-profile-pkgs-and-node-agent.service",
						"CONTENTS": `
						[Unit]
						Description=Profile and node agents Package Installation
						After=update-netplan.service getty@tty1.service
						ConditionPathExists = !/home/postinstall/Setup/.base_pkg_install_done
		
						[Service]
						ExecStartPre=/bin/sleep 10 
						WorkingDirectory=/home/postinstall/Setup
						ExecStart=/home/postinstall/Setup/installer.sh
						StandardOutput=tty
						StandardError=tty
						TTYPath=/dev/tty1
						Restart=always
		
						[Install]
						WantedBy=multi-user.target`,
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0644",
						"DIRMODE": "0755",
					},
				},
				{
					Name:    ActionInstallScriptEnable,
					Image:   tinkActionCexecImage(deviceInfo.TinkerVersion),
					Timeout: timeOutAvg200,
					Environment: map[string]string{
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "systemctl enable install-profile-pkgs-and-node-agent.service",
					},
				},
				{
					Name:    ActionNetplan,
					Image:   tinkActionWriteFileImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/netplan/config.yaml",
						"CONTENTS": `network:
                  version: 2
                  renderer: networkd
                  ethernets:
                    id0:
                      match:
                        name: en*
                      dhcp4: yes`,
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0644",
						"DIRMODE": "0755",
					},
				},

				{
					Name:    ActionNetplanConfigure,
					Image:   tinkActionWriteFileImage(deviceInfo.TinkerVersion),
					Timeout: timeOutAvg200,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/home/postinstall/Setup/update_netplan_config.sh",
						"CONTENTS": fmt.Sprintf(`#!/bin/bash
while [ 1 ]
do
interface=$(ip route show default | awk '/default/ {print $5}')
gateway=$(ip route show default | awk '/default/ {print $3}')
sub_net=$(ip addr show | grep $interface | grep -E 'inet ./*' | awk '{print $2}' | awk -F'/' '{print $2}')
if [ -z $interface ] || [ -z $gateway ] || [ -z $sub_net ]; then
   sleep 2
   continue
else
   break
fi
done
# Define the network configuration in YAML format with variables
config_yaml="
network:
  version: 2
  renderer: networkd
  ethernets:
    id0:
      match:
        name: en*
      dhcp4: no
      addresses: [ %s/$sub_net ]
      gateway4: $gateway
      nameservers:
        addresses: [ %s ]
"
# Write the YAML configuration to the file
echo "$config_yaml" | tee /etc/netplan/config.yaml
ln -sf /run/systemd/resolve/stub-resolv.conf /etc/resolv.conf
touch .netplan_update_done
netplan apply`, deviceInfo.HwIP, strings.Join(infraConfig.DNSServers, ", ")),
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0755",
						"DIRMODE": "0755",
					},
				},
				{
					Name:    ActionNetplanService,
					Image:   tinkActionWriteFileImage(deviceInfo.TinkerVersion),
					Timeout: timeOutAvg200,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/systemd/system/update-netplan.service",
						"CONTENTS": `
                                                [Unit]
                                                Description=update the netplan with to make static ip
                                                After=network.target
                                                ConditionPathExists = !/home/postinstall/Setup/.netplan_update_done

                                                [Service]
                                                WorkingDirectory=/home/postinstall/Setup
                                                ExecStart=/home/postinstall/Setup/update_netplan_config.sh

                                                [Install]
                                                WantedBy=multi-user.target`,
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0644",
						"DIRMODE": "0755",
					},
				},

				{
					Name:    ActionNetplanServiceEnable,
					Image:   tinkActionCexecImage(deviceInfo.TinkerVersion),
					Timeout: timeOutAvg200,
					Environment: map[string]string{
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "systemctl enable update-netplan.service",
					},
				},
				{
					Name:    ActionSystemdNetworkOptimize,
					Image:   tinkActionCexecImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE": "sed -i 's|ExecStart=/lib/systemd/systemd-networkd-wait-online|ExecStart=" +
							"/lib/systemd/systemd-networkd-wait-online --timeout=5|' " +
							"/usr/lib/systemd/system/systemd-networkd-wait-online.service",
					},
				},
				{
					Name:    ActionDisableSnapdOptimize,
					Image:   tinkActionCexecImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "systemctl disable snapd.seeded.service",
					},
				},

				{
					Name:    ActionFdeEncryption,
					Image:   tinkActionFdeImage(deviceInfo.TinkerVersion),
					Timeout: timeOutAvg560,
				},
				{
					Name:    ActionKernelupgrade,
					Image:   tinkActionKernelupgradeImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMax9800,
				},
				{
					Name:    ActionEfibootset,
					Image:   tinkActionEfibootImage(deviceInfo.TinkerVersion),
					Timeout: timeOutAvg560,
				},

				{
					Name:    ActionReboot,
					Image:   "public.ecr.aws/l0g8r8j6/tinkerbell/hub/reboot-action:latest",
					Timeout: timeOutMin90,
					Volumes: []string{
						"/worker:/worker",
					},
				},
			},
		}},
	}

	// FDE removal if security feature flag is not set for FDE
	// #nosec G115
	if deviceInfo.SecurityFeature !=
		osv1.SecurityFeature_SECURITY_FEATURE_SECURE_BOOT_AND_FULL_DISK_ENCRYPTION {
		for i, task := range wf.Tasks {
			for j, action := range task.Actions {
				if action.Name == ActionFdeEncryption {
					// Remove the action from the slice
					wf.Tasks[i].Actions = append(wf.Tasks[i].Actions[:j], wf.Tasks[i].Actions[j+1:]...)
					break
				}
			}
		}
	}

	return marshalWorkflow(&wf)
}
