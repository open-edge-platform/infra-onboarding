/*
   Copyright (C) 2023 Intel Corporation
   SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"log"
	"os"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v2"
)

// Struct to hold input configuration from YAML.
type InputConfig struct {
	ArtifactData []struct {
		Name        string `yaml:"name"`
		Version     string `yaml:"version"`
		Platform    string `yaml:"platform"`
		Category    int    `yaml:"category"`
		Description string `yaml:"description"`
		Details     struct {
			Name    string `yaml:"name"`
			URL     string `yaml:"url"`
			Contact string `yaml:"contact"`
		} `yaml:"details"`
		PackageURL   string `yaml:"packageUrl"`
		Author       string `yaml:"author"`
		License      string `yaml:"license"`
		Vendor       string `yaml:"vendor"`
		Manufacturer string `yaml:"manufacturer"`
		ReleaseData  string `yaml:"releaseData"`
		ArtifactID   string `yaml:"artifactId"`
		Result       int    `yaml:"result"`
	} `yaml:"artifactData"`

	HwData []struct {
		HwID      string `yaml:"hwid"`
		MacID     string `yaml:"macid"`
		SutIP     string `yaml:"sutip"`
		CusParams struct {
			DpsScopeID          string `yaml:"dpsscopeid"`
			DpsRegistrationID   string `yaml:"dpsregistrationid"`
			DpsEnrollmentSymKey string `yaml:"dpsenrollmentsymkey"`
		} `yaml:"cusparams"`
		DiskPartition string `yaml:"disk"`
		PlatformType  string `yaml:"platformtype"`
	} `yaml:"hwdata"`

	OnbParams struct {
		PdIP           string `yaml:"pdip"`
		PdMac          string `yaml:"pdmac"`
		LoadBalancerIP string `yaml:"loadbalancerip"`
		DiskPartition  string `yaml:"diskpartition"`
		Env            string `yaml:"env"`
	} `yaml:"onboarding"`
}

func OnboardingTest(client pb.OnBoardingEBClient) (*pb.OnboardingResponse, error) {
	var obm pb.OnboardingRequest
	log.Printf("start onboarding")
	dirPath, _ := os.Getwd()
	// Read YAML file
	yamlData, err := os.ReadFile(dirPath + "/profile_sample.yaml")
	if err != nil {
		log.Fatalf("Error reading YAML file: %v", err)
		return nil, err
	}

	// Unmarshal YAML data into struct
	var inputConfig InputConfig
	err = yaml.Unmarshal(yamlData, &inputConfig)
	if err != nil {
		log.Fatalf("Error unmarshalling YAML: %v", err)
		return nil, err
	}

	// Iterate through artifact data and populate obm
	for _, artifactData := range inputConfig.ArtifactData {
		obm.ArtifactData = append(obm.ArtifactData, &pb.ArtifactData{
			Name:        artifactData.Name,
			Version:     artifactData.Version,
			Platform:    artifactData.Platform,
			Category:    pb.ArtifactData_ArtifactCategory(artifactData.Category),
			Description: artifactData.Description,
			Details: &pb.Supplier{
				Name:    artifactData.Details.Name,
				Url:     artifactData.Details.URL,
				Contact: artifactData.Details.Contact,
			},
			PackageUrl:   artifactData.PackageURL,
			Author:       artifactData.Author,
			License:      artifactData.License,
			Vendor:       artifactData.Vendor,
			Manufacturer: artifactData.Manufacturer,
			ReleaseData:  artifactData.ReleaseData,
			ArtifactId:   artifactData.ArtifactID,
			Result:       pb.ArtifactData_Response(artifactData.Result),
		})
	}

	// Iterate through hardware data and populate obm
	for _, hwData := range inputConfig.HwData {
		obm.Hwdata = append(obm.Hwdata, &pb.HwData{
			HwId:  hwData.HwID,
			MacId: hwData.MacID,
			SutIp: hwData.SutIP,
			CusParams: &pb.CustomerParams{
				DpsScopeId:          hwData.CusParams.DpsScopeID,
				DpsRegistrationId:   hwData.CusParams.DpsRegistrationID,
				DpsEnrollmentSymKey: hwData.CusParams.DpsEnrollmentSymKey,
			},
			DiskPartition: hwData.DiskPartition,
			PlatformType:  hwData.PlatformType,
		})
	}

	// Populate onboarding parameters
	obm.OnbParams = &pb.OnboardingParams{
		PdIp:           inputConfig.OnbParams.PdIP,
		PdMac:          inputConfig.OnbParams.PdMac,
		LoadBalancerIp: inputConfig.OnbParams.LoadBalancerIP,
		DiskPartition:  inputConfig.OnbParams.DiskPartition,
		Env:            inputConfig.OnbParams.Env,
	}
	// Add other variables in onboarding request
	res, err := client.StartOnboarding(context.Background(), &obm)
	if err != nil {
		log.Fatalf("Failed to get data: %v", err)
		return nil, err
	}
	return res, nil
}

func main() {
	onbAddr := os.Getenv("MGR_HOST")
	onbPort := os.Getenv("ONBMGR_PORT")
	address := onbAddr + ":" + onbPort

	if onbAddr == "" || onbPort == "" {
		log.Printf("Invalid environment variables MGR_HOST and ONBMGR_PORT please export")
		os.Exit(1)
	}

	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewOnBoardingEBClient(conn)
	res, err := OnboardingTest(client)
	if err != nil {
		log.Printf("Onboarding failed: %v", err)
		return
	}

	log.Printf("Onboarding state: %s", res.Status)
}
