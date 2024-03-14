/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onbworkflowclient

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
)

var (
	clientName = "WorkflowCreator"
	zlog       = logging.GetLogger(clientName)
)
var rvEnabled = flag.Bool("rvenabled", false, "Set to true if you have enabled rv")

const (
	hwPrefixName       = "machine-"
	workFlowPrefixName = "workflow-"
	retryAttempts      = 2
)

func readUIDFromFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func unsetEnvironmentVariables() {
	// List of environment variables to unset
	variablesToUnset := []string{"http_proxy", "https_proxy"}

	for _, variable := range variablesToUnset {
		if err := os.Unsetenv(variable); err != nil {
			fmt.Printf("Failed to unset %s: %v\n", variable, err)
		} else {
			fmt.Printf("Unset %s\n", variable)
		}
	}
}

func VoucherExtension(hostIP, deviceSerial string) (string, error) {
	// Construct the path to the script directory
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	scriptDir := usr.HomeDir + "/pri-fidoiot/component-samples/demo/scripts"

	// Change the current working directory to the script directory
	oldWorkingDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if chdirErr := os.Chdir(scriptDir); chdirErr != nil {
		return "", chdirErr
	}

	log.Printf("Job %s has completed", scriptDir)
	unsetEnvironmentVariables()
	variableName := "https_proxy"

	// Use os.LookupEnv to check if the environment variable is present
	val, present := os.LookupEnv(variableName)

	if present {
		fmt.Printf("%s env variable present with value: %s\n", variableName, val)
	} else {
		fmt.Printf("%s env variable not present\n", variableName)
	}

	// Make the script executable
	cmdChmod := exec.Command("chmod", "+x", "extend_upload.sh")
	if runErr := cmdChmod.Run(); runErr != nil {
		return "", runErr
	}
	fmt.Printf("host ip: %s\n", hostIP)

	// Run the shell script with arguments
	cmdExtendUpload := exec.Command("./extend_upload.sh", "-m", "sh", "-c",
		"./secrets/", "-e", "mtls", "-m", hostIP, "-o", hostIP, "-s", deviceSerial)

	output, err := cmdExtendUpload.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error executing script: %w, Output: %s", err, output)
	}

	fmt.Printf("Script Output: voucher done\n%s\n", output)
	// Create the GUID file path
	guidFilePath := deviceSerial + "_guid.txt"

	// Read the GUID from the file
	uid, err := readUIDFromFile(guidFilePath)
	if err != nil {
		log.Printf("Error reading UID from file: %v", err)
		// You can handle this error as needed, e.g., return an error or retry.
		return "", err
	}

	if err := os.Chdir(oldWorkingDir); err != nil {
		return "", err
	}

	return uid, nil
}

func VoucherScript(deviceinfo utils.DeviceInfo) (string, error) {
	var (
		attestationType string
		mfgIP           string
		onrIP           string
		apiUser         string
		mfgAPIPasswd    string
		onrAPIPasswd    string
		mfgPort         string
		onrPort         string
		serialNo        string
	)
	const httpPrefix = "http://"
	attestationType = "SECP256R1"
	mfgIP = deviceinfo.FdoMfgDNS
	onrIP = deviceinfo.FdoOwnerDNS
	mfgPort = deviceinfo.FdoMfgPort
	onrPort = deviceinfo.FdoOwnerPort
	serialNo = deviceinfo.HwSerialID

	// default values
	defaultMfgIP := "mi-fdo-mfg"
	defaultOnrIP := "mi-fdo-owner"
	defaultAPIUser := "apiUser"
	defaultMfgAPIPasswd := ""
	defaultOnrAPIPasswd := ""
	defaultmfgPort := "58039"
	defaultonrPort := "58042"

	if mfgIP == "" {
		mfgIP = defaultMfgIP
	}
	if onrIP == "" {
		onrIP = defaultOnrIP
	}
	if apiUser == "" {
		apiUser = defaultAPIUser
	}
	if mfgAPIPasswd == "" {
		mfgAPIPasswd = defaultMfgAPIPasswd
	}
	if onrAPIPasswd == "" {
		onrAPIPasswd = defaultOnrAPIPasswd
	}
	if mfgPort == "" {
		mfgPort = defaultmfgPort
	}
	if onrPort == "" {
		onrPort = defaultonrPort
	}
	if serialNo == "" {
		return "", fmt.Errorf("serial number of device is mandatory")
	}
	// TODO : remove the use of Goto statement
	guid, err := triggerTOApiCalls(httpPrefix, onrIP, onrPort, attestationType, apiUser,
		onrAPIPasswd, mfgIP, mfgPort, serialNo, mfgAPIPasswd)
	if err != nil {
		return "", err
	}
	return guid, nil
}

