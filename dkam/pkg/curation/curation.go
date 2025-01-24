// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package curation

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/errors"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/pkg/util"

	"github.com/Masterminds/sprig/v3"
	"gopkg.in/yaml.v2"

	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/api/os/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/logging"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/pkg/config"
)

var zlog = logging.GetLogger("MIDKAMAuth")

var agentsList []AgentsVersion
var distribution string

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
	Metadata struct {
		DebianRepositories []struct {
			Name         string `yaml:"name"`
			URL          string `yaml:"url"`
			Architecture string `yaml:"architecture"`
			Key          string `yaml:"key"`
			Section      string `yaml:"section"`
			Distribution string `yaml:"distribution"`
			Root         string `yaml:"root"`
			ThirdParty   bool   `yaml:"thirdParty"`
			AuthType     string `yaml:"authType"`
		} `yaml:"debianRepositories"`
	} `yaml:"metadata"`
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

func GetArtifactsVersion() ([]AgentsVersion, error) {
	releaseFilePath, err := util.GetReleaseFilePathIfExists()
	if err != nil {
		return nil, err
	}

	configs, err := GetReleaseArtifactList(releaseFilePath)
	if err != nil {
		zlog.InfraSec().Info().Msgf("Error checking path %v", err)
		return []AgentsVersion{}, err
	}
	agentsList = []AgentsVersion{}
	agentsList = append(agentsList, configs.BMA.Debs...)

	distribution = configs.Metadata.DebianRepositories[0].Distribution

	zlog.InfraSec().Info().Msgf("Agents List' %s", agentsList)
	if len(agentsList) == 0 {
		zlog.InfraSec().Info().Msg("Failed to get the agent list")
		return []AgentsVersion{}, err
	}

	return agentsList, nil
}

