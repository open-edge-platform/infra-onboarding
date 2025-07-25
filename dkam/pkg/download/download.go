// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package download

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/config"
)

var (
	zlog   = logging.GetLogger("InfraDKAMDownload")
	Client = &http.Client{
		Transport: &http.Transport{
			Proxy:             http.ProxyFromEnvironment,
			ForceAttemptHTTP2: false,
		},
	}
)

const (
	UOSFileName = "emb_uos_x86_64.tar.gz"
)

//nolint:revive // Keeping the function name for clarity and consistency.
func DownloadMicroOS(ctx context.Context) (bool, error) {
	zlog.Info().Msgf("Inside Download and sign artifact... %s", config.DownloadPath)
	fileServerAddress := config.GetInfraConfig().CDN
	if fileServerAddress == "" {
		invErr := inv_errors.Errorf("FileServerURL is not set in the configuration")
		zlog.Err(invErr).Msg("")
		return false, invErr
	}

	embImgURL := config.GetInfraConfig().EMBImageURL
	if embImgURL == "" {
		invErr := inv_errors.Errorf("EMBImageURL is not set in the configuration")
		zlog.Err(invErr).Msg("")
		return false, invErr
	}

	uOSUrl, err := url.JoinPath(fileServerAddress, embImgURL)
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msgf("Failed to generate MicroOS URL")
		return false, err
	}
	if !strings.HasPrefix(uOSUrl, "http://") && !strings.HasPrefix(uOSUrl, "https://") {
		uOSUrl = "https://" + uOSUrl
	}
	zlog.InfraSec().Info().Msgf("Downloading uOS from URL: %s", uOSUrl)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uOSUrl, http.NoBody)
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msgf("Failed to create GET request to release server: %v", err)
		return false, err
	}

	// Perform the HTTP GET request
	resp, err := Client.Do(req)
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msgf("Failed to connect to release server to download package manifest: %v", err)
		return false, err
	}
	defer resp.Body.Close()

	uOSFilePath := config.DownloadPath + "/" + UOSFileName

	file, fileerr := os.Create(uOSFilePath)
	if fileerr != nil {
		zlog.InfraSec().Error().Err(fileerr).Msgf("Failed to create file:%v", fileerr)
		return false, fileerr
	}
	defer file.Close()

	// Copy the response body to the local file
	_, copyErr := io.Copy(file, resp.Body)
	if copyErr != nil {
		zlog.InfraSec().Error().Err(copyErr).Msgf("Error while copying content ")
	}

	zlog.InfraSec().Info().Msg("File downloaded")
	return true, nil
}
