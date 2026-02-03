// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// Package curation provides functionality for curating and customizing infrastructure configurations.
package curation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/config"
)

var zlog = logging.GetLogger("InfraCuration")

// FirewallRule UFW Firewall structure in JSON, expected to be provided as environment variable.
//
//nolint:tagliatelle // json tags use camelCase to match external API format
type FirewallRule struct {
	SourceIP string `json:"sourceIp,omitempty"`
	Ports    string `json:"ports,omitempty"`
	IPVer    string `json:"ipVer,omitempty"`
	Protocol string `json:"protocol,omitempty"`
}

// GetBMAgentsInfo retrieves bare metal agents information from the manifest.
func GetBMAgentsInfo() (agentsList []config.AgentsVersion, distribution string, err error) {
	configs := config.GetInfraConfig().ENManifest

	agentsList = configs.Packages

	distribution = configs.Repository.Codename

	zlog.InfraSec().Info().Msgf("Agents List' %s", agentsList)

	return agentsList, distribution, nil
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil // path exists
	}
	if os.IsNotExist(err) {
		return false, nil // path does not exist
	}
	return false, err // an error occurred (other than not existing)
}

func getCaCert() (string, error) {
	caPath := config.OrchCACertificateFile
	caexists, err := pathExists(caPath)
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

// Returns the concatenated content of orch-ca-cert and boots-ca-cert as a string.
func getCombinedCaPEM() (string, error) {
	orchCa, err := os.ReadFile(config.OrchCACertificateFile)
	if err != nil {
		return "", fmt.Errorf("failed to read orch-ca-cert: %w", err)
	}
	bootsCa, err := os.ReadFile(config.BootsCaCertificateFile)
	if err != nil {
		return "", fmt.Errorf("failed to read boots-ca-cert: %w", err)
	}
	var b strings.Builder
	b.Write(orchCa)
	b.WriteString("\n")
	b.Write(bootsCa)
	return b.String(), nil
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
		zlog.InfraSec().Info().Msgf("Package: %s, Version: %s\n", agent.Name, agent.Version)
		templateVariables[agent.Name+"-VERSION"] = agent.Version
	}

	templateVariables["APT_DISTRO"] = distro

	return templateVariables, nil
}