func getCaCert() (string, error) {
	caPath := config.OrchCACertificateFile
	caexists, err := util.PathExists(caPath)
	if err != nil {
		errMsg := "Failed to check if CA certificate path exists"
		zlog.Error().Err(err).Msg(errMsg)
		return "", inv_errors.Errorf(errMsg)
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

func getCustomFirewallRules() ([]string, error) {
	// Parse each rule map into a Rule struct
	rules, err := ParseJSONUfwRules(os.Getenv("FIREWALL_REQ_ALLOW"))
	if err != nil {
		return nil, err
	}
	rules2, err := ParseJSONUfwRules(os.Getenv("FIREWALL_CFG_ALLOW"))
	if err != nil {
		return nil, err
	}
	// TODO: refactor this code to generate UFW or iptables rules, depending on the input variable
	ipTablesCommands := make([]string, 0)
	for _, rule := range append(rules, rules2...) {
		ipTablesCommands = append(ipTablesCommands, GenerateIptablesCommands(rule)...)
	}

	return ipTablesCommands, nil
}

func getScriptTemplateVariables() (map[string]interface{}, error) {
	caCert, err := getCaCert()
	if err != nil {
		return nil, err
	}

	firewallRules, err := getCustomFirewallRules()
	if err != nil {
		return nil, err
	}

	templateVariables := map[string]interface{}{
		"MODE": os.Getenv("MODE"),

		"CA_CERT": caCert,

		"ORCH_CLUSTER":                   os.Getenv("ORCH_CLUSTER"),
		"ORCH_INFRA":                     os.Getenv("ORCH_INFRA"),
		"ORCH_UPDATE":                    os.Getenv("ORCH_UPDATE"),
		"ORCH_PLATFORM_OBS_HOST":         os.Getenv("ORCH_PLATFORM_OBS_HOST"),
		"ORCH_PLATFORM_OBS_PORT":         os.Getenv("ORCH_PLATFORM_OBS_PORT"),
		"ORCH_PLATFORM_OBS_METRICS_HOST": os.Getenv("ORCH_PLATFORM_OBS_METRICS_HOST"),
		"ORCH_PLATFORM_OBS_METRICS_PORT": os.Getenv("ORCH_PLATFORM_OBS_METRICS_PORT"),
		"ORCH_TELEMETRY_HOST":            os.Getenv("ORCH_TELEMETRY_HOST"),
		"ORCH_TELEMETRY_PORT":            os.Getenv("ORCH_TELEMETRY_PORT"),
		"KEYCLOAK_URL":                   os.Getenv("ORCH_KEYCLOAK"),
		"RELEASE_TOKEN_URL":              os.Getenv("ORCH_RELEASE"),
		"ORCH_APT_PORT":                  os.Getenv("ORCH_APT_PORT"),
		"ORCH_IMG_PORT":                  os.Getenv("ORCH_IMG_PORT"),
		"FILE_SERVER":                    os.Getenv("FILE_SERVER"),
		"IMG_REGISTRY_URL":               os.Getenv("REGISTRY_SERVICE"),
		"NTP_SERVERS":                    os.Getenv("NTP_SERVERS"),

		"EN_HTTP_PROXY":  os.Getenv("EN_HTTP_PROXY"),
		"EN_HTTPS_PROXY": os.Getenv("EN_HTTPS_PROXY"),
		"EN_NO_PROXY":    os.Getenv("EN_NO_PROXY"),
		"EN_FTP_PROXY":   os.Getenv("EN_FTP_PROXY"),
		"EN_SOCKS_PROXY": os.Getenv("EN_SOCKS_PROXY"),

		"EXTRA_HOSTS": strings.Split(os.Getenv("EXTRA_HOSTS"), ","),

		"IPTABLES_RULES": firewallRules,
	}
	return templateVariables, nil
}

func CurateScript(osRes *osv1.OperatingSystemResource) error {
	installerScriptPath, err := util.GetInstallerLocation(osRes, config.PVC)
	if err != nil {
		return err
	}

	agentsList, err := GetArtifactsVersion()
	zlog.InfraSec().Info().Msgf("Agents List' %s", agentsList)
	if len(agentsList) == 0 {
		zlog.InfraSec().Info().Msg("Failed to get the agent list")
		return err
	}

	if osRes.GetOsType() == osv1.OsType_OS_TYPE_IMMUTABLE {
		templateVariables, err := getScriptTemplateVariables()
		if err != nil {
			zlog.InfraSec().Error().Err(err).Msg("Failed to get template variables for curation")
			return err
		}

		curatedScriptData, createErr := CurateScriptFromTemplate(config.ScriptPath, templateVariables)
		if createErr != nil {
			zlog.InfraSec().Error().Msgf("Error checking path %v", createErr)
			return createErr
		}

		writeErr := WriteFileToPath(installerScriptPath, []byte(curatedScriptData))
		if writeErr != nil {
			zlog.InfraSec().Error().Err(writeErr).Msgf("Failed to write file to path %s", installerScriptPath)
			return writeErr
		}
	} else {
		createErr := CreateOverlayScript(osRes)
		if createErr != nil {
			zlog.InfraSec().Info().Msgf("Error checking path %v", createErr)
			return createErr
		}
	}
	return nil

}

func GetReleaseArtifactList(filePath string) (Config, error) {

	// Open the file
	zlog.InfraSec().Info().Msg("Inside GetReleaseArtifactList")
	zlog.InfraSec().Info().Msg(filePath)

	file, err := os.Open(filePath)
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msgf("Error opening file: %v", err)
		return Config{}, err
	}
	defer file.Close()

	// Read the content of the file
	content, err := io.ReadAll(file)
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msgf("Error reading file: %v", err)
		return configs, err
	}
	// Unmarshal the YAML content into the Config struct

	err = yaml.Unmarshal(content, &configs)
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msgf("Error unmarshalling YAML: %v", err)
		return configs, err
	}
	return configs, nil
}

// TODO: this function is intended to be generic, so in future we can use it to render Installer script for Ubuntu as well.
func CurateScriptFromTemplate(scriptTemplatePath string, templateVariables map[string]interface{}) (string, error) {
	cfgFilePath := filepath.Join(scriptTemplatePath, "Installer.cfg")

	// Read the template of cloud-init script
	tmplCloudInit, err := os.ReadFile(cfgFilePath)
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msgf(
			"Failed to read template of cloud-init script from path %v", scriptTemplatePath)
		return "", err
	}

	// Parse and execute the template
	// We use sprig to extend basic Go's text/template with more powerful keywords
	// See: https://masterminds.github.io/sprig/
	// This function will fail if any of keys is not provided
	t, err := template.New("yaml").Option("missingkey=error").Funcs(sprig.TxtFuncMap()).Parse(string(tmplCloudInit))
	if err != nil {
		invErr := inv_errors.Errorf("Failed to parse cloud-init template")
		zlog.Error().Err(err).Msg(invErr.Error())
		return "", invErr
	}

	var rendered bytes.Buffer
	if renderErr := t.Execute(&rendered, templateVariables); renderErr != nil {
		invErr := inv_errors.Errorf("Failed to render cloud-init script")
		zlog.Error().Err(renderErr).Msg(invErr.Error())
		return "", invErr
	}

	return rendered.String(), nil
}

