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
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/util"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"

	om_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/status"

	dkam "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/api/grpc/dkammgr"

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
	pb.OnBoardingSBServer
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
			GUID:            hw.Uuid,
			HwSerialID:      hw.Serialnum,
			HwMacID:         hw.MacId,
			HwIP:            hw.SutIp,
			SecurityFeature: hw.SecurityFeature,
			DiskType:        os.Getenv("DISK_PARTITION"),
			LoadBalancerIP:  os.Getenv("IMG_URL"),
			Gateway:         generateGatewayFromBaseIP(hw.SutIp),
			ProvisionerIP:   os.Getenv("PD_IP"),
			ImType:          os.Getenv("IMAGE_TYPE"),
			RootfspartNo:    os.Getenv("OVERLAY_URL"),
			FdoMfgDNS:       os.Getenv("FDO_MFG_URL"),
			FdoOwnerDNS:     os.Getenv("FDO_OWNER_URL"),
			FdoMfgPort:      os.Getenv("FDO_MFG_PORT"),
			FdoOwnerPort:    os.Getenv("FDO_OWNER_PORT"),
			FdoRvPort:       os.Getenv("FDO_RV_PORT"),
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
) error {
	deviceInfo := deepcopy.Copy(deviceDetails).(utils.DeviceInfo)
	artifactinfo := deepcopy.Copy(artifactDetails).(utils.ArtifactData)
	zlog.Debug().Msgf("Onboarding triggered for host %s (IP: %s). Host info %+v, Artifact info %+v",
		deviceDetails.GUID, deviceDetails.HwIP, deviceInfo, artifactinfo)

	// Get client data
	clientSecret, clientID, err := onbworkflowclient.GetClientData(deviceInfo.GUID)
	if err != nil {
		return fmt.Errorf("error getting client data: %v", err)
	}
	deviceInfo.ClientID = clientID
	deviceInfo.ClientSecret = clientSecret
	var dierror error
	// var guid string
	if *enableDI {
		zlog.Debug().Msgf("Device initialization started for host %s (IP: %s)",
			deviceInfo.GUID, deviceInfo.HwIP)

		guid, workflowErr := onbworkflowclient.DiWorkflowCreation(deviceInfo)
		if workflowErr != nil {
			dierror = inv_errors.Errorf(
				"Error in DiWorkflowCreation for host %s (IP %s): %v",
				deviceInfo.GUID, deviceInfo.HwIP, workflowErr)

			UpdateHostStatusByHostGUID(ctx, _invClient, deviceInfo.GUID,
				computev1.HostStatus_HOST_STATUS_INIT_FAILED,
				"", // TODO: empty status details for now, add more details in future
				om_status.InitializationFailed)
			return dierror
		}
		zlog.Debug().Msgf("DI workflow creation succeeded for GUID %s", guid)

		UpdateHostStatusByHostGUID(ctx, _invClient, deviceInfo.GUID,
			computev1.HostStatus_HOST_STATUS_INITIALIZED,
			"", // TODO: empty status details for now, add more details in future
			om_status.InitializationDone)

		deviceInfo.FdoGUID = guid
		// TODO: change the certificate path to the common location once fdo services are working

		err := onbworkflowclient.InitializeDeviceSecretData(deviceInfo)
		if err != nil {
			log.Fatalf("Error initializing device: %v", err)
		}

		url := fmt.Sprintf("http://%s:%s/api/v1/owner/state/%s", deviceInfo.FdoOwnerDNS, deviceInfo.FdoOwnerPort, deviceInfo.FdoGUID)
		provisionErr := MakeGETRequestWithRetry(url, deviceInfo.FdoOwnerDNS, deviceInfo.FdoGUID)
		if provisionErr != nil {
			dierror = inv_errors.Errorf(
				"Error while MakeGETRequestWithRetry for host %s (IP %s): %v",
				deviceInfo.GUID, deviceInfo.HwIP, provisionErr)
			return dierror
		}
		zlog.Debug().Msgf("TO2 completed  for host %s", deviceInfo.GUID)

		zlog.Debug().Msgf("Device initialization completed for host %s (IP: %s)",
			deviceInfo.GUID, deviceInfo.HwIP)
		// Production Workflow goroutine
	}
	zlog.Debug().Msgf("ProdWorkflowCreation triggered for host %s", deviceInfo.GUID)

	UpdateHostStatusByHostGUID(ctx, _invClient, deviceInfo.GUID,
		computev1.HostStatus_HOST_STATUS_ONBOARDING,
		"", // TODO: empty status details for now, add more details in future
		om_status.OnboardingStatusInProgress)
	UpdateInstanceStatusByGUID(ctx, _invClient, deviceInfo.GUID,
		computev1.InstanceStatus_INSTANCE_STATUS_PROVISIONING, om_status.ProvisioningStatusInProgress)

	proderror := onbworkflowclient.ProdWorkflowCreation(deviceInfo, deviceInfo.ImType, artifactinfo, *enableDI)
	if proderror != nil {
		err := inv_errors.Errorf("Failed to create production workflow for host %s (IP: %s): %v",
			deviceInfo.GUID, deviceInfo.HwIP, proderror)
		UpdateInstanceStatusByGUID(ctx, _invClient, deviceInfo.GUID,
			computev1.InstanceStatus_INSTANCE_STATUS_PROVISION_FAILED, om_status.ProvisioningStatusFailed)
		return err
	}
	UpdateHostStatusByHostGUID(ctx, _invClient, deviceInfo.GUID,
		computev1.HostStatus_HOST_STATUS_ONBOARDED,
		"", // TODO: empty status details for now, add more details in future
		om_status.OnboardingStatusDone)
	UpdateInstanceStatusByGUID(ctx, _invClient, deviceInfo.GUID,
		computev1.InstanceStatus_INSTANCE_STATUS_PROVISIONED, om_status.ProvisioningStatusDone)

	zlog.Debug().Msgf("ProdWorkflowCreation finished for host %s", deviceInfo.GUID)
	return nil

	// TODO: Delete the hardware workflow remaining
}

