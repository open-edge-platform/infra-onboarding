/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onbworkflowclient

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"strings"
	"time"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/tinkerbell"
	tinkv1alpha1 "github.com/tinkerbell/tink/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	clientName = "WorkflowCreator"
	zlog       = logging.GetLogger(clientName)
)
var rvEnabled = flag.Bool("rvenabled", false, "Set to true if you have enabled rv")

func GenerateMacIDString(macID string) string {
	macWithoutColon := strings.ReplaceAll(macID, ":", "")
	return strings.ToLower(macWithoutColon)
}

func newK8SClient() (client.Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	if schemeErr := tinkv1alpha1.AddToScheme(scheme.Scheme); schemeErr != nil {
		return nil, schemeErr
	}

	kubeClient, err := client.New(config, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return nil, err
	}
	return kubeClient, nil
}

func readUIDFromFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func unsetEnvironmentVariables() {
	// List of environment variables to unset
	variablesToUnset := []string{"http_proxy", "https_proxy"}

	for _, variable := range variablesToUnset {
		if err := os.Unsetenv(variable); err != nil {
			fmt.Printf("Failed to unset %s: %v\n", variable, err)
		} else {
			fmt.Printf("Unset %s\n", variable)
		}
	}
}

func VoucherExtension(hostIP, deviceSerial string) (string, error) {
	// Construct the path to the script directory
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	scriptDir := usr.HomeDir + "/pri-fidoiot/component-samples/demo/scripts"

	// Change the current working directory to the script directory
	oldWorkingDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if chdirErr := os.Chdir(scriptDir); chdirErr != nil {
		return "", chdirErr
	}

	log.Printf("Job %s has completed", scriptDir)
	unsetEnvironmentVariables()
	variableName := "https_proxy"

	// Use os.LookupEnv to check if the environment variable is present
	val, present := os.LookupEnv(variableName)

	if present {
		fmt.Printf("%s env variable present with value: %s\n", variableName, val)
	} else {
		fmt.Printf("%s env variable not present\n", variableName)
	}

	// Make the script executable
	cmdChmod := exec.Command("chmod", "+x", "extend_upload.sh")
	if runErr := cmdChmod.Run(); runErr != nil {
		return "", runErr
	}
	fmt.Printf("host ip: %s\n", hostIP)

	// Run the shell script with arguments
	cmdExtendUpload := exec.Command("./extend_upload.sh", "-m", "sh", "-c",
		"./secrets/", "-e", "mtls", "-m", hostIP, "-o", hostIP, "-s", deviceSerial)

	output, err := cmdExtendUpload.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error executing script: %w, Output: %s", err, output)
	}

	fmt.Printf("Script Output: voucher done\n%s\n", output)
	// Create the GUID file path
	guidFilePath := deviceSerial + "_guid.txt"

	// Read the GUID from the file
	uid, err := readUIDFromFile(guidFilePath)
	if err != nil {
		log.Printf("Error reading UID from file: %v", err)
		// You can handle this error as needed, e.g., return an error or retry.
		return "", err
	}

	if err := os.Chdir(oldWorkingDir); err != nil {
		return "", err
	}

	return uid, nil
}

