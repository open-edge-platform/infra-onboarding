/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/invclient"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	dkam "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/api/grpc/dkammgr"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/os/v1"
	logging "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/onboardingmgr"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/onbworkflowclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/utils"
	"github.com/mohae/deepcopy"
	"google.golang.org/grpc"
)

var (
	clientName = "Onboarding"
	zlog       = logging.GetLogger(clientName)
	_dkamAddr  = "localhost:5581"
	_invClient *invclient.OnboardingInventoryClient
)

type OnboardingManager struct {
	pb.OnBoardingEBServer
}

type ResponseData struct {
	To2CompletedOn string `json:"to2CompletedOn"`
	To0Expiry      string `json:"to0Expiry"`
}

func InitOnboarding(invClient *invclient.OnboardingInventoryClient, dkamAddr string) {
	if invClient == nil {
		zlog.Debug().Msgf("Warning: invClient is nil")
		return
	}
	_invClient = invClient
	_dkamAddr = dkamAddr
}

var enableDI = flag.Bool("enableDI", false, "Set to true to enable Device Initialization routine")
var enableImgDownload = flag.Bool("enableImgDownload", false, "Set to true to enable Image Download")

func generateGatewayFromBaseIP(baseIP string) string {
	// Extract the last part of the base IP and replace it with "1" to get the gateway
	lastPart := strings.Split(baseIP, ".")[3]
	return baseIP[:len(baseIP)-len(lastPart)] + "1"
}

func createDeviceInfoListNAzureEnv(copyOfRequest *pb.OnboardingRequest) ([]utils.DeviceInfo, error) {
	var deviceInfoList []utils.DeviceInfo

	//TODO : Exported PDIP/LOAD_BALANCER_IP/DISK_PARITION instead of passing as parameters

	for _, hw := range copyOfRequest.Hwdata {
		deviceInfo := utils.DeviceInfo{
			HwSerialID:     hw.HwId,
			HwMacID:        hw.MacId,
			HwIP:           hw.SutIp,
			DiskType:       os.Getenv("DISK_PARTITION"),
			LoadBalancerIP: os.Getenv("IMG_URL"),
			Gateway:        generateGatewayFromBaseIP(hw.SutIp),
			ProvisionerIp:  os.Getenv("PD_IP"),
			ImType:         os.Getenv("IMAGE_TYPE"),
			RootfspartNo:   os.Getenv("OVERLAY_URL"),
			/* DpsScopeId:        hw.CusParams.DpsScopeId,
			DpsRegistrationId: hw.CusParams.DpsRegistrationId,
			DpsSymmKey:        hw.CusParams.DpsEnrollmentSymKey, */
		}
		deviceInfoList = append(deviceInfoList, deviceInfo)

		if deviceInfo.ImType == "prod_focal-ms" {
			err := createAzureEnvFile(deviceInfo)
			if err != nil {
				log.Fatalf("error while createing azure-credentials.env_%s is %v", deviceInfo.HwMacID, err)
				return nil, err
			}
		}

		// Log utils.DeviceInfo details
		log.Printf("DeviceInfo - HwSerialID: %s, HwMacID: %s, HwIP: %s, DiskType: %s, LoadBalancerIP: %s, DpsSymmKey: %s",
			deviceInfo.HwSerialID, deviceInfo.HwMacID, deviceInfo.HwIP, deviceInfo.DiskType, deviceInfo.LoadBalancerIP, deviceInfo.DpsSymmKey)
	}

	return deviceInfoList, nil
}

func createAzureEnvFile(deviceInfo utils.DeviceInfo) error {
	var content []byte
	fileName := "azure-credentials.env_" + deviceInfo.HwMacID
	dirPath, _ := os.Getwd()
	dirPath, _ = strings.CutSuffix(dirPath, "/cmd/onboardingmgr")
	content = fmt.Append(content, fmt.Sprintf("export ID_SCOPE=\"%s\"\n", deviceInfo.DpsScopeId))
	content = fmt.Append(content, fmt.Sprintf("export REGISTRATION_ID=\"%s\"\n", deviceInfo.DpsRegistrationId))
	content = fmt.Append(content, fmt.Sprintf("export SYMMETRIC_KEY=\"%s\"\n", deviceInfo.DpsSymmKey))
	err := os.WriteFile(dirPath+"/internal/onboardingmgr/azure_env/"+fileName, []byte(content), 0644)
	if err != nil {
		return err
	}
	return nil
}

