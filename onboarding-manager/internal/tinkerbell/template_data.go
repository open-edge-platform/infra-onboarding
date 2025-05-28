// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tinkerbell

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strconv"

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
	ActionSecurityFeatures                 = "enable-security-features"
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
	SecureBootFlagRead    string
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

type WorkflowInputs struct {
	Env               Env
	DeviceInfo        onboarding_types.DeviceInfo
	TinkerActionImage TinkerActionImages
	CloudInitData     string
	InstallerScript   string
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
			key = prefix + key
		}

		if fieldVal.Kind() == reflect.Struct || (fieldVal.Kind() == reflect.Ptr && fieldVal.Elem().Kind() == reflect.Struct) {
			flattenStruct(fieldVal, key, result)
		} else {
			result[key] = fmt.Sprintf("%v", fieldVal.Interface())
		}
	}
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

func GenerateWorkflowInputs(ctx context.Context, deviceInfo onboarding_types.DeviceInfo) (map[string]string, error) {
	inputs := WorkflowInputs{
		DeviceInfo: deviceInfo,
		TinkerActionImage: TinkerActionImages{
			EraseNonRemovableDisk: tinkActionEraseNonRemovableDisk(deviceInfo.TinkerVersion),
			WriteFile:             tinkActionWriteFileImage(deviceInfo.TinkerVersion),
			SecureBootFlagRead:    tinkActionSecurebootFlagReadImage(deviceInfo.TinkerVersion),
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

	var installerScript string
	if deviceInfo.OsType == osv1.OsType_OS_TYPE_MUTABLE {
		if platformBundleData.InstallerScript != "" {
			installerScript = platformBundleData.InstallerScript
		} else {
			installerScript = platformbundleubuntu2204.Installer
		}
	}

	if infraConfig.NetIP == netIPStatic {
		opts = append(opts, cloudinit.WithPreserveIP(deviceInfo.HwIP, infraConfig.DNSServers))
	}
	cloudInitData, err := cloudinit.GenerateFromInfraConfig(platformBundleData.CloudInitTemplate, infraConfig, opts...)
	if err != nil {
		return nil, err
	}

	inputs.InstallerScript = strconv.Quote(installerScript)
	inputs.CloudInitData = strconv.Quote(cloudInitData)
	inputs.Env = Env{
		ENProxyHTTP:    infraConfig.ENProxyHTTP,
		ENProxyHTTPS:   infraConfig.ENProxyHTTPS,
		ENProxyNoProxy: infraConfig.ENProxyNoProxy,
	}

	return structToMapStringString(inputs), nil
}
