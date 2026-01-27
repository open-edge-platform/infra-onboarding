// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package download_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/config"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/download"
)

// MockRoundTripper implements http.RoundTripper for testing.
type MockRoundTripper struct {
	ResponseBody string
	StatusCode   int
	Err          error
}

func (m *MockRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return &http.Response{
		StatusCode: m.StatusCode,
		Body:       io.NopCloser(bytes.NewBufferString(m.ResponseBody)),
	}, nil
}

func TestDownloadMicroOS_Success(t *testing.T) {
	// Setup mock HTTP client
	oldClient := download.Client
	download.Client = &http.Client{Transport: &MockRoundTripper{ResponseBody: "testdata", StatusCode: 200}}
	defer func() { download.Client = oldClient }()

	// Setup config using SetInfraConfig
	cfg := config.InfraConfig{
		CDN:         "localhost",
		EMBImageURL: "test-file",
	}
	config.SetInfraConfig(cfg)

	ok, err := download.DownloadMicroOS(context.Background())
	if !ok || err != nil {
		t.Fatalf("expected success, got err: %v", err)
	}

	// Check file exists
	filePath := config.DownloadPath + "/" + download.UOSFileName
	data, err := os.ReadFile(filePath) //nolint:gosec // Test code with controlled path
	if err != nil {
		t.Fatalf("expected file to be created, got err: %v", err)
	}
	if string(data) != "testdata" {
		t.Fatalf("file contents mismatch: got %s", string(data))
	}
}

func TestDownloadMicroOS_MissingConfig(t *testing.T) {
	// Set empty config
	cfg := config.InfraConfig{
		CDN:         "",
		EMBImageURL: "",
	}
	config.SetInfraConfig(cfg)
	ok, err := download.DownloadMicroOS(context.Background())
	if ok || err == nil {
		t.Fatalf("expected failure due to missing config")
	}
}

func TestDownloadMicroOS_HTTPError(t *testing.T) {
	oldClient := download.Client
	download.Client = &http.Client{Transport: &MockRoundTripper{Err: io.EOF}}
	defer func() { download.Client = oldClient }()

	cfg := config.InfraConfig{
		CDN:         "http://localhost",
		EMBImageURL: "test-file",
	}
	config.SetInfraConfig(cfg)

	ok, err := download.DownloadMicroOS(context.Background())
	if ok || err == nil {
		t.Fatalf("expected HTTP error")
	}
}