func parseNGetBkcUrl(onboardingRequest *pb.OnboardingRequest) utils.ArtifactData {
	var artifactinfo utils.ArtifactData
	for _, artifactData := range onboardingRequest.ArtifactData {
		if artifactData.Category.String() == "OS" || artifactData.Name == "OS" {
			artifactinfo.BkcUrl = artifactData.PackageUrl
		} else if artifactData.Category.String() == "PLATFORM" || artifactData.Name == "PLATFORM" {
			artifactinfo.BkcBasePkgUrl = artifactData.PackageUrl
		} else {
			fmt.Printf("Unsupported category: %s\n", artifactData.Category.String())
			continue
		}
	}
	return artifactinfo
}

func DeviceOnboardingManagerZt(deviceInfo utils.DeviceInfo, kubeconfigPath string, sutlabel string) error {
	// for each device profile it will run
	// Open the file for appending
	file, err := os.OpenFile("sut_onboarding_list.txt", os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return err
	}
	defer file.Close()

	// Append device details to the file with the SUT label
	line := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", sutlabel, deviceInfo.HwMacID, deviceInfo.LoadBalancerIP, deviceInfo.HwIP, deviceInfo.DiskType, deviceInfo.ImType, deviceInfo.DpsScopeId, deviceInfo.DpsRegistrationId, deviceInfo.DpsSymmKey)
	_, err = file.WriteString(line)
	if err != nil {
		return err
	}

	return nil
}

