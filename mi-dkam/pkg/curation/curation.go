// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package curation

import (
	"bufio"
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

	if MODE == "dev" {
		fileServer = config.DevFileServer
		harborServer = config.DevHarbor
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

	// Substitute relevant data in the script
	//modifiedScript := strings.ReplaceAll(string(content), "__SUBSTITUTE_PACKAGE_COMMANDS__", packages)
	modifiedScript := strings.ReplaceAll(string(content), "__FILE_SERVER__", fileServer)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__AUTH_SERVER__", config.AuthServer)
	modifiedScript = strings.ReplaceAll(modifiedScript, "__GPGKey__", config.GPGKey)

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

	if MODE == "dev" {
		//Add proxies to the installer script for dev environment.
		newLines := []string{"echo 'http_proxy=http://proxy-dmz.intel.com:911' >> /etc/environment;",
			"echo 'https_proxy=http://proxy-dmz.intel.com:912' >> /etc/environment;",
			"echo 'ftp_proxy=http://proxy-dmz.intel.com:911' >> /etc/environment;",
			"echo 'socks_server=http://proxy-dmz.intel.com:1080' >> /etc/environment;",
			"echo 'no_proxy=localhost,*.intel.com,*intel.com,127.0.0.1,intel.com' >> /etc/environment;",
			" . /etc/environment;",
			"export http_proxy https_proxy ftp_proxy socks_server no_proxy;",
		}
		AddProxies(scriptFileName, newLines)

	}
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
		if strings.TrimSpace(line) == "#!/bin/bash" {
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
