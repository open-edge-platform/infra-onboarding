/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"fmt"
	"regexp"
	"strings"
)

func GenerateGatewayFromBaseIP(baseIP string) string {
	// Extract the last part of the base IP and replace it with "1" to get the gateway
	lastPart := strings.Split(baseIP, ".")[3]
	return baseIP[:len(baseIP)-len(lastPart)] + "1"
}

func CalculateRootFS(imageType, diskDev string) string {
	rootFSPartNo := "1"

	if imageType == "bkc" {
		rootFSPartNo = "1"
	}

	// Use regular expression to check if diskDev ends with a numeric digit
	match, err := regexp.MatchString(".*[0-9]$", diskDev)
	if err != nil {
		return rootFSPartNo
	}
	if match {
		return fmt.Sprintf("p%s", rootFSPartNo)
	}

	return rootFSPartNo
}

// ReplaceHostIP finds %host_ip% in the url string and replaces it with ip.
func ReplaceHostIP(url, ip string) string {
	// Define the regular expression pattern to match #host_ip
	re := regexp.MustCompile(`%host_ip%`)
	return re.ReplaceAllString(url, ip)
}

// TODO : Will scale it in future accordingly.
func IsValidOSURLFormat(osURL string) bool {
	expectedSuffix := ".raw.gz" // Checks if the OS URL is in the expected format
	return strings.HasSuffix(osURL, expectedSuffix)
}
