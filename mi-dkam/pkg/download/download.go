// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

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

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/util"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/os/v1"
	as "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/artifactservice"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/errors"

	"gopkg.in/yaml.v2"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/config"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/logging"
)

var zlog = logging.GetLogger("MIDKAMAuth")

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

var client = &http.Client{
	Transport: &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		ForceAttemptHTTP2: false,
		MaxIdleConns:      10,
		IdleConnTimeout:   30,
	},
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

func DownloadMicroOS(ctx context.Context, scriptPath string) (bool, error) {
	zlog.Info().Msgf("Inside Download and sign artifact... %s", config.DownloadPath)
	yamlFile := filepath.Join(config.DownloadPath, "tmp", config.ReleaseVersion+".yaml")
	exists, err := PathExists(yamlFile)
	if err != nil {
		zlog.MiSec().Info().Msgf("Error checking path %v", err)
	}
	releaseFilePath := ""
	if exists {
		zlog.MiSec().Info().Msg("Path exists:")
		releaseFilePath = yamlFile
	} else {
		zlog.MiSec().Info().Msg("Path not exists:")
		releaseFilePath = filepath.Join(scriptPath, config.ReleaseVersion+".yaml")
	}

	file, err := os.Open(releaseFilePath)
	if err != nil {
		zlog.MiSec().Error().Err(err).Msgf("Error opening file: %v", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		zlog.MiSec().Error().Err(err).Msgf("Error reading file: %v", err)
	}

	// Parse YAML
	var data Data
	if err := yaml.Unmarshal(content, &data); err != nil {
		zlog.MiSec().Error().Err(err).Msgf("Error unmarshalling YAML: %v", err)
	}

	for _, file := range data.Provisioning.Files {
		if file.Path == config.HookOSRepo {
			zlog.MiSec().Info().Msgf("Version for hook os:%s", file.Version)
			version = file.Version
		}
	}

	zlog.MiSec().Info().Msgf("Hook OS version %s", version)

	repo := config.HookOSRepo
	zlog.MiSec().Info().Msgf("Hook OS repo URL is %s", repo)
	artifacts, err := as.DownloadArtifacts(ctx, repo, version)
	if err != nil {
		invErr := inv_errors.Errorf("Error downloading HookOS for tag %s", version)
		zlog.Err(invErr).Msg("")
	}

	if artifacts != nil && len(*artifacts) > 0 {
		for _, artifact := range *artifacts {
			zlog.MiSec().Info().Msgf("Downloading artifact %s", artifact.Name)
			filePath := config.DownloadPath + "/" + artifact.Name

			err = CreateFile(filePath, &artifact)
			if err != nil {
				zlog.MiSec().Error().Err(err).Msg("Error writing to file")
				return false, err
			}
		}

	}

	zlog.MiSec().Info().Msg("File downloaded")
	return true, nil

}

// DownloadTiberOSImage downloads OS image from the Release Service,
// verifies the SHA256 checksum and copies the OS image to targetDir.
func DownloadTiberOSImage(ctx context.Context, osRes *osv1.OperatingSystemResource, targetDir string) error {

	url := config.RSProxyTiberOSManifest + osRes.GetImageUrl()
	zlog.MiSec().Info().Msg(url)

	req, httperr := http.NewRequestWithContext(ctx, "GET", url, nil)
	if httperr != nil {
		zlog.MiSec().Error().Err(httperr).Msgf("Failed create GET request to release server:%v", httperr)
		return httperr

	}

	// Perform the HTTP GET request
	resp, clienterr := client.Do(req)
	if clienterr != nil {
		zlog.MiSec().Error().Err(clienterr).Msgf("Failed to connect to release server to download hookOS:%v", clienterr)
		return clienterr

	}
	defer resp.Body.Close()

	tmpOsImageFilePath := config.DownloadPath + "/" + osRes.GetProfileName() + util.GetFileExtensionFromOSImageURL(osRes)

	file, fileErr := os.Create(tmpOsImageFilePath)
	if fileErr != nil {
		zlog.MiSec().Error().Err(fileErr).Msgf("Failed to create file:%v", fileErr)
		return fileErr
	}
	defer file.Close()

	// Copy the response body to the local file
	_, copyErr := io.Copy(file, resp.Body)
	if copyErr != nil {
		zlog.MiSec().Error().Err(copyErr).Msgf("Error while coping content ")
	}

	zlog.MiSec().Info().Msgf("Tiber OS Image downloaded from %s", url)

	zlog.Info().Msg("Calculating SHA256 checksum of downloaded image...")
	computedChecksum, err := getSHA256Checksum(tmpOsImageFilePath)
	if err != nil {
		zlog.MiSec().Error().Err(err).Msgf("Error calculating MD5 checksum:%v", err)
	}

	zlog.MiSec().Info().Msgf("Expected checksum: %s\n", osRes.GetSha256())
	zlog.MiSec().Info().Msgf("Computed checksum: %s\n", computedChecksum)

	if osRes.GetSha256() == computedChecksum {
		zlog.MiSec().Info().Msgf("Checksum verification succeeded!")
	} else {
		zlog.MiSec().Error().Err(err).Msgf("Checksum verification failed! Expected checksum:%s and Computed checksum:%s", osRes.GetSha256(), computedChecksum)
	}

	copyErr = CopyFile(
		tmpOsImageFilePath,
		util.GetOSImageLocation(osRes, targetDir),
	)
	if copyErr != nil {
		zlog.MiSec().Error().Err(copyErr).Msgf("Failed to copy file to PV:%v", copyErr)

	}

	return nil
}

func DownloadPrecuratedScript(ctx context.Context, profile string) error {
	// FIXME: hardcode profile script version for now, will be addressed in https://jira.devtools.intel.com/browse/NEX-11556
	profileScriptVersion := "1.0.2"
	repo := config.ProfileScriptRepo + profile
	zlog.MiSec().Info().Msgf("Profile script repo URL is:%s", repo)
	artifacts, err := as.DownloadArtifacts(ctx, repo, profileScriptVersion)
	if err != nil {
		invErr := inv_errors.Errorf("Error downloading profile script for tag %s", profileScriptVersion)
		zlog.Err(invErr).Msg("")
	}
	if artifacts != nil && len(*artifacts) > 0 {
		artifact := (*artifacts)[0]
		zlog.MiSec().Info().Msgf("Downloading artifact %s", artifact.Name)
		filePath := config.DownloadPath + "/" + profile + ".sh"

		err = CreateFile(filePath, &artifact)
		if err != nil {
			zlog.MiSec().Error().Err(err).Msg("Error writing to file")
			return err
		}

	}

	zlog.MiSec().Info().Msg("Precurated script downloaded")
	return nil

}

func DownloadArtifacts(ctx context.Context, manifestTag string) error {

	outDir := filepath.Join(config.DownloadPath, "tmp")
	// 0. cleanup
	os.RemoveAll(outDir)

	mkErr := os.MkdirAll(outDir, 0755) // 0755 sets read, write, and execute permissions for owner, and read and execute permissions for others
	if mkErr != nil {
		zlog.MiSec().Error().Err(mkErr).Msgf("Error creating directory: %v", mkErr)
		return mkErr
	}
	zlog.MiSec().Info().Msg("tmp folder created successfully")
	zlog.MiSec().Info().Msgf("Tag is:%s", manifestTag)

	repo := config.ENManifestRepo
	zlog.MiSec().Info().Msgf("Manifest repo URL is:%s", repo)
	artifacts, err := as.DownloadArtifacts(ctx, repo, manifestTag)
	if err != nil {
		invErr := inv_errors.Errorf("Error downloading EN Manifest file for tag %s", manifestTag)
		zlog.Err(invErr).Msg("")
	}
	if artifacts != nil && len(*artifacts) > 0 {
		artifact := (*artifacts)[0]
		zlog.MiSec().Info().Msgf("Downloading artifact %s", artifact.Name)
		filePath := outDir + "/" + config.ReleaseVersion + ".yaml"

		err = CreateFile(filePath, &artifact)
		if err != nil {
			zlog.MiSec().Error().Err(err).Msg("Error writing to file")
			return err
		}

	}
	zlog.MiSec().Info().Msg("File downloaded")
	return nil

}

func CreateFile(filePath string, artifact *as.Artifact) error {

	file, fileErr := os.Create(filePath)
	if fileErr != nil {
		zlog.MiSec().Error().Err(fileErr).Msgf("Error while creating file %v", fileErr)
		return fileErr
	}
	defer file.Close()

	_, err := file.Write(artifact.Data)
	if err != nil {
		zlog.MiSec().Error().Err(err).Msgf("Error writing to file:%v", err)
		return err
	}
	return nil
}

// Ensure that pigz and qemu-img are installed
func ensureDependencies() error {
	if !commandExists("pigz") {
		zlog.MiSec().Info().Msg("Installing pigz...")
		err := installPackage("pigz")
		if err != nil {
			return err
		}
	}

	if !commandExists("qemu-img") {
		zlog.MiSec().Info().Msg("Installing qemu-utils...")
		err := installPackage("qemu-utils")
		if err != nil {
			return err
		}
	}

	return nil
}

// Check if a command is available in the system
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// Install a package using the system's package manager
func installPackage(packageName string) error {
	cmdStr := fmt.Sprintf("sudo apt-get install -y %s", packageName) // Assuming Ubuntu Linux
	cmd := exec.Command("sh", "-c", cmdStr)
	return cmd.Run()
}

// Download an image from a URL and save it to a file
func downloadImage(ctx context.Context, url string, targetFilePath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create HTTP GET request to %s", url)
		zlog.MiSec().MiErr(err).Msgf(errMsg)
		return inv_errors.Errorf(errMsg)
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to download file from %s", url)
		zlog.MiSec().MiErr(err).Msgf(errMsg)
		return inv_errors.Errorf(errMsg)
	}
	defer response.Body.Close()

	file, err := os.Create(targetFilePath)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create file %s", targetFilePath)
		zlog.MiSec().MiErr(err).Msgf(errMsg)
		return inv_errors.Errorf(errMsg)
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to save file %s", targetFilePath)
		zlog.MiSec().MiErr(err).Msgf(errMsg)
		return inv_errors.Errorf(errMsg)
	}

	return nil
}