func DeviceOnboardingManager(ctx context.Context, deviceInfoList []utils.DeviceInfo, artifactinfo utils.ArtifactData) error {
	for _, deviceInfo := range deviceInfoList {
		err := DeviceOnboardingManagerNzt(ctx, deviceInfo, artifactinfo)
		if err != nil {
			zlog.Debug().Msgf("OnboardingStatusFailed for host %s", deviceInfo.HwIP)
			UpdateHostStatusByHostGUID(ctx, _invClient, deviceInfo.GUID,
				computev1.HostStatus_HOST_STATUS_ONBOARDING_FAILED,
				"", // TODO: empty status details for now, add more details in future
				om_status.OnboardingStatusFailed)
			return err

		}
	}

	zlog.Debug().Msgf("Onboarding is completed")
	return nil
}

func MakeGETRequestWithRetry(to2URL, pdip, guid string) error {
	const timeOut = time.Hour
	const timeSleep = 5 * time.Second
	startTime := time.Now()
	for {
		if time.Since(startTime) >= timeOut {
			return inv_errors.Errorf("Timeout for T02 Process for host %s", guid)
		}

		fmt.Println("Making GET request to:", to2URL)
		response, err := http.Get(to2URL)
		if err != nil {
			respErr := inv_errors.Errorf("Error making HTTP GET request %v", err)
			zlog.MiSec().MiErr(err).Msgf("")
			return respErr
		}
		defer response.Body.Close()
		// Read response body
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return errors.New("error reading response body")
		}

		fmt.Println("Response:", string(body))

		if len(body) == 0 {
			time.Sleep(timeSleep)
			continue
		}
		// Unmarshal response JSON
		responseData := ResponseData{}
		if err := json.Unmarshal(body, &responseData); err != nil {
			zlog.MiSec().Err(err).Msgf("")
			return inv_errors.Errorf("Failed to perform GET request to %s for host %s",
				pdip, guid)
		}

		if responseData.To2CompletedOn != "" {
			// The "to2CompletedOn" field is not empty, indicating completion
			zlog.Debug().Msgf("Received response from %s. to2CompletedOn: %s, to0Expiry: %s",
				pdip, responseData.To2CompletedOn, responseData.To0Expiry)

			break // Exit the loop when "to2CompletedOn" is completed
		}
		// Add your logic here, e.g., echo "$dev_serial CLIENT_SDK_TPM_TO2_SUCCESSFUL"
		// If "to2CompletedOn" is still empty, wait for 5 seconds and then make the next request
		time.Sleep(timeSleep)
	}

	return nil
}
func ConvertInstanceForOnboarding(osResources []*osv1.OperatingSystemResource, host *computev1.HostResource, instances *computev1.InstanceResource) ([]*pb.OnboardingRequest, error) {
	var onboardingRequests []*pb.OnboardingRequest

	var overlayURL string

	bmcNics, err := util.GetBmcNicsFromHost(host)
	if err != nil {
		return nil, err
	}

	if len(bmcNics) > 1 {
		zlog.Warn().Msgf("Using the first BMC NIC, but more than one retrieved: %v.", bmcNics)
	}

	// we always assume that there is only one BMC NIC for a given host
	bmcNIC := bmcNics[0]

	for _, osr := range osResources {
		osURL := osr.RepoUrl

		invURL := strings.Split(osURL, ";")

		if len(invURL) > 0 {
			osURL = invURL[0]
		}
		// Validate the format of osURL
		if !isValidOSURLFormat(osURL) {
			return nil, inv_errors.Errorf("osURL %s is not in the expected format", osURL)
		}

		if len(invURL) > 1 {
			overlayURL = invURL[1]
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
					Serialnum:       host.GetSerialNumber(),
					SutIp:           host.GetBmcIp(),
					DiskPartition:   "123", // Adjust these accordingly
					PlatformType:    host.GetHardwareKind(),
					Uuid:            host.GetUuid(),
					SecurityFeature: uint32(instances.GetSecurityFeature()),
					// Add other hardware data if needed
				},
			},
		}

		// Set MAC address of HostnicResource if bmcInterface is true
		onboardingRequest.Hwdata[0].MacId = bmcNIC.MacAddr

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

