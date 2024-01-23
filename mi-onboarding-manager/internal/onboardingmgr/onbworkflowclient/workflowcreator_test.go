/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onbworkflowclient

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/utils"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func generatekubeconfigPath() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}

	kubeconfigPath := filepath.Join(currentUser.HomeDir, ".kube/config")
	return kubeconfigPath, err
}

func TestFileExists(t *testing.T) {
	// Create a temporary file for testing
	tempFile := "temp_test_file.txt"
	defer os.Remove(tempFile)

	// Test when file exists
	if err := createTempFile(tempFile); err != nil {
		t.Fatalf("Error creating temp file: %v", err)
	}
	exists := fileExists(tempFile)
	if !exists {
		t.Errorf("Expected file to exist, but it does not.")
	}

	// Test when file does not exist
	os.Remove(tempFile) // Remove the file to simulate non-existence
	exists = fileExists(tempFile)
	if exists {
		t.Errorf("Expected file not to exist, but it does.")
	}
}

func createTempFile(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	return nil
}

func TestGenerateStringDataFromYAML(t *testing.T) {
	// Create a temporary YAML file for testing
	tempYamlFile := filepath.Join("..", "workflows", "/manifests/to/test_template.yaml")

	// Write YAML content to the temporary file
	yamlContent := "apiVersion: \"tinkerbell.org/v1alpha1\"\nkind: Template\nmetadata:\n  name: testing"
	err := ioutil.WriteFile(tempYamlFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Error creating temp file: %v", err)
	}
	defer os.RemoveAll(tempYamlFile)

	// Test reading valid YAML file
	content, err := generateStringDataFromYAML(tempYamlFile)
	if err != nil {
		t.Fatalf("Unexpected error reading YAML file: %v", err)
	}
	if content != yamlContent {
		t.Errorf("Expected content '%s', but got '%s'", yamlContent, content)
	}
	// Test reading non-existent file
	nonExistentFile := "non_existent_file.yaml"
	_, err = generateStringDataFromYAML(nonExistentFile)
	if err == nil {
		t.Error("Expected error for non-existent file, but got nil.")
	}
}
func TestUnmarshalYAMLContent(t *testing.T) {
	// Test valid YAML content
	validYAML := `
name: John Doe
age: 30
`
	result, err := unmarshalYAMLContent(validYAML)
	if err != nil {
		t.Fatalf("Unexpected error for valid YAML: %v", err)
	}

	expectedResult := map[string]interface{}{
		"name": "John Doe",
		"age":  30,
	}

	if !isEqual(result, expectedResult) {
		t.Errorf("Expected result to be %v, but got %v", expectedResult, result)
	}

	// Test invalid YAML content
	invalidYAML := `
- invalid YAML
`
	_, err = unmarshalYAMLContent(invalidYAML)
	if err == nil {
		t.Error("Expected error for invalid YAML, but got nil.")
	}
}

func isEqual(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for key, aValue := range a {
		if bValue, ok := b[key]; !ok || bValue != aValue {
			return false
		}
	}
	return true
}
func TestCaSlculateRootF(t *testing.T) {
	// Test case 1: imageType is "bkc" and diskDev ends with a numeric digit
	partition, number := CalculateRootFS("bkc", "sda1")
	assert.Equal(t, "p3", partition, "Expected partition 'p3'")
	assert.Equal(t, "3", number, "Expected number '3'")

	// Test case 2: imageType is "ms" and diskDev ends with a numeric digit
	partition, number = CalculateRootFS("ms", "nvme0n1p2")
	assert.Equal(t, "p1", partition, "Expected partition 'p1'")
	assert.Equal(t, "1", number, "Expected number '1'")

	// Test case 3: imageType is "bkc" and diskDev does not end with a numeric digit
	partition, number = CalculateRootFS("bkc", "sdb")
	assert.Equal(t, "3", partition, "Expected partition '3'")
	assert.Equal(t, "3", number, "Expected number '3'")

	// Test case 4: imageType is  "ms" and diskDev ends with a numeric digit
	partition, number = CalculateRootFS("other", "nvme0n1p3")
	assert.Equal(t, "p1", partition, "Expected partition 'p1'")
	assert.Equal(t, "1", number, "Expected number '1'")
}

// MockHTTPServer creates a mock HTTP server and returns its URL
func MockHTTPServer(handler http.Handler) (*httptest.Server, string) {
	server := httptest.NewServer(handler)
	return server, server.URL
}