func VoucherScript(deviceinfo utils.DeviceInfo) (string, error) {
	var (
		attestationType  string
		mfgIp            string
		onrIp            string
		apiUser          string
		mfgApiPasswd     string
		onrApiPasswd     string
		mfgPort          string
		onrPort          string
		authType         string
		serialNo         string
		statusCode       int
		deviceGuid       []byte
		extendVoucher    []byte
		ownerCertificate []byte
	)

	attestationType = "SECP256R1"
	authType = "digest"
	mfgIp = deviceinfo.FdoMfgDNS
	onrIp = deviceinfo.FdoOwnerDNS
	mfgPort = deviceinfo.FdoMfgPort
	onrPort = deviceinfo.FdoOwnerPort
	serialNo = deviceinfo.HwSerialID

	//default values
	defaultAttestationType := "SECP256R1"
	defaultMfgIp := "mi-fdo-mfg"
	defaultOnrIp := "mi-fdo-owner"
	defaultApiUser := "apiUser"
	defaultMfgApiPasswd := ""
	defaultOnrApiPasswd := ""
	defaultmfgPort := "58039"
	defaultonrPort := "58042"

	if attestationType == "" {
		attestationType = defaultAttestationType
	}
	if mfgIp == "" {
		mfgIp = defaultMfgIp
	}
	if onrIp == "" {
		onrIp = defaultOnrIp
	}
	if apiUser == "" {
		apiUser = defaultApiUser
	}
	if mfgApiPasswd == "" {
		mfgApiPasswd = defaultMfgApiPasswd
	}
	if onrApiPasswd == "" {
		onrApiPasswd = defaultOnrApiPasswd
	}
	if mfgPort == "" {
		mfgPort = defaultmfgPort
	}
	if onrPort == "" {
		onrPort = defaultonrPort
	}
	if authType == "" {
		return "", fmt.Errorf("auth method is mandatory")
	}
	if serialNo == "" {
		return "", fmt.Errorf("serial number of device is mandatory")
	}
	// TODO : remove the use of Goto statement
api:
	//used to GET the certificate
	url := "http://" + onrIp + ":" + onrPort + "/api/v1/certificate?alias=" + attestationType
	resp, err := apiCalls("GET", url, authType, apiUser, onrApiPasswd, []byte{}, deviceinfo.HwMacID)
	if err != nil {
		return "", fmt.Errorf("Error1 Details:%v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		ownerCertificate, err = io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("error reading the file:%v", err)
		}
		statusCode = 0
	api2:
		//used GET the mfg voucher
		url = "http://" + mfgIp + ":" + mfgPort + "/api/v1/mfg/vouchers/" + serialNo
		resp, err := apiCalls("POST", url, authType, apiUser, mfgApiPasswd, ownerCertificate, deviceinfo.HwMacID)
		if err != nil {
			return "", fmt.Errorf("error Details:%v ", err)
		}
		if resp.StatusCode == http.StatusOK {
			extendVoucher, err = io.ReadAll(resp.Body)
			if err != nil {
				return "", fmt.Errorf("error writing the response to the file:%v", err)
			}
			statusCode = 0
		api3:
			//used GET the owner voucher
			url = "http://" + onrIp + ":" + onrPort + "/api/v1/owner/vouchers/"
			resp, err = apiCalls("POST", url, authType, apiUser, onrApiPasswd, extendVoucher, deviceinfo.HwMacID)
			if err != nil {
				return "", fmt.Errorf("error details :%v", err)
			}
			if resp.StatusCode == http.StatusOK {
				deviceGuid, err = io.ReadAll(resp.Body)
				if err != nil {
					return "", fmt.Errorf("error reading the file:%v", err)
				}
				statusCode = 0
			api4:
				if *rvEnabled {
					//starts TO0
					url := fmt.Sprintf("http://%s:%s/api/v1/to0/%s", onrIp, onrPort, deviceGuid)
					resp, err := apiCalls("GET", url, authType, apiUser, onrApiPasswd, deviceGuid, deviceinfo.HwMacID)
					if err != nil {
						return "", fmt.Errorf("error Details:%v", err)
					}
					if resp.StatusCode == http.StatusOK {
						return string(deviceGuid), nil
					} else {
						statusCode++
						if statusCode < 2 {
							goto api4
						}
						return "", fmt.Errorf("failure in triggering TO0 for %s with GUID %s ", serialNo, deviceGuid)
					}
				} else {
					return string(deviceGuid), nil
				}
			} else {
				statusCode++
				if statusCode < 2 {
					goto api3
				}
				return "", fmt.Errorf("failure in uploading voucher to owner for device with serial number %s with response code: %d", serialNo, resp.StatusCode)
			}
		} else {
			statusCode++
			if statusCode < 2 {
				goto api2
			}
			return "", fmt.Errorf("failure in getting extended voucher for device with serial number %s with response code: %d", serialNo, resp.StatusCode)
		}
	} else {
		statusCode++
		if statusCode < 2 {
			goto api
		}
		return "", fmt.Errorf("failure in getting owner certificate for type %s with response code: %d", attestationType, resp.StatusCode)
	}
}

