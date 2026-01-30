// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package signing

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/config"
)

var zlog = logging.GetLogger("InfraDKAMAuth")

const (
	fileMode  = 0o755
	writeMode = 0o600
)

// SignMicroOS signs the MicroOS kernel image with secure boot keys.
//
//nolint:funlen // Complex signing process, length is justified
func SignMicroOS() (bool, error) {
	zlog.InfraSec().Info().Msgf("Script dir %s", config.ScriptPath)
	buildScriptPath, err := setupUOSDirectories()
	if err != nil {
		return false, err
	}

	infraConfig := config.GetInfraConfig()

	zlog.InfraSec().Info().Msgf("CDN boot DNS name %s", infraConfig.ProvisioningServerURL)
	zlog.InfraSec().Info().Msgf("Domain: %s", infraConfig.ProvisioningService)

	content, err := os.ReadFile("config")
	if err != nil {
		zlog.InfraSec().Fatal().Err(err).Msgf("Error %v", err)
	}
	modifiedConfig := replaceConfigPlaceholders(content)
	// Write the modified config back to the file
	errconf := os.WriteFile("config", []byte(modifiedConfig), writeMode)
	if errconf != nil {
		zlog.InfraSec().Fatal().Err(errconf).Msgf("Error writing modified config file: %v", errconf)
	}
	cpioPath := buildScriptPath + "/cpio_build"
	zlog.InfraSec().Info().Msgf("cpioPath dir %s", cpioPath)

	errcpio := os.Chdir(cpioPath)
	if errcpio != nil {
		zlog.InfraSec().Fatal().Err(errcpio).Msgf("Error changing working directory: %v\n", errcpio)
		return false, errcpio
	}

	modeCmd := exec.CommandContext(context.Background(), "chmod", "+x", "secure_uos.sh")
	result, modeErr := modeCmd.CombinedOutput()
	if modeErr != nil {
		zlog.InfraSec().Fatal().Err(modeErr).Msgf("Failed to change mode secure_uos %v", modeErr)
		return false, modeErr
	}
	zlog.Info().Msgf("Script output: %s", string(result))

	mdCmd := exec.CommandContext(context.Background(), "chmod", "+x", "update_initramfs.sh")
	mdresult, mdErr := mdCmd.CombinedOutput()
	if mdErr != nil {
		zlog.InfraSec().Fatal().Err(mdErr).Msgf("Failed to change mode of update_initramfs.sh script %v", mdErr)
		return false, mdErr
	}
	zlog.Info().Msgf("Script output: %s", string(mdresult))

	// Ensure the working directory is correct before running the script
	if err := verifyWorkingDirectory(cpioPath); err != nil {
		zlog.InfraSec().Fatal().Err(err).Msgf("Working directory verification failed: %v", err)
		return false, err
	}

	//nolint:gosec // The script and arguments are trusted and validated before execution.
	buildCmd := exec.CommandContext(context.Background(), "bash", "./update_initramfs.sh", config.DownloadPath)
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

func verifyWorkingDirectory(expected string) error {
	wd, err := os.Getwd()
	if err != nil {
		zlog.InfraSec().Fatal().Err(err).Msgf("Error getting current working directory: %v", err)
		return err
	}
	if wd != expected {
		zlog.InfraSec().Fatal().Msgf("Working directory mismatch: expected %s, got %s", expected, wd)
		return os.ErrInvalid
	}
	return nil
}

func setupUOSDirectories() (string, error) {
	uosDir := config.ScriptPath + "/uos"
	buildScriptPath := config.DownloadPath + "/uos"
	zlog.InfraSec().Info().Msgf("UOS dir %s", buildScriptPath)
	if _, err := os.Stat(buildScriptPath); err == nil {
		removeErr := os.RemoveAll(buildScriptPath)
		if removeErr != nil {
			zlog.InfraSec().Error().Err(removeErr).Msgf("Error removing existing directory: %v", removeErr)
			return "", removeErr
		}
	}
	mkdirErr := os.MkdirAll(buildScriptPath, fileMode)
	if mkdirErr != nil {
		zlog.InfraSec().Error().Err(mkdirErr).Msgf("Error creating directory: %v", mkdirErr)
		return "", mkdirErr
	}
	if err := copyDir(uosDir, buildScriptPath); err != nil {
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

func replaceConfigPlaceholders(content []byte) string {
	infraConfig := config.GetInfraConfig()

	modifiedConfig := strings.ReplaceAll(string(content), "__http_proxy__", infraConfig.ENProxyHTTP)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__https_proxy__", infraConfig.ENProxyHTTPS)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__ftp_proxy__", infraConfig.ENProxyFTP)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__socks_proxy__", infraConfig.ENProxySocks)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__no_proxy__", infraConfig.ENProxyNoProxy)

	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__release_svc__", infraConfig.CDN)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__tink_stack_svc__", infraConfig.ProvisioningService)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__tink_server_svc__", infraConfig.TinkServerURL)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__keycloak_url__", infraConfig.KeycloakURL)
	modifiedConfig = strings.ReplaceAll(
		modifiedConfig,
		"__oci_release_svc__",
		strings.Split(infraConfig.RegistryURL, ":")[0],
	)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__logging_svc__",
		strings.Split(infraConfig.LogsObservabilityURL, ":")[0])
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__onboarding_manager_svc__", infraConfig.OnboardingURL)
	modifiedConfig = strings.ReplaceAll(modifiedConfig, "__onboarding_stream_svc__", infraConfig.OnboardingStreamURL)

	return modifiedConfig
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

