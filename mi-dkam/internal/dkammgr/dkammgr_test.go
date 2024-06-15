package dkammgr

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	pa "path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/api/dkammgr/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/config"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/download"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

const testDigest = "TEST_DIGEST"
const testFile = "TEST_FILE"
const exampleManifest = `
		{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json",
		"config":{"mediaType":"application/vnd.intel.ensp.en",
		"digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2},
		"layers":[{
			"mediaType":"application/vnd.oci.image.layer.v1.tar",
			"digest":"` + testDigest + `",
			"size":24800,
			"annotations":{"org.opencontainers.image.title":"` + testFile + `"}
		}],
		"annotations":{"org.opencontainers.image.created":"2024-03-26T10:32:25Z"}}`

// Manifest example with no Annotation in Layers
const exampleManifestWrong = `
		{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json",
		"config":{"mediaType":"application/vnd.intel.ensp.en",
		"digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2},
		"layers":[{
			"mediaType":"application/vnd.oci.image.layer.v1.tar",
			"digest":"` + testDigest + `",
			"size":24800
		}],
		"annotations":{"org.opencontainers.image.created":"2024-03-26T10:32:25Z"}}`

func TestGetArtifacts(t *testing.T) {
	dir := config.PVC
	dummyData := `#!/bin/bash
	enable_netipplan
# Add your installation commands here
`
	err := os.WriteFile(dir+"/installer.sh", []byte(dummyData), 0755)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}
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
	defer func() {
		os.Remove(config.PVC + "/installer.sh")
	}()

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

func TestGetCuratedScript(t *testing.T) {
	dir := config.PVC
	os.MkdirAll(dir, 0755)
	dummyData := `#!/bin/bash
	enable_netipplan
        install_intel_CAcertificates
# Add your installation commands here
`
	err := os.WriteFile(dir+"/installer.sh", []byte(dummyData), 0755)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}
	filename, version := GetCuratedScript("profile", "platform")

	// Check if the returned filename matches the expected format
	expectedFilename := config.PVC + "/" + "installer.sh"
	if filename != expectedFilename {
		t.Errorf("Expected filename '%s', but got '%s'", expectedFilename, filename)
	}
	if len(version) == 0 {
		t.Errorf("Version not found")
	}
	defer func() {
		os.Remove(dir + "/installer.sh")
	}()
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

// func TestDownloadOS(t *testing.T) {

// 	// Test download function
// 	if err := DownloadOS(); err != nil {
// 		t.Errorf("Download failed: %v", err)
// 	}

// }

func TestAccessConfigs(t *testing.T) {
	val := AccessConfigs()
	if len(val) <= 0 {
		t.Errorf("Unexpected error!")
	}
}

func TestFileNameFromURL(t *testing.T) {
	imageURL := "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img"
	result := fileNameFromURL(imageURL)
	assert.Equal(t, "jammy-server-cloudimg-amd64.img", result)
}

func TestDownloadArtifacts_Case(t *testing.T) {

	// Create a UploadBaseImageRequest
	os.Setenv("MODE", "preint")
	err := DownloadArtifacts()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer func() {
		os.Unsetenv("MODE")
	}()
}