func apiCalls(httpMethod, url, authType, apiUser, onrApiPasswd string, bodyData []byte, hwMac string) (*http.Response, error) {
	var client *http.Client
	reader := bytes.NewReader(bodyData)
	req, err := http.NewRequest(httpMethod, url, nil)
	if err != nil {
		return nil, err
	}
	if httpMethod == "POST" {
		req.Body = io.NopCloser(reader)
	}
	if strings.ToLower(authType) == "digest" {
		req.SetBasicAuth(apiUser, onrApiPasswd)
		client = &http.Client{}
	} else if strings.ToLower(authType) == "mtls" {
		return nil, fmt.Errorf("MTLS authentication is not supported over HTTP")
	} else {
		return nil, fmt.Errorf("provided Auth type is not valid")
	}
	req.Header.Add("Content-Type", "text/plain")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s request failed with status code:%s", httpMethod, resp.Status)
	}
	return resp, nil
}

func CalculateRootFS(imageType, diskDev string) string {
	rootFSPartNo := "1"

	if imageType == "bkc" {
		rootFSPartNo = "1"
	}

	// Use regular expression to check if diskDev ends with a numeric digit
	match, _ := regexp.MatchString(".*[0-9]$", diskDev)

	if match {
		return fmt.Sprintf("p%s", rootFSPartNo)
	}

	return rootFSPartNo
}

