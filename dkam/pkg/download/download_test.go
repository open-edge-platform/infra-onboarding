// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package download

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	as "github.com/intel/infra-core/inventory/v2/pkg/artifactservice"
	"github.com/intel/infra-onboarding/dkam/pkg/config"
	"github.com/intel/infra-onboarding/dkam/pkg/util"
	dkam_testing "github.com/intel/infra-onboarding/dkam/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

const (
	testDigest = "TEST_DIGEST"
	testFile   = "TEST_FILE"
)

// Manifest example from OCI repo, used by DKAM to gather the hookOS
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

const exampleManifest1 = `
		{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json",
		"config":{"mediaType":"application/vnd.intel.ensp.en",
		"digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2},
		"layers":[{
			"mediaType":"application/vnd.oci.image.layer.v1.tar",
			"digest":"` + testDigest + `",
			"size":24800
		}],
		"annotations":{"org.opencontainers.image.created":"2024-03-26T10:32:25Z"}}`

var projectRoot string

func TestMain(m *testing.M) {
	var err error
	config.PVC, err = os.MkdirTemp(os.TempDir(), "test_pvc")
	if err != nil {
		panic(fmt.Sprintf("Error creating temp directory: %v", err))
	}

	wd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("Error getting current directory: %v", err))
	}
	projectRoot = filepath.Dir(filepath.Dir(wd))

	run := m.Run()
	os.Exit(run)
}

func TestPathExists(t *testing.T) {
	tmpFolderPath, err := os.MkdirTemp("/tmp", "test_path_exist")
	require.NoError(t, err)
	defer os.RemoveAll(tmpFolderPath)

	expectedFilePath := tmpFolderPath + "/test"
	_, err = os.Create(expectedFilePath)
	require.NoError(t, err)

	exists, err := util.PathExists(expectedFilePath)
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = util.PathExists(tmpFolderPath + "/non_exist")
	require.NoError(t, err)
	assert.False(t, exists)

	invalidPath := string([]byte{0x00})
	exists, err = util.PathExists(invalidPath)
	assert.Error(t, err)
	assert.False(t, exists)
}

func TestDownloadMicroOS(t *testing.T) {
	dkam_testing.PrepareTestReleaseFile(t, projectRoot)
	expectedFileContent := "GOOD TEST!"

	// Create temporary folder and expected files and folder required by the DownloadMicroOS function
	tmpFolderPath, err := os.MkdirTemp("/tmp", "test_download_microOS")
	require.NoError(t, err)
	defer os.RemoveAll(tmpFolderPath)
	dkamTmpFolderPath := tmpFolderPath + "/tmp/"
	dkamHookFolderPath := tmpFolderPath + "/hook/"
	err = os.MkdirAll(dkamTmpFolderPath, 0o755)
	require.NoError(t, err)

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
	// Path comes from digest in the exampleManifest
	mux.HandleFunc("/blobs/"+testDigest, func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedFileContent))
	})
	svr := httptest.NewServer(mux)
	defer svr.Close()

	// Override the RSProxy with test HTTP server
	config.HookOSRepo = svr.URL + "/"
	dir := config.PVC
	os.MkdirAll(dir, 0o755)
	// Test: No tmpFolderPath/hook dir
	t.Run("Fail", func(t *testing.T) {
		_, err = DownloadMicroOS(context.Background())
		require.NoError(t, err)
		// assert.Contains(t, err.Error(), "no such file or directory")
	})

	err = os.MkdirAll(dkamHookFolderPath, 0o755)
	require.NoError(t, err)

	// Test: empty manifest
	t.Run("NoAnnotationLayer", func(t *testing.T) {
		returnWrongManifest = true
		_, err = DownloadMicroOS(context.Background())
		require.NoError(t, err)
	})

	// Test: successful, create tmpFolderPath/hook dir
	t.Run("Success", func(t *testing.T) {
		returnWrongManifest = false
		_, err = DownloadMicroOS(context.Background())
		require.NoError(t, err)
	})
}

func TestDownloadMicroOS_Case1(t *testing.T) {
	dkam_testing.PrepareTestReleaseFile(t, projectRoot)
	expectedFileContent := "GOOD TEST!"
	tmpFolderPath, err := os.MkdirTemp("/tmp", "test_download_microOS")
	require.NoError(t, err)
	defer os.RemoveAll(tmpFolderPath)
	dkamTmpFolderPath := tmpFolderPath + "/tmp/"
	dkamHookFolderPath := tmpFolderPath + "/hook/"
	err = os.MkdirAll(dkamTmpFolderPath, 0o755)
	require.NoError(t, err)

	mux := http.NewServeMux()
	returnWrongManifest := false
	mux.HandleFunc("/manifests/HOOK_OS_VERSION", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		if returnWrongManifest {
			w.Write([]byte(exampleManifestWrong))
		} else {
			w.Write([]byte(exampleManifest1))
		}
	})
	mux.HandleFunc("/blobs/"+testDigest, func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedFileContent))
	})
	svr := httptest.NewServer(mux)
	defer svr.Close()
	config.HookOSRepo = svr.URL + "/"
	t.Run("Fail", func(t *testing.T) {
		_, err = DownloadMicroOS(context.Background())
		// require.Error(t, err)
		// assert.Contains(t, err.Error(), "no such file or directory")
	})

	err = os.MkdirAll(dkamHookFolderPath, 0o755)
	require.NoError(t, err)

	// Test: empty manifest
	t.Run("NoAnnotationLayer", func(t *testing.T) {
		returnWrongManifest = true
		_, err = DownloadMicroOS(context.Background())
		// require.NoError(t, err)
	})

	t.Run("Success", func(t *testing.T) {
		existingDir := config.PVC
		subfolder := "tmp"
		subfolderPath := filepath.Join(existingDir, subfolder)
		err := os.MkdirAll(subfolderPath, 0o755)
		if err != nil {
			fmt.Println("Error creating subfolder:", err)
			return
		}

		data := config.ENManifest{
			Provisioning: config.Provisioning{
				Files: []config.File{
					{
						Description: "Dummy file 1",
						Server:      "server1",
						Path:        "edge-orch/en/files/provisioning-hook-os",
						Version:     "v1",
					},
				},
			},
		}
		yamlData, err := yaml.Marshal(&data)
		if err != nil {
			fmt.Printf("Error marshalling YAML: %v\n", err)
			return
		}
		yamlFilePath := filepath.Join(subfolderPath, "latest-dev.yaml")
		err = os.WriteFile(yamlFilePath, yamlData, 0o644)
		if err != nil {
			fmt.Printf("Error writing YAML to file: %v\n", err)
			return
		}
		//defer func() {
		//	// ignore error, usually not exists
		//	os.Remove(yamlFilePath)
		//}()
		returnWrongManifest = true
	})
}

func TestCreateFile(t *testing.T) {
	type args struct {
		filePath string
		artifact *as.Artifact
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "negative test case",
			args:    args{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateFile(tt.args.filePath, tt.args.artifact); (err != nil) != tt.wantErr {
				t.Errorf("CreateFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