// GetCommonInfraTemplateVariables prepares template variables for infrastructure configuration.
func GetCommonInfraTemplateVariables(
	infraConfig config.InfraConfig,
	osType osv1.OsType,
) (map[string]interface{}, error) {
	caCert, err := getCaCert()
	if err != nil {
		return nil, err
	}

	caPem, err := getCombinedCaPEM()
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

		"CA_PEM": caPem,

		"ORCH_CLUSTER":                     infraConfig.ClusterURL,
		"ORCH_INFRA":                       infraConfig.InfraURL,
		"ORCH_UPDATE":                      infraConfig.UpdateURL,
		"ORCH_PLATFORM_OBS_HOST":           strings.Split(infraConfig.LogsObservabilityURL, ":")[0],
		"ORCH_PLATFORM_OBS_PORT":           strings.Split(infraConfig.LogsObservabilityURL, ":")[1],
		"ORCH_PLATFORM_OBS_METRICS_HOST":   strings.Split(infraConfig.MetricsObservabilityURL, ":")[0],
		"ORCH_PLATFORM_OBS_METRICS_PORT":   strings.Split(infraConfig.MetricsObservabilityURL, ":")[1],
		"ORCH_TELEMETRY_HOST":              strings.Split(infraConfig.TelemetryURL, ":")[0],
		"ORCH_TELEMETRY_PORT":              strings.Split(infraConfig.TelemetryURL, ":")[1],
		"KEYCLOAK_URL":                     infraConfig.KeycloakURL,
		"KEYCLOAK_FQDN":                    strings.Split(infraConfig.KeycloakURL, ":")[0],
		"RELEASE_FQDN":                     strings.Split(infraConfig.ReleaseServiceURL, ":")[0],
		"RELEASE_TOKEN_URL":                infraConfig.ReleaseServiceURL,
		"ORCH_APT_PORT":                    strings.Split(infraConfig.FileServerURL, ":")[1],
		"ORCH_IMG_PORT":                    strings.Split(infraConfig.RegistryURL, ":")[1],
		"FILE_SERVER":                      strings.Split(infraConfig.FileServerURL, ":")[0],
		"IMG_REGISTRY_URL":                 strings.Split(infraConfig.RegistryURL, ":")[0],
		"NTP_SERVERS":                      strings.Join(infraConfig.NTPServers, ","),
		"DEB_PACKAGES_REPO":                infraConfig.ENDebianPackagesRepo,
		"FILE_RS_ROOT":                     infraConfig.ENFilesRsRoot,
		"RS_TYPE":                          infraConfig.RSType,
		"ORCH_PLATFORM_MANAGEABILITY_HOST": strings.Split(infraConfig.ManageabilityURL, ":")[0],
		"ORCH_PLATFORM_MANAGEABILITY_PORT": strings.Split(infraConfig.ManageabilityURL, ":")[1],
		"RPS_ADDRESS":                      strings.Split(infraConfig.RPSAddress, ":")[0],
		"SERVICE_CLIENTS":                  strings.Join(infraConfig.ENServiceClients, ","),
		"OUTBOUND_CLIENTS":                 strings.Join(infraConfig.ENOutboundClients, ","),
		"METRICS_ENABLED":                  infraConfig.ENMetricsEnabled,
		"TOKEN_CLIENTS":                    strings.Join(infraConfig.ENTokenClients, ","),

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

		"FIREWALL_RULES": firewallRules,

		// TODO: keeping OS-dependence for now, but will be removed once we reach the final solution
		"IS_MICROVISOR": osType == osv1.OsType_OS_TYPE_IMMUTABLE,

		"DISABLE_CO_PROFILE":   infraConfig.DisableCOProfile,
		"DISABLE_O11Y_PROFILE": infraConfig.DisableO11YProfile,
		"SKIP_OS_PROVISIONING": infraConfig.SkipOSProvisioning,

		"ONBOARDING_SVC_URL":        infraConfig.OnboardingURL,
		"ONBOARDING_STREAM_SVC_URL": infraConfig.OnboardingStreamURL,
	}

	switch osType {
	case osv1.OsType_OS_TYPE_MUTABLE:
		templateVariables["FIREWALL_PROVIDER"] = "ufw"
	case osv1.OsType_OS_TYPE_IMMUTABLE:
		templateVariables["FIREWALL_PROVIDER"] = "iptables"
	default:
		// OS_TYPE_UNSPECIFIED - use default
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

// CurateFromTemplate generates content from a template with the provided variables.
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

// GenerateIptablesCommands generates iptables commands from firewall rules.
func GenerateIptablesCommands(rule FirewallRule) []string {
	ipAddr := ""
	if rule.SourceIP != "" {
		ipAddr = resolveIP(rule.SourceIP)
	}
	portsList := strings.Split(rule.Ports, ",")
	//nolint:revive // Ignoring due to specific need for this structure
	if rule.Protocol != "" {
		if len(portsList) > 0 && portsList[0] != "" {
			commands := []string{} //nolint:prealloc // Size unknown, dynamic allocation needed
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
			commands = append(commands,
				fmt.Sprintf("iptables -A INPUT -p tcp -s %s --dport %s -j ACCEPT", ipAddr, port),
				fmt.Sprintf("iptables -A INPUT -p udp -s %s --dport %s -j ACCEPT", ipAddr, port))
		} else {
			commands = append(commands,
				fmt.Sprintf("iptables -A INPUT -p tcp --dport %s -j ACCEPT", port),
				fmt.Sprintf("iptables -A INPUT -p udp --dport %s -j ACCEPT", port))
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
