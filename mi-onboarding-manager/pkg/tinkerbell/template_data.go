// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package tinkerbell

import (
	"fmt"

	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/os/v1"
)

func NewTemplateDataProd(name, rootPart, rootPartNo, hostIP, provIP string) ([]byte, error) {
	wf := Workflow{
		Version:       "0.1",
		Name:          name,
		GlobalTimeout: 9800,
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
					Timeout: 9600,
					Environment: map[string]string{
						"DEST_DISK":  "{{ index .Hardware.Disks 0 }}",
						"IMG_URL":    fmt.Sprintf("http://%s:8080/focal-server-cloudimg-amd64.raw.gz", hostIP),
						"COMPRESSED": "true",
					},
				},

				{
					Name:    "copy-secrets",
					Image:   provIP + ":5015/cred_copy:latest",
					Timeout: 90,
					Environment: map[string]string{
						"BLOCK_DEVICE": "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":      "ext4",
					},
				},
				{
					Name:    "grow-partition-install-script",
					Image:   "quay.io/tinkerbell-actions/writefile:v1.0.0",
					Timeout: 90,
					Environment: map[string]string{
						"DEST_DISK": "{{ index .Hardware.Disks 0 }}" + rootPart,
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
					Name:    "install-openssl",
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
					Timeout: 90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "apt -y update && apt -y install openssl",
					},
				},
				{
					Name:    "create-user",
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
					Timeout: 90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "useradd -p $(openssl passwd -1 tink) -s /bin/bash -d /home/tink/ -m -G sudo tink",
					},
				},
				{
					Name:    "enable-ssh",
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
					Timeout: 90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE": "ssh-keygen -A; systemctl enable ssh.service; " +
							"sed -i 's/^PasswordAuthentication no/#PasswordAuthentication yes/g' /etc/ssh/sshd_config",
					},
				},
				{
					Name:    "disable-apparmor",
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
					Timeout: 90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "systemctl disable apparmor; systemctl disable snapd",
					},
				},
				{
					Name:    "write-netplan",
					Image:   "quay.io/tinkerbell-actions/writefile:v1.0.0",
					Timeout: 90,
					Environment: map[string]string{
						"DEST_DISK": "{{ index .Hardware.Disks 0 }}" + rootPart,
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
					Name:    "service-script-for-grow-partion-installer",
					Image:   "quay.io/tinkerbell-actions/writefile:v1.0.0",
					Timeout: 200,
					Environment: map[string]string{
						"DEST_DISK": "{{ index .Hardware.Disks 0 }}" + rootPart,
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
					Timeout: 200,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "systemctl enable install-grow-part.service",
					},
				},
				{
					Name:    "reboot",
					Image:   "public.ecr.aws/l0g8r8j6/tinkerbell/hub/reboot-action:latest",
					Timeout: 90,
					Volumes: []string{
						"/worker:/worker",
					},
				},
			},
		}},
	}

	return marshalWorkflow(&wf)
}

