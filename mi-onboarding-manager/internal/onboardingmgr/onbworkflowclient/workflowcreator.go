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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/utils"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/pkg/tinkerbell"
	tinkv1alpha1 "github.com/tinkerbell/tink/api/v1alpha1"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ConvertToJSONSerializable(input interface{}) interface{} {
	switch v := input.(type) {
	case map[interface{}]interface{}:
		m := make(map[string]interface{})
		for key, val := range v {
			m[fmt.Sprintf("%v", key)] = ConvertToJSONSerializable(val)
		}
		return m
	case []interface{}:
		for i, val := range v {
			v[i] = ConvertToJSONSerializable(val)
		}
		return v
	default:
		return input
	}
}

func GenerateMacIdString(macId string) string {
	macWithoutColon := strings.ReplaceAll(macId, ":", "")
	return strings.ToLower(macWithoutColon)
}

func GenerateDevSerial(macID string) (string, error) {
	// Remove colons from the MAC address
	uniqueID := strings.ReplaceAll(macID, ":", "")

	// Generate a random alphanumeric string of length 5
	rand.Seed(time.Now().UnixNano())
	randID := GenerateRandomString(5)

	// Truncate the uniqueID to remove the first 6 characters
	truncatedID := uniqueID[6:]

	// Concatenate truncatedID and randID to create devSerial
	devSerial := truncatedID + randID

	return devSerial, nil
}

func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func updateContentWithValues(content string, deviceInfo utils.DeviceInfo) string {
	content = strings.ReplaceAll(content, "$TINKERBELL_CLIENT_MAC", deviceInfo.HwMacID)
	content = strings.ReplaceAll(content, "$TINKERBELL_CLIENT_UID", GenerateMacIdString(deviceInfo.HwMacID))
	content = strings.ReplaceAll(content, "$ROOTFS_PART_NO", deviceInfo.RootfspartNo)
	content = strings.ReplaceAll(content, "$ROOTFS_PARTITION", deviceInfo.Rootfspart)
	content = strings.ReplaceAll(content, "$TINKERBELL_HOST_IP", deviceInfo.LoadBalancerIP)
	content = strings.ReplaceAll(content, "$TINKERBELL_CLIENT_IMG", deviceInfo.ClientImgName)
	content = strings.ReplaceAll(content, "$TINKERBELL_DEV_SERIAL", deviceInfo.HwSerialID)
	content = strings.ReplaceAll(content, "$TINKERBELL_CLIENT_GW", deviceInfo.Gateway)
	content = strings.ReplaceAll(content, "$TINKERBELL_CLIENT_IP", deviceInfo.HwIP)
	content = strings.ReplaceAll(content, "$DISK_DEVICE", deviceInfo.DiskType)
	content = strings.ReplaceAll(content, "$TINKERBELL_IMG_TYPE", deviceInfo.ImType)

	content = strings.ReplaceAll(content, "$PROVISIONER_HOST_IP", deviceInfo.ProvisionerIp)
	content = strings.ReplaceAll(content, "$FDO_CLIENT_TYPE", "CLIENT-SDK-TPM")
	print("deviceinfo", deviceInfo.ImType)
	content = strings.ReplaceAll(content, "$OS_TEMPLATE_NAME", deviceInfo.ImType)
	return content
}

func generateStringDataFromYAML(filePath string) (string, error) {
	hardwareFile, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("Error opening %s: %v", filePath, err)
	}
	defer hardwareFile.Close()

	hardwareYAML, err := io.ReadAll(hardwareFile)
	if err != nil {
		return "", fmt.Errorf("Error reading %s: %v", filePath, err)
	}

	content := string(hardwareYAML)

	return content, nil
}

func unmarshalYAMLContent(content string) (map[string]interface{}, error) {
	var hardwareData map[interface{}]interface{}
	err := yaml.Unmarshal([]byte(content), &hardwareData)
	if err != nil {
		return nil, fmt.Errorf("Error parsing YAML: %v", err)
	}

	serializableHardwareData := ConvertToJSONSerializable(hardwareData)

	objectData, ok := serializableHardwareData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Error converting YAML to map[string]interface{}")
	}

	return objectData, nil
}

func generateUnstructuredFromYAML(filePath string, deviceInfo utils.DeviceInfo) (*unstructured.Unstructured, error) {
	u := &unstructured.Unstructured{}

	content, err := generateStringDataFromYAML(filePath)
	if err != nil {
		return nil, err
	}

	content = updateContentWithValues(content, deviceInfo)

	objectData, err := unmarshalYAMLContent(content)
	if err != nil {
		return nil, err
	}

	log.Printf("object data-----------------------  %s", objectData)
	u.Object = objectData

	return u, nil
}

func createDynamicClient() (dynamic.Interface, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("Failed to load kubeconfig: %v", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Failed to create dynamic client: %v", err)
	}

	return dynamicClient, nil
}

