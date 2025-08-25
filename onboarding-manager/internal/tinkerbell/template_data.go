// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tinkerbell

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/config"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/env"
	onboarding_types "github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/onboarding/types"
	networkconfig "github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/tinkerbell/config"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/cloudinit"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/platformbundle"
	platformbundleubuntu2204 "github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/platformbundle/ubuntu-22.04"
	"gopkg.in/yaml.v3"
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
)

var (
	zlog = logging.GetLogger("TinkerTemplateData")
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
		ENProxyNoProxy: infraConfig.ENProxyNoProxy,
	}

	return structToMapStringString(inputs), nil
}

func getCustomConfigs(deviceInfo onboarding_types.DeviceInfo) string {
	var allConfigs []string

	// Check for network configuration file and generate multi-NIC config if available
	networkConfigManager := networkconfig.NewNetworkConfigManager()
	networkConfig, err := networkConfigManager.LoadNetworkConfig()
	if err != nil {
		zlog.Warn().Err(err).Msg("Failed to load network configuration, using single NIC setup")
	} else if networkConfig != nil {
		zlog.Info().Msg("Generating multi-NIC configuration from config file")
		multiNICConfig := generateMultiNICCloudInitConfig(networkConfig)
		if multiNICConfig != "" {
			allConfigs = append(allConfigs, multiNICConfig)
		}
	} else {
		zlog.Info().Msg("No network configuration file found, using single NIC setup")
	}

	// Process any additional custom configs from DeviceInfo
	if len(deviceInfo.CustomConfigs) > 0 {
		processedConfigs := processCustomConfigs(deviceInfo.CustomConfigs)
		allConfigs = append(allConfigs, processedConfigs...)
	}

	if len(allConfigs) == 0 {
		return ""
	}

	// Merge all configurations intelligently
	finalConfig := mergeAllConfigurations(allConfigs)
	return strconv.Quote(finalConfig)
}

// generateMultiNICCloudInitConfig generates cloud-init configuration from network config file
func generateMultiNICCloudInitConfig(networkConfig *networkconfig.NetworkConfig) string {
	if networkConfig == nil || len(networkConfig.Interfaces) == 0 {
		return ""
	}

	config := map[string]interface{}{
		"network": generateNetworkSection(networkConfig),
	}

	// Add custom routes and commands if specified
	if len(networkConfig.Routes) > 0 {
		runcmd := generateRouteCommands(networkConfig.Routes)
		if len(runcmd) > 0 {
			config["runcmd"] = runcmd
		}
	}

	yamlData, err := yaml.Marshal(config)
	if err != nil {
		zlog.Error().Err(err).Msg("Failed to marshal network config to cloud-init")
		return ""
	}

	return "#cloud-config\n" + string(yamlData)
}

// generateNetworkSection generates the network section for cloud-init
func generateNetworkSection(networkConfig *networkconfig.NetworkConfig) map[string]interface{} {
	network := map[string]interface{}{
		"version": 2,
	}

	// Generate ethernets section
	if len(networkConfig.Interfaces) > 0 {
		ethernets := make(map[string]interface{})
		for _, iface := range networkConfig.Interfaces {
			ethernets[iface.Name] = generateInterfaceConfig(iface)
		}
		network["ethernets"] = ethernets
	}

	// Generate VLANs section
	if len(networkConfig.VLANs) > 0 {
		vlans := make(map[string]interface{})
		for _, vlan := range networkConfig.VLANs {
			vlans[vlan.Name] = generateVLANConfig(vlan)
		}
		network["vlans"] = vlans
	}

	// Generate bonds section
	if len(networkConfig.Bonds) > 0 {
		bonds := make(map[string]interface{})
		for _, bond := range networkConfig.Bonds {
			bonds[bond.Name] = generateBondConfig(bond)
		}
		network["bonds"] = bonds
	}

	return network
}

// generateInterfaceConfig generates configuration for a single interface
func generateInterfaceConfig(iface networkconfig.InterfaceConfig) map[string]interface{} {
	config := map[string]interface{}{
		"match": map[string]interface{}{
			"macaddress": iface.MacAddress,
		},
		"set-name": iface.Name,
	}

	if iface.DHCPMode {
		config["dhcp4"] = true
	} else if len(iface.Addresses) > 0 {
		config["addresses"] = iface.Addresses
	}

	if iface.Gateway != "" {
		config["gateway4"] = iface.Gateway
	}

	if len(iface.DNS) > 0 {
		config["nameservers"] = map[string]interface{}{
			"addresses": iface.DNS,
		}
	}

	if iface.MTU > 0 {
		config["mtu"] = iface.MTU
	}

	if iface.Optional {
		config["optional"] = true
	}

	return config
}

