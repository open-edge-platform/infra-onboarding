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
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	dkam "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/api/grpc/dkammgr"
	om_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/status"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/os/v1"
	logging "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/api"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/onbworkflowclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"

	"github.com/mohae/deepcopy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	clientName = "Onboarding"
	zlog       = logging.GetLogger(clientName)
	_invClient *invclient.OnboardingInventoryClient
)

type OnboardingManager struct {
	pb.OnBoardingEBServer
}

type ResponseData struct {
	To2CompletedOn string `json:"to2CompletedOn"`
	To0Expiry      string `json:"to0Expiry"`
}

func InitOnboarding(invClient *invclient.OnboardingInventoryClient, _ string) {
	if invClient == nil {
		zlog.Debug().Msgf("Warning: invClient is nil")
		return
	}
	_invClient = invClient
}

var (
	enableDI = flag.Bool("enableDI", false, "Set to true to enable Device Initialization routine")
)

const (
	instanceReconcilerLoggerName = "InstanceReconciler"
)

// Misc variables.
var (
	zlogInst = logging.GetLogger(instanceReconcilerLoggerName)
)

func generateGatewayFromBaseIP(baseIP string) string {
	// Extract the last part of the base IP and replace it with "1" to get the gateway
	lastPart := strings.Split(baseIP, ".")[3]
	return baseIP[:len(baseIP)-len(lastPart)] + "1"
}

func createDeviceInfoListNAzureEnv(copyOfRequest *pb.OnboardingRequest) ([]utils.DeviceInfo, error) {
	deviceInfoList := make([]utils.DeviceInfo, 0)

	// TODO : Exported PDIP/LOAD_BALANCER_IP/DISK_PARITION instead of passing as parameters

	for _, hw := range copyOfRequest.Hwdata {
		deviceInfo := utils.DeviceInfo{
			GUID:           hw.Uuid,
			HwSerialID:     hw.Serialnum,
			HwMacID:        hw.MacId,
			HwIP:           hw.SutIp,
			DiskType:       os.Getenv("DISK_PARTITION"),
			LoadBalancerIP: os.Getenv("IMG_URL"),
			Gateway:        generateGatewayFromBaseIP(hw.SutIp),
			ProvisionerIP:  os.Getenv("PD_IP"),
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
				zlog.MiErr(err).Msgf("error while createing azure-credentials.env_%s", deviceInfo.HwMacID)
				return nil, err
			}
		}

		zlog.MiSec().Debug().Msgf("DeviceInfo added to the list: %+v", deviceInfo)
	}

	return deviceInfoList, nil
}

func createAzureEnvFile(deviceInfo utils.DeviceInfo) error {
	var content []byte
	const filePermission = 0o644
	fileName := "azure-credentials.env_" + deviceInfo.HwMacID
	dirPath, _ := os.Getwd()
	dirPath, _ = strings.CutSuffix(dirPath, "/cmd/onboardingmgr")
	content = fmt.Append(content, fmt.Sprintf("export ID_SCOPE=%q\n", deviceInfo.DpsScopeID))
	content = fmt.Append(content, fmt.Sprintf("export REGISTRATION_ID=%q\n", deviceInfo.DpsRegistrationID))
	content = fmt.Append(content, fmt.Sprintf("export SYMMETRIC_KEY=%q\n", deviceInfo.DpsSymmKey))
	err := os.WriteFile(dirPath+"/internal/onboardingmgr/azure_env/"+fileName, content, filePermission)
	if err != nil {
		return err
	}
	return nil
}

func parseNGetBkcURL(onboardingRequest *pb.OnboardingRequest) utils.ArtifactData {
	var artifactinfo utils.ArtifactData
	for _, artifactData := range onboardingRequest.ArtifactData {
		category := artifactData.Category.String()
		switch {
		case category == "OS" || artifactData.Name == "OS":
			artifactinfo.BkcURL = artifactData.PackageUrl
		case category == "PLATFORM" || artifactData.Name == "PLATFORM":
			artifactinfo.BkcBasePkgURL = artifactData.PackageUrl
		default:
			zlog.Warn().Msgf("Unsupported category: %s, continuing", category)
			continue
		}
	}
	return artifactinfo
}