func createCustomResource(dynamicClient dynamic.Interface, group, version, resource, namespace string, u *unstructured.Unstructured) error {
	_, err := dynamicClient.Resource(schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}).Namespace(namespace).Create(context.TODO(), u, metav1.CreateOptions{})

	return err
}

// working
func createCustomResourcename(dynamicClient dynamic.Interface, u *unstructured.Unstructured) error {

	kind := strings.ToLower(u.GetKind())
	resource := kind
	if kind != "hardware" {
		resource += "s"
	}

	groupVersionResource := schema.GroupVersionResource{
		Group:    u.GroupVersionKind().Group,
		Version:  u.GroupVersionKind().Version,
		Resource: resource,
	}

	namespace := u.GetNamespace()
	log.Printf("- %s", namespace)
	log.Printf("group  %s", u.GroupVersionKind().Group)
	log.Printf(" version %s", u.GroupVersionKind().Version)
	log.Printf("- %s", resource)
	_, err := dynamicClient.Resource(groupVersionResource).Namespace(namespace).Create(context.TODO(), u, metav1.CreateOptions{})

	return err
}

func ListPodsInNamespace(kubeconfigPath, namespace string) error {
	// Load the kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return fmt.Errorf("Failed to load kubeconfig: %v", err)
	}

	// Create a Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("Failed to create Kubernetes clientset: %v", err)
	}

	// List all pods in the specified namespace
	podList, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("Failed to list pods: %v", err)
	}

	// Print the list of pods
	log.Printf("Pods in %s namespace:", namespace)
	for _, pod := range podList.Items {
		log.Printf("- %s", pod.Name)
	}

	return nil
}

// Function to check the status of a Kubernetes Job
func checkJobStatus(namespace, jobName, HwId string) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("Failed to load kubeconfig: %v", err)
	}

	// Create a Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("Failed to create Kubernetes clientset: %v", err)
	}

	// Define a function to check the Job status
	checkStatus := func() (bool, error) {
		job, err := clientset.BatchV1().Jobs(namespace).Get(context.TODO(), jobName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		if job.Status.Succeeded > 0 {
			// The Job has completed successfully
			return true, nil
		} else if job.Status.Failed > 0 {
			// The Job has failed
			return true, fmt.Errorf("Job %s failed for SUT IP %s", jobName, HwId)
		}

		// The Job is still running
		return false, nil
	}

	// Poll the Job status every 10 seconds
	err = wait.PollImmediate(10*time.Second, time.Hour, func() (bool, error) {
		completed, err := checkStatus()
		if err != nil {
			log.Printf("Error checking Job status: %v", err)
			return false, nil
		}
		if completed {
			log.Printf("Job %s has completed for SUT IP %s", jobName, HwId)
			return true, nil
		}
		log.Printf("Job %s is still running for SUT IP %s", jobName, HwId)
		return false, nil
	})

	if err != nil {
		return fmt.Errorf("Job %s did not complete successfully: %v", jobName, err)
	}

	return nil
}

func newK8SClient() (client.Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	if err := tinkv1alpha1.AddToScheme(scheme.Scheme); err != nil {
		return nil, err
	}

	client, err := client.New(config, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return nil, err
	}
	return client, nil
}

func CreateTemplateWorkflow(deviceInfo utils.DeviceInfo, workflowName string) (string, error) {
	// Perform your logic here based on the req parameter

	// commenting the logic to list all ...........
	// err := ListPodsInNamespace(kubeconfigPath, namespace)

	// if err != nil {
	// 	// Handle the error, for example, log it or return an error response
	// 	log.Printf("Error listing pods: %v", err)
	// }

	dynamicClient, err := createDynamicClient()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return "", err
	}
	hardwareFilePath := filepath.Join("..", "workflows", workflowName)
	log.Printf("hardwareFilePath- %s", hardwareFilePath)
	u, err := generateUnstructuredFromYAML(hardwareFilePath, deviceInfo)
	workflowname := u.GetName()
	log.Printf("worfklow name %s for deletion", workflowname)
	if err != nil {
		return "", fmt.Errorf("Error generating unstructured from YAML: %v", err)
	}

	err = createCustomResourcename(dynamicClient, u)
	if err != nil {
		return workflowname, fmt.Errorf("Failed to create custom workflow resource: %v", err)
	}

	// If everything is successful, return nil (no error)
	return workflowname, nil
}

