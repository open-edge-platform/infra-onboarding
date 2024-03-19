// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package tinkerbell

import (
	"fmt"
	"os"
	"strings"

	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/os/v1"
)

const (
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

type ProxySetup struct {
	httpProxy  string
	httpsProxy string
	noProxy    string
	dns        string
}

const (
	hardWareDesk             = "{{ index .Hardware.Disks 0 }}"
	tinkActionWriteFileImage = "localhost:7443/one-intel-edge/edge-node/tinker-actions/writefile:"
	tinkActionCexecImage     = "localhost:7443/one-intel-edge/edge-node/tinker-actions/cexec:"
	timeOutMax9800           = 9800
	timeOutMax8000           = 8000
	timeOutAvg560            = 560
	timeOutAvg200            = 200
	timeOutMin90             = 90
	timeOutMin30             = 30
	leaseTime86400           = 86400
)

func GetProxyEnv() ProxySetup {
	var proxySettings ProxySetup

	proxySettings.httpProxy = os.Getenv("EN_HTTP_PROXY")
	proxySettings.httpsProxy = os.Getenv("EN_HTTPS_PROXY")
	proxySettings.noProxy = os.Getenv("EN_NO_PROXY")
	proxySettings.dns = os.Getenv("EN_NAMESERVERS")

	return proxySettings
}

//nolint:funlen // May effect the functionality, need to simplify this in future
func NewTemplateDataProd(name, rootPart, rootPartNo, hostIP, provIP string) ([]byte, error) {
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
					Image:   "quay.io/tinkerbell-actions/image2disk:v1.0.0",
					Timeout: timeOutMax9800,
					Environment: map[string]string{
						"DEST_DISK":  hardWareDesk,
						"IMG_URL":    fmt.Sprintf("http://%s:8080/focal-server-cloudimg-amd64.raw.gz", hostIP),
						"COMPRESSED": "true",
					},
				},

				{
					Name:    ActionCopySecrets,
					Image:   provIP + ":5015/cred_copy:latest",
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"BLOCK_DEVICE": hardWareDesk + rootPart,
						"FS_TYPE":      "ext4",
					},
				},
				{
					Name:    ActionGrowPartitionInstallScript,
					Image:   "quay.io/tinkerbell-actions/writefile:v1.0.0",
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
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
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
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
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
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
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
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
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
					Image:   "quay.io/tinkerbell-actions/writefile:v1.0.0",
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
					Image:   "quay.io/tinkerbell-actions/writefile:v1.0.0",
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
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
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
func NewTemplateDataProdBKC(name, _, rootPartNo, hostIP, clientIP, _, _, _ string,
	securityFeature uint32, clientID, clientSecret string, enableDI bool, tinkerversion string, hostname string,
) ([]byte, error) {
	proxySetting := GetProxyEnv()
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
					Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/image2disk:" + tinkerversion,
					Timeout: timeOutMax9800,
					Environment: map[string]string{
						"IMG_URL":    hostIP,
						"COMPRESSED": "true",
					},
				},
				{
					Name:    ActionAddEnvProxy,
					Image:   tinkActionWriteFileImage + tinkerversion,
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/environment",
						"CONTENTS": fmt.Sprintf(`
						http_proxy=%s
						https_proxy=%s
						no_proxy=%s`, proxySetting.httpProxy, proxySetting.httpsProxy, proxySetting.noProxy),
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0755",
						"DIRMODE": "0755",
					},
				},

				{
					Name:    ActionAddAptProxy,
					Image:   tinkActionWriteFileImage + tinkerversion,
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/apt/apt.conf",
						"CONTENTS": fmt.Sprintf(`
						Acquire::http::Proxy "%s";
						Acquire::https::Proxy "%s";`, proxySetting.httpProxy, proxySetting.httpsProxy),
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0755",
						"DIRMODE": "0755",
					},
				},
				{
					Name:    ActionWriteHostname,
					Image:   tinkActionWriteFileImage + tinkerversion,
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/hostname",
						"CONTENTS": fmt.Sprintf(`
%s`, hostname),
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0755",
						"DIRMODE": "0755",
					},
				},
				{
					Name:    ActionWriteEtcHosts,
					Image:   tinkActionWriteFileImage + tinkerversion,
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/hosts",
						"CONTENTS": fmt.Sprintf(`
127.0.0.1 %s
            `, hostname),
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0755",
						"DIRMODE": "0755",
					},
				},
				{
					Name:    ActionAddDNSNamespace,
					Image:   tinkActionWriteFileImage + tinkerversion,
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/systemd/resolved.conf",
						"CONTENTS": fmt.Sprintf(`
						[Resolve]
						DNS "%s"`, proxySetting.dns),
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0755",
						"DIRMODE": "0755",
					},
				},

				{
					Name:    ActionGrowPartitionInstallScript,
					Image:   tinkActionWriteFileImage + tinkerversion,
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
					Image:   tinkActionCexecImage + tinkerversion,
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "useradd -p $(openssl passwd -1 user) -s /bin/bash -d /home/user/ -m -G sudo user",
					},
				},

				{
					Name:    ActionEnableSSH,
					Image:   tinkActionCexecImage + tinkerversion,
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE": "ssh-keygen -A;sed -i 's/^PasswordAuthentication " +
							"no/#PasswordAuthentication yes/g' /etc/ssh/sshd_config",
					},
				},

				{
					Name:    ActionInstallScriptDownload,
					Image:   tinkActionCexecImage + tinkerversion,
					Timeout: timeOutAvg200,
					Environment: map[string]string{
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"SCRIPT_URL":          rootPartNo,
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE": fmt.Sprintf("mkdir -p /home/user/Setup;chown user:user /home/user/Setup;"+
							"wget -P /home/user/Setup %s; chmod 755 /home/user/Setup/installer.sh", rootPartNo),
					},
				},
				{
					Name:    ActionInstallScript,
					Image:   tinkActionWriteFileImage + tinkerversion,
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/systemd/system/install-profile-pkgs-and-node-agent.service",
						"CONTENTS": `
						[Unit]
						Description=Profile and node agents Package Installation
						After=update-netplan.service
						ConditionPathExists = !/home/user/Setup/.base_pkg_install_done
		
						[Service]
						ExecStartPre=/bin/sleep 20 
						WorkingDirectory=/home/user/Setup
						ExecStart=/home/user/Setup/installer.sh
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
					Image:   tinkActionCexecImage + tinkerversion,
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
					Image:   tinkActionWriteFileImage + tinkerversion,
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
					Image:   tinkActionWriteFileImage + tinkerversion,
					Timeout: timeOutAvg200,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/home/user/Setup/update_netplan_config.sh",
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
netplan apply`, clientIP, strings.ReplaceAll(proxySetting.dns, " ", ", ")),
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0755",
						"DIRMODE": "0755",
					},
				},

				{
					Name:    ActionGrowPartitionService,
					Image:   tinkActionWriteFileImage + tinkerversion,
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
					Image:   tinkActionCexecImage + tinkerversion,
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
					Image:   tinkActionWriteFileImage + tinkerversion,
					Timeout: timeOutAvg200,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/systemd/system/update-netplan.service",
						"CONTENTS": `
                                                [Unit]
                                                Description=update the netplan with to make static ip
                                                After=network.target
                                                ConditionPathExists = !/home/user/Setup/.netplan_update_done

                                                [Service]
                                                ExecStartPre=/bin/sleep 60 
                                                WorkingDirectory=/home/user/Setup
                                                ExecStart=/home/user/Setup/update_netplan_config.sh

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
					Image:   tinkActionCexecImage + tinkerversion,
					Timeout: timeOutAvg200,
					Environment: map[string]string{
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "systemctl enable update-netplan.service",
					},
				},

				{
					Name:    ActionEfibootset,
					Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/efibootset:" + tinkerversion,
					Timeout: timeOutAvg560,
				},

				{
					Name:    ActionFdeEncryption,
					Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/fde:" + tinkerversion,
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
				Image:   tinkActionCexecImage + tinkerversion,
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
				Image:   tinkActionWriteFileImage + tinkerversion,
				Timeout: timeOutMin90,
				Environment: map[string]string{
					"FS_TYPE":   "ext4",
					"DEST_PATH": "/etc/intel_edge_node/client-credentials/client_id",
					"CONTENTS":  clientID,
					"UID":       "0",
					"GID":       "0",
					"MODE":      "0755",
					"DIRMODE":   "0755",
				},
			},
			{
				Name:    ActionWriteClientSecret,
				Image:   tinkActionWriteFileImage + tinkerversion,
				Timeout: timeOutMin90,
				Environment: map[string]string{
					"FS_TYPE":   "ext4",
					"DEST_PATH": "/etc/intel_edge_node/client-credentials/client_secret",
					"CONTENTS":  clientSecret,
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
				Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/cred_copy:" + tinkerversion,
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
	if osv1.SecurityFeature(securityFeature) != osv1.SecurityFeature_SECURITY_FEATURE_SECURE_BOOT_AND_FULL_DISK_ENCRYPTION {
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
	return marshalWorkflow(&wf)
}

//nolint:funlen // May effect the functionality, need to simplify this in future
func NewTemplateDataProdMS(name, rootPart, _, hostIP, clientIP, gateway, mac, provIP string,
) ([]byte, error) {
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
					Image:   "quay.io/tinkerbell-actions/image2disk:v1.0.0",
					Timeout: timeOutMax9800,
					Environment: map[string]string{
						"DEST_DISK":  hardWareDesk,
						"IMG_URL":    fmt.Sprintf("http://%s:8080/focal-server-cloudimg-amd64.raw.gz", hostIP),
						"COMPRESSED": "true",
					},
				},
				{
					Name:    "copy-secrets",
					Image:   provIP + ":5015/cred_copy:latest",
					Timeout: timeOutMin90,
					Environment: map[string]string{
						"BLOCK_DEVICE": hardWareDesk + rootPart,
						"FS_TYPE":      "ext4",
					},
				},
				{
					Name:    ActionGrowPartitionInstallScript,
					Image:   "quay.io/tinkerbell-actions/writefile:v1.0.0",
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
					Image:   "quay.io/tinkerbell-actions/writefile:v1.0.0",
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
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
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
					Image:   "quay.io/tinkerbell-actions/writefile:v1.0.0",
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
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
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
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
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
					Image:   "quay.io/tinkerbell-actions/writefile:v1.0.0",
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
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
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
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
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
					Image:   "quay.io/tinkerbell-actions/writefile:v1.0.0",
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
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
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
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
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
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
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
					Image:   "quay.io/tinkerbell-actions/writefile:v1.0.0",
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
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
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
					Image:   "quay.io/tinkerbell-actions/writefile:v1.0.0",
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
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
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
					Image:   "quay.io/tinkerbell-actions/writefile:v1.0.0",
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
