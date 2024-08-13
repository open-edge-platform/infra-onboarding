// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package download

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
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

const example = `#!/bin/bash
		echo "This is a example script."
		`

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
	expectedFileName := "final.raw.gz"

	dir := config.PVC
	os.MkdirAll(dir, 0755)

	// TODO(NEXFMPID-3359): imgName MUST be image.img because DownloadUbuntuImage has hardcoded values inside
	imgName := "image.img"
	// TODO: 3rd parameter is unused
	err = DownloadUbuntuImage(svr.URL, imgName, expectedFileName, config.DownloadPath, "")
	require.NoError(t, err)

	// Check the expected file is created
	// _, err = os.Stat(config.PVC + "/" + expectedFileName)
	// assert.NoError(t, err)

	defer func() {
		os.Remove(dir)
	}()
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

	invalidPath := string([]byte{0x00})
	exists, err = PathExists(invalidPath)
	assert.Error(t, err)
	assert.False(t, exists)
}

func TestDownloadArtifacts(t *testing.T) {
	tmpFolderPath, err := os.MkdirTemp("/tmp", "test_download_artifacts")
	require.NoError(t, err)
	defer os.RemoveAll(tmpFolderPath)

	expectedFileContent := "GOOD TEST!"
	// Expected file path comes from the internal path manipulation done by the DownloadArtifact function
	//expectedFilePath := tmpFolderPath + "/tmp/" + config.ReleaseVersion + ".yaml"

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
	err = DownloadArtifacts(config.PVC, testTag, testManifest)
	assert.NoError(t, err)

	// // Assert file is created with expected content
	// _, err = os.Stat(expectedFilePath)
	// require.NoError(t, err)
	// data, err := os.ReadFile(expectedFilePath)
	// require.NoError(t, err)
	// assert.Equal(t, expectedFileContent, string(data))
}

func TestDownloadMicroOS(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	localPath := path.Dir(filename)

	expectedFileContent := "GOOD TEST!"
	originalDir, _ := os.Getwd()
	src := strings.Replace(originalDir, "curation", "script", -1)
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
	dir := config.PVC
	os.MkdirAll(dir, 0755)
	// Test: No tmpFolderPath/hook dir
	t.Run("Fail", func(t *testing.T) {
		_, err = DownloadMicroOS(config.PVC, src)
		require.NoError(t, err)
		// assert.Contains(t, err.Error(), "no such file or directory")
	})

	err = os.MkdirAll(dkamHookFolderPath, 0755)
	require.NoError(t, err)

	// Test: empty manifest
	t.Run("NoAnnotationLayer", func(t *testing.T) {
		returnWrongManifest = true
		_, err = DownloadMicroOS(config.PVC, src)
		require.NoError(t, err)
	})

	// Test: successful, create tmpFolderPath/hook dir
	t.Run("Success", func(t *testing.T) {
		returnWrongManifest = false
		_, err = DownloadMicroOS(config.PVC, src)
		require.NoError(t, err)
	})
}

