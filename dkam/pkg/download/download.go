// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package download

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/api/os/v1"
	as "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/artifactservice"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/logging"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/pkg/config"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/pkg/util"
)

var zlog = logging.GetLogger("InfraDKAMAuth")

const fileMode = 0o755

//nolint:tagliatelle // Renaming the json keys may effect while unmarshalling/marshaling so, used nolint.
type Response struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	ArtifactType  string `json:"artifactType"`
	Config        struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
		Size      int    `json:"size"`
		Data      string `json:"data"`
	} `json:"config"`
	Layers []struct {
		MediaType   string            `json:"mediaType"`
		Digest      string            `json:"digest"`
		Size        int               `json:"size"`
		Annotations map[string]string `json:"annotations"`
	} `json:"layers"`
}

type File struct {
	Description string `yaml:"description"`
	Server      string `yaml:"server"`
	Path        string `yaml:"path"`
	Version     string `yaml:"version"`
}

var version string

type Data struct {
	Provisioning struct {
		Files []File `yaml:"files"`
	} `yaml:"provisioning"`
}

//nolint:revive // Keeping the function name for clarity and consistency.
func DownloadMicroOS(ctx context.Context) (bool, error) {
	zlog.Info().Msgf("Inside Download and sign artifact... %s", config.DownloadPath)

	releaseFilePath, err := util.GetReleaseFilePathIfExists()
	if err != nil {
		return false, err
	}

	file, err := os.Open(releaseFilePath)
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msgf("Error opening file: %v", err)
	}
	defer file.Close()

	version = getHookOSVersion(file)

	zlog.InfraSec().Info().Msgf("Hook OS version %s", version)

	repo := config.HookOSRepo
	zlog.InfraSec().Info().Msgf("Hook OS repo URL is %s", repo)
	artifacts, err := as.DownloadArtifacts(ctx, repo, version)
	if err != nil {
		invErr := inv_errors.Errorf("Error downloading HookOS for tag %s", version)
		zlog.Err(invErr).Msg("")
	}

	if artifacts != nil && len(*artifacts) > 0 {
		for _, artifact := range *artifacts {
			zlog.InfraSec().Info().Msgf("Downloading artifact %s", artifact.Name)
			filePath := config.DownloadPath + "/" + artifact.Name

			err = CreateFile(filePath, &artifact)
			if err != nil {
				zlog.InfraSec().Error().Err(err).Msg("Error writing to file")
				return false, err
			}
		}
	}

	zlog.InfraSec().Info().Msg("File downloaded")
	return true, nil
}

func getHookOSVersion(file *os.File) string {
	content, err := io.ReadAll(file)
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msgf("Error reading file: %v", err)
	}

	// Parse YAML
	var data Data
	if unamarshalErr := yaml.Unmarshal(content, &data); unamarshalErr != nil {
		zlog.InfraSec().Error().Err(unamarshalErr).Msgf("Error unmarshalling YAML: %v", unamarshalErr)
	}

	for _, file := range data.Provisioning.Files {
		if file.Path == config.HookOSRepo {
			zlog.InfraSec().Info().Msgf("Version for hook os:%s", file.Version)
			return file.Version
		}
	}
	return ""
}

