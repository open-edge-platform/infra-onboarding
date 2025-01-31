package dkammgr

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
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

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/pkg/util"

	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/api/os/v1"
	inv_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/testing"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/pkg/config"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/pkg/download"
	dkam_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

const testDigest = "TEST_DIGEST"
const testFile = "TEST_FILE"
const testImage = "TEST_IMAGE.raw.xz"
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

const exampleManifests = `
		{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json",
		"config":{"mediaType":"application/vnd.intel.ensp.en",
		"digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2},
		"layers":[{
			"mediaType":"application/vnd.oci.image.layer.v1.tar",
			"digest":"` + testDigest + `",
			"size":24800,
			"annotations":{"org.opencontainers.image.title":"` + testImage + `"}
		}],
		"annotations":{"org.opencontainers.image.created":"2024-03-26T10:32:25Z"}}`

var projectRoot string

func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	config.PVC, err = os.MkdirTemp(os.TempDir(), "test_pvc")
	if err != nil {
		panic(fmt.Sprintf("Error creating temp directory: %v", err))
	}

	projectRoot = filepath.Dir(filepath.Dir(wd))
	policyPath := projectRoot + "/build"
	migrationsDir := projectRoot + "/build"

	cleanupFunc := dkam_testing.StartTestReleaseService("profile")
	defer cleanupFunc()

	inv_testing.StartTestingEnvironment(policyPath, "", migrationsDir)
	run := m.Run()
	inv_testing.StopTestingEnvironment()

	os.Exit(run)
}