func TestDownloadArtifacts_Case1(t *testing.T) {
	os.Setenv("KUBERNETES_SERVICE_HOST", "localhost")
	os.Setenv("KUBERNETES_SERVICE_PORT", "2521")
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Println("Failed to generate private key:", err)
		return
	}
	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{"Dummy Org"}},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caCertBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		fmt.Println("Failed to create CA certificate:", err)
		return
	}
	path := "/var"
	dummypath := "/run/secrets/kubernetes.io/serviceaccount/"
	cerr := os.MkdirAll(path+dummypath, 0755)
	if cerr != nil {
		t.Fatalf("Error creating directory: %v", cerr)
	}
	file, crErr := os.Create(path + dummypath + "token")
	if crErr != nil {
		t.Fatalf("Error creating file: %v", crErr)
	}
	fmt.Println("token File :", file.Name())
	defer func() {
		remErr := os.RemoveAll("/run/secrets/kubernetes.io/serviceaccount/token")
		if remErr != nil {
			t.Fatalf("Error while removing file: %v", remErr)
		}
	}()
	dummyData := "Thisissomedummydata"
	_, err = file.WriteString(dummyData)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}
	certOut, cerrErr := os.Create(path + dummypath + "ca.crt")
	if cerrErr != nil {
		t.Fatalf("Error creating cert file: %v", cerrErr)
	}
	fmt.Println("certOut File :", certOut.Name())
	fmt.Println("CA certificate created successfully as ca.crt")
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: caCertBytes})
	defer func() {
		remErr := os.RemoveAll("/run/secrets/kubernetes.io/serviceaccount/ca.crt")
		if remErr != nil {
			t.Fatalf("Error while removing file: %v", remErr)
		}
	}()
	file.Close()
	certOut.Close()
	testTag := "manifest"
	testManifest := "testManifest"
	exampleDownloadManifest := `{"layers":[{"digest":"` + testDigest + `"}]}`

	mux := http.NewServeMux()
	data := download.Data{}
	data.Provisioning.Files = append(data.Provisioning.Files, download.File{
		Description: "Script file",
		Server:      "example.com",
		Path:        "",
		Version:     "2.3",
	})
	yamlData, err := yaml.Marshal(&data)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	mux.HandleFunc("/"+testTag+"/manifests/"+testManifest, func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(exampleDownloadManifest))
	})
	mux.HandleFunc("/"+testTag+"/blobs/"+testDigest, func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(yamlData))
	})
	svr := httptest.NewServer(mux)
	defer svr.Close()
	config.RSProxyManifest = svr.URL + "/"
	os.Setenv("MANIFEST_TAG", "testManifest")
	_, filename, _, _ := runtime.Caller(0)
	localPath := pa.Dir(filename)
	expectedFileContent := "GOOD TEST!"
	tmpFolderPath, err := os.MkdirTemp("/tmp", "test_download_microOS")
	require.NoError(t, err)
	defer os.RemoveAll(tmpFolderPath)
	dkamTmpFolderPath := tmpFolderPath + "/tmp/"
	dkamHookFolderPath := tmpFolderPath + "/hook/"
	err = os.MkdirAll(dkamTmpFolderPath, 0755)
	require.NoError(t, err)
	expectedManifestFilePath := dkamTmpFolderPath + config.ReleaseVersion + ".yaml"
	fileData, filrErr := os.ReadFile(localPath + "/../../test/testdata/example-manifest-internal-rs.yaml")
	require.NoError(t, filrErr)
	err = os.WriteFile(expectedManifestFilePath, fileData, 0755)
	require.NoError(t, filrErr)
	returnWrongManifest := false
	mux.HandleFunc("/manifests/", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		if returnWrongManifest {
			w.Write([]byte(exampleManifestWrong))
		} else {
			w.Write([]byte(exampleManifest))
		}
	})
	mux.HandleFunc("/blobs/"+testDigest, func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedFileContent))
	})
	os.MkdirAll(dkamHookFolderPath, 0755)
	sver := httptest.NewServer(mux)
	defer sver.Close()
	config.RSProxy = sver.URL + "/"
	DownloadErr := DownloadArtifacts()
	if DownloadErr != nil {
		t.Errorf("Unexpected error: %v", DownloadErr)
	}
	originalDir, _ := os.Getwd()
	result := strings.Replace(originalDir, "script", "script/tmp", -1)
	res := filepath.Join(result, "latest-dev.yaml")
	if err := os.MkdirAll(filepath.Dir(res), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	src := strings.Replace(originalDir, "curation", "script/latest-dev.yaml", -1)
	CopyFile(src, res)
	defer func() {
		CopyFile(res, src)
		os.Remove(res)
		os.Remove(originalDir + "/hook/TEST_FILE")
	}()
}

func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	if err := os.MkdirAll(filepath.Dir(src), 0755); err != nil {
		return err
	}
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	return nil
}

func TestDownloadUbuntuImage(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(make([]byte, 20))
	}))
	defer svr.Close()
	tmpFolderPath, err := os.MkdirTemp("/tmp", "test_download_ubuntu")
	require.NoError(t, err)
	defer os.RemoveAll(tmpFolderPath)
	expectedFileName := "final.raw.gz"
	imgName := "image.img"
	dir := config.PVC
	os.MkdirAll(dir, 0755)
	err = download.DownloadUbuntuImage(svr.URL, imgName, expectedFileName, dir)
	require.NoError(t, err)
	_, err = os.Stat(config.PVC + "/" + expectedFileName)
	assert.NoError(t, err)
	defer func() {
		os.Remove(dir)
	}()
}

func TestGetScriptDir_Case1(t *testing.T) {
	currentDir, err := os.Getwd()
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error getting current working directory: %v", err)
		return
	}
	zlog.MiSec().Info().Msgf("Current dir %s", currentDir)
	dir := currentDir + "/pkg/script"
	os.MkdirAll(dir, 0755)
	GetScriptDir()
	defer func() {
		os.Remove(dir)
	}()
}

// func TestDownloadOS_Case1(t *testing.T) {
// 	originalDir, _ := os.Getwd()
// 	result := strings.Replace(originalDir, "internal/dkammgr", "pkg/script/jammy-server-cloudimg-amd64.raw.gz", -1)
// 	os.MkdirAll(result, 0755)
// 	if err := DownloadOS(); err != nil {
// 		t.Errorf("Download failed: %v", err)
// 	}
// }
