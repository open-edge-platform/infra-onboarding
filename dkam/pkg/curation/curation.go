// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package curation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	osv1 "github.com/intel/infra-core/inventory/v2/pkg/api/os/v1"
	as "github.com/intel/infra-core/inventory/v2/pkg/artifactservice"
	inv_errors "github.com/intel/infra-core/inventory/v2/pkg/errors"
	"github.com/intel/infra-core/inventory/v2/pkg/logging"
	"github.com/intel/infra-onboarding/dkam/internal/env"
	"github.com/intel/infra-onboarding/dkam/internal/flag"
	"github.com/intel/infra-onboarding/dkam/pkg/config"
	"github.com/intel/infra-onboarding/dkam/pkg/util"
)

var zlog = logging.GetLogger("InfraDKAMAuth")

const (
	// file owner RW permissions.
	writeMode = 0o600
	// file owner RWX, other user RX permissions.
	fileMode = 0o755
)

// FirewallRule UFW Firewall structure in JSON, expected to be provided as environment variable.
type FirewallRule struct {
	//nolint:tagliatelle // Renaming the json keys may effect while unmarshalling/marshaling so, used nolint.
	SourceIP string `json:"sourceIp,omitempty"`
	Ports    string `json:"ports,omitempty"`
	//nolint:tagliatelle // Renaming the json keys may effect while unmarshalling/marshaling so, used nolint.
	IPVer    string `json:"ipVer,omitempty"`
	Protocol string `json:"protocol,omitempty"`
}

func GetBMAgentsInfo() (agentsList []config.AgentsVersion, distribution string, err error) {
	configs := config.GetInfraConfig().ENManifest

	agentsList = configs.BMA.Debs

	distribution = configs.Metadata.DebianRepositories[0].Distribution

	zlog.InfraSec().Info().Msgf("Agents List' %s", agentsList)

	return agentsList, distribution, nil
}

func getCaCert() (string, error) {
	caPath := config.OrchCACertificateFile
	caexists, err := util.PathExists(caPath)
	if err != nil {
		errMsg := "Failed to check if CA certificate path exists"
		zlog.Error().Err(err).Msg(errMsg)
		return "", inv_errors.Errorf("%s", errMsg)
	}

	if !caexists {
		zlog.Error().Msgf("Cannot find CA certificate under path %s", caPath)
		return "", inv_errors.Errorf("Cannot find CA certificate under given path")
	}

	caContent, err := os.ReadFile(caPath)
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msg("")
		return "", inv_errors.Errorf("Failed to read CA certificate file")
	}

	return string(caContent), nil
}

// ufw rules if true, iptables otherwise.
func getCustomFirewallRules(ufw bool) ([]string, error) {
	reqRules, err := ParseJSONFirewallRules(config.GetInfraConfig().FirewallReqAllow)
	if err != nil {
		return nil, err
	}

	cfgRules, err := ParseJSONFirewallRules(config.GetInfraConfig().FirewallCfgAllow)
	if err != nil {
		return nil, err
	}

	firewallRules := make([]string, 0)
	for _, rule := range append(reqRules, cfgRules...) {
		if ufw {
			firewallRules = append(firewallRules, GenerateUFWCommands(rule)...)
		} else {
			firewallRules = append(firewallRules, GenerateIptablesCommands(rule)...)
		}
	}

	return firewallRules, nil
}

func getAgentsListTemplateVariables() (map[string]interface{}, error) {
	agentsList, distro, err := GetBMAgentsInfo()
	zlog.InfraSec().Info().Msgf("Agents List' %s", agentsList)
	if len(agentsList) == 0 {
		zlog.InfraSec().Info().Msg("Failed to get the agent list")
		return nil, err
	}

	templateVariables := make(map[string]interface{}, len(agentsList))
	for _, agent := range agentsList {
		zlog.InfraSec().Info().Msgf("Package: %s, Version: %s\n", agent.Package, agent.Version)
		templateVariables[agent.Package+"-VERSION"] = agent.Version
	}

	templateVariables["APT_DISTRO"] = distro

	return templateVariables, nil
}

