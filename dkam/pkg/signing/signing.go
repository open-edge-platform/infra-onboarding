// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package signing

import (
	"errors"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/logging"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/pkg/config"
)

var zlog = logging.GetLogger("InfraDKAMAuth")

const (
	fileMode  = 0o755
	writeMode = 0o600
)

func SignHookOS() (bool, error) {
	zlog.InfraSec().Info().Msgf("Script dir %s", config.ScriptPath)
	buildScriptPath, err := setupHookDirectories()
	if err != nil {
		return false, err
	}
	ReleaseService := os.Getenv("REGISTRY_SERVICE")
	dnsName := os.Getenv("DNS_NAME")
	zlog.InfraSec().Info().Msgf("CDN boot DNS name %s", dnsName)
	parsedURL, parseerr := url.Parse(dnsName)
	if parseerr != nil {
		zlog.InfraSec().Fatal().Err(parseerr).Msgf("Error parsing URL: %v", parseerr)
		return false, parseerr
	}
	// Extract the host (including subdomain) from the URL
	host := parsedURL.Hostname()
	zlog.InfraSec().Info().Msgf("Domain: %s", host)
	content, err := os.ReadFile("config")
	if err != nil {
		zlog.InfraSec().Fatal().Err(err).Msgf("Error %v", err)
	}
	modifiedConfig := replaceConfigPlaceholders(content, ReleaseService, host)
	// Write the modified config back to the file
	errconf := os.WriteFile("config", []byte(modifiedConfig), writeMode)
	if errconf != nil {
		zlog.InfraSec().Fatal().Err(errconf).Msgf("Error writing modified config file: %v", errconf)
	}
	modeCmd := exec.Command("chmod", "+x", "secure_hookos.sh")
	result, modeErr := modeCmd.CombinedOutput()
	if modeErr != nil {
		zlog.InfraSec().Fatal().Err(modeErr).Msgf("Failed to change mode secure_hookos %v", modeErr)
		return false, modeErr
	}
	zlog.Info().Msgf("Script output: %s", string(result))
	mode, buildErr := validateAndSetMode(buildScriptPath)
	if buildErr != nil {
		return false, buildErr
	}
	mdCmd := exec.Command("chmod", "+x", "build_image_at_DKAM.sh")
	mdresult, mdErr := mdCmd.CombinedOutput()
	if mdErr != nil {
		zlog.InfraSec().Fatal().Err(mdErr).Msgf("Failed to change mode build_image_at_DKAM.sh script %v", mdErr)
		return false, mdErr
	}
	zlog.Info().Msgf("Script output: %s", string(mdresult))
	buildCmd := exec.Command("bash", "./build_image_at_DKAM.sh", mode, config.DownloadPath)
	output, buildErr := buildCmd.CombinedOutput()
	if buildErr != nil {
		zlog.InfraSec().Fatal().Err(buildErr).Msgf("Failed to sign microOS script %v", buildErr)
		return false, buildErr
	}
	zlog.Info().Msgf("Script output: %s", string(output))
	errch := os.Chdir(config.ScriptPath)
	if errch != nil {
		zlog.InfraSec().Fatal().Err(errch).Msgf("Error changing working directory: %v\n", errch)
		return false, errch
	}
	return true, nil
}

func setupHookDirectories() (string, error) {
	hookDir := config.ScriptPath + "/hook"
	buildScriptPath := config.DownloadPath + "/hook"
	zlog.InfraSec().Info().Msgf("Hook OS dir %s", buildScriptPath)
	mkdirErr := os.MkdirAll(buildScriptPath, fileMode)
	if mkdirErr != nil {
		zlog.InfraSec().Error().Err(mkdirErr).Msgf("Error creating directory: %v", mkdirErr)
		return "", mkdirErr
	}
	if err := copyDir(hookDir, buildScriptPath); err != nil {
		zlog.InfraSec().Info().Msgf("Error copying directory:%v", err)
		return "", err
	}

	errp := os.Chdir(buildScriptPath)
	if errp != nil {
		zlog.InfraSec().Fatal().Err(errp).Msgf("Error changing working directory: %v\n", errp)
		return "", errp
	}

	return buildScriptPath, nil
}

func validateAndSetMode(buildScriptPath string) (string, error) {
	cpioPath := buildScriptPath + "/cpio_build"
	zlog.InfraSec().Info().Msgf("cpioPath dir %s", cpioPath)

	errcpio := os.Chdir(cpioPath)
	if errcpio != nil {
		zlog.InfraSec().Fatal().Err(errcpio).Msgf("Error changing working directory: %v\n", errcpio)
		return "", errcpio
	}

	mode := os.Getenv("MODE")
	if mode == "" {
		mode = "prod"
	}

	allowedValues := []string{"dev", "prod"}
	if !contains(allowedValues, mode) {
		zlog.InfraSec().Fatal().Err(errcpio).Msg("Invalid MODE")
		err := errors.New("invalid mode input")
		return "", err
	}
	return mode, nil
}

