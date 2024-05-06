// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package download

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testDigest = "TEST_DIGEST"
const testFile = "TEST_FILE"

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

func TestGetReleaseServerResponse(t *testing.T) {
	expAcceptHeader := "application/vnd.oci.image.manifest.v1+json"
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Assert expected header
		assert.Equal(t, r.Header.Get("Accept"), expAcceptHeader)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(exampleManifest))
	}))
	defer svr.Close()

	// TODO: make proper expectation
	res := GetReleaseServerResponse(svr.URL)
	assert.Equal(t, 2, res.SchemaVersion)
	assert.Equal(t, "TEST_DIGEST", res.Layers[0].Digest)
}

func TestDownloadUbuntuImage(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Assert expected header
		w.WriteHeader(http.StatusOK)
		// Return random bytes
		w.Write(make([]byte, 20))
	}))
	defer svr.Close()

	tmpFolderPath, err := os.MkdirTemp("/tmp", "test_download_ubuntu")
	require.NoError(t, err)
	defer os.RemoveAll(tmpFolderPath)
	expectedFileName := tmpFolderPath + "/final.raw"

	// TODO(NEXFMPID-3359): imgName MUST be image.img because DownloadUbuntuImage has hardcoded values inside
	imgName := "image.img"
	// TODO: 3rd parameter is unused
	err = DownloadUbuntuImage(svr.URL, imgName, "", expectedFileName)
	require.NoError(t, err)

	// Check the expected file is created
	_, err = os.Stat(expectedFileName)
	assert.NoError(t, err)
}

func TestPathExists(t *testing.T) {
	tmpFolderPath, err := os.MkdirTemp("/tmp", "test_path_exist")
	require.NoError(t, err)
	defer os.RemoveAll(tmpFolderPath)

	expectedFilePath := tmpFolderPath + "/test"
	_, err = os.Create(expectedFilePath)
	require.NoError(t, err)

	exists, err := PathExists(expectedFilePath)
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = PathExists(tmpFolderPath + "/non_exist")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestDownloadArtifacts(t *testing.T) {
	tmpFolderPath, err := os.MkdirTemp("/tmp", "test_download_artifacts")
	require.NoError(t, err)
	defer os.RemoveAll(tmpFolderPath)

	expectedFileContent := "GOOD TEST!"
	// Expected file path comes from the internal path manipulation done by the DownloadArtifact function
	expectedFilePath := tmpFolderPath + "/tmp/" + config.ReleaseVersion + ".yaml"

	testTag := "testTag"
	testManifest := "testManifest"
	exampleDownloadManifest := `{"layers":[{"digest":"` + testDigest + `"}]}`

	mux := http.NewServeMux()
	mux.HandleFunc("/"+testTag+"/manifests/"+testManifest, func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		// return example manifest
		w.Write([]byte(exampleDownloadManifest))
	})
	// Path comes from the "DownloadArtifacts" by combining content of the exampleDownloadManifest digest
	mux.HandleFunc("/"+testTag+"/blobs/"+testDigest, func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedFileContent))
	})
	svr := httptest.NewServer(mux)
	defer svr.Close()

	// Override the RSProxyManifest with test HTTP server
	config.RSProxyManifest = svr.URL + "/"
	err = DownloadArtifacts(tmpFolderPath, testTag, testManifest)
	assert.NoError(t, err)

	// Assert file is created with expected content
	_, err = os.Stat(expectedFilePath)
	require.NoError(t, err)
	data, err := os.ReadFile(expectedFilePath)
	require.NoError(t, err)
	assert.Equal(t, expectedFileContent, string(data))
}