// //////////////Image download logic/////////////////////////////////////
func ImageDownload(artifactinfo utils.ArtifactData, deviceInfo utils.DeviceInfo, BkcImgDdLock, JammyImgDdLock, FocalImgDdLock, FocalMsImgDdLock sync.Locker) error {
	switch deviceInfo.ImType {
	case "prod_bkc":
		BkcImgDdLock.Lock()
		defer BkcImgDdLock.Unlock()
		if artifactinfo.BkcUrl == "" || artifactinfo.BkcBasePkgUrl == "" {
			return errors.New("required image download Bkc url or Bkc basee pkg url are missing from ArtifactData")
		}
		log.Println("Bkc image Download process is started for ", deviceInfo.HwIP)
		imgurl := artifactinfo.BkcUrl
		filenameBz2 := filepath.Base(imgurl)
		filenameWithoutExt := strings.TrimSuffix(filenameBz2, ".bz2")
		bkcRawGz := filenameWithoutExt + ".raw.gz"

		// Check if the file exists at the specified path
		filePath := "/opt/hook/" + bkcRawGz
		toDownload := !fileExists(filePath)
		fileName := "ubuntu-download_bkc.yaml"
		if toDownload {
			// TODO: Need to Remove hardcoding path
			err := ReadingYamlNCreatingResourse(imgurl, deviceInfo.ImType, "../../onboardingmgr/workflows/manifests/image_dload", fileName, deviceInfo.HwIP)
			if err != nil {
				return err
			}
			fmt.Println("Prod_bkc Image file is downloaded")
		} else {
			fmt.Printf("using old downloaded bkc %s\n", bkcRawGz)
		}
		pkgFileName := "ubuntu-download-pkg-agents_bkc.yaml"
		// TODO: Need to Remove hardcoding path
		err := ReadingYamlNCreatingResourse(artifactinfo.BkcBasePkgUrl, "prod_bkc-pkg", "../../onboardingmgr/workflows/manifests/image_dload", pkgFileName, deviceInfo.HwIP)
		if err != nil {
			return err
		}
	case "prod_jammy":
		JammyImgDdLock.Lock()
		defer JammyImgDdLock.Unlock()
		log.Println("Jammy image Download process is started for", deviceInfo.HwIP)
		fileName := "ubuntu-download_jammy.yaml"
		toDownload := !fileExists("/opt/hook/jammy-server-cloudimg-amd64.raw.gz")
		if toDownload {
			// TODO: Need to Remove hardcoding path
			err := ReadingYamlNCreatingResourse("", deviceInfo.ImType, "../../onboardingmgr/workflows/manifests/image_dload", fileName, deviceInfo.HwIP)
			if err != nil {
				return err
			}
			fmt.Println("Prod_jammy Image file is downloaded")
		} else {
			fmt.Printf("using old downloaded jammy \n")
		}

	case "prod_focal":
		FocalImgDdLock.Lock()
		defer FocalImgDdLock.Unlock()
		fileName := "ubuntu-download.yaml"
		toDownload := !fileExists("/opt/hook/focal-server-cloudimg-amd64.raw.gz")
		log.Println("Focal image Download process is started for", deviceInfo.HwIP)
		if toDownload {
			log.Println("Focal image Download process is started")
			// TODO: Need to Remove hardcoding path
			err := ReadingYamlNCreatingResourse("", deviceInfo.ImType, "../../onboardingmgr/workflows/manifests/image_dload", fileName, deviceInfo.HwIP)
			if err != nil {
				return err
			}
			fmt.Println("Prod_focal Image file is downloaded")
		} else {
			fmt.Printf("using old downloaded focal \n")
		}

	case "prod_focal-ms":
		FocalMsImgDdLock.Lock()
		defer FocalMsImgDdLock.Unlock()
		log.Println("Prod Focal MS image Download process is started for", deviceInfo.HwIP)
		fileName := "ubuntu-download_focal-ms.yaml"
		if !fileExists("/opt/hook/linux-image-5.15.96-lts.deb") || !fileExists("/opt/hook/linux-headers-5.15.96-lts.deb") || !fileExists("/opt/hook/focal-server-cloudimg-amd64.raw.gz") || !fileExists("/opt/hook/azure-credentials.env_"+deviceInfo.HwMacID) || !fileExists("/opt/hook/azure_dps_installer.sh") || !fileExists("/opt/hook/log.sh") {
			// TODO: Need to Remove hardcoding path
			err := ReadingYamlNCreatingResourse("", deviceInfo.ImType, "../../onboardingmgr/workflows/manifests/image_dload", fileName, deviceInfo.HwIP)
			if err != nil {
				return err
			}
			fmt.Println("Prod_focal-ms Image file is downloaded")
		} else {
			fmt.Printf("using old downloaded prod focal ms \n")
		}

	default:
		return errors.New("Unknown image_type:" + deviceInfo.ImType)
	}
	return nil
}

