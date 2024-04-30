// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package signing

import (
	"errors"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
)

var zlog = logging.GetLogger("MIDKAMAuth")

func SignHookOS(scriptPath string) (bool, error) {

	zlog.MiSec().Info().Msgf("Script dir %s", scriptPath)
	buildScriptPath := scriptPath + "/hook"
	zlog.MiSec().Info().Msgf("Hook OS dir %s", buildScriptPath)

	errp := os.Chdir(buildScriptPath)
	if errp != nil {
		zlog.MiSec().Fatal().Err(errp).Msgf("Error changing working directory: %v\n", errp)
		return false, errp
	}

	fileServer := os.Getenv("FILE_SERVER")
	//harborServer := config.DevHarbor
	ReleaseService := os.Getenv("REGISTRY_SERVICE")
	dnsName := os.Getenv("DNS_NAME")
	zlog.MiSec().Info().Msgf("CDN boot DNS name %s", dnsName)
	parsedURL, parseerr := url.Parse(dnsName)
	if parseerr != nil {
		zlog.MiSec().Fatal().Err(parseerr).Msgf("Error parsing URL: %v", parseerr)
		return false, parseerr
	}

	// Extract the host (including subdomain) from the URL
	host := parsedURL.Hostname()

	zlog.MiSec().Info().Msgf("Domain: %s", host)

	// inputs for config
	keycloak_url := os.Getenv("KEYCLOAK_URL")
	//harbor_url_tinker_actions := harborServer + "/one-intel-edge/edge-node/tinker-actions"
	////////// Proxies **********************************
	http_proxy := os.Getenv("HTTP_PROXY")
	https_proxy := os.Getenv("HTTPS_PROXY")
	ftp_proxy := os.Getenv("FTP_PROXY")
	socks_proxy := os.Getenv("SOCKS_PROXY")
	no_proxy := os.Getenv("NO_PROXY")

	//Name server
	//nameserver := os.Getenv("NAMESERVERS")

	//////////// FQDNS ********************************
	fdo_manufacturer_svc := os.Getenv("FDO_MANUFACTURE_SVC")
	fdo_owner_svc := os.Getenv("FDO_OWNER_SVC")
	release_svc := fileServer
	tink_stack_svc := host
	releaseSVC := os.Getenv("RELEASE_SVC")
	tink_server_svc := os.Getenv("TINKER_SVC")
	oci_release_svc := ReleaseService
	logging_svc := os.Getenv("ORCH_PLATFORM_OBS_HOST")
	extra_hosts := os.Getenv("EXTRA_HOSTS")

	config := buildScriptPath + "/config"
	content, err := os.ReadFile(config)
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error %v", err)
	}

	modifiedConfig := strings.ReplaceAll(string(content), "__http_proxy__", http_proxy)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__https_proxy__", https_proxy)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__ftp_proxy__", ftp_proxy)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__socks_proxy__", socks_proxy)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__no_proxy__", no_proxy)
	//modifiedConfig = strings.ReplaceAll(modifiedConfig, "__nameserver__", nameserver)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__fdo_manufacturer_svc__", fdo_manufacturer_svc)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__fdo_owner_svc__", fdo_owner_svc)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__release_svc__", release_svc)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__tink_stack_svc__", tink_stack_svc)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__tink_server_svc__", tink_server_svc)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__extra_hosts__", extra_hosts)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__releaseSVC__", releaseSVC)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__keycloak_url__", keycloak_url)
	//modifiedConfig = strings.ReplaceAll(modifiedConfig, "__harbor_url_tinker_actions__", harbor_url_tinker_actions)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__oci_release_svc__", oci_release_svc)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__logging_svc__", logging_svc)
	// Write the modified config back to the file
	errconf := os.WriteFile(config, []byte(modifiedConfig), 0644)
	if errconf != nil {
		zlog.MiSec().Fatal().Err(errconf).Msgf("Error writing modified config file: %v", errconf)
	}

	modeCmd := exec.Command("chmod", "+x", "secure_hookos.sh")
	result, modeErr := modeCmd.CombinedOutput()
	if modeErr != nil {
		zlog.MiSec().Fatal().Err(modeErr).Msgf("Failed to change mode secure_hookos %v", modeErr)
		return false, modeErr
	}
	zlog.Info().Msgf("Script output: %s", string(result))

	cpioPath := buildScriptPath + "/cpio_build"
	zlog.MiSec().Info().Msgf("cpioPath dir %s", cpioPath)

	errcpio := os.Chdir(cpioPath)
	if errcpio != nil {
		zlog.MiSec().Fatal().Err(errcpio).Msgf("Error changing working directory: %v\n", errp)
		return false, errp
	}

	mdCmd := exec.Command("chmod", "+x", "build_image_at_DKAM.sh")
	mdresult, mdErr := mdCmd.CombinedOutput()
	if mdErr != nil {
		zlog.MiSec().Fatal().Err(mdErr).Msgf("Failed to change mode build_image_at_DKAM.sh script %v", mdErr)
		return false, mdErr
	}
	zlog.Info().Msgf("Script output: %s", string(mdresult))

	mode := os.Getenv("MODE")
	if mode == "" {
		mode = "prod"
	}

	allowedValues := []string{"dev", "prod"}
	if !contains(allowedValues, mode) {
		zlog.MiSec().Fatal().Err(mdErr).Msg("Invalid MODE")
		err := errors.New("invalid mode input")
		return false, err
	}

	buildCmd := exec.Command("bash", "./build_image_at_DKAM.sh", mode)
	output, buildErr := buildCmd.CombinedOutput()
	if buildErr != nil {
		zlog.MiSec().Fatal().Err(buildErr).Msgf("Failed to sign microOS script %v", buildErr)
		return false, buildErr
	}
	zlog.Info().Msgf("Script output: %s", string(output))

	errch := os.Chdir(scriptPath)
	if errch != nil {
		zlog.MiSec().Fatal().Err(errch).Msgf("Error changing working directory: %v\n", errch)
		return false, errch
	}

	return true, nil
}

func contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func BuildSignIpxe(scriptPath string, dnsName string) (bool, error) {

	zlog.MiSec().Info().Msgf("CDN boot DNS name %s", dnsName)
	parsedURL, parseerr := url.Parse(dnsName)
	if parseerr != nil {
		zlog.MiSec().Fatal().Err(parseerr).Msgf("Error parsing URL: %v", parseerr)
		return false, parseerr
	}

	// Extract the host (including subdomain) from the URL
	host := parsedURL.Hostname()

	zlog.MiSec().Info().Msgf("Domain: %s", host)

	tinkUrlString := "<TINK_STACK_URL>"
	errp := os.Chdir(scriptPath)
	if errp != nil {
		zlog.MiSec().Fatal().Err(errp).Msgf("Error changing working directory: %v\n", errp)
		return false, errp
	}

	// Copy the file
	cpErr := copyFile("chain.ipxe", "org_chain.ipxe")
	if cpErr != nil {
		zlog.MiSec().Fatal().Err(cpErr).Msgf("Error: %v", cpErr)
	}

	zlog.MiSec().Info().Msg("chain.ipxe File copied successfully.")

	content, err := os.ReadFile("chain.ipxe")
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error %v", err)
	}

	if strings.Contains(string(content), tinkUrlString) {

		// Substitute relevant data in the script
		modifiedScript := strings.ReplaceAll(string(content), tinkUrlString, dnsName)

		// Save the modified script to the specified output path
		err = os.WriteFile("chain.ipxe", []byte(modifiedScript), 0644)
		if err != nil {
			zlog.MiSec().Fatal().Err(err).Msgf("Error: %v", err)
		}
		zlog.Info().Msg("Tink url updated.")
	} else {
		zlog.Info().Msg("Search string not found in the file.")
	}

	modeCmd := exec.Command("chmod", "+x", "build_sign_ipxe.sh")
	result, modeErr := modeCmd.CombinedOutput()
	if modeErr != nil {
		zlog.MiSec().Fatal().Err(modeErr).Msgf("Failed to chnage mode of build_sign_ipxe.sh %v", modeErr)
		return false, modeErr
	}
	zlog.Info().Msgf("Script output: %s", string(result))
	cmd := exec.Command("sh", "./build_sign_ipxe.sh", scriptPath, host)
	zlog.Info().Msgf("signCmd: %s", cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Failed to run build iPXE")
		return false, err
	}
	zlog.Info().Msgf("Script output: %s", string(output))
	return true, nil
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