// BuildSignIpxe builds and signs the iPXE bootloader with secure boot keys.
func BuildSignIpxe() (bool, error) {
	provisioningServerURL := config.GetInfraConfig().ProvisioningServerURL
	zlog.InfraSec().Info().Msgf("CDN boot DNS name %s", provisioningServerURL)
	zlog.InfraSec().Info().Msgf("Domain: %s", config.GetInfraConfig().ProvisioningService)

	tinkURLString := "<TINK_STACK_URL>"
	ipxePath := config.ScriptPath + "/ipxe"
	chainPath := ipxePath + "/" + "chain.ipxe"
	targetChainPath := config.DownloadPath + "/" + "chain.ipxe"
	// Copy the file
	cpErr := copyFile(chainPath, targetChainPath)
	if cpErr != nil {
		zlog.InfraSec().Fatal().Err(cpErr).Msgf("Error: %v", cpErr)
	}

	zlog.InfraSec().Info().Msg("chain.ipxe File copied successfully.")

	content, err := os.ReadFile(targetChainPath) //nolint:gosec // Path is from trusted config
	if err != nil {
		zlog.InfraSec().Fatal().Err(err).Msgf("Error %v", err)
	}

	if strings.Contains(string(content), tinkURLString) {
		// Substitute relevant data in the script
		modifiedScript := strings.ReplaceAll(string(content), tinkURLString, provisioningServerURL)

		// Save the modified script to the specified output path
		err = os.WriteFile(targetChainPath, []byte(modifiedScript), writeMode)
		if err != nil {
			zlog.InfraSec().Fatal().Err(err).Msgf("Error: %v", err)
		}
		zlog.Info().Msg("Tink url updated.")
	} else {
		zlog.Info().Msg("Search string not found in the file.")
	}

	errIpxe := os.Chdir(ipxePath)
	if errIpxe != nil {
		zlog.InfraSec().Fatal().Err(errIpxe).Msgf("Error changing working directory: %v\n", errIpxe)
		return false, errIpxe
	}
	//nolint:gosec // The script and arguments are trusted and validated before execution.
	cmd := exec.CommandContext(context.Background(), "bash", "./build_sign_ipxe.sh", config.DownloadPath)
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
	source, err := os.Open(src) //nolint:gosec // Paths are from trusted config
	if err != nil {
		return err
	}
	defer func() {
		if err := source.Close(); err != nil {
			zlog.InfraSec().Error().Err(err).Msg("Failed to close source file")
		}
	}()

	destination, err := os.Create(dst) //nolint:gosec // Paths are from trusted config
	if err != nil {
		return err
	}
	defer func() {
		if err := destination.Close(); err != nil {
			zlog.InfraSec().Error().Err(err).Msg("Failed to close destination file")
		}
	}()

	_, err = io.Copy(destination, source)
	if err != nil {
		return err
	}

	return nil
}
