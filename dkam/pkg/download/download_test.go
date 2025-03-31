// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package download_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-onboarding/dkam/internal/env"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/config"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/download"
)

const (
	testDigest = "TEST_DIGEST"
	testFile   = "TEST_FILE"
)

// Manifest example from OCI repo, used by DKAM to gather the hookOS.
const exampleManifest = `
		{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json",
         "config":{"mediaType":"application/vnd.intel.hookos.file",
        "digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2},
        "layers":[{
            "mediaType":"application/vnd.oci.image.layer.v1.tar",
            "digest":"` + testDigest + `",
            "size":264193897,
            "annotations":{"org.opencontainers.image.title":"hook_x86_64.tar.gz"}
        }],
        "annotations":{"org.opencontainers.image.created":"2025-03-18T16:44:00Z"}}`

// Manifest example with no Annotation in Layers.
const exampleManifestWrong = `
		{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json",
         "config":{"mediaType":"application/vnd.intel.hookos.file",
        "digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2},
		"layers":[{
			"mediaType":"application/vnd.oci.image.layer.v1.tar",
			"digest":"` + testDigest + `",
			"size":24800,
 			"annotations":{"org.opencontainers.image.title":"hook_x86_64.tar.gz"}
		}],
		"annotations":{"org.opencontainers.image.created":"2025-03-18T16:44:00Z"}}`

func TestMain(m *testing.M) {
	var err error
	config.PVC, err = os.MkdirTemp(os.TempDir(), "test_pvc")
	if err != nil {
		panic(fmt.Sprintf("Error creating temp directory: %v", err))
	}

	run := m.Run()
	os.Exit(run)
}

func TestDownloadMicroOS(t *testing.T) {
	// TODO: refactor the test to improve functionality testing.
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
	mux.HandleFunc("/manifest/hookOS", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if returnWrongManifest {
			w.Write([]byte(exampleManifestWrong))
		} else {
			w.Write([]byte(exampleManifest))
		}
	})
	// Path comes from digest in the exampleManifest
	mux.HandleFunc("/blobs/"+testDigest, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedFileContent))
	})
	svr := httptest.NewServer(mux)
	defer svr.Close()

	// Override the RSProxy with test HTTP server
	env.HookOSRepo = svr.URL + "/manifest/hookOS"
	dir := config.PVC
	mkdirerr := os.MkdirAll(dir, 0o755)
	if mkdirerr != nil {
		fmt.Println("Error creating dir:", mkdirerr)
	}
	// Test: No tmpFolderPath/hook dir
	t.Run("Fail", func(t *testing.T) {
		_, err = download.DownloadMicroOS(context.Background())
		require.NoError(t, err)
	})

	err = os.MkdirAll(dkamHookFolderPath, 0o755)
	require.NoError(t, err)

	// Test: empty manifest
	t.Run("NoAnnotationLayer", func(t *testing.T) {
		returnWrongManifest = true
		_, err = download.DownloadMicroOS(context.Background())
		require.NoError(t, err)
	})

	// Test: successful, create tmpFolderPath/hook dir
	t.Run("Success", func(t *testing.T) {
		returnWrongManifest = false
		_, err = download.DownloadMicroOS(context.Background())
		require.NoError(t, err)
	})
}