func DeviceOnboardingManagerZt(deviceInfo utils.DeviceInfo, sutlabel string) error {
	// for each device profile it will run
	// Open the file for appending
	file, err := os.OpenFile("sut_onboarding_list.txt", os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return err
	}
	defer file.Close()

	// Append device details to the file with the SUT label
	line := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", sutlabel, deviceInfo.HwMacID,
		deviceInfo.LoadBalancerIP, deviceInfo.HwIP, deviceInfo.DiskType, deviceInfo.ImType,
		deviceInfo.DpsScopeID, deviceInfo.DpsRegistrationID, deviceInfo.DpsSymmKey)
	_, err = file.WriteString(line)
	if err != nil {
		return err
	}

	return nil
}

func DeviceOnboardingManagerNzt(ctx context.Context, deviceDetails utils.DeviceInfo, artifactDetails utils.ArtifactData,
	nodeExistCount *int, totalNodes int, errCh chan error,
) {
	deviceInfo := deepcopy.Copy(deviceDetails).(utils.DeviceInfo)
	artifactinfo := deepcopy.Copy(artifactDetails).(utils.ArtifactData)
	zlog.Debug().Msgf("Onboarding triggered for host %s (IP: %s). Host info %+v, Artifact info %+v",
		deviceDetails.GUID, deviceDetails.HwIP, deviceInfo, artifactinfo)

	var dierror error
	// DI goroutine
	go func() {
		// Check if DI routine should be executed
		if *enableDI {
			zlog.Debug().Msgf("Device initialization started for host %s (IP: %s)",
				deviceInfo.GUID, deviceInfo.HwIP)

			UpdateHostStatusByHostGUID(ctx, _invClient, deviceInfo.GUID,
				computev1.HostStatus_HOST_STATUS_INITIALIZING,
				"", // TODO: empty status details for now, add more details in future
				om_status.InitializationInProgress)
			guid, workflowErr := onbworkflowclient.DiWorkflowCreation(deviceInfo)
			if workflowErr != nil {
				dierror = inv_errors.Errorf(
					"Error in DiWorkflowCreation for host %s (IP %s): %v",
					deviceInfo.GUID, deviceInfo.HwIP, workflowErr)

				UpdateHostStatusByHostGUID(ctx, _invClient, deviceInfo.GUID,
					computev1.HostStatus_HOST_STATUS_INIT_FAILED,
					"", // TODO: empty status details for now, add more details in future
					om_status.InitializationFailed)
				return
			}
			zlog.Debug().Msgf("DI workflow creation succeeded for GUID %s", guid)

			UpdateHostStatusByHostGUID(ctx, _invClient, deviceInfo.GUID,
				computev1.HostStatus_HOST_STATUS_INITIALIZED,
				"", // TODO: empty status details for now, add more details in future
				om_status.InitializationDone)

			deviceInfo.GUID = guid
			// TODO: change the certificate path to the common location once fdo services are working
			caCertPath := "/home/" + os.Getenv("USER") + "/.fdo-secrets/scripts/secrets/ca-cert.pem"
			certPath := "/home/" + os.Getenv("USER") + "/.fdo-secrets/scripts/secrets/api-user.pem"

			provisionErr := MakeGETRequestWithRetry(deviceInfo.ProvisionerIP, caCertPath, certPath, deviceInfo.GUID)
			if provisionErr != nil {
				dierror = inv_errors.Errorf(
					"Error while MakeGETRequestWithRetry for host %s (IP %s): %v",
					deviceInfo.GUID, deviceInfo.HwIP, provisionErr)
				return
			}

			zlog.Debug().Msgf("Device initialization completed for host %s (IP: %s)",
				deviceInfo.GUID, deviceInfo.HwIP)
		} else {
			zlog.Warn().Msgf("Device initialization disabled")
		}
	}()

	// Production Workflow goroutine
	go func() {
		zlog.Debug().Msgf("ProdWorkflowCreation triggered for host %s", deviceInfo.GUID)

		// TODO:change this and pass the file naem instead of conversion
		imgurl := artifactinfo.BkcURL
		filenameBz2 := filepath.Base(imgurl)
		filenameWithoutExt := strings.TrimSuffix(filenameBz2, ".bz2")
		deviceInfo.ClientImgName = filenameWithoutExt + ".raw.gz"

		defer func() {
			if totalNodes == *nodeExistCount {
				close(errCh)
				zlog.Debug().Msgf("Closed device onboarding error channel")
			}
		}()

		if dierror != nil {
			if deviceInfo.GUID != "" {
				UpdateHostStatusByHostGUID(ctx, _invClient, deviceInfo.GUID,
					computev1.HostStatus_HOST_STATUS_ONBOARDING_FAILED,
					"", // TODO: empty status details for now, add more details in future
					om_status.OnboardingStatusFailed)
			}
			errCh <- dierror
			*nodeExistCount++
			return
		}

		UpdateHostStatusByHostGUID(ctx, _invClient, deviceInfo.GUID,
			computev1.HostStatus_HOST_STATUS_ONBOARDING,
			"", // TODO: empty status details for now, add more details in future
			om_status.OnboardingStatusInProgress)
		UpdateInstanceStatusByGUID(ctx, _invClient, deviceInfo.GUID,
			computev1.InstanceStatus_INSTANCE_STATUS_PROVISIONING, om_status.ProvisioningStatusInProgress)

		proderror := onbworkflowclient.ProdWorkflowCreation(deviceInfo, deviceInfo.ImType, artifactinfo)
		if proderror != nil {
			err := inv_errors.Errorf("Failed to create production workflow for host %s (IP: %s): %v",
				deviceInfo.GUID, deviceInfo.HwIP, proderror)
			errCh <- err
			*nodeExistCount++
			UpdateInstanceStatusByGUID(ctx, _invClient, deviceInfo.GUID,
				computev1.InstanceStatus_INSTANCE_STATUS_PROVISION_FAILED, om_status.ProvisioningStatusFailed)
			return
		}
		UpdateHostStatusByHostGUID(ctx, _invClient, deviceInfo.GUID,
			computev1.HostStatus_HOST_STATUS_ONBOARDED,
			"", // TODO: empty status details for now, add more details in future
			om_status.OnboardingStatusDone)
		UpdateInstanceStatusByGUID(ctx, _invClient, deviceInfo.GUID,
			computev1.InstanceStatus_INSTANCE_STATUS_PROVISIONED, om_status.ProvisioningStatusDone)

		zlog.Debug().Msgf("ProdWorkflowCreation finished for host %s", deviceInfo.GUID)
		*nodeExistCount++
	}()
	// TODO: Delete the hardware workflow remaining
}