func TestGenerateUnstructuredFromYAML(t *testing.T) {
	// Prepare a temporary YAML file for testing
	yamlContent := `
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec:
  containers:
  - name: test-container
    image: nginx
`
	tmpfile, err := ioutil.TempFile("", "test-yaml-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write([]byte(yamlContent)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Call the function with the temporary YAML file and deviceInfo
	deviceInfo := utils.DeviceInfo{ /* fill with appropriate data */ }
	result, err := generateUnstructuredFromYAML(tmpfile.Name(), deviceInfo)

	// Assertions
	assert.Nil(t, err, "Expected no error")
	assert.NotNil(t, result, "Expected non-nil result")

	// Additional assertions based on your specific requirements
	// For example, you might want to check if certain fields in the result are set as expected.
	assert.Equal(t, "Pod", result.GetKind())
	assert.Equal(t, "test-pod", result.GetName())
	// ...

	// Clean up the temporary file
	os.Remove(tmpfile.Name())
}

var (
	bkcImgDdLock     sync.Locker
	focalImgDdLock   sync.Locker
	jammyImgDdLock   sync.Locker
	focalMsImgDdLock sync.Locker
)

func TestImageDownload_empty(t *testing.T) {
	artifactInfo := utils.ArtifactData{
		BkcUrl:        "http://example.com/bkc_image.bz2",
		BkcBasePkgUrl: "http://example.com/bkc_base_pkg",
	}

	deviceInfo := utils.DeviceInfo{
		ImType: "",
	}

	err := ImageDownload(artifactInfo, deviceInfo, bkcImgDdLock, focalImgDdLock, jammyImgDdLock, focalMsImgDdLock)

	assert.Error(t, err)
	// Add additional assertions as needed
}

func TestImageDownload(t *testing.T) {
	type args struct {
		artifactinfo     utils.ArtifactData
		deviceInfo       utils.DeviceInfo
		kubeconfigPath   string
		BkcImgDdLock     sync.Locker
		JammyImgDdLock   sync.Locker
		FocalImgDdLock   sync.Locker
		FocalMsImgDdLock sync.Locker
	}
	bkcImgDdLocks := &sync.Mutex{}
	jammyImgDdLocks := &sync.Mutex{}
	focalImgDdLocks := &sync.Mutex{}
	focalMsImgDdLocks := &sync.Mutex{}
	inputArgs := args{
		artifactinfo: utils.ArtifactData{
			BkcUrl:        "1bkc",
			BkcBasePkgUrl: "Bkc",
		},
		deviceInfo: utils.DeviceInfo{
			ImType: "prod_bkc",
		},
		kubeconfigPath:   "configPath",
		BkcImgDdLock:     bkcImgDdLocks,
		JammyImgDdLock:   jammyImgDdLocks,
		FocalImgDdLock:   focalImgDdLocks,
		FocalMsImgDdLock: focalMsImgDdLocks,
	}
	inputArgs1 := args{
		artifactinfo: utils.ArtifactData{
			BkcUrl:        "1bkc",
			BkcBasePkgUrl: "Bkc",
		},
		deviceInfo: utils.DeviceInfo{
			ImType: "prod_jammy",
		},
		kubeconfigPath:   "configPath",
		BkcImgDdLock:     bkcImgDdLocks,
		JammyImgDdLock:   jammyImgDdLocks,
		FocalImgDdLock:   focalImgDdLocks,
		FocalMsImgDdLock: focalMsImgDdLocks,
	}
	inputArgs2 := args{
		artifactinfo: utils.ArtifactData{
			BkcUrl:        "1bkc",
			BkcBasePkgUrl: "Bkc",
		},
		deviceInfo: utils.DeviceInfo{
			ImType: "",
		},
		kubeconfigPath:   "configPath",
		BkcImgDdLock:     bkcImgDdLocks,
		JammyImgDdLock:   jammyImgDdLocks,
		FocalImgDdLock:   focalImgDdLocks,
		FocalMsImgDdLock: focalMsImgDdLocks,
	}
	inputArgs3 := args{
		artifactinfo: utils.ArtifactData{
			BkcUrl:        "1bkc",
			BkcBasePkgUrl: "Bkc",
		},
		deviceInfo: utils.DeviceInfo{
			ImType: "prod_focal-ms",
		},
		kubeconfigPath:   "configPath",
		BkcImgDdLock:     bkcImgDdLocks,
		JammyImgDdLock:   jammyImgDdLocks,
		FocalImgDdLock:   focalImgDdLocks,
		FocalMsImgDdLock: focalMsImgDdLocks,
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"neg1",
			inputArgs,
			true,
		},
		{
			"neg2",
			inputArgs1,
			true,
		},
		{
			"neg3",
			inputArgs2,
			true,
		},
		{
			"neg4",
			inputArgs3,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ImageDownload(tt.args.artifactinfo, tt.args.deviceInfo, tt.args.BkcImgDdLock, tt.args.JammyImgDdLock, tt.args.FocalImgDdLock, tt.args.FocalMsImgDdLock); (err != nil) != tt.wantErr {
				t.Errorf("ImageDownload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReadingYamlNCreatingResourse(t *testing.T) {
	type args struct {
		kubeconfigPath string
		imgurl         string
		imgType        string
		filePath       string
		fileName       string
		hwid           string
	}
	path, _ := generatekubeconfigPath()
	inputArgs := args{
		kubeconfigPath: path,
		imgurl:         "",
		imgType:        "",
		filePath:       "",
		fileName:       "",
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			"neg",
			inputArgs,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ReadingYamlNCreatingResourse(tt.args.imgurl, tt.args.imgType, tt.args.filePath, tt.args.fileName, tt.args.hwid); (err != nil) != tt.wantErr {
				t.Errorf("ReadingYamlNCreatingResourse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteCustomResource(t *testing.T) {
	type args struct {
		kubeconfigPath string
		u              *unstructured.Unstructured
	}
	inputArgs := args{
		kubeconfigPath: "",
		u:              &unstructured.Unstructured{},
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			"neg",
			inputArgs,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DeleteCustomResource(tt.args.u); (err != nil) != tt.wantErr {
				t.Errorf("DeleteCustomResource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDiWorkflowCreation(t *testing.T) {
	k, _ := generatekubeconfigPath()
	type args struct {
		deviceInfo     utils.DeviceInfo
		kubeconfigpath string
	}
	inputargs := args{
		deviceInfo:     utils.DeviceInfo{},
		kubeconfigpath: k,
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			"test",
			inputargs,
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DiWorkflowCreation(tt.args.deviceInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("DiWorkflowCreation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DiWorkflowCreation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVoucherScript(t *testing.T) {
	type args struct {
		hostIp       string
		deviceSerial string
	}
	inputargs := args{
		hostIp:       "",
		deviceSerial: "123",
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			"test",
			inputargs,
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := VoucherScript(tt.args.hostIp, tt.args.deviceSerial)
			if (err != nil) != tt.wantErr {
				t.Errorf("VoucherScript() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("VoucherScript() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeleteWorkflow(t *testing.T) {
	type args struct {
		kubeconfigPath string
		namespace      string
		workflowName   string
		resource       string
	}
	inputargs := args{
		kubeconfigPath: "",
		namespace:      "",
		workflowName:   "",
		resource:       "",
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			"",
			inputargs,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DeleteWorkflow(tt.args.kubeconfigPath, tt.args.namespace, tt.args.workflowName); (err != nil) != tt.wantErr {
				t.Errorf("DeleteWorkflow() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProdWorkflowCreation(t *testing.T) {
	k, _ := generatekubeconfigPath()
	type args struct {
		deviceInfo     utils.DeviceInfo
		kubeconfigpath string
		imgtype        string
	}
	inputargs := args{
		deviceInfo:     utils.DeviceInfo{},
		kubeconfigpath: k,
		imgtype:        "",
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"test",
			inputargs,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ProdWorkflowCreation(tt.args.deviceInfo, tt.args.kubeconfigpath); (err != nil) != tt.wantErr {
				t.Errorf("ProdWorkflowCreation() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetAllVariablesFromFile(t *testing.T) {
	content := []byte("KEY1=value1\nKEY2=value2\nKEY3=value3")
	filePath := createTempFiles(content)
	defer os.Remove(filePath)

	expected := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
		"KEY3": "value3",
	}

	runTest(t, filePath, expected)
	nonExistentFilePath := "nonexistentfile.txt"
	_, err := GetAllVariablesFromFile(nonExistentFilePath)
	if err == nil {
		t.Errorf("Expected an error for file not found, but got nil.")
	}
	emptyContent := []byte("")
	emptyFilePath := createTempFiles(emptyContent)
	defer os.Remove(emptyFilePath)

	emptyExpected := map[string]string{}
	runTest(t, emptyFilePath, emptyExpected)
	malformedContent := []byte("KEY1=value1\nINVALID_LINE\nKEY2=value2")
	malformedFilePath := createTempFiles(malformedContent)
	defer os.Remove(malformedFilePath)

	malformedExpected := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
	}
	runTest(t, malformedFilePath, malformedExpected)
}

func runTest(t *testing.T, filePath string, expected map[string]string) {
	result, err := GetAllVariablesFromFile(filePath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Result does not match the expected values.\nExpected: %v\nActual: %v", expected, result)
	}
}

func createTempFiles(content []byte) string {
	tmpFile, err := ioutil.TempFile("", "test_file_*.txt")
	if err != nil {
		panic(err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(content); err != nil {
		panic(err)
	}

	return tmpFile.Name()
}

var runtimeScheme = runtime.NewScheme()

func init() {
	_ = v1.AddToScheme(runtimeScheme)
}