func GetCommonInfraTemplateVariables(infraConfig config.InfraConfig, osType osv1.OsType) (map[string]interface{}, error) {
	caCert, err := getCaCert()
	if err != nil {
		return nil, err
	}

	firewallRules, err := getCustomFirewallRules(osType == osv1.OsType_OS_TYPE_MUTABLE)
	if err != nil {
		return nil, err
	}

	templateVariables := map[string]interface{}{
		"MODE": os.Getenv("MODE"),

		"CA_CERT": caCert,

		"ORCH_CLUSTER":                   infraConfig.ClusterURL,
		"ORCH_INFRA":                     infraConfig.InfraURL,
		"ORCH_UPDATE":                    infraConfig.UpdateURL,
		"ORCH_PLATFORM_OBS_HOST":         strings.Split(infraConfig.LogsObservabilityURL, ":")[0],
		"ORCH_PLATFORM_OBS_PORT":         strings.Split(infraConfig.LogsObservabilityURL, ":")[1],
		"ORCH_PLATFORM_OBS_METRICS_HOST": strings.Split(infraConfig.MetricsObservabilityURL, ":")[0],
		"ORCH_PLATFORM_OBS_METRICS_PORT": strings.Split(infraConfig.MetricsObservabilityURL, ":")[1],
		"ORCH_TELEMETRY_HOST":            strings.Split(infraConfig.TelemetryURL, ":")[0],
		"ORCH_TELEMETRY_PORT":            strings.Split(infraConfig.TelemetryURL, ":")[1],
		"KEYCLOAK_URL":                   infraConfig.KeycloakURL,
		"KEYCLOAK_FQDN":                  strings.Split(infraConfig.KeycloakURL, ":")[0],
		"RELEASE_FQDN":                   strings.Split(infraConfig.ReleaseServiceURL, ":")[0],
		"RELEASE_TOKEN_URL":              infraConfig.ReleaseServiceURL,
		"ORCH_APT_PORT":                  strings.Split(infraConfig.FileServerURL, ":")[1],
		"ORCH_IMG_PORT":                  strings.Split(infraConfig.RegistryURL, ":")[1],
		"FILE_SERVER":                    strings.Split(infraConfig.FileServerURL, ":")[0],
		"IMG_REGISTRY_URL":               strings.Split(infraConfig.RegistryURL, ":")[0],
		"NTP_SERVERS":                    strings.Join(infraConfig.NTPServers, ","),
		"DEB_PACKAGES_REPO":              infraConfig.ENDebianPackagesRepo,

		"EN_HTTP_PROXY":  infraConfig.ENProxyHTTP,
		"EN_HTTPS_PROXY": infraConfig.ENProxyHTTPS,
		"EN_NO_PROXY":    infraConfig.ENProxyNoProxy,
		"EN_FTP_PROXY":   infraConfig.ENProxyFTP,
		"EN_SOCKS_PROXY": infraConfig.ENProxySocks,

		"KERNEL_CONFIG_OVER_COMMIT_MEMORY": infraConfig.SystemConfigVmOverCommitMemory,
		"KERNEL_CONFIG_PANIC_ON_OOPS":      infraConfig.SystemConfigKernelPanicOnOops,
		"KERNEL_CONFIG_KERNEL_PANIC":       infraConfig.SystemConfigKernelPanic,
		"KERNEL_CONFIG_MAX_USER_INSTANCE":  infraConfig.SystemConfigFsInotifyMaxUserInstances,

		"NETIP": infraConfig.NetIP,

		"EXTRA_HOSTS": infraConfig.ExtraHosts,

		"FIREWALL_RULES": firewallRules,

		// TODO: keeping OS-dependence for now, but will be removed once we reach the final solution
		"IS_TIBERMICROVISOR": osType == osv1.OsType_OS_TYPE_IMMUTABLE,
	}

	if osType == osv1.OsType_OS_TYPE_MUTABLE {
		templateVariables["FIREWALL_PROVIDER"] = "ufw"
	} else if osType == osv1.OsType_OS_TYPE_IMMUTABLE {
		templateVariables["FIREWALL_PROVIDER"] = "iptables"
	}

	if osType == osv1.OsType_OS_TYPE_MUTABLE {
		agentsListVariables, err := getAgentsListTemplateVariables()
		if err != nil {
			return nil, err
		}

		for agentsPackage, agentsVersion := range agentsListVariables {
			templateVariables[agentsPackage] = agentsVersion
		}
	}

	return templateVariables, nil
}

func WritePlatformBundleToPV(ctx context.Context, osRes *osv1.OperatingSystemResource, installerScriptPath string) error {
	// FIXME: hardcoded path for now. Will be fixed once we integrate EEF Platform Bundle coming from RS
	platformBundleScriptData, err := os.ReadFile(config.ScriptPath + "/platform-bundle/ubuntu-22.04/Installer")
	if err != nil {
		return err
	}
	platformBundleScript := string(platformBundleScriptData)

	platformBundleScript, err = FetchAndAppendProfileScript(ctx, osRes.GetProfileName(), platformBundleScript)
	if err != nil {
		return err
	}

	writeErr := WriteFileToPath(installerScriptPath, []byte(platformBundleScript))
	if writeErr != nil {
		zlog.InfraSec().Error().Err(writeErr).Msgf("Failed to write file to path %s", installerScriptPath)
		return writeErr
	}

	return nil
}