func DeviceOnboardingManagerNzt(deviceDetails utils.DeviceInfo, artifactDetails utils.ArtifactData, nodeExistCount *int, totalNodes int, ErrCh chan error, ImgDownldLock, focalImgDdLock, jammyImgDdLock, focalMsImgDdLock *sync.Mutex) {
	log.Println("Onboarding is Triggered for ", deviceDetails.HwIP, *nodeExistCount, totalNodes)
	deviceInfo := deepcopy.Copy(deviceDetails).(utils.DeviceInfo)
	artifactinfo := deepcopy.Copy(artifactDetails).(utils.ArtifactData)
	log.Println("deviceInfo--", deviceInfo)
	log.Println("artifactinfo--", artifactinfo)

	var (
		imageDownloadErr, dierror error
		imageDownloadStatus       = make(chan struct{})
		diStatus                  = make(chan struct{})
		ctx                       = context.Background()
	)

	// ImageDownload goroutine
	go func() {
		if *enableImgDownload {
			defer close(imageDownloadStatus)
			log.Println("Image Download started for ", deviceInfo.HwIP)
			imageDownloadErr = onbworkflowclient.ImageDownload(artifactinfo, deviceInfo, ImgDownldLock, focalImgDdLock, jammyImgDdLock, focalMsImgDdLock)
			if imageDownloadErr != nil {
				imageDownloadErr = fmt.Errorf("SutIP %s: %w", deviceInfo.HwIP, imageDownloadErr)
				fmt.Printf("Error in ImageDownload for %v\n", imageDownloadErr)
				return
			}
			log.Println("Image Download Finished for ", deviceInfo.HwIP)
		} else {
			log.Printf("Image download disabled")
		}

	}()

	// DI goroutine
	go func() {
		// Check if DI routine should be executed
		if *enableDI {
			defer close(diStatus)
			log.Printf("Device initialization started for Device: %s", deviceInfo.HwIP)
			var guid string
			guid, dierror = onbworkflowclient.DiWorkflowCreation(deviceInfo)
			if dierror != nil {
				dierror = fmt.Errorf("SutIP %s: %w", deviceInfo.HwIP, dierror)
				fmt.Printf("Error in DiWorkflowCreation for %v\n", dierror)
				return
			}
			UpdateHostStatusByHostGuid(ctx, _invClient, guid, computev1.HostStatus_HOST_STATUS_PROVISIONING)
			log.Printf("GUID: %s\n", guid)
			deviceInfo.Guid = guid
			// TODO: change the certificate path to the common location once fdo services are working
			caCertPath := "/home/" + os.Getenv("USER") + "/.fdo-secrets/scripts/secrets/ca-cert.pem"
			certPath := "/home/" + os.Getenv("USER") + "/.fdo-secrets/scripts/secrets/api-user.pem"
			log.Println("----guid--------", deviceInfo.Guid)

			dierror = MakeGETRequestWithRetry(deviceInfo.HwSerialID, deviceInfo.ProvisionerIp, caCertPath, certPath, deviceInfo.Guid)
			if dierror != nil {
				fmt.Printf("Error while MakeGETRequestWithRetry T02: %v\n", dierror)
				dierror = fmt.Errorf("SutIP %s: %w", deviceInfo.HwIP, dierror)
				return
			}
			log.Printf("Device initialization completed for device: %s", deviceInfo.HwIP)
		} else {
			log.Printf("Device initialization disabled")
			//return
		}
	}()

	// Production Workflow goroutine
	go func() {
		//TODO:change this and pass the file naem instead of conversion
		log.Println("ProdWorkflowCreation triggired for ", deviceInfo.HwIP)
		imgurl := artifactinfo.BkcUrl
		filenameBz2 := filepath.Base(imgurl)
		filenameWithoutExt := strings.TrimSuffix(filenameBz2, ".bz2")
		deviceInfo.ClientImgName = filenameWithoutExt + ".raw.gz"

		log.Println("ProdWorkflowCreation Waiting for Image Download Status for node", deviceInfo.HwIP)

		log.Println("ProdWorkflowCreation Waiting for DI and TO Status for node", deviceInfo.HwIP)

		if *enableImgDownload {
			<-imageDownloadStatus
			<-diStatus
		}

		defer func() {
			if totalNodes == *nodeExistCount {
				close(ErrCh)
				log.Println("closed Errch channel")
			}
		}()

		if *enableImgDownload {
			if imageDownloadErr != nil {
				ErrCh <- imageDownloadErr
				*nodeExistCount += 1
				return
			}
			if dierror != nil {
				if deviceInfo.Guid != "" {
					UpdateHostStatusByHostGuid(ctx, _invClient, deviceInfo.Guid, computev1.HostStatus_HOST_STATUS_PROVISION_FAILED)
				}
				ErrCh <- dierror
				*nodeExistCount += 1
				return
			}

		} else {
			log.Println("DI is disabled")
		}
		UpdateHostStatusByHostGuid(ctx, _invClient, deviceInfo.Guid, computev1.HostStatus_HOST_STATUS_PROVISIONED)
		UpdateInstanceStatusByGuid(ctx, _invClient, deviceInfo.Guid, computev1.InstanceStatus_INSTANCE_STATUS_PROVISIONING)
		log.Println("ProdWorkflowCreation started for ", deviceInfo.HwIP)
		// Production Workflow creation
		proderror := onbworkflowclient.ProdWorkflowCreation(deviceInfo, deviceInfo.ImType)
		if proderror != nil {
			proderror = fmt.Errorf("SutIP %s: %w", deviceInfo.HwIP, proderror)
			ErrCh <- proderror
			*nodeExistCount += 1
			UpdateInstanceStatusByGuid(ctx, _invClient, deviceInfo.Guid, computev1.InstanceStatus_INSTANCE_STATUS_PROVISION_FAILED)
			return
		}
		UpdateInstanceStatusByGuid(ctx, _invClient, deviceInfo.Guid, computev1.InstanceStatus_INSTANCE_STATUS_PROVISIONED)
		log.Println("ProdWorkflowCreation Finished for ", deviceInfo.HwIP)
		*nodeExistCount += 1
	}()
	// TODO: Delete the hardware workflow remaining
}