func ProdWorkflowCreation(deviceInfo utils.DeviceInfo, imgtype string, artifactinfo utils.ArtifactData, enableDI bool) error {
	zlog.Info().Msgf("ProdWorkflowCreation starting for host %s (IP: %s)",
		deviceInfo.GUID, deviceInfo.HwIP)

	kubeClient, err := newK8SClient()
	if err != nil {
		return err
	}

	var (
		// TODO: use context from reconciler once refactored to asynchronous handling
		ctx = context.Background()
		// TODO: use env variable to get namespace from Helm chart deployment
		ns       = "maestro-iaas-system"
		id       = GenerateMacIDString(deviceInfo.HwMacID)
		tmplName string
		tmplData []byte
	)
	const pollTimeDuration = 20 * time.Second

	if !enableDI {
		hw := tinkerbell.NewHardware("machine-"+id, ns, deviceInfo.HwMacID,
			deviceInfo.DiskType, deviceInfo.HwIP, deviceInfo.Gateway)

		if kubeCreateErr := kubeClient.Create(ctx, hw); kubeCreateErr != nil {
			return kubeCreateErr
		}

		fmt.Printf("hardware workflow applied hardwarename:%s", hw.Name)
		fmt.Printf("hardware workflow Image URL :%s", deviceInfo.LoadBalancerIP)
	}

	switch imgtype {
	case utils.ProdBkc:
		tmplName = fmt.Sprintf("bkc-%s-prod", id)
		deviceInfo.ClientImgName = "jammy-server-cloudimg-amd64.raw.gz"
		deviceInfo.ImType = "bkc"
		deviceInfo.Rootfspart = CalculateRootFS(deviceInfo.ImType, deviceInfo.DiskType)
		deviceInfo.LoadBalancerIP = artifactinfo.BkcURL
		deviceInfo.RootfspartNo = artifactinfo.BkcBasePkgURL
		tmplData, err = tinkerbell.NewTemplateDataProdBKC(tmplName, deviceInfo.Rootfspart, deviceInfo.RootfspartNo,
			deviceInfo.LoadBalancerIP, deviceInfo.HwIP, deviceInfo.Gateway, deviceInfo.ClientImgName, deviceInfo.ProvisionerIP, deviceInfo.SecurityFeature, deviceInfo.ClientID, deviceInfo.ClientSecret, enableDI)
		if err != nil {
			return err
		}
	case utils.ProdFocal:
		tmplName = fmt.Sprintf("focal-%s-prod", id)
		deviceInfo.ClientImgName = "focal-server-cloudimg-amd64.raw.gz"
		deviceInfo.ImType = "focal"
		deviceInfo.Rootfspart = CalculateRootFS(deviceInfo.ImType, deviceInfo.DiskType)
		tmplData, err = tinkerbell.NewTemplateDataProd(tmplName, deviceInfo.Rootfspart,
			deviceInfo.RootfspartNo, deviceInfo.LoadBalancerIP, deviceInfo.ProvisionerIP)
		if err != nil {
			return err
		}
	case utils.ProdFocalMs:
		tmplName = fmt.Sprintf("focal-ms-%s-prod", id)
		deviceInfo.ImType = "focal-ms"
		deviceInfo.Rootfspart = CalculateRootFS(deviceInfo.ImType, deviceInfo.DiskType)
		tmplData, err = tinkerbell.NewTemplateDataProdMS(tmplName, deviceInfo.Rootfspart, deviceInfo.RootfspartNo,
			deviceInfo.LoadBalancerIP, deviceInfo.HwIP, deviceInfo.Gateway, deviceInfo.HwMacID, deviceInfo.ProvisionerIP)
		if err != nil {
			return err
		}
	default:
		tmplName = fmt.Sprintf("focal-%s-prod", id)
		deviceInfo.ClientImgName = "jammy-server-cloudimg-amd64.raw.gz"
		deviceInfo.ImType = "jammy"
		deviceInfo.Rootfspart = CalculateRootFS(deviceInfo.ImType, deviceInfo.DiskType)
		tmplData, err = tinkerbell.NewTemplateDataProd(tmplName, deviceInfo.Rootfspart,
			deviceInfo.RootfspartNo, deviceInfo.LoadBalancerIP, deviceInfo.ProvisionerIP)
		if err != nil {
			return err
		}
	}

	fmt.Println("production workflow started.......................................")
	// have notification from sut
	log.Printf("ROOTFS_PART_NO %s /// ROOTFS_PARTITION %s", deviceInfo.RootfspartNo, deviceInfo.Rootfspart)

	tmpl := tinkerbell.NewTemplate(string(tmplData), tmplName, ns)
	if err := kubeClient.Create(ctx, tmpl); err != nil {
		return err
	}
	fmt.Printf("template workflow applied workflowname:%s", tmpl.Name)

	wf := tinkerbell.NewWorkflow(fmt.Sprintf("workflow-%s-prod", id), ns, deviceInfo.HwMacID)
	wf.Spec.HardwareRef = "machine-" + id
	wf.Spec.TemplateRef = fmt.Sprintf("%s-%s-prod", deviceInfo.ImType, id)
	if err := kubeClient.Create(ctx, wf); err != nil {
		return err
	}
	fmt.Printf("workflow applied workflowname:%s", wf.Name)

	check := func() (bool, error) {
		err := kubeClient.Get(ctx, types.NamespacedName{Namespace: wf.Namespace, Name: wf.Name}, wf)
		if err != nil {
			return false, err
		}
		log.Printf("Workflow %s state: %s\n", wf.Name, wf.Status.State)
		return strings.EqualFold(string(wf.Status.State), "STATE_SUCCESS"), nil
	}

	if err := wait.PollUntilContextTimeout(ctx, pollTimeDuration, time.Hour, false, func(_ context.Context) (bool, error) {
		success, err := check()
		if err != nil {
			log.Printf("Error checking workflow status: %v", err)
			return false, err
		}
		if success {
			log.Printf("Workflow %s has reached STATE_SUCCESS", wf.Name)
		}
		return success, nil
	}); err != nil {
		return fmt.Errorf("workflow did not reach STATE_SUCCESS: %w", err)
	}

	////////////////////////////////To workflow Cleanup//////////////////////////
	if err := kubeClient.Delete(ctx, tmpl); err != nil {
		log.Printf("error while deleting template: %v", err)
		return err
	}

	if err := kubeClient.Delete(ctx, wf); err != nil {
		log.Printf("error while deleting workflow: %v", err)
		return err
	}

	return nil
}

