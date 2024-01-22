/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/onboardingmgr"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/utils"
	"github.com/stretchr/testify/assert"
)

var baseIP string = "192.168.1.100"
var serialNumber string = "dummySerialNumber"
var caCertPath string = "ca.crt"
var certPath string = "client.crt"
var pdip string = "example.com"
var guid string = "dummyGUID"

var deviceInfo = utils.DeviceInfo{
	//add dummy data here
	HwMacID:           "01:23:45:67:89:AB",
	HwSerialID:        "serial123",
	ProvisionerIp:     "dummyProvisionerIP",
	Guid:              "dummyGUID",
	ImType:            "prod_focal",
	ClientImgName:     "dummyClientImgName.raw.gz",
	HwIP:              "dummyHwIP",
	DiskType:          "dummyDiskType",
	LoadBalancerIP:    "dummyLoadBalancerIP",
	Gateway:           "dummyGateway",
	DpsScopeId:        "DpsScopeId",
	DpsRegistrationId: "DpsRegistrationId",
	DpsSymmKey:        "DpsSymmKey",
}
var artifactInfo = utils.ArtifactData{
	BkcUrl:        "dummy",
	BkcBasePkgUrl: "dummy",
}

var onboardingRequest = &pb.OnboardingRequest{
	OnbParams: &pb.OnboardingParams{
		PdIp:           "192.168.1.100",
		LoadBalancerIp: "10.0.0.1",
		Env:            "NZT",
	},
	ArtifactData: []*pb.ArtifactData{
		{
			Name:       "Valid OS Artifact",
			Category:   pb.ArtifactData_OS,
			PackageUrl: "https://os.package.url",
		},
		{
			Name:       "Valid PLATFORM Artifact",
			Category:   pb.ArtifactData_PLATFORM,
			PackageUrl: "https://platform.package.url",
		},
	},
	Hwdata: []*pb.HwData{
		{
			HwId:          "serial123",
			MacId:         "01:23:45:67:89:AB",
			SutIp:         "192.168.1.2",
			DiskPartition: "ssd",
			PlatformType:  "prod_focal-ms",
			CusParams: &pb.CustomerParams{
				DpsScopeId:          "scope123",
				DpsRegistrationId:   "reg123",
				DpsEnrollmentSymKey: "key123",
			},
		},
	},
}

func MockMakeHTTPGETRequest(hostIP, guid, caCertPath, certPath string) ([]byte, error) {
	return []byte(`{"to2CompletedOn": "2023-12-05T12:34:56", "to0Expiry": "2023-12-06T00:00:00"}`), nil
}

func MockImageDownload(artifactInfo utils.ArtifactData, deviceInfo utils.DeviceInfo, kubeconfigPath string) error {

	return nil
}
func MockDiWorkflowCreation(deviceInfo utils.DeviceInfo, kubeconfigPath string) (string, error) {
	guid = "dummyGUID"
	return guid, nil
}
func MockProdWorkflowCreation(deviceInfo utils.DeviceInfo, kubeconfigPath string, imType string) error {

	return nil
}

func MockMakeGETRequestWithRetry(serialNumber, pdip string, caCertPath, certPath string, guid string) error {
	timeout := 5 * time.Minute
	startTime := time.Now()
	for {
		if time.Since(startTime) >= timeout {
			return errors.New("time out for T02 Process")
		}

		response, err := MockMakeHTTPGETRequest(pdip, guid, caCertPath, certPath)
		if err != nil {
			log.Fatalf("Error making HTTP GET request: %v", err)
		}

		if len(response) == 0 {
			log.Println("Empty response received. Retrying in 5 seconds...")
			time.Sleep(5 * time.Second)
			continue
		}

		responseData := ResponseData{}
		if err := json.Unmarshal(response, &responseData); err != nil {
			log.Fatalf("Error unmarshaling JSON: %v", err)
		}

		if responseData.To2CompletedOn != "" {

			fmt.Println("to2CompletedOn:", responseData.To2CompletedOn)
			fmt.Println("to0Expiry:", responseData.To0Expiry)

			break
		}

		time.Sleep(5 * time.Second)
	}

	return nil
}