//nolint:cyclop // May effect the functionality, need to simplify this in future
func triggerTOApiCalls(httpPrefix, onrIP, onrPort, attestationType, apiUser, onrAPIPasswd,
	mfgIP, mfgPort, serialNo, mfgAPIPasswd string,
) (string, error) {
	var (
		statusCode       int
		deviceGUID       []byte
		extendVoucher    []byte
		ownerCertificate []byte
	)
	// TODO : remove the use of Goto statement
api:
	// used to GET the certificate
	url := httpPrefix + onrIP + ":" + onrPort + "/api/v1/certificate?alias=" + attestationType
	resp1, err := apiCalls("GET", url, apiUser, onrAPIPasswd, []byte{})
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("")
		return "", fmt.Errorf("Error1 Details:%w", err)
	}
	defer resp1.Body.Close()
	if resp1.StatusCode == http.StatusOK {
		ownerCertificate, err = io.ReadAll(resp1.Body)
		if err != nil {
			return "", fmt.Errorf("error reading the file:%w", err)
		}
		statusCode = 0
	api2:
		// used GET the mfg voucher
		url = httpPrefix + mfgIP + ":" + mfgPort + "/api/v1/mfg/vouchers/" + serialNo
		resp2, err := apiCalls("POST", url, apiUser, mfgAPIPasswd, ownerCertificate)
		if err != nil {
			zlog.MiSec().MiErr(err).Msgf("")
			return "", fmt.Errorf("error Details:%w ", err)
		}
		defer resp2.Body.Close()
		if resp2.StatusCode == http.StatusOK {
			extendVoucher, err = io.ReadAll(resp2.Body)
			if err != nil {
				return "", fmt.Errorf("error writing the response to the file:%w", err)
			}
			statusCode = 0
		api3:
			// used GET the owner voucher
			url = httpPrefix + onrIP + ":" + onrPort + "/api/v1/owner/vouchers/"
			resp3, err := apiCalls("POST", url, apiUser, onrAPIPasswd, extendVoucher)
			if err != nil {
				zlog.MiSec().MiErr(err).Msgf("")
				return "", fmt.Errorf("error details :%w", err)
			}
			defer resp3.Body.Close()
			if resp3.StatusCode == http.StatusOK {
				deviceGUID, err = io.ReadAll(resp3.Body)
				if err != nil {
					return "", fmt.Errorf("error reading the file:%w", err)
				}
				statusCode = 0
			api4:
				if *rvEnabled {
					// starts TO0
					url := fmt.Sprintf("http://%s:%s/api/v1/to0/%s", onrIP, onrPort, deviceGUID)
					resp4, err := apiCalls("GET", url, apiUser, onrAPIPasswd, deviceGUID)
					if err != nil {
						zlog.MiSec().MiErr(err).Msgf("")
						return "", fmt.Errorf("error Details:%w", err)
					}
					defer resp4.Body.Close()
					if resp4.StatusCode == http.StatusOK {
						return string(deviceGUID), nil
					}
					statusCode++
					if statusCode < retryAttempts {
						goto api4
					}
					return "", fmt.Errorf("failure in triggering TO0 for %s with GUID %s ", serialNo, deviceGUID)
				}
				return string(deviceGUID), nil
			}
			statusCode++
			if statusCode < retryAttempts {
				goto api3
			}
			return "",
				fmt.Errorf("failure in uploading voucher to owner for device with serial number %s with response code: %d",
					serialNo, resp3.StatusCode)
		}
		statusCode++
		if statusCode < retryAttempts {
			goto api2
		}
		return "", fmt.Errorf("failure in getting extended voucher for device with serial number %s with response code: %d",
			serialNo, resp2.StatusCode)
	}
	statusCode++
	if statusCode < retryAttempts {
		goto api
	}
	return "", fmt.Errorf("failure in getting owner certificate for type %s with response code: %d",
		attestationType, resp1.StatusCode)
}

func apiCalls(httpMethod, url, apiUser, onrAPIPasswd string, bodyData []byte) (*http.Response, error) {
	var httpClient *http.Client
	authType := "digest"
	reader := bytes.NewReader(bodyData)
	req, err := http.NewRequestWithContext(context.Background(), httpMethod, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	if httpMethod == "POST" {
		req.Body = io.NopCloser(reader)
	}
	switch strings.ToLower(authType) {
	case "digest":
		req.SetBasicAuth(apiUser, onrAPIPasswd)
		httpClient = &http.Client{}
	case "mtls":
		return nil, fmt.Errorf("MTLS authentication is not supported over HTTP")
	default:
		return nil, fmt.Errorf("provided Auth type is not valid")
	}
	req.Header.Add("Content-Type", "text/plain")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s request failed with status code:%s", httpMethod, resp.Status)
	}
	return resp, nil
}
