/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/provisioningproto"
	"google.golang.org/protobuf/proto"
)

func DeepCopyOnboardingRequest(req *pb.OnboardingRequest) *pb.OnboardingRequest {
	if req == nil {
		return nil
	}

	// Use proto.Clone to create a deep copy of the request.
	copyOfRequest := proto.Clone(req).(*pb.OnboardingRequest)
	log.Printf("fromcomon.go")

	return copyOfRequest
}

func ChangeWorkingDirectory(targetDir string) error {
	currDir, err := os.Getwd()
	if err != nil {
		return err
	}

	// Calculate the absolute path of the target directory
	absPath := filepath.Join(currDir, targetDir)

	// Change the working directory to the target directory
	if err := os.Chdir(absPath); err != nil {
		return err
	}

	return nil
}

func MakeHTTPGETRequest(hostIP, guidValue, caCertPath, certPath string) ([]byte, error) {
	// Read the CA certificate
	caCert, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("Error reading CA certificate: %v", err)
	}

	// Load client certificate and key
	cert, err := tls.LoadX509KeyPair(certPath, certPath)
	if err != nil {
		return nil, fmt.Errorf("Error loading client certificate: %v", err)
	}

	// Create a custom certificate pool and add the CA certificate
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caCert)

	// Configure the HTTP client to use the custom certificates and skip hostname verification
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:            pool,
				Certificates:       []tls.Certificate{cert},
				InsecureSkipVerify: true, // Skip hostname verification
			},
		},
	}

	// Make an HTTP GET request
	url := fmt.Sprintf("https://%s:8043/api/v1/owner/state/%s", hostIP, guidValue)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating HTTP request: %v", err)
	}

	// Perform the GET request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading response body: %v", err)
	}

	return body, nil
}

func ParseAndUpdateUrl(onboardingRequest *pb.OnboardingRequest) {
	for _, artifactData := range onboardingRequest.ArtifactData {
		//sets and parse the enviroment variable
		// Determine which environment variable to set based on the Category
		print("artifact data------", artifactData)
		var envVarName string
		if artifactData.Category.String() == "OS" || artifactData.Name == "OS" {
			envVarName = "BKC_URL"
		} else if artifactData.Category.String() == "PLATFORM" || artifactData.Name == "PLATFORM" {
			envVarName = "BKC_BASEPKG"
		} else {
			//Todo:Add support for other  category
			fmt.Printf("Unsupported category: %s\n", artifactData.Category.String())
			continue
		}

		// Set the environment variable
		print("envvarname", envVarName, "packageurl", artifactData.PackageUrl)
		err := os.Setenv(envVarName, artifactData.PackageUrl)
		if err != nil {
			fmt.Printf("Error setting environment variable %s: %v\n", envVarName, err)
		}
	}
}

// this function cleanup the file for nect time use after onboarding is done
func ClearFileAndWriteHeader(filePath string) error {
	// Open the file for writing and truncate it to 0 bytes
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the desired string to the file
	header := `################################################################
#
#  This file is a input for the zero touch onboarding automation
#  provide the details as per below order for sut's to provision
#
################################################################

#Example
#SUT_NAME  #MAC_ID           #Load_Balancer_IP   #SUT_IP           #Disk_type   	#Image_Type

#SUT1	  00:49:fa:07:8d:05   10.199.199.100	10.199.199.101   /dev/nvme0n1    prod_bkc/prod_focal/prod_jammy/prod_focal-ms
`

	_, err = file.WriteString(header)
	if err != nil {
		return err
	}

	return nil
}
