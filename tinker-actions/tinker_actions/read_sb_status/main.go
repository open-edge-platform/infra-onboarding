/*
#####################################################################################

# INTEL CONFIDENTIAL                                                                #
# Copyright (C) 2024 Intel Corporation                                              #
# This software and the related documents are Intel copyrighted materials,          #
# and your use of them is governed by the express license under which they          #
# were provided to you ("License"). Unless the License provides otherwise,          #
# you may not use, modify, copy, publish, distribute, disclose or transmit          #
# this software or the related documents without Intel's prior written permission.  #
# This software and the related documents are provided as is, with no express       #
# or implied warranties, other than those that are expressly stated in the License. #
#####################################################################################
*/
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

const (
	address            = "localhost:50054" // Onboarding Manager server address
	SecureBootDisabled = "1"               // Failure case
	SecureBootEnabled  = "0"               // Success case
)

// Function to get guid ID
func GetGUID() (string, error) {
	const boardSerialFilePath = "/sys/class/dmi/id/product_uuid"

	content, err := ioutil.ReadFile(boardSerialFilePath)
	if err != nil {
		log.Fatalf("error in reading guid: %v\n", err)
		return "", err
	}

	guid := strings.TrimSpace(string(content))
	return guid, nil
}

func main() {
	securityFeatureFlagSetBySI := os.Getenv("SECURITY_FEATURE_FLAG")
	// Extract the secure boot status from dmesg command
	cmd := exec.Command("bash", "-c", `sudo dmesg | grep -i "secure boot" | grep -i "enabled"; echo $?`)
	output, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
		return
	}
	ENsecBootstr := strings.TrimSpace(string(output))
	if (ENsecBootstr == SecureBootDisabled && securityFeatureFlagSetBySI == "SECURITY_FEATURE_SECURE_BOOT_AND_FULL_DISK_ENCRYPTION") || (ENsecBootstr == SecureBootEnabled && securityFeatureFlagSetBySI == "SECURITY_FEATURE_UNSPECIFIED") ||
		(ENsecBootstr == SecureBootEnabled && securityFeatureFlagSetBySI == "SECURITY_FEATURE_NONE") {
		fmt.Printf("SecureBoot Status Mismatch\n")
		return
	}
	fmt.Printf("SecureBoot Status Match\n")
}