func MockDeviceOnboardingManagerNzt(deviceInfo utils.DeviceInfo, artifactinfo utils.ArtifactData, kubeconfigPath string) error {

	ImageDownloadErr := MockImageDownload(artifactinfo, deviceInfo, kubeconfigPath)
	if ImageDownloadErr != nil {
		log.Println("Error while ImageDownloading: ", ImageDownloadErr)
		return ImageDownloadErr
	}

	log.Printf("Device initialization started for Device: %s", deviceInfo.HwMacID)

	guid, dierror := MockDiWorkflowCreation(deviceInfo, kubeconfigPath)
	if dierror != nil {
		fmt.Printf("Error in DiWorkflowCreation: %v\n", dierror)
	} else {
		fmt.Printf("GUID: %s\n", guid)
		deviceInfo.Guid = guid
	}
	log.Printf("Device initialization completed for device: %s", deviceInfo.HwMacID)

	caCertPath := "/home/" + os.Getenv("USER") + "/.fdo-secrets/scripts/secrets/ca-cert.pem"
	certPath := "/home/" + os.Getenv("USER") + "/.fdo-secrets/scripts/secrets/api-user.pem"

	errto2 := MockMakeGETRequestWithRetry(deviceInfo.HwSerialID, deviceInfo.ProvisionerIp, caCertPath, certPath, deviceInfo.Guid)
	if errto2 != nil {
		log.Println("Error for ", deviceInfo.HwMacID, errto2)
		return errto2
	}

	imgurl := artifactinfo.BkcUrl
	filenameBz2 := filepath.Base(imgurl)
	filenameWithoutExt := strings.TrimSuffix(filenameBz2, ".bz2")
	deviceInfo.ClientImgName = filenameWithoutExt + ".raw.gz"

	proderror := MockProdWorkflowCreation(deviceInfo, kubeconfigPath, deviceInfo.ImType)
	if proderror != nil {
		return proderror
	}

	return nil

}
func MockDeviceOnboardingManager(deviceInfoList []utils.DeviceInfo, artifactinfo utils.ArtifactData, kubeconfigPath string, onbtype string) error {

	oldWorkingDir, errolddir := os.Getwd()
	if errolddir != nil {
		return errolddir
	}
	log.Printf("oldWorkingDir-: %s", oldWorkingDir)

	log.Printf("Onbtype: %s", onbtype)
	if onbtype == "ZT" {
		targetDir := "../../scripts/edge-iaas-platform/platform-director/onboarding"
		if err := utils.ChangeWorkingDirectory(targetDir); err != nil {
			log.Fatalf("Failed to change working directory: %v", err)
		}

		errcleanup := utils.ClearFileAndWriteHeader("sut_onboarding_list.txt")
		if errcleanup != nil {

			return fmt.Errorf("cleanup error: %v", errcleanup)
		}

		currDir, errDIR := os.Getwd()
		if errDIR != nil {
			return errDIR
		}
		log.Printf("Onboarding ZT: %s", currDir)
	} else if onbtype == "NZT" {
		targetDir := "../../internal/onboardingmgr/onboarding/"
		if err := utils.ChangeWorkingDirectory(targetDir); err != nil {
			log.Fatalf("Failed to change working directory: %v", err)
		}

		currDir, errDIR := os.Getwd()
		if errDIR != nil {
			return errDIR
		}
		log.Printf("Onboarding NZT: %s", currDir)
	} else {
		log.Printf("onbtype not correct %s", onbtype)
	}

	for i, deviceInfo := range deviceInfoList {
		if onbtype == "NZT" {
			err := MockDeviceOnboardingManagerNzt(deviceInfo, artifactinfo, kubeconfigPath)
			if err != nil {
				return err
			}

		} else if onbtype == "ZT" {
			sutLabel := fmt.Sprintf("SUT%d", i+1)
			err := DeviceOnboardingManagerZt(deviceInfo, kubeconfigPath, sutLabel)

			if err != nil {
				return err
			}
		}
	}

	if onbtype == "ZT" {
		cmdChmod := exec.Command("chmod", "+x", "zero_touch_onboarding_installation.sh")
		if err := cmdChmod.Run(); err != nil {
			return err
		}

		cmdExtendUpload := exec.Command("./zero_touch_onboarding_installation.sh")
		output, err := cmdExtendUpload.CombinedOutput()

		if errold := os.Chdir(oldWorkingDir); errold != nil {
			return errold
		}
		log.Printf("Onboarding Completed")
		if err != nil {
			return fmt.Errorf("Error executing script: %v, Output: %s", err, output)
		}
	} else {
		log.Printf("Onboarding is NZT:")
	}
	return nil
}
func MockCreateAzureEnvFile(deviceInfo utils.DeviceInfo) error {
	var content []byte
	fileName := "azure-credentials.env_" + deviceInfo.HwMacID
	dirPath, _ := os.Getwd()
	dirPath, _ = strings.CutSuffix(dirPath, "internal/onboardingmgr/onboarding")
	content = fmt.Append(content, fmt.Sprintf("export ID_SCOPE=\"%s\"\n", deviceInfo.DpsScopeId))
	content = fmt.Append(content, fmt.Sprintf("export REGISTRATION_ID=\"%s\"\n", deviceInfo.DpsRegistrationId))
	content = fmt.Append(content, fmt.Sprintf("export SYMMETRIC_KEY=\"%s\"\n", deviceInfo.DpsSymmKey))
	err := os.WriteFile(dirPath+"/internal/onboardingmgr/azure_env/"+fileName, []byte(content), 0644)
	if err != nil {
		return err
	}
	os.Remove(dirPath + "/internal/onboardingmgr/azure_env/" + fileName)
	return nil

}
func MockCreateDeviceInfoListNAzureEnv(copyOfRequest *pb.OnboardingRequest) ([]utils.DeviceInfo, error) {
	var deviceInfoList []utils.DeviceInfo

	gateway := generateGatewayFromBaseIP(baseIP)
	log.Println(gateway)

	deviceInfoList = append(deviceInfoList, deviceInfo)

	if deviceInfo.ImType == "dummyImType" {
		err := MockCreateAzureEnvFile(deviceInfo)
		if err != nil {
			log.Fatalf("error while createing azure-credentials.env_%s is %v", deviceInfo.HwMacID, err)
			return nil, err
		}
	}

	log.Printf("DeviceInfo - HwSerialID: %s, HwMacID: %s, HwIP: %s, DiskType: %s, LoadBalancerIP: %s, DpsSymmKey: %s",
		deviceInfo.HwSerialID, deviceInfo.HwMacID, deviceInfo.HwIP, deviceInfo.DiskType, deviceInfo.LoadBalancerIP, deviceInfo.DpsSymmKey)

	return deviceInfoList, nil
}
func MockStartOnboarding(ctx context.Context, deviceInfo utils.DeviceInfo) (string, error) {
	mu.Lock()
	defer mu.Unlock()
	// Increment the request counter for each incoming request
	requestCounter++
	fmt.Printf("Request Number: %d\n", requestCounter)

	// Step 1: Copy all request data to a variable using the DeepCopyOnboardingRequest function.
	copyOfRequest := utils.DeepCopyOnboardingRequest(onboardingRequest)
	currentUser, err := user.Current()
	if err != nil {
		fmt.Println("Error:", err)

	}

	// Construct the kubeconfig path
	kubeconfigPath := filepath.Join(currentUser.HomeDir, ".kube/config")

	fmt.Printf("Kubeconfig Path: %s\n", kubeconfigPath)

	var deviceInfoList []utils.DeviceInfo
	//Create the deviceInfoList and azure env files using the createDeviceInfoListNAzureEnv function.
	deviceInfoList, err = MockCreateDeviceInfoListNAzureEnv(copyOfRequest)
	if err != nil {
		fmt.Println("Error:", err)
	}

	var artifactinfo utils.ArtifactData
	artifactinfo = parseNGetBkcUrl(copyOfRequest)

	// Call the DeviceOnboardingManager function to manage the onboarding of devices
	onbtype := copyOfRequest.OnbParams.Env
	//log.Printf("Onbtype first function: %s", onbtype)
	targetDir := "../../../cmd/onboardingmgr"
	if err := utils.ChangeWorkingDirectory(targetDir); err != nil {
		log.Fatalf("Failed to change working directory: %v", err)
	}
	err = MockDeviceOnboardingManager(deviceInfoList, artifactinfo, kubeconfigPath, onbtype)
	if err != nil {
		fmt.Println("Error:", err)
	}
	result := fmt.Sprintf("Exited with success:")
	log.Println(result)
	return result, nil

}
func TestGenerateGatewayFromBaseIP(t *testing.T) {

	expectedGateway := "192.168.1.1"

	gateway := generateGatewayFromBaseIP(baseIP)

	assert.Equal(t, expectedGateway, gateway)

}