func replaceConfigPlaceholders(content []byte, releaseService, host string) string {
	keycloakURL := os.Getenv("KEYCLOAK_URL")
	//nolint:gocritic // might be used in future so, used nolint.
	// harbor_url_tinker_actions := harborServer + "/one-intel-edge/edge-node/tinker-actions"
	////////// Proxies **********************************
	httpProxy := os.Getenv("EN_HTTP_PROXY")
	httpsProxy := os.Getenv("EN_HTTPS_PROXY")
	ftpProxy := os.Getenv("EN_FTP_PROXY")
	socksProxy := os.Getenv("EN_SOCKS_PROXY")
	noProxy := os.Getenv("EN_NO_PROXY")
	// Name server
	// nameserver := os.Getenv("NAMESERVERS")
	//////////// FQDNS ********************************
	releaseSvc := os.Getenv("CDN_SVC")
	tinkStackSvc := host
	releaseSVC := os.Getenv("RELEASE_SVC")
	tinkServerSvc := os.Getenv("TINKER_SVC")
	ociReleaseSvc := releaseService
	loggingSvc := os.Getenv("ORCH_PLATFORM_OBS_HOST")
	extraHosts := os.Getenv("EXTRA_HOSTS")
	onboardingManagerSvc := os.Getenv("OM_SERVICE_URL")
	onboardingStreamSvc := os.Getenv("OM_STREAM_URL")
	modifiedConfig := strings.ReplaceAll(string(content), "__http_proxy__", httpProxy)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__https_proxy__", httpsProxy)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__ftp_proxy__", ftpProxy)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__socks_proxy__", socksProxy)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__no_proxy__", noProxy)
	//nolint:gocritic // might be used in future so, used nolint.
	// modifiedConfig = strings.ReplaceAll(modifiedConfig, "__nameserver__", nameserver)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__release_svc__", releaseSvc)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__tink_stack_svc__", tinkStackSvc)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__tink_server_svc__", tinkServerSvc)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__extra_hosts__", extraHosts)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__releaseSVC__", releaseSVC)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__keycloak_url__", keycloakURL)
	//nolint:gocritic // might be used in future so, used nolint.
	// modifiedConfig = strings.ReplaceAll(modifiedConfig, "__harbor_url_tinker_actions__", harbor_url_tinker_actions)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__oci_release_svc__", ociReleaseSvc)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__logging_svc__", loggingSvc)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__onboarding_manager_svc__", onboardingManagerSvc)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__onboarding_stream_svc__", onboardingStreamSvc)
	return modifiedConfig
}

func contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func copyDir(src, dst string) error {
	// Create the destination directory
	if err := os.MkdirAll(dst, fileMode); err != nil {
		return err
	}

	// Walk through the source directory
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get the relative path to the file or directory
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Create the destination path
		dstPath := filepath.Join(dst, relPath)

		// If it's a directory, create it in the destination
		if info.IsDir() {
			if err := os.MkdirAll(dstPath, fileMode); err != nil {
				return err
			}
		} else {
			// If it's a file, copy it to the destination
			if err := copyFile(path, dstPath); err != nil {
				return err
			}
		}

		return nil
	})
}

func BuildSignIpxe(dnsName string) (bool, error) {
	zlog.InfraSec().Info().Msgf("CDN boot DNS name %s", dnsName)
	parsedURL, parseerr := url.Parse(dnsName)
	if parseerr != nil {
		zlog.InfraSec().Fatal().Err(parseerr).Msgf("Error parsing URL: %v", parseerr)
		return false, parseerr
	}

	// Extract the host (including subdomain) from the URL
	host := parsedURL.Hostname()

	zlog.InfraSec().Info().Msgf("Domain: %s", host)

	tinkURLString := "<TINK_STACK_URL>"
	chainPath := config.ScriptPath + "/" + "chain.ipxe"
	targetChainPath := config.DownloadPath + "/" + "chain.ipxe"
	// Copy the file
	cpErr := copyFile(chainPath, targetChainPath)
	if cpErr != nil {
		zlog.InfraSec().Fatal().Err(cpErr).Msgf("Error: %v", cpErr)
	}

	zlog.InfraSec().Info().Msg("chain.ipxe File copied successfully.")

	content, err := os.ReadFile(targetChainPath)
	if err != nil {
		zlog.InfraSec().Fatal().Err(err).Msgf("Error %v", err)
	}

	if strings.Contains(string(content), tinkURLString) {
		// Substitute relevant data in the script
		modifiedScript := strings.ReplaceAll(string(content), tinkURLString, dnsName)

		// Save the modified script to the specified output path
		err = os.WriteFile(targetChainPath, []byte(modifiedScript), writeMode)
		if err != nil {
			zlog.InfraSec().Fatal().Err(err).Msgf("Error: %v", err)
		}
		zlog.Info().Msg("Tink url updated.")
	} else {
		zlog.Info().Msg("Search string not found in the file.")
	}

	errcpio := os.Chdir(config.ScriptPath)
	if errcpio != nil {
		zlog.InfraSec().Fatal().Err(errcpio).Msgf("Error changing working directory: %v\n", errcpio)
		return false, errcpio
	}
	//nolint:gosec // The script and arguments are trusted and validated before execution.
	cmd := exec.Command("bash", "./build_sign_ipxe.sh", config.DownloadPath)
	zlog.Info().Msgf("signCmd: %s", cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		zlog.InfraSec().Fatal().Err(err).Msg("Failed to run build iPXE")
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