func DeviceOnboardingManager(ctx context.Context, deviceInfoList []utils.DeviceInfo, artifactinfo utils.ArtifactData) error {
	// setup the sutonboarding file
	CurrentDeviceList := make(map[string]string)
	nodeExistCount := 0
	ErrCh := make(chan error, len(deviceInfoList))
	for _, deviceInfo := range deviceInfoList {
		if _, found := CurrentDeviceList[deviceInfo.HwMacID]; !found {
			CurrentDeviceList[deviceInfo.HwMacID] = deviceInfo.HwIP
			DeviceOnboardingManagerNzt(ctx, deviceInfo, artifactinfo,
				&nodeExistCount, len(deviceInfoList), ErrCh)
		} else {
			nodeExistCount++
			zlog.Warn().Msgf("Duplicate host %s from the profile request", deviceInfo.GUID)
		}
	}

	for err := range ErrCh {
		zlog.MiSec().MiErr(err).Msg("Error while onboarding host")
	}

	zlog.Debug().Msgf("Onboarding is completed")
	return nil
}

// TODO: make this function asynchronous once reconciler is refactored
func MakeGETRequestWithRetry(pdip, caCertPath, certPath, guid string) error {
	const timeOut = 5 * time.Minute
	const timeSleep = 5 * time.Second
	startTime := time.Now()
	for {
		if time.Since(startTime) >= timeOut {
			return errors.New(" time out for T02 Process")
		}
		// Make an HTTP GET request
		response, err := utils.MakeHTTPGETRequest(pdip, guid, caCertPath, certPath)
		if err != nil {
			respErr := inv_errors.Errorf("Error making HTTP GET request %v", err)
			zlog.MiSec().MiErr(err).Msgf("")
			return respErr
		}

		if len(response) == 0 {
			zlog.Debug().Msgf("Empty response received for IP %s and host %s, "+
				"retrying in %d seconds...", pdip, guid, timeSleep)
			time.Sleep(timeSleep)
			continue
		}

		// Unmarshal the JSON response
		responseData := ResponseData{}
		if jsonErr := json.Unmarshal(response, &responseData); jsonErr != nil {
			zlog.MiSec().Err(jsonErr).Msgf("")
			return inv_errors.Errorf("Failed to perform GET request to %s for host %s",
				pdip, guid)
		}

		if responseData.To2CompletedOn != "" {
			// The "to2CompletedOn" field is not empty, indicating completion
			zlog.Debug().Msgf("Received response from %s. to2CompletedOn: %s, to0Expiry: %s",
				pdip, responseData.To2CompletedOn, responseData.To0Expiry)
			// Add your logic here, e.g., echo "$dev_serial CLIENT_SDK_TPM_TO2_SUCCESSFUL"
			break // Exit the loop when "to2CompletedOn" is completed
		}

		// If "to2CompletedOn" is still empty, wait for 5 seconds and then make the next request
		time.Sleep(timeSleep)
	}

	return nil
}

