// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package download

import (
	"context"
	"io"
	"net/http"
	"os"

	as "github.com/open-edge-platform/infra-core/inventory/v2/pkg/artifactservice"
	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-onboarding/dkam/internal/env"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/config"
)

var (
	zlog   = logging.GetLogger("InfraDKAMDownload")
	client = &http.Client{
		Transport: &http.Transport{
			Proxy:             http.ProxyFromEnvironment,
			ForceAttemptHTTP2: false,
		},
	}
)

//nolint:revive // Keeping the function name for clarity and consistency.
func DownloadMicroOS(ctx context.Context) (bool, error) {
	zlog.Info().Msgf("Inside Download and sign artifact... %s", config.DownloadPath)
	repo := env.UOSRepo
	uOSVersion := env.UOSVersion
	rsProxyAddress := env.RSProxyAddress
	if rsProxyAddress == "" {
		invErr := inv_errors.Errorf("%s env variable is not set", rsProxyAddress)
		zlog.Err(invErr).Msg("")
		return false, invErr
	}
	uOSFileName := "emt_uos_x86_64_" + uOSVersion + ".tar.gz"
	url := "http://" + rsProxyAddress + repo + uOSFileName
	zlog.InfraSec().Info().Msgf("Downloading uOS from URL: %s", url)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msgf("Failed to create GET request to release server: %v", err)
		return false, err
	}

	// Perform the HTTP GET request
	resp, err := client.Do(req)
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msgf("Failed to connect to release server to download package manifest: %v", err)
		return false, err
	}
	defer resp.Body.Close()

	uOSFilePath := config.DownloadPath + "/" + "emt_uos_x86_64.tar.gz"

	file, fileerr := os.Create(uOSFilePath)
	if fileerr != nil {
		zlog.InfraSec().Error().Err(fileerr).Msgf("Failed to create file:%v", fileerr)
		return false, fileerr
	}
	defer file.Close()

	// Copy the response body to the local file
	_, copyErr := io.Copy(file, resp.Body)
	if copyErr != nil {
		zlog.InfraSec().Error().Err(copyErr).Msgf("Error while coping content ")
	}

	zlog.InfraSec().Info().Msg("File downloaded")
	return true, nil
}

func CreateFile(filePath string, artifact *as.Artifact) error {
	file, fileErr := os.Create(filePath)
	if fileErr != nil {
		zlog.InfraSec().Error().Err(fileErr).Msgf("Error while creating file %v", filePath)
		return fileErr
	}
	defer file.Close()

	_, err := file.Write(artifact.Data)
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msgf("Error writing to file:%v", err)
		return err
	}
	return nil
}
