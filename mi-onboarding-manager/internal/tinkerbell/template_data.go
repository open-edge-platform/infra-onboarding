// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package tinkerbell

import (
	"fmt"
	"os"
	"strings"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/env"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/os/v1"
)

const (
	ActionSecureBootStatusFlagRead   = "secure-boot-status-flag-read"
	ActionStreamUbuntuImage          = "stream-ubuntu-image"
	ActionCopySecrets                = "copy-secrets"
	ActionGrowPartitionInstallScript = "grow-partition-install-script"
	ActionInstallOpenssl             = "install-openssl"
	ActionCreateUser                 = "create-user"
	ActionEnableSSH                  = "enable-ssh"
	ActionDisableApparmor            = "disable-apparmor"
	ActionInstallScriptDownload      = "profile-pkg-and-node-agents-install-script-download"
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
	ActionReboot                     = "reboot"
	ActionCopyENSecrets              = "copy-ensp-node-secrets" //nolint:gosec // hardcoded secrets need to handle in future.
	ActionStoringAlpine              = "store-Alpine"
	ActionRunFDO                     = "run-fdo"
	ActionAddEnvProxy                = "add-env-proxy"
	ActionAddAptProxy                = "add-apt-proxy"
	ActionAddDNSNamespace            = "add-dns-namespace"
	ActionCreateSecretsDirectory     = "create-ensp-node-directory" //nolint:gosec // hardcoded secrets need to handle in future.
	ActionWriteClientID              = "write-client-id"
	ActionWriteClientSecret          = "write-client-secret"
	ActionWriteHostname              = "write-hostname"
	ActionWriteEtcHosts              = "Write-Hosts-etc"
)