func ReadingYamlNCreatingResourse(imgurl, imgType, filePath, fileName, HwId string) error {
	dynamicClient, err := createDynamicClient()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return err
	}
	currentPath, _ := os.Getwd()
	ubuntubkcpath := filepath.Join(currentPath, filePath, fileName)

	u := &unstructured.Unstructured{}
	yamlContent, err := generateStringDataFromYAML(ubuntubkcpath)
	if err != nil {
		return fmt.Errorf("failed to generate YAML content: %v", err)
	}
	contentInfo := strings.Split(string(yamlContent), "---")
	for _, content := range contentInfo {
		if imgType == "prod_bkc" {
			content = strings.ReplaceAll(content, "BKC_IMG_LINK", imgurl)
		} else if imgType == "prod_bkc-pkg" {
			currentPath, _ = os.Getwd()
			repoPath, _ := strings.CutSuffix(currentPath, "/internal/onboardingmgr/onboarding")
			content = strings.ReplaceAll(content, "CurrentRepoPath", repoPath)
			content = strings.ReplaceAll(content, "BKC_BASEPKG_URL", imgurl)
			content = strings.ReplaceAll(content, "HOST_IP", os.Getenv("MGR_HOST"))
		} else if imgType == "prod_focal-ms" {
			currentPath, _ = os.Getwd()
			azureEnvPath, _ := strings.CutSuffix(currentPath, "/internal/onboardingmgr/onboarding")
			content = strings.ReplaceAll(content, "azure_env_path", azureEnvPath)
		}

		objectData, err := unmarshalYAMLContent(content)
		if err != nil {
			return fmt.Errorf("failed to unmarshal YAML content: %v", err)
		}
		u.Object = objectData

		//Deleting the resourse if exist
		err = DeleteCustomResource(u)
		if err != nil {
			log.Printf("warning Error msg while deleting resourse: %s is %v", strings.ToLower(u.GetKind()), err)
		}
		time.Sleep(5 * time.Second)

		//Creating the resourse
		err = createCustomResourcename(dynamicClient, u)
		if err != nil {
			return fmt.Errorf("failed to create custom workflow resource: %v", err)
		}

		if strings.ToLower(u.GetKind()) == "job" {
			log.Println("Checking the Job status of ", u.GetName())
			err = checkJobStatus("tink-system", u.GetName(), HwId)
			if err != nil {
				log.Fatalf("Error while waiting for workflow success: %v", err)
				return err
			}
		}
	}
	return nil
}

func DeleteCustomResource(u *unstructured.Unstructured) error {
	// Create a Kubernetes client configuration from the provided kubeconfig path
	dynamicClient, err := createDynamicClient()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return err
	}
	// Delete the custom resource
	kind := strings.ToLower(u.GetKind())
	resource := kind
	if kind != "hardware" {
		resource += "s"
	}

	groupVersionResource := schema.GroupVersionResource{
		Group:    u.GroupVersionKind().Group,
		Version:  u.GroupVersionKind().Version,
		Resource: resource,
	}
	// Set the propagationPolicy to Background when deleting the Job.
	deletePolicy := metav1.DeletePropagationBackground
	err = dynamicClient.Resource(groupVersionResource).Namespace(u.GetNamespace()).Delete(context.TODO(), u.GetName(), metav1.DeleteOptions{PropagationPolicy: &deletePolicy})

	if err != nil {
		return fmt.Errorf("warning deleting workflow: %v", err)
	}

	fmt.Printf("Workflow %s deleted successfully in namespace %s\n", u.GetName(), u.GetNamespace())
	return nil
}

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func waitForWorkflowSuccess(namespace, workflowName string) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("Failed to load kubeconfig: %v", err)
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("Failed to create dynamic client: %v", err)
	}

	checkStatus := func() (bool, error) {
		// Define the resource request for your custom resource
		resource := "workflows"

		gvr := schema.GroupVersionResource{
			Group:    "tinkerbell.org",
			Version:  "v1alpha1",
			Resource: resource,
		}

		// Get the custom resource
		cr, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), workflowName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		// Extract the status field from the custom resource
		statusField, found, _ := unstructured.NestedString(cr.Object, "status", "state")
		if !found {
			return false, fmt.Errorf("Status field not found in the custom resource")
		}

		log.Printf("Workflow %s state: %s", workflowName, statusField)

		// Check if the workflow has reached STATE_SUCCESS
		if strings.ToUpper(statusField) == "STATE_SUCCESS" {
			return true, nil
		}

		return false, nil
	}

	// Poll the workflow status every 20 seconds
	err = wait.PollImmediate(20*time.Second, time.Hour, func() (bool, error) {
		success, err := checkStatus()
		if err != nil {
			log.Printf("Error checking workflow status: %v", err)
			return false, nil
		}
		if success {
			log.Printf("Workflow %s has reached STATE_SUCCESS", workflowName)
			return true, nil
		}
		return false, nil
	})

	if err != nil {
		return fmt.Errorf("Workflow did not reach STATE_SUCCESS: %v", err)
	}

	return nil
}

func GetAllVariablesFromFile(fileName string) (map[string]string, error) {
	// Read the content of the file
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	// Split the content into lines
	lines := strings.Split(string(content), "\n")

	// Create a map to store the variables
	variables := make(map[string]string)

	// Parse the lines and store the variables in the map
	for _, line := range lines {
		parts := strings.Split(line, "=")
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			variables[key] = value
		}
	}

	return variables, nil
}

