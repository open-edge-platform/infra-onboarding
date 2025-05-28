// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tinkerbell

import (
	"context"
	"fmt"
	"os"
	"reflect"

	osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/config"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/env"
	onboarding_types "github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/onboarding/types"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/cloudinit"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/platformbundle"
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
	ActionEfibootset                       = "efibootset-for-diskboot"
	ActionFdeEncryption                    = "fde-encryption"
	ActionEnableDmv                        = "enable-dm-verity"
	ActionFdeDmv                           = "fde-encryption-and-dm-verity-check"
	ActionKernelupgrade                    = "kernel-upgrade"
	ActionReboot                           = "reboot"
	ActionAddAptProxy                      = "add-apt-proxy"
	ActionCreateSecretsDirectory           = "create-node-directory" //#nosec G101 -- ignore false positive.
	ActionWriteClientID                    = "write-client-id"
	ActionWriteClientSecret                = "write-client-secret"
	ActionWriteHostname                    = "write-hostname"
	ActionSystemdNetworkOptimize           = "systemd-network-online-optimize"
	ActionDisableSnapdOptimize             = "systemd-snapd-disable-optimize"
	ActionCloudinitDsidentity              = "cloud-init-ds-identity"
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

	envTinkActionFdeDmvImage = "TINKER_FDE_DMV_IMAGE"

	envTinkActionKerenlUpgradeImage = "TINKER_KERNELUPGRD_IMAGE"

	envTinkActionQemuNbdImage2DiskImage = "TINKER_QEMU_NBD_IMAGE2DISK_IMAGE"

	envDkamDevMode = "dev"
	netIPStatic    = "static"

	tinkerActionEraseNonRemovableDisks = "erase_non_removable_disks"
	tinkerActionCexec                  = "cexec"
	tinkerActionFdeDmv                 = "fde_dmv"
	tinkerActionQemuNbdImage2Disk      = "qemu_nbd_image2disk"
	tinkerActionKernelUpgrade          = "kernelupgrd"
	tinkerActionEfibootset             = "efibootset"
	tinkerActionImage2Disk             = "image2disk"
	tinkerActionWritefile              = "writefile"
	tinkerActionSecurebootflag         = "securebootflag"
)

type TinkerActionImages struct {
	EraseNonRemovableDisk string
	WriteFile             string
	SecurebootFlagRead    string
	Cexec                 string
	Efibootset            string
	KernelUpgrade         string
	FdeDmv                string
	QemuNbdImage2Disk     string
	Image2Disk            string
}

type Env struct {
	ENProxyHTTP    string
	ENProxyHTTPS   string
	ENProxyNoProxy string
}

var (
	defaultEraseNonRemovableDiskImage        = getTinkerActionImage(tinkerActionEraseNonRemovableDisks)
	defaultTinkActionSecurebootFlagReadImage = getTinkerActionImage(tinkerActionSecurebootflag)
	defaultTinkActionWriteFileImage          = getTinkerActionImage(tinkerActionWritefile)
	defaultTinkActionCexecImage              = getTinkerActionImage(tinkerActionCexec)
	defaultTinkActionDiskImage               = getTinkerActionImage(tinkerActionImage2Disk)
	defaultTinkActionEfibootImage            = getTinkerActionImage(tinkerActionEfibootset)
	defaultTinkActionFdeDmvImage             = getTinkerActionImage(tinkerActionFdeDmv)
	defaultTinkActionKernelUpgradeImage      = getTinkerActionImage(tinkerActionKernelUpgrade)
	defaultTinkActionQemuNbdImage2DiskImage  = getTinkerActionImage(tinkerActionQemuNbdImage2Disk)
)

type WorkflowInputs struct {
	Env               Env
	DeviceInfo        onboarding_types.DeviceInfo
	TinkerActionImage TinkerActionImages
	CloudInitData     string
	InstallerScript   string
}

func structToMapStringString(input interface{}) map[string]string {
	result := make(map[string]string)
	flattenStruct(reflect.ValueOf(input), "", result)
	return result
}

func flattenStruct(val reflect.Value, prefix string, result map[string]string) {
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		key := field.Name
		if prefix != "" {
			key = prefix + "." + key
		}

		if fieldVal.Kind() == reflect.Struct || (fieldVal.Kind() == reflect.Ptr && fieldVal.Elem().Kind() == reflect.Struct) {
			flattenStruct(fieldVal, key, result)
		} else {
			result[key] = fmt.Sprintf("%v", fieldVal.Interface())
		}
	}
}