func CurateScript(ctx context.Context, osRes *osv1.OperatingSystemResource) error {
	if osRes.GetOsType() == osv1.OsType_OS_TYPE_IMMUTABLE {
		zlog.InfraSec().Info().Msgf("Skipping script curation for immutable OS %s", osRes.GetResourceId())
		return nil
	}

	installerScriptPath, err := util.GetInstallerLocation(osRes, config.PVC)
	if err != nil {
		return err
	}

	if osRes.GetOsType() == osv1.OsType_OS_TYPE_MUTABLE && !*flag.LegacyMode {
		return WritePlatformBundleToPV(ctx, osRes, installerScriptPath)
	}

	localInstallerPath, err := util.GetLocalInstallerPath(osRes.GetOsType())
	if err != nil {
		return err
	}

	templateVariables, err := GetCommonInfraTemplateVariables(config.GetInfraConfig(), osRes.GetOsType())
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msg("Failed to get template variables for curation")
		return err
	}

	tmplScript, err := os.ReadFile(localInstallerPath)
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msgf(
			"Failed to read template of installation script from path %v", localInstallerPath)
		return err
	}

	curatedScriptData, createErr := CurateFromTemplate(string(tmplScript), templateVariables)
	if createErr != nil {
		zlog.InfraSec().Error().Msgf("Error checking path %v", createErr)
		return createErr
	}

	return finalizeCuratedScript(ctx, osRes, installerScriptPath, curatedScriptData)
}

func finalizeCuratedScript(ctx context.Context, osRes *osv1.OperatingSystemResource,
	installerScriptPath, curatedScriptData string,
) error {
	// Append profile script, to be removed once Platform Bundle is integrated
	var err error
	curatedScriptData, err = FetchAndAppendProfileScript(ctx, osRes.GetProfileName(), curatedScriptData)
	if err != nil {
		return err
	}

	writeErr := WriteFileToPath(installerScriptPath, []byte(curatedScriptData))
	if writeErr != nil {
		zlog.InfraSec().Error().Err(writeErr).Msgf("Failed to write file to path %s", installerScriptPath)
		return writeErr
	}

	return nil
}

func CurateFromTemplate(tmpl string, templateVariables map[string]interface{}) (string, error) {
	// Parse and execute the template
	// We use sprig to extend basic Go's text/template with more powerful keywords
	// See: https://masterminds.github.io/sprig/
	// This function will fail if any of keys is not provided
	t, err := template.New("installer").Option("missingkey=error").Funcs(sprig.TxtFuncMap()).Parse(tmpl)
	if err != nil {
		invErr := inv_errors.Errorf("Failed to parse installation script template")
		zlog.Error().Err(err).Msg(invErr.Error())
		return "", invErr
	}

	var rendered bytes.Buffer
	if renderErr := t.Execute(&rendered, templateVariables); renderErr != nil {
		invErr := inv_errors.Errorf("Failed to render installation script")
		zlog.Error().Err(renderErr).Msg(invErr.Error())
		return "", invErr
	}

	return rendered.String(), nil
}

func WriteFileToPath(filePath string, content []byte) error {
	zlog.Debug().Msgf("Writing data to path %s", filePath)

	err := os.MkdirAll(filepath.Dir(filePath), fileMode)
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msg("")
		return inv_errors.Errorf("Failed to create sub-directories to save file")
	}

	err = os.WriteFile(filePath, content, writeMode)
	if err != nil {
		errMsg := "Failed save the data to output path"
		zlog.Error().Err(err).Msg(errMsg)
		return inv_errors.Errorf("%s", errMsg)
	}

	return nil
}

func FetchAndAppendProfileScript(ctx context.Context, profileName, originalScript string) (string, error) {
	// FIXME: hardcode profile script version for now, will be addressed in https://jira.devtools.intel.com/browse/NEX-11556
	profileScriptVersion := "1.0.2"

	repo := env.ProfileScriptRepo + profileName
	zlog.InfraSec().Info().Msgf("Profile script repo URL is:%s", repo)

	artifacts, err := as.DownloadArtifacts(ctx, repo, profileScriptVersion)
	if err != nil {
		invErr := inv_errors.Errorf("Error downloading profile script for tag %s", profileScriptVersion)
		zlog.Err(invErr).Msg("")
		return "", invErr
	}

	if artifacts == nil {
		invErr := inv_errors.Errorf("Unexpected nil data received for %s:%s", repo, profileScriptVersion)
		zlog.Err(invErr).Msg("")
		return "", invErr
	}

	if artifacts != nil && len(*artifacts) == 0 {
		zlog.InfraSec().Debug().Msgf("No artifacts found for profile %s", profileName)
		return originalScript, nil
	}

	profileScript := (*artifacts)[0].Data

	return originalScript + string(profileScript), nil
}

