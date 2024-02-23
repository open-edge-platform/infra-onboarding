/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onbworkflowclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
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

func VoucherScript(hostIP, deviceSerial string) (string, error) {
	var (
		attestationType string
		mfgIP           string
		onrIP           string
		apiUser         string
		mfgAPIPasswd    string
		onrAPIPasswd    string
		mfgPort         string
		onrPort         string
		serialNo        string
		certPath        string
	)
	attestationType = "SECP256R1"
	mfgIP = hostIP
	onrIP = hostIP
	serialNo = deviceSerial
	// default values
	defaultAttestationType := "SECP256R1"
	defaultMfgIP := "localhost"
	defaultOnrIP := "localhost"
	defaultAPIUser := "apiUser"
	defaultMfgAPIPasswd := ""
	defaultOnrAPIPasswd := ""
	mfgPort = "8038"
	onrPort = "8043"
	if attestationType == "" {
		attestationType = defaultAttestationType
	}
	if mfgIP == "" {
		mfgIP = defaultMfgIP
	}
	if onrIP == "" {
		onrIP = defaultOnrIP
	}
	if apiUser == "" {
		apiUser = defaultAPIUser
	}
	if mfgAPIPasswd == "" {
		mfgAPIPasswd = defaultMfgAPIPasswd
	}
	if onrAPIPasswd == "" {
		onrAPIPasswd = defaultOnrAPIPasswd
	}
	if serialNo == "" {
		zlog.Debug().Msgf("Serial number of device is mandatory, ")
		os.Exit(0)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		zlog.Debug().Msgf("%v", err)
	}
	// TODO: modify the path for certificates
	certPath = homeDir + "/.fdo-secrets/scripts/secrets"
	url := "https://" + onrIP + ":" + onrPort + "/api/v1/certificate?alias=" + attestationType
	resp1, err := apiCalls("GET", url, apiUser, onrAPIPasswd, certPath, []byte{})
	if err != nil {
		return "", fmt.Errorf("Error1 Details:%w", err)
	}
	defer resp1.Body.Close()
	if resp1.StatusCode == http.StatusOK {
		file, err := os.Create(fmt.Sprintf("/home/%s/.fdo-secrets/scripts/owner_cert_%s.txt",
			os.Getenv("USER"), attestationType))
		if err != nil {
			return "", fmt.Errorf("error creating the file:%w", err)
		}
		defer file.Close()
		_, err = io.Copy(file, resp1.Body)
		if err != nil {
			return "", fmt.Errorf("error writing the response to the file:%w", err)
		}
		fmt.Printf("Success in downloading %s owner certificate to owner_cert_%s.txt\n", attestationType, attestationType)
		ownerCertificate, err := os.ReadFile(fmt.Sprintf("/home/%s/.fdo-secrets/scripts/owner_cert_%s.txt",
			os.Getenv("USER"), attestationType))
		if err != nil {
			return "", fmt.Errorf("error reading the file:%w", err)
		}
		url = "https://" + mfgIP + ":" + mfgPort + "/api/v1/mfg/vouchers/" + serialNo
		zlog.Debug().Msgf(url)
		resp2, err := apiCalls("POST", url, apiUser, mfgAPIPasswd, certPath, ownerCertificate)
		if err != nil {
			zlog.Debug().Msgf("%v", err)
			return "", fmt.Errorf("error Details:%w", err)
		}
		defer resp2.Body.Close()
		if resp2.StatusCode == http.StatusOK {
			file1, err := os.Create(fmt.Sprintf("/home/%s/.fdo-secrets/scripts/%s_voucher.txt", os.Getenv("USER"), serialNo))
			if err != nil {
				return "", fmt.Errorf("error creating the file:%w", err)
			}
			defer file1.Close()
			_, err = io.Copy(file1, resp2.Body)
			if err != nil {
				return "", fmt.Errorf("error writing the response to the file:%w", err)
			}
			fmt.Printf("Success in downloading extended voucher for device with serial number:%s\n", serialNo)
			extendVoucher, err := os.ReadFile(fmt.Sprintf("/home/%s/.fdo-secrets/scripts/%s_voucher.txt",
				os.Getenv("USER"), serialNo))
			if err != nil {
				return "", fmt.Errorf("error reading the file:%w", err)
			}
			url = "https://" + onrIP + ":" + onrPort + "/api/v1/owner/vouchers/"
			resp3, err := apiCalls("POST", url, apiUser, onrAPIPasswd, certPath, extendVoucher)
			if err != nil {
				return "", fmt.Errorf("error details :%w", err)
			}
			defer resp3.Body.Close()
			if resp3.StatusCode == http.StatusOK {
				file2, err := os.Create(fmt.Sprintf("/home/%s/.fdo-secrets/scripts/%s_guid.txt", os.Getenv("USER"), serialNo))
				if err != nil {
					zlog.Debug().Msgf("Error creating the file: %v", err)
					return "", fmt.Errorf("error creating the file:%w", err)
				}
				_, err = io.Copy(file2, resp3.Body)
				if err != nil {
					return "", fmt.Errorf("error writing the response to the file:%w", err)
				}
				deviceGUID, err := os.ReadFile(fmt.Sprintf("/home/%s/.fdo-secrets/scripts/%s_guid.txt",
					os.Getenv("USER"), serialNo))
				if err != nil {
					return "", fmt.Errorf("error reading the file:%w", err)
				}
				url := fmt.Sprintf("https://%s:%s/api/v1/to0/%s", onrIP, onrPort, deviceGUID)
				resp4, err := apiCalls("GET", url, apiUser, onrAPIPasswd, certPath, deviceGUID)
				if err != nil {
					return "", fmt.Errorf("error Details:%w", err)
				}
				defer resp4.Body.Close()
				if resp4.StatusCode == http.StatusOK {
					fmt.Printf("Success in triggering TO0 for %s with GUID %s\n", serialNo, deviceGUID)
					return string(deviceGUID), nil
				}
				return "", fmt.Errorf("failure in triggering TO0 for %s  with GUID %s ", serialNo, deviceGUID)
			}
			return "", fmt.Errorf("failure in uploading voucher to owner for device with serial number"+
				" %s with response code: %d", serialNo, resp3.StatusCode)
		}
		return "", fmt.Errorf("failure in getting extended voucher for device with serial number %s with response code: %d",
			serialNo, resp2.StatusCode)
	}
	return "", fmt.Errorf("failure in getting owner certificate for type %s with response code: %d",
		attestationType, resp1.StatusCode)
}

