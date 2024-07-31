// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package download

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/config"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/curation"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
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

// Extract the digest value from the appropriate layer
var res Response

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

func DownloadMicroOS(targetDir string, scriptPath string) (bool, error) {
	zlog.Info().Msgf("Inside Download and sign artifact... %s", targetDir)
	yamlFile := filepath.Join(targetDir, "tmp", config.ReleaseVersion+".yaml")
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
		if file.Path == "one-intel-edge/edge-node/file/provisioning-hook-os" {
			zlog.MiSec().Info().Msgf("Version for hook os:%s", file.Version)
			version = file.Version
		}
	}

	zlog.MiSec().Info().Msgf("Hook OS version %s", version)

	url := config.RSProxy + "manifests/" + version
	zlog.MiSec().Info().Msgf("URL is %s", url)
	res := GetReleaseServerResponse(url)
	if res.Layers != nil {
		// Iterate through layers and print digest and title

		for _, layer := range res.Layers {
			zlog.MiSec().Info().Msgf("Layer Digest:%s", layer.Digest)
			digest := layer.Digest
			title, exists := layer.Annotations["org.opencontainers.image.title"]
			if exists {
				zlog.MiSec().Info().Msgf("Image Title:%s", title)
				title := title
				// Create an HTTP GET request with the specified URL
				file_url := config.RSProxy + "blobs/" + digest
				req, httperr := http.NewRequest("GET", file_url, nil)
				if httperr != nil {
					//zlog.MiSec().Fatal().Err(httperr).Msgf("Error creating request: %v\n", httperr)
					zlog.MiSec().Info().Msg("Failed create GET request to release server.")
					return false, httperr

				}

				// Perform the HTTP GET request
				resp, clienterr := client.Do(req)
				if clienterr != nil {
					//zlog.MiSec().Fatal().Err(clienterr).Msgf("Error performing request: %v\n", clienterr)
					zlog.MiSec().Info().Msg("Failed to connect to release server to download hookOS.")
					return false, clienterr

				}
				defer resp.Body.Close()
				zlog.MiSec().Info().Msgf("Downloading %s", title)
				filePath := targetDir + "/" + title

				file, fileerr := os.Create(filePath)
				if fileerr != nil {
					//zlog.MiSec().Fatal().Err(fileerr).Msgf("Error while creating release manifest file.")
					zlog.MiSec().Info().Msg("Failed to create file")
					return false, fileerr
				}
				defer file.Close()

				// Copy the response body to the local file
				_, copyErr := io.Copy(file, resp.Body)
				if copyErr != nil {
					zlog.MiSec().Error().Err(copyErr).Msgf("Error while coping content ")
				}

			} else {
				zlog.MiSec().Info().Msg("Image Title not found")
			}
		}
	}

	zlog.MiSec().Info().Msg("File downloaded")
	return true, nil

}

func DownloadPrecuratedScript(profile string) error {

	url := config.RSProxyProfileManifest + strings.Split(profile, ":")[0] + "/manifests/" + strings.Split(profile, ":")[1]
	zlog.MiSec().Info().Msgf("Manifest download URL is:%s", url)
	res := GetReleaseServerResponse(url)
	if res.Layers != nil {
		// Access the digest value
		digest := res.Layers[0].Digest
		zlog.MiSec().Info().Msgf("Digest: %s", digest)

		file_url := config.RSProxyProfileManifest + strings.Split(profile, ":")[0] + "/blobs/" + digest

		req2, geterr2 := http.NewRequest("GET", file_url, nil)
		if geterr2 != nil {
			zlog.MiSec().Error().Err(geterr2).Msgf("Error while making 2nd get request: %v\n", geterr2)

		}
		resp2, err2 := client.Do(req2)
		if err2 != nil {
			zlog.MiSec().Error().Err(err2).Msgf("Client Error: %v\n", err2)
		}
		defer resp2.Body.Close()
		filePath := config.DownloadPath + "/" + strings.Split(profile, ":")[0] + ".sh"

		//Create or open the local file for writing
		file, fileerr := os.Create(filePath)
		if fileerr != nil {
			zlog.MiSec().Error().Err(fileerr).Msgf("Error while creating precurated script.")
			return fileerr
		}
		defer file.Close()

		// Copy the response body to the local file
		_, copyErr := io.Copy(file, resp2.Body)
		if copyErr != nil {
			zlog.MiSec().Error().Err(copyErr).Msgf("Error while coping content ")
		}
	}
	zlog.MiSec().Info().Msg("Precurated script downloaded")
	return nil

}

