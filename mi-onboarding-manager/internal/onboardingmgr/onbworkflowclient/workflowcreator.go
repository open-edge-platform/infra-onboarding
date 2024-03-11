/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onbworkflowclient

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
)

var (
	clientName = "WorkflowCreator"
	zlog       = logging.GetLogger(clientName)
)
var rvEnabled = flag.Bool("rvenabled", false, "Set to true if you have enabled rv")

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
		attestationType  string
		mfgIp            string
		onrIp            string
		apiUser          string
		mfgApiPasswd     string
		onrApiPasswd     string
		mfgPort          string
		onrPort          string
		authType         string
		serialNo         string
		statusCode       int
		deviceGuid       []byte
		extendVoucher    []byte
		ownerCertificate []byte
	)

	attestationType = "SECP256R1"
	authType = "digest"
	mfgIp = deviceinfo.FdoMfgDNS
	onrIp = deviceinfo.FdoOwnerDNS
	mfgPort = deviceinfo.FdoMfgPort
	onrPort = deviceinfo.FdoOwnerPort
	serialNo = deviceinfo.HwSerialID

	//default values
	defaultAttestationType := "SECP256R1"
	defaultMfgIp := "mi-fdo-mfg"
	defaultOnrIp := "mi-fdo-owner"
	defaultApiUser := "apiUser"
	defaultMfgApiPasswd := ""
	defaultOnrApiPasswd := ""
	defaultmfgPort := "58039"
	defaultonrPort := "58042"

	if attestationType == "" {
		attestationType = defaultAttestationType
	}
	if mfgIp == "" {
		mfgIp = defaultMfgIp
	}
	if onrIp == "" {
		onrIp = defaultOnrIp
	}
	if apiUser == "" {
		apiUser = defaultApiUser
	}
	if mfgApiPasswd == "" {
		mfgApiPasswd = defaultMfgApiPasswd
	}
	if onrApiPasswd == "" {
		onrApiPasswd = defaultOnrApiPasswd
	}
	if mfgPort == "" {
		mfgPort = defaultmfgPort
	}
	if onrPort == "" {
		onrPort = defaultonrPort
	}
	if authType == "" {
		return "", fmt.Errorf("auth method is mandatory")
	}
	if serialNo == "" {
		return "", fmt.Errorf("serial number of device is mandatory")
	}
	// TODO : remove the use of Goto statement
api:
	//used to GET the certificate
	url := "http://" + onrIp + ":" + onrPort + "/api/v1/certificate?alias=" + attestationType
	resp, err := apiCalls("GET", url, authType, apiUser, onrApiPasswd, []byte{}, deviceinfo.HwMacID)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("")
		return "", fmt.Errorf("Error1 Details:%v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		ownerCertificate, err = io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("error reading the file:%v", err)
		}
		statusCode = 0
	api2:
		//used GET the mfg voucher
		url = "http://" + mfgIp + ":" + mfgPort + "/api/v1/mfg/vouchers/" + serialNo
		resp, err := apiCalls("POST", url, authType, apiUser, mfgApiPasswd, ownerCertificate, deviceinfo.HwMacID)
		if err != nil {
			zlog.MiSec().MiErr(err).Msgf("")
			return "", fmt.Errorf("error Details:%v ", err)
		}
		if resp.StatusCode == http.StatusOK {
			extendVoucher, err = io.ReadAll(resp.Body)
			if err != nil {
				return "", fmt.Errorf("error writing the response to the file:%v", err)
			}
			statusCode = 0
		api3:
			//used GET the owner voucher
			url = "http://" + onrIp + ":" + onrPort + "/api/v1/owner/vouchers/"
			resp, err = apiCalls("POST", url, authType, apiUser, onrApiPasswd, extendVoucher, deviceinfo.HwMacID)
			if err != nil {
				zlog.MiSec().MiErr(err).Msgf("")
				return "", fmt.Errorf("error details :%v", err)
			}
			if resp.StatusCode == http.StatusOK {
				deviceGuid, err = io.ReadAll(resp.Body)
				if err != nil {
					return "", fmt.Errorf("error reading the file:%v", err)
				}
				statusCode = 0
			api4:
				if *rvEnabled {
					//starts TO0
					url := fmt.Sprintf("http://%s:%s/api/v1/to0/%s", onrIp, onrPort, deviceGuid)
					resp, err := apiCalls("GET", url, authType, apiUser, onrApiPasswd, deviceGuid, deviceinfo.HwMacID)
					if err != nil {
						zlog.MiSec().MiErr(err).Msgf("")
						return "", fmt.Errorf("error Details:%v", err)
					}
					if resp.StatusCode == http.StatusOK {
						return string(deviceGuid), nil
					} else {
						statusCode++
						if statusCode < 2 {
							goto api4
						}
						return "", fmt.Errorf("failure in triggering TO0 for %s with GUID %s ", serialNo, deviceGuid)
					}
				} else {
					return string(deviceGuid), nil
				}
			} else {
				statusCode++
				if statusCode < 2 {
					goto api3
				}
				return "", fmt.Errorf("failure in uploading voucher to owner for device with serial number %s with response code: %d", serialNo, resp.StatusCode)
			}
		} else {
			statusCode++
			if statusCode < 2 {
				goto api2
			}
			return "", fmt.Errorf("failure in getting extended voucher for device with serial number %s with response code: %d", serialNo, resp.StatusCode)
		}
	} else {
		statusCode++
		if statusCode < 2 {
			goto api
		}
		return "", fmt.Errorf("failure in getting owner certificate for type %s with response code: %d", attestationType, resp.StatusCode)
	}
}

func apiCalls(httpMethod, url, authType, apiUser, onrApiPasswd string, bodyData []byte, hwMac string) (*http.Response, error) {
	var client *http.Client
	reader := bytes.NewReader(bodyData)
	req, err := http.NewRequest(httpMethod, url, nil)
	if err != nil {
		return nil, err
	}
	if httpMethod == "POST" {
		req.Body = io.NopCloser(reader)
	}
	if strings.ToLower(authType) == "digest" {
		req.SetBasicAuth(apiUser, onrApiPasswd)
		client = &http.Client{}
	} else if strings.ToLower(authType) == "mtls" {
		return nil, fmt.Errorf("MTLS authentication is not supported over HTTP")
	} else {
		return nil, fmt.Errorf("provided Auth type is not valid")
	}
	req.Header.Add("Content-Type", "text/plain")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s request failed with status code:%s", httpMethod, resp.Status)
	}
	return resp, nil
}