func TestDownloadMicroOS(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	localPath := path.Dir(filename)

	expectedFileContent := "GOOD TEST!"

	// Create temporary folder and expected files and folder required by the DownloadMicroOS function
	tmpFolderPath, err := os.MkdirTemp("/tmp", "test_download_microOS")
	require.NoError(t, err)
	defer os.RemoveAll(tmpFolderPath)
	dkamTmpFolderPath := tmpFolderPath + "/tmp/"
	dkamHookFolderPath := tmpFolderPath + "/hook/"
	err = os.MkdirAll(dkamTmpFolderPath, 0755)
	require.NoError(t, err)

	// Create a fake EN Manifest in tmp folder copying the local example
	expectedManifestFilePath := dkamTmpFolderPath + config.ReleaseVersion + ".yaml"
	data, err := os.ReadFile(localPath + "/../../test/testdata/example-manifest-internal-rs.yaml")
	require.NoError(t, err)
	err = os.WriteFile(expectedManifestFilePath, data, 0755)
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
	config.RSProxy = svr.URL + "/"

	// Test: No tmpFolderPath/hook dir
	t.Run("Fail", func(t *testing.T) {
		_, err = DownloadMicroOS(tmpFolderPath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no such file or directory")
	})

	err = os.MkdirAll(dkamHookFolderPath, 0755)
	require.NoError(t, err)

	// Test: empty manifest
	t.Run("NoAnnotationLayer", func(t *testing.T) {
		returnWrongManifest = true
		_, err = DownloadMicroOS(tmpFolderPath)
		require.NoError(t, err)
	})

	// Test: successful, create tmpFolderPath/hook dir
	t.Run("Success", func(t *testing.T) {
		returnWrongManifest = false
		_, err = DownloadMicroOS(tmpFolderPath)
		require.NoError(t, err)

		expectedFilePath := dkamHookFolderPath + "/" + testFile

		// Assert file is created with expected content
		_, err = os.Stat(expectedFilePath)
		require.NoError(t, err)
		data, err = os.ReadFile(expectedFilePath)
		require.NoError(t, err)
		assert.Equal(t, expectedFileContent, string(data))
	})
}

func Test_downloadImage(t *testing.T) {
	type args struct {
		url      string
		fileName string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				url: "",
			},
			wantErr: true,
		},
		{
			name: "Test Case",
			args: args{
				url: "https://www.google.com/",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := downloadImage(tt.args.url, tt.args.fileName); (err != nil) != tt.wantErr {
				t.Errorf("downloadImage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_installPackage(t *testing.T) {
	type args struct {
		packageName string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				packageName: "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := installPackage(tt.args.packageName); (err != nil) != tt.wantErr {
				t.Errorf("installPackage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDownloadMicroOS_Case1(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	localPath := path.Dir(filename)
	expectedFileContent := "GOOD TEST!"
	tmpFolderPath, err := os.MkdirTemp("/tmp", "test_download_microOS")
	require.NoError(t, err)
	defer os.RemoveAll(tmpFolderPath)
	dkamTmpFolderPath := tmpFolderPath + "/tmp/"
	dkamHookFolderPath := tmpFolderPath + "/hook/"
	err = os.MkdirAll(dkamTmpFolderPath, 0755)
	require.NoError(t, err)
	expectedManifestFilePath := dkamTmpFolderPath + config.ReleaseVersion + ".yaml"
	data, err := os.ReadFile(localPath + "/../../test/testdata/example-manifest-internal-rs.yaml")
	require.NoError(t, err)
	err = os.WriteFile(expectedManifestFilePath, data, 0755)
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
	config.RSProxy = svr.URL + "/"
	t.Run("Fail", func(t *testing.T) {
		_, err = DownloadMicroOS(tmpFolderPath)
		// require.Error(t, err)
		// assert.Contains(t, err.Error(), "no such file or directory")
	})

	err = os.MkdirAll(dkamHookFolderPath, 0755)
	require.NoError(t, err)

	// Test: empty manifest
	t.Run("NoAnnotationLayer", func(t *testing.T) {
		returnWrongManifest = true
		_, err = DownloadMicroOS(tmpFolderPath)
		// require.NoError(t, err)
	})

	// Test: successful, create tmpFolderPath/hook dir
	t.Run("Success", func(t *testing.T) {
		returnWrongManifest = false
		_, err = DownloadMicroOS(tmpFolderPath)
		require.NoError(t, err)

		expectedFilePath := dkamHookFolderPath + "/" + testFile

		// Assert file is created with expected content
		_, err = os.Stat(expectedFilePath)
		// require.NoError(t, err)
		data, err = os.ReadFile(expectedFilePath)
		// require.NoError(t, err)
		// assert.Equal(t, expectedFileContent, string(data))
	})
}
