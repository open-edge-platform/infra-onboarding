// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package curation

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"net"

	"gopkg.in/yaml.v2"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/config"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
)

var zlog = logging.GetLogger("MIDKAMAuth")

var fileServer string
var registryService string
var agentsList []AgentsVersion

type AgentsVersion struct {
	Package string `yaml:"package"`
	Version string `yaml:"version"`
}

type Config struct {
	Packages struct {
		Debians []string `yaml:"deb_packages"`
	} `yaml:"packages"`
	BMA struct {
		Debs []AgentsVersion `yaml:"debs"`
	} `yaml:"bma"`
	Provisioning struct {
		Images []Image `yaml:"images"`
	} `yaml:"provisioning"`
}

// Rule UFW Firewall structure in JSON, expected to be provided as environment variable.
type Rule struct {
	SourceIp string `json:"sourceIp,omitempty"`
	Ports    string `json:"ports,omitempty"`
	IpVer    string `json:"ipVer,omitempty"`
	Protocol string `json:"protocol,omitempty"`
}

var configs Config

type Image struct {
	Description string `yaml:"description"`
	Registry    string `yaml:"registry"`
	Image       string `yaml:"image"`
	Version     string `yaml:"version"`
}

func GetCuratedScript(profile string, platform string) (string, string) {
	MODE := os.Getenv("MODE")
	//MODE := "dev"

	fileServer = os.Getenv("FILE_SERVER")
	registryService = os.Getenv("REGISTRY_SERVICE")
	zlog.MiSec().Info().Msgf("MODE: %s", MODE)

	//Current dir
	currentDir, err := os.Getwd()
	if err != nil {
		zlog.MiSec().Info().Msg("Error getting current working directory:")

	}
	zlog.MiSec().Info().Msgf("Current working directory: %s", currentDir)
	parentDir := filepath.Dir(filepath.Dir(currentDir))

	scriptDir := filepath.Join(parentDir, "pkg", "script")
	yamlFile := filepath.Join(scriptDir, "tmp", config.ReleaseVersion+".yaml")
	exists, err := PathExists(yamlFile)
	if err != nil {
		zlog.MiSec().Info().Msgf("Error checking path %v", err)
	}
	releaseFilePath := ""
	if exists {
		zlog.MiSec().Info().Msg("Path exists:")
		releaseFilePath = yamlFile
		tmp_yaml_file := filepath.Join(scriptDir, config.ReleaseVersion+".yaml")
		zlog.MiSec().Info().Msg("Remove latest-dev.yaml temp file")
		yamlexists, err := PathExists(tmp_yaml_file)
		if err != nil {
			zlog.MiSec().Info().Msgf("Error checking path %v", err)
		}
		if yamlexists {
			if err := os.Remove(tmp_yaml_file); err != nil {
				zlog.MiSec().Fatal().Err(err).Msgf("Error removing temporary file: latest-dev.yaml: %v", err)
			}
		}
	} else {
		zlog.MiSec().Info().Msg("Path not exists:")
		releaseFilePath = filepath.Join(scriptDir, config.ReleaseVersion+".yaml")
	}

	zlog.MiSec().Info().Msg(releaseFilePath)
	configs, err := GetReleaseArtifactList(releaseFilePath)
	agentsList = []AgentsVersion{}
	agentsList = append(agentsList, configs.BMA.Debs...)
	tinkeractionList := configs.Provisioning.Images
	var tinkeraction_version string
	if len(tinkeractionList) != 0 {
		for _, image := range tinkeractionList {
			if image.Image == "one-intel-edge/edge-node/tinker-actions/client_auth" {
				zlog.MiSec().Info().Msgf("Tinker action:%s", image.Version)
				tinkeraction_version = image.Version
			}
		}
	}

	zlog.MiSec().Info().Msgf("Agents List' %s", agentsList)
	if len(agentsList) == 0 {
		zlog.MiSec().Info().Msg("Failed to get the agent list")
		return err.Error(), "Tinker action version not found"
	}
	if len(tinkeraction_version) == 0 {
		zlog.MiSec().Info().Msg("Failed to get the Tinker action version")
		return err.Error(), "Tinker action version not found"
	}
	filename := CreateOverlayScript(currentDir, profile, MODE)
	return filename, tinkeraction_version

}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil // path exists
	}
	if os.IsNotExist(err) {
		return false, nil // path does not exist
	}
	return false, err // an error occurred (other than not existing)
}

