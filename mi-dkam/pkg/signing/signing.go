// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package signing

import (
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"

	"os"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/config"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/logging"
)

var zlog = logging.GetLogger("MIDKAMAuth")

func DownloadMicroOS(scriptPath string) (bool, error) {
	zlog.Info().Msgf("Inside Download and sign artifact... %s", scriptPath)
	url := "http://rs-proxy-files.rs-proxy.svc.cluster.local:8081/publish/fm_en_artifacts/hook-os/0.8.1-dev-e87f024/alpine_image/hook_x86_64.tar.gz"
	client := &http.Client{
		Transport: &http.Transport{
			ForceAttemptHTTP2: false,
			MaxIdleConns:      10,
			IdleConnTimeout:   30,
		},
	}
	// Create an HTTP GET request with the specified URL
	req, httperr := http.NewRequest("GET", url, nil)
	if httperr != nil {
		zlog.MiSec().Fatal().Err(httperr).Msgf("Error creating request: %v\n", httperr)
		return false, httperr

	}

	// Set the HTTP version to 1.1
	req.Proto = "HTTP/1.1"

	// Perform the HTTP GET request
	resp, clienterr := client.Do(req)
	if clienterr != nil {
		zlog.MiSec().Fatal().Err(clienterr).Msgf("Error performing request: %v\n", clienterr)
		return false, clienterr

	}
	defer resp.Body.Close()

	filePath := config.PVC + "hook_x86_64.tar.gz"
	//Read the response body
	//Create or open the local file for writing
	file, fileerr := os.Create(filePath)
	if fileerr != nil {
		zlog.MiSec().Fatal().Err(fileerr).Msgf("Error while creating release manifest file.")
		return false, fileerr
	}
	defer file.Close()

	// Copy the response body to the local file
	_, copyErr := io.Copy(file, resp.Body)
	if copyErr != nil {
		zlog.MiSec().Fatal().Err(copyErr).Msgf("Error while coping content ")
	}
	zlog.MiSec().Info().Msg("File downloaded")
	return true, nil

	// cmd := "curl -o /data/hook_x86_64.tar.gz --request GET http://rs-proxy-files.rs-proxy.svc.cluster.local:8081/publish/fm_en_artifacts/hook-os/0.7.0-dev-2636510/alpine_image/hook_x86_64.tar.gz --http1.1"
	// osCmd := exec.Command("bash", "-c", cmd)
	// result, err := osCmd.CombinedOutput()
	// if err != nil {
	// 	// Handle error
	// 	zlog.MiSec().Fatal().Err(err).Msgf("Failed to download %v", err)
	// 	return false, err
	// }

	// zlog.Info().Msgf("Script output: %s", string(result))

	// return true, nil
}

func SignHookOS(scriptPath string) (bool, error) {

	// Specify the sb_keys directory to store sign keys
	// sbKeysDir := "sb_keys"

	zlog.MiSec().Info().Msgf("Script dir %s", scriptPath)

	// Specify the full path of sb_keys directory.
	// sbKeysDirPath := filepath.Join(scriptPath, sbKeysDir)
	// zlog.MiSec().Info().Msgf("SB Keys dir %s", sbKeysDirPath)

	// if _, err := os.Stat(sbKeysDirPath); os.IsNotExist(err) {

	// 	mkErr := os.MkdirAll(sbKeysDirPath, 0755) // 0755 sets read, write, and execute permissions for owner, and read and execute permissions for others
	// 	if mkErr != nil {
	// 		zlog.MiSec().Fatal().Err(mkErr).Msgf("Error creating directory: %v", mkErr)
	// 		return false, mkErr
	// 	}
	// 	zlog.MiSec().Info().Msg("sign keys folder created successfully")
	// // Change into the newly created directory
	// chErr := os.Chdir(sbKeysDirPath)
	// if chErr != nil {
	// 	zlog.MiSec().Fatal().Err(chErr).Msgf("Error changing into directory: %v", chErr)
	// 	return false, chErr
	// }

	// // Run a shell command to generate files (for demonstration, touch command is used)
	// cmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:4096", "-keyout", "db.key", "-out", "db.crt", "-days", "1000", "-nodes", "-subj", "/CN=4c4c4544-0035-3010-8030-c2c04f4a4633", "-addext", "subjectAltName=DNS:4c4c4544-0035-3010-8030-c2c04f4a4633")
	// exeErr := cmd.Run()
	// if exeErr != nil {
	// 	zlog.MiSec().Fatal().Err(exeErr).Msgf("Error running command: %v", exeErr)
	// 	return false, exeErr
	// }
	// } else {
	// 	zlog.MiSec().Info().Msg("sign keys folder already exists.")
	// }

	// chdirErr := os.Chdir("..")
	// if chdirErr != nil {
	// 	zlog.MiSec().Fatal().Err(chdirErr).Msgf("Error changing back to the parent directory: %v", chdirErr)
	// 	return false, chdirErr
	// }

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
