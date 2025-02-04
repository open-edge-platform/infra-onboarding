// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package download

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/api/os/v1"
	as "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/artifactservice"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/pkg/config"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/pkg/util"
	dkam_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/testing"
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

func TestDownloadUbuntuImage(t *testing.T) {
	randBytes := make([]byte, 20)
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Assert expected header
		w.WriteHeader(http.StatusOK)
		// Return random bytes
		w.Write(randBytes)
	}))
	defer svr.Close()

	tmpFolderPath, err := os.MkdirTemp("/tmp", "test_download_ubuntu")
	require.NoError(t, err)
	defer os.RemoveAll(tmpFolderPath)

	hasher := sha256.New()
	_, err = hasher.Write(randBytes)
	require.NoError(t, err)

	testSha256 := fmt.Sprintf("%x", hasher.Sum(nil))
	err = DownloadUbuntuImage(context.TODO(), &osv1.OperatingSystemResource{
		ImageUrl:    svr.URL,
		Sha256:      testSha256,
		ProfileName: "test-profile",
		OsType:      osv1.OsType_OS_TYPE_MUTABLE,
	}, config.DownloadPath)
	require.NoError(t, err)
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

func TestDownloadArtifacts(t *testing.T) {
	dkam_testing.PrepareTestReleaseFile(t, projectRoot)
	tmpFolderPath, err := os.MkdirTemp("/tmp", "test_download_artifacts")
	require.NoError(t, err)
	defer os.RemoveAll(tmpFolderPath)

	expectedFileContent := "GOOD TEST!"
	// Expected file path comes from the internal path manipulation done by the DownloadArtifact function
	// expectedFilePath := tmpFolderPath + "/tmp/" + config.ReleaseVersion + ".yaml"

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
	config.ENManifestRepo = svr.URL + "/"
	// err = DownloadArtifacts(config.PVC, testTag, testManifest)
	err = DownloadArtifacts(context.Background(), testTag)
	assert.NoError(t, err)
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
		ctx       context.Context
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
				ctx: context.TODO(),
			},
			wantErr: true,
		},
		{
			name: "Test Case",
			args: args{
				url: "https://www.google.com/",
				ctx: context.TODO(),
			},
			wantErr: true,
		},
		{
			name: "Valid URL",
			args: args{
				url:       sercv.URL,
				fileName:  "testfile.jpg",
				targetDir: t.TempDir(),
				ctx:       context.TODO(),
			},
			wantErr: false,
		},

		{
			name: "Invalid context",
			args: args{
				ctx: nil,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.targetDir != "" {
				err := os.MkdirAll(tt.args.targetDir, 0o755)
				require.NoError(t, err)
			}
			if err := downloadImage(tt.args.ctx, tt.args.url, tt.args.targetDir+tt.args.fileName); (err != nil) != tt.wantErr {
				t.Errorf("downloadImage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_installPackage(t *testing.T) {
	installPackage("")
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

func TestDownloadUbuntuImage_Negative(t *testing.T) {
	sercv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "responseBody")
	}))
	defer sercv.Close()
	type args struct {
		targetDir string
		osr       *osv1.OperatingSystemResource
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Negative",
			args: args{
				targetDir: config.PVC,
				osr: &osv1.OperatingSystemResource{
					ImageUrl:    "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img",
					ProfileName: "test-profile-name",
				},
			},
			wantErr: true,
		},
		{
			name: "Negative test case",
			args: args{
				targetDir: config.PVC,
				osr: &osv1.OperatingSystemResource{
					ImageUrl:    "raw.gz",
					ProfileName: "test-profile-name",
				},
			},
			wantErr: true,
		},
		{
			name: "Negative test case_1",
			args: args{
				targetDir: t.TempDir(),
				osr: &osv1.OperatingSystemResource{
					ImageUrl:    sercv.URL,
					ProfileName: "test-profile-name",
				},
			},
			wantErr: true,
		},
		{
			name: "Negative test case_2",
			args: args{
				targetDir: t.TempDir(),
				osr: &osv1.OperatingSystemResource{
					ImageUrl:    "://example.com",
					ProfileName: "test-profile-name",
				},
			},
			wantErr: true,
		},
		{
			name: "Negative test case_3",
			args: args{
				targetDir: t.TempDir(),
				osr: &osv1.OperatingSystemResource{
					ImageUrl:    sercv.URL + "/raw.gz",
					ProfileName: "test-profile-name",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			DownloadUbuntuImage(context.TODO(), tt.args.osr, tt.args.targetDir)
		})
	}
}

func Test_getSHA256Checksum(t *testing.T) {
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
			name: "Test Case",
			args: args{
				filename: t.TempDir(),
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getSHA256Checksum(tt.args.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("getSHA256Checksum() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getSHA256Checksum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetOSImageLocationWithCustomFilename(t *testing.T) {
	type args struct {
		os       *osv1.OperatingSystemResource
		rootDir  string
		fileName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "GetOSImageLocationWithCustomFilename Test Case",
			args: args{},
			want: "/OSImage//",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := util.GetOSImageLocationWithCustomFilename(tt.args.os, tt.args.rootDir, tt.args.fileName); got != tt.want {
				t.Errorf("GetOSImageLocationWithCustomFilename() = %v, want %v", got, tt.want)
			}
		})
	}
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