//nolint:revive // Keeping the function name for clarity and consistency.
func DownloadArtifacts(ctx context.Context, manifestTag string) error {
	outDir := filepath.Join(config.DownloadPath, "tmp")

	mkErr := os.MkdirAll(outDir, fileMode)
	if mkErr != nil {
		zlog.InfraSec().Error().Err(mkErr).Msgf("Error creating directory: %v", mkErr)
		return mkErr
	}
	zlog.InfraSec().Info().Msg("tmp folder created successfully")
	zlog.InfraSec().Info().Msgf("Tag is:%s", manifestTag)

	repo := config.ENManifestRepo
	zlog.InfraSec().Info().Msgf("Manifest repo URL is:%s", repo)
	artifacts, err := as.DownloadArtifacts(ctx, repo, manifestTag)
	if err != nil {
		invErr := inv_errors.Errorf("Error downloading EN Manifest file for tag %s", manifestTag)
		zlog.Err(invErr).Msg("")
	}
	if artifacts != nil && len(*artifacts) > 0 {
		artifact := (*artifacts)[0]
		zlog.InfraSec().Info().Msgf("Downloading artifact %s", artifact.Name)
		filePath := outDir + "/" + config.ReleaseVersion + ".yaml"

		err = CreateFile(filePath, &artifact)
		if err != nil {
			zlog.InfraSec().Error().Err(err).Msg("Error writing to file")
			return err
		}
	}
	zlog.InfraSec().Info().Msg("File downloaded")
	return nil
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

// Ensure that pigz and qemu-img are installed.
func ensureDependencies() error {
	if !commandExists("pigz") {
		zlog.InfraSec().Info().Msg("Installing pigz...")
		err := installPackage("pigz")
		if err != nil {
			return err
		}
	}

	if !commandExists("qemu-img") {
		zlog.InfraSec().Info().Msg("Installing qemu-utils...")
		err := installPackage("qemu-utils")
		if err != nil {
			return err
		}
	}

	return nil
}

// Check if a command is available in the system.
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// Install a package using the system's package manager.
func installPackage(packageName string) error {
	cmdStr := fmt.Sprintf("sudo apt-get install -y %s", packageName) // Assuming Ubuntu Linux
	cmd := exec.Command("sh", "-c", cmdStr)
	return cmd.Run()
}

// Download an image from a URL and save it to a file.
func downloadImage(ctx context.Context, imageURL, targetFilePath string) error {
	//nolint:gocritic // Keeping nil as request body for compatibility reasons.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create HTTP GET request to %s", imageURL)
		zlog.InfraSec().InfraErr(err).Msgf("%s", errMsg)
		return inv_errors.Errorf("%s", errMsg)
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to download file from %s", imageURL)
		zlog.InfraSec().InfraErr(err).Msgf("%s", errMsg)
		return inv_errors.Errorf("%s", errMsg)
	}
	defer response.Body.Close()

	file, err := os.Create(targetFilePath)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create file %s", targetFilePath)
		zlog.InfraSec().InfraErr(err).Msgf("%s", errMsg)
		return inv_errors.Errorf("%s", errMsg)
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to save file %s", targetFilePath)
		zlog.InfraSec().InfraErr(err).Msgf("%s", errMsg)
		return inv_errors.Errorf("%s", errMsg)
	}

	return nil
}

// Convert an image to raw format using qemu-img.
func convertImage(inputFile, outputFile string) error {
	cmdStr := fmt.Sprintf("qemu-img convert -O raw %s %s", inputFile, outputFile)
	cmd := exec.Command("sh", "-c", cmdStr)
	return cmd.Run()
}

// Compress an image using pigz.
func compressImage(inputFile, outputFile string) error {
	cmdStr := fmt.Sprintf("pigz < %s > %s", inputFile, outputFile)
	cmd := exec.Command("sh", "-c", cmdStr)
	return cmd.Run()
}