// Convert an image to raw format using qemu-img
func convertImage(inputFile, outputFile string) error {
	cmdStr := fmt.Sprintf("qemu-img convert -O raw %s %s", inputFile, outputFile)
	cmd := exec.Command("sh", "-c", cmdStr)
	return cmd.Run()
}

// Compress an image using pigz
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
			zlog.MiSec().Error().Err(err).Msgf("Error downloading image:%v", err)
			return err
		}

	} else {
		// Check and install dependencies if necessary
		zlog.Info().Msgf("File is in img format")
		// Check and install dependencies if necessary
		if err := ensureDependencies(); err != nil {
			zlog.MiSec().Error().Err(err).Msgf("Error installing dependencies: %v", err)
			return err
		}

		// Download the image
		if err := downloadImage(ctx, osRes.GetImageUrl(), tempDownloadDir+"/image.img"); err != nil {
			zlog.MiSec().Error().Err(err).Msgf("Error downloading image:%v", err)
			return err
		}

		zlog.Info().Msg("Calculating SHA256 checksum of downloaded image...")
		computedChecksum, err := getSHA256Checksum(tempDownloadDir + "/" + "image.img")
		if err != nil {
			zlog.MiSec().Error().Err(err).Msgf("Error calculating MD5 checksum:%v", err)
			return err
		}

		zlog.MiSec().Info().Msgf("Expected checksum: %s\n", osRes.GetSha256())
		zlog.MiSec().Info().Msgf("Computed checksum: %s\n", computedChecksum)

		if osRes.GetSha256() == computedChecksum {
			zlog.MiSec().Info().Msgf("Checksum verification succeeded!")
		} else {
			zlog.MiSec().Error().Err(err).Msgf(
				"Checksum verification failed! Expected checksum:%s and Computed checksum:%s",
				osRes.GetSha256(), computedChecksum)
		}

		// Convert the image to raw format
		if err := convertImage(tempDownloadDir+"/"+"image.img", tempDownloadDir+"/"+"image.raw"); err != nil {
			zlog.MiSec().Error().Err(err).Msgf("Error converting image:%v", err)
			return err
		}

		// Compress the raw image using pigz
		if err := compressImage(tempDownloadDir+"/"+"image.raw", tempDownloadDir+"/image.raw.gz"); err != nil {
			zlog.MiSec().Error().Err(err).Msgf("Error compressing image:%v", err)
			return err
		}

		// Clean up temporary files
		if err := os.Remove(tempDownloadDir + "/" + "image.img"); err != nil {
			zlog.MiSec().Error().Err(err).Msgf("Error removing temporary file: image.img: %v", err)
		}
		if err := os.Remove(tempDownloadDir + "/" + "image.raw"); err != nil {
			zlog.MiSec().Error().Err(err).Msgf("Error removing temporary file: image.raw %v", err)
		}
	}
	copyErr := CopyFile(
		tempDownloadDir+"/image.raw.gz",
		util.GetOSImageLocation(osRes, targetDir),
	)
	if copyErr != nil {
		zlog.MiSec().Error().Err(copyErr).Msgf("Failed to copy file to PV:%v", copyErr)
		return copyErr
	}

	zlog.MiSec().Info().Msg("File downloaded, converted into raw format and move to PVC")
	return nil

}

func CopyFile(source, destination string) error {
	exists, patherr := PathExists(source)
	if patherr != nil {
		zlog.MiSec().Error().Err(patherr).Msgf("Error checking file path %v", source)
		return patherr
	}
	if exists {
		destDir := filepath.Dir(destination)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			zlog.MiSec().Error().Err(err).Msgf("Failed to create destination dir %s", destDir)
			return err
		}

		cmd := exec.Command("mv", source, destination)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			zlog.MiSec().Info().Msgf("error running 'mv' command: %v", err)
			return err
		} else {
			zlog.MiSec().Info().Msgf("File %s copied to %s", source, destination)
		}
	} else {
		zlog.MiSec().Debug().Msgf("Source file path %s does not exist", source)
	}
	return nil
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil // path exists
	}
	if os.IsNotExist(err) {
		return false, nil // path does not exist
	}
	return false, err // an error occurred (other than not existing)
}