// generateVLANConfig generates configuration for a VLAN
func generateVLANConfig(vlan networkconfig.VLANConfig) map[string]interface{} {
	config := map[string]interface{}{
		"id":   vlan.ID,
		"link": vlan.Link,
	}

	if len(vlan.Addresses) > 0 {
		config["addresses"] = vlan.Addresses
	}

	if vlan.Gateway != "" {
		config["gateway4"] = vlan.Gateway
	}

	if len(vlan.DNS) > 0 {
		config["nameservers"] = map[string]interface{}{
			"addresses": vlan.DNS,
		}
	}

	return config
}

// generateBondConfig generates configuration for a bond
func generateBondConfig(bond networkconfig.BondConfig) map[string]interface{} {
	config := map[string]interface{}{
		"interfaces": bond.Interfaces,
		"parameters": map[string]interface{}{
			"mode": bond.Mode,
		},
	}

	if len(bond.Addresses) > 0 {
		config["addresses"] = bond.Addresses
	}

	if bond.Gateway != "" {
		config["gateway4"] = bond.Gateway
	}

	if len(bond.DNS) > 0 {
		config["nameservers"] = map[string]interface{}{
			"addresses": bond.DNS,
		}
	}

	return config
}

// generateRouteCommands generates runcmd entries for custom routes
func generateRouteCommands(routes []networkconfig.RouteConfig) []string {
	var commands []string

	for _, route := range routes {
		cmd := fmt.Sprintf("ip route add %s", route.To)
		
		if route.Via != "" {
			cmd += fmt.Sprintf(" via %s", route.Via)
		}
		
		if route.Interface != "" {
			cmd += fmt.Sprintf(" dev %s", route.Interface)
		}
		
		if route.Metric > 0 {
			cmd += fmt.Sprintf(" metric %d", route.Metric)
		}

		commands = append(commands, cmd)
	}

	return commands
}

// mergeAllConfigurations merges all configurations intelligently
func mergeAllConfigurations(configs []string) string {
	if len(configs) == 0 {
		return ""
	}
	if len(configs) == 1 {
		return configs[0]
	}

	// Separate cloud-init configs from others
	var cloudInitConfigs []string
	var otherConfigs []string

	for _, config := range configs {
		if isCloudInitConfig(config) {
			cloudInitConfigs = append(cloudInitConfigs, config)
		} else {
			otherConfigs = append(otherConfigs, config)
		}
	}

	var result []string

	// Merge cloud-init configs
	if len(cloudInitConfigs) > 0 {
		merged := mergeCloudInitConfigs(cloudInitConfigs)
		if merged != "" {
			result = append(result, merged)
		}
	}

	// Add other configs
	result = append(result, otherConfigs...)

	return strings.Join(result, customConfigDelimiter)
}

// processCustomConfigs handles cloud-init config processing with network merging support
func processCustomConfigs(configs map[string]string) []string {
	var cloudInitConfigs []string
	var otherConfigs []string

	// Separate cloud-init configs from other types
	for name, config := range configs {
		if strings.TrimSpace(config) == "" {
			zlog.Warn().Msgf("Skipping empty custom config: %s", name)
			continue
		}

		if isCloudInitConfig(config) {
			cloudInitConfigs = append(cloudInitConfigs, config)
		} else {
			zlog.Warn().Msgf("Custom config '%s' is not cloud-init format, including as-is", name)
			otherConfigs = append(otherConfigs, config)
		}
	}

	var result []string

	// Process cloud-init configs with intelligent merging
	if len(cloudInitConfigs) > 0 {
		merged := mergeCloudInitConfigs(cloudInitConfigs)
		if merged != "" {
			result = append(result, merged)
		}
	}

	// Add other configs as-is
	result = append(result, otherConfigs...)

	return result
}

// isCloudInitConfig checks if a config is a valid cloud-init configuration
func isCloudInitConfig(config string) bool {
	trimmed := strings.TrimSpace(config)
	return strings.HasPrefix(trimmed, "#cloud-config")
}

// mergeCloudInitConfigs intelligently merges multiple cloud-init configurations
func mergeCloudInitConfigs(configs []string) string {
	if len(configs) == 0 {
		return ""
	}
	if len(configs) == 1 {
		return configs[0]
	}

	zlog.Info().Msgf("Merging %d cloud-init configurations", len(configs))

	// Parse first config as base
	var baseConfig map[string]interface{}
	if err := yaml.Unmarshal([]byte(configs[0]), &baseConfig); err != nil {
		zlog.Error().Err(err).Msg("Failed to parse base cloud-init config, falling back to concatenation")
		return fallbackConcatenation(configs)
	}

	// Merge subsequent configs
	for i, config := range configs[1:] {
		var customConfig map[string]interface{}
		if err := yaml.Unmarshal([]byte(config), &customConfig); err != nil {
			zlog.Error().Err(err).Msgf("Failed to parse custom config %d, skipping", i+1)
			continue
		}
		baseConfig = mergeCloudInitMaps(baseConfig, customConfig)
	}

	// Marshal back to YAML
	merged, err := yaml.Marshal(baseConfig)
	if err != nil {
		zlog.Error().Err(err).Msg("Failed to marshal merged config, falling back to concatenation")
		return fallbackConcatenation(configs)
	}

	return "#cloud-config\n" + string(merged)
}