// GenerateUFWCommands convert a FirewallRule into the corresponding ufw command.
func GenerateUFWCommands(rule FirewallRule) []string {
	commands := []string{}
	ipAddr := ""
	if rule.SourceIP != "" {
		ip := net.ParseIP(rule.SourceIP)
		if ip == nil {
			ipAddr = "$(dig +short " + rule.SourceIP + " | tail -n1)"
		} else {
			ipAddr = rule.SourceIP
		}
		if rule.Protocol != "" {
			if rule.Ports != "" {
				commands = append(commands, fmt.Sprintf("ufw allow from %s to any port %s proto %s",
					ipAddr, rule.Ports, rule.Protocol))
			} else {
				commands = append(commands, fmt.Sprintf("ufw allow from %s proto %s", ipAddr, rule.Protocol))
			}
		} else {
			if rule.Ports != "" {
				commands = append(commands, fmt.Sprintf("ufw allow from %s to any port %s", ipAddr, rule.Ports))
			} else {
				commands = append(commands, fmt.Sprintf("ufw allow from %s", ipAddr))
			}
		}
	} else {
		if rule.Protocol != "" {
			if rule.Ports != "" {
				commands = append(commands, fmt.Sprintf("ufw allow in to any port %s proto %s", rule.Ports, rule.Protocol))
			}
		} else {
			if rule.Ports != "" {
				commands = append(commands, fmt.Sprintf("ufw allow in to any port %s", rule.Ports))
			}
		}
	}
	return commands
}

func GenerateIptablesCommands(rule FirewallRule) []string {
	ipAddr := ""
	if rule.SourceIP != "" {
		ipAddr = resolveIP(rule.SourceIP)
	}
	portsList := strings.Split(rule.Ports, ",")
	//nolint:revive // Ignoring due to specific need for this structure
	if rule.Protocol != "" {
		if len(portsList) > 0 && portsList[0] != "" {
			commands := []string{}
			for _, port := range portsList {
				port = strings.TrimSpace(port)
				commands = append(commands, generateIptablesForProtocol(rule.Protocol, ipAddr, port))
			}
			return commands
		} else {
			if ipAddr != "" {
				return []string{fmt.Sprintf("iptables -A INPUT -p %s -s %s -j ACCEPT", rule.Protocol, ipAddr)}
			}
			return []string{fmt.Sprintf("iptables -A INPUT -p %s -j ACCEPT", rule.Protocol)}
		}
	} else {
		if len(portsList) > 0 && portsList[0] != "" {
			commands := generateIptablesForPorts(portsList, ipAddr)
			return commands
		} else {
			if ipAddr != "" {
				return []string{fmt.Sprintf(
					"iptables -A INPUT -p tcp -s %s -j ACCEPT && iptables -A INPUT -p udp -s %s -j ACCEPT",
					ipAddr, ipAddr)}
			}
			return []string{}
		}
	}
}

func resolveIP(sourceIP string) string {
	ip := net.ParseIP(sourceIP)
	if ip == nil {
		return "$(dig +short " + sourceIP + " | tail -n1)"
	}
	return sourceIP
}

func generateIptablesForProtocol(protocol, ipAddr, port string) string {
	if ipAddr != "" {
		return fmt.Sprintf("iptables -A INPUT -p %s -s %s --dport %s -j ACCEPT", protocol, ipAddr, port)
	}
	return fmt.Sprintf("iptables -A INPUT -p %s --dport %s -j ACCEPT", protocol, port)
}

func generateIptablesForPorts(portsList []string, ipAddr string) []string {
	commands := []string{}
	for _, port := range portsList {
		port = strings.TrimSpace(port)
		if ipAddr != "" {
			//nolint:gocritic
			commands = append(commands, fmt.Sprintf("iptables -A INPUT -p tcp -s %s --dport %s -j ACCEPT", ipAddr, port))
			commands = append(commands, fmt.Sprintf("iptables -A INPUT -p udp -s %s --dport %s -j ACCEPT", ipAddr, port))
		} else {
			//nolint:gocritic
			commands = append(commands, fmt.Sprintf("iptables -A INPUT -p tcp --dport %s -j ACCEPT", port))
			commands = append(commands, fmt.Sprintf("iptables -A INPUT -p udp --dport %s -j ACCEPT", port))
		}
	}
	return commands
}

// ParseJSONFirewallRules parse the firewall rule provided as JSON, expected JSON is expected to
// follow the JSON defined by FirewallRule struct. Exported for testing purposes.
func ParseJSONFirewallRules(rulesStr string) ([]FirewallRule, error) {
	if rulesStr == "" {
		return make([]FirewallRule, 0), nil
	}
	var rules []FirewallRule
	err := json.Unmarshal([]byte(rulesStr), &rules)
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msg("Failed to unmarshal firwall rules")
		return nil, err
	}
	return rules, nil
}