func Test_downloadImage(t *testing.T) {
	sercv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "responseBody")
	}))
	defer sercv.Close()
	type args struct {
		url       string
		fileName  string
		targetDir string
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
		{
			name: "Valid URL",
			args: args{
				url:       sercv.URL,
				fileName:  "testfile.jpg",
				targetDir: t.TempDir(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := downloadImage(tt.args.url, tt.args.fileName, tt.args.targetDir); (err != nil) != tt.wantErr {
				t.Errorf("downloadImage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_installPackage(t *testing.T) {
	installPackage("")
}

func TestDownloadMicroOS_Case1(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	localPath := path.Dir(filename)
	expectedFileContent := "GOOD TEST!"
	originalDir, _ := os.Getwd()
	src := strings.Replace(originalDir, "curation", "script", -1)
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
		_, err = DownloadMicroOS(config.PVC, src)
		// require.Error(t, err)
		// assert.Contains(t, err.Error(), "no such file or directory")
	})

	err = os.MkdirAll(dkamHookFolderPath, 0755)
	require.NoError(t, err)

	// Test: empty manifest
	t.Run("NoAnnotationLayer", func(t *testing.T) {
		returnWrongManifest = true
		_, err = DownloadMicroOS(config.PVC, src)
		// require.NoError(t, err)
	})

	t.Run("Success", func(t *testing.T) {
		existingDir := config.PVC
		subfolder := "tmp"
		subfolderPath := filepath.Join(existingDir, subfolder)
		err := os.MkdirAll(subfolderPath, 0755)
		if err != nil {
			fmt.Println("Error creating subfolder:", err)
			return
		}

		dummy := originalDir + "dummy.yaml"
		data := Data{
			Provisioning: struct {
				Files []File `yaml:"files"`
			}{
				Files: []File{
					{
						Description: "Dummy file 1",
						Server:      "server1",
						Path:        "one-intel-edge/edge-node/file/provisioning-hook-os",
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
		err = os.WriteFile(yamlFilePath, yamlData, 0644)
		if err != nil {
			fmt.Printf("Error writing YAML to file: %v\n", err)
			return
		}
		returnWrongManifest = true
		_, err = DownloadMicroOS(config.PVC, dummy)
		defer func() {
			os.RemoveAll(existingDir)
		}()
		// require.NoError(t, err)
	})
}

func TestDownloadUbuntuImage_Negative(t *testing.T) {
	sercv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "responseBody")
	}))
	defer sercv.Close()
	type args struct {
		imageUrl  string
		format    string
		fileName  string
		targetDir string
		sha256    string
	}
	fileName := fileNameFromURL(config.ImageUrl)
	rawFileName := strings.TrimSuffix(fileName, ".img") + ".raw.gz"
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Negative",
			args: args{
				imageUrl:  config.ImageUrl,
				format:    "image.img",
				fileName:  rawFileName,
				targetDir: config.PVC,
			},
			wantErr: true,
		},
		{
			name: "Negative test case",
			args: args{
				imageUrl:  "raw.gz",
				format:    "image.img",
				fileName:  rawFileName,
				targetDir: config.PVC,
			},
			wantErr: true,
		},
		{
			name: "Negative test case_1",
			args: args{
				imageUrl:  sercv.URL,
				format:    "image.img",
				fileName:  rawFileName,
				targetDir: t.TempDir(),
			},
			wantErr: true,
		},
		{
			name: "Negative test case_2",
			args: args{
				imageUrl:  "://example.com",
				format:    "image.img",
				fileName:  rawFileName,
				targetDir: t.TempDir(),
			},
			wantErr: true,
		},
		{
			name: "Negative test case_3",
			args: args{
				imageUrl:  sercv.URL + "/raw.gz",
				format:    "image.img",
				fileName:  rawFileName,
				targetDir: t.TempDir(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			DownloadUbuntuImage(tt.args.imageUrl, tt.args.format, tt.args.fileName, tt.args.targetDir, tt.args.sha256)
		})
	}
}

func TestDownloadUbuntuimage(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "responseBody *final.img")
	}))
	defer svr.Close()
	tmpFolderPath, err := os.MkdirTemp("/tmp", "test_download_ubuntu")
	require.NoError(t, err)
	defer os.RemoveAll(tmpFolderPath)
	expectedFileName := "final.raw.gz"
	dir := config.PVC
	os.MkdirAll(dir, 0755)
	imgName := "image.img"
	err = DownloadUbuntuImage(svr.URL, imgName, expectedFileName, config.DownloadPath, "")
	require.NoError(t, err)
	defer func() {
		os.Remove(dir)
	}()
}

func fileNameFromURL(url string) string {
	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}

func TestDownloadPrecuratedScript(t *testing.T) {
	expAcceptHeader := "application/vnd.oci.image.manifest.v1+json"
	svr1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Header.Get("Accept"), expAcceptHeader)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(exampleManifest))
	}))
	defer svr1.Close()
	svr2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(example))
	}))
	defer svr2.Close()
	originalRSProxy := config.RSProxy
	originalRSProxyManifest := config.RSProxyManifest
	defer func() {
		config.RSProxy = originalRSProxy
		config.RSProxyManifest = originalRSProxyManifest
	}()
	config.RSProxy = svr1.URL + "/"
	config.RSProxyManifest = svr2.URL + "/"
	type args struct {
		profile string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				profile: "profile:profile",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DownloadPrecuratedScript(tt.args.profile); (err != nil) != tt.wantErr {
				t.Errorf("DownloadPrecuratedScript() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_getMD5Checksum(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "getMD5Checksum test case",
			args: args{
				filename: "",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "getMD5Checksum test case negative",
			args: args{
				filename: t.TempDir(),
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getMD5Checksum(tt.args.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("getMD5Checksum() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getMD5Checksum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseMD5SUMS(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]string
		wantErr bool
	}{
		{
			name: "Test case",
			args: args{
				filename: "",
			},
			want:    map[string]string{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMD5SUMS(tt.args.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMD5SUMS() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseMD5SUMS() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetReleaseServerRespons(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name string
		args args
		want Response
	}{
		{
			name: "Empty url",
			args: args{},
			want: Response{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GetReleaseServerResponse(tt.args.url)
		})
	}
}