func NewTemplateDataProdBKC(name, rootPart, rootPartNo, hostIP, clientIP, clientID, clientSecret, gateway, _, _ string, securityFeature uint32) ([]byte, error) {
	wf := Workflow{
		Version:       "0.1",
		Name:          name,
		GlobalTimeout: 9800,
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
					Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/image2disk:0.7.1-dev",
					Timeout: 9600,
					Environment: map[string]string{
						"IMG_URL":    hostIP,
						"COMPRESSED": "true",
					},
				},
				{
					Name:    "add-env-proxy",
					Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/writefile:0.7.1-dev",
					Timeout: 90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/environment",
						"CONTENTS": `
http_proxy=http://proxy-dmz.intel.com:911
https_proxy=http://proxy-dmz.intel.com:912
ftp_proxy=http://proxy-dmz.intel.com:911
socks_proxy=http://proxy-dmz.intel.com:1080
no_proxy=localhost,*.intel.com,*intel.com,127.0.0.1,intel.com,.internal`,
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0755",
						"DIRMODE": "0755",
					},
				},

				{
					Name:    "add-apt-proxy",
					Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/writefile:0.7.1-dev",
					Timeout: 90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/apt/apt.conf",
						"CONTENTS": `
Acquire::http::Proxy "http://proxy-dmz.intel.com:911";
Acquire::https::Proxy "http://proxy-dmz.intel.com:911";`,
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0755",
						"DIRMODE": "0755",
					},
				},
				{
					Name:    "add-dns-namespace",
					Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/writefile:0.7.1-dev",
					Timeout: 90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/systemd/resolved.conf",
						"CONTENTS": `
[Resolve]
DNS=10.248.2.1 172.30.90.4 10.223.45.36`,
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0755",
						"DIRMODE": "0755",
					},
				},

				{
					Name:    "grow-partition-install-script",
					Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/writefile:0.7.1-dev",
					Timeout: 90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/usr/local/bin/grow_part.sh",
						"CONTENTS": fmt.Sprintf(`#!/bin/bash
growpart $DEST_DISK 1 
resize2fs $DEST_DISK$ID 
touch /usr/local/bin/.grow_part_done`),
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0755",
						"DIRMODE": "0755",
					},
				},
				{
					Name:    "create-ensp-node-directory",
					Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/cexec:0.7.1-dev",
					Timeout: 60,
					Environment: map[string]string{
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "mkdir -p /etc/ensp/node/client-credentials/",
					},
				},
				{
					Name:    "write-client-id",
					Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/writefile:0.7.1-dev",
					Timeout: 90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/ensp/node/client-credentials/client_id",
						"CONTENTS":  clientID,
						"UID":       "0",
						"GID":       "0",
						"MODE":      "0755",
						"DIRMODE":   "0755",
					},
				},
				{
					Name:    "write-client-secret",
					Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/writefile:0.7.1-dev",
					Timeout: 90,
					Environment: map[string]string{
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/ensp/node/client-credentials/client_secret",
						"CONTENTS":  clientSecret,
						"UID":       "0",
						"GID":       "0",
						"MODE":      "0755",
						"DIRMODE":   "0755",
					},
				},
				{
					Name:    "create-user",
					Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/cexec:0.7.1-dev",
					Timeout: 90,
					Environment: map[string]string{
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "useradd -p $(openssl passwd -1 user) -s /bin/bash -d /home/user/ -m -G sudo user",
					},
				},

				{
					Name:    "enable-ssh",
					Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/cexec:0.7.1-dev",
					Timeout: 90,
					Environment: map[string]string{
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE": "ssh-keygen -A;sed -i 's/^PasswordAuthentication " +
							"no/#PasswordAuthentication yes/g' /etc/ssh/sshd_config",
					},
				},

				{
					Name:    "profile-pkg-and-node-agents-install-script-download",
					Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/cexec:0.7.1-dev",
					Timeout: 200,
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
					Name:    "service-script-for-profile-pkg-and-node-agents-install",
					Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/writefile:0.7.1-dev",
					Timeout: 90,
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
					Name:    "enable-service-script-for-profile-pkg-node-agents",
					Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/cexec:0.7.1-dev",
					Timeout: 200,
					Environment: map[string]string{
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "systemctl enable install-profile-pkgs-and-node-agent.service",
					},
				},
				{
					Name:    "write-netplan",
					Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/writefile:0.7.1-dev",
					Timeout: 90,
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
					Name:    "update-netplan-to-make-ip-static",
					Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/writefile:0.7.1-dev",
					Timeout: 200,
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
        addresses: [10.248.2.1,172.30.90.4,10.223.45.36]
"
# Write the YAML configuration to the file
echo "$config_yaml" | tee /etc/netplan/config.yaml
ln -sf /run/systemd/resolve/stub-resolv.conf /etc/resolv.conf
touch .netplan_update_done
netplan apply`, clientIP),
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0755",
						"DIRMODE": "0755",
					},
				},

				{
					Name:    "service-script-for-grow-partion-installer",
					Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/writefile:0.7.1-dev",
					Timeout: 200,
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
					Name:    "enable-grow-partinstall-service-script",
					Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/cexec:0.7.1-dev",
					Timeout: 200,
					Environment: map[string]string{
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "systemctl disable install-grow-part.service",
					},
				},

				{
					Name:    "service-script-for-netplan-update",
					Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/writefile:0.7.1-dev",
					Timeout: 200,
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
					Name:    "enable-update-netplan.service-script",
					Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/cexec:0.7.1-dev",
					Timeout: 200,
					Environment: map[string]string{
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "systemctl enable update-netplan.service",
					},
				},

				{
                                        Name:    "efibootset-for-diskboot",
                                        Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/efibootset:0.7.1-dev",
                                        Timeout: 300,
                                },

				{
					Name:    "fde-encryption",
					Image:   "localhost:7443/one-intel-edge/edge-node/tinker-actions/fde:0.7.1-dev",
					Timeout: 560,
				},

				{
					Name:    "reboot",
					Image:   "public.ecr.aws/l0g8r8j6/tinkerbell/hub/reboot-action:latest",
					Timeout: 90,
					Volumes: []string{
						"/worker:/worker",
					},
				},
			},
		}},
	}

	// FDE removal if security feature flag is not set for FDE
	if osv1.SecurityFeature(securityFeature) != osv1.SecurityFeature_SECURITY_FEATURE_SECURE_BOOT_AND_FULL_DISK_ENCRYPTION {
		for i, task := range wf.Tasks {
			for j, action := range task.Actions {
				if action.Name == "fde-encryption" {
					// Remove the action from the slice
					wf.Tasks[i].Actions = append(wf.Tasks[i].Actions[:j], wf.Tasks[i].Actions[j+1:]...)
					break
				}
			}
		}
		// Enable the grow partition
		for _, task := range wf.Tasks {
			for _, action := range task.Actions {
				if action.Name == "enable-grow-partinstall-service-script" {
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

func NewTemplateDataProdMS(name, rootPart, _, hostIP, clientIP, gateway, mac, provIP string) ([]byte, error) {
	wf := Workflow{
		Version:       "0.1",
		Name:          name,
		GlobalTimeout: 9800,
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
					Timeout: 9600,
					Environment: map[string]string{
						"DEST_DISK":  "{{ index .Hardware.Disks 0 }}",
						"IMG_URL":    fmt.Sprintf("http://%s:8080/focal-server-cloudimg-amd64.raw.gz", hostIP),
						"COMPRESSED": "true",
					},
				},
				{
					Name:    "copy-secrets",
					Image:   provIP + ":5015/cred_copy:latest",
					Timeout: 90,
					Environment: map[string]string{
						"BLOCK_DEVICE": "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":      "ext4",
					},
				},
				{
					Name:    "grow-partition-install-script",
					Image:   "quay.io/tinkerbell-actions/writefile:v1.0.0",
					Timeout: 90,
					Environment: map[string]string{
						"DEST_DISK": "{{ index .Hardware.Disks 0 }}" + rootPart,
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
					Timeout: 90,
					Environment: map[string]string{
						"DEST_DISK": "{{ index .Hardware.Disks 0 }}" + rootPart,
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
					Timeout: 90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
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
					Timeout: 90,
					Environment: map[string]string{
						"DEST_DISK": "{{ index .Hardware.Disks 0 }}" + rootPart,
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
					Timeout: 90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "apt -y update && apt -y install wget openssl",
					},
				},
				{
					Name:    "create-user",
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
					Timeout: 90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "useradd -p $(openssl passwd -1 user) -s /bin/bash -d /home/user/ -m -G sudo user",
					},
				},
				{
					Name:    "hookos-bootmenu-delete-script",
					Image:   "quay.io/tinkerbell-actions/writefile:v1.0.0",
					Timeout: 90,
					Environment: map[string]string{
						"DEST_DISK": "{{ index .Hardware.Disks 0 }}" + rootPart,
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
					Timeout: 90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "/tmp/hook_part_del.sh",
					},
				},
				{
					Name:    "enable-ssh",
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
					Timeout: 90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
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
					Timeout: 90,
					Environment: map[string]string{
						"DEST_DISK": "{{ index .Hardware.Disks 0 }}" + rootPart,
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
					Timeout: 90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
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
					Timeout: 90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "cd /home/user/Setup && dpkg -i *.deb",
					},
				},
				{
					Name:    "download-azure-scripts",
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
					Timeout: 90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
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
					Timeout: 90,
					Environment: map[string]string{
						"DEST_DISK": "{{ index .Hardware.Disks 0 }}" + rootPart,
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
					Timeout: 90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "systemctl enable install-azure-dps.service",
					},
				},
				{
					Name:    "service-script-for-grow-partion-installer",
					Image:   "quay.io/tinkerbell-actions/writefile:v1.0.0",
					Timeout: 200,
					Environment: map[string]string{
						"DEST_DISK": "{{ index .Hardware.Disks 0 }}" + rootPart,
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
					Timeout: 200,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "systemctl enable install-grow-part.service",
					},
				},

				{
					Name:    "add-apt-proxies",
					Image:   "quay.io/tinkerbell-actions/writefile:v1.0.0",
					Timeout: 90,
					Environment: map[string]string{
						"DEST_DISK": "{{ index .Hardware.Disks 0 }}" + rootPart,
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
					Timeout: 90,
					Volumes: []string{
						"/worker:/worker",
					},
				},
			},
		}},
	}

	return marshalWorkflow(&wf)
}