func GetReleaseArtifactList(filePath string) (Config, error) {

	// Open the file
	zlog.MiSec().Info().Msg("Inside GetReleaseArtifactList")
	zlog.MiSec().Info().Msg(filePath)

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			zlog.MiSec().Info().Msg("File not present")
			return configs, err
		}
	}
	defer file.Close()

	// Read the content of the file
	content, err := io.ReadAll(file)
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error reading file: %v", err)
		return configs, err
	}
	// Unmarshal the YAML content into the Config struct

	err = yaml.Unmarshal(content, &configs)
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error unmarshalling YAML: %v", err)
		return configs, err
	}
	return configs, nil
}

func CreateOverlayScript(pwd string, profile string, MODE string) string {
	parentDir := filepath.Dir(filepath.Dir(pwd))
	beginString := "rm /etc/apt/apt.conf"
	scriptDir := filepath.Join(parentDir, "pkg", "script")
	installerPath := filepath.Join(scriptDir, "Installer")
	scriptFileName := ""
	exists, err := PathExists("/data")
	if err != nil {
		zlog.MiSec().Info().Msgf("Error checking path %v", err)
	}
	if exists {
		zlog.MiSec().Info().Msg("Path exists:")
		scriptFileName = "/data/installer.sh"
	} else {
		scriptFileName = "installer.sh"
	}

	// Copy the file
	cpErr := copyFile(installerPath, scriptFileName)
	if cpErr != nil {
		zlog.MiSec().Fatal().Err(cpErr).Msgf("Error: %v", cpErr)
	}

	zlog.MiSec().Info().Msg("File copied successfully.")

	// Read the installer
	content, err := os.ReadFile(scriptFileName)
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error %v", err)
	}
	//Get FQDN names for agents:
	orchCluster := os.Getenv("ORCH_CLUSTER")
	orchInfra := os.Getenv("ORCH_INFRA")
	orchUpdate := os.Getenv("ORCH_UPDATE")
	orchPlatformObsHost := os.Getenv("ORCH_PLATFORM_OBS_HOST")
	orchPlatformObsPort := os.Getenv("ORCH_PLATFORM_OBS_PORT")
	orchPlatformObsMetricsHost := os.Getenv("ORCH_PLATFORM_OBS_METRICS_HOST")
	orchPlatformObsMetricsPort := os.Getenv("ORCH_PLATFORM_OBS_METRICS_PORT")
	orchTelemetryHost := os.Getenv("ORCH_TELEMETRY_HOST")
	orchTelemetryPort := os.Getenv("ORCH_TELEMETRY_PORT")
	orchKeycloak := os.Getenv("ORCH_KEYCLOAK")
	orchRelease := os.Getenv("ORCH_RELEASE")
	orchAptSrcPort := os.Getenv("ORCH_APT_PORT")
	orchImgRegProxyPort := os.Getenv("ORCH_IMG_PORT")

	//Proxies
	httpProxy := os.Getenv("HTTP_PROXY")
	httpsProxy := os.Getenv("HTTPS_PROXY")
	noProxy := os.Getenv("NO_PROXY")
	ftpProxy := os.Getenv("FTP_PROXY")
	sockProxy := os.Getenv("SOCKS_PROXY")

	//Extra hosts
	extra_hosts := os.Getenv("EXTRA_HOSTS")

	//Kernel configurations
	systemConfigVmOverCommitMemory := os.Getenv("OVER_COMMIT_MEMORY")
	systemConfigKernelPanicOnOops := os.Getenv("PANIC_ON_OOPS")
	systemConfigKernelPanic := os.Getenv("KERNEL_PANIC")
	systemConfigFsInotifyMaxUserInstances := os.Getenv("MAX_USER_INSTANCE")

	//NTP configurations
	ntpServer := os.Getenv("NTP_SERVERS")

	//Firewall configurations
	firewallReqAllow := os.Getenv("FIREWALL_REQ_ALLOW")
	zlog.Info().Msg(firewallReqAllow)
	firewallCfgAllow := os.Getenv("FIREWALL_CFG_ALLOW")
	zlog.Info().Msg(firewallCfgAllow)
	caexists, err := PathExists("/etc/ssl/orch-ca-cert/ca.crt")
	if err != nil {
		zlog.MiSec().Info().Msgf("Error checking path %v", err)
		zlog.MiSec().Fatal().Err(err).Msgf("Error: %v", err)
	}

	var caContent []byte
	if caexists {
		caContent, err = os.ReadFile("/etc/ssl/orch-ca-cert/ca.crt")
		if err != nil {
			zlog.MiSec().Error().Msgf("Error: %v", err)
		}
	}

	dockerFilePath := filepath.Join(scriptDir, "docker.key")

	dockerKeyExists, err := PathExists(dockerFilePath)
	if err != nil {
		zlog.MiSec().Info().Msgf("Error checking path %v", err)
		zlog.MiSec().Fatal().Err(err).Msgf("Error: %v", err)
	}

	var dockerContent []byte
	if dockerKeyExists {
		dockerContent, err = os.ReadFile(dockerFilePath)
		if err != nil {
			zlog.MiSec().Error().Msgf("Error: %v", err)
		}
	}

	caddyFilePath := filepath.Join(scriptDir, "gpg.key")

	caddyKeyExists, err := PathExists(caddyFilePath)
	if err != nil {
		zlog.MiSec().Info().Msgf("Error checking path %v", err)
		zlog.MiSec().Fatal().Err(err).Msgf("Error: %v", err)
	}

	var caddyContent []byte
	if caddyKeyExists {
		caddyContent, err = os.ReadFile(caddyFilePath)
		if err != nil {
			zlog.MiSec().Error().Msgf("Error: %v", err)
		}
	}

	// Substitute relevant data in the script
	//modifiedScript := strings.ReplaceAll(string(content), "__SUBSTITUTE_PACKAGE_COMMANDS__", packages)
	modifiedScript := strings.ReplaceAll(string(content), "__REGISTRY_URL__", registryService)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__FILE_SERVER__", fileServer)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__AUTH_SERVER__", config.AuthServer)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__ORCH_CLUSTER__", orchCluster)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__ORCH_INFRA__", orchInfra)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__ORCH_UPDATE__", orchUpdate)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__ORCH_PLATFORM_OBS_HOST__", orchPlatformObsHost)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__ORCH_PLATFORM_OBS_PORT__", orchPlatformObsPort)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__ORCH_PLATFORM_OBS_METRICS_HOST__", orchPlatformObsMetricsHost)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__ORCH_PLATFORM_OBS_METRICS_PORT__", orchPlatformObsMetricsPort)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__ORCH_TELEMETRY_HOST__", orchTelemetryHost)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__ORCH_TELEMETRY_PORT__", orchTelemetryPort)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__KEYCLOAK__", strings.Split(orchKeycloak, ":")[0])
	modifiedScript = strings.ReplaceAll(modifiedScript, "__RELEASE_FQDN__", strings.Split(orchRelease, ":")[0])
	modifiedScript = strings.ReplaceAll(modifiedScript, "__KEYCLOAK_URL__", orchKeycloak)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__RELEASE_TOKEN_URL__", orchRelease)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__IMG_REGISTRY_URL__", registryService)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__ORCH_APT_PORT__", orchAptSrcPort)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__ORCH_IMG_PORT__", orchImgRegProxyPort)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__OVER_COMMIT_MEMORY__", systemConfigVmOverCommitMemory)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__KERNEL_PANIC__", systemConfigKernelPanic)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__PANIC_ON_OOPS__", systemConfigKernelPanicOnOops)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__MAX_USER_INSTANCE__", systemConfigFsInotifyMaxUserInstances)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__NTP_SERVERS__", ntpServer)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__CA_CERT__", string(caContent))
	modifiedScript = strings.ReplaceAll(modifiedScript, "__DOCKER_KEY__", string(dockerContent))
	modifiedScript = strings.ReplaceAll(modifiedScript, "__CADDY_KEY__", string(caddyContent))
	// Loop through the agentsList
	for _, agent := range agentsList {
		// Access the fields of each struct
		zlog.MiSec().Info().Msgf("Package: %s, Version: %s\n", agent.Package, agent.Version)
		modifiedScript = strings.ReplaceAll(modifiedScript, agent.Package+"-VERSION", agent.Version)
	}

	//netplan
	netip_enable_flag := os.Getenv("NETIP")
	// Name of the function to remove
	functionToRemove := "enable_netipplan"

	// Find the start and end positions of the function
	startIdx := strings.Index(modifiedScript, functionToRemove)
	if startIdx == -1 {
		fmt.Println("Function not found in script")
	}
	endIdx := strings.Index(modifiedScript[startIdx:], "}") + startIdx
	if endIdx == -1 {
		fmt.Println("Function end not found in script")
	}

	// Remove the function from the script
	if netip_enable_flag == "static" {
		newcontent := []byte(modifiedScript)
		newScript := bytes.Replace(newcontent, newcontent[startIdx:endIdx+1], []byte{}, 1)
		modifiedScript = string(newScript)
		// Remove any lines containing calls to the function
		lines := strings.Split(string(modifiedScript), "\n")
		for i := range lines {
			if strings.Contains(lines[i], functionToRemove) {
				lines[i] = ""
			}
		}
		modifiedScript = strings.Join(lines, "\n")
	}

	// Save the modified script to the specified output path
	err = os.WriteFile(scriptFileName, []byte(modifiedScript), 0644)
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error: %v", err)
	}

	functionToRemove = "install_intel_CAcertificates"
	// Find the start and end positions of the function
	startIdx = strings.Index(modifiedScript, functionToRemove)
	if startIdx == -1 {
		fmt.Println("Function not found in script")
	}
	endIdx = strings.Index(modifiedScript[startIdx:], "}") + startIdx
	if endIdx == -1 {
		fmt.Println("Function end not found in script")
	}

	// Remove the function from the script
	if MODE == "prod" {
		newcontent := []byte(modifiedScript)
		newScript := bytes.Replace(newcontent, newcontent[startIdx:endIdx+1], []byte{}, 1)
		modifiedScript = string(newScript)
		// Remove any lines containing calls to the function
		lines := strings.Split(string(modifiedScript), "\n")
		for i := range lines {
			if strings.Contains(lines[i], functionToRemove) {
				lines[i] = ""
			}
		}
		modifiedScript = strings.Join(lines, "\n")
	}

	// Save the modified script to the specified output path
	err = os.WriteFile(scriptFileName, []byte(modifiedScript), 0644)
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error: %v", err)
	}

	var newLines []string
	var kindLines []string
	//check if its a kind cluster
	if strings.Contains(orchCluster, "kind.internal") {
		zlog.MiSec().Info().Msg("Its a kind cluster")
		kindLines = append(kindLines, fmt.Sprintf("extra_hosts=\"%s\"", extra_hosts))
		kindLines = append(kindLines, "IFS=',' read -ra hosts <<< \"$extra_hosts\"")
		kindLines = append(kindLines, "for host in \"${hosts[@]}\"; do")
		kindLines = append(kindLines, "    IFS=' ' read -ra parts <<< \"$host\"")
		kindLines = append(kindLines, "    ip=\"${parts[0]}\"")
		kindLines = append(kindLines, "     hostname=\"${parts[1]}\"")
		kindLines = append(kindLines, "     echo \"$ip $hostname\" >> /etc/hosts")
		kindLines = append(kindLines, "done")
		AddProxies(scriptFileName, kindLines, beginString)

	} else {
		zlog.MiSec().Info().Msg("Its not a kind cluster")
	}

	proxies := map[string]string{
		"http_proxy":  httpProxy,
		"https_proxy": httpsProxy,
		"ftp_proxy":   ftpProxy,
		"socks_proxy": sockProxy,
		"no_proxy":    noProxy,
	}

	//Add proxies to the installer script for dev environment.
	if len(proxies) > 0 {

		for key, value := range proxies {
			if value != "" {
				newLines = append(newLines, fmt.Sprintf("%s=\"%s\"", key, value))
			}
		}
		newLines = append(newLines, "if grep -q \"http_proxy\" /etc/environment && grep -q \"https_proxy\" /etc/environment && grep -q \"ftp_proxy\" /etc/environment && grep -q \"no_proxy\" /etc/environment; then")
		newLines = append(newLines, "    echo \"Proxies are already present in /etc/environment.\"")
		newLines = append(newLines, "else")
		newLines = append(newLines, "    echo \"http_proxy=$http_proxy\" >> /etc/environment;")
		newLines = append(newLines, "    echo \"https_proxy=$https_proxy\" >> /etc/environment;")
		newLines = append(newLines, "    echo \"ftp_proxy=$ftp_proxy\" >> /etc/environment;")
		newLines = append(newLines, "    echo \"socks_server=$socks_proxy\" >> /etc/environment;")
		newLines = append(newLines, "    echo \"no_proxy=$no_proxy\" >> /etc/environment;")
		newLines = append(newLines, "    echo \"Proxies added to /etc/environment.\"")
		newLines = append(newLines, "fi")
		newLines = append(newLines, ". /etc/environment;")
		newLines = append(newLines, "export http_proxy https_proxy ftp_proxy socks_server no_proxy;")

	}

	//Disable ssh for production environment
	var sshLines []string
	zlog.MiSec().Info().Msgf("Mode is:%s", MODE)
	if MODE == "prod" {
		zlog.MiSec().Info().Msgf("Mode is:%s", MODE)
		sshLines = append(sshLines, "ssh_config_file=\"/etc/ssh/sshd_config.d/60-cloudimg-settings.conf\"")
		sshLines = append(sshLines, "if [ -f \"$ssh_config_file\" ]; then")
		sshLines = append(sshLines, "  if grep -q \"^PasswordAuthentication yes\" \"$ssh_config_file\"; then")
		sshLines = append(sshLines, "    sed -i 's/^PasswordAuthentication yes/PasswordAuthentication no/' \"$ssh_config_file\"")
		sshLines = append(sshLines, "  else")
		sshLines = append(sshLines, "    echo \"Password-based authentication is already disabled or configured differently.\"")
		sshLines = append(sshLines, "  fi")
		sshLines = append(sshLines, "else")
		sshLines = append(sshLines, "  echo \"SSH configuration file not found: $ssh_config_file\"")
		sshLines = append(sshLines, "fi")
	}

	AddProxies(scriptFileName, sshLines, "ssh_config(){")
	AddProxies(scriptFileName, newLines, beginString)

	zlog.MiSec().Debug().Msgf("Starting modifying ufw Rules")

	// Parse each rule map into a Rule struct
	rules, err := ParseJSONUfwRules(firewallReqAllow)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("Error while un-marshaling the UFW req firewall Rules")
	}
	rules2, err := ParseJSONUfwRules(firewallCfgAllow)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("Error while un-marshaling the UFW cfg firewall Rules")
	}
	ufwCommands := make([]string, len(rules)+len(rules2))
	for i, rule := range append(rules, rules2...) {
		ufwCommands[i] = "    " + GenerateUFWCommand(rule)
	}
	AddFirewallRules(scriptFileName, ufwCommands)

	return scriptFileName
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	if err != nil {
		return err
	}

	return nil
}

