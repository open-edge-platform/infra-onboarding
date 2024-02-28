/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"log"
	"os"
	"path/filepath"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/api"
	logging "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
	"google.golang.org/protobuf/proto"
)

var (
	clientName = "Onbcommon"
	zlog       = logging.GetLogger(clientName)
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
		zlog.Debug().Msgf("error while changing working directory to the target directory: %v", err)
		return err
	}

	return nil
}

func ParseAndUpdateURL(onboardingRequest *pb.OnboardingRequest) {
	for _, artifactData := range onboardingRequest.ArtifactData {
		// sets and parse the environment variable
		// Determine which environment variable to set based on the Category
		print("artifact data------", artifactData)
		var envVarName string
		category := artifactData.Category.String()
		switch {
		case category == "OS" || artifactData.Name == "OS":
			envVarName = "BKC_URL"
		case category == "PLATFORM" || artifactData.Name == "PLATFORM":
			envVarName = "BKC_BASEPKG"
		default:
			// Todo:Add support for other  category
			zlog.Debug().Msgf("Unsupported category: %s\n", artifactData.Category.String())
			continue
		}

		// Set the environment variable
		print("envvarname", envVarName, "packageurl", artifactData.PackageUrl)
		err := os.Setenv(envVarName, artifactData.PackageUrl)
		if err != nil {
			zlog.Debug().Msgf("Error setting environment variable %s: %v\n", envVarName, err)
		}
	}
}

// this function cleanup the file for nect time use after onboarding is done.
func ClearFileAndWriteHeader(filePath string) error {
	const filePermission = 0o644
	// Open the file for writing and truncate it to 0 bytes
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, filePermission)
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
