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


func main() {
	securityFeatureFlagSetBySI := os.Getenv("SECURITY_FEATURE_FLAG")
	// Extract the secure boot status from dmesg command
	cmd := exec.Command("/bin/sh", "-c", `cat /host/sblog.txt | grep -i "secure boot enabled" > /dev/null ;  echo $?`)
	output, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
		return
	}

	outputString := string(output)
	// Split the output into lines
	lines := strings.Split(outputString, "\n")
	// Get the last line (containing the exit status)
	exitStatus := lines[len(lines)-2]
	ENsecBootstr := exitStatus

	if (ENsecBootstr == SecureBootDisabled && securityFeatureFlagSetBySI == "SECURITY_FEATURE_SECURE_BOOT_AND_FULL_DISK_ENCRYPTION") ||
		(ENsecBootstr == SecureBootDisabled && securityFeatureFlagSetBySI == "SECURITY_FEATURE_UNSPECIFIED") ||
		(ENsecBootstr == SecureBootEnabled && securityFeatureFlagSetBySI == "SECURITY_FEATURE_NONE") {
		/* Note : Do not change the case in 'Mismatch', as this message is grep'ed in run_sb.sh */
		fmt.Printf("Verifying Secure Boot Settings Mismatch\n")
		return
	}
	fmt.Printf("Verifying Secure Boot Settings Match\n")
}
