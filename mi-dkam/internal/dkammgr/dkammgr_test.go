package dkammgr

import (
	"context"
	"os"
	"reflect"
	"testing"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/api/grpc/dkammgr"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
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

}

func TestDownloadArtifacts(t *testing.T) {

	// Create a UploadBaseImageRequest

	err := DownloadArtifacts()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestDataType(t *testing.T) {
	// Call the function or create an instance of the struct
	data := GetData()

	// Check if the variable 'data' is of type 'Data'
	if reflect.TypeOf(data) != reflect.TypeOf(Data{}) {
		t.Errorf("Expected type 'Data', but got %T", data)
	}
}

func TestResponseContainsYAML(t *testing.T) {
	type Data struct {
		OsUrl            string
		OverlayScriptUrl string
	}

	data := GetData()

	// Marshal the data to YAML
	yamlContent, err := yaml.Marshal(data)
	if err != nil {
		t.Fatalf("Error marshaling data to YAML: %v", err)
	}

	var responseData Data
	err = yaml.Unmarshal(yamlContent, &responseData)
	if err != nil {
		t.Fatalf("Error unmarshaling YAML data: %v", err)
	}

	// Check if the YAML data contains the expected message
	expectedMessage := "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img"
	if responseData.OsUrl != expectedMessage {
		t.Errorf("Expected message '%s', but got '%s'", expectedMessage, responseData.OsUrl)
	}
}

func TestGetCuratedScript(t *testing.T) {
	filename := GetCuratedScript("profile", "platform")

	// Check if the returned filename matches the expected format
	expectedFilename := "installer.sh"
	if filename != expectedFilename {
		t.Errorf("Expected filename '%s', but got '%s'", expectedFilename, filename)
	}
}

func TestServerUrl(t *testing.T) {
	// Save the original value of MODE so that it can be restored later
	originalMode := os.Getenv("ServerUrl")

	// Defer the restoration of the original value
	defer func() {
		os.Setenv("ServerUrl", originalMode)
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
			os.Setenv("ServerUrl", tt.testMode)

			result := GetServerUrl()
			if result != tt.expectedMode {
				t.Errorf("Expected %v, but got %v", tt.expectedMode, result)
			}
		})
	}
}
