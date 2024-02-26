// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package curation

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/config"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/logging"
)

var zlog = logging.GetLogger("MIDKAMAuth")
var fileServer = config.ProdFileServer
var harborServer = config.ProdHarbor
var registryService = config.RegistryServiceProd
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
}

var configs Config

func GetCuratedScript(profile string, platform string) string {
	MODE := os.Getenv("MODE")
	//MODE := "dev"

	if MODE == "dev" || MODE == "preint" {
		fileServer = config.DevFileServer
		harborServer = config.DevHarbor
		registryService = config.RegistryServiceDev
	}
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
	} else {
		zlog.MiSec().Info().Msg("Path not exists:")
		releaseFilePath = filepath.Join(scriptDir, config.ReleaseVersion+".yaml")
	}

	zlog.MiSec().Info().Msg(releaseFilePath)
	configs, err := GetReleaseArtifactList(releaseFilePath)
	agentsList = append(agentsList, configs.BMA.Debs...)
	zlog.MiSec().Info().Msgf("Agents List' %s", agentsList)
	if len(agentsList) == 0 {
		zlog.MiSec().Info().Msg("Failed to get the agent list")
		return err.Error()
	}
	filename := CreateOverlayScript(currentDir, profile, MODE)
	return filename

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

	//packages := strings.Join(packageList, ",")

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
	orchVault := os.Getenv("ORCH_VAULT")
	orchKeycloak := os.Getenv("ORCH_KEYCLOAK")
	orchRelease := os.Getenv("ORCH_RELEASE")
	orchPkiRole := os.Getenv("ORCH_PKI_ROLE")
	orchPkiPath := os.Getenv("ORCH_PKI_PATH")
	// azureUser := os.Getenv("USERNAME")
	// azurePassword := os.Getenv("PASSWORD")
	orchAptSrcPort := os.Getenv("ORCH_APT_PORT")
	orchImgRegProxyPort := os.Getenv("ORCH_IMG_PORT")

	//Proxies
	httpProxy := os.Getenv("HTTP_PROXY")
	httpsProxy := os.Getenv("HTTPS_PROXY")
	noProxy := os.Getenv("NO_PROXY")
	ftpProxy := os.Getenv("FTP_PROXY")
	sockProxy := os.Getenv("SOCKS_PROXY")

	//KEYCLOAK and VAULT
	//keycloak := os.Getenv("KEYCLOAK_URL")
	vault := os.Getenv("VAULT_URL")

	//Extra hosts
	extra_hosts := os.Getenv("EXTRA_HOSTS")

	// Substitute relevant data in the script
	//modifiedScript := strings.ReplaceAll(string(content), "__SUBSTITUTE_PACKAGE_COMMANDS__", packages)
	modifiedScript := strings.ReplaceAll(string(content), "__REGISTRY_URL__", harborServer)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__FILE_SERVER__", fileServer)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__AUTH_SERVER__", config.AuthServer)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__GPGKey__", config.GPGKey)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__ORCH_CLUSTER__", orchCluster)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__ORCH_INFRA__", orchInfra)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__ORCH_UPDATE__", orchUpdate)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__ORCH_PLATFORM_OBS_HOST__", orchPlatformObsHost)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__ORCH_PLATFORM_OBS_PORT__", orchPlatformObsPort)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__ORCH_PLATFORM_OBS_METRICS_HOST__", orchPlatformObsMetricsHost)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__ORCH_PLATFORM_OBS_METRICS_PORT__", orchPlatformObsMetricsPort)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__ORCH_TELEMETRY_HOST__", orchTelemetryHost)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__ORCH_TELEMETRY_PORT__", orchTelemetryPort)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__ORCH_VAULT__", orchVault)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__ORCH_PKI_ROLE__", orchPkiRole)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__ORCH_PKI_PATH__", orchPkiPath)
	// modifiedScript = strings.ReplaceAll(modifiedScript, "__USERNAME__", azureUser)
	// modifiedScript = strings.ReplaceAll(modifiedScript, "__PASSWORD__", azurePassword)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__KEYCLOAK__", strings.Split(orchKeycloak, ":")[0])
	modifiedScript = strings.ReplaceAll(modifiedScript, "__VAULT__", vault)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__RELEASE_FQDN__", strings.Split(orchRelease, ":")[0])
	modifiedScript = strings.ReplaceAll(modifiedScript, "__KEYCLOAK_URL__", orchKeycloak)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__RELEASE_TOKEN_URL__", orchRelease)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__IMG_REGISTRY_URL__", registryService)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__ORCH_APT_PORT__", orchAptSrcPort)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__ORCH_IMG_PORT__", orchImgRegProxyPort)

	// Loop through the agentsList
	for _, agent := range agentsList {
		// Access the fields of each struct
		zlog.MiSec().Info().Msgf("Package: %s, Version: %s\n", agent.Package, agent.Version)
		modifiedScript = strings.ReplaceAll(modifiedScript, agent.Package+"-VERSION", agent.Version)
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
		AddProxies(scriptFileName, kindLines)

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

	AddProxies(scriptFileName, newLines)

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

func AddProxies(fileName string, newLines []string) {
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
		if strings.TrimSpace(line) == "rm /etc/apt/apt.conf" {
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
