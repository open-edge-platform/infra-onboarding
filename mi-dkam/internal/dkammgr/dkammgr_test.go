package dkammgr

import (
        "testing"
	"strings"
        "github.com/stretchr/testify/assert"
        pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/api/grpc/dkammgr"
)

func isISOFile(filename string) bool {
    return strings.HasSuffix(filename, ".iso")
}

func TestGetBaseImage(t *testing.T) {
	
	// Create a GetBaseImageRequest
	request := &pb.GetProfileRequest{"Profile":"common"}

	// Assert that the request is not nil
	assert.NotNil(t, request)
	
	response := &pb.GetBaseImageResponse{BaseImageUrl: "https://af01p-png.devtools.intel.com/artifactory/hspe-edge-png-local/ubuntu-adlps/CI/20230801-1755/default/jammy-desktop-amd64+intel-iot-101-custom.img.bz2" , StatusCode: true}

	// Assert that the status code is valid	
	assert.Equal(t, true, response.StatusCode)

	// Assert that the BaseImageUrl is valid
	assert.Equal(t, true, isISOFile(response.BaseImageUrl))

}

func TestGetOverlaysScript(t *testing.T) {
	
	// Create a UploadBaseImageRequest
        request := &pb.GetProfileRequest{"Profile":"common"}
                

        // Assert that the request is not nil
        assert.NotNil(t, request)

	res := &pb.GetOverlaysScriptResponse{OverlaysScriptUrl:"https://ubit-artifactory-sh.intel.com/artifactory/sed-dgn-local/yocto/dev-test-image/DKAM/IAAS/ADL/installer.sh", StatusCode: true}

        // Assert that the status code is valid
        assert.Equal(t, true, res.StatusCode)

}


func TestGetArtifacts(t *testing.T) {

        // Create a UploadBaseImageRequest
        request := &pb.GetProfileRequest{"Profile":"common"}
        

        // Assert that the response is not nil
        assert.NotNil(t, request)

	res := &pb.GetArtifactsResponse{Message: "Successfully create artifacts", StatusCode: true}

        // Assert that the status code is valid
        assert.Equal(t, true, res.StatusCode)

        // Assert that the response message
        assert.Equal(t, "Successfully create artifacts", res.Message)
}

