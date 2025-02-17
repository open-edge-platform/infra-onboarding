// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tinkerbell

import (
	"fmt"
	"os"
	"strings"

	osv1 "github.com/intel/infra-core/inventory/v2/pkg/api/os/v1"
	inv_errors "github.com/intel/infra-core/inventory/v2/pkg/errors"
	"github.com/intel/infra-onboarding/dkam/pkg/config"
	"github.com/intel/infra-onboarding/onboarding-manager/internal/env"
	"github.com/intel/infra-onboarding/onboarding-manager/internal/onboardingmgr/utils"
)

const (
	ActionEraseNonRemovableDisk      = "erase-non-removable-disk" //#nosec G101 -- ignore false positive.
	ActionSecureBootStatusFlagRead   = "secure-boot-status-flag-read"
	ActionStreamUbuntuImage          = "stream-ubuntu-image"
	ActionStreamTiberOSImage         = "stream-tiberos-image"
	ActionCopySecrets                = "copy-secrets"
	ActionGrowPartitionInstallScript = "grow-partition-install-script"
	ActionInstallOpenssl             = "install-openssl"
	ActionCreateUser                 = "create-user"
	ActionEnableSSH                  = "enable-ssh"
	ActionDisableApparmor            = "disable-apparmor"
	ActionInstallScriptDownload      = "profile-pkg-and-node-agents-install-script-download"
	ActionCloudInitfileDownload      = "cloud-init-file-for-post-install-script-download"
	ActionInstallScript              = "service-script-for-profile-pkg-and-node-agents-install"
	ActionInstallScriptEnable        = "enable-service-script-for-profile-pkg-node-agents"
	ActionNetplan                    = "write-netplan"
	ActionNetplanConfigure           = "update-netplan-to-make-ip-static"
	ActionGrowPartitionService       = "service-script-for-grow-partion-installer"
	ActionGrowPartitionServiceEnable = "enable-grow-partinstall-service-script"
	ActionNetplanService             = "service-script-for-netplan-update"
	ActionNetplanServiceEnable       = "enable-update-netplan.service-script"
	ActionEfibootset                 = "efibootset-for-diskboot"
	ActionFdeEncryption              = "fde-encryption"
	ActionKernelupgrade              = "kernel-upgrade"
	ActionReboot                     = "reboot"
	ActionCopyENSecrets              = "copy-ensp-node-secrets" //#nosec G101 -- ignore false positive.
	ActionStoringAlpine              = "store-Alpine"
	ActionAddEnvProxy                = "add-env-proxy"
	ActionAddAptProxy                = "add-apt-proxy"
	ActionAddDNSNamespace            = "add-dns-namespace"
	ActionCreateSecretsDirectory     = "create-ensp-node-directory" //#nosec G101 -- ignore false positive.
	ActionWriteClientID              = "write-client-id"
	ActionWriteClientSecret          = "write-client-secret"
	ActionWriteHostname              = "write-hostname"
	ActionWriteEtcHosts              = "Write-Hosts-etc"
	ActionTenantID                   = "tenant-id"
	ActionSystemdNetworkOptimize     = "systemd-network-online-optimize"
	ActionDisableSnapdOptimize       = "systemd-snapd-disable-optimize"
	ActionTiberOSPartition           = "tiber-os-partition"
	ActionCloudinitDsidentity        = "cloud-init-ds-identity"
	ActionSetSeliuxRelabel           = "set-selinux-relabel-policy"
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
	defaultEraseNonRemovableDiskImage       = "localhost:7443/one-intel-edge/edge-node/tinker-actions/erase_non_removable_disks"

	envTinkActionSecurebootFlagReadImage     = "TINKER_SECUREBOOTFLAGREAD_IMAGE"
	defaultTinkActionSecurebootFlagReadImage = "localhost:7443/one-intel-edge/edge-node/tinker-actions/securebootflag"

	envTinkActionWriteFileImage     = "TINKER_WRITEFILE_IMAGE"
	defaultTinkActionWriteFileImage = "localhost:7443/one-intel-edge/edge-node/tinker-actions/writefile"

	envTinkActionCexecImage     = "TINKER_CEXEC_IMAGE"
	defaultTinkActionCexecImage = "localhost:7443/one-intel-edge/edge-node/tinker-actions/cexec"

	envTinkActionDiskImage     = "TINKER_DISK_IMAGE"
	defaultTinkActionDiskImage = "localhost:7443/one-intel-edge/edge-node/tinker-actions/image2disk"

	envTinkActionEfibootImage     = "TINKER_EFIBOOT_IMAGE"
	defaultTinkActionEfibootImage = "localhost:7443/one-intel-edge/edge-node/tinker-actions/efibootset"

	envTinkActionFdeImage     = "TINKER_FDE_IMAGE"
	defaultTinkActionFdeImage = "localhost:7443/one-intel-edge/edge-node/tinker-actions/fde"

	envTinkActionKerenlUpgradeImage     = "TINKER_KERNELUPGRD_IMAGE"
	defaultTinkActionKernelUpgradeImage = "localhost:7443/one-intel-edge/edge-node/tinker-actions/kernelupgrd"

	envTinkActionTiberOSPartitionImage     = "TINKER_TIBEROS_IMAGE_PARTITION"
	defaultTinkActionTiberOSPartitionImage = "localhost:7443/one-intel-edge/edge-node/tinker-actions/tiberos_partition"

	envTinkActionQemuNbdImage2DiskImage     = "TINKER_QEMU_NBD_IMAGE2DISK_IMAGE"
	defaultTinkActionQemuNbdImage2DiskImage = "localhost:7443/one-intel-edge/edge-node/tinker-actions/qemu_nbd_image2disk"

	envDkamDevMode = "dev"
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

func tinkActionTiberOSPartitionImage(tinkerImageVersion string) string {
	iv := getTinkerImageVersion(tinkerImageVersion)
	if v := os.Getenv(envTinkActionTiberOSPartitionImage); v != "" {
		return fmt.Sprintf("%s:%s", v, iv)
	}
	return fmt.Sprintf("%s:%s", defaultTinkActionTiberOSPartitionImage, iv)
}

func tinkActionQemuNbdImage2DiskImage(tinkerImageVersion string) string {
	iv := getTinkerImageVersion(tinkerImageVersion)
	if v := os.Getenv(envTinkActionQemuNbdImage2DiskImage); v != "" {
		return fmt.Sprintf("%s:%s", v, iv)
	}
	return fmt.Sprintf("%s:%s", defaultTinkActionQemuNbdImage2DiskImage, iv)
}

//nolint:funlen,cyclop // May effect the functionality, need to simplify this in future
func NewTemplateDataProdTIBEROS(name string, deviceInfo utils.DeviceInfo) ([]byte, error) {
	// #nosec G115
	securityFeatureTypeVar := osv1.SecurityFeature(deviceInfo.SecurityFeature)
	securityFeatureStr := securityFeatureTypeVar.String()

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
						"SECURITY_FEATURE_FLAG": securityFeatureStr,
					},
					Volumes: []string{
						"/:/host:rw",
					},
				},
				{
					Name:    ActionStreamTiberOSImage,
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
					Name:    ActionTiberOSPartition,
					Image:   tinkActionTiberOSPartitionImage(deviceInfo.TinkerVersion),
					Timeout: timeOutAvg560,
				},

				{
					Name:    ActionCreateUser,
					Image:   tinkActionCexecImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE": fmt.Sprintf("useradd -p $(openssl passwd -1 %s) -s /bin/bash -d /home/%s/ -m -G sudo %s",
							env.ENPassWord, env.ENUserName, env.ENUserName),
					},
				},

				{
					Name:    ActionTenantID,
					Image:   tinkActionWriteFileImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/intel_edge_node/tenantId",
						"CONTENTS":  fmt.Sprintf("TENANT_ID=%s", deviceInfo.TenantID),
						"UID":       "0",
						"GID":       "0",
						"MODE":      "0755",
						"DIRMODE":   "0755",
					},
				},

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
					Name:    ActionCreateSecretsDirectory,
					Image:   tinkActionCexecImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "mkdir -p /etc/intel_edge_node/client-credentials/",
					},
				},
				{
					Name:    ActionWriteClientID,
					Image:   tinkActionWriteFileImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/intel_edge_node/client-credentials/client_id",
						"CONTENTS":  deviceInfo.AuthClientID,
						"UID":       "0",
						"GID":       "0",
						"MODE":      "0755",
						"DIRMODE":   "0755",
					},
				},
				{
					Name:    ActionWriteClientSecret,
					Image:   tinkActionWriteFileImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/intel_edge_node/client-credentials/client_secret",
						"CONTENTS":  deviceInfo.AuthClientSecret,
						"UID":       "0",
						"GID":       "0",
						"MODE":      "0755",
						"DIRMODE":   "0755",
					},
				},

				{
					Name:    ActionCloudInitfileDownload,
					Image:   tinkActionCexecImage(deviceInfo.TinkerVersion),
					Timeout: timeOutAvg200,
					Environment: map[string]string{
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE": fmt.Sprintf("curl -o /etc/cloud/cloud.cfg.d/installer.cfg %s;"+
							"chmod +x /etc/cloud/cloud.cfg.d/installer.cfg",
							deviceInfo.InstallerScriptURL),
					},
					Pid: "host",
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
	if osv1.SecurityFeature(deviceInfo.SecurityFeature) ==
		osv1.SecurityFeature_SECURITY_FEATURE_SECURE_BOOT_AND_FULL_DISK_ENCRYPTION {
		for i, task := range wf.Tasks {
			for j, action := range task.Actions {
				if action.Name == ActionTiberOSPartition {
					// Remove the action from the slice
					wf.Tasks[i].Actions = append(wf.Tasks[i].Actions[:j], wf.Tasks[i].Actions[j+1:]...)
				}
				if action.Name == ActionSetSeliuxRelabel {
					// Remove the action from the slice
					wf.Tasks[i].Actions = append(wf.Tasks[i].Actions[:j], wf.Tasks[i].Actions[j+1:]...)
				}
			}
		}
	} else if osv1.SecurityFeature(deviceInfo.SecurityFeature) ==
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

	// Creat the User credentials only for dev mode and remove the action for production mode
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