func WriteFileToPath(filePath string, content []byte) error {
	zlog.Debug().Msgf("Writing data to path %s", filePath)

	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msg("")
		return inv_errors.Errorf("Failed to create sub-directories to save file")
	}

	err = os.WriteFile(filePath, content, 0644)
	if err != nil {
		errMsg := "Failed save the data to output path"
		zlog.Error().Err(err).Msg(errMsg)
		return inv_errors.Errorf(errMsg)
	}

	return nil
}

func CreateOverlayScript(osRes *osv1.OperatingSystemResource) error {
	MODE := os.Getenv("MODE")
	zlog.InfraSec().Info().Msgf("MODE: %s", MODE)

	beginString := "true >/etc/environment"
	scriptDir := config.ScriptPath
	zlog.InfraSec().Info().Msg(scriptDir)
	installerPath := filepath.Join(scriptDir, "Installer")
	scriptFileName := ""

	exists, err := util.PathExists(config.PVC)
	if err != nil {
		zlog.InfraSec().Info().Msgf("Error checking path %v", err)
	}
	if exists {
		zlog.InfraSec().Info().Msg("Path exists:")

		scriptFileName, err = util.GetInstallerLocation(osRes, config.PVC)
		if err != nil {
			return err
		}

		dir := filepath.Dir(scriptFileName)
		if err := os.MkdirAll(dir, 0755); err != nil {
			zlog.InfraSec().Info().Msgf("Error creating path %v", err)
		}
	} else {
		zlog.InfraSec().Info().Msg("Path does not exists")
	}

	cpErr := copyFile(installerPath, scriptFileName)
	if cpErr != nil {
		zlog.InfraSec().Error().Err(cpErr).Msgf("Error: %v", cpErr)
	}

	zlog.InfraSec().Info().Msg("File copied successfully.")

	profileName := osRes.GetProfileName()

	profileExists, err := util.PathExists(config.DownloadPath + "/" + profileName + ".sh")
	if err != nil {
		zlog.InfraSec().Info().Msgf("Error checking path %v", err)
	}

	if profileExists {
		// Read the source file content
		sourceContent, err := os.Open(config.DownloadPath + "/" + profileName + ".sh")
		if err != nil {
			zlog.InfraSec().Info().Msgf("Error reading donwloaded profile script file:%v", err)
			return err
		}
		defer sourceContent.Close()

		destinationFile, err := os.OpenFile(scriptFileName, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			zlog.InfraSec().Info().Msgf("Error opening installer.sh script:%v", err)
			return err
		}
		defer destinationFile.Close()

		reader := bufio.NewReader(sourceContent)
		_, err = io.Copy(destinationFile, reader)
		if err != nil {
			zlog.InfraSec().Info().Msgf("Error appending profile script to installer.sh:%v", err)
			return err
		}

		zlog.InfraSec().Info().Msg("Contents appended successfully!")
	} else {
		zlog.InfraSec().Info().Msg("Use default profile.")
	}

	// Read the installer
	content, err := os.ReadFile(scriptFileName)
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msgf("Error %v", err)
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

	fileServer := os.Getenv("FILE_SERVER")
	registryService := os.Getenv("REGISTRY_SERVICE")

	//Proxies
	httpProxy := os.Getenv("EN_HTTP_PROXY")
	httpsProxy := os.Getenv("EN_HTTPS_PROXY")
	noProxy := os.Getenv("EN_NO_PROXY")
	ftpProxy := os.Getenv("EN_FTP_PROXY")
	sockProxy := os.Getenv("EN_SOCKS_PROXY")

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
	caexists, err := util.PathExists(config.OrchCACertificateFile)
	if err != nil {
		zlog.InfraSec().Info().Msgf("Error checking path %v", err)
		zlog.InfraSec().Error().Err(err).Msgf("Error: %v", err)
	}

	var caContent []byte
	if caexists {
		caContent, err = os.ReadFile(config.OrchCACertificateFile)
		if err != nil {
			zlog.InfraSec().Error().Msgf("Error: %v", err)
		}
	}

	// Substitute relevant data in the script
	//modifiedScript := strings.ReplaceAll(string(content), "__SUBSTITUTE_PACKAGE_COMMANDS__", packages)
	modifiedScript := strings.ReplaceAll(string(content), "__REGISTRY_URL__", registryService)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__FILE_SERVER__", fileServer)
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
	modifiedScript = strings.ReplaceAll(modifiedScript, "__APT_SRC__", string(distribution))

	// Loop through the agentsList
	for _, agent := range agentsList {
		// Access the fields of each struct
		zlog.InfraSec().Info().Msgf("Package: %s, Version: %s\n", agent.Package, agent.Version)
		modifiedScript = strings.ReplaceAll(modifiedScript, agent.Package+"-VERSION", agent.Version)
	}

	//netplan
	netip_enable_flag := os.Getenv("NETIP")
	// Name of the function to remove
	functionToRemove := "enable_netipplan"

	// Find the start and end positions of the function
	startIdx := strings.Index(modifiedScript, functionToRemove)
	if startIdx == -1 {
		zlog.InfraSec().Info().Msg("Function not found in script")
	}
	endIdx := strings.Index(modifiedScript[startIdx:], "}") + startIdx
	if endIdx == -1 {
		zlog.InfraSec().Info().Msg("Function end not found in script")
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
		zlog.InfraSec().Error().Err(err).Msgf("Error: %v", err)
	}

	functionToRemove = "install_intel_CAcertificates"
	// Find the start and end positions of the function
	startIdx = strings.Index(modifiedScript, functionToRemove)
	if startIdx == -1 {
		zlog.InfraSec().Info().Msg("Function not found in script")
	}
	endIdx = strings.Index(modifiedScript[startIdx:], "}") + startIdx
	if endIdx == -1 {
		zlog.InfraSec().Info().Msg("Function end not found in script")
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
		zlog.InfraSec().Error().Err(err).Msgf("Error: %v", err)
	}

	var newLines []string
	var kindLines []string
	//check if its a kind cluster
	if strings.Contains(orchCluster, "kind.internal") {
		zlog.InfraSec().Info().Msg("Its a kind cluster")
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
		zlog.InfraSec().Info().Msg("Its not a kind cluster")
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
		if httpProxy != "" {
			newLines = append(newLines, "if ! grep -q \"http_proxy\" /etc/environment; then")
			newLines = append(newLines, "    echo \"http_proxy=$http_proxy\" >> /etc/environment;")
			newLines = append(newLines, "fi")
		}

		if httpsProxy != "" {
			newLines = append(newLines, "if ! grep -q \"https_proxy\" /etc/environment; then")
			newLines = append(newLines, "    echo \"https_proxy=$https_proxy\" >> /etc/environment;")
			newLines = append(newLines, "fi")
		}

		if ftpProxy != "" {
			newLines = append(newLines, "if ! grep -q \"ftp_proxy\" /etc/environment; then")
			newLines = append(newLines, "    echo \"ftp_proxy=$ftp_proxy\" >> /etc/environment;")
			newLines = append(newLines, "fi")
		}

		if sockProxy != "" {
			newLines = append(newLines, "if ! grep -q \"socks_proxy\" /etc/environment; then")
			newLines = append(newLines, "    echo \"socks_server=$socks_proxy\" >> /etc/environment;")
			newLines = append(newLines, "fi")
		}

		if noProxy != "" {
			newLines = append(newLines, "if ! grep -q \"no_proxy\" /etc/environment; then")
			newLines = append(newLines, "    echo \"no_proxy=$no_proxy\" >> /etc/environment;")
			newLines = append(newLines, "fi")
		}
		newLines = append(newLines, "    echo \"Proxies added to /etc/environment.\"")
		newLines = append(newLines, ". /etc/environment;")
		newLines = append(newLines, "export http_proxy https_proxy ftp_proxy socks_server no_proxy;")

	}

	AddProxies(scriptFileName, newLines, beginString)

	zlog.InfraSec().Debug().Msgf("Starting modifying ufw Rules")

	// Parse each rule map into a Rule struct
	rules, err := ParseJSONUfwRules(firewallReqAllow)
	if err != nil {
		zlog.InfraSec().InfraErr(err).Msgf("Error while un-marshaling the UFW req firewall Rules")
	}
	rules2, err := ParseJSONUfwRules(firewallCfgAllow)
	if err != nil {
		zlog.InfraSec().InfraErr(err).Msgf("Error while un-marshaling the UFW cfg firewall Rules")
	}
	ufwCommands := make([]string, len(rules)+len(rules2))
	for i, rule := range append(rules, rules2...) {
		ufwCommands[i] = "    " + GenerateUFWCommand(rule)
	}
	AddFirewallRules(scriptFileName, ufwCommands)

	return nil
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
		zlog.InfraSec().Error().Err(err).Msgf("Error: %v", err)
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
		zlog.InfraSec().Error().Err(err).Msgf("Error: %v", err)
		return
	}

	// If the target line was not found, return an error
	if !foundTargetLine {
		zlog.InfraSec().Error().Err(err).Msgf("target line '%s' not found in the file", beginLine)
	}

	// Write the modified content back to the file
	err = os.WriteFile(fileName, []byte(strings.Join(lines, "\n")), 0644)
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msgf("Error: %v", err)
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

