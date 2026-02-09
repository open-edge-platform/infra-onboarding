// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package platformbundle_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/platformbundle"
)

func TestParsePlatformBundle(t *testing.T) {
	type args struct {
		platformBundle string
	}
	tests := []struct {
		name    string
		args    args
		want    platformbundle.PlatformBundleManifest
		wantErr bool
	}{
		{
			name:    "Failed_WrongFormat",
			args:    args{platformBundle: "platformBundle"},
			want:    platformbundle.PlatformBundleManifest{},
			wantErr: true,
		},
		{
			name: "Positive",
			args: args{platformBundle: `{"cloudInitScript":"CloudInitScript","installerScript":"InstallerScript"}`},
			want: platformbundle.PlatformBundleManifest{
				CloudInitScript: "CloudInitScript",
				InstallerScript: "InstallerScript",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := platformbundle.ParsePlatformBundle(tt.args.platformBundle)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePlatformBundle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParsePlatformBundle() = %v, want %v", got, tt.want)
			}
		})
	}
}

func mockPlatformBundleHandler() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/edge-orch/en/files/platformbundle/ubuntu-22.04-lts-generic-ext/manifests/1.0.1",
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
			w.WriteHeader(http.StatusOK)

			mockResponse := `{
			"mediaType": "application/vnd.oci.image.manifest.v1+json",
			"artifactType": "application/vnd.oci.image.manifest.v1+json",
			"config": {
				"mediaType": "application/vnd.oci.image.config.v1+json",
				"digest": "sha256:123456789abcdef",
				"size": 123,
				"urls": ["http://example.com/mock-config"],
				"annotations": {
					"org.opencontainers.image.description": "Mocked platform bundle config"
				},
				"data": "bW9ja2VkLWRhdGE=",
				"artifactType": "application/vnd.oci.image.config.v1+json"
			},
			"layers": [
				{
					"mediaType": "application/vnd.oci.image.layer.v1.tar+gzip",
					"digest": "sha256:abcdef123456789",
					"size": 456,
					"urls": ["http://example.com/mock-layer"],
					"annotations": {
						"org.opencontainers.image.description": "Mocked platform bundle layer"
					},
					"data": "bW9ja2VkLWxheWVy",
					"artifactType": "application/vnd.oci.image.layer.v1.tar+gzip"
				}
			],
			"annotations": {
				"org.opencontainers.image.created": "2025-03-25T12:00:00Z",
				"org.opencontainers.image.title": "Mocked Platform Bundle"
			}
		}`

			_, err := w.Write([]byte(mockResponse))
			if err != nil {
				fmt.Println("Error writing response:", err)
			}
		})
	mux.HandleFunc("/v2/edge-orch/en/files/platformbundle/cloudinit:1.0.1",
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
			w.WriteHeader(http.StatusOK)

			mockResponse := `{
			"mediaType": "application/vnd.oci.image.manifest.v1+json",
			"artifactType": "application/vnd.oci.image.manifest.v1+json",
			"config": {
				"mediaType": "application/vnd.oci.image.config.v1+json",
				"digest": "sha256:123456789abcdef",
				"size": 123,
				"urls": ["http://example.com/mock-config"],
				"annotations": {
					"org.opencontainers.image.description": "Mocked platform bundle config"
				},
				"data": "bW9ja2VkLWRhdGE=",
				"artifactType": "application/vnd.oci.image.config.v1+json"
			},
			"layers": [
				{
					"mediaType": "application/vnd.oci.image.layer.v1.tar+gzip",
					"digest": "sha256:abcdef123456789",
					"size": 456,
					"urls": ["http://example.com/mock-layer"],
					"annotations": {
						"org.opencontainers.image.description": "Mocked platform bundle layer"
					},
					"data": "bW9ja2VkLWxheWVy",
					"artifactType": "application/vnd.oci.image.layer.v1.tar+gzip"
				}
			],
			"annotations": {
				"org.opencontainers.image.created": "2025-03-25T12:00:00Z",
				"org.opencontainers.image.title": "Mocked Platform Bundle"
			}
		}`

			_, err := w.Write([]byte(mockResponse))
			if err != nil {
				fmt.Println("Error writing response:", err)
			}
		})
	svr := httptest.NewServer(mux)

	testRegistryEndpoint, _ := strings.CutPrefix(svr.URL, "http://")
	if err := os.Setenv("RSPROXY_ADDRESS", testRegistryEndpoint); err != nil {
		panic(err) // In test setup, panic is acceptable
	}

	return svr
}

func TestFetchPlatformBundleScripts(t *testing.T) {
	server := mockPlatformBundleHandler()
	defer func() {
		server.Close()
		if err := os.Unsetenv("RSPROXY_ADDRESS"); err != nil {
			t.Logf("Failed to unset env: %v", err)
		}
	}()
	type args struct {
		ctx            context.Context
		platformBundle string
	}
	tests := []struct {
		name    string
		args    args
		want    platformbundle.PlatformBundleData
		wantErr bool
	}{
		// {
		// 	name: "Negative",
		// 	args: args{
		// 		ctx:            context.Background(),
		// 		platformBundle: "platformBundle",
		// 	},
		// 	wantErr: true,
		// },
		{
			name: "Empty platformBundle",
			args: args{
				ctx:            context.Background(),
				platformBundle: "null",
			},
			want:    platformbundle.PlatformBundleData{},
			wantErr: false,
		},
		// {
		// 	name: "Wrong cloud-init format",
		// 	args: args{
		// 		ctx: context.Background(),
		// 		platformBundle: `{"cloudInitScript":"edge-orch/en/files/platformbundle/cloudinit:1.0.1",` +
		// 			`"installerScript":"InstallerScript"}`,
		// 	},
		// 	want:    platformbundle.PlatformBundleData{},
		// 	wantErr: true,
		// },
		// {
		// 	name: "Wrong Installer script format",
		// 	args: args{
		// 		ctx: context.Background(),
		// 		platformBundle: `{"cloudInitScript":"CloudInitScript",` +
		// 			`"installerScript":"edge-orch/en/files/platformbundle/ubuntu-22.04-lts-generic-ext:1.0.1"}`,
		// 	},
		// 	want:    platformbundle.PlatformBundleData{},
		// 	wantErr: true,
		// },
		// {
		// 	name: "Success",
		// 	args: args{
		// 		ctx: context.Background(),
		// 		platformBundle: `{"cloudInitScript":"edge-orch/en/files/platformbundle/cloudinit:1.0.1",` +
		// 			`"installerScript":"edge-orch/en/files/platformbundle/ubuntu-22.04-lts-generic-ext:1.0.1"}`,
		// 	},
		// 	want:    platformbundle.PlatformBundleData{},
		// 	wantErr: true,
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := platformbundle.FetchPlatformBundleScripts(tt.args.ctx, tt.args.platformBundle)
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchPlatformBundleScripts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FetchPlatformBundleScripts() = %v, want %v", got, tt.want)
			}
		})
	}
}