const (
	hardWareDesk   = "{{ index .Hardware.Disks 0 }}"
	timeOutMax9800 = 9800
	timeOutMax8000 = 8000
	timeOutAvg560  = 560
	timeOutAvg200  = 200
	timeOutMin90   = 90
	timeOutMin30   = 30
	leaseTime86400 = 86400

	envTinkerImageVersion     = "TINKER_IMAGE_VERSION"
	defaultTinkerImageVersion = "v1.0.0"

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

	envTinkActionCredcopyImage     = "TINKER_CREDCOPY_IMAGE"                                            // #nosec G101
	defaultTinkActionCredcopyImage = "localhost:7443/one-intel-edge/edge-node/tinker-actions/cred_copy" // #nosec G101
	envDkamDevMode                 = "dev"
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

func tinkActionCredcopyImage(tinkerImageVersion string) string {
	iv := getTinkerImageVersion(tinkerImageVersion)
	if v := os.Getenv(envTinkActionCredcopyImage); v != "" {
		return fmt.Sprintf("%s:%s", v, iv)
	}
	return fmt.Sprintf("%s:%s", defaultTinkActionCredcopyImage, iv)
}

//nolint:funlen // May effect the functionality, need to simplify this in future
func NewTemplateDataProd(name, rootPart, rootPartNo, hostIP, tinkerVersion string) ([]byte, error) {
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
					Name:    ActionStreamUbuntuImage,
					Image:   tinkActionDiskImage(tinkerVersion),
					Timeout: timeOutMax9800,
					Environment: map[string]string{
						"DEST_DISK":  hardWareDesk,
						"IMG_URL":    fmt.Sprintf("http://%s:8080/focal-server-cloudimg-amd64.raw.gz", hostIP),
						"COMPRESSED": "true",
					},
				},

				{
					Name:    ActionCopySecrets,
					Image:   env.ProvisionerIP + ":5015/cred_copy:latest",
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"BLOCK_DEVICE": hardWareDesk + rootPart,
						"FS_TYPE":      "ext4",
					},
				},
				{
					Name:    ActionGrowPartitionInstallScript,
					Image:   tinkActionWriteFileImage(tinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"DEST_DISK": hardWareDesk + rootPart,
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/usr/local/bin/grow_part.sh",
						"CONTENTS": fmt.Sprintf(`#!/bin/bash
growpart {{ index .Hardware.Disks 0 }} %s
resize2fs {{ index .Hardware.Disks 0 }}%s
touch /usr/local/bin/.grow_part_done`, rootPartNo, rootPart),
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0755",
						"DIRMODE": "0755",
					},
				},
				{
					Name:    ActionInstallOpenssl,
					Image:   tinkActionCexecImage(tinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        hardWareDesk + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "apt -y update && apt -y install openssl",
					},
				},
				{
					Name:    ActionCreateUser,
					Image:   tinkActionCexecImage(tinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        hardWareDesk + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "useradd -p $(openssl passwd -1 tink) -s /bin/bash -d /home/tink/ -m -G sudo tink",
					},
				},
				{
					Name:    ActionEnableSSH,
					Image:   tinkActionCexecImage(tinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        hardWareDesk + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE": "ssh-keygen -A; systemctl enable ssh.service; " +
							"sed -i 's/^PasswordAuthentication no/#PasswordAuthentication yes/g' /etc/ssh/sshd_config",
					},
				},
				{
					Name:    ActionDisableApparmor,
					Image:   tinkActionCexecImage(tinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        hardWareDesk + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "systemctl disable apparmor; systemctl disable snapd",
					},
				},
				{
					Name:    ActionNetplan,
					Image:   tinkActionWriteFileImage(tinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"DEST_DISK": hardWareDesk + rootPart,
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/netplan/config.yaml",
						"CONTENTS": `
                network:
                  version: 2
                  renderer: networkd
                  ethernets:
                    id0:
                      match:
                        name: en*
                      dhcp4: true`,
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0644",
						"DIRMODE": "0755",
					},
				},
				{
					Name:    ActionGrowPartitionService,
					Image:   tinkActionWriteFileImage(tinkerVersion),
					Timeout: timeOutAvg200,
					Environment: map[string]string{
						"DEST_DISK": hardWareDesk + rootPart,
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/systemd/system/install-grow-part.service",
						"CONTENTS": `
                                                [Unit]
                                                Description=disk size grow installer
                                                After=network.target
                                                ConditionPathExists = !/usr/local/bin/.grow_part_done

                                                [Service]
                                                ExecStartPre=/bin/sleep 30
                                                WorkingDirectory=/usr/local/bin
                                                ExecStart=/usr/local/bin/grow_part.sh

                                                [Install]
                                                WantedBy=multi-user.target`,
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0644",
						"DIRMODE": "0755",
					},
				},
				{
					Name:    ActionGrowPartitionServiceEnable,
					Image:   tinkActionCexecImage(tinkerVersion),
					Timeout: timeOutAvg200,
					Environment: map[string]string{
						"BLOCK_DEVICE":        hardWareDesk + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "systemctl enable install-grow-part.service",
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

	return marshalWorkflow(&wf)
}

//nolint:funlen,cyclop // May effect the functionality, need to simplify this in future
func NewTemplateDataProdBKC(name string, deviceInfo utils.DeviceInfo, enableDI bool) ([]byte, error) {
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
				},
				{
					Name:    ActionStreamUbuntuImage,
					Image:   tinkActionDiskImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMax9800,
					Environment: map[string]string{
						"IMG_URL":    deviceInfo.OSImageURL,
						"COMPRESSED": "true",
					},
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
						no_proxy=%s`, env.ENProxyHTTP, env.ENProxyHTTPS, env.ENProxyNo),
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
						Acquire::https::Proxy "%s";`, env.ENProxyHTTP, env.ENProxyHTTPS),
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
						DNS "%s"`, env.ENNameservers),
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0755",
						"DIRMODE": "0755",
					},
				},

				{
					Name:    ActionGrowPartitionInstallScript,
					Image:   tinkActionWriteFileImage(deviceInfo.TinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/usr/local/bin/grow_part.sh",
						"CONTENTS": `#!/bin/bash
growpart $DEST_DISK 1 
resize2fs $DEST_DISK$ID 
touch /usr/local/bin/.grow_part_done`,
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0755",
						"DIRMODE": "0755",
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
						"SCRIPT_URL":          deviceInfo.InstallerScriptURL,
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE": fmt.Sprintf("mkdir -p /home/postinstall/Setup;chown user:user /home/postinstall/Setup;"+
							"wget -P /home/postinstall/Setup %s; chmod 755 /home/postinstall/Setup/installer.sh",
							deviceInfo.InstallerScriptURL),
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
						After=update-netplan.service
						ConditionPathExists = !/home/postinstall/Setup/.base_pkg_install_done
		
						[Service]
						ExecStartPre=/bin/sleep 20 
						WorkingDirectory=/home/postinstall/Setup
						ExecStart=/home/postinstall/Setup/installer.sh
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
interface=$(ip route show default | awk '/default/ {print $5}')
gateway=$(ip route show default | awk '/default/ {print $3}')
sub_net=$(ip addr show | grep $interface | grep -E 'inet ./*' | awk '{print $2}' | awk -F'/' '{print $2}')
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
        addresses: [%s]
"
# Write the YAML configuration to the file
echo "$config_yaml" | tee /etc/netplan/config.yaml
ln -sf /run/systemd/resolve/stub-resolv.conf /etc/resolv.conf
touch .netplan_update_done
netplan apply`, deviceInfo.HwIP, strings.ReplaceAll(env.ENNameservers, " ", ", ")),
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0755",
						"DIRMODE": "0755",
					},
				},

				{
					Name:    ActionGrowPartitionService,
					Image:   tinkActionWriteFileImage(deviceInfo.TinkerVersion),
					Timeout: timeOutAvg200,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/systemd/system/install-grow-part.service",
						"CONTENTS": `
						[Unit]
						Description=disk size grow installer
                				After=network.target
                				ConditionPathExists = !/usr/local/bin/.grow_part_done

                				[Service]
                				ExecStartPre=/bin/sleep 30
                				WorkingDirectory=/usr/local/bin
                				ExecStart=/usr/local/bin/grow_part.sh
		
						[Install]
						WantedBy=multi-user.target`,
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0644",
						"DIRMODE": "0755",
					},
				},
				{
					Name:    ActionGrowPartitionServiceEnable,
					Image:   tinkActionCexecImage(deviceInfo.TinkerVersion),
					Timeout: timeOutAvg200,
					Environment: map[string]string{
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "systemctl disable install-grow-part.service",
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
                                                ExecStartPre=/bin/sleep 60 
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

	if !enableDI {
		// Di not enable
		directoryActions := []Action{
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
		}

		// Find the index of the "grow-partition-install-script" action
		var growPartitionIndex int
		for i, action := range wf.Tasks[0].Actions {
			if action.Name == ActionGrowPartitionInstallScript {
				growPartitionIndex = i
				break
			}
		}

		// Insert the new actions after the "grow-partition-install-script" action
		wf.Tasks[0].Actions = append(wf.Tasks[0].Actions[:growPartitionIndex+1],
			append(directoryActions, wf.Tasks[0].Actions[growPartitionIndex+1:]...)...)
	} else {
		// Di is enabled
		directoryActions := []Action{
			{
				Name:    ActionCopyENSecrets,
				Image:   tinkActionCredcopyImage(deviceInfo.TinkerVersion),
				Timeout: timeOutMin90,
				Environment: map[string]string{
					"OS_DST_DIR": "/etc/intel_edge_node/client-credentials/",
					"FS_TYPE":    "ext4",
				},
			},
		}

		// Find the index of the "stream-ubuntu-image" action
		var streamubuntuimage int
		for i, action := range wf.Tasks[0].Actions {
			if action.Name == ActionStreamUbuntuImage {
				streamubuntuimage = i
				break
			}
		}

		// Insert the new actions after the "grow-partition-install-script" action
		wf.Tasks[0].Actions = append(wf.Tasks[0].Actions[:streamubuntuimage+1],
			append(directoryActions, wf.Tasks[0].Actions[streamubuntuimage+1:]...)...)
	}

	// FDE removal if security feature flag is not set for FDE
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
		// Enable the grow partition
		for _, task := range wf.Tasks {
			for _, action := range task.Actions {
				if action.Name == ActionGrowPartitionServiceEnable {
					if action.Environment["CMD_LINE"] == "systemctl disable install-grow-part.service" {
						action.Environment["CMD_LINE"] = "systemctl enable install-grow-part.service"
					}
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

//nolint:funlen // May effect the functionality, need to simplify this in future
func NewTemplateDataProdMS(name, rootPart, _, hostIP, clientIP, gateway, mac, tinkerVersion string) ([]byte, error) {
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
					Name:    "stream-ubuntu-image",
					Image:   tinkActionDiskImage(tinkerVersion),
					Timeout: timeOutMax9800,
					Environment: map[string]string{
						"DEST_DISK":  hardWareDesk,
						"IMG_URL":    fmt.Sprintf("http://%s:8080/focal-server-cloudimg-amd64.raw.gz", hostIP),
						"COMPRESSED": "true",
					},
				},
				{
					Name:    "copy-secrets",
					Image:   env.ProvisionerIP + ":5015/cred_copy:latest",
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"BLOCK_DEVICE": hardWareDesk + rootPart,
						"FS_TYPE":      "ext4",
					},
				},
				{
					Name:    ActionGrowPartitionInstallScript,
					Image:   tinkActionWriteFileImage(tinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"DEST_DISK": hardWareDesk + rootPart,
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/usr/local/bin/grow_part.sh",
						"CONTENTS": fmt.Sprintf(`#!/bin/bash
growpart {{ index .Hardware.Disks 0 }} %s
resize2fs {{ index .Hardware.Disks 0 }}%s
touch /usr/local/bin/.grow_part_done`, rootPart, rootPart),
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0755",
						"DIRMODE": "0755",
					},
				},
				{
					Name:    "add-env-proxies",
					Image:   tinkActionWriteFileImage(tinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"DEST_DISK": hardWareDesk + rootPart,
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/environment",
						"CONTENTS": `
						http_proxy=http://proxy-dmz.intel.com:911
						https_proxy=http://proxy-dmz.intel.com:912
						no_proxy=localhost,127.0.0.1,.intel.com`,
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0644",
						"DIRMODE": "0755",
					},
				},
				{
					Name:    "create-docker-proxy-directory",
					Image:   tinkActionWriteFileImage(tinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        hardWareDesk + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE": "mkdir /etc/systemd/system/docker.service.d/;" +
							" touch /etc/systemd/system/docker.service.d/proxy.conf;touch /etc/apt/apt.conf",
					},
				},
				{
					Name:    "add-docker-proxies",
					Image:   tinkActionWriteFileImage(tinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"DEST_DISK": hardWareDesk + rootPart,
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/systemd/system/docker.service.d/proxy.conf",
						"CONTENTS": `
						[Service]
						Environment="HTTP_PROXY=http://proxy-dmz.intel.com:912"
						Environment="HTTPS_PROXY=http://proxy-dmz.intel.com:912"
						Environment="NO_PROXY=localhost,127.0.0.1,.intel.com"`,
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0644",
						"DIRMODE": "0755",
					},
				},
				{
					Name:    "install-openssl",
					Image:   tinkActionCexecImage(tinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        hardWareDesk + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "apt -y update && apt -y install wget openssl",
					},
				},
				{
					Name:    "create-user",
					Image:   tinkActionCexecImage(tinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        hardWareDesk + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "useradd -p $(openssl passwd -1 user) -s /bin/bash -d /home/user/ -m -G sudo user",
					},
				},
				{
					Name:    "hookos-bootmenu-delete-script",
					Image:   tinkActionWriteFileImage(tinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"DEST_DISK": hardWareDesk + rootPart,
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/tmp/hook_part_del.sh",
						"CONTENTS": `#!/bin/bash
while IFS= read -r boot_part_number; do
sudo efibootmgr -b $boot_part_number -B
done < <(efibootmgr | grep -i hookos | awk '{print $1}'| cut -c 5-8 )`,
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0755",
						"DIRMODE": "0755",
					},
				},
				{
					Name:    "executing-del-hookos-from-boot-menu",
					Image:   tinkActionCexecImage(tinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        hardWareDesk + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "/tmp/hook_part_del.sh",
					},
				},
				{
					Name:    "enable-ssh",
					Image:   tinkActionCexecImage(tinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        hardWareDesk + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE": "ssh-keygen -A; systemctl enable ssh.service;" +
							" sed -i 's/^PasswordAuthentication no/#PasswordAuthentication yes/g' /etc/ssh/sshd_config",
					},
				},
				{
					Name:    "write-netplan",
					Image:   tinkActionWriteFileImage(tinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"DEST_DISK": hardWareDesk + rootPart,
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/netplan/config.yaml",
						"CONTENTS": fmt.Sprintf(`
                network:
                  version: 2
                  renderer: networkd
                  ethernets:
                    id0:
                      match:
                        name: en*
		      dhcp4: no
		      addresses: [%s/24]
		      gateway4: %s
		      nameservers:
		        addresses: [ 10.248.2.1,172.30.90.4,10.223.45.36]`, clientIP, gateway),
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0644",
						"DIRMODE": "0755",
					},
				},
				{
					Name:    "download-kernel-deb-files",
					Image:   tinkActionCexecImage(tinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        hardWareDesk + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"KER_HEADER_URL":      fmt.Sprintf("http://%s:8080/linux-headers-5.15.96-lts.deb", hostIP),
						"KER_IMG_URL":         fmt.Sprintf("http://%s:8080/linux-image-5.15.96-lts.deb", hostIP),
						"CMD_LINE": fmt.Sprintf("mkdir -p /home/user/Setup;chown user:user /home/user/Setup;"+
							"wget -P /home/user/Setup http://%s:8080/linux-headers-5.15.96-lts.deb;"+
							" wget -P /home/user/Setup http://%s:8080/linux-image-5.15.96-lts.deb",
							hostIP, hostIP),
					},
				},
				{
					Name:    "install-kernel",
					Image:   tinkActionCexecImage(tinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        hardWareDesk + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "cd /home/user/Setup && dpkg -i *.deb",
					},
				},
				{
					Name:    "download-azure-scripts",
					Image:   tinkActionCexecImage(tinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        hardWareDesk + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"AZR_ENV":             fmt.Sprintf("http://%s:8080/azure-credentials.env_%s", hostIP, mac),
						"AZR_LOG_FILE":        fmt.Sprintf("http://%s:8080/log.sh", hostIP),
						"AZR_INSTLR_FILE":     fmt.Sprintf("http://%s:8080/azure_dps_installer.sh", hostIP),
						"CMD_LINE": fmt.Sprintf("mkdir -p /home/user/Setup/.creds;"+
							"wget -P /home/user/Setup/.creds http://%s:8080/azure-credentials.env_%s;"+
							" wget -P /home/user/Setup http://%s:8080/log.sh;"+
							"  wget -P /home/user/Setup http://%s:8080/azure_dps_installer.sh;chmod 755  /home/user/Setup/*;"+
							" cd /home/user/Setup/.creds; mv azure-credentials.env_%s azure-credentials.env",
							hostIP, mac, hostIP, hostIP, mac),
					},
				},
				{
					Name:    "service-script-for-azure-dps-installer",
					Image:   tinkActionWriteFileImage(tinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"DEST_DISK": hardWareDesk + rootPart,
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/systemd/system/install-azure-dps.service",
						"CONTENTS": `
						[Unit]
						Description=Azure DPS installer
						After=network.target
						ConditionPathExists = !/home/user/Setup/.azure_dps_setp_done
		
						[Service]
						ExecStartPre=/bin/sleep 70 
						WorkingDirectory=/home/user/Setup
						ExecStart=bash -E /home/user/Setup/azure_dps_installer.sh -e  .creds/azure-credentials.env
		
						[Install]
						WantedBy=multi-user.target`,
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0644",
						"DIRMODE": "0755",
					},
				},
				{
					Name:    "enable-service-script",
					Image:   tinkActionCexecImage(tinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        hardWareDesk + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "systemctl enable install-azure-dps.service",
					},
				},
				{
					Name:    "service-script-for-grow-partion-installer",
					Image:   tinkActionWriteFileImage(tinkerVersion),
					Timeout: timeOutAvg200,
					Environment: map[string]string{
						"DEST_DISK": hardWareDesk + rootPart,
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/systemd/system/install-grow-part.service",
						"CONTENTS": `
                                                [Unit]
                                                Description=disk size grow installer
                                                After=network.target
                                                ConditionPathExists = !/usr/local/bin/.grow_part_done

                                                [Service]
                                                ExecStartPre=/bin/sleep 30
                                                WorkingDirectory=/usr/local/bin
                                                ExecStart=/usr/local/bin/grow_part.sh

                                                [Install]
                                                WantedBy=multi-user.target`,
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0644",
						"DIRMODE": "0755",
					},
				},
				{
					Name:    "enable-grow-partinstall-service-script",
					Image:   tinkActionCexecImage(tinkerVersion),
					Timeout: timeOutAvg200,
					Environment: map[string]string{
						"BLOCK_DEVICE":        hardWareDesk + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "systemctl enable install-grow-part.service",
					},
				},

				{
					Name:    "add-apt-proxies",
					Image:   tinkActionWriteFileImage(tinkerVersion),
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"DEST_DISK": hardWareDesk + rootPart,
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/apt/apt.conf",
						"CONTENTS": `
						Acquire::http::Proxy "http://proxy-dmz.intel.com:911";
						Acquire::https::Proxy "http://proxy-dmz.intel.com:912";`,
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0644",
						"DIRMODE": "0755",
					},
				},
				{
					Name:    "reboot",
					Image:   "public.ecr.aws/l0g8r8j6/tinkerbell/hub/reboot-action:latest",
					Timeout: timeOutMin90,
					Volumes: []string{
						"/worker:/worker",
					},
				},
			},
		}},
	}

	return marshalWorkflow(&wf)
}