func convertMACAddress(mac string) string {
	// Remove colons from the MAC address
	return strings.ReplaceAll(mac, ":", "")
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

func OnboardSetupms(ImType string) error {

	oldWorkingDir, err := os.Getwd()
	if err != nil {
		return err
	}
	log.Printf("old working dir---  %s ", oldWorkingDir)
	fmt.Printf(" ----------------imtype:%s", ImType)
	onboardingstartupdir := filepath.Join("..", "scripts")
	if err := os.Chdir(onboardingstartupdir); err != nil {
		return err
	}
	log.Printf("Job %s has completed", onboardingstartupdir)
	// onboardingfilepath := filepath.Join(onboardingstartupdir, "onboardingstartupms.sh")
	// Make the script executable
	cmdChmod := exec.Command("chmod", "+x", "onboardingstartupms.sh")
	if err := cmdChmod.Run(); err != nil {
		return err
	}

	// Run the shell script with arguments
	var cmdExtendUpload *exec.Cmd

	if ImType == "bkc" {
		cmdExtendUpload = exec.Command("./onboardingstartupms.sh", ImType, "ms")
	} else {
		// Todo: check based on dpsscopeid for ms
		cmdExtendUpload = exec.Command("./onboardingstartupms.sh", ImType, "ms")
	}

	output, err := cmdExtendUpload.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Error executing script: %v, Output: %s", err, output)
	}

	variableMap, err := GetAllVariablesFromFile("env_variable.txt")
	if err != nil {
		fmt.Println("Error:", err)
		return err
	}
	if tinkImg, ok := variableMap["TINKER_CLIENT_IMG"]; ok {
		fmt.Printf("TINKER_CLIENT_IMG: %s\n", tinkImg)
	} else {
		fmt.Println("TINKER_CLIENT_IMG not found in the file")
	}
	if err := os.Chdir(oldWorkingDir); err != nil {
		return err
	}

	return nil
}

func readUIDFromFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
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
	if err := os.Chdir(scriptDir); err != nil {
		return "", err
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
	if err := cmdChmod.Run(); err != nil {
		return "", err
	}
	fmt.Printf("host ip: %s\n", hostIP)

	// Run the shell script with arguments
	cmdExtendUpload := exec.Command("./extend_upload.sh", "-m", "sh", "-c", "./secrets/", "-e", "mtls", "-m", hostIP, "-o", hostIP, "-s", deviceSerial)

	output, err := cmdExtendUpload.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Error executing script: %v, Output: %s", err, output)
	}

	fmt.Printf("Script Output: voucher done\n%s\n", output)
	// Create the GUID file path
	guidFilePath := filepath.Join(deviceSerial + "_guid.txt")

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