func AddProxies(fileName string, newLines []string, beginLine string) {
	// Read the content of the file
	file, err := os.Open(fileName)
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	foundTargetLine := false

	// Scan through the file and locate the target line
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)

		// Check if the current line matches the target line
		if strings.TrimSpace(line) == beginLine {
			foundTargetLine = true
			// Insert the new lines after the target line
			lines = append(lines, newLines...)
		}
	}

	if err := scanner.Err(); err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error: %v", err)
		return
	}

	// If the target line was not found, return an error
	if !foundTargetLine {
		zlog.MiSec().Fatal().Err(err).Msgf("target line '%s' not found in the file", "#!/bin/bash")
	}

	// Write the modified content back to the file
	err = os.WriteFile(fileName, []byte(strings.Join(lines, "\n")), 0644)
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error: %v", err)
		return
	}

}

// GenerateUFWCommand convert a Rule into the corresponding ufw command.
func GenerateUFWCommand(rule Rule) string {
	ipAddr := ""
	if rule.SourceIp != "" {
		ip := net.ParseIP(rule.SourceIp)
		if ip == nil {
			ipAddr = "$(dig +short " + rule.SourceIp + " | tail -n1)"
		} else {
			ipAddr = rule.SourceIp
		}
		if rule.Protocol != "" {
			if rule.Ports != "" {
				return fmt.Sprintf("ufw allow from %s to any port %s proto %s", ipAddr, rule.Ports, rule.Protocol)
			} else {
				return fmt.Sprintf("ufw allow from %s proto %s", ipAddr, rule.Protocol)
			}
		} else {
			if rule.Ports != "" {
				return fmt.Sprintf("ufw allow from %s to any port %s", ipAddr, rule.Ports)
			} else {
				return fmt.Sprintf("ufw allow from %s", ipAddr)
			}
		}
	} else {
		if rule.Protocol != "" {
			if rule.Ports != "" {
				return fmt.Sprintf("ufw allow in to any port %s proto %s", rule.Ports, rule.Protocol)
			} else {
				return fmt.Sprintf("echo Firewall rule not set %d", 0)
			}
		} else {
			if rule.Ports != "" {
				return fmt.Sprintf("ufw allow in to any port %s", rule.Ports)
			} else {
				return fmt.Sprintf("echo Firewall rule not set %d", 0)
			}
		}
	}
}

func AddFirewallRules(fileName string, newLines []string) {
	// Read the content of the file
	file, err := os.Open(fileName)
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	foundTargetLine := false

	// Scan through the file and locate the target line
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)

		// Check if the current line matches the target line
		if strings.TrimSpace(line) == "ufw default allow outgoing" {
			foundTargetLine = true
			// Insert the new lines after the target line
			lines = append(lines, newLines...)
		}
	}
	// If the target line was not found, return an error
	if !foundTargetLine {
		zlog.MiSec().Fatal().Err(err).Msgf("target line '%s' not found in the file", "#!/bin/bash")
	}

	// Write the modified content back to the file
	err = os.WriteFile(fileName, []byte(strings.Join(lines, "\n")), 0644)
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error: %v", err)
		return
	}
}

// ParseJSONUfwRules parse the ufw rule provided as JSON, expected JSON is expected to
// follow the JSON defined by Rule struct. Exported for testing purposes.
func ParseJSONUfwRules(ufwRules string) ([]Rule, error) {
	if ufwRules == "" {
		return make([]Rule, 0), nil
	}
	var rules []Rule
	err := json.Unmarshal([]byte(ufwRules), &rules)
	if err != nil {
		return nil, err
	}
	return rules, nil
}