func IsSecureBootConfigAtEdgeNodeMismatch(ctx context.Context, req *pb.SecureBootResponse) error {
	zlog.Info().Msgf("IsSecureBootConfigAtEdgeNodeMismatch")

	// Getting details from Host for now using GUID/UUID
	instanceDetails, err := _invClient.GetHostResourceByUUID(ctx, req.Guid)
	if err != nil {
		zlog.Err(err).Msg("Failed to get Instance Details")
		return err // Return the error if failed to get instance details
	}
	// Check if SecureBootStatus mismatches
	if ((instanceDetails.Instance.GetSecurityFeature().String() == "SECURITY_FEATURE_SECURE_BOOT_AND_FULL_DISK_ENCRYPTION") &&
		(req.Result.String() == "FAILURE")) ||
		((instanceDetails.Instance.GetSecurityFeature().String() == "SECURITY_FEATURE_UNSPECIFIED") &&
			(req.Result.String() == "SUCCESS")) ||
		((instanceDetails.Instance.GetSecurityFeature().String() == "SECURITY_FEATURE_NONE") &&
			(req.Result.String() == "SUCCESS")) {
		// If there's a mismatch, update the instance status to INSTANCE_STATE_ERROR
		err := UpdateInstanceStatusByGUID(ctx, _invClient, req.Guid, computev1.InstanceStatus_INSTANCE_STATUS_ERROR, om_status.OnboardingStatusFailed)
		if err != nil {
			zlog.Err(err).Msg("Failed to Update the instance status")
			return err
		}

		// Update host status with fail status and statusDetails
		err = UpdateHostStatusByHostGUID(ctx, _invClient, req.Guid, computev1.HostStatus_HOST_STATUS_BOOT_FAILED, "SecureBoot status mismatch", om_status.OnboardingStatusFailed)
		if err != nil {
			zlog.Err(err).Msg("Failed to Update the host status")
			return err
		}

		// Return an error indicating SecureBoot status mismatch
		return errors.New("SecureBoot status mismatch")
	}

	zlog.Info().Msgf("IsSecureBootConfigAtEdgeNodeMismatch(): SB flags matched")

	// If there's no error and no mismatch, return nil
	return nil
}

func (s *OnboardingManager) SecureBootStatus(ctx context.Context, req *pb.SecureBootStatRequest) (*pb.SecureBootResponse, error) {
	zlog.Info().Msgf("------- SecureBootStatus() ----------------\n")
	resp := &pb.SecureBootResponse{
		Guid:   req.Guid,
		Result: pb.SecureBootResponse_Status(req.Result),
	}
	//	err := HandleSecureBootMismatch(ctx, resp)
	err := IsSecureBootConfigAtEdgeNodeMismatch(ctx, resp)
	if err != nil {
		return resp, errors.New("SecureBoot Status mismatch")
	}
	return resp, nil
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