func TestDownloadArtifacts(t *testing.T) {
	dkam_testing.PrepareTestReleaseFile(t, projectRoot)
	// Create a UploadBaseImageRequest

	err := DownloadArtifacts(context.Background())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGetCuratedScript(t *testing.T) {
	dkam_testing.PrepareTestReleaseFile(t, projectRoot)
	dkam_testing.PrepareTestCaCertificateFile(t)

	dir := config.PVC
	os.MkdirAll(dir, 0755)
	os.MkdirAll(config.DownloadPath, 0755)
	currentDir, err := os.Getwd()
	if err != nil {
		zlog.InfraSec().Fatal().Err(err).Msgf("Error getting current working directory: %v", err)
		return
	}
	zlog.InfraSec().Info().Msgf("Current dir %s", currentDir)
	parentDir := filepath.Join(currentDir, "..", "..")
	config.ScriptPath = parentDir + "/pkg/script"
	dummyData := `#!/bin/bash
	enable_netipplan
        install_intel_CAcertificates
# Add your installation commands here
`
	os.Setenv("ORCH_CLUSTER", "kind.internal")
	defer os.Unsetenv("ORCH_CLUSTER")
	err = os.WriteFile(dir+"/installer.sh", []byte(dummyData), 0755)
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}
	defer func() {
		os.Remove(dir + "/installer.sh")
	}()

	osr := &osv1.OperatingSystemResource{
		ProfileName: "profile",
		OsType:      osv1.OsType_OS_TYPE_MUTABLE,
	}
	err = GetCuratedScript(context.TODO(), osr)

	// Check if the returned filename matches the expected format
	assert.NoError(t, err)
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

func TestSignMicroOS(t *testing.T) {
	currentDir, err := os.Getwd()
	if err != nil {
		zlog.InfraSec().Fatal().Err(err).Msgf("Error getting current working directory: %v", err)
		return
	}
	zlog.InfraSec().Info().Msgf("Current dir %s", currentDir)
	parentDir := filepath.Join(currentDir, "..", "..")
	config.ScriptPath = parentDir + "/pkg/script"

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

func TestBuildSignIpxe1(t *testing.T) {
	currentDir, err := os.Getwd()
	if err != nil {
		zlog.InfraSec().Fatal().Err(err).Msgf("Error getting current working directory: %v", err)
		return
	}
	zlog.InfraSec().Info().Msgf("Current dir %s", currentDir)
	parentDir := filepath.Join(currentDir, "..", "..")
	config.ScriptPath = parentDir + "/pkg/script"

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
	osUrl := "https://cloud-images.ubuntu.com/releases/22.04/release-20240912/ubuntu-22.04-server-cloudimg-amd64.img"
	sha256 := "5da0b3d37d02ca6c6760caa4041b4df14e08abc7bc9b2db39133eef8ee145f6d"
	osr := &osv1.OperatingSystemResource{
		ImageUrl:   osUrl,
		OsType:     osv1.OsType_OS_TYPE_MUTABLE,
		Sha256:     sha256,
		OsProvider: osv1.OsProviderKind_OS_PROVIDER_KIND_INFRA,
	}

	expectedFilePath := util.GetOSImageLocation(osr, config.PVC)
	err := os.MkdirAll(filepath.Dir(expectedFilePath), 0755)
	if err != nil {
		t.Fatalf("Failed to create directories: %v", err)
	}
	file, err := os.Create(expectedFilePath)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	file.Close()
	defer func() {
		err := os.Remove(expectedFilePath)
		if err != nil && !os.IsNotExist(err) {
			t.Fatalf("Failed to remove file: %v", err)
		}
		err = os.RemoveAll(filepath.Dir(expectedFilePath))
		if err != nil {
			t.Fatalf("Failed to clean up directories: %v", err)
		}
	}()

	if err := DownloadOS(context.TODO(), osr); err != nil {
		t.Errorf("Download failed: %v", err)
	}
}

func TestDownloadArtifacts_Case(t *testing.T) {
	dkam_testing.PrepareTestReleaseFile(t, projectRoot)

	// Fake server to serve expected requests
	mux := http.NewServeMux()
	returnWrongManifest := false
	mux.HandleFunc("/manifests/HOOK_OS_VERSION", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		if returnWrongManifest {
			w.Write([]byte(exampleManifestWrong))
		} else {
			w.Write([]byte(exampleManifest))
		}
	})
	svr := httptest.NewServer(mux)
	defer svr.Close()
	// Override the RSProxy with test HTTP server
	config.HookOSRepo = svr.URL + "/"

	// Create a UploadBaseImageRequest
	os.Setenv("MODE", "preint")
	err := DownloadArtifacts(context.Background())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer func() {
		os.Unsetenv("MODE")
	}()
}

func TestDownloadArtifacts_Case1(t *testing.T) {
	dkam_testing.PrepareTestReleaseFile(t, projectRoot)
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
	config.ENManifestRepo = svr.URL + "/"
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
	os.WriteFile(expectedManifestFilePath, fileData, 0755)
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
	config.ENManifestRepo = sver.URL + "/"
	DownloadErr := DownloadArtifacts(context.Background())
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
	dkam_testing.CopyFile(src, res)
	defer func() {
		dkam_testing.CopyFile(res, src)
		os.Remove(res)
		os.Remove(originalDir + "/hook/TEST_FILE")
	}()
}

func TestDownloadOs(t *testing.T) {
	osUrl := "repository/TiberOS/TiberOS-RT/tiber-readonly-rt-1.0.20241117.1004.raw.gz"
	sha256 := "de04d58dc5ccc4b9671c3627fb8d626fe4a15810bc1fe3e724feea761965f666"
	parts := strings.Split(osUrl, "/")
	fileName := parts[len(parts)-1]
	rawFileName := strings.TrimSuffix(fileName, ".img") + ".raw.gz"
	expectedFilePath := config.PVC + "/OSImage/" + sha256 + "/" + rawFileName
	err := os.MkdirAll(filepath.Dir(expectedFilePath), 0755)
	if err != nil {
		t.Fatalf("Failed to create directories: %v", err)
	}
	file, err := os.Create(expectedFilePath)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	path := config.DownloadPath + "/profile.raw.gz"
	err = os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		t.Fatalf("Failed to create directories: %v", err)
	}
	osfile, oserr := os.Create(path)
	if oserr != nil {
		t.Fatalf("Failed to create file: %v", oserr)
	}
	osfile.Close()
	mux := http.NewServeMux()
	mux.HandleFunc("/repository/TiberOS/TiberOS-RT/tiber-readonly-rt-1.0.20241117.1004.raw.gz", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(exampleManifests))
	})
	svr := httptest.NewServer(mux)
	defer svr.Close()

	file.Close()
	defer func() {
		err := os.Remove(expectedFilePath)
		if err != nil && !os.IsNotExist(err) {
			t.Fatalf("Failed to remove file: %v", err)
		}
		err = os.RemoveAll(filepath.Dir(expectedFilePath))
		if err != nil {
			t.Fatalf("Failed to clean up directories: %v", err)
		}
		err = os.Remove(path)
		if err != nil && !os.IsNotExist(err) {
			t.Fatalf("Failed to remove file: %v", err)
		}
	}()
	type args struct {
		osRes *osv1.OperatingSystemResource
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				osRes: &osv1.OperatingSystemResource{
					ImageUrl: osUrl,
					OsType:   0,
					Sha256:   sha256,
				},
			},
			wantErr: false,
		},
		{
			name: "Test Case With Os type",
			args: args{
				osRes: &osv1.OperatingSystemResource{
					ImageUrl: osUrl,
					OsType:   osv1.OsType_OS_TYPE_IMMUTABLE,
					Sha256:   sha256,
				},
			},
			wantErr: false,
		},
		{
			name: "Test Case With Os Immutable type",
			args: args{
				osRes: &osv1.OperatingSystemResource{
					ProfileName: "profile",
					ImageUrl:    osUrl,
					OsType:      osv1.OsType_OS_TYPE_IMMUTABLE,
					Sha256:      sha256,
					OsProvider:  osv1.OsProviderKind_OS_PROVIDER_KIND_INFRA,
				},
			},
			wantErr: false,
		},
		{
			name: "Test Case With dummy url",
			args: args{
				osRes: &osv1.OperatingSystemResource{
					ProfileName: "profile",
					ImageUrl:    "osUrl",
					OsType:      osv1.OsType_OS_TYPE_IMMUTABLE,
					Sha256:      sha256,
					OsProvider:  osv1.OsProviderKind_OS_PROVIDER_KIND_INFRA,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DownloadOS(context.TODO(), tt.args.osRes); (err != nil) != tt.wantErr {
				t.Errorf("DownloadOS() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

}
