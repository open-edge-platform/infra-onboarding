// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package tinkerbell

import "fmt"

func NewTemplateDataProd(name, rootPart, rootPartNo, hostIP, provIp string) ([]byte, error) {
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
					Image:   provIp + ":5015/cred_copy:latest",
					Timeout: 90,
					Environment: map[string]string{
						"BLOCK_DEVICE":  "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":    "ext4",
					},
				},
				{
					Name:    "grow-partition",
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
					Timeout: 90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            fmt.Sprintf("growpart {{ index .Hardware.Disks 0 }} %s && resize2fs {{ index .Hardware.Disks 0 }}%s", rootPartNo, rootPart),
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
						"CMD_LINE":            "ssh-keygen -A; systemctl enable ssh.service; sed -i 's/^PasswordAuthentication no/PasswordAuthentication yes/g' /etc/ssh/sshd_config",
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
					Name:    "reboot",
					Image:   "public.ecr.aws/l0g8r8j6/tinkerbell/hub/reboot-action:latest",
					Timeout: 90,
					Volumes: []string{
						"/worker:/worker",
					},
				}},
		}},
	}

	return marshalWorkflow(&wf)
}

func NewTemplateDataProdBKC(name, rootPart, hostIP, clientImg, provIp string) ([]byte, error) {
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
						"IMG_URL":    fmt.Sprintf("http://%s:8080/%s", hostIP, clientImg),
						"COMPRESSED": "true",
					},
				},
				{
					Name:    "copy-secrets",
					Image:   provIp + ":5015/cred_copy:latest",
					Timeout: 90,
					Environment: map[string]string{
						"BLOCK_DEVICE":  "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":    "ext4",
					},
				},
				{
					Name:    "base-pkg-install-script-download",
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
					Timeout: 200,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"SCRIPT_URL":          fmt.Sprintf("http://%s:8080/base_installer.sh", hostIP),
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            fmt.Sprintf("mkdir -p /home/user/Setup;chown user:user /home/user/Setup;wget -P /home/user/Setup http://%s:8080/base_installer.sh; chmod 755 /home/user/Setup/base_installer.sh", hostIP),
					},
				},
				{
					Name:    "service-script-for-base-pkg-install",
					Image:   "quay.io/tinkerbell-actions/writefile:v1.0.0",
					Timeout: 90,
					Environment: map[string]string{
						"DEST_DISK": "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/systemd/system/install-base-pkgs.service",
						"CONTENTS": `
						[Unit]
						Description=Base Package Installation
						After=network.target
						ConditionPathExists = !/home/user/Setup/.base_pkg_install_done
		
						[Service]
						ExecStartPre=/bin/sleep 60
						WorkingDirectory=/home/user/Setup
						ExecStart=/home/user/Setup/base_installer.sh
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
					Name:    "enable-service-script",
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
					Timeout: 200,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "systemctl enable install-base-pkgs.service",
					},
				},
				{
					Name:    "add-dynamic-env-variables-on-node",
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
					Timeout: 200,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"SCRIPT_URL":          fmt.Sprintf("http://%s:8080/agent_node_env.txt", hostIP),
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            fmt.Sprintf("wget -P /home/user/Setup http://%s:8080/agent_node_env.txt ;chmod 755 /home/user/Setup/agent_node_env.txt", hostIP),
					},
				},
				{
					Name:    "add-agent-env-to-bashrc",
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
					Timeout: 200,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "cat /home/user/Setup/agent_node_env.txt >>/home/user/.bashrc;chown user:user /home/user/.bashrc",
					},
				},
				{
					Name:    "download-edge-agent-installer-file",
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
					Timeout: 200,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"SCRIPT_URL":          fmt.Sprintf("http://%s:8080/edge_node_installer.sh", hostIP),
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            fmt.Sprintf("wget -P /home/user/Setup http://%s:8080/edge_node_installer.sh; cd /home/user/Setup && chmod 755 edge_node_installer.sh", hostIP),
					},
				},
				{
					Name:    "download-inventory-agent-docker-file",
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
					Timeout: 200,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"SCRIPT_URL":          fmt.Sprintf("http://%s:8080/docker-compose-inv.yml", hostIP),
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            fmt.Sprintf("mkdir -p /home/user/Setup/inv_agent;wget -P /home/user/Setup/inv_agent http://%s:8080/docker-compose-inv.yml; cd /home/user/Setup/inv_agent && mv docker-compose-inv.yml docker-compose.yml", hostIP),
					},
				},
				{
					Name:    "download-update-mgr-agent-docker-file",
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
					Timeout: 200,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"SCRIPT_URL":          fmt.Sprintf("http://%s:8080/docker-compose-upd.yml", hostIP),
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            fmt.Sprintf("mkdir -p /home/user/Setup/upd_mgr_agent;wget -P /home/user/Setup/upd_mgr_agent http://%s:8080/docker-compose-upd.yml; cd /home/user/Setup/upd_mgr_agent && mv docker-compose-upd.yml docker-compose.yml", hostIP),
					},
				},
				{
					Name:    "download-telemetry-agent-file",
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
					Timeout: 200,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"SCRIPT_URL":          fmt.Sprintf("http://%s:8080/telemetry_agent_files.tar", hostIP),
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            fmt.Sprintf("mkdir -p /home/user/Setup/telmtry_agent;wget -P /home/user/Setup/telmtry_agent http://%s:8080/telemetry_agent_files.tar;cd /home/user/Setup/telmtry_agent;chmod 755 *", hostIP),
					},
				},
				{
					Name:    "service-script-for-node-agents-install",
					Image:   "quay.io/tinkerbell-actions/writefile:v1.0.0",
					Timeout: 200,
					Environment: map[string]string{
						"DEST_DISK": "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":   "ext4",
						"DEST_PATH": "/etc/systemd/system/install-edge-node-agents.service",
						"CONTENTS": `
						[Unit]
						Description=edge node agents Installation
						After=network.target
						ConditionPathExists = /home/user/Setup/.base_pkg_install_done
						ConditionPathExists = !/home/user/Setup/.agent_install_done
		
						[Service]
						ExecStartPre=/bin/sleep 10
						WorkingDirectory=/home/user/Setup
						ExecStart=/home/user/Setup/edge_node_installer.sh
		
						[Install]
						WantedBy=multi-user.target`,
						"UID":     "0",
						"GID":     "0",
						"MODE":    "0644",
						"DIRMODE": "0755",
					},
				},
				{
					Name:    "enable-agent-service-script",
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
					Timeout: 200,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "systemctl enable install-edge-node-agents.service",
					},
				},
				{
					Name:    "reboot",
					Image:   "public.ecr.aws/l0g8r8j6/tinkerbell/hub/reboot-action:latest",
					Timeout: 90,
					Volumes: []string{
						"/worker:/worker",
					},
				}},
		}},
	}

	return marshalWorkflow(&wf)
}

func NewTemplateDataProdMS(name, rootPart, rootPartNo, hostIP, clientIP, gateway, mac, provIp string) ([]byte, error) {
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
					Image:   provIp + ":5015/cred_copy:latest",
					Timeout: 90,
					Environment: map[string]string{
						"BLOCK_DEVICE":  "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":    "ext4",
					},
				},
				{
					Name:    "grow-partition",
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
					Timeout: 90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            fmt.Sprintf("growpart {{ index .Hardware.Disks 0 }} %s && resize2fs {{ index .Hardware.Disks 0 }}%s", rootPartNo, rootPart),
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
						"CMD_LINE":            "mkdir /etc/systemd/system/docker.service.d/; touch /etc/systemd/system/docker.service.d/proxy.conf;touch /etc/apt/apt.conf",
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
					Name:    "enable-ssh",
					Image:   "quay.io/tinkerbell-actions/cexec:v1.0.0",
					Timeout: 90,
					Environment: map[string]string{
						"BLOCK_DEVICE":        "{{ index .Hardware.Disks 0 }}" + rootPart,
						"FS_TYPE":             "ext4",
						"CHROOT":              "y",
						"DEFAULT_INTERPRETER": "/bin/sh -c",
						"CMD_LINE":            "ssh-keygen -A; systemctl enable ssh.service; sed -i 's/^PasswordAuthentication no/PasswordAuthentication yes/g' /etc/ssh/sshd_config",
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
						"CMD_LINE":            fmt.Sprintf("mkdir -p /home/user/Setup;chown user:user /home/user/Setup;wget -P /home/user/Setup http://%s:8080/linux-headers-5.15.96-lts.deb; wget -P /home/user/Setup http://%s:8080/linux-image-5.15.96-lts.deb", hostIP, hostIP),
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
						"CMD_LINE":            fmt.Sprintf("mkdir -p /home/user/Setup/.creds;wget -P /home/user/Setup/.creds http://%s:8080/azure-credentials.env_%s; wget -P /home/user/Setup http://%s:8080/log.sh;  wget -P /home/user/Setup http://%s:8080/azure_dps_installer.sh;chmod 755  /home/user/Setup/*; cd /home/user/Setup/.creds; mv azure-credentials.env_%s azure-credentials.env", hostIP, mac, hostIP, hostIP, mac),
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
