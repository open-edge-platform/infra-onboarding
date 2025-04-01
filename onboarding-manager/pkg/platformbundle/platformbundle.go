// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package platformbundle

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"google.golang.org/grpc/codes"

	as "github.com/open-edge-platform/infra-core/inventory/v2/pkg/artifactservice"
	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

var (
	zlog  = logging.GetLogger("PlatformBundle")
	cache sync.Map
)

type PlatformBundleManifest struct { //nolint:revive // Struct field names must match JSON keys for unmarshaling
	CloudInitScript string `json:"cloudInitScript"` //nolint:tagliatelle // Struct field names must match JSON keys
	InstallerScript string `json:"installerScript"` //nolint:tagliatelle // Struct field names must match JSON keys
}

type PlatformBundleData struct { //nolint:revive // Struct field names must match JSON keys for unmarshaling
	CloudInitTemplate string
	InstallerScript   string
}

// ParsePlatformBundle parses the platform bundle JSON string into a PlatformBundle struct.
func ParsePlatformBundle(platformBundle string) (PlatformBundleManifest, error) {
	var platformBundleData PlatformBundleManifest
	zlog.InfraSec().Info().Msgf("Parse Platform Bundle Data: %s", platformBundle)
	err := json.Unmarshal([]byte(platformBundle), &platformBundleData)
	if err != nil {
		zlog.InfraSec().Error().Msgf("Error unmarshaling Platform Bundle Data: %v", err)
		return PlatformBundleManifest{}, err
	}
	return platformBundleData, nil
}

func validateAndParseArtifactURL(artifact string) (repo, tag string, err error) {
	zlog.InfraSec().Info().Msgf("Validate Platform Bundle %s", artifact)
	parts := strings.Split(artifact, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", inv_errors.Errorfc(
			codes.InvalidArgument,
			"Invalid artifact format. Expected 'repo:tag', got: %s",
			artifact,
		)
	}
	return parts[0], parts[1], nil
}

func FetchPlatformBundleData(ctx context.Context, artifact string) (string, error) {
	zlog.InfraSec().Info().Msgf("Fetch Platform Bundle for %s", artifact)
	// Validate and parse the artifact string
	repo, tag, err := validateAndParseArtifactURL(artifact)
	if err != nil {
		return "", err
	}

	key := artifact

	if content, found := cache.Load(key); found {
		// Return the cached content if found
		zlog.InfraSec().Info().Msgf("Returning cached content for %s", key)
		contentStr, ok := content.(string)
		if !ok {
			return "", inv_errors.Errorf("unexpected type for string: %T", content)
		}
		return contentStr, nil
	}

	// If not in cache, download the content
	zlog.InfraSec().Info().Msgf("Downloading content for %s", key)
	content, err := DownloadPlatformBundle(ctx, repo, tag)
	if err != nil {
		return "", err
	}

	// Store the downloaded content in the cache for future use
	cache.Store(key, content)

	return content, nil
}

func FetchPlatformBundleScripts(ctx context.Context, platformBundle string) (PlatformBundleData, error) {
	zlog.InfraSec().Info().Msgf("Platform bundle %s", platformBundle)
	scripts := PlatformBundleData{}

	if platformBundle == "null" || platformBundle == "" {
		scripts.CloudInitTemplate = ""
		scripts.InstallerScript = ""
	} else {
		// Fetch the cloud-init template
		zlog.InfraSec().Info().Msgf("Fetching Platform Bundle Data from Registry: %s", platformBundle)
		platformBundleData, err := ParsePlatformBundle(platformBundle)
		if err != nil {
			return scripts, err
		}
		// Fetch and store the CloudInitScript
		if platformBundleData.CloudInitScript != "" {
			cloudInitContent, err := FetchPlatformBundleData(ctx, platformBundleData.CloudInitScript)
			if err != nil {
				return scripts, err
			}
			scripts.CloudInitTemplate = cloudInitContent
		}
		// Fetch and store the InstallerScript
		if platformBundleData.InstallerScript != "" {
			installerContent, err := FetchPlatformBundleData(ctx, platformBundleData.InstallerScript)
			if err != nil {
				return scripts, err
			}
			scripts.InstallerScript = installerContent
		}
	}
	return scripts, nil
}

func DownloadPlatformBundle(ctx context.Context, repo, tag string) (string, error) {
	if repo == "" || tag == "" {
		return "", inv_errors.Errorfc(codes.InvalidArgument, "Repo or tag is empty. Repo: %s, Tag: %s", repo, tag)
	}

	zlog.InfraSec().Debug().Msgf("Starting download for repo: %s, tag: %s", repo, tag)

	artifacts, err := as.DownloadArtifacts(ctx, repo, tag)
	if err != nil {
		return "", inv_errors.Errorfc(codes.Unavailable, "PlatfromBundle not found %v", err)
	}

	if artifacts == nil || len(*artifacts) == 0 {
		return "", inv_errors.Errorfc(codes.Unavailable, "Empty artifact data")
	}

	scriptContent := (*artifacts)[0].Data

	zlog.InfraSec().Debug().Msgf("Successfully retrieved script content for repo: %s, tag: %s", repo, tag)

	return string(scriptContent), nil
}