func apiCalls(httpMethod, url, apiUser, onrAPIPasswd, certPath string, bodyData []byte) (*http.Response, error) {
	var httpClient *http.Client
	authType := "mtls"
	reader := bytes.NewReader(bodyData)
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, httpMethod, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	if httpMethod == "POST" {
		req.Body = io.NopCloser(reader)
	}
	switch strings.ToLower(authType) {
	case "digest":
		zlog.Debug().Msgf("Digest authentication mode is being used")
		req.SetBasicAuth(apiUser, onrAPIPasswd)
		httpClient = &http.Client{}
	case "mtls":
		zlog.Debug().Msgf("Client Certificate authentication mode is being used")
		var caCert []byte
		caCert, err = os.ReadFile(certPath + "/ca-cert.pem")
		if err != nil {
			return nil, err
		}
		var cert tls.Certificate
		cert, err = tls.LoadX509KeyPair(certPath+"/api-user.pem", certPath+"/api-user.pem")
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(caCert)
		if err != nil {
			return nil, err
		}
		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:            pool,
					Certificates:       []tls.Certificate{cert},
					InsecureSkipVerify: true, // Skip hostname verification
				},
			},
		}
	default:
		zlog.Debug().Msgf("Provided Auth type is not valid, ")
		os.Exit(1)
	}
	req.Header.Add("Content-Type", "text/plain")
	resp, err := httpClient.Do(req)
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

func ProdWorkflowCreation(deviceInfo utils.DeviceInfo, imgtype string, artifactinfo utils.ArtifactData) error {
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

	hw := tinkerbell.NewHardware("machine-"+id, ns, deviceInfo.HwMacID,
		deviceInfo.DiskType, deviceInfo.HwIP, deviceInfo.Gateway)

	if kubeCreateErr := kubeClient.Create(ctx, hw); kubeCreateErr != nil {
		return kubeCreateErr
	}

	fmt.Printf("hardware workflow applied hardwarename:%s", hw.Name)
	fmt.Printf("hardware workflow Image URL :%s", deviceInfo.LoadBalancerIP)
	switch imgtype {
	case utils.ProdBkc:
		tmplName = fmt.Sprintf("bkc-%s-prod", id)
		deviceInfo.ClientImgName = "jammy-server-cloudimg-amd64.raw.gz"
		deviceInfo.ImType = "bkc"
		deviceInfo.Rootfspart = CalculateRootFS(deviceInfo.ImType, deviceInfo.DiskType)
		deviceInfo.LoadBalancerIP = artifactinfo.BkcURL
		deviceInfo.RootfspartNo = artifactinfo.BkcBasePkgURL
		tmplData, err = tinkerbell.NewTemplateDataProdBKC(tmplName, deviceInfo.Rootfspart, deviceInfo.RootfspartNo,
			deviceInfo.LoadBalancerIP, deviceInfo.HwIP, deviceInfo.Gateway, deviceInfo.ClientImgName, deviceInfo.ProvisionerIP, deviceInfo.SecurityFeature)
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
	tmplData, err := tinkerbell.NewTemplateData(tmplName, deviceInfo.ProvisionerIP, "CLIENT-SDK-TPM",
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

	if err = wait.PollUntilContextTimeout(ctx, pollTimeDuration, time.Hour, false, func(_ context.Context) (bool, error) {
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
	guid, err := VoucherScript(deviceInfo.ProvisionerIP, deviceInfo.HwSerialID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("GUID: %s\n", guid)
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
