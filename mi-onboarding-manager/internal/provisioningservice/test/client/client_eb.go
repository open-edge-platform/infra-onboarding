package main

import (
	"context"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	pbi "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/onboardingmgr"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/provisioningproto"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"
)

// Struct to hold input configuration from YAML
type InputConfig struct {
	ArtifactData []struct {
		Name        string `yaml:"name"`
		Version     string `yaml:"version"`
		Platform    string `yaml:"platform"`
		Category    int    `yaml:"category"`
		Description string `yaml:"description"`
		Details     struct {
			Name    string `yaml:"name"`
			Url     string `yaml:"url"`
			Contact string `yaml:"contact"`
		} `yaml:"details"`
		PackageUrl   string `yaml:"packageUrl"`
		Author       string `yaml:"author"`
		License      string `yaml:"license"`
		Vendor       string `yaml:"vendor"`
		Manufacturer string `yaml:"manufacturer"`
		ReleaseData  string `yaml:"releaseData"`
		ArtifactId   string `yaml:"artifactId"`
		Result       int    `yaml:"result"`
	} `yaml:"artifactData"`

	HwData []struct {
		HwId      string `yaml:"hwid"`
		MacId     string `yaml:"macid"`
		SutIp     string `yaml:"sutip"`
		CusParams struct {
			DpsScopeId          string `yaml:"dpsscopeid"`
			DpsRegistrationId   string `yaml:"dpsregistrationid"`
			DpsEnrollmentSymKey string `yaml:"dpsenrollmentsymkey"`
		} `yaml:"cusparams"`
		DiskPartition string `yaml:"disk"`
		PlatformType  string `yaml:"platformtype"`
	} `yaml:"hwdata"`

	OnbParams struct {
		PdIp           string `yaml:"pdip"`
		PdMac          string `yaml:"pdmac"`
		LoadBalancerIp string `yaml:"loadbalancerip"`
		DiskPartition  string `yaml:"diskpartition"`
		Env            string `yaml:"env"`
	} `yaml:"onboarding"`
}

func generateDevSerial(macID string) (string, error) {
	// Remove colons from the MAC address
	uniqueID := strings.ReplaceAll(macID, ":", "")

	// Generate a random alphanumeric string of length 5
	rand.Seed(time.Now().UnixNano())
	randID := generateRandomString(5)

	// Truncate the uniqueID to remove the first 6 characters
	truncatedID := uniqueID[6:]

	// Concatenate truncatedID and randID to create devSerial
	devSerial := truncatedID + randID

	return devSerial, nil
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func OnboardingTest(client pb.OnBoardingEBClient) (*pb.OnboardingResponse, error) {
	var obm pb.OnboardingRequest
	log.Printf("start onboarding")
	dirPath, _ := os.Getwd()
	// Read YAML file
	yamlData, err := ioutil.ReadFile(dirPath + "/profile_sample.yaml")
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
		obm.ArtifactData = append(obm.ArtifactData, &pbi.ArtifactData{
			Name:        artifactData.Name,
			Version:     artifactData.Version,
			Platform:    artifactData.Platform,
			Category:    pbi.ArtifactData_ArtifactCategory(artifactData.Category),
			Description: artifactData.Description,
			Details: &pbi.Supplier{
				Name:    artifactData.Details.Name,
				Url:     artifactData.Details.Url,
				Contact: artifactData.Details.Contact,
			},
			PackageUrl:   artifactData.PackageUrl,
			Author:       artifactData.Author,
			License:      artifactData.License,
			Vendor:       artifactData.Vendor,
			Manufacturer: artifactData.Manufacturer,
			ReleaseData:  artifactData.ReleaseData,
			ArtifactId:   artifactData.ArtifactId,
			Result:       pbi.ArtifactData_Response(artifactData.Result),
		})
	}

	// Iterate through hardware data and populate obm
	for _, hwData := range inputConfig.HwData {
		obm.Hwdata = append(obm.Hwdata, &pbi.HwData{
			HwId:  hwData.HwId,
			MacId: hwData.MacId,
			SutIp: hwData.SutIp,
			CusParams: &pbi.CustomerParams{
				DpsScopeId:          hwData.CusParams.DpsScopeId,
				DpsRegistrationId:   hwData.CusParams.DpsRegistrationId,
				DpsEnrollmentSymKey: hwData.CusParams.DpsEnrollmentSymKey,
			},
			DiskPartition: hwData.DiskPartition,
			PlatformType:  hwData.PlatformType,
		})
	}

	// Populate onboarding parameters
	obm.OnbParams = &pbi.OnboardingParams{
		PdIp:           inputConfig.OnbParams.PdIp,
		PdMac:          inputConfig.OnbParams.PdMac,
		LoadBalancerIp: inputConfig.OnbParams.LoadBalancerIp,
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
	onb_addr := os.Getenv("MGR_HOST")
	onb_port := os.Getenv("ONBMGR_PORT")
	address := onb_addr + ":" + onb_port

	if onb_addr == "" || onb_port == "" {
		log.Printf("Invalid environment variables MGR_HOST and ONBMGR_PORT please export")
		os.Exit(1)
	}

	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewOnBoardingEBClient(conn)
	res, err := OnboardingTest(client)
	if err != nil {
		log.Fatalf("Onboarding failed: %v", err)
	}

	log.Printf("Onboarding state: %s", res.Status)
}
