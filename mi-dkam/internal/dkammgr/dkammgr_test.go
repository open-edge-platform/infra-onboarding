package dkammgr

import (
        "testing"
        "context"
	"github.com/stretchr/testify/assert"
        pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/api/grpc/dkammgr"
)


func TestGetBaseImage(t *testing.T) {
	
	// Create a GetBaseImageRequest
	request := &pb.GetArtifactsRequest{ProfileName:"common", Platform:"ASUS"}

	// Assert that the request is not nil
	assert.NotNil(t, request)
	
	response := &pb.GetArtifactsResponse{ManifestFile: "manifest.yaml" , StatusCode: true}

	// Assert that the status code is valid	
	assert.Equal(t, true, response.StatusCode)


}


func TestGetArtifacts(t *testing.T) {
	
	// Initialize the service
	service := &Service{}

        // Create a UploadBaseImageRequest
        request := &pb.GetArtifactsRequest{ProfileName:"common", Platform:"ASUS"}
        

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