func VoucherScript(hostIp, deviceSerial string) (string, error) {
	var (
		attestationType string
		mfgIp           string
		onrIp           string
		apiUser         string
		mfgApiPasswd    string
		onrApiPasswd    string
		mfgPort         string
		onrPort         string
		authType        string
		serialNo        string
		certPath        string
	)
	attestationType = "SECP256R1"
	authType = "mtls"
	mfgIp = hostIp
	onrIp = hostIp
	serialNo = deviceSerial
	//default values
	defaultAttestationType := "SECP256R1"
	defaultMfgIp := "localhost"
	defaultOnrIp := "localhost"
	defaultApiUser := "apiUser"
	defaultMfgApiPasswd := ""
	defaultOnrApiPasswd := ""
	mfgPort = "8038"
	onrPort = "8043"
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
	if authType == "" {
		log.Println("Auth method is mandatory, ")
		os.Exit(0)
	}
	if serialNo == "" {
		log.Println("Serial number of device is mandatory, ")
		os.Exit(0)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(err)
	}
	// TODO: modify the path for certificates
	certPath = homeDir + "/.fdo-secrets/scripts/secrets"
	url := "https://" + onrIp + ":" + onrPort + "/api/v1/certificate?alias=" + attestationType
	resp, err := apiCalls("GET", url, authType, apiUser, onrApiPasswd, certPath, []byte{})
	if err != nil {
		return "", fmt.Errorf("Error1 Details", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		file, err := os.Create(fmt.Sprintf("/home/%s/.fdo-secrets/scripts/owner_cert_%s.txt", os.Getenv("USER"), attestationType))
		if err != nil {
			return "", fmt.Errorf("Error creating the file:", err)
		}
		defer file.Close()
		_, err = io.Copy(file, resp.Body)
		if err != nil {
			return "", fmt.Errorf("Error writing the response to the file:", err)
		}
		fmt.Printf("Success in downloading %s owner certificate to owner_cert_%s.txt\n", attestationType, attestationType)
		ownerCertificate, err := os.ReadFile(fmt.Sprintf("/home/%s/.fdo-secrets/scripts/owner_cert_%s.txt", os.Getenv("USER"), attestationType))
		if err != nil {
			return "", fmt.Errorf("Error reading the file:", err)
		}
		url = "https://" + mfgIp + ":" + mfgPort + "/api/v1/mfg/vouchers/" + serialNo
		fmt.Println(url)
		resp, err := apiCalls("POST", url, authType, apiUser, mfgApiPasswd, certPath, ownerCertificate)
		if err != nil {
			fmt.Println(err)
			return "", fmt.Errorf("Error Details ", err)
		}
		if resp.StatusCode == 200 {
			file1, err := os.Create(fmt.Sprintf("/home/%s/.fdo-secrets/scripts/%s_voucher.txt", os.Getenv("USER"), serialNo))
			if err != nil {
				return "", fmt.Errorf("Error creating the file:", err)
			}
			defer file1.Close()
			_, err = io.Copy(file1, resp.Body)
			if err != nil {
				return "", fmt.Errorf("Error writing the response to the file:", err)
			}
			fmt.Printf("Success in downloading extended voucher for device with serial number:%s\n", serialNo)
			extendVoucher, err := os.ReadFile(fmt.Sprintf("/home/%s/.fdo-secrets/scripts/%s_voucher.txt", os.Getenv("USER"), serialNo))
			if err != nil {
				return "", fmt.Errorf("Error reading the file:", err)
			}
			url = "https://" + onrIp + ":" + onrPort + "/api/v1/owner/vouchers/"
			resp, err = apiCalls("POST", url, authType, apiUser, onrApiPasswd, certPath, extendVoucher)
			if err != nil {
				return "", fmt.Errorf("error details :", err)
			}
			if resp.StatusCode == 200 {
				file2, err := os.Create(fmt.Sprintf("/home/%s/.fdo-secrets/scripts/%s_guid.txt", os.Getenv("USER"), serialNo))
				if err != nil {
					fmt.Println("Error creating the file:", err)
					return "", fmt.Errorf("Error creating the file:", err)
				}
				_, err = io.Copy(file2, resp.Body)
				if err != nil {
					return "", fmt.Errorf("Error writing the response to the file:", err)
				}
				deviceGuid, err := os.ReadFile(fmt.Sprintf("/home/%s/.fdo-secrets/scripts/%s_guid.txt", os.Getenv("USER"), serialNo))
				if err != nil {
					return "", fmt.Errorf("Error reading the file:", err)
				}
				url := fmt.Sprintf("https://%s:%s/api/v1/to0/%s", onrIp, onrPort, deviceGuid)
				resp, err := apiCalls("GET", url, authType, apiUser, onrApiPasswd, certPath, deviceGuid)
				if err != nil {
					return "", fmt.Errorf("Error Details:", err)
				}
				if resp.StatusCode == 200 {
					fmt.Printf("Success in triggering TO0 for %s with GUID %s\n", serialNo, deviceGuid)
					return string(deviceGuid), nil
				} else {
					return "", fmt.Errorf("Failure in triggering TO0 for %s  with GUID %s ", serialNo, deviceGuid)
				}
			} else {
				return "", fmt.Errorf("Failure in uploading voucher to owner for device with serial number %s with response code: %d", serialNo, resp.StatusCode)
			}
		} else {
			return "", fmt.Errorf("Failure in getting extended voucher for device with serial number %s with response code: %d", serialNo, resp.StatusCode)
		}
	} else {
		return "", fmt.Errorf("Failure in getting owner certificate for type %s with response code: %d\n", attestationType, resp.StatusCode)
	}
}

func apiCalls(httpMethod, url, authType, apiUser, onrApiPasswd, certPath string, bodyData []byte) (*http.Response, error) {
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
		log.Println("Digest authentication mode is being used")
		req.SetBasicAuth(apiUser, onrApiPasswd)
		client = &http.Client{}
	} else if strings.ToLower(authType) == "mtls" {
		log.Println("Client Certificate authentication mode is being used")
		caCert, err := os.ReadFile(certPath + "/ca-cert.pem")
		if err != nil {
			return nil, err
		}
		cert, err := tls.LoadX509KeyPair(certPath+"/api-user.pem", certPath+"/api-user.pem")
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(caCert)
		if err != nil {
			return nil, err
		}
		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:            pool,
					Certificates:       []tls.Certificate{cert},
					InsecureSkipVerify: true, // Skip hostname verification
				},
			},
		}
	} else {
		log.Println("Provided Auth type is not valid, ")
		os.Exit(1)
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

func CalculateRootFS(imageType, diskDev string) (string, string) {
	ROOTFS_PART_NO := "1"

	if imageType == "bkc" {
		ROOTFS_PART_NO = "3"
	}

	// Use regular expression to check if diskDev ends with a numeric digit
	match, _ := regexp.MatchString(".*[0-9]$", diskDev)

	if match {
		return fmt.Sprintf("p%s", ROOTFS_PART_NO), ROOTFS_PART_NO
	}

	return ROOTFS_PART_NO, ROOTFS_PART_NO
}

func DeleteWorkflow(namespace, workflowName, resource string) error {
	// Create a Kubernetes client configuration from the provided kubeconfig path
	dynamicClient, err := createDynamicClient()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return err
	}

	// Define the resource type and group for your custom resource
	// resource := "workflows.tinkerbell.org"
	// resourceGroup := "tinkerbell.org"

	// Delete the custom resource
	groupVersionResource := schema.GroupVersionResource{
		Group:    "tinkerbell.org",
		Version:  "v1alpha1",
		Resource: resource,
	}
	err = dynamicClient.Resource(groupVersionResource).Namespace(namespace).Delete(context.TODO(), workflowName, metav1.DeleteOptions{})

	if err != nil {
		return fmt.Errorf("Error deleting workflow: %v", err)
	}

	fmt.Printf("Workflow %s deleted successfully in namespace %s\n", workflowName, namespace)
	return nil
}