// mergeCloudInitMaps performs deep merge of cloud-init configuration maps
func mergeCloudInitMaps(base, custom map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy base configuration
	for k, v := range base {
		result[k] = v
	}

	// Merge custom configuration with special handling for network sections
	for k, v := range custom {
		if k == "#cloud-config" || strings.HasPrefix(k, "#") {
			continue // Skip comments and cloud-config markers
		}

		if existing, exists := result[k]; exists {
			switch k {
			case "network":
				result[k] = mergeNetworkConfig(existing, v)
			case "write_files":
				result[k] = mergeWriteFiles(existing, v)
			case "runcmd":
				result[k] = mergeRunCmds(existing, v)
			case "packages":
				result[k] = mergePackages(existing, v)
			default:
				// For other sections, custom config takes precedence
				result[k] = v
			}
		} else {
			result[k] = v
		}
	}

	return result
}

// mergeNetworkConfig merges network configurations with special handling for multiple NICs
func mergeNetworkConfig(existing, custom interface{}) interface{} {
	existingMap, existingOk := existing.(map[string]interface{})
	customMap, customOk := custom.(map[string]interface{})

	if !existingOk || !customOk {
		zlog.Warn().Msg("Network config not in expected format, using custom")
		return custom
	}

	result := make(map[string]interface{})

	// Copy existing network config
	for k, v := range existingMap {
		result[k] = v
	}

	// Merge custom network config with special handling for ethernets
	for k, v := range customMap {
		if k == "ethernets" {
			result[k] = mergeEthernetConfigs(result[k], v)
		} else {
			// For version, renderer, etc., custom takes precedence
			result[k] = v
		}
	}

	return result
}

// mergeEthernetConfigs merges ethernet interface configurations (allows multiple NICs)
func mergeEthernetConfigs(existing, custom interface{}) interface{} {
	existingMap, existingOk := existing.(map[string]interface{})
	customMap, customOk := custom.(map[string]interface{})

	if !existingOk && !customOk {
		return nil
	}
	if !existingOk {
		return custom
	}
	if !customOk {
		return existing
	}

	result := make(map[string]interface{})

	// Copy existing ethernet configs
	for k, v := range existingMap {
		result[k] = v
	}

	// Add/override custom ethernet configs (this allows multiple NICs)
	for k, v := range customMap {
		if _, exists := result[k]; exists {
			zlog.Warn().Msgf("Overriding existing ethernet config for interface: %s", k)
		}
		result[k] = v
	}

	zlog.Info().Msgf("Merged ethernet configurations for %d interfaces", len(result))
	return result
}

// mergeWriteFiles merges write_files arrays
func mergeWriteFiles(existing, custom interface{}) interface{} {
	existingSlice, existingOk := existing.([]interface{})
	customSlice, customOk := custom.([]interface{})

	if !existingOk && !customOk {
		return nil
	}
	if !existingOk {
		return custom
	}
	if !customOk {
		return existing
	}

	// Concatenate file lists
	result := make([]interface{}, 0, len(existingSlice)+len(customSlice))
	result = append(result, existingSlice...)
	result = append(result, customSlice...)

	return result
}

// mergeRunCmds merges runcmd arrays
func mergeRunCmds(existing, custom interface{}) interface{} {
	existingSlice, existingOk := existing.([]interface{})
	customSlice, customOk := custom.([]interface{})

	if !existingOk && !customOk {
		return nil
	}
	if !existingOk {
		return custom
	}
	if !customOk {
		return existing
	}

	// Concatenate command lists
	result := make([]interface{}, 0, len(existingSlice)+len(customSlice))
	result = append(result, existingSlice...)
	result = append(result, customSlice...)

	return result
}

// mergePackages merges package arrays
func mergePackages(existing, custom interface{}) interface{} {
	existingSlice, existingOk := existing.([]interface{})
	customSlice, customOk := custom.([]interface{})

	if !existingOk && !customOk {
		return nil
	}
	if !existingOk {
		return custom
	}
	if !customOk {
		return existing
	}

	// Deduplicate and merge package lists
	packageSet := make(map[string]bool)
	var result []interface{}

	// Add existing packages
	for _, pkg := range existingSlice {
		if pkgStr, ok := pkg.(string); ok {
			if !packageSet[pkgStr] {
				packageSet[pkgStr] = true
				result = append(result, pkg)
			}
		} else {
			result = append(result, pkg)
		}
	}

	// Add custom packages (deduplicate)
	for _, pkg := range customSlice {
		if pkgStr, ok := pkg.(string); ok {
			if !packageSet[pkgStr] {
				packageSet[pkgStr] = true
				result = append(result, pkg)
			}
		} else {
			result = append(result, pkg)
		}
	}

	return result
}

// fallbackConcatenation provides a simple fallback when YAML merging fails
func fallbackConcatenation(configs []string) string {
	zlog.Warn().Msg("Using fallback concatenation for cloud-init configs")
	return strings.Join(configs, customConfigDelimiter)
}
