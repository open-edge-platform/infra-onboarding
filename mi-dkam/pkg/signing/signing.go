// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package signing

import (
	"io"
	"net/url"
	"os/exec"
	"strings"

	"os"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/logging"
)

var zlog = logging.GetLogger("MIDKAMAuth")

func SignHookOS(scriptPath string) (bool, error) {

	zlog.MiSec().Info().Msgf("Script dir %s", scriptPath)

	errp := os.Chdir(scriptPath)
	if errp != nil {
		zlog.MiSec().Fatal().Err(errp).Msgf("Error changing working directory: %v\n", errp)
		return false, errp
	}

	modeCmd := exec.Command("chmod", "+x", "secure_hookos.sh")
	result, modeErr := modeCmd.CombinedOutput()
	if modeErr != nil {
		zlog.MiSec().Fatal().Err(modeErr).Msgf("Failed to sign microOS script %v", modeErr)
		return false, modeErr
	}
	zlog.Info().Msgf("Script output: %s", string(result))

	signCmd := exec.Command("sh", "./secure_hookos.sh", scriptPath)
	zlog.Info().Msgf("signCmd: %s", signCmd)
	output, signErr := signCmd.CombinedOutput()
	if signErr != nil {
		zlog.MiSec().Fatal().Err(signErr).Msgf("Failed to sign microOS script %v", signErr)
		return false, signErr
	}
	zlog.Info().Msgf("Script output: %s", string(output))
	return true, nil
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
		zlog.MiSec().Fatal().Err(err).Msgf("Failed to run build microOS script")
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