func ToWorkflowCreation(deviceInfo utils.DeviceInfo) error {
	//TODO: Remove hardcoding of the filename decide based on input
	totemplatename, tempale_err := CreateTemplateWorkflow(deviceInfo, "/manifests/to/template_to.yaml")
	if tempale_err != nil {
		// Handle the error, for example, log it or return an error response
		return tempale_err
	}
	fmt.Printf("template workflow applied workflowname:%s", totemplatename)
	// fmt.Println("Tempalte Workflow applied")
	toworkflowname, err4 := CreateTemplateWorkflow(deviceInfo, "/manifests/to/workflow.yaml")
	if err4 != nil {
		// Handle the error, for example, log it or return an error response
		return err4
	}
	fmt.Printf("workflow applied workflowname:%s--------------", toworkflowname)

	err5 := waitForWorkflowSuccess("tink-system", toworkflowname)
	if err5 != nil {
		log.Fatalf("Error waiting for workflow success: %v", err5)
		return err5
	}
	fmt.Println("workflow-1c697a0eb228 has reached STATE_SUCCESS")

	////////////////////////////////To workflow Cleanup//////////////////////////
	//TODO:replace the namespace other info from Groupinfo struct
	template_to_delete_err := DeleteWorkflow("tink-system", totemplatename, "templates")
	if template_to_delete_err != nil {
		fmt.Printf("Error: %v\n", template_to_delete_err)
	}
	workflow_to_delete_err := DeleteWorkflow("tink-system", toworkflowname, "workflows")
	if workflow_to_delete_err != nil {
		fmt.Printf("Error: %v\n", workflow_to_delete_err)
	}
	return nil
}