func DeviceOnboardingManager(deviceInfoList []utils.DeviceInfo, artifactinfo utils.ArtifactData) error {

	// setup the sutonboarding file
	var (
		bkcImgDdLock     sync.Mutex
		focalImgDdLock   sync.Mutex
		jammyImgDdLock   sync.Mutex
		focalMsImgDdLock sync.Mutex
	)
	var CurrentDeviceList = make(map[string]string)
	nodeExistCount := 0
	ErrCh := make(chan error, len(deviceInfoList))
	for _, deviceInfo := range deviceInfoList {
		if _, found := CurrentDeviceList[deviceInfo.HwMacID]; !found {
			CurrentDeviceList[deviceInfo.HwMacID] = deviceInfo.HwIP
			DeviceOnboardingManagerNzt(deviceInfo, artifactinfo, &nodeExistCount, len(deviceInfoList), ErrCh, &bkcImgDdLock, &focalImgDdLock, &jammyImgDdLock, &focalMsImgDdLock)
		} else {
			nodeExistCount += 1
			log.Println("Duplicate Device from ther profile request", deviceInfo.HwIP)
		}
	}

	log.Println("Handling DeviceOnboardingManagerNzt errors if present----")
	for err := range ErrCh {
		log.Printf("Error while onboarding Node/SUT IP is %v\n", err)
	}

	log.Printf("Onboarding is completed")
	return nil
}

func MakeGETRequestWithRetry(serialNumber, pdip string, caCertPath, certPath string, guid string) error {
	timeout := 5 * time.Minute
	startTime := time.Now()
	for {
		if time.Since(startTime) >= timeout {
			return errors.New(" time out for T02 Process")
		}
		// Make an HTTP GET request
		response, err := utils.MakeHTTPGETRequest(pdip, guid, caCertPath, certPath)
		if err != nil {
			log.Fatalf("Error making HTTP GET request: %v", err)
		}

		if len(response) == 0 {
			log.Println("Empty response received. Retrying in 5 seconds...")
			time.Sleep(5 * time.Second)
			continue
		}

		// Unmarshal the JSON response
		responseData := ResponseData{}
		if err := json.Unmarshal(response, &responseData); err != nil {
			log.Fatalf("Error unmarshaling JSON: %v", err)
		}

		if responseData.To2CompletedOn != "" {
			// The "to2CompletedOn" field is not empty, indicating completion
			fmt.Println("to2CompletedOn:", responseData.To2CompletedOn)
			fmt.Println("to0Expiry:", responseData.To0Expiry)
			// Add your logic here, e.g., echo "$dev_serial CLIENT_SDK_TPM_TO2_SUCCESSFUL"
			break // Exit the loop when "to2CompletedOn" is completed
		}

		// If "to2CompletedOn" is still empty, wait for 5 seconds and then make the next request
		time.Sleep(5 * time.Second)
	}

	return nil
}

// Temporary extraction of manifest file from DKAM using hostname
func extractUrlsFromManifest(manifest string) (osUrl, overlayUrl string, err error) {
	// Define regular expressions to match the URLs
	osUrlRegex := regexp.MustCompile(`osurl:\s*(.*?)\n`)
	overlayUrlRegex := regexp.MustCompile(`overlayscripturl:\s*(.*?)\n`)

	// Find matches using the regular expressions
	osUrlMatches := osUrlRegex.FindStringSubmatch(manifest)
	overlayUrlMatches := overlayUrlRegex.FindStringSubmatch(manifest)

	if len(osUrlMatches) < 2 || len(overlayUrlMatches) < 2 {
		return "", "", fmt.Errorf("could not extract URLs from manifest")
	}

	return osUrlMatches[1], overlayUrlMatches[1], nil
}

