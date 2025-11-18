// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tinkerbell

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/config"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/env"
	onboarding_types "github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/onboarding/types"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/cloudinit"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/platformbundle"
	platformbundleubuntu2204 "github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/platformbundle/ubuntu-22.04"
)

const (
	ActionEraseNonRemovableDisk    = "erase-non-removable-disk" //#nosec G101 -- ignore false positive.
	ActionSecureBootStatusFlagRead = "secure-boot-status-flag-read"
	ActionInstallScriptDownload    = "profile-pkg-and-node-agents-install-script-download"
	ActionStreamOSImage            = "stream-os-image"
	ActionCloudInitInstall         = "install-cloud-init"
	ActionSystemConfiguration      = "system-configuration"
	ActionCustomConfigInstall      = "custom-configs"
	ActionCustomConfigSplit        = "custom-configs-split"
	ActionInstallScript            = "service-script-for-profile-pkg-and-node-agents-install"
	ActionEfibootset               = "efibootset-for-diskboot"
	ActionFdeEncryption            = "fde-encryption"
	ActionSecurityFeatures         = "enable-security-features"
	ActionKernelupgrade            = "kernel-upgrade"
	ActionReboot                   = "reboot"
	ActionAddAptProxy              = "add-apt-proxy"
	ActionCloudinitDsidentity      = "cloud-init-ds-identity"
)

const (
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

	// Use a delimiter that is highly unlikely to appear in any config or script.
	// ASCII Unit Separator (0x1F) is a safe choice.
	customConfigDelimiter = "\x1F"
	rawImageFormat        = "raw"
	qcow2ImageFormat      = "qcow2"
	httpTimeout           = 30 * time.Second
	qcow2HeaderSize       = 4
)

type TinkerActionImages struct {
	EraseNonRemovableDisk string
	WriteFile             string
	SecureBootFlagRead    string
	Cexec                 string
	Efibootset            string
	KernelUpgrade         string
	FdeDmv                string
	StreamOSImageToDisk   string
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
	CustomConfigs     string
	InstallerScript   string
	// OsResourceID resource ID of Operating System that was specified initially at the provisioning time
	OsResourceID string
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

// structToMapStringString this function takes an arbitrary object (e.g., nested struct)
// and recursively converts it into a flat map. Example:
//
//	type InnerStruct struct {
//	  A string // set to "example1"
//	  B string // set to "example2"
//	}
//
//	type OuterStruct struct {
//	  Inner InnerStruct
//	}
//
// will be converted to a map with the following elements:
// InnerA: example1
// InnerB: example2
// .
func structToMapStringString(input interface{}) map[string]string {
	result := make(map[string]string)
	flattenStruct(reflect.ValueOf(input), "", result)
	return result
}

// flattenStruct recursively reads a nested struct and generates a flat map.
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
		return v
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
		return v
	}
	return fmt.Sprintf("%s:%s", defaultTinkActionQemuNbdImage2DiskImage, iv)
}