func ConvertInstanceForOnboarding(osResources []*osv1.OperatingSystemResource, host *computev1.HostResource) ([]*pb.OnboardingRequest, error) {
	var onboardingRequests []*pb.OnboardingRequest

	var overlayURL string
	hostNics := host.GetHostNics()
	for _, osr := range osResources {
		osURL := osr.RepoUrl

		invURL := strings.Split(osURL, ";")

		if len(invURL) > 0 {
			osURL = invURL[0]
		}
		// Validate the format of osURL
		if !isValidOSURLFormat(osURL) {
			return nil, errors.New("osURL is not in the expected format")
		}

		if len(invURL) > 1 {
			overlayURL = invURL[1]
		}

		// Check if hostNics is empty
		if len(hostNics) == 0 {
			return nil, errors.New("no macID found")
		}

		// Check if the HostnicResource has bmcInterface set to true
		if !hostNics[0].BmcInterface {
			return nil, errors.New("BMC interface is not enabled")
		}

		sutIP := host.GetBmcIp()
		// Replace #host_ip with SUT IP address in osURL and overlayURL
		osURL = replaceHostIP(osURL, sutIP)
		overlayURL = replaceHostIP(overlayURL, sutIP)

		// Create an instance of OnboardingRequest and populate it
		onboardingRequest := &pb.OnboardingRequest{
			ArtifactData: []*pb.ArtifactData{
				{
					Name:       "OS",
					PackageUrl: osURL,
					Category:   1,
				},
				{
					Name:       "PLATFORM",
					PackageUrl: overlayURL,
					Category:   1,
				},
			},
			Hwdata: []*pb.HwData{
				{
					Serialnum:     host.GetSerialNumber(),
					SutIp:         host.GetBmcIp(),
					DiskPartition: "123", // Adjust these accordingly
					PlatformType:  host.GetHardwareKind(),
					Uuid:          host.GetUuid(),
					// Add other hardware data if needed
				},
			},
		}

		// Set MAC address of HostnicResource if bmcInterface is true
		onboardingRequest.Hwdata[0].MacId = hostNics[0].MacAddr

		zlog.Debug().Msgf("Instance resource converted to onboarding request (MAC=%s, OS URL=%s, Overlay URL=%s",
			onboardingRequest.Hwdata[0].MacId,
			onboardingRequest.ArtifactData[0].PackageUrl,
			onboardingRequest.ArtifactData[1].PackageUrl)

		onboardingRequests = append(onboardingRequests, onboardingRequest)
	}

	// Return the onboarding requests
	return onboardingRequests, nil
}

func replaceHostIP(url, ip string) string {
	// Define the regular expression pattern to match #host_ip
	re := regexp.MustCompile(`%host_ip%`)
	return re.ReplaceAllString(url, ip)
}

// TODO : Will scale it in future accordingly
func isValidOSURLFormat(osURL string) bool {
	expectedSuffix := ".raw.gz" // Checks if the OS URL is in the expected format
	return strings.HasSuffix(osURL, expectedSuffix)
}