func ConvertInstanceForOnboarding(instances []*computev1.InstanceResource, osinstances []*osv1.OperatingSystemResource, host *computev1.HostResource) ([]*pb.OnboardingRequest, error) {

	disableFeatureX := os.Getenv("DISABLE_FEATUREX")
	var onboardingRequests []*pb.OnboardingRequest
	var osUrl, overlayUrl string

	for _, osr := range osinstances {

		if disableFeatureX == "true" {
			_, _, urlerr := extractUrlsFromManifest(osr.RepoUrl)
			if urlerr != nil {
				zlog.Err(urlerr).Msg("Failed to extract URLs from manifest file")
				return nil, urlerr
			}

		}

		// Create an instance of OnboardingRequest and populate it
		onboardingRequest := &pb.OnboardingRequest{

			ArtifactData: []*pb.ArtifactData{
				{
					Name:       "OS",
					PackageUrl: osr.RepoUrl,
					Category:   1,
				},
				{
					Name:       "PLATFORM",
					PackageUrl: "overlayUrl",
					Category:   1,
				},
			},
			Hwdata: []*pb.HwData{
				{
					HwId:          host.GetUuid(),
					MacId:         host.GetPxeMac(),
					SutIp:         host.GetMgmtIp(),
					DiskPartition: "123", // Adjust these accordingly
					PlatformType:  host.GetHardwareKind(),
					// Add other hardware data if needed
				},
				// Add other hardware data if needed
			},
			//TODO : parameters has to be mapped accordingly
		}

		if disableFeatureX == "true" {
			onboardingRequest.ArtifactData[0].PackageUrl = osUrl
			onboardingRequest.ArtifactData[1].PackageUrl = overlayUrl
		}

		onboardingRequests = append(onboardingRequests, onboardingRequest)
	}

	// Return the onboarding request
	return onboardingRequests, nil
}

type GetArtifactsResponse struct {
	StatusCode   bool   `protobuf:"varint,1,opt,name=statusCode,proto3" json:"statusCode,omitempty"`
	ManifestFile string `protobuf:"bytes,2,opt,name=manifest_file,json=manifestFile,proto3" json:"manifest_file,omitempty"`
}

func GetOSResourceFromDkamService(ctx context.Context) (*dkam.GetArtifactsResponse, error) {

	// Get the DKAM manager host and port
	host := os.Getenv("DKAMHOST")
	port := os.Getenv("DKAMPORT")

	// Dial DKAM Manager API
	dkamAddr := fmt.Sprintf("%s:%s", host, port)

	log.Printf("DKAM Address: %s", dkamAddr)
	// Create a gRPC connection to DKAM server
	dkamConn, err := grpc.Dial(dkamAddr, grpc.WithInsecure())
	if err != nil {
		zlog.Err(err).Msg("Failed to connect to DKAM server")
		return nil, err
	}
	defer dkamConn.Close()

	// Create an instance of DkamServiceClient using the connection
	dkamClient := dkam.NewDkamServiceClient(dkamConn)
	response, err := dkamClient.GetArtifacts(ctx, &dkam.GetArtifactsRequest{
		// TODO: Pass relevant parameters
	})
	if err != nil {
		zlog.Err(err).Msg("Failed to get software details from DKAM")
		return nil, err
	}
	if response == nil {
		log.Fatalf("DKAM response is nil")
		return nil, errors.New("DKAM response is nil")
	}
	log.Printf("DKAM Response: %v", response)

	return response, nil
}

var mu sync.Mutex
var requestCounter int

func StartOnboard(req *pb.OnboardingRequest) (*pb.OnboardingResponse, error) {
	// Lock to ensure only one request is processed at a time
	mu.Lock()
	defer mu.Unlock()

	// Increment the request counter for each incoming request
	requestCounter++
	fmt.Printf("Request Number: %d\n", requestCounter)

	// Step 1: Copy all request data to a variable using the DeepCopyOnboardingRequest function.
	copyOfRequest := utils.DeepCopyOnboardingRequest(req)

	var deviceInfoList []utils.DeviceInfo
	//Create the deviceInfoList and azure env files using the createDeviceInfoListNAzureEnv function.
	deviceInfoList, err := createDeviceInfoListNAzureEnv(copyOfRequest)
	if err != nil {
		return nil, err
	}

	artifactinfo := parseNGetBkcUrl(copyOfRequest)

	// Call the DeviceOnboardingManager function to manage the onboarding of devices
	err = DeviceOnboardingManager(deviceInfoList, artifactinfo)
	if err != nil {
		return nil, err
	}
	result := "Success"
	return &pb.OnboardingResponse{Status: result}, nil
}

func (s *OnboardingManager) StartOnboarding(ctx context.Context, req *pb.OnboardingRequest) (*pb.OnboardingResponse, error) {
	//Moving changes to seperate function to enable both gRPC endpoint and onboarding manager to call from Instance Reconsile
	return StartOnboard(req)

}