func TestCreateDeviceInfoListNAzureEnv(t *testing.T) {

	deviceInfoList, err := MockCreateDeviceInfoListNAzureEnv(onboardingRequest)

	assert.NoError(t, err)

	if len(deviceInfoList) == 0 {
		log.Println("Device info list is empty")
	}

}

func TestCreateAzureEnvFile(t *testing.T) {

	err := MockCreateAzureEnvFile(deviceInfo)

	assert.NoError(t, err)

}

func TestParseNGetBkcUrl(t *testing.T) {
	result := parseNGetBkcUrl(onboardingRequest)

	expectedResult := utils.ArtifactData{
		BkcUrl:        "https://os.package.url",
		BkcBasePkgUrl: "https://platform.package.url",
	}

	assert.Equal(t, result, expectedResult)

}

func TestMakeGETRequestWithRetry(t *testing.T) {

	err := MockMakeGETRequestWithRetry(serialNumber, pdip, caCertPath, certPath, guid)

	assert.NoError(t, err)
}

func TestDeviceOnboardingManagerNzt(t *testing.T) {

	err := MockDeviceOnboardingManagerNzt(deviceInfo, artifactInfo, "dummyKubeconfigPath")
	assert.NoError(t, err)

}
func TestDeviceOnboardingManager(t *testing.T) {

	onbtype := "NZT" // or "ZT"

	deviceInfoList, err := MockCreateDeviceInfoListNAzureEnv(onboardingRequest)
	assert.NoError(t, err)
	targetDir := "../../../cmd/onboardingmgr"
	if err := utils.ChangeWorkingDirectory(targetDir); err != nil {
		log.Fatalf("Failed to change working directory: %v", err)
	}
	err = MockDeviceOnboardingManager(deviceInfoList, artifactInfo, "dummyKubeconfigPath", onbtype)

	assert.Nil(t, err, "Expected no error")

}

func TestStartOnboarding(t *testing.T) {

	result, StartonboardErr := MockStartOnboarding(context.Background(), deviceInfo)
	expected := "Exited with success:"
	assert.Equal(t, result, expected)
	assert.NoError(t, StartonboardErr)
}