func GenerateIptablesCommands(rule Rule) []string {
	ipAddr := ""
	if rule.SourceIp != "" {
		ip := net.ParseIP(rule.SourceIp)
		if ip == nil {
			ipAddr = "$(dig +short " + rule.SourceIp + " | tail -n1)"
		} else {
			ipAddr = rule.SourceIp
		}
	}
	portsList := strings.Split(rule.Ports, ",")
	if rule.Protocol != "" {
		if len(portsList) > 0 && portsList[0] != "" {
			commands := []string{}
			for _, port := range portsList {
				port = strings.TrimSpace(port)
				if ipAddr != "" {
					commands = append(commands, fmt.Sprintf("iptables -A INPUT -p %s -s %s --dport %s -j ACCEPT", rule.Protocol, ipAddr, port))
				} else {
					commands = append(commands, fmt.Sprintf("iptables -A INPUT -p %s --dport %s -j ACCEPT", rule.Protocol, port))
				}
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
			commands := []string{}
			for _, port := range portsList {
				port = strings.TrimSpace(port)
				if ipAddr != "" {
					commands = append(commands, fmt.Sprintf("iptables -A INPUT -p tcp -s %s --dport %s -j ACCEPT", ipAddr, port))
					commands = append(commands, fmt.Sprintf("iptables -A INPUT -p udp -s %s --dport %s -j ACCEPT", ipAddr, port))
				} else {
					commands = append(commands, fmt.Sprintf("iptables -A INPUT -p tcp --dport %s -j ACCEPT", port))
					commands = append(commands, fmt.Sprintf("iptables -A INPUT -p udp --dport %s -j ACCEPT", port))
				}
			}
			return commands
		} else {
			if ipAddr != "" {
				return []string{fmt.Sprintf("iptables -A INPUT -p tcp -s %s -j ACCEPT && iptables -A INPUT -p udp -s %s -j ACCEPT", ipAddr, ipAddr)}
			}
			return []string{}
		}
	}
}

func AddFirewallRules(fileName string, newLines []string) {
	// Read the content of the file
	file, err := os.Open(fileName)
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msgf("Error: %v", err)
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
		zlog.InfraSec().Error().Err(err).Msgf("target line '%s' not found in the file", "#!/bin/bash")
	}

	// Write the modified content back to the file
	err = os.WriteFile(fileName, []byte(strings.Join(lines, "\n")), 0644)
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msgf("Error: %v", err)
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
		zlog.InfraSec().Error().Err(err).Msg("Failed to unmarshal firwall rules")
		return nil, err
	}
	return rules, nil
}