type GetArtifactsResponse struct {
	StatusCode   bool   `protobuf:"varint,1,opt,name=statusCode,proto3" json:"statusCode,omitempty"`
	ManifestFile string `protobuf:"bytes,2,opt,name=manifest_file,json=manifestFile,proto3" json:"manifest_file,omitempty"`
}

func GetOSResourceFromDkamService(ctx context.Context, profilename, platform string) (*dkam.GetArtifactsResponse, error) {
	// Get the DKAM manager host and port
	host := os.Getenv("DKAMHOST")
	port := os.Getenv("DKAMPORT")

	if host == "" || port == "" {
		err := inv_errors.Errorf("DKAM endpoint is not set")
		zlog.MiErr(err).Msgf("")
		return nil, err
	}

	// Dial DKAM Manager API
	dkamAddr := fmt.Sprintf("%s:%s", host, port)

	// Create a gRPC connection to DKAM server
	dkamConn, err := grpc.Dial(dkamAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("Failed to connect to DKAM server %s, retry in next iteration...", dkamAddr)
		return nil, inv_errors.Errorf("Failed to connect to DKAM server")
	}

	defer dkamConn.Close()

	// Create an instance of DkamServiceClient using the connection
	dkamClient := dkam.NewDkamServiceClient(dkamConn)
	response, err := dkamClient.GetArtifacts(ctx, &dkam.GetArtifactsRequest{
		ProfileName: profilename,
		Platform:    platform,
	})
	if err != nil {
		zlog.Err(err).Msg("Failed to get software details from DKAM")
		return nil, err
	}
	if response == nil {
		responseErr := inv_errors.Errorf("DKAM response is nil")
		zlog.MiErr(responseErr).Msg("")
		return nil, responseErr
	}

	zlog.Debug().Msgf("Software details successfully obtained from DKAM: %v", response)

	return response, nil
}

var (
	requestCounter int
)

func StartOnboard(ctx context.Context, req *pb.OnboardingRequest, resID string) (*pb.OnboardingResponse, error) {

	// Increment the request counter for each incoming request
	requestCounter++

	zlog.Info().Msgf("Starting onboarding for %v (counter %d)", req.Hwdata, requestCounter)

	// Step 1: Copy all request data to a variable using the DeepCopyOnboardingRequest function.
	copyOfRequest := utils.DeepCopyOnboardingRequest(req)

	var deviceInfoList []utils.DeviceInfo
	// Create the deviceInfoList and azure env files using the createDeviceInfoListNAzureEnv function.
	deviceInfoList, err := createDeviceInfoListNAzureEnv(copyOfRequest)
	if err != nil {
		return nil, err
	}

	artifactinfo := parseNGetBkcURL(copyOfRequest)

	// Call the DeviceOnboardingManager function to manage the onboarding of devices
	err = DeviceOnboardingManager(ctx, deviceInfoList, artifactinfo)
	if err != nil {
		zlogInst.MiSec().MiErr(err).Msgf("Failed to StartOnboard by ID %s", resID)
		return nil, err
	}

	inst := &computev1.InstanceResource{
		ResourceId:   resID,
		CurrentState: computev1.InstanceState_INSTANCE_STATE_RUNNING,
	}

	err = _invClient.UpdateInstanceCurrentState(ctx, inst)
	if err != nil {
		zlogInst.MiSec().MiErr(err).Msgf("Failed to Get Host Resource by ID %s", resID)
		return nil, err
	}

	result := "Success"
	return &pb.OnboardingResponse{Status: result}, nil
}

func (s *OnboardingManager) StartOnboarding(ctx context.Context, req *pb.OnboardingRequest) (*pb.OnboardingResponse, error) {
	// Moving changes to separate function to enable both gRPC endpoint and onboarding manager to call from Instance Reconcile
	// This endpoint is only for internal testing, will be removed if end-to-end flow works properly.
	_, err := StartOnboard(ctx, req, "")
	if err != nil {
		// Handle error
		zlogInst.MiSec().MiErr(err).Msgf("Failed to StartOnboarding")
		return nil, err
	}

	result := "Success"
	return &pb.OnboardingResponse{Status: result}, nil
}