//nolint:funlen,cyclop // May effect the functionality, need to simplify this in future
func NewTemplateDataUbuntu(name string, deviceInfo utils.DeviceInfo) ([]byte, error) {
	// #nosec G115
	securityFeatureTypeVar := osv1.SecurityFeature(deviceInfo.SecurityFeature)
	securityFeatureStr := securityFeatureTypeVar.String()

	infraConfig := config.GetInfraConfig()

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
						"SECURITY_FEATURE_FLAG": securityFeatureStr,
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

				{
					Name:    ActionAddEnvProxy,
					Image:   tinkActionWriteFileImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/environment",
						"CONTENTS": fmt.Sprintf(`
						http_proxy=%s
						https_proxy=%s
						no_proxy=%s`, infraConfig.ENProxyHTTP, infraConfig.ENProxyHTTPS, infraConfig.ENProxyNoProxy),
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0755",
						"DIRMODE": "0755",
					},
				},

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
					Name:    ActionWriteEtcHosts,
					Image:   tinkActionWriteFileImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/hosts",
						"CONTENTS": fmt.Sprintf(`
127.0.0.1 %s
            `, deviceInfo.Hostname),
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0755",
						"DIRMODE": "0755",
					},
				},
				{
					Name:    ActionAddDNSNamespace,
					Image:   tinkActionWriteFileImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/systemd/resolved.conf",
						"CONTENTS": fmt.Sprintf(`
						[Resolve]
						DNS "%s"`, strings.Join(infraConfig.DNSServers, " ")),
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0755",
						"DIRMODE": "0755",
					},
				},

				{
					Name:    ActionCreateSecretsDirectory,
					Image:   tinkActionCexecImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "mkdir -p /etc/intel_edge_node/client-credentials/",
					},
				},
				{
					Name:    ActionWriteClientID,
					Image:   tinkActionWriteFileImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/intel_edge_node/client-credentials/client_id",
						"CONTENTS":  deviceInfo.AuthClientID,
						"UID":       "0",
						"GID":       "0",
						"MODE":      "0755",
						"DIRMODE":   "0755",
					},
				},
				{
					Name:    ActionWriteClientSecret,
					Image:   tinkActionWriteFileImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/intel_edge_node/client-credentials/client_secret",
						"CONTENTS":  deviceInfo.AuthClientSecret,
						"UID":       "0",
						"GID":       "0",
						"MODE":      "0755",
						"DIRMODE":   "0755",
					},
				},

				{
					Name:    ActionCreateUser,
					Image:   tinkActionCexecImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE": fmt.Sprintf("useradd -p $(openssl passwd -1 %s) -s /bin/bash -d /home/%s/ -m -G sudo %s",
							env.ENPassWord, env.ENUserName, env.ENUserName),
					},
				},

				{
					Name:    ActionInstallScriptDownload,
					Image:   tinkActionCexecImage(deviceInfo.TinkerVersion),
					Timeout: timeOutAvg200,
					Environment: map[string]string{
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE": fmt.Sprintf("mkdir -p /home/postinstall/Setup;chown %s:%s /home/postinstall/Setup;"+
							"wget -P /home/postinstall/Setup %s; chmod 755 /home/postinstall/Setup/installer.sh",
							env.ENUserName, env.ENUserName, deviceInfo.InstallerScriptURL),
					},
					Pid: "host",
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
					Name:    ActionTenantID,
					Image:   tinkActionWriteFileImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/intel_edge_node/tenantId",
						"CONTENTS":  fmt.Sprintf("TENANT_ID=%s", deviceInfo.TenantID),
						"UID":       "0",
						"GID":       "0",
						"MODE":      "0755",
						"DIRMODE":   "0755",
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

	// flag shared with DKAM
	if *config.FlagEnforceCloudInit {
		// Find the index of the "add-dns-namespace" action
		dnsNamespaceIndex := -1
		for i, action := range wf.Tasks[0].Actions {
			if action.Name == ActionAddDNSNamespace {
				dnsNamespaceIndex = i
				break
			}
		}

		if dnsNamespaceIndex == -1 || dnsNamespaceIndex > len(wf.Tasks[0].Actions) {
			return nil, inv_errors.Errorf("action %s not found in the workflow", ActionAddDNSNamespace)
		}

		cloudInitPathForUbuntu := strings.ReplaceAll(deviceInfo.InstallerScriptURL, ".sh", ".cfg")
		cloudInitActions := []Action{
			{
				Name:    ActionCloudInitfileDownload,
				Image:   tinkActionCexecImage(deviceInfo.TinkerVersion),
				Timeout: timeOutAvg200,
				Environment: map[string]string{
					"FS_TYPE":             "ext4",
					"CHROOT":              "y",
					"DEFAULT_INTERPRETER": "/bin/sh -c",
					"CMD_LINE": fmt.Sprintf("curl -o /etc/cloud/cloud.cfg.d/installer.cfg %s",
						cloudInitPathForUbuntu),
				},
				Pid: "host",
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
		}

		// Insert the new actions after the "add-dns-namespace" action
		wf.Tasks[0].Actions = append(wf.Tasks[0].Actions[:dnsNamespaceIndex+1],
			append(cloudInitActions, wf.Tasks[0].Actions[dnsNamespaceIndex+1:]...)...)
	}

	// FDE removal if security feature flag is not set for FDE
	// #nosec G115
	if osv1.SecurityFeature(deviceInfo.SecurityFeature) !=
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
	//  Creat the User credentials only for dev mode and remove the action for production mode
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