func GenerateWorkflowHardwareMap(ctx context.Context, deviceInfo onboarding_types.DeviceInfo) (map[string]string, error) {
	inputs := WorkflowInputs{
		DeviceInfo: deviceInfo,
		TinkerActionImage: TinkerActionImages{
			EraseNonRemovableDisk: tinkActionEraseNonRemovableDisk(deviceInfo.TinkerVersion),
			WriteFile:             tinkActionWriteFileImage(deviceInfo.TinkerVersion),
			SecurebootFlagRead:    tinkActionSecurebootFlagReadImage(deviceInfo.TinkerVersion),
			Cexec:                 tinkActionCexecImage(deviceInfo.TinkerVersion),
			Efibootset:            tinkActionEfibootImage(deviceInfo.TinkerVersion),
			KernelUpgrade:         tinkActionKernelupgradeImage(deviceInfo.TinkerVersion),
			FdeDmv:                tinkActionFdeDmvImage(deviceInfo.TinkerVersion),
			QemuNbdImage2Disk:     tinkActionQemuNbdImage2DiskImage(deviceInfo.TinkerVersion),
			Image2Disk:            tinkActionDiskImage(deviceInfo.TinkerVersion),
		},
	}

	infraConfig := config.GetInfraConfig()
	opts := []cloudinit.Option{
		cloudinit.WithOSType(deviceInfo.OsType),
		cloudinit.WithTenantID(deviceInfo.TenantID),
		cloudinit.WithHostname(deviceInfo.Hostname),
		cloudinit.WithClientCredentials(deviceInfo.AuthClientID, deviceInfo.AuthClientSecret),
		cloudinit.WithHostMACAddress(deviceInfo.HwMacID),
	}

	if env.ENDkamMode == envDkamDevMode {
		opts = append(opts, cloudinit.WithDevMode(env.ENUserName, env.ENPassWord))
	}

	if deviceInfo.LocalAccountUserName != "" && deviceInfo.SSHKey != "" {
		opts = append(opts, cloudinit.WithLocalAccount(deviceInfo.LocalAccountUserName, deviceInfo.SSHKey))
	}

	platformBundleData, err := platformbundle.FetchPlatformBundleScripts(ctx, deviceInfo.PlatformBundle)
	if err != nil {
		return nil, err
	}

	if infraConfig.NetIP == netIPStatic {
		opts = append(opts, cloudinit.WithPreserveIP(deviceInfo.HwIP, infraConfig.DNSServers))
	}
	cloudInitData, err := cloudinit.GenerateFromInfraConfig(platformBundleData.CloudInitTemplate, infraConfig, opts...)
	if err != nil {
		return nil, err
	}

	inputs.CloudInitData = cloudInitData
	inputs.Env = Env{
		ENProxyHTTP:    infraConfig.ENProxyHTTP,
		ENProxyHTTPS:   infraConfig.ENProxyHTTPS,
		ENProxyNoProxy: infraConfig.ENProxyNoProxy,
	}

	return structToMapStringString(inputs), nil
}

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

func tinkActionFdeDmvImage(tinkerImageVersion string) string {
	iv := getTinkerImageVersion(tinkerImageVersion)
	if v := os.Getenv(envTinkActionFdeDmvImage); v != "" {
		return fmt.Sprintf("%s:%s", v, iv)
	}
	return fmt.Sprintf("%s:%s", defaultTinkActionFdeDmvImage, iv)
}

func tinkActionKernelupgradeImage(tinkerImageVersion string) string {
	iv := getTinkerImageVersion(tinkerImageVersion)
	if v := os.Getenv(envTinkActionKerenlUpgradeImage); v != "" {
		return fmt.Sprintf("%s:%s", v, iv)
	}
	return fmt.Sprintf("%s:%s", defaultTinkActionKernelUpgradeImage, iv)
}

func tinkActionQemuNbdImage2DiskImage(tinkerImageVersion string) string {
	iv := getTinkerImageVersion(tinkerImageVersion)
	if v := os.Getenv(envTinkActionQemuNbdImage2DiskImage); v != "" {
		return fmt.Sprintf("%s:%s", v, iv)
	}
	return fmt.Sprintf("%s:%s", defaultTinkActionQemuNbdImage2DiskImage, iv)
}

//nolint:funlen,cyclop // Function length and cyclomatic complexity are high, but refactoring is deferred for now.
func NewTemplateDataUbuntu(ctx context.Context, name string, deviceInfo onboarding_types.DeviceInfo) ([]byte, error) {
	infraConfig := config.GetInfraConfig()
	opts := []cloudinit.Option{
		cloudinit.WithOSType(deviceInfo.OsType),
		cloudinit.WithTenantID(deviceInfo.TenantID),
		cloudinit.WithHostname(deviceInfo.Hostname),
		cloudinit.WithClientCredentials(deviceInfo.AuthClientID, deviceInfo.AuthClientSecret),
		cloudinit.WithHostMACAddress(deviceInfo.HwMacID),
	}

	if env.ENDkamMode == envDkamDevMode {
		opts = append(opts, cloudinit.WithDevMode(env.ENUserName, env.ENPassWord))
	}
	if deviceInfo.LocalAccountUserName != "" && deviceInfo.SSHKey != "" {
		opts = append(opts, cloudinit.WithLocalAccount(deviceInfo.LocalAccountUserName, deviceInfo.SSHKey))
	}

	platformBundleData, err := platformbundle.FetchPlatformBundleScripts(ctx, deviceInfo.PlatformBundle)
	if err != nil {
		return nil, err
	}

	var installerScript string
	if platformBundleData.InstallerScript != "" {
		installerScript = platformBundleData.InstallerScript
	} else {
		installerScript = platformbundleubuntu2204.Installer
	}

	if infraConfig.NetIP == netIPStatic {
		opts = append(opts, cloudinit.WithPreserveIP(deviceInfo.HwIP, infraConfig.DNSServers))
	}

	cloudInitData, err := cloudinit.GenerateFromInfraConfig(platformBundleData.CloudInitTemplate, infraConfig, opts...)
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
						"CONTENTS":  installerScript,
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
					Image:   tinkActionFdeDmvImage(deviceInfo.TinkerVersion),
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