// detectImageFormat probes the image URL to detect if it's qcow2 or raw format.
// Returns "qcow2" or "raw".
//
//nolint:cyclop
func detectImageFormat(ctx context.Context, imageURL, httpProxy string) string {
	// For .img files, probe the first few bytes to detect format
	// Remove newline characters from imageURL
	imageURL = strings.TrimSpace(imageURL)
	zlog.Info().Msgf("Detecting image format for URL: %s", imageURL)
	if strings.Contains(imageURL, ".img") {
		transport := &http.Transport{}

		// Configure proxy if provided
		if strings.TrimSpace(httpProxy) != "" {
			proxyURL, err := url.Parse(httpProxy)
			if err != nil {
				zlog.Warn().Err(err).Str("proxy", httpProxy).Msg("Failed to parse HTTP proxy URL, proceeding without proxy")
			} else {
				transport.Proxy = http.ProxyURL(proxyURL)
				zlog.Info().Str("proxy", httpProxy).Msg("Using HTTP proxy for image format detection")
			}
		} else {
			// Use proxy from environment variables if available
			transport.Proxy = http.ProxyFromEnvironment
		}

		client := &http.Client{
			Timeout:   httpTimeout, // Increased timeout for slow connections
			Transport: transport,
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
		if err != nil {
			zlog.Warn().Err(err).Msg("Unable to create http request, defaulting to raw")
			return rawImageFormat // default to raw on error
		}
		// Request only first qcow2HeaderSize bytes to check magic number
		req.Header.Set("Range", "bytes=0-3")

		resp, err := client.Do(req)
		if err != nil {
			zlog.Warn().Err(err).Str("url", imageURL).Msg("Unable to fetch image header, defaulting to raw")
			return rawImageFormat // default to raw on error
		}
		defer resp.Body.Close()

		// Check if server supports range requests
		if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
			zlog.Warn().Int("status", resp.StatusCode).Msg("Unexpected HTTP status, defaulting to raw")
			return rawImageFormat
		}
		// Read first qcow2HeaderSize bytes
		header := make([]byte, qcow2HeaderSize)
		n, err := io.ReadFull(resp.Body, header)
		if err != nil || n < qcow2HeaderSize {
			zlog.Warn().Err(err).Int("bytes_read", n).Msg("Unable to read image header, defaulting to raw")
			return rawImageFormat // default to raw on error
		}

		// QCOW2 magic number is 'Q', 'F', 'I', 0xfb (0x514649fb)
		if header[0] == 'Q' && header[1] == 'F' && header[2] == 'I' && header[3] == 0xfb {
			zlog.Info().Msg("Detected qcow2 image format")
			return qcow2ImageFormat
		}

		zlog.Info().Msg("Image format is not qcow2, defaulting to raw")
		return rawImageFormat
	}

	// For other extensions, default to raw
	zlog.Info().Msg("Not a .img file, defaulting to raw format")
	return rawImageFormat
}

func getStreamOSToDiskTinkerActionImage(ctx context.Context, imageURL, httpProxy, tinkerImageVersion string) string {
	imageFormat := detectImageFormat(ctx, imageURL, httpProxy)
	zlog.Info().Msgf("Detected image format:%s for image URL: %s", imageFormat, imageURL)
	if imageFormat == "qcow2" {
		return tinkActionQemuNbdImage2DiskImage(tinkerImageVersion)
	}
	// raw image
	return tinkActionDiskImage(tinkerImageVersion)
}

func GenerateWorkflowInputs(ctx context.Context, deviceInfo onboarding_types.DeviceInfo) (map[string]string, error) {
	infraConfig := config.GetInfraConfig()

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
			StreamOSImageToDisk: getStreamOSToDiskTinkerActionImage(ctx, deviceInfo.OSImageURL,
				infraConfig.ENProxyHTTP, deviceInfo.TinkerVersion),
		},
	}

	opts := []cloudinit.Option{
		cloudinit.WithOSType(deviceInfo.OsType),
		cloudinit.WithTenantID(deviceInfo.TenantID),
		cloudinit.WithHostname(deviceInfo.Hostname),
		cloudinit.WithClientCredentials(deviceInfo.AuthClientID, deviceInfo.AuthClientSecret),
		cloudinit.WithHostMACAddress(deviceInfo.HwMacID),
	}

	if deviceInfo.IsStandaloneNode {
		opts = append(opts, cloudinit.WithRunAsStandalone())
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
	inputs.CustomConfigs = getCustomConfigs(deviceInfo)

	inputs.Env = Env{
		ENProxyHTTP:    infraConfig.ENProxyHTTP,
		ENProxyHTTPS:   infraConfig.ENProxyHTTPS,
		ENProxyNoProxy: infraConfig.ENProxyNoProxy + ",.devtools.intel.com",
	}

	inputs.DeviceInfo.OSTLSCACert = deviceInfo.OSTLSCACert
	inputs.DeviceInfo.KernelVersion = deviceInfo.KernelVersion
	inputs.DeviceInfo.SkipKernelUpgrade = deviceInfo.SkipKernelUpgrade

	return structToMapStringString(inputs), nil
}

func getCustomConfigs(deviceInfo onboarding_types.DeviceInfo) string {
	concatenated := collections.ConcatMapValuesSorted(deviceInfo.CustomConfigs, customConfigDelimiter)
	if concatenated != "" {
		return strconv.Quote(concatenated)
	}
	return ""
}