func DownloadArtifacts(targetDir string, tag string, manifestTag string) error {

	outDir := filepath.Join(targetDir, "tmp")
	// 0. cleanup
	os.RemoveAll(outDir)

	mkErr := os.MkdirAll(outDir, 0755) // 0755 sets read, write, and execute permissions for owner, and read and execute permissions for others
	if mkErr != nil {
		zlog.MiSec().Error().Err(mkErr).Msgf("Error creating directory: %v", mkErr)
		return mkErr
	}
	zlog.MiSec().Info().Msg("tmp folder created successfully")
	zlog.MiSec().Info().Msgf("Tag is:%s", manifestTag)

	url := config.RSProxyManifest + tag + "/manifests/" + manifestTag
	zlog.MiSec().Info().Msgf("Manifest download URL is:%s", url)
	res := GetReleaseServerResponse(url)
	if res.Layers != nil {
		// Access the digest value
		digest := res.Layers[0].Digest
		zlog.MiSec().Info().Msgf("Digest: %s", digest)

		file_url := config.RSProxyManifest + tag + "/blobs/" + digest

		req2, geterr2 := http.NewRequest("GET", file_url, nil)
		if geterr2 != nil {
			zlog.MiSec().Error().Err(geterr2).Msgf("Error while making 2nd get request: %v\n", geterr2)

		}
		resp2, err2 := client.Do(req2)
		if err2 != nil {
			zlog.MiSec().Error().Err(err2).Msgf("Client Error: %v\n", err2)
		}
		defer resp2.Body.Close()
		filePath := outDir + "/" + config.ReleaseVersion + ".yaml"

		//Create or open the local file for writing
		file, fileerr := os.Create(filePath)
		if fileerr != nil {
			zlog.MiSec().Error().Err(fileerr).Msgf("Error while creating release manifest file.")
			return fileerr
		}
		defer file.Close()

		// Copy the response body to the local file
		_, copyErr := io.Copy(file, resp2.Body)
		if copyErr != nil {
			zlog.MiSec().Error().Err(copyErr).Msgf("Error while coping content ")
		}
	}
	zlog.MiSec().Info().Msg("File downloaded")
	return nil

}

func GetReleaseServerResponse(url string) Response {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		zlog.MiSec().Error().Err(err).Msgf("Error making get request: %v\n", err)

	}
	req.Header.Add("Accept", "application/vnd.oci.image.manifest.v1+json")

	response, clienterr := client.Do(req)
	if clienterr != nil {
		zlog.MiSec().Info().Msgf("Client Error: %v\n", clienterr)
	}
	if response != nil {
		defer response.Body.Close()

		// response details
		zlog.MiSec().Info().Msgf("Response Body:%s", response.Body)

		body, readerr := io.ReadAll(response.Body)
		if readerr != nil {
			panic(readerr)
		}

		//unmarshal the JSON response
		marshalerr := json.Unmarshal([]byte(body), &res)
		if marshalerr != nil {
			zlog.MiSec().Error().Err(marshalerr).Msgf("Error while json unmarshelling: %v\n", marshalerr)

		}
	}
	return res
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
func downloadImage(url string, fileName string, targetDir string) error {
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	filePath := targetDir + "/" + fileName
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	return err
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

func DownloadUbuntuImage(imageUrl string, format string, fileName string, targetDir string, sha256 string) error {
	// TODO(NEXFMPID-3359): avoid hardcoded file names, and use tmp folder for temporary files
	zlog.Info().Msgf("Inside Download and Raw form conversion...")
	if strings.HasSuffix(imageUrl, "raw.gz") {
		zlog.Info().Msgf("File is in raw format")
		if err := downloadImage(imageUrl, config.ImageFileName, targetDir); err != nil {
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
		if err := downloadImage(imageUrl, format, targetDir); err != nil {
			zlog.MiSec().Error().Err(err).Msgf("Error downloading image:%v", err)
			return err
		}

		// Convert the image to raw format
		if err := convertImage(targetDir+"/"+"image.img", targetDir+"/"+"image.raw"); err != nil {
			zlog.MiSec().Error().Err(err).Msgf("Error converting image:%v", err)
			return err
		}

		// Compress the raw image using pigz
		if err := compressImage(targetDir+"/"+"image.raw", targetDir+"/"+fileName); err != nil {
			zlog.MiSec().Error().Err(err).Msgf("Error compressing image:%v", err)
			return err
		}

		// Clean up temporary files
		if err := os.Remove(targetDir + "/" + "image.img"); err != nil {
			zlog.MiSec().Error().Err(err).Msgf("Error removing temporary file: image.img: %v", err)
		}
		if err := os.Remove(targetDir + "/" + "image.raw"); err != nil {
			zlog.MiSec().Error().Err(err).Msgf("Error removing temporary file: image.raw %v", err)
		}
	}
	exists, patherr := PathExists(targetDir + "/" + fileName)
	if patherr != nil {
		zlog.MiSec().Info().Msgf("Error checking image file path %v", patherr)
	}
	if exists {
		zlog.MiSec().Info().Msg("image raw file Path exists")
		exists, err := PathExists(config.PVC)
		if err != nil {
			zlog.MiSec().Info().Msgf("Error checking PVC path %v", err)
		}
		if exists {
			zlog.MiSec().Info().Msg("PVC Path exists")
			osImagePath := config.PVC + "/" + "OSImage"
			err = curation.CreateDir(osImagePath)
			if err != nil {
				zlog.MiSec().Info().Msgf("Error creating path %v", err)
			}
			osImagefilePath := osImagePath + "/" + sha256
			err = curation.CreateDir(osImagefilePath)
			if err != nil {
				zlog.MiSec().Info().Msgf("Error creating path %v", err)
			}

			zlog.MiSec().Info().Msg(fileName)
			pvcFilePath := osImagefilePath + "/" + fileName
			cmd := exec.Command("mv", targetDir+"/"+fileName, pvcFilePath)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			err := cmd.Run()
			if err != nil {
				zlog.MiSec().Info().Msgf("error running 'mv' command: %v", err)
			} else {
				zlog.MiSec().Info().Msg("OS image copied to PVC")
			}
		} else {
			zlog.MiSec().Info().Msg("PVC Path not exists")
		}

	} else {
		zlog.MiSec().Info().Msg("image raw file Path not exists:")

	}
	zlog.MiSec().Info().Msg("File downloaded, converted into raw format and move to PVC")
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