func DiWorkflowCreation(deviceInfo utils.DeviceInfo) (string, error) {
	kubeClient, err := newK8SClient()
	if err != nil {
		return "", err
	}

	// TODO: use context from reconciler once refactored to asynchronous handling
	ctx := context.Background()
	ns := "maestro-iaas-system"
	id := GenerateMacIDString(deviceInfo.HwMacID)
	const pollTimeDuration = 20 * time.Second

	hw := tinkerbell.NewHardware("machine-"+id, ns, deviceInfo.HwMacID,
		deviceInfo.DiskType, deviceInfo.HwIP, deviceInfo.Gateway)
	if kubeCreateErr := kubeClient.Create(ctx, hw); kubeCreateErr != nil {
		return "", kubeCreateErr
	}
	fmt.Printf("hardware workflow applied hardwarename:%s", hw.Name)

	tmplName := "fdodi-" + id
	tmplData, err := tinkerbell.NewTemplateData(tmplName, deviceInfo.HwIP, "CLIENT-SDK-TPM",
		deviceInfo.DiskType, deviceInfo.HwSerialID)
	if err != nil {
		return "", err
	}
	tmpl := tinkerbell.NewTemplate(string(tmplData), tmplName, ns)
	if kubeCreateErr := kubeClient.Create(ctx, tmpl); kubeCreateErr != nil {
		return "", kubeCreateErr
	}
	fmt.Printf("template workflow applied workflowname:%s", tmpl.Name)

	wf := tinkerbell.NewWorkflow("workflow-"+id, ns, deviceInfo.HwMacID)
	wf.Spec.HardwareRef = hw.Name
	wf.Spec.TemplateRef = tmpl.Name
	if kubeCreateErr := kubeClient.Create(ctx, wf); kubeCreateErr != nil {
		return "", kubeCreateErr
	}
	fmt.Printf("workflow applied workflowname:%s", wf.Name)

	check := func() (bool, error) {
		clientErr := kubeClient.Get(ctx, types.NamespacedName{Namespace: wf.Namespace, Name: wf.Name}, wf)
		if clientErr != nil {
			return false, clientErr
		}
		log.Printf("Workflow %s state: %s\n", wf.Name, wf.Status.State)
		return strings.EqualFold(string(wf.Status.State), "STATE_SUCCESS"), nil
	}

	if err = wait.PollUntilContextTimeout(ctx, pollTimeDuration, 2*time.Hour, false, func(_ context.Context) (bool, error) {
		success, statusErr := check()
		if statusErr != nil {
			log.Printf("Error checking workflow status: %v", statusErr)
			return false, statusErr
		}
		if success {
			log.Printf("Workflow %s has reached STATE_SUCCESS", wf.Name)
		}
		return success, nil
	}); err != nil {
		return "", fmt.Errorf("workflow did not reach STATE_SUCCESS: %w", err)
	}

	/////////////////////Voucher extension//////////////////////

	guid, err := VoucherScript(deviceInfo)
	if err != nil {
		// fmt.Printf("Error: %v\n", err)
		zlog.Err(err).Msg("Failed to  voucher Extension")
	} else {
		fmt.Printf("GUID: %s\n", guid)
		zlog.Info().Msgf("FDO-GUID  %s for the UUID  %s", guid, deviceInfo.GUID)
	}

	////////////////////////////////Di workflow Cleanup//////////////////////////
	if kubeDeleteErr := kubeClient.Delete(ctx, tmpl); kubeDeleteErr != nil {
		return "", kubeDeleteErr
	}

	if kubeDeleteErr := kubeClient.Delete(ctx, wf); kubeDeleteErr != nil {
		return "", kubeDeleteErr
	}

	return guid, nil
}