func ProdWorkflowCreation(deviceInfo utils.DeviceInfo, imgtype string) error {
	client, err := newK8SClient()
	if err != nil {
		return err
	}

	var (
		ctx      = context.Background()
		ns       = "tink-system"
		id       = GenerateMacIdString(deviceInfo.HwMacID)
		tmplName string
		tmplData []byte
	)

	if imgtype == "prod_bkc" {
		tmplName = fmt.Sprintf("bkc-%s-prod", id)
		deviceInfo.ImType = "bkc"
		deviceInfo.Rootfspart, deviceInfo.RootfspartNo = CalculateRootFS(deviceInfo.ImType, deviceInfo.DiskType)
		tmplData, err = tinkerbell.NewTemplateDataProdBKC(tmplName, deviceInfo.Rootfspart,
			deviceInfo.LoadBalancerIP, deviceInfo.ClientImgName)
		if err != nil {
			return err
		}
	} else if imgtype == "prod_focal" {
		tmplName = fmt.Sprintf("focal-%s-prod", id)
		deviceInfo.ClientImgName = "focal-server-cloudimg-amd64.raw.gz"
		deviceInfo.ImType = "focal"
		deviceInfo.Rootfspart, deviceInfo.RootfspartNo = CalculateRootFS(deviceInfo.ImType, deviceInfo.DiskType)
		tmplData, err = tinkerbell.NewTemplateDataProd(tmplName, deviceInfo.Rootfspart,
			deviceInfo.RootfspartNo, deviceInfo.LoadBalancerIP)
		if err != nil {
			return err
		}
	} else if imgtype == "prod_focal-ms" {
		tmplName = fmt.Sprintf("focal-ms-%s-prod", id)
		deviceInfo.ImType = "focal-ms"
		deviceInfo.Rootfspart, deviceInfo.RootfspartNo = CalculateRootFS(deviceInfo.ImType, deviceInfo.DiskType)
		tmplData, err = tinkerbell.NewTemplateDataProdMS(tmplName, deviceInfo.Rootfspart, deviceInfo.RootfspartNo,
			deviceInfo.LoadBalancerIP, deviceInfo.HwIP, deviceInfo.Gateway, deviceInfo.HwMacID)
		if err != nil {
			return err
		}
	} else {
		tmplName = fmt.Sprintf("focal-%s-prod", id)
		deviceInfo.ClientImgName = "jammy-server-cloudimg-amd64.raw.gz"
		deviceInfo.ImType = "jammy"
		deviceInfo.Rootfspart, deviceInfo.RootfspartNo = CalculateRootFS(deviceInfo.ImType, deviceInfo.DiskType)
		tmplData, err = tinkerbell.NewTemplateDataProd(tmplName, deviceInfo.Rootfspart,
			deviceInfo.RootfspartNo, deviceInfo.LoadBalancerIP)
		if err != nil {
			return err
		}
	}

	fmt.Println("production workflow started.......................................")
	// have notification from sut
	log.Printf("ROOTFS_PART_NO %s /// ROOTFS_PARTITION %s", deviceInfo.RootfspartNo, deviceInfo.Rootfspart)

	tmpl := tinkerbell.NewTemplate(string(tmplData), tmplName, ns)
	if err := client.Create(ctx, tmpl); err != nil {
		return err
	}
	fmt.Printf("template workflow applied workflowname:%s", tmpl.Name)

	wf := tinkerbell.NewWorkflow(fmt.Sprintf("workflow-%s-prod", id), ns, deviceInfo.HwMacID)
	wf.Spec.HardwareRef = "machine-" + id
	wf.Spec.TemplateRef = fmt.Sprintf("%s-%s-prod", deviceInfo.ImType, id)
	if err := client.Create(ctx, wf); err != nil {
		return err
	}
	fmt.Printf("workflow applied workflowname:%s", wf.Name)

	check := func() (bool, error) {
		err := client.Get(ctx, types.NamespacedName{Namespace: wf.Namespace, Name: wf.Name}, wf)
		if err != nil {
			return false, err
		}
		log.Printf("Workflow %s state: %s\n", wf.Name, wf.Status.State)
		return strings.ToUpper(string(wf.Status.State)) == "STATE_SUCCESS", nil
	}

	if err := wait.PollUntilContextTimeout(ctx, 20*time.Second, time.Hour, false, func(_ context.Context) (bool, error) {
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
		return fmt.Errorf("Workflow did not reach STATE_SUCCESS: %v", err)
	}

	////////////////////////////////To workflow Cleanup//////////////////////////
	if err := client.Delete(ctx, tmpl); err != nil {
		return err
	}

	if err := client.Delete(ctx, wf); err != nil {
		return err
	}

	return nil
}

func DiWorkflowCreation(deviceInfo utils.DeviceInfo) (string, error) {
	client, err := newK8SClient()
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	ns := "tink-system"
	id := GenerateMacIdString(deviceInfo.HwMacID)

	hw := tinkerbell.NewHardware("machine-"+id, ns, deviceInfo.HwMacID,
		deviceInfo.DiskType, deviceInfo.HwIP, deviceInfo.Gateway)
	if err := client.Create(ctx, hw); err != nil {
		return "", err
	}
	fmt.Printf("hardware workflow applied hardwarename:%s", hw.Name)

	tmplName := "fdodi-" + id
	tmplData, err := tinkerbell.NewTemplateData(tmplName, deviceInfo.ProvisionerIp, "CLIENT-SDK-TPM",
		deviceInfo.DiskType, deviceInfo.HwSerialID)
	if err != nil {
		return "", err
	}
	tmpl := tinkerbell.NewTemplate(string(tmplData), tmplName, ns)
	if err := client.Create(ctx, tmpl); err != nil {
		return "", err
	}
	fmt.Printf("template workflow applied workflowname:%s", tmpl.Name)

	wf := tinkerbell.NewWorkflow("workflow-"+id, ns, deviceInfo.HwMacID)
	wf.Spec.HardwareRef = hw.Name
	wf.Spec.TemplateRef = tmpl.Name
	if err := client.Create(ctx, wf); err != nil {
		return "", err
	}
	fmt.Printf("workflow applied workflowname:%s", wf.Name)

	check := func() (bool, error) {
		err := client.Get(ctx, types.NamespacedName{Namespace: wf.Namespace, Name: wf.Name}, wf)
		if err != nil {
			return false, err
		}
		log.Printf("Workflow %s state: %s\n", wf.Name, wf.Status.State)
		return strings.ToUpper(string(wf.Status.State)) == "STATE_SUCCESS", nil
	}

	if err := wait.PollUntilContextTimeout(ctx, 20*time.Second, time.Hour, false, func(_ context.Context) (bool, error) {
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
		return "", fmt.Errorf("Workflow did not reach STATE_SUCCESS: %v", err)
	}

	/////////////////////Voucher extension//////////////////////
	guid, err := VoucherScript(deviceInfo.ProvisionerIp, deviceInfo.HwSerialID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("GUID: %s\n", guid)
	}

	////////////////////////////////Di workflow Cleanup//////////////////////////
	if err := client.Delete(ctx, tmpl); err != nil {
		return "", err
	}

	if err := client.Delete(ctx, wf); err != nil {
		return "", err
	}

	return guid, nil
}