func getSHA256Checksum(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// DownloadUbuntuImage downloads Ubuntu OS from the upstream mirror,
// verifies the SHA256 checksum and copies the OS image to targetDir.

//nolint:revive // Keeping the function name for clarity and consistency.
func DownloadUbuntuImage(ctx context.Context, osRes *osv1.OperatingSystemResource, targetDir string) error {
	// TODO(NEXFMPID-3359): avoid hardcoded file names, and use tmp folder for temporary files
	parsedURL, err := url.Parse(osRes.GetImageUrl())
	if err != nil {
		zlog.Info().Msgf("Error parsing URL:%v", err)
		return err
	}
	parsedURL.Path = path.Dir(parsedURL.Path)

	// Reconstruct the URL without the filename
	resultURL := parsedURL.String()
	zlog.Info().Msgf("URL without filename:%s", resultURL)

	tempDownloadDir := config.DownloadPath

	zlog.Info().Msgf("Inside Download and Raw form conversion...")
	if strings.HasSuffix(osRes.GetImageUrl(), "raw.gz") {
		zlog.Info().Msgf("File is in raw format")
		if err := downloadImage(ctx, osRes.GetImageUrl(),
			tempDownloadDir+"/"+osRes.GetProfileName()+".raw.gz"); err != nil {
			zlog.InfraSec().Error().Err(err).Msgf("Error downloading image:%v", err)
			return err
		}
	} else {
		if err := processImgFormat(ctx, osRes, tempDownloadDir); err != nil {
			return err
		}
	}
	moveErr := MoveFile(
		tempDownloadDir+"/image.raw.gz",
		util.GetOSImageLocation(osRes, targetDir),
	)
	if moveErr != nil {
		zlog.InfraSec().Error().Err(moveErr).Msgf("Failed to move file to PV:%v", moveErr)
		return moveErr
	}

	zlog.InfraSec().Info().Msg("File downloaded, converted into raw format and move to PVC")
	return nil
}

func processImgFormat(ctx context.Context, osRes *osv1.OperatingSystemResource, tempDownloadDir string) error {
	// Check and install dependencies if necessary
	zlog.Info().Msgf("File is in img format")
	// Check and install dependencies if necessary
	if err := ensureDependencies(); err != nil {
		zlog.InfraSec().Error().Err(err).Msgf("Error installing dependencies: %v", err)
		return err
	}

	// Download the image
	if err := downloadImage(ctx, osRes.GetImageUrl(), tempDownloadDir+"/image.img"); err != nil {
		zlog.InfraSec().Error().Err(err).Msgf("Error downloading image:%v", err)
		return err
	}

	zlog.Info().Msg("Calculating SHA256 checksum of downloaded image...")
	computedChecksum, err := getSHA256Checksum(tempDownloadDir + "/" + "image.img")
	if err != nil {
		zlog.InfraSec().Error().Err(err).Msgf("Error calculating MD5 checksum:%v", err)
		return err
	}

	zlog.InfraSec().Info().Msgf("Expected checksum: %s\n", osRes.GetSha256())
	zlog.InfraSec().Info().Msgf("Computed checksum: %s\n", computedChecksum)

	if osRes.GetSha256() == computedChecksum {
		zlog.InfraSec().Info().Msgf("Checksum verification succeeded!")
	} else {
		zlog.InfraSec().Error().Err(err).Msgf(
			"Checksum verification failed! Expected checksum:%s and Computed checksum:%s",
			osRes.GetSha256(), computedChecksum)
	}

	// Convert the image to raw format
	if err := convertImage(tempDownloadDir+"/"+"image.img", tempDownloadDir+"/"+"image.raw"); err != nil {
		zlog.InfraSec().Error().Err(err).Msgf("Error converting image:%v", err)
		return err
	}

	// Compress the raw image using pigz
	if err := compressImage(tempDownloadDir+"/"+"image.raw", tempDownloadDir+"/image.raw.gz"); err != nil {
		zlog.InfraSec().Error().Err(err).Msgf("Error compressing image:%v", err)
		return err
	}

	// Clean up temporary files
	if err := os.Remove(tempDownloadDir + "/" + "image.img"); err != nil {
		zlog.InfraSec().Error().Err(err).Msgf("Error removing temporary file: image.img: %v", err)
	}
	if err := os.Remove(tempDownloadDir + "/" + "image.raw"); err != nil {
		zlog.InfraSec().Error().Err(err).Msgf("Error removing temporary file: image.raw %v", err)
	}

	return nil
}

func MoveFile(source, destination string) error {
	exists, patherr := util.PathExists(source)
	if patherr != nil {
		zlog.InfraSec().Error().Err(patherr).Msgf("Error checking file path %v", source)
		return patherr
	}
	//nolint:revive // Ignoring due to specific need for this structure
	if exists {
		destDir := filepath.Dir(destination)
		if err := os.MkdirAll(destDir, fileMode); err != nil {
			zlog.InfraSec().Error().Err(err).Msgf("Failed to create destination dir %s", destDir)
			return err
		}

		cmd := exec.Command("mv", source, destination)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			zlog.InfraSec().Info().Msgf("error running 'mv' command: %v", err)
			return err
		} else {
			zlog.InfraSec().Info().Msgf("File %s copied to %s", source, destination)
		}
	} else {
		zlog.InfraSec().Debug().Msgf("Source file path %s does not exist", source)
	}
	return nil
}
