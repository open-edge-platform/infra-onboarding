package dkammgr

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/api/dkammgr/v1"
	"github.com/stretchr/testify/assert"
)

func TestGetArtifacts(t *testing.T) {

	// Initialize the service
	service := &Service{}

	// Create a UploadBaseImageRequest
	request := &pb.GetArtifactsRequest{ProfileName: "common", Platform: "ASUS"}

	// Call the GetTelemetryQuery function
	response, err := service.GetArtifacts(context.Background(), request)
	if err != nil {
		t.Fatalf("Error calling GetArtifacts: %v", err)
	}
	// Check for errors in the response
	assert.NoError(t, err)
	// Assert that the response is not nil
	assert.NotNil(t, response)
	assert.Equal(t, true, isImageFile(response.OsUrl))

}

func isImageFile(filename string) bool {
	return strings.HasSuffix(filename, ".raw.gz")
}

func TestDownloadArtifacts(t *testing.T) {

	// Create a UploadBaseImageRequest

	err := DownloadArtifacts()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

// func TestDataType(t *testing.T) {
// 	// Call the function or create an instance of the struct
// 	data := GetData()

// 	// Check if the variable 'data' is of type 'Data'
// 	if reflect.TypeOf(data) != reflect.TypeOf(Data{}) {
// 		t.Errorf("Expected type 'Data', but got %T", data)
// 	}
// }

// func TestResponseContainsYAML(t *testing.T) {
// 	type Data struct {
// 		OsUrl            string
// 		OverlayScriptUrl string
// 	}

// 	data := GetData()

// 	// Marshal the data to YAML
// 	yamlContent, err := yaml.Marshal(data)
// 	if err != nil {
// 		t.Fatalf("Error marshaling data to YAML: %v", err)
// 	}

// 	var responseData Data
// 	err = yaml.Unmarshal(yamlContent, &responseData)
// 	if err != nil {
// 		t.Fatalf("Error unmarshaling YAML data: %v", err)
// 	}

// 	// Check if the YAML data contains the expected message
// 	expectedMessage := "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img"
// 	if responseData.OsUrl != expectedMessage {
// 		t.Errorf("Expected message '%s', but got '%s'", expectedMessage, responseData.OsUrl)
// 	}
//}

func TestGetCuratedScript(t *testing.T) {
	filename, version := GetCuratedScript("profile", "platform")

	// Check if the returned filename matches the expected format
	expectedFilename := "installer.sh"
	if filename != expectedFilename {
		t.Errorf("Expected filename '%s', but got '%s'", expectedFilename, filename)
	}
	if len(version) == 0 {
		t.Errorf("Version not found")
	}

}

func TestServerUrl(t *testing.T) {
	// Save the original value of MODE so that it can be restored later
	originalurl := os.Getenv("DNS_NAME")

	// Defer the restoration of the original value
	defer func() {
		os.Setenv("DNS_NAME", originalurl)
	}()

	tests := []struct {
		name         string
		testMode     string
		expectedMode string
	}{
		{"Mode is set", "dev", "dev"},
		{"Mode is not set", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the test value for MODE
			os.Setenv("DNS_NAME", tt.testMode)

			result := GetServerUrl()
			if result != tt.expectedMode {
				t.Errorf("Expected %v, but got %v", tt.expectedMode, result)
			}
		})
	}
}

func TestGetMode(t *testing.T) {
	// Save the original value of MODE so that it can be restored later
	originalMode := os.Getenv("MODE")

	// Defer the restoration of the original value
	defer func() {
		os.Setenv("MODE", originalMode)
	}()

	tests := []struct {
		name         string
		testMode     string
		expectedMode string
	}{
		{"Mode is set", "production", "production"},
		{"Mode is not set", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the test value for MODE
			os.Setenv("MODE", tt.testMode)

			result := GetMODE()
			if result != tt.expectedMode {
				t.Errorf("Expected %v, but got %v", tt.expectedMode, result)
			}
		})
	}
}

func TestGetScriptDir(t *testing.T) {
	currentDir, err := os.Getwd()
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error getting current working directory: %v", err)
		return
	}
	zlog.MiSec().Info().Msgf("Current dir %s", currentDir)
	// Call the function you want to test
	scriptPath := GetScriptDir()
	parentDir := filepath.Join(currentDir, "..", "..")
	expectedPath := filepath.Join(parentDir, "pkg", "script")

	// Check if the actual path matches the expected path
	if scriptPath != expectedPath {
		t.Errorf("Expected script path: %s, got: %s", expectedPath, scriptPath)
	}
}

func TestSignMicroOS(t *testing.T) {

	// Call the function you want to test
	result, err := SignMicroOS()

	// Check if the result matches the expected value
	if result != true {
		t.Errorf("Expected result to be true, got %t", result)
	}

	// Check if the error is nil
	if err != nil {
		t.Errorf("Expected error to be nil, got %v", err)
	}
}

func TestBuildSignIpxe(t *testing.T) {

	// Call the function you want to test
	result, err := BuildSignIpxe()

	// Check if the result matches the expected value
	if result != true {
		t.Errorf("Expected result to be true, got %t", result)
	}

	// Check if the error is nil
	if err != nil {
		t.Errorf("Expected error to be nil, got %v", err)
	}
}

func TestDownloadOS(t *testing.T) {

	// Test download function
	if err := DownloadOS(); err != nil {
		t.Errorf("Download failed: %v", err)
	}

}
